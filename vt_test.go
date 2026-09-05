package vt

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func feed(t *testing.T, s string) []string {
	t.Helper()
	v := New()
	v.Advance([]byte(s))
	return v.Output()
}

func mustEqualLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("line count mismatch: got %d %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOSC_BELTerminator(t *testing.T) {
	// xterm 风格窗口标题：ESC ] 0 ; title BEL，后面应继续显示 hello
	got := feed(t, "\x1b]0;my title\x07hello")
	mustEqualLines(t, got, []string{"hello"})
}

func TestOSC_STTerminator(t *testing.T) {
	got := feed(t, "\x1b]0;my title\x9chello")
	mustEqualLines(t, got, []string{"hello"})
}

func TestOSC_ESCBackslashTerminator(t *testing.T) {
	got := feed(t, "\x1b]0;my title\x1b\\hello")
	mustEqualLines(t, got, []string{"hello"})
}

func TestESC_CharsetSelectIsConsumed(t *testing.T) {
	// ESC ( B 选择 G0 = ASCII，B 不应作为字符出现
	got := feed(t, "\x1b(Bhi")
	mustEqualLines(t, got, []string{"hi"})
}

func TestESC_SaveRestoreCursor(t *testing.T) {
	// ESC 7 (DECSC) 保存光标、ESC 8 (DECRC) 恢复：
	// "a" 后保存（col 1），写 "b"，恢复后写 "c" 应覆盖掉 "b"
	got := feed(t, "a\x1b7b\x1b8c")
	mustEqualLines(t, got, []string{"ac"})
}

func TestBackspaceMovesCursorWithoutDeleting(t *testing.T) {
	// 写 abc，BS 两次，再写 X —— 标准 BS 语义下应得到 "aXc"
	got := feed(t, "abc\b\bX")
	mustEqualLines(t, got, []string{"aXc"})
}

func TestBackspaceSpaceErasePattern(t *testing.T) {
	// bash 重绘提示符常用模式：写字、BS、空格 —— 字符应被空格覆盖
	got := feed(t, "abc\b ")
	mustEqualLines(t, got, []string{"ab"})
}

func TestCursorPositionThenWrite(t *testing.T) {
	// 把光标移到第 1 行第 5 列再写 X，前面应有空格补齐
	got := feed(t, "\x1b[1;5HX")
	mustEqualLines(t, got, []string{"    X"})
}

func TestCRLF(t *testing.T) {
	got := feed(t, "line1\r\nline2")
	if len(got) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(got), got)
	}
	if got[len(got)-2] != "line1" || got[len(got)-1] != "line2" {
		t.Errorf("CRLF handling broken: %q", got)
	}
}

func TestEraseBelowClearsCursorRowRight(t *testing.T) {
	// 写 "hello"，光标回到第 3 列，J 0 应清掉 "llo"
	v := New()
	v.Advance([]byte("hello"))
	v.Advance([]byte("\x1b[1;3H")) // 移到第 1 行第 3 列
	v.Advance([]byte("\x1b[0J"))   // 从光标到屏幕末尾
	out := v.Output()
	if len(out) != 1 || out[0] != "he" {
		t.Errorf("got %q, want [he]", out)
	}
}

func TestEraseAboveClearsCursorRowLeft(t *testing.T) {
	v := New()
	v.Advance([]byte("hello"))
	v.Advance([]byte("\x1b[1;3H")) // 移到第 1 行第 3 列（光标在 'l' 上）
	v.Advance([]byte("\x1b[1J"))   // 从屏幕开头到光标
	out := v.Output()
	if len(out) != 1 {
		t.Fatalf("got %d lines, want 1: %q", len(out), out)
	}
	// eraseLeft 把 [0..index) 清掉，剩下 "lo" 之类，至少不应保留 "hel"
	if strings.Contains(out[0], "h") || strings.Contains(out[0], "e") {
		t.Errorf("erase above did not clear chars before cursor: %q", out)
	}
}

func TestEraseLeftCursorPastRowEndDoesNotPanic(t *testing.T) {
	// 复现：先把光标定位到空行的中段（行 data 长度为 0、index>0），再发 EL 1
	// 老实现里 r.data[r.index:] 会触发 slice bounds out of range。
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("eraseLeft panicked when cursor past row end: %v", r)
		}
	}()
	v := New()
	v.Advance([]byte("\x1b[1;6H")) // 移到第 1 行第 6 列，当前行 data 仍为空
	v.Advance([]byte("\x1b[1K"))   // EL 1：从行首到光标
	_ = v.Output()
}

func TestVPositionRelativeIsRelative(t *testing.T) {
	// VPR (CSI Pn e) 应该向下移 N 行且保持 col。
	// 写 "abc"，光标在第 0 行第 3 列；CSI 1e 后应到第 1 行第 3 列；写 X 应得到 "abc\n   X"
	v := New()
	v.Advance([]byte("abc"))
	v.Advance([]byte("\x1b[1e"))
	v.Advance([]byte("X"))
	out := v.Output()
	if len(out) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(out), out)
	}
	if out[0] != "abc" {
		t.Errorf("row 0: got %q want %q", out[0], "abc")
	}
	if out[1] != "   X" {
		t.Errorf("row 1: got %q want %q", out[1], "   X")
	}
}

func TestHPositionRelativeIsRelative(t *testing.T) {
	// HPR (CSI Pn a) 向右移 N 列。
	v := New()
	v.Advance([]byte("ab"))
	v.Advance([]byte("\x1b[3a")) // 向右 3 列
	v.Advance([]byte("Z"))
	out := v.Output()
	if len(out) != 1 || out[0] != "ab   Z" {
		t.Errorf("got %q, want [ab   Z]", out)
	}
}

func TestCursorMoveZeroParamTreatedAsOne(t *testing.T) {
	// CSI 0A 在 VT100 规范中应当作 CSI 1A
	v := New()
	v.Advance([]byte("a\nb"))    // 第 0 行 "a"，第 1 行 "b"，光标在第 1 行第 1 列
	v.Advance([]byte("\x1b[0A")) // CSI 0A：上移，按规范当 1
	v.Advance([]byte("X"))
	out := v.Output()
	// 光标应在第 0 行第 1 列，覆盖时把空格补到列 1（"a" 之后无字符），再写 X
	if len(out) < 1 {
		t.Fatalf("no output")
	}
	if out[0] != "aX" {
		t.Errorf("got row0 %q, want %q", out[0], "aX")
	}
}

func TestSetScrollRegionParsesBottom(t *testing.T) {
	// 仅验证不 panic，且解析多参数不抛异常（原实现 getNumberOrDefault 拿不到 bottom 但被掩盖）
	v := New()
	v.Advance([]byte("a\nb\nc\n"))
	v.Advance([]byte("\x1b[1;3r")) // 设置滚动区 1..3
	v.Advance([]byte("X"))
	// 设置滚动区后光标移到 home，写 X 覆盖第一行第一列
	out := v.Output()
	if len(out) < 1 || out[0] == "" || out[0][0] != 'X' {
		t.Errorf("expected first row to start with X, got %q", out)
	}
}

// ---------- LineHandler ----------

func TestLineHandlerFiresOnLF(t *testing.T) {
	var got []string
	v := NewWithOptions(WithLineHandler(func(e LineEvent) {
		got = append(got, e.Line)
	}))
	v.Advance([]byte("first\nsecond\nthird"))
	// "first" 和 "second" 各自被 LF 提交；"third" 没 LF 不会触发
	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(got), got)
	}
	if got[0] != "first" || got[1] != "second" {
		t.Errorf("unexpected lines: %q", got)
	}
}

func TestLineHandlerCRLF(t *testing.T) {
	var got []string
	v := NewWithOptions(WithLineHandler(func(e LineEvent) {
		got = append(got, e.Line)
	}))
	v.Advance([]byte("ls -la\r\necho hi\r\n"))
	if len(got) != 2 {
		t.Fatalf("got %d lines: %q", len(got), got)
	}
	if got[0] != "ls -la" || got[1] != "echo hi" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestLineHandlerMergesSoftWrap(t *testing.T) {
	// cols=10，写一条 25 个字符的行，再 LF——应该合并为 25 字符的逻辑行
	var got []string
	v := NewWithOptions(
		WithCols(10),
		WithLineHandler(func(e LineEvent) { got = append(got, e.Line) }),
	)
	long := strings.Repeat("x", 25)
	v.Advance([]byte(long + "\n"))
	if len(got) != 1 {
		t.Fatalf("got %d, want 1: %q", len(got), got)
	}
	if got[0] != long {
		t.Errorf("merged line wrong: got %q want %q", got[0], long)
	}
}

func TestLineHandlerCalledAfterUnlock(t *testing.T) {
	// 在 handler 里再调一次 Advance 不应死锁
	v := NewWithOptions()
	var n atomic.Int32
	var ref VirtualTerminal
	v2 := NewWithOptions(WithLineHandler(func(_ LineEvent) {
		if n.Add(1) == 1 {
			// 在 handler 里复用同一个虚拟终端写入
			ref.Advance([]byte("inner\n"))
		}
	}))
	ref = v2
	_ = v
	v2.Advance([]byte("outer\n"))
	if got := n.Load(); got != 2 {
		t.Errorf("expected 2 callbacks, got %d", got)
	}
}

// ---------- Cols / 软 wrap ----------

func TestColsSoftWrapsToNewRow(t *testing.T) {
	v := NewWithOptions(WithCols(5))
	v.Advance([]byte("abcdefg")) // 5 个字符后软 wrap
	out := v.Output()
	if len(out) != 2 {
		t.Fatalf("got %d rows, want 2: %q", len(out), out)
	}
	if out[0] != "abcde" || out[1] != "fg" {
		t.Errorf("got %q", out)
	}
}

func TestColsZeroDisablesWrap(t *testing.T) {
	// 默认行为：不限列宽
	v := NewWithOptions()
	v.Advance([]byte(strings.Repeat("x", 200)))
	out := v.Output()
	if len(out) != 1 {
		t.Errorf("expected 1 row, got %d", len(out))
	}
}

// ---------- Private mode tracking ----------

func TestPrivateModeBracketedPaste(t *testing.T) {
	v := New()
	if v.IsPrivateModeSet(2004) {
		t.Fatal("bracketed paste should not be set initially")
	}
	v.Advance([]byte("\x1b[?2004h"))
	if !v.IsPrivateModeSet(2004) {
		t.Errorf("bracketed paste should be set after CSI ?2004h")
	}
	v.Advance([]byte("\x1b[?2004l"))
	if v.IsPrivateModeSet(2004) {
		t.Errorf("bracketed paste should be cleared after CSI ?2004l")
	}
}

func TestPrivateModeAltScreen(t *testing.T) {
	v := New()
	v.Advance([]byte("\x1b[?1049h"))
	if !v.IsPrivateModeSet(1049) {
		t.Errorf("alt screen mode should be set")
	}
}

func TestPrivateModeMultiParam(t *testing.T) {
	v := New()
	v.Advance([]byte("\x1b[?25;2004h")) // 同时启用 cursor visible + bracketed paste
	if !v.IsPrivateModeSet(25) || !v.IsPrivateModeSet(2004) {
		t.Errorf("both modes should be set; 25=%v 2004=%v",
			v.IsPrivateModeSet(25), v.IsPrivateModeSet(2004))
	}
}

func TestIsAltScreen(t *testing.T) {
	v := New()
	if IsAltScreen(v) {
		t.Fatal("should not be in alt screen initially")
	}
	v.Advance([]byte("\x1b[?1049h"))
	if !IsAltScreen(v) {
		t.Errorf("?1049h should mark alt screen")
	}
	v.Advance([]byte("\x1b[?1049l"))
	if IsAltScreen(v) {
		t.Errorf("?1049l should leave alt screen")
	}
	// 老式 ?47/?1047 也应被识别
	v.Advance([]byte("\x1b[?47h"))
	if !IsAltScreen(v) {
		t.Errorf("?47h should mark alt screen")
	}
	v.Advance([]byte("\x1b[?47l"))
	v.Advance([]byte("\x1b[?1047h"))
	if !IsAltScreen(v) {
		t.Errorf("?1047h should mark alt screen")
	}
}

func TestLineHandlerSkipsAltScreenContent(t *testing.T) {
	// 集成场景：alt screen 中即使出现 prompt-like 行，也应被快照里的 IsAltScreen 过滤
	var seen []string

	v := NewWithOptions(WithLineHandler(func(e LineEvent) {
		if e.Modes.IsAltScreen() {
			return
		}
		seen = append(seen, e.Line)
	}))

	chunks := []string{
		"$ first\r\n",
		"\x1b[?1049h",                  // 进入 alt screen
		"$ fake_inside_alt_screen\r\n", // 应被跳过
		"\x1b[?1049l",                  // 退出 alt screen
		"$ second\r\n",
	}
	for _, c := range chunks {
		v.Advance([]byte(c))
	}

	if len(seen) != 2 {
		t.Fatalf("expected 2 visible lines, got %d: %q", len(seen), seen)
	}
	if seen[0] != "$ first" || seen[1] != "$ second" {
		t.Errorf("got %q, want [$ first, $ second]", seen)
	}
}

// TestLineEventModesSnapshot 验证：硬 LF 时携带的 Modes 是提交那一瞬间的快照，
// 即使 Advance 在同一段输入里随后改变了模式（如 \r\n\x1b[?2004l），handler
// 拿到的仍是提交时的状态。这是修 bracketed-paste prompt 识别 race 的关键能力。
func TestLineEventModesSnapshot(t *testing.T) {
	var events []LineEvent
	v := NewWithOptions(WithLineHandler(func(e LineEvent) {
		events = append(events, e)
	}))

	// 模拟 bash readline 序列：先开 ?2004h、写 prompt + 命令、\r\n、再 ?2004l
	v.Advance([]byte("\x1b[?2004hroot@host:~# ls -la\r\n\x1b[?2004l"))

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if !events[0].Modes.IsSet(2004) {
		t.Errorf("snapshot at commit should have ?2004 set; got false")
	}
	// 此时实时查询应该是 false（已经 ?2004l）
	if v.IsPrivateModeSet(2004) {
		t.Errorf("real-time mode should be cleared after ?2004l")
	}
}

func TestPrivateModeResetCleared(t *testing.T) {
	v := New()
	v.Advance([]byte("\x1b[?2004h"))
	v.Reset()
	if v.IsPrivateModeSet(2004) {
		t.Errorf("Reset should clear private modes")
	}
}

// TestOSCHandlerCapturesPayload 验证 OSC 序列识别后 payload 被回调出来。
func TestOSCHandlerCapturesPayload(t *testing.T) {
	var got []string
	v := NewWithOptions(WithOSCHandler(func(payload string) {
		got = append(got, payload)
	}))

	// 三种终止符（BEL / ST / ESC \）都应当工作
	v.Advance([]byte("\x1b]1337;CurrentDir=/tmp\x07"))  // BEL
	v.Advance([]byte("\x1b]7;file://host/var/log\x9c")) // ST
	v.Advance([]byte("\x1b]0;window title\x1b\\"))      // ESC \

	want := []string{
		"1337;CurrentDir=/tmp",
		"7;file://host/var/log",
		"0;window title",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestResetScreenKeepsModes 验证 ResetScreen 只清屏幕、保留私有模式——
// 这是给 outputRecognizer 在每条命令后回收内存而不丢 ?2004h 状态用的。
func TestResetScreenKeepsModes(t *testing.T) {
	v := New()
	v.Advance([]byte("\x1b[?2004hroot@host:~# ls\r\n"))
	v.Advance([]byte("\x1b[?1049h"))

	v.ResetScreen()

	if !v.IsPrivateModeSet(2004) {
		t.Errorf("ResetScreen should keep ?2004 mode")
	}
	if !v.IsPrivateModeSet(1049) {
		t.Errorf("ResetScreen should keep ?1049 mode")
	}
	if out := v.Output(); len(out) != 0 {
		t.Errorf("ResetScreen should empty Output; got %q", out)
	}
}

// ---------- 集成场景：在 SSH 风格输出里识别命令 ----------

func TestLineHandlerIdentifiesCommandUnderPrompt(t *testing.T) {
	// 模拟 bash：每条命令前都有 "user@host$ "，命令后有 LF + 输出 + LF
	var lines []string
	v := NewWithOptions(WithLineHandler(func(e LineEvent) {
		lines = append(lines, e.Line)
	}))
	v.Advance([]byte("user@host$ ls -la\r\nfile1\r\nfile2\r\nuser@host$ pwd\r\n/home/user\r\n"))

	wantLines := []string{
		"user@host$ ls -la",
		"file1",
		"file2",
		"user@host$ pwd",
		"/home/user",
	}
	if len(lines) != len(wantLines) {
		t.Fatalf("got %d lines, want %d: %q", len(lines), len(wantLines), lines)
	}
	for i := range lines {
		if lines[i] != wantLines[i] {
			t.Errorf("line %d: got %q want %q", i, lines[i], wantLines[i])
		}
	}
}

func TestAdvanceConcurrentSafe(t *testing.T) {
	// 不应触发 race detector / panic
	v := New()
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 100 {
				v.Advance([]byte("hello\r\n"))
			}
		})
	}
	wg.Wait()
	_ = v.Output()
}

// ---------- 跨 Advance 分包（TCP 分块场景）----------

func TestSplitChunkCSI(t *testing.T) {
	// CSI 序列被切成两个 Advance：参数和 final byte 分属两个 chunk。
	// 解析状态必须跨调用保持——ESC[2D 是"左移2格"，'b' 应写到行首覆盖 'a'。
	v := New()
	v.Advance([]byte("a\x1b[2"))
	v.Advance([]byte("Db"))
	mustEqualLines(t, v.Output(), []string{"b"})
}

func TestSplitChunkOSC(t *testing.T) {
	// OSC payload 被切成两个 Advance：payload 应完整拼出，不污染屏幕
	var got []string
	v := NewWithOptions(WithOSCHandler(func(payload string) { got = append(got, payload) }))
	v.Advance([]byte("\x1b]0;ti"))
	v.Advance([]byte("tle\x07after"))
	if len(got) != 1 || got[0] != "0;title" {
		t.Errorf("osc payload: got %q, want [0;title]", got)
	}
	mustEqualLines(t, v.Output(), []string{"after"})
}

func TestSplitChunkDCS(t *testing.T) {
	// DCS 序列（如 tmux 的 ESC Ptmux;...ESC\）跨 chunk 应被完整消费
	v := New()
	v.Advance([]byte("a\x1bPtmux;"))
	v.Advance([]byte("stuff\x1b\\b"))
	mustEqualLines(t, v.Output(), []string{"ab"})
}

// ---------- 修复回归：ECH / CNL / CPL / EL1 / ESC D E ----------

func TestECHOverwritesWithSpaces(t *testing.T) {
	// ECH (CSI X) 用空格覆盖，不删除移位："abcdef" 光标到第2列，ECH 2 → "a  def"
	v := New()
	v.Advance([]byte("abcdef"))
	v.Advance([]byte("\x1b[1;2H"))
	v.Advance([]byte("\x1b[2X"))
	mustEqualLines(t, v.Output(), []string{"a  def"})
}

func TestCNLReturnsToColumnZero(t *testing.T) {
	// 在已有内容的行之间跳转时，CNL 必须回到行首
	v := New()
	v.Advance([]byte("abc\ndef"))
	v.Advance([]byte("\x1b[1A")) // 上移到第 0 行，col 仍是 3
	v.Advance([]byte("\x1b[1E")) // CNL：下一行行首
	v.Advance([]byte("X"))
	mustEqualLines(t, v.Output(), []string{"abc", "Xef"})
}

func TestCPLReturnsToColumnZero(t *testing.T) {
	v := New()
	v.Advance([]byte("abc\ndef"))
	v.Advance([]byte("\x1b[1F")) // CPL：上一行行首
	v.Advance([]byte("X"))
	mustEqualLines(t, v.Output(), []string{"Xbc", "def"})
}

func TestEraseLeftIncludesCursor(t *testing.T) {
	// EL 1 擦除范围含光标位："abcdef" 光标在第 2 列('c')，EL1 → "def"
	v := New()
	v.Advance([]byte("abcdef"))
	v.Advance([]byte("\x1b[1;3H"))
	v.Advance([]byte("\x1b[1K"))
	mustEqualLines(t, v.Output(), []string{"def"})
}

func TestESCIndexIsHardLineBreak(t *testing.T) {
	// ESC D (IND) 是硬换行，应触发 LineHandler 且不回行首
	var got []string
	v := NewWithOptions(WithLineHandler(func(e LineEvent) { got = append(got, e.Line) }))
	v.Advance([]byte("ab\x1bDcd"))
	if len(got) != 1 || got[0] != "ab" {
		t.Errorf("line events: got %q, want [ab]", got)
	}
	mustEqualLines(t, v.Output(), []string{"ab", "cd"})
}

func TestESCNextLineCommitsAndCr(t *testing.T) {
	// ESC E (NEL)：硬换行 + 回行首
	var got []string
	v := NewWithOptions(WithLineHandler(func(e LineEvent) { got = append(got, e.Line) }))
	v.Advance([]byte("ab\x1bEcd"))
	if len(got) != 1 || got[0] != "ab" {
		t.Errorf("line events: got %q, want [ab]", got)
	}
	mustEqualLines(t, v.Output(), []string{"ab", "cd"})
}

func TestESCRISFullReset(t *testing.T) {
	// ESC c (RIS) 完全复位：屏幕 + 私有模式都清掉
	v := New()
	v.Advance([]byte("\x1b[?2004hhello"))
	v.Advance([]byte("\x1bc"))
	if v.IsPrivateModeSet(2004) {
		t.Errorf("RIS should clear private modes")
	}
	if out := v.Output(); len(out) != 0 {
		t.Errorf("RIS should clear screen; got %q", out)
	}
}

func TestDCHAndICHParamZeroIsOne(t *testing.T) {
	// 参数 0 按规范视为 1
	v := New()
	v.Advance([]byte("abc"))
	v.Advance([]byte("\x1b[1;2H")) // 光标在 'b'
	v.Advance([]byte("\x1b[0P"))   // DCH 0 → 1：删掉 'b' → "ac"
	mustEqualLines(t, v.Output(), []string{"ac"})
}

// ---------- Tab 制表位 ----------

func TestTabMovesToNextTabStop(t *testing.T) {
	// \t 移到下一个 8 列制表位，tab 本身不进入行数据
	v := New()
	v.Advance([]byte("a\tb"))
	mustEqualLines(t, v.Output(), []string{"a       b"})
}

func TestTabAtLineStart(t *testing.T) {
	v := New()
	v.Advance([]byte("\tX"))
	mustEqualLines(t, v.Output(), []string{"        X"})
}

func TestCHTAndCBT(t *testing.T) {
	v := New()
	v.Advance([]byte("\x1b[2I")) // CHT 2：到第 16 列
	v.Advance([]byte("X"))
	want := strings.Repeat(" ", 16) + "X"
	if got := v.Output()[0]; got != want {
		t.Errorf("CHT: got %q, want %q", got, want)
	}
	v.Advance([]byte("\x1b[2Z")) // CBT 2：光标在 17，依次回退 16、8
	v.Advance([]byte("Y"))
	got := v.Output()[0]
	if got[8] != 'Y' {
		t.Errorf("CBT: got %q, want Y at col 8", got)
	}
}

func TestHTSCustomTabStop(t *testing.T) {
	v := New()
	v.Advance([]byte("abcd"))  // 光标在第 4 列
	v.Advance([]byte("\x1bH")) // HTS：在第 4 列设制表位
	v.Advance([]byte("\b\b"))  // 回到第 2 列
	v.Advance([]byte("\t"))    // 下一个制表位是自定义的第 4 列（比默认 8 近）
	v.Advance([]byte("X"))
	mustEqualLines(t, v.Output(), []string{"abcdX"})
}

func TestTBCClearsAllStops(t *testing.T) {
	v := New()
	v.Advance([]byte("\x1b[3g")) // TBC 3：清除全部制表位（含默认网格）
	v.Advance([]byte("\t"))      // 无处可去，原地不动
	v.Advance([]byte("X"))
	mustEqualLines(t, v.Output(), []string{"X"})
}

func TestResetRestoresTabStops(t *testing.T) {
	v := New()
	v.Advance([]byte("\x1b[3g"))
	v.Reset()
	v.Advance([]byte("\tX"))
	if got := v.Output()[0]; got[8] != 'X' {
		t.Errorf("after Reset tab stops should be default grid; got %q", got)
	}
}

// ---------- 宽字符（CJK）显示宽度 ----------

func TestWideCharWrapByDisplayWidth(t *testing.T) {
	// cols=4："中a文"——中(2)+a(1)=3 列，文需要 2 列放不下 → 软 wrap
	v := NewWithOptions(WithCols(4))
	v.Advance([]byte("中a文"))
	out := v.Output()
	if len(out) != 2 || out[0] != "中a" || out[1] != "文" {
		t.Errorf("got %q, want [中a 文]", out)
	}
}

func TestWideCharLogicalLineMerge(t *testing.T) {
	// 软 wrap 的中文行应合并为一条逻辑行
	var got []string
	v := NewWithOptions(WithCols(4), WithLineHandler(func(e LineEvent) { got = append(got, e.Line) }))
	v.Advance([]byte("中文测试\n"))
	if len(got) != 1 || got[0] != "中文测试" {
		t.Errorf("got %q, want [中文测试]", got)
	}
}

func TestWideCharFitsExactBoundary(t *testing.T) {
	// cols=6："中文中" 恰好占满 6 列，不应 wrap
	v := NewWithOptions(WithCols(6))
	v.Advance([]byte("中文中"))
	if out := v.Output(); len(out) != 1 || out[0] != "中文中" {
		t.Errorf("got %q", out)
	}
}

func TestWideCharOverwriteSecondHalfClearsLead(t *testing.T) {
	// 光标落进宽字符占位列再写普通字符：宽字符清成空格（xterm 语义）。
	// 文本层宽字符是 1 字符占 2 列：可见列为 中(0-1)、空格(2)、X(3)，
	// 提取文本应为 "中 X"（占位空格不进入文本，与终端复制行为一致）。
	v := New()
	v.Advance([]byte("中文")) // data=[中,' ',文,' ']，光标列 4
	v.Advance([]byte("\b"))  // 光标到"文"的占位列（列 3）
	v.Advance([]byte("X"))   // "文" 清成空格，X 写在列 3
	if got := v.Output()[0]; got != "中 X" {
		t.Errorf("got %q, want %q", got, "中 X")
	}
}

func TestWideCharTrailingTrim(t *testing.T) {
	// 行尾宽字符的占位空格应被 String 的尾部裁剪处理掉
	v := New()
	v.Advance([]byte("中文"))
	if out := v.Output(); len(out) != 1 || out[0] != "中文" {
		t.Errorf("got %q, want [中文]", out)
	}
}

func TestTabFromColumnAdjacentToStop(t *testing.T) {
	// 回归：光标在第 15 列时，下一个制表位是 16，而不是 24。
	// 文本层宽字符是 1 字符：列 0-14 提取为 "$ echo 中文目录"，列 15 空格，X 在列 16。
	v := New()
	v.Advance([]byte("$ echo 中文目录")) // 恰好 15 列
	v.Advance([]byte("\tX"))
	mustEqualLines(t, v.Output(), []string{"$ echo 中文目录 X"})
}
