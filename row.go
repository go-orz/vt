package vt

type Row struct {
	data  []rune // 当前行
	index int
}

func (r *Row) setIndex(index int) {
	if index < 0 {
		index = 0
	}
	r.index = index
}

// 添加新输入的字符
func (r *Row) append(code rune) {
	if r.index < len(r.data) {
		// 覆盖模式：替换当前位置的字符
		r.data[r.index] = code
	} else {
		// 在末尾添加
		r.data = append(r.data, code)
	}
	r.index++
}

// 向下标位置插入字符
func (r *Row) insert(code ...rune) {
	for _, c := range code {
		if r.index < 0 {
			r.index = 0
		}
		if r.index > len(r.data) {
			r.index = len(r.data)
		}
		r.data = insert(r.data, r.index, c)
		r.index++ // 插入后光标向右移动
	}
}

// 从下标位置删除N个字符
func (r *Row) delete(ps int) {
	if r.index < 0 || r.index >= len(r.data) || ps <= 0 {
		return
	}
	r.data = remove(r.data, r.index, ps)
}

// 删除指定位置的一个字符
func (r *Row) deleteAt(pos int) {
	if pos >= 0 && pos < len(r.data) {
		r.data = append(r.data[:pos], r.data[pos+1:]...)
	}
}

// 退格：删除光标前一个字符
func (r *Row) backspace() {
	if r.index > 0 && r.index <= len(r.data) {
		// 删除 index-1 位置的字符
		r.data = append(r.data[:r.index-1], r.data[r.index:]...)
		r.index--
	}
}

// 删除当前光标所在位置右侧的字符
func (r *Row) eraseRight() {
	if r.index < len(r.data) {
		r.data = r.data[:r.index]
	}
}

// 删除当前光标所在位置左侧的字符（包含光标位置）
func (r *Row) eraseLeft() {
	if r.index > 0 {
		// 保留光标位置之后的内容
		r.data = r.data[r.index:]
		r.index = 0
	} else {
		// 如果光标在开头，清空整行
		r.data = []rune{}
		r.index = 0
	}
}

func (r *Row) String() string {
	result := string(r.data)
	// 移除尾部空格
	for len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return result
}
