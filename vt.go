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
	"github.com/mattn/go-runewidth"
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
		rowList:       make([]*Row, 0),
		rows:          0,
		logger:        nil,
	}
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
	rowList []*Row // 行数据
	rows    int    // 当前行索引（0-based）
	cols    int    // 列宽，0 表示不限

	parser *ansi.Parser // 跨 Advance 持久保存解析状态的 ANSI 字节流状态机

	inputHandlers map[byte]inputHandler
	privateModes  map[int]bool

	lineHandler  LineHandler
	pendingLines []pendingLine // commitLogicalLine 在锁内追加，Advance 解锁后回放给 lineHandler

	oscHandler OSCHandler

	tabstops       map[int]bool // 显式设置的制表位（HTS 添加、TBC 移除）
	defaultTabGrid bool         // 是否启用每 8 列的默认制表位网格（TBC 3 关闭）

	hasSavedCursor bool // ESC 7 保存的光标状态
	savedRow       int
	savedCol       int

	logger *log.Logger
}

// pendingLine 记录一条尚未交给 LineHandler 的提交事件。
type pendingLine struct {
	line  string
	modes ModeSnapshot
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

	return vt.rowList[vt.rows]
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
// 子参数（冒号分隔）按 x/ansi 的方式还原，现有 handler 均不使用子参数。
func csiParams(cmd ansi.Cmd, params ansi.Params) []rune {
	var b strings.Builder
	if p := cmd.Prefix(); p != 0 {
		b.WriteByte(p)
	}
	params.ForEach(0, func(i, param int, more bool) {
		if i > 0 {
			if more {
				b.WriteByte(':')
			} else {
				b.WriteByte(';')
			}
		}
		b.WriteString(strconv.Itoa(param))
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
	case 'H': // HTS 在当前列设置制表位
		vt.tabstops[vt.getCurrentRow().index] = true
	case 'c': // RIS 完全复位：等价 Reset（屏幕 + 私有模式 + 保存的光标 + 制表位）
		vt.resetScreenLocked()
		vt.privateModes = make(map[int]bool)
		vt.hasSavedCursor = false
		vt.resetTabStops()
	}
}

// resetTabStops 把制表位恢复为默认的每 8 列网格。
func (vt *virtualTerminal) resetTabStops() {
	vt.tabstops = make(map[int]bool)
	vt.defaultTabGrid = true
}

// nextTab 光标移到之后第 n 个制表位。候选位取默认网格与显式设置（HTS）中
// 更近的一个；没有可用制表位时，cols>0 则移到最后一列（xterm 语义），否则原地不动。
func (vt *virtualTerminal) nextTab(n int) {
	row := vt.getCurrentRow()
	for range n {
		next := 0
		if vt.defaultTabGrid {
			// 光标在 15 列 → 下一个 stop 是 16；恰好在 stop 上（16）→ 跳到 24
			next = ((row.index / defaultTabStop) + 1) * defaultTabStop
		}
		for stop := range vt.tabstops {
			if stop > row.index && (next == 0 || stop < next) {
				next = stop
			}
		}
		if next == 0 {
			if vt.cols > 0 && row.index < vt.cols-1 {
				vt.setCol(vt.cols - 1)
			}
			return
		}
		if vt.cols > 0 && next > vt.cols-1 {
			next = vt.cols - 1
		}
		vt.setCol(next)
	}
}

// prevTab 光标移到之前第 n 个制表位，没有更靠前的制表位时停在行首。
func (vt *virtualTerminal) prevTab(n int) {
	row := vt.getCurrentRow()
	for range n {
		prev := 0
		if vt.defaultTabGrid && row.index > 0 {
			prev = ((row.index - 1) / defaultTabStop) * defaultTabStop
		}
		for stop := range vt.tabstops {
			if stop < row.index && stop > prev {
				prev = stop
			}
		}
		vt.setCol(prev)
	}
}

// handleOsc 回调 OSCHandler。解析器给出的 data 是 ESC ] 与终止符之间的
// 完整 payload（含前导 cmd，如 "1337;CurrentDir=/tmp"），与手写解析层
// 时代的 payload 格式一致，直接透传。
func (vt *virtualTerminal) handleOsc(_ int, data []byte) {
	if vt.oscHandler == nil || len(data) == 0 {
		return
	}
	vt.oscHandler(string(data))
}

// commitLogicalLine 在硬 LF 触发时把当前行（含其连续的软 wrap 上游行）拼成一条
// 逻辑行，连同提交时刻的 mode 快照一起暂存到 pendingLines，等 Advance 解锁后回放。
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
	snap := make(ModeSnapshot, len(vt.privateModes))
	for k, v := range vt.privateModes {
		snap[k] = v
	}
	vt.pendingLines = append(vt.pendingLines, pendingLine{
		line:  b.String(),
		modes: snap,
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
func (vt *virtualTerminal) appendCharacter(code rune) {
	w := runewidth.RuneWidth(code)
	if w < 1 {
		w = 1
	}
	if vt.cols > 0 {
		row := vt.getCurrentRow()
		if row.index+w > vt.cols {
			// 软 wrap：进入下一物理行，并标记为 wrappedFromPrev，让 commitLogicalLine 能合并回去
			vt.moveDown(1)
			vt.setCol(0)
			vt.getCurrentRow().wrappedFromPrev = true
		}
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
			handler(LineEvent{Line: pl.line, Modes: pl.modes})
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
func (vt *virtualTerminal) ResetScreen() {
	vt.Lock()
	defer vt.Unlock()

	vt.resetScreenLocked()
}

// resetScreenLocked 假定调用方已持有写锁。
func (vt *virtualTerminal) resetScreenLocked() {
	for i := range vt.rowList {
		if vt.rowList[i] != nil {
			vt.rowList[i].data = nil
			vt.rowList[i] = nil
		}
	}
	vt.rowList = make([]*Row, 0)
	vt.rows = 0
	vt.pendingLines = nil
}
