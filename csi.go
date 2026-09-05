package vt

import (
	"strconv"
	"strings"
)

func (vt *virtualTerminal) initCsiHandler() {
	vt.addCsiHandler('@', vt.insertChar)
	vt.addCsiHandler('A', vt.cursorUp)
	vt.addCsiHandler('B', vt.cursorDown)
	vt.addCsiHandler('C', vt.cursorForward)
	vt.addCsiHandler('D', vt.cursorBackward)
	vt.addCsiHandler('E', vt.cursorNextLine)
	vt.addCsiHandler('F', vt.cursorPrecedingLine)
	vt.addCsiHandler('G', vt.cursorCharAbsolute)
	vt.addCsiHandler('H', vt.cursorPosition)
	vt.addCsiHandler('J', vt.eraseInDisplay)
	vt.addCsiHandler('K', vt.eraseInLine)
	vt.addCsiHandler('P', vt.deleteChars)
	vt.addCsiHandler('X', vt.eraseChars)
	vt.addCsiHandler('Z', vt.cursorBackwardTab)
	vt.addCsiHandler('`', vt.charPosAbsolute)
	vt.addCsiHandler('a', vt.hPositionRelative)
	vt.addCsiHandler('d', vt.linePosAbsolute)
	vt.addCsiHandler('e', vt.vPositionRelative)
	vt.addCsiHandler('f', vt.hVPosition)
	vt.addCsiHandler('g', vt.tabClear)
	vt.addCsiHandler('h', vt.setMode)
	vt.addCsiHandler('I', vt.cursorHorizontalTab)
	vt.addCsiHandler('l', vt.resetMode)
	vt.addCsiHandler('m', vt.charAttributes)
	vt.addCsiHandler('r', vt.setScrollRegion)
}

func (vt *virtualTerminal) cursorChange(params []rune, action func(ps int)) {
	ps := vt.getNumberOrDefault(params, 0, 1)
	// VT100/ECMA-48：CSI A/B/C/D/G 等"移动 N 格"的命令，0 应当作 1
	if ps == 0 {
		ps = 1
	}
	action(ps)
}

// insert Ps (Blank) Character(s) (default = 1) (ICH).
func (vt *virtualTerminal) insertChar(params []rune) error {
	row := vt.getCurrentRow()
	ps := vt.getNumberOrDefault(params, 0, 1)
	if ps == 0 {
		ps = 1
	}
	for range ps {
		row.insert(space)
	}
	return nil
}

// 光标向指定的方向移动{n（默认1）格。如果光标已在屏幕边缘，则无效。
func (vt *virtualTerminal) cursorUp(params []rune) error {
	vt.cursorChange(params, vt.moveUp)
	return nil
}

// 光标向指定的方向移动{n（默认1）格。如果光标已在屏幕边缘，则无效。
func (vt *virtualTerminal) cursorDown(params []rune) error {
	vt.cursorChange(params, vt.moveDown)
	return nil
}

// 光标向指定的方向移动{n（默认1）格。如果光标已在屏幕边缘，则无效。
func (vt *virtualTerminal) cursorForward(params []rune) error {
	vt.cursorChange(params, vt.moveForward)
	return nil
}

// 光标向指定的方向移动{n（默认1）格。如果光标已在屏幕边缘，则无效。
func (vt *virtualTerminal) cursorBackward(params []rune) error {
	vt.cursorChange(params, vt.moveBackward)
	return nil
}

// 光标移动到下面第n（默认1）行的开头。CNL 除了下移还必须回到行首（CR 语义）。
func (vt *virtualTerminal) cursorNextLine(params []rune) error {
	vt.cursorChange(params, vt.moveDown)
	vt.setCol(0)
	return nil
}

// 光标移动到上面第n（默认1）行的开头。CPL 除了上移还必须回到行首（CR 语义）。
func (vt *virtualTerminal) cursorPrecedingLine(params []rune) error {
	vt.cursorChange(params, vt.moveUp)
	vt.setCol(0)
	return nil
}

// 光标移动到第n（默认1）列。CSI G 的列是 1-based。
func (vt *virtualTerminal) cursorCharAbsolute(params []rune) error {
	vt.cursorChange(params, func(ps int) {
		vt.moveTo(ps-1, vt.rows)
	})
	return nil
}

// 光标移动到第n行、第m列。值从1开始，且默认为1（左上角）。
// 例如 CSI ;5H 和 CSI 1;5H 含义相同；CSI 17;H、CSI 17H 和 CSI 17;1H 三者含义相同。
func (vt *virtualTerminal) cursorPosition(params []rune) error {
	_, values := vt.parseCSIParams(params)

	row := 1
	col := 1

	if len(values) > 0 && values[0] > 0 {
		row = values[0]
	}
	if len(values) > 1 && values[1] > 0 {
		col = values[1]
	}

	vt.moveTo(col-1, row-1)
	return nil
}

// 清除屏幕的部分区域。如果n是0（或缺失），则清除从光标位置到屏幕末尾的部分。
// 如果n是1，则清除从光标位置到屏幕开头的部分。
// 如果n是2，则清除整个屏幕（在DOS ANSI.SYS中，光标还会向左上方移动）。
// 如果n是3，则清除整个屏幕，并删除回滚缓存区中的所有行（这个特性是xterm添加的，其他终端应用程序也支持）。
func (vt *virtualTerminal) eraseInDisplay(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 0)
	switch ps {
	case 0:
		return vt.eraseBelow()
	case 1:
		return vt.eraseAbove()
	case 2:
		return vt.eraseAll()
	case 3:
		// 忽略 Erase Saved Lines (xterm)
	}
	return nil
}

// 清除行内的部分区域。
// 如果n是0（或缺失），清除从光标位置到该行末尾的部分。
// 如果n是1，清除从光标位置到该行开头的部分。
// 如果n是2，清除整行。光标位置不变。
func (vt *virtualTerminal) eraseInLine(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 0)
	switch ps {
	case 0:
		return vt.eraseRight()
	case 1:
		return vt.eraseLeft()
	case 2:
		return vt.eraseCurrentLine()
	}
	return nil
}

// Delete Ps Character(s) (default = 1) (DCH).
func (vt *virtualTerminal) deleteChars(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 1)
	if ps == 0 {
		ps = 1
	}
	row := vt.getCurrentRow()
	row.delete(ps)
	return nil
}

// Erase Ps Character(s) (default = 1) (ECH).
// ECH 与 DCH 不同：它用空格覆盖 N 个字符，后续内容保持原位不左移。
func (vt *virtualTerminal) eraseChars(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 1)
	if ps == 0 {
		ps = 1
	}
	row := vt.getCurrentRow()
	for i := range ps {
		if idx := row.index + i; idx < len(row.data) {
			row.data[idx] = space
		}
	}
	return nil
}

// Character Position Absolute  [column] (default = [rows,1])。CSI ` 列是 1-based。
func (vt *virtualTerminal) charPosAbsolute(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 1)
	vt.moveTo(ps-1, vt.rows)
	return nil
}

// Character Position Relative (HPR, CSI Pn a)：光标向右移 N 列，行保持不变。
func (vt *virtualTerminal) hPositionRelative(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 1)
	if ps == 0 {
		ps = 1
	}
	vt.move(ps, 0)
	return nil
}

// 行定位绝对 (VPA)。CSI d 行是 1-based。
func (vt *virtualTerminal) linePosAbsolute(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 1)
	vt.setRow(ps - 1)
	return nil
}

// Line Position Relative (VPR, CSI Pn e)：光标向下移 N 行，列保持不变。
// 原实现 moveTo(0, ps) 是绝对定位且把 col 清零，与规范不符。
func (vt *virtualTerminal) vPositionRelative(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 1)
	if ps == 0 {
		ps = 1
	}
	vt.move(0, ps)
	return nil
}

// Horizontal and Vertical Position [rows;column] (default = [1,1]) (HVP).
func (vt *virtualTerminal) hVPosition(params []rune) error {
	return vt.cursorPosition(params)
}

// Cursor Horizontal Tabulation (CHT, CSI Pn I)：光标前进 n 个制表位。
func (vt *virtualTerminal) cursorHorizontalTab(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 1)
	if ps == 0 {
		ps = 1
	}
	vt.nextTab(ps)
	return nil
}

// Cursor Backward Tabulation (CBT, CSI Pn Z)：光标后退 n 个制表位。
func (vt *virtualTerminal) cursorBackwardTab(params []rune) error {
	ps := vt.getNumberOrDefault(params, 0, 1)
	if ps == 0 {
		ps = 1
	}
	vt.prevTab(ps)
	return nil
}

// Tab Clear (TBC, CSI Pn g)：0 清除当前列的制表位；3 清除全部制表位
// （包括默认的 8 列网格，之后只有 HTS 重新设置的制表位生效）。
func (vt *virtualTerminal) tabClear(params []rune) error {
	switch vt.getNumberOrDefault(params, 0, 0) {
	case 0:
		delete(vt.tabstops, vt.getCurrentRow().index)
	case 3:
		vt.tabstops = make(map[int]bool)
		vt.defaultTabGrid = false
	}
	return nil
}

func (vt *virtualTerminal) parseCSIParams(params []rune) (isPrivate bool, values []int) {
	paramStr := string(params)
	if strings.HasPrefix(paramStr, "?") {
		isPrivate = true
		paramStr = paramStr[1:]
	}

	if paramStr == "" {
		return isPrivate, []int{}
	}

	for part := range strings.SplitSeq(paramStr, ";") {
		if part == "" {
			values = append(values, 0)
			continue
		}
		val, err := strconv.Atoi(part)
		if err != nil {
			values = append(values, 0) // Default on error
		} else {
			values = append(values, val)
		}
	}
	return isPrivate, values
}

/**
 * CSI Pm h  Set Mode (SM).
 *     Ps = 2  -> Keyboard Action Mode (AM).
 *     Ps = 4  -> insert Mode (IRM). Insert/Replace Mode
 *     Ps = 1 2  -> Send/receive (SRM).
 *     Ps = 2 0  -> Automatic Newline (LNM).
 *
 * @virtualTerminal: #P[Only IRM is supported.]    CSI SM    "Set Mode"  "CSI Pm h"  "Set various terminal modes."
 * Supported param values by SM:
 *
 * | Param | Action                                 | Support |
 * | ----- | -------------------------------------- | ------- |
 * | 2     | Keyboard Action Mode (KAM). Always on. | #N      |
 * | 4     | insert Mode (IRM).                     | #Y      |
 * | 12    | Send/receive (SRM). Always off.        | #N      |
 * | 20    | Automatic Newline (LNM). Always off.   | #N      |
 */
// setMode 处理 CSI Pm h。私有模式（带 ?，如 ?2004 bracketed paste、?1049 alt screen、?25 cursor）
// 仅记录到 privateModes，由调用方通过 IsPrivateModeSet 查询。ANSI 标准模式当前忽略。
func (vt *virtualTerminal) setMode(params []rune) error {
	isPrivate, values := vt.parseCSIParams(params)
	if !isPrivate {
		return nil
	}
	for _, v := range values {
		vt.privateModes[v] = true
	}
	return nil
}

// resetMode 处理 CSI Pm l——私有模式从 privateModes 中删除；ANSI 标准模式当前忽略。
func (vt *virtualTerminal) resetMode(params []rune) error {
	isPrivate, values := vt.parseCSIParams(params)
	if !isPrivate {
		return nil
	}
	for _, v := range values {
		delete(vt.privateModes, v)
	}
	return nil
}

// Set Scrolling Region [top;bottom] (default = full size of window) (DECSTBM), VT100.
// 本实现没有真正的滚动区，仅按规范将光标移到 home (0,0)。
// 原实现用 getNumberOrDefault 取第二个参数有 bug——它从指定 byte 偏移开始读，遇到分号立刻 break，永远拿不到 bottom。
func (vt *virtualTerminal) setScrollRegion(params []rune) error {
	_, values := vt.parseCSIParams(params)
	top := 1
	bottom := 0
	if len(values) > 0 && values[0] > 0 {
		top = values[0]
	}
	if len(values) > 1 && values[1] > 0 {
		bottom = values[1]
	}
	if bottom == 0 || bottom > len(vt.rowList) {
		bottom = len(vt.rowList)
	}
	if bottom > top {
		vt.moveTo(0, 0)
	}
	return nil
}

func (vt *virtualTerminal) eraseBelow() error {
	// J 0：从光标到屏幕末尾——当前行光标右侧 + 当前行之后所有行
	if vt.rows >= 0 && vt.rows < len(vt.rowList) {
		vt.getCurrentRow().eraseRight()
		vt.rowList = vt.rowList[:vt.rows+1]
	}
	return nil
}

func (vt *virtualTerminal) eraseAbove() error {
	// J 1：从屏幕开头到光标——当前行之前所有行 + 当前行光标左侧
	if vt.rows >= 0 && vt.rows < len(vt.rowList) {
		// 用空行替换当前行之前的内容，保留行号避免 cursor 失位
		for i := 0; i < vt.rows; i++ {
			vt.rowList[i] = &Row{data: []rune{}, index: 0}
		}
		vt.getCurrentRow().eraseLeft()
	}
	return nil
}

func (vt *virtualTerminal) eraseAll() error {
	vt.rowList = nil
	vt.resetCursor()
	return nil
}

func (vt *virtualTerminal) eraseRight() error {
	row := vt.getCurrentRow()
	row.eraseRight()
	return nil
}

func (vt *virtualTerminal) eraseLeft() error {
	row := vt.getCurrentRow()
	row.eraseLeft()
	return nil
}

func (vt *virtualTerminal) eraseCurrentLine() error {
	row := vt.getCurrentRow()
	row.data = []rune{}
	row.index = 0
	return nil
}

func (vt *virtualTerminal) charAttributes(params []rune) error {
	// 忽略字符的颜色属性
	return nil
}
