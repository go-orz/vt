package vt

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
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
	_DEL rune = 0x7f // Delete (Caret = ^?)

	_ST rune = 0x9c // String Terminator

	space rune = 0x20 // 空格
)

type inputHandler func(params []rune) error

// LineHandler 在每次硬 LF 时回调一次，参数是被 LF 提交的"逻辑行"——
// 如果开启了 WithCols 且该行是被软 wrap 拆开的，多段会被合并回完整字符串。
type LineHandler func(line string)

type VirtualTerminal interface {
	Advance(p []byte)
	Output() []string
	Reset()
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
	vt.initCsiHandler()
	return vt
}

type virtualTerminal struct {
	sync.RWMutex
	rowList []*Row // 行数据
	rows    int    // 当前行索引（0-based）
	cols    int    // 列宽，0 表示不限

	inputHandlers map[byte]inputHandler
	privateModes  map[int]bool

	lineHandler  LineHandler
	pendingLines []string // commitLogicalLine 在锁内追加，Advance 解锁后回放给 lineHandler

	logger *log.Logger
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

func (vt *virtualTerminal) handleSequence(inputs []byte) []byte {
	if len(inputs) == 0 {
		return inputs
	}
	code, size := utf8.DecodeRune(inputs)
	inputs = inputs[size:]
	switch code {
	case '[': // CSI - 控制序列导入器（Control Sequence Introducer）
		inputs = vt.handleCSISequence(inputs)
	case ']': // OSC – 操作系统命令（Operating System Command）
		inputs = vt.handleOSCSequence(inputs)
	case 'P', 'X', '^', '_': // DCS / SOS / PM / APC - 与 OSC 一样按字符串处理
		inputs = vt.handleStringSequence(inputs)
	case '(', ')', '*', '+', '-', '.', '/': // 字符集选择（G0/G1/G2/G3），再吃掉一个 final byte
		if len(inputs) > 0 {
			inputs = inputs[1:]
		}
	case ' ', '#', '%': // 各类两字节 ESC 序列（如 ESC SP F、ESC # 8）
		if len(inputs) > 0 {
			inputs = inputs[1:]
		}
	default:
		// 其它单字节 ESC 序列（7 8 = > D E M c N O 等）已在上面 size 步骤里被吃掉，无需额外处理
	}
	return inputs
}

func (vt *virtualTerminal) handleCSISequence(p []byte) []byte {
	index := bytes.IndexFunc(p, func(r rune) bool {
		return isCSISequence(r)
	})
	if index > -1 {
		b := p[index]
		handler, ok := vt.inputHandlers[b]
		if ok {
			params := []rune(string(p[:index]))
			if err := handler(params); err != nil {
				vt.log(fmt.Sprintf("handle csi sequence err %v", err.Error()))
			}
		} else {
			vt.log(fmt.Sprintf("no match input handler for %q %v", b, b))
		}
		return p[index+1:]
	}

	return p
}

// handleStringSequence 消费一段以 BEL / ST(0x9c) / ESC \ 终止的"字符串型"控制序列（OSC/DCS/SOS/PM/APC）。
// 之前的实现只识别 ST，遇到 xterm 风格的 BEL 终止就会把后续所有输出全部吞掉。
func (vt *virtualTerminal) handleStringSequence(p []byte) []byte {
	for i := range len(p) {
		switch p[i] {
		case byte(_BEL), byte(_ST):
			return p[i+1:]
		case byte(_ESC):
			// ESC \ 也是终止符
			if i+1 < len(p) && p[i+1] == '\\' {
				return p[i+2:]
			}
			// 其它 ESC 视为异常终止，把控制权交还给主循环
			return p[i:]
		}
	}
	// 没有任何终止符——可能是被截断的输入；保守地丢弃剩余字节，避免污染屏幕
	return nil
}

func (vt *virtualTerminal) handleOSCSequence(p []byte) []byte {
	return vt.handleStringSequence(p)
}

func (vt *virtualTerminal) handleC0Sequence(code rune) {
	switch code {
	case _BEL: // \a 响铃，无副作用
	case _BS: // \b BS 在 VT 语义里只是"光标左移一格"，并不删除字符。
		// bash 的 readline 重绘提示符时常用 BS + 空格擦除残留字符，如果这里直接删字符会让行内容丢失。
		vt.getCurrentRow().moveLeft()
	case _HT: // \t 这里简化为输出一个 TAB 字符
		vt.appendCharacter(_HT)
	case _LF, _VT, _FF: // \n / \v / \f 都向下移动一行——硬换行，提交逻辑行
		vt.commitLogicalLine()
		vt.moveDown(1)
		vt.setCol(0)
	case _CR: // \r 回到行首
		vt.setCol(0)
	case _SO, _SI: // 字符集 G1/G0 切换，本实现不区分字符集
	case _DEL: // VT 终端通常忽略 DEL；现代终端的退格键发送的是 BS 或 CSI ~。
	}
}

// commitLogicalLine 在硬 LF 触发时把当前行（含其连续的软 wrap 上游行）拼成一条逻辑行，
// 暂存到 pendingLines 等 Advance 解锁后再交给 lineHandler。
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
	vt.pendingLines = append(vt.pendingLines, b.String())
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

	var numStr string
	for i := index; i < len(params); i++ {
		if params[i] >= '0' && params[i] <= '9' {
			numStr += string(params[i])
		} else {
			break
		}
	}

	if numStr == "" {
		return _default
	}

	if num, err := strconv.Atoi(numStr); err == nil {
		return num
	}

	return _default
}

func (vt *virtualTerminal) appendCharacter(code rune) {
	if vt.cols > 0 {
		row := vt.getCurrentRow()
		if row.index >= vt.cols {
			// 软 wrap：进入下一物理行，并标记为 wrappedFromPrev，让 commitLogicalLine 能合并回去
			vt.moveDown(1)
			vt.setCol(0)
			vt.getCurrentRow().wrappedFromPrev = true
		}
	}
	vt.getCurrentRow().append(code)
}

func (vt *virtualTerminal) Advance(p []byte) {
	vt.Lock()
	vt.advance(p)
	pending := vt.pendingLines
	vt.pendingLines = nil
	handler := vt.lineHandler
	vt.Unlock()

	// 回调在锁外执行，避免 handler 阻塞影响其它读写
	if handler != nil {
		for _, line := range pending {
			handler(line)
		}
	}
}

func (vt *virtualTerminal) IsPrivateModeSet(n int) bool {
	vt.RLock()
	defer vt.RUnlock()
	return vt.privateModes[n]
}

func (vt *virtualTerminal) advance(inputs []byte) {
	for len(inputs) > 0 {
		code, size := utf8.DecodeRune(inputs)
		if code == utf8.RuneError && size == 1 {
			vt.log("无效的UTF-8字符")
			inputs = inputs[1:]
			continue
		}

		inputs = inputs[size:]
		if _ESC == code {
			inputs = vt.handleSequence(inputs)
			continue
		}
		if isC0Sequence(code) {
			vt.handleC0Sequence(code)
		} else {
			vt.appendCharacter(code)
		}
	}
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

	// 清理现有行数据，避免内存泄漏
	for i := range vt.rowList {
		if vt.rowList[i] != nil {
			vt.rowList[i].data = nil
			vt.rowList[i] = nil
		}
	}
	vt.rowList = make([]*Row, 0)
	vt.rows = 0
	vt.privateModes = make(map[int]bool)
	vt.pendingLines = nil
}
