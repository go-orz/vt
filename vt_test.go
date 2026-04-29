package vt

import (
	"strings"
	"sync"
	"testing"
)

func feed(t *testing.T, s string) []string {
	t.Helper()
	v := New()
	v.Advance([]byte(s))
	return v.Output()
}

func mustEqualLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("line count mismatch: got %d %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOSC_BELTerminator(t *testing.T) {
	got := feed(t, "\x1b]0;my title\x07hello")
	mustEqualLines(t, got, []string{"hello"})
}

func TestOSC_STTerminator(t *testing.T) {
	got := feed(t, "\x1b]0;my title\x9chello")
	mustEqualLines(t, got, []string{"hello"})
}

func TestOSC_ESCBackslashTerminator(t *testing.T) {
	got := feed(t, "\x1b]0;my title\x1b\\hello")
	mustEqualLines(t, got, []string{"hello"})
}

func TestESC_CharsetSelectIsConsumed(t *testing.T) {
	got := feed(t, "\x1b(Bhi")
	mustEqualLines(t, got, []string{"hi"})
}

func TestESC_SaveRestoreCursorIsConsumed(t *testing.T) {
	got := feed(t, "a\x1b7b\x1b8c")
	mustEqualLines(t, got, []string{"abc"})
}

func TestBackspaceMovesCursorWithoutDeleting(t *testing.T) {
	got := feed(t, "abc\b\bX")
	mustEqualLines(t, got, []string{"aXc"})
}

func TestBackspaceSpaceErasePattern(t *testing.T) {
	got := feed(t, "abc\b ")
	mustEqualLines(t, got, []string{"ab"})
}

func TestCursorPositionThenWrite(t *testing.T) {
	got := feed(t, "\x1b[1;5HX")
	mustEqualLines(t, got, []string{"    X"})
}

func TestCRLF(t *testing.T) {
	got := feed(t, "line1\r\nline2")
	if len(got) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(got), got)
	}
	if got[len(got)-2] != "line1" || got[len(got)-1] != "line2" {
		t.Errorf("CRLF handling broken: %q", got)
	}
}

func TestEraseBelowClearsCursorRowRight(t *testing.T) {
	v := New()
	v.Advance([]byte("hello"))
	v.Advance([]byte("\x1b[1;3H"))
	v.Advance([]byte("\x1b[0J"))
	out := v.Output()
	if len(out) != 1 || out[0] != "he" {
		t.Errorf("got %q, want [he]", out)
	}
}

func TestEraseAboveClearsCursorRowLeft(t *testing.T) {
	v := New()
	v.Advance([]byte("hello"))
	v.Advance([]byte("\x1b[1;3H"))
	v.Advance([]byte("\x1b[1J"))
	out := v.Output()
	if len(out) != 1 {
		t.Fatalf("got %d lines, want 1: %q", len(out), out)
	}
	if strings.Contains(out[0], "h") || strings.Contains(out[0], "e") {
		t.Errorf("erase above did not clear chars before cursor: %q", out)
	}
}

func TestVPositionRelativeIsRelative(t *testing.T) {
	v := New()
	v.Advance([]byte("abc"))
	v.Advance([]byte("\x1b[1e"))
	v.Advance([]byte("X"))
	out := v.Output()
	if len(out) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(out), out)
	}
	if out[0] != "abc" {
		t.Errorf("row 0: got %q want %q", out[0], "abc")
	}
	if out[1] != "   X" {
		t.Errorf("row 1: got %q want %q", out[1], "   X")
	}
}

func TestHPositionRelativeIsRelative(t *testing.T) {
	v := New()
	v.Advance([]byte("ab"))
	v.Advance([]byte("\x1b[3a"))
	v.Advance([]byte("Z"))
	out := v.Output()
	if len(out) != 1 || out[0] != "ab   Z" {
		t.Errorf("got %q, want [ab   Z]", out)
	}
}

func TestCursorMoveZeroParamTreatedAsOne(t *testing.T) {
	v := New()
	v.Advance([]byte("a\nb"))
	v.Advance([]byte("\x1b[0A"))
	v.Advance([]byte("X"))
	out := v.Output()
	if len(out) < 1 {
		t.Fatalf("no output")
	}
	if out[0] != "aX" {
		t.Errorf("got row0 %q, want %q", out[0], "aX")
	}
}

func TestSetScrollRegionParsesBottom(t *testing.T) {
	v := New()
	v.Advance([]byte("a\nb\nc\n"))
	v.Advance([]byte("\x1b[1;3r"))
	v.Advance([]byte("X"))
	out := v.Output()
	if len(out) < 1 || out[0] == "" || out[0][0] != 'X' {
		t.Errorf("expected first row to start with X, got %q", out)
	}
}

func TestAdvanceConcurrentSafe(t *testing.T) {
	v := New()
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 100 {
				v.Advance([]byte("hello\r\n"))
			}
		})
	}
	wg.Wait()
	_ = v.Output()
}
