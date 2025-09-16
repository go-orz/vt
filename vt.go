package vt

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"sync"
	"unicode/utf8"
)

const (
	_BEL rune = 0x07 // Bell (Caret = ^G, C = \a)
	_BS  rune = 0x08 // Backspace (Caret = ^H, C = \b)
	_HT  rune = 0x09 // Position to the next character tab stop.(Caret = ^I, C = \t)
	_LF  rune = 0x0a // LF Line Feed (Caret = ^J, C = \n)
	_VT  rune = 0x0b // Position the form at the next line tab stop.(Caret = ^K, C = \v)
	_CR  rune = 0x0d // Carriage Return (Caret = ^M, C = \r)

	_ESC rune = 0x1b // Escape (Caret = ^[, C = \e)
	_DEL rune = 0x7f // Delete (Caret = ^?)

	_ST rune = 0x9c // String Terminator

	space rune = 0x20 // 空格
)

type inputHandler func(params []rune) error

type VirtualTerminal interface {
	Advance(p []byte)
	Output() []string
	Reset()
}

// Opt is a function that configures a virtualTerminal.
type Opt func(*virtualTerminal)

// WithLogger sets a logger for the virtual terminal.
func WithLogger(logger *log.Logger) Opt {
	return func(vt *virtualTerminal) {
		vt.logger = logger
	}
}

func New() VirtualTerminal {
	return NewWithOptions()
}

func NewWithOptions(opts ...Opt) VirtualTerminal {
	vt := &virtualTerminal{
		inputHandlers: make(map[byte]inputHandler),
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
	rows    int    // 行数量

	inputHandlers map[byte]inputHandler
	logger        *log.Logger
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
	code, size := utf8.DecodeRune(inputs)
	inputs = inputs[size:]
	switch code {
	case '[': // CSI - 控制序列导入器（Control Sequence Introducer）
		inputs = vt.handleCSISequence(inputs)
	case ']': // OSC – 操作系统命令（Operating System Command）
		inputs = vt.handleOSCSequence(inputs)
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

func (vt *virtualTerminal) handleOSCSequence(p []byte) []byte {
	// 找到终止符
	index := bytes.IndexRune(p, _ST)
	if index > -1 {
		return p[index+1:]
	}
	return []byte{}
}

func (vt *virtualTerminal) handleC0Sequence(code rune) {
	switch code {
	case _BEL: // \a 发出可听见的噪音。
	case _BS: // \b 将光标向左移动一个字符并删除
		row := vt.getCurrentRow()
		// 使用 backspace 方法，它会同时处理删除和光标移动
		row.backspace()
	case _HT: // \t 定位到下一个制表位。
		// 插入制表符或空格
		vt.appendCharacter(_HT)
	case _LF:
		vt.moveDown(1)
		vt.setCol(0) // 换行时也要回到行首
	case _CR: // \n or \r
		vt.setCol(0) // 仅回到行首，不换行
	case _VT: // \v 定位到下一行的制表位。
		vt.moveDown(1) // 移动到下一行，保持当前列位置
	case _DEL: // 最初用于穿孔纸带上删除一个字符。因为任何位置的字符都可以被全部穿孔（全1）。VT100兼容终端，按键⌫产生这个字符，常称为backspace，但不对应于PC键盘的delete key。
		row := vt.getCurrentRow()
		row.backspace() // 使用退格操作删除前一个字符
	}
}

func (vt *virtualTerminal) log(v ...interface{}) {
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

func (vt *virtualTerminal) getNumberOrDefaultOfBytes(params []byte, index, _default int) int {
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
	currentRow := vt.getCurrentRow()
	currentRow.append(code)
}

func (vt *virtualTerminal) Advance(p []byte) {
	vt.advance(p)
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
	return
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
	return
}
