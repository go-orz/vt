package vt

func (vt *virtualTerminal) resetCursor() {
	vt.rows = 1
	if len(vt.rowList) > 0 {
		vt.getCurrentRow().setIndex(0)
	}
}

func (vt *virtualTerminal) moveTo(col, row int) {
	vt.setCol(col)
	vt.setRow(row)
}

func (vt *virtualTerminal) setRow(row int) {
	if row < 1 {
		row = 1
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
	if vt.rows < 1 {
		vt.rows = 1
	}
}

func (vt *virtualTerminal) moveDown(ps int) {
	vt.rows += ps
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
