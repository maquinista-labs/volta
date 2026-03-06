package tmux

import (
	"errors"
	"testing"
)

func TestIsWindowDead(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("something else"), false},
		{errors.New("session not found"), true},
		{errors.New("can't find window"), true},
		{errors.New("no such window: @5"), true},
	}
	for _, tt := range tests {
		got := IsWindowDead(tt.err)
		if got != tt.want {
			t.Errorf("IsWindowDead(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestHasChromeSeparator(t *testing.T) {
	noSep := "hello\nworld\n"
	if hasChromeSeparator(noSep) {
		t.Error("should not detect separator in plain text")
	}

	withSep := "some output\n" + string(make([]rune, 25)) + "\n"
	// Build a line with 25 ─ characters
	sepLine := ""
	for i := 0; i < 25; i++ {
		sepLine += "─"
	}
	withSep = "some output\n" + sepLine + "\n"
	if !hasChromeSeparator(withSep) {
		t.Error("should detect separator")
	}
}

func TestInsideTmux(t *testing.T) {
	// Just verify it doesn't panic — result depends on environment
	_ = InsideTmux()
}
