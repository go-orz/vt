package vt

// 注意：vt.rows 与 row.index 都是 0-based 内部坐标。
// CSI H/G/d/` 等协议层使用 1-based 列/行，调用方在传入这些函数前已做过 -1 转换。

func (vt *virtualTerminal) resetCursor() {
	vt.rows = 0
	if len(vt.rowList) > 0 {
		vt.getCurrentRow().setIndex(0)
	}
}

func (vt *virtualTerminal) moveTo(col, row int) {
	vt.setRow(row)
	vt.setCol(col)
}

func (vt *virtualTerminal) setRow(row int) {
	if row < 0 {
		row = 0
	}
	// 巨型跳行钳制：不超过已有行数与 maxScreenDim 的较大者（见常量注释）。
	// 否则 CSI 2147483646;1H 这类序列会把 vt.rows 推到 21 亿，
	// 下一次 getCurrentRow 循环建行直接把进程撑爆。
	if row > max(len(vt.rowList), maxScreenDim) {
		row = max(len(vt.rowList), maxScreenDim)
	}
	vt.rows = row
}

func (vt *virtualTerminal) setCol(col int) {
	if col < 0 {
		col = 0
	}
	vt.getCurrentRow().setIndex(col)
}

func (vt *virtualTerminal) moveUp(ps int) {
	vt.rows -= ps
	if vt.rows < 0 {
		vt.rows = 0
	}
}

func (vt *virtualTerminal) moveDown(ps int) {
	vt.rows += ps
	// 与 setRow 相同的钳制：单次下移（含 CUD/CNL 巨型参数）最多落到
	// 已有行数与 maxScreenDim 的较大者；LF 每次 +1 始终在界内，不受影响
	if vt.rows > max(len(vt.rowList), maxScreenDim) {
		vt.rows = max(len(vt.rowList), maxScreenDim)
	}
}

func (vt *virtualTerminal) moveBackward(ps int) {
	index := vt.getCurrentRow().index
	index -= ps
	if index < 0 {
		index = 0
	}
	vt.setCol(index)
}

func (vt *virtualTerminal) moveForward(ps int) {
	index := vt.getCurrentRow().index + ps
	vt.setCol(index)
}

func (vt *virtualTerminal) move(col int, row int) {
	newCol := vt.getCurrentRow().index + col
	newRow := vt.rows + row
	vt.moveTo(newCol, newRow)
}
