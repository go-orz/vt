package vt

import (
	"strings"

	"github.com/mattn/go-runewidth"
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
	r.index = index
}

// append 追加一个显示宽度为 w（1 或 2）的字符，光标前进 w 列。
// 宽字符（CJK 等）占两列：紧随其后补一个占位空格，保证 data 下标与列号一致。
// 覆盖模式下若写在前一个宽字符的占位列上，先把宽字符清成空格（xterm 语义）。
func (r *Row) append(code rune, w int) {
	if r.index < len(r.data) {
		if r.index > 0 && runewidth.RuneWidth(r.data[r.index-1]) > 1 {
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
	var b strings.Builder
	for i, c := range r.data {
		// 宽字符的占位空格不输出：我们的不变式是宽字符后必然紧跟其占位空格
		if c == space && i > 0 && runewidth.RuneWidth(r.data[i-1]) > 1 {
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
