package vt

import (
	"slices"
	"strings"
)

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
	// 巨型跳列钳制（CSI G/`/a/C 等）：不超过已有内容与 maxScreenDim 的
	// 较大者，否则后续写入时按 index 补齐空格会一次性分配天文数字内存
	if index > max(len(r.data), maxScreenDim) {
		index = max(len(r.data), maxScreenDim)
	}
	r.index = index
}

// append 追加一个显示宽度为 w（1 或 2）的字符，光标前进 w 列。
// 宽字符（CJK 等）占两列：紧随其后补一个占位空格，保证 data 下标与列号一致。
// 覆盖模式下若写在前一个宽字符的占位列上，先把宽字符清成空格（xterm 语义）。
func (r *Row) append(code rune, w int) {
	if r.index < len(r.data) {
		if r.index > 0 && runeWidth(r.data[r.index-1]) > 1 {
			r.data[r.index-1] = space
		}
		r.data[r.index] = code
		if w > 1 {
			if r.index+1 < len(r.data) {
				r.data[r.index+1] = space
			} else {
				r.data = append(r.data, space)
			}
		}
	} else {
		// 光标超出当前行长度时先用空格补齐，避免字符落到错误的列
		for len(r.data) < r.index {
			r.data = append(r.data, space)
		}
		r.data = append(r.data, code)
		if w > 1 {
			r.data = append(r.data, space)
		}
	}
	r.index += w
}

// 光标左移一格，不删除字符（标准 BS 语义）
func (r *Row) moveLeft() {
	if r.index > 0 {
		r.index--
	}
}

// isWideLead 报告 data[i] 是否是一个占两列的宽字符（CJK 等）。
// 行不变式：宽字符在 data 中必然紧跟一个占位空格，因此宽度 >1 即 lead。
func isWideLead(data []rune, i int) bool {
	return runeWidth(data[i]) > 1
}

// eraseCells 用空格覆盖从光标开始的 n 个显示单元（ECH），后续内容保持原位。
// 宽字符按整字符覆盖（无法半擦除）；循环以行内剩余单元数为上界，
// 畸形超大参数（如 CSI 2147483647 X）不会造成长时间循环。
func (r *Row) eraseCells(n int) {
	i := r.index
	if n <= 0 || i < 0 || i >= len(r.data) {
		return
	}
	// 光标落在宽字符的占位单元上：先把它的 lead 清成空格
	if i > 0 && isWideLead(r.data, i-1) {
		r.data[i-1] = space
	}
	for i < len(r.data) && n > 0 {
		w := runeWidth(r.data[i])
		r.data[i] = space
		if w > 1 && i+1 < len(r.data) {
			r.data[i+1] = space
			i++
		}
		i++
		n -= w
	}
}

// deleteCells 从光标开始删除 n 个显示单元（DCH），后续内容左移。
// 宽字符成对删除（无法拆分）；光标落在占位单元上时先清掉 lead。
// 循环以行内剩余单元数为上界。
func (r *Row) deleteCells(n int) {
	start := r.index
	if n <= 0 || start < 0 || start >= len(r.data) {
		return
	}
	if start > 0 && isWideLead(r.data, start-1) {
		r.data[start-1] = space
	}
	end := start
	for end < len(r.data) && n > 0 {
		w := runeWidth(r.data[end])
		end++
		if w > 1 {
			end++ // 占位空格随宽字符一起删除
		}
		n -= w
	}
	r.data = append(r.data[:start], r.data[end:]...)
}

// insertCells 在光标处插入 n 个空白显示单元（ICH），后续内容右移。
// 一次批量拼接，避免逐字符插入的二次复杂度；插入点落在宽字符的占位
// 单元上时先清掉该宽字符（无法拆分）。
func (r *Row) insertCells(n int) {
	if n <= 0 {
		return
	}
	idx := r.index
	if idx < 0 {
		idx = 0
	}
	if idx > len(r.data) {
		// 光标超出行尾：先补空格对齐，避免插入位置错列
		for len(r.data) < idx {
			r.data = append(r.data, space)
		}
	}
	if idx > 0 && isWideLead(r.data, idx-1) {
		r.data[idx-1] = space
	}
	r.data = slices.Insert(r.data, idx, slices.Repeat([]rune{space}, n)...)
	r.index += n
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
	return stringFrom(r.data)
}

// textFromCol 返回从列 col（0-based）起至行尾的文本，用于 OSC 133
// 命令输入区切分：col 左方是 prompt，右方是用户命令。
// 行不变式是 data 下标与列号一致（宽字符 lead + 占位空格 = 2 个下标），
// 光标 index 也按同一计法推进，因此直接按下标切分。
// col 恰好落在宽字符的占位空格上时（防御：prompt 结束列由 shell 控制，
// 正常不会停在宽字符中间），从 lead 起切以保留完整宽字符。
func (r *Row) textFromCol(col int) string {
	if col < 0 {
		col = 0
	}
	if col > 0 && col < len(r.data) && r.data[col] == space && runeWidth(r.data[col-1]) > 1 {
		col--
	}
	if col >= len(r.data) {
		return ""
	}
	return stringFrom(r.data[col:])
}

// stringFrom 把行 rune 序列输出为文本：跳过宽字符的占位空格，去除尾部空格。
func stringFrom(data []rune) string {
	var b strings.Builder
	for i, c := range data {
		// 宽字符的占位空格不输出：我们的不变式是宽字符后必然紧跟其占位空格
		if c == space && i > 0 && runeWidth(data[i-1]) > 1 {
			continue
		}
		b.WriteRune(c)
	}
	// 移除尾部空格
	result := b.String()
	for len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return result
}
