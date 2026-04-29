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

func TestESC_SaveRestoreCursorIsConsumed(t *testing.T) {
	// ESC 7 / ESC 8 是 0 参数 ESC 序列，应被消费而不会把 7/8 当成屏幕字符。
	// 注意：本实现未真的保存/恢复光标，只验证序列字节被吃掉。
	got := feed(t, "a\x1b7b\x1b8c")
	mustEqualLines(t, got, []string{"abc"})
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
	v.Advance([]byte("a\nb"))      // 第 0 行 "a"，第 1 行 "b"，光标在第 1 行第 1 列
	v.Advance([]byte("\x1b[0A"))    // CSI 0A：上移，按规范当 1
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
	v := NewWithOptions(WithLineHandler(func(line string) {
		got = append(got, line)
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
	v := NewWithOptions(WithLineHandler(func(line string) {
		got = append(got, line)
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
		WithLineHandler(func(line string) { got = append(got, line) }),
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
	v2 := NewWithOptions(WithLineHandler(func(line string) {
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
	// 集成场景：alt screen 中即使出现 prompt-like 行，也应被 IsAltScreen 过滤
	var v VirtualTerminal
	var seen []string

	v = NewWithOptions(WithLineHandler(func(line string) {
		if IsAltScreen(v) {
			return
		}
		seen = append(seen, line)
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

func TestPrivateModeResetCleared(t *testing.T) {
	v := New()
	v.Advance([]byte("\x1b[?2004h"))
	v.Reset()
	if v.IsPrivateModeSet(2004) {
		t.Errorf("Reset should clear private modes")
	}
}

// ---------- 集成场景：在 SSH 风格输出里识别命令 ----------

func TestLineHandlerIdentifiesCommandUnderPrompt(t *testing.T) {
	// 模拟 bash：每条命令前都有 "user@host$ "，命令后有 LF + 输出 + LF
	var lines []string
	v := NewWithOptions(WithLineHandler(func(line string) {
		lines = append(lines, line)
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
