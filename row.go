package vt

type Row struct {
	data  []rune // 当前行
	index int
	// wrappedFromPrev 标记本行是 cols 软折行从上一行延续而来——对 LineHandler 而言，
	// 这一行与它之前的连续 wrappedFromPrev 行属于同一条"逻辑行"。
	wrappedFromPrev bool
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
		// 光标超出当前行长度时先用空格补齐，避免字符落到错误的列
		for len(r.data) < r.index {
			r.data = append(r.data, space)
		}
		r.data = append(r.data, code)
	}
	r.index++
}

// 光标左移一格，不删除字符（标准 BS 语义）
func (r *Row) moveLeft() {
	if r.index > 0 {
		r.index--
	}
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

// 删除当前光标所在位置右侧的字符
func (r *Row) eraseRight() {
	if r.index < len(r.data) {
		r.data = r.data[:r.index]
	}
}

// 删除当前光标所在位置左侧的字符（包含光标位置，ECMA-48 EL 1 语义）
func (r *Row) eraseLeft() {
	if r.index < len(r.data) {
		// 保留光标之后的内容，擦除 [0..index] 闭区间
		r.data = r.data[r.index+1:]
	} else {
		// 光标已超过行尾——整行清空
		r.data = []rune{}
	}
	r.index = 0
}

func (r *Row) String() string {
	result := string(r.data)
	// 移除尾部空格
	for len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return result
}
