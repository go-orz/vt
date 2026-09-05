package vt

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

const (
	_BEL rune = 0x07 // Bell (Caret = ^G, C = \a)
	_BS  rune = 0x08 // Backspace (Caret = ^H, C = \b)
	_HT  rune = 0x09 // Horizontal Tab (Caret = ^I, C = \t)
	_LF  rune = 0x0a // Line Feed (Caret = ^J, C = \n)
	_VT  rune = 0x0b // Vertical Tab (Caret = ^K, C = \v)
	_FF  rune = 0x0c // Form Feed (Caret = ^L, C = \f)
	_CR  rune = 0x0d // Carriage Return (Caret = ^M, C = \r)
	_SO  rune = 0x0e // Shift Out
	_SI  rune = 0x0f // Shift In

	_ESC rune = 0x1b // Escape (Caret = ^[, C = \e)

	space rune = 0x20 // 空格

	// defaultTabStop 默认制表位网格间隔（每 8 列一个制表位，与 xterm 一致）。
	defaultTabStop = 8
)

// maxStringDataSize 限制单个 OSC/DCS 等 string sequence 的 payload 大小，
// 防止畸形超长序列撑爆内存。1MB 远超正常 OSC（标题、CurrentDir 等）的体积。
const maxStringDataSize = 1024 * 1024

// maxInsertCells 限制 ICH (CSI @) 单次插入的空白单元数——cols==0 没有行宽
// 可钳制时生效，防止 12 字节的畸形序列（如 CSI 2147483647 @）放大成
// 天文数字级的内存分配与循环。cols>0 时以 cols 为准。
const maxInsertCells = 1 << 20

// maxScreenDim 钳制行/列位置跳转（CUP/CUD/VPA/CHA/CUF 等）的上界。
// termios winsize 的行列都是 uint16，真实终端屏幕不会超过 65535 行/列，
// 更大的定位参数只会来自畸形或恶意输出。钳制规则是"不超过已有内容与该
// 上界的较大者"：屏幕内的正常跳转不受影响，LF 驱动的滚动增长不设限，
// 而巨型跳转（如 CSI 2147483646;1H）单次最多物化 65537 个空行，重复
// 发送会被已有内容长度卡住，不会持续放大内存。
const maxScreenDim = 1 << 16

// maxRowRunes 钳制单个物理行的最大长度（cols==0 不软 wrap 时生效）。
// 无换行的超长输出（如 cat 单行大文件、进度条回车重绘前的意外长行）
// 会让一行的 data 无限增长；达到上限后强制软 wrap 到新物理行，并标记
// wrappedFromPrev——commitLogicalLine 仍会把各段拼回完整逻辑行，识别
// 语义无损，只是物理行内存有界。cols>0 时每行已被列宽限制，不受影响。
const maxRowRunes = 1 << 20

// screenState 保存一块屏幕缓冲的状态：行窗口与光标所在行索引。
// main 与 alt 是两块独立的屏（参考终端模拟器的 scrs[2] 设计）——
// alt screen 内 TUI 应用的重绘只影响 alt 自己的行，退出时 main 原样恢复。
type screenState struct {
	rowList []*Row // 行数据
	rows    int    // 当前行索引（0-based，相对本屏窗口）
}

type inputHandler func(params []rune) error

// LineEvent 是每次硬 LF 时 LineHandler 收到的事件。
//
// Modes 是该行被提交那一瞬间的私有模式快照副本——回调可以放心查询历史
// 状态（例如 "提交此行时是否还在 bracketed paste ?2004h"），不会受到
// Advance 处理后续字节时模式变化的影响。
//
// 直接查 vt.IsPrivateModeSet 拿到的是回调时刻的最新值，对很多审计/识别
// 场景是错的——一段输入里 \r\n 之后通常紧跟 ?2004l（关闭 readline），
// 解锁回调时已经看不到 ?2004h 了。
type LineEvent struct {
	Line  string
	Modes ModeSnapshot
	// CmdText 是 shell integration（OSC 133）提供的结构化命令文本：
	// shell 发 133;A 标记 prompt 开始、133;B 标记命令输入区开始（B 时刻
	// 光标列即 prompt 宽度），用户按 Enter 提交行时，B 到行尾的文本就是
	// 精确命令——含 Tab 补全与 history 回退后的最终形态，无需 prompt
	// 正则猜测。非 133 流（shell 未装 integration）为空串，调用方应退回
	// Modes + 正则识别；多行续行命令无法单行切分时同样为空串。
	CmdText string
}

// ModeSnapshot 是一组私有模式状态的快照副本。零值表示空快照（所有模式未设置）。
type ModeSnapshot map[int]bool

// IsSet 查询私有模式 n 是否在快照中为 true。
func (s ModeSnapshot) IsSet(n int) bool { return s[n] }

// IsAltScreen 返回快照里是否处于 alt screen（?1049 / ?1047 / ?47 任一为真）。
func (s ModeSnapshot) IsAltScreen() bool {
	return s[1049] || s[1047] || s[47]
}

// LineHandler 在每次硬 LF 时回调一次。参数是被提交的"逻辑行"事件——
// 多段软 wrap 行会被合并为一条逻辑 Line；Modes 是提交时刻的私有模式快照。
type LineHandler func(LineEvent)

type VirtualTerminal interface {
	Advance(p []byte)
	Output() []string
	// Reset 重置全部状态：屏幕缓冲、光标、私有模式、待回放的行事件。
	Reset()
	// ResetScreen 仅重置屏幕缓冲与光标，保留私有模式与待回放事件——
	// 适合"长会话定期清屏"等场景：调用方希望释放内存但维持 ?2004 / alt
	// screen 等会话级状态，避免清掉后下一条 prompt 行的 mode 快照不正确。
	ResetScreen()
	// IsPrivateModeSet 查询 DEC 私有模式（如 25 光标可见、1049 alt screen、2004 bracketed paste）。
	IsPrivateModeSet(n int) bool
}

// Opt is a function that configures a virtualTerminal.
type Opt func(*virtualTerminal)

// WithLogger sets a logger for the virtual terminal.
func WithLogger(logger *log.Logger) Opt {
	return func(vt *virtualTerminal) {
		vt.logger = logger
	}
}

// WithCols 设置终端列宽。开启后字符越过列宽会触发软 wrap，
// 但 LineHandler 仍按硬 LF 切分逻辑行——多段软 wrap 会被合并。
// 默认 0 表示不限制（不 wrap，每行可无限拉长）。
func WithCols(cols int) Opt {
	return func(vt *virtualTerminal) {
		if cols > 0 {
			vt.cols = cols
		}
	}
}

// WithMaxLines 设置单块屏幕行窗口的上限（scrollback + 可见区的总行数）。
// 超限时最老的行从窗口头部被驱逐——与整屏清空不同，光标行与近期行始终
// 保留，正在编辑的 prompt 行不受影响，行提交回调照常工作。
// main 与 alt 两块屏各自独立计算上限。默认 0 表示无界（兼容旧行为）。
func WithMaxLines(n int) Opt {
	return func(vt *virtualTerminal) {
		if n > 0 {
			vt.maxLines = n
		}
	}
}

// WithLineHandler 注册行提交回调。每次硬 LF 触发一次，
// 回调在 Advance 释放内部锁之后同步执行，避免在锁内回调阻塞。
func WithLineHandler(h LineHandler) Opt {
	return func(vt *virtualTerminal) {
		vt.lineHandler = h
	}
}

// OSCHandler 在每个完整的 OSC（Operating System Command）序列识别后被调用。
// payload 是 ESC ] 与终止符（BEL 或 ESC \）之间的文本。
//
// 例：终端收到 "\x1b]1337;CurrentDir=/tmp\x07"，OSCHandler 收到 payload
// 字符串 "1337;CurrentDir=/tmp"。
//
// 典型用途：识别 iTerm2 风格 OSC 1337 / VTE 风格 OSC 7 上报当前目录。
type OSCHandler func(payload string)

// WithOSCHandler 注册 OSC 序列回调。回调在 Advance 处理 OSC 时同步执行
// （持有内部写锁），handler 应当轻量、不在内部再调 Advance 等取锁的方法。
func WithOSCHandler(h OSCHandler) Opt {
	return func(vt *virtualTerminal) {
		vt.oscHandler = h
	}
}

func New() VirtualTerminal {
	return NewWithOptions()
}

func NewWithOptions(opts ...Opt) VirtualTerminal {
	vt := &virtualTerminal{
		inputHandlers: make(map[byte]inputHandler),
		privateModes:  make(map[int]bool),
		logger:        nil,
	}
	vt.screenState = &vt.mainScr
	for _, opt := range opts {
		opt(vt)
	}
	vt.resetTabStops()
	vt.initCsiHandler()
	vt.initParser()
	return vt
}

type virtualTerminal struct {
	sync.RWMutex
	// screenState 是当前活动屏（嵌入指针，rowList/rows 字段被提升），
	// ?1049/?1047/?47 切屏时换成另一块的指针——所有通过 vt.rowList /
	// vt.rows 的既有访问自动作用于活动屏。
	*screenState
	mainScr  screenState // 主屏：shell 交互与滚动历史
	altScr   screenState // alt 屏：vim/less/top 等 TUI 的独立缓冲
	cols     int         // 列宽，0 表示不限
	maxLines int         // 单屏行窗口上限，0 表示无界

	parser *ansi.Parser // 跨 Advance 持久保存解析状态的 ANSI 字节流状态机

	inputHandlers map[byte]inputHandler
	privateModes  map[int]bool

	lineHandler  LineHandler
	pendingLines []pendingLine // commitLogicalLine 在锁内追加，Advance 解锁后回放给 lineHandler

	oscHandler OSCHandler

	tabstops        map[int]bool // 显式设置的制表位（HTS 添加、TBC 移除）
	clearedTabStops map[int]bool // 被 TBC 0 清除的默认网格位（nextTab/prevTab 跳过）
	defaultTabGrid  bool         // 是否启用每 8 列的默认制表位网格（TBC 3 关闭）

	hasSavedCursor bool // ESC 7 保存的光标状态
	savedRow       int
	savedCol       int

	// ?1049 专用的光标保存（DECSC 与 1049 是两套独立机制）
	altSavedRow    int
	altSavedCol    int
	inAltScreen    bool // 当前是否处于 alt 屏（与 privateModes 位同步维护）

	// OSC 133 命令输入区状态（shell integration 语义）：
	// 133;B 时光标位于命令输入起点（其左方即 prompt），快照 (row, col)
	// 后，该逻辑行提交时从 col 起切出的文本就是精确命令；133;C（命令
	// 输出开始）或 133;A/D（新循环/结束）关闭区域。
	cmdZoneRow    int
	cmdZoneCol    int
	cmdZoneActive bool

	logger *log.Logger
}

// pendingLine 记录一条尚未交给 LineHandler 的提交事件。
type pendingLine struct {
	line    string
	modes   ModeSnapshot
	cmdText string
}

// IsAltScreen 检查终端是否处于 alt screen 缓冲区。
// vim、nano、less、man、tmux、htop 等 TUI 应用进入时会发送 CSI ?1049h
// （或较老的 ?1047h、?47h）切到 alt screen 绘制全屏 UI。
//
// 在 LineHandler 中调用时返回的是查询时刻的状态，可能比该 line 的 LF 提交
// 时刻稍晚——但 TUI 进入/退出与 shell prompt 通常分布在不同输入 chunk，
// 实际场景中该 race 不会让 alt screen 内的内容被误识别成 shell 命令。
func IsAltScreen(vt VirtualTerminal) bool {
	return vt.IsPrivateModeSet(1049) ||
		vt.IsPrivateModeSet(1047) ||
		vt.IsPrivateModeSet(47)
}

func (vt *virtualTerminal) addCsiHandler(b byte, handler inputHandler) {
	vt.inputHandlers[b] = handler
}

func (vt *virtualTerminal) getCurrentRow() *Row {
	if len(vt.rowList) == 0 {
		vt.newRow()
	}

	if vt.rows >= len(vt.rowList) {
		for i := len(vt.rowList); i <= vt.rows; i++ {
			vt.newRow()
		}
	}
	vt.evictLocked()

	return vt.rowList[vt.rows]
}

// evictLocked 执行行窗口上限驱逐（假定已持写锁）。窗口超出 maxLines 时
// 从头部移除最老的行，并把光标行索引同步前移——正在编辑的行与新近内容
// 始终保留在窗口内。驱逐的行不参与任何后续行提交（提交发生在 LF 时刻，
// 远在窗口头部的行早已提交过或本就不属于当前逻辑行）。
func (vt *virtualTerminal) evictLocked() {
	if vt.maxLines <= 0 {
		return
	}
	overflow := len(vt.rowList) - vt.maxLines
	if overflow <= 0 {
		return
	}
	for i := range overflow {
		vt.rowList[i] = nil // 释放引用，帮助 GC
	}
	copy(vt.rowList, vt.rowList[overflow:])
	vt.rowList = vt.rowList[:len(vt.rowList)-overflow]
	vt.rows -= overflow
	if vt.rows < 0 {
		vt.rows = 0
	}
	// OSC 133 命令区快照行号随窗口前移；被驱逐越过时区域失效
	if vt.cmdZoneActive {
		vt.cmdZoneRow -= overflow
		if vt.cmdZoneRow < 0 {
			vt.cmdZoneRow = 0
			vt.cmdZoneActive = false
		}
	}
}

// enterAltScreen 切到 alt 屏。clear 为 true 对应 ?1049h 语义（进入即清空
// alt 屏）；saveCursor 为 true 时保存 main 屏光标，供 ?1049l 恢复。
// main 屏的行窗口与光标在 alt 期间原样冻结。
func (vt *virtualTerminal) enterAltScreen(clear, saveCursor bool) {
	if vt.inAltScreen {
		return
	}
	if saveCursor {
		vt.altSavedRow = vt.rows
		vt.altSavedCol = vt.getCurrentRow().index
	}
	if clear {
		vt.altScr = screenState{}
	}
	vt.screenState = &vt.altScr
	vt.inAltScreen = true
}

// exitAltScreen 切回 main 屏。restoreCursor 对应 ?1049l 恢复进入前保存的
// 光标；clearAlt 对应 ?1047l 的"清 alt 再切回"语义。
func (vt *virtualTerminal) exitAltScreen(clearAlt, restoreCursor bool) {
	if !vt.inAltScreen {
		return
	}
	if clearAlt {
		vt.altScr = screenState{}
	}
	vt.screenState = &vt.mainScr
	vt.inAltScreen = false
	if restoreCursor {
		vt.moveTo(vt.altSavedCol, vt.altSavedRow)
	}
}

func (vt *virtualTerminal) newRow() *Row {
	row := &Row{
		data:  make([]rune, 0),
		index: 0,
	}
	vt.rowList = append(vt.rowList, row)
	return row
}

// initParser 建立 ANSI 字节流状态机。解析状态跨 Advance 调用持久保存，
// 因此被 TCP 分包截断的 CSI/OSC/DCS 序列能在下一个 chunk 到达后正确完成。
func (vt *virtualTerminal) initParser() {
	vt.parser = ansi.NewParser()
	vt.parser.SetParamsSize(parser.MaxParamsSize)
	vt.parser.SetDataSize(maxStringDataSize)
	vt.parser.SetHandler(ansi.Handler{
		Print:     vt.handlePrint,
		Execute:   vt.handleExecute,
		HandleCsi: vt.handleCsi,
		HandleEsc: vt.handleEsc,
		HandleOsc: vt.handleOsc,
		// DCS / SOS / PM / APC 对行识别无用，解析器会正确消费到终止符，直接丢弃
		HandleDcs: func(ansi.Cmd, ansi.Params, []byte) {},
		HandleSos: func([]byte) {},
		HandlePm:  func([]byte) {},
		HandleApc: func([]byte) {},
	})
}

// csiParams 把解析器产出的结构化参数还原为现有 CSI handler 使用的
// []rune 参数串：私有标记前缀（如 '?'）+ 分号分隔的参数。
// 子参数（冒号分隔）按 x/ansi 的打包方式还原：两个参数之间的分隔符取决于
// 前一个参数是否带 HasMore 标志（其后跟的是 ':' 还是 ';'）。
func csiParams(cmd ansi.Cmd, params ansi.Params) []rune {
	var b strings.Builder
	if p := cmd.Prefix(); p != 0 {
		b.WriteByte(p)
	}
	prevMore := false
	params.ForEach(0, func(i, param int, more bool) {
		if i > 0 {
			if prevMore {
				b.WriteByte(':')
			} else {
				b.WriteByte(';')
			}
		}
		b.WriteString(strconv.Itoa(param))
		prevMore = more
	})
	return []rune(b.String())
}

func (vt *virtualTerminal) handlePrint(r rune) {
	if r == utf8.RuneError {
		vt.log("无效的UTF-8字符")
		return
	}
	vt.appendCharacter(r)
}

func (vt *virtualTerminal) handleExecute(b byte) {
	switch rune(b) {
	case _BEL: // \a 响铃，无副作用
	case _BS: // \b BS 在 VT 语义里只是"光标左移一格"，并不删除字符。
		// bash 的 readline 重绘提示符时常用 BS + 空格擦除残留字符，如果这里直接删字符会让行内容丢失。
		vt.getCurrentRow().moveLeft()
	case _HT: // \t 移到下一个制表位（默认每 8 列），tab 字符本身不进入行数据
		vt.nextTab(1)
	case _LF, _VT, _FF: // \n / \v / \f 都向下移动一行——硬换行，提交逻辑行
		vt.commitLogicalLine()
		vt.moveDown(1)
		vt.setCol(0)
	case _CR: // \r 回到行首
		vt.setCol(0)
	default:
		// SO/SI/DEL 及其它 C0/C1 控制码无副作用
	}
}

func (vt *virtualTerminal) handleCsi(cmd ansi.Cmd, params ansi.Params) {
	handler, ok := vt.inputHandlers[cmd.Final()]
	if !ok {
		vt.log(fmt.Sprintf("no match csi handler for %q", string(cmd.Final())))
		return
	}
	if err := handler(csiParams(cmd, params)); err != nil {
		vt.log(fmt.Sprintf("handle csi sequence err %v", err.Error()))
	}
}

// handleEsc 处理非 CSI 的 ESC 序列。带中间字节的序列（字符集选择 ESC ( B、
// ESC SP F、ESC # 8 等）没有屏幕副作用，直接消费。
func (vt *virtualTerminal) handleEsc(cmd ansi.Cmd) {
	if cmd.Intermediate() != 0 {
		return
	}
	switch cmd.Final() {
	case '7': // DECSC 保存光标位置
		vt.hasSavedCursor = true
		vt.savedRow = vt.rows
		vt.savedCol = vt.getCurrentRow().index
	case '8': // DECRC 恢复光标位置
		if vt.hasSavedCursor {
			vt.moveTo(vt.savedCol, vt.savedRow)
		}
	case 'D': // IND 索引：下移一行，等同 LF（但不回行首），属于硬换行
		vt.commitLogicalLine()
		vt.moveDown(1)
	case 'E': // NEL 下一行：下移一行并回到行首
		vt.commitLogicalLine()
		vt.moveDown(1)
		vt.setCol(0)
	case 'M': // RI 反向索引：上移一行
		vt.moveUp(1)
	case 'H': // HTS 在当前列设置制表位（重新启用被 TBC 0 清除的网格位）
		col := vt.getCurrentRow().index
		vt.tabstops[col] = true
		delete(vt.clearedTabStops, col)
	case 'c': // RIS 完全复位：等价 Reset（两块屏 + 私有模式 + 保存的光标 + 制表位）
		vt.mainScr = screenState{}
		vt.altScr = screenState{}
		vt.screenState = &vt.mainScr
		vt.inAltScreen = false
		vt.cmdZoneActive = false
		vt.privateModes = make(map[int]bool)
		vt.hasSavedCursor = false
		vt.resetTabStops()
	}
}

// resetTabStops 把制表位恢复为默认的每 8 列网格。
func (vt *virtualTerminal) resetTabStops() {
	vt.tabstops = make(map[int]bool)
	vt.clearedTabStops = make(map[int]bool)
	vt.defaultTabGrid = true
}

// setTabCol 按制表位移动设置列：cols>0 时钳制到最后一列（xterm 语义）。
func (vt *virtualTerminal) setTabCol(col int) {
	if vt.cols > 0 && col > vt.cols-1 {
		col = vt.cols - 1
	}
	vt.setCol(col)
}

// nextTab 光标移到之后第 n 个制表位。
//
// 制表位有两类：默认每 8 列的网格，以及 HTS 显式设置（TBC 0 可清除，记录在
// clearedTabStops）的位。显式位与被清除的网格位都是有限个；每越过其中一个，
// 就进入一段不含任何特殊位的纯网格区间——等差数列，剩余步数直接算出。
// 因此无论 n 多大（如畸形的 CSI 2147483647 I），开销都只与特殊位个数相关，
// 不会长时间占用写锁。没有可用制表位时，cols>0 则移到最后一列，否则原地不动。
func (vt *virtualTerminal) nextTab(n int) {
	row := vt.getCurrentRow()
	for n > 0 {
		// 光标前方最近的特殊位（显式制表位或被清除的网格位）
		special := 0
		for s := range vt.tabstops {
			if s > row.index && (special == 0 || s < special) {
				special = s
			}
		}
		for s := range vt.clearedTabStops {
			if s > row.index && (special == 0 || s < special) {
				special = s
			}
		}
		if vt.defaultTabGrid {
			// (row.index, special) 区间内只剩等距网格位，可整段批量消费；
			// 前方没有特殊位时，剩余的全部步数都在网格上，一次算出
			k := n
			if special > 0 {
				k = (special-1)/defaultTabStop - row.index/defaultTabStop
			}
			if n <= k {
				vt.setTabCol((row.index/defaultTabStop + n) * defaultTabStop)
				return
			}
			n -= k
		}
		if special == 0 {
			// 前方已无任何制表位：cols>0 时移到最后一列（xterm 语义），否则原地不动
			if vt.cols > 0 && row.index < vt.cols-1 {
				vt.setCol(vt.cols - 1)
			}
			return
		}
		if vt.cols > 0 && special > vt.cols-1 {
			// 特殊位已在行宽之外：到最后一列为止
			vt.setCol(vt.cols - 1)
			return
		}
		// 落到 special 上：显式制表位消耗一步，被清除的网格位只是路过
		if vt.tabstops[special] {
			if n == 1 {
				vt.setTabCol(special)
				return
			}
			n--
		}
		row.index = special
	}
}

// prevTab 光标移到之前第 n 个制表位，没有更靠前的制表位时停在行首。
// 与 nextTab 相同的分段策略：越过有限个特殊位（显式制表位 / 被清除的网格位）
// 之后是纯网格等差区间，整段批量消费，n 再大也不会长时间循环。
func (vt *virtualTerminal) prevTab(n int) {
	row := vt.getCurrentRow()
	for n > 0 && row.index > 0 {
		// 光标后方最近的特殊位
		special := 0
		for s := range vt.tabstops {
			if s < row.index && s > special {
				special = s
			}
		}
		for s := range vt.clearedTabStops {
			if s < row.index && s > special {
				special = s
			}
		}
		if vt.defaultTabGrid {
			// (special, row.index) 区间内只剩等距网格位，整段批量消费
			k := (row.index-1)/defaultTabStop - special/defaultTabStop
			if n <= k {
				vt.setCol(((row.index-1)/defaultTabStop - (n - 1)) * defaultTabStop)
				return
			}
			n -= k
		}
		// 落到 special 上：显式制表位消耗一步，被清除的网格位只是路过
		if vt.tabstops[special] {
			if n == 1 {
				vt.setCol(special)
				return
			}
			n--
		}
		if special == 0 {
			vt.setCol(0) // 已无更靠前的制表位，停在行首
			return
		}
		row.index = special
	}
}

// handleOsc 回调 OSCHandler，并维护 OSC 133 命令输入区状态。解析器给出的
// data 是 ESC ] 与终止符之间的完整 payload（含前导 cmd，如
// "1337;CurrentDir=/tmp"），与手写解析层时代的 payload 格式一致，直接透传。
func (vt *virtualTerminal) handleOsc(_ int, data []byte) {
	if payload := string(data); strings.HasPrefix(payload, "133;") {
		vt.handleSemanticMarker(payload[len("133;"):])
	}
	if vt.oscHandler == nil || len(data) == 0 {
		return
	}
	vt.oscHandler(string(data))
}

// handleSemanticMarker 处理 shell integration 的 OSC 133 语义标记：
//
//	A = prompt 开始；B = 命令输入区开始；C = 命令输出开始；D = 命令结束。
//
// 命令文本的切分只依赖 B：B 时刻光标 (row, col) 的左方是 prompt、右方是
// 用户即将输入（及 Tab 补全后的最终形态）的命令；首个 LF 提交该行时从
// col 起切即为精确命令。C/D/A 关闭区域——C 之后的 LF 属于命令输出，
// 不应再切命令。
func (vt *virtualTerminal) handleSemanticMarker(sub string) {
	if sub == "" {
		return
	}
	switch sub[0] {
	case 'A', 'C', 'D':
		vt.cmdZoneActive = false
	case 'B':
		vt.cmdZoneActive = true
		vt.cmdZoneRow = vt.rows
		vt.cmdZoneCol = vt.getCurrentRow().index
	}
}

// commitLogicalLine 在硬 LF 触发时把当前行（含其连续的软 wrap 上游行）拼成一条
// 逻辑行，连同提交时刻的 mode 快照一起暂存到 pendingLines，等 Advance 解锁后回放。
// 处于 OSC 133 命令输入区时，额外从 B 标记的光标列起切出精确命令文本。
func (vt *virtualTerminal) commitLogicalLine() {
	if vt.lineHandler == nil {
		return
	}
	if vt.rows < 0 || vt.rows >= len(vt.rowList) {
		return
	}
	end := vt.rows
	start := end
	for start > 0 && vt.rowList[start].wrappedFromPrev {
		start--
	}
	var b strings.Builder
	for i := start; i <= end; i++ {
		b.WriteString(vt.rowList[i].String())
	}
	// OSC 133：逻辑行起点与 B 快照行一致时，首物理行从 B 列起切，后续
	// wrap 行整行拼接——Tab 补全重绘会覆盖行内容但 B 列不变，切出的
	// 命令始终是最终形态。引号续行时首段照常切出（正则兜底对续行同样
	// 只能拿到首段，两者行为对齐）。
	var cmd string
	if vt.cmdZoneActive && start == vt.cmdZoneRow {
		var cb strings.Builder
		cb.WriteString(vt.rowList[start].textFromCol(vt.cmdZoneCol))
		for i := start + 1; i <= end; i++ {
			cb.WriteString(vt.rowList[i].String())
		}
		cmd = cb.String()
	}
	// 命令输入区只在首个 LF 上切分——C/A/D 会关闭区域，但流异常时
	// （C 丢失）靠这里自关闭，避免区域泄漏到命令输出行。
	vt.cmdZoneActive = false
	snap := make(ModeSnapshot, len(vt.privateModes))
	for k, v := range vt.privateModes {
		snap[k] = v
	}
	vt.pendingLines = append(vt.pendingLines, pendingLine{
		line:    b.String(),
		modes:   snap,
		cmdText: cmd,
	})
}

func (vt *virtualTerminal) log(v ...any) {
	if vt.logger != nil {
		vt.logger.Println(v...)
	}
}

func (vt *virtualTerminal) getNumberOrDefault(params []rune, index, _default int) int {
	if index >= len(params) {
		return _default
	}

	var numStr strings.Builder
	for i := index; i < len(params); i++ {
		if params[i] >= '0' && params[i] <= '9' {
			numStr.WriteString(string(params[i]))
		} else {
			break
		}
	}

	if numStr.String() == "" {
		return _default
	}

	if num, err := strconv.Atoi(numStr.String()); err == nil {
		return num
	}

	return _default
}

// appendCharacter 追加一个可打印字符。宽度按显示宽度计算（CJK 等
// 宽字符占 2 列，组合字符等零宽字符按 1 列处理），cols>0 时按显示宽度软 wrap。
// cols==0 时虽然没有列宽，物理行仍以 maxRowRunes 为界强制软 wrap——
// 防止无换行的超长输出把单行 data 撑到无限大；wrap 段标记 wrappedFromPrev，
// 行提交时仍拼回完整逻辑行。
func (vt *virtualTerminal) appendCharacter(code rune) {
	w := runeWidth(code)
	if vt.cols > 0 {
		row := vt.getCurrentRow()
		if row.index+w > vt.cols {
			// 软 wrap：进入下一物理行，并标记为 wrappedFromPrev，让 commitLogicalLine 能合并回去
			vt.moveDown(1)
			vt.setCol(0)
			vt.getCurrentRow().wrappedFromPrev = true
		}
	} else if row := vt.getCurrentRow(); len(row.data)+w > maxRowRunes {
		vt.moveDown(1)
		vt.setCol(0)
		vt.getCurrentRow().wrappedFromPrev = true
	}
	vt.getCurrentRow().append(code, w)
}

func (vt *virtualTerminal) Advance(p []byte) {
	vt.Lock()
	for i := range p {
		vt.parser.Advance(p[i])
	}
	pending := vt.pendingLines
	vt.pendingLines = nil
	handler := vt.lineHandler
	vt.Unlock()

	// 回调在锁外执行，避免 handler 阻塞影响其它读写
	if handler != nil {
		for _, pl := range pending {
			handler(LineEvent{Line: pl.line, Modes: pl.modes, CmdText: pl.cmdText})
		}
	}
}

func (vt *virtualTerminal) IsPrivateModeSet(n int) bool {
	vt.RLock()
	defer vt.RUnlock()
	return vt.privateModes[n]
}

func (vt *virtualTerminal) Output() []string {
	vt.RLock()
	defer vt.RUnlock()

	var result []string
	for _, row := range vt.rowList {
		result = append(result, row.String())
	}
	return result
}

func (vt *virtualTerminal) Reset() {
	vt.Lock()
	defer vt.Unlock()

	vt.resetScreenLocked()
	vt.privateModes = make(map[int]bool)
	vt.hasSavedCursor = false
	vt.resetTabStops()
}

// ResetScreen 仅重置屏幕状态——保留私有模式（如 ?2004 readline、?1049
// alt screen），避免清掉后下一条 LineEvent 的 Modes 快照失真。
// 两块屏都清空并回到 main 屏。
func (vt *virtualTerminal) ResetScreen() {
	vt.Lock()
	defer vt.Unlock()

	vt.resetScreenLocked()
}

// resetScreenLocked 假定调用方已持有写锁。
func (vt *virtualTerminal) resetScreenLocked() {
	vt.mainScr = screenState{}
	vt.altScr = screenState{}
	vt.screenState = &vt.mainScr
	vt.inAltScreen = false
	vt.cmdZoneActive = false
	vt.pendingLines = nil
}
