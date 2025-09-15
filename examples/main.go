package main

import (
	"github.com/go-orz/vt"
)

func main() {
	// This is a placeholder for your terminal data stream.
	// You would typically get this from a file or another process.
	content := []byte("hello\n\033[31mworld\033[0m\n")

	v := vt.New()
	v.Advance(content)

	// The Output() method gives you the final state of the screen as a slice of strings.
	lines := v.Output()
	for _, line := range lines {
		println(line)
	}
}
