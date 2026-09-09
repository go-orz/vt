package vt

import "testing"

func TestSnapshotCommandPreservesCursorAndIncludesWrappedSuffix(t *testing.T) {
	v := NewWithOptions(WithCols(8))
	v.Advance([]byte("\x1b]133;A\a$ \x1b]133;B\aabcdefghijk\x1b[1A\x1b[4G"))
	before := v.Output()
	got := v.SnapshotCommand()
	if !got.CommandKnown || got.Command != "abcdefghijk" {
		t.Fatalf("snapshot = %+v", got)
	}
	after := v.Output()
	if len(before) != len(after) {
		t.Fatal("snapshot modified screen")
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatal("snapshot modified screen")
		}
	}
	v.Advance([]byte("Z"))
	if got := v.SnapshotCommand(); got.Command != "aZcdefghijk" {
		t.Fatalf("cursor changed by snapshot: %+v", got)
	}
}

func TestSnapshotCommandEmptyAndAltScreen(t *testing.T) {
	v := NewWithOptions()
	v.Advance([]byte("custom>\x1b]133;B\a"))
	if got := v.SnapshotCommand(); !got.CommandKnown || got.Command != "" {
		t.Fatalf("empty command: %+v", got)
	}
	v.Advance([]byte("\x1b[?1049h"))
	if got := v.SnapshotCommand(); got.CommandKnown || !got.Modes.IsAltScreen() {
		t.Fatalf("alt screen: %+v", got)
	}
}

func TestSnapshotCommandUnicodeAndResize(t *testing.T) {
	v := NewWithOptions(WithCols(80))
	v.Advance([]byte("中文$ \x1b]133;B\aecho 文件"))
	if got := v.SnapshotCommand(); !got.CommandKnown || got.Command != "echo 文件" {
		t.Fatalf("unicode snapshot: %+v", got)
	}
	v.Reset()
	v.ResizeCols(4)
	v.Advance([]byte("$ \x1b]133;B\aabcdefgh"))
	if got := v.SnapshotCommand(); got.Command != "abcdefgh" {
		t.Fatalf("resized snapshot: %+v", got)
	}
}
