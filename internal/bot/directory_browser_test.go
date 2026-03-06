package bot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildDirectoryBrowser_ListsDirs(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "alpha"), 0o755)
	os.Mkdir(filepath.Join(dir, "beta"), 0o755)
	os.Mkdir(filepath.Join(dir, ".hidden"), 0o755) // should be excluded
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hi"), 0o644)

	text, kb, dirs := buildDirectoryBrowser(dir, 0)

	if len(dirs) != 2 {
		t.Fatalf("expected 2 dirs, got %d: %v", len(dirs), dirs)
	}
	if dirs[0] != "alpha" || dirs[1] != "beta" {
		t.Errorf("dirs should be [alpha, beta], got %v", dirs)
	}
	if text == "" {
		t.Error("text should not be empty")
	}
	if len(kb.InlineKeyboard) == 0 {
		t.Error("keyboard should have rows")
	}
}

func TestBuildDirectoryBrowser_Pagination(t *testing.T) {
	dir := t.TempDir()
	// Create 8 directories to trigger pagination (dirsPerPage = 6)
	for i := 0; i < 8; i++ {
		os.Mkdir(filepath.Join(dir, "dir"+string(rune('a'+i))), 0o755)
	}

	_, kb, dirs := buildDirectoryBrowser(dir, 0)
	if len(dirs) != 8 {
		t.Fatalf("expected 8 dirs, got %d", len(dirs))
	}

	// Should have pagination row
	found := false
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData != nil && *btn.CallbackData == "dir_page:1" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected page 2 button")
	}

	// Page 1 should show remaining dirs
	_, kb2, _ := buildDirectoryBrowser(dir, 1)
	hasBack := false
	for _, row := range kb2.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData != nil && *btn.CallbackData == "dir_page:0" {
				hasBack = true
			}
		}
	}
	if !hasBack {
		t.Error("page 2 should have back button")
	}
}

func TestBuildDirectoryBrowser_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	text, kb, dirs := buildDirectoryBrowser(dir, 0)
	if len(dirs) != 0 {
		t.Errorf("expected 0 dirs, got %d", len(dirs))
	}
	if text == "" {
		t.Error("text should not be empty")
	}
	// Should still have action row (.. | Select | Cancel)
	if len(kb.InlineKeyboard) == 0 {
		t.Error("keyboard should have action row")
	}
}

func TestBuildDirectoryBrowser_InvalidPath(t *testing.T) {
	text, _, dirs := buildDirectoryBrowser("/nonexistent/path/that/does/not/exist", 0)
	if dirs != nil {
		t.Error("dirs should be nil for invalid path")
	}
	if text == "" {
		t.Error("should return error text")
	}
}

func TestBuildDirectoryBrowser_ActionRow(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "sub"), 0o755)

	_, kb, _ := buildDirectoryBrowser(dir, 0)

	// Last row should be the action row
	lastRow := kb.InlineKeyboard[len(kb.InlineKeyboard)-1]
	if len(lastRow) != 3 {
		t.Fatalf("action row should have 3 buttons, got %d", len(lastRow))
	}

	expected := []string{"dir_up", "dir_confirm", "dir_cancel"}
	for i, btn := range lastRow {
		if btn.CallbackData == nil || *btn.CallbackData != expected[i] {
			t.Errorf("action button %d: got %v, want %s", i, btn.CallbackData, expected[i])
		}
	}
}

func TestTruncateName(t *testing.T) {
	tests := []struct {
		name   string
		maxLen int
		want   string
	}{
		{"short", 13, "short"},
		{"exactly13char", 13, "exactly13char"},
		{"this-is-a-very-long-name", 13, "this-is-a-ve\u2026"},
		{"ab", 3, "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateName(tt.name, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateName(%q, %d) = %q, want %q", tt.name, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		path string
		want string
	}{
		{home + "/code/project", "~/code/project"},
		{"/tmp/foo", "/tmp/foo"},
		{home, "~"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := shortenPath(tt.path)
			if got != tt.want {
				t.Errorf("shortenPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBuildDirectoryBrowser_SortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "zebra"), 0o755)
	os.Mkdir(filepath.Join(dir, "apple"), 0o755)
	os.Mkdir(filepath.Join(dir, "mango"), 0o755)

	_, _, dirs := buildDirectoryBrowser(dir, 0)
	if len(dirs) != 3 {
		t.Fatalf("expected 3 dirs, got %d", len(dirs))
	}
	if dirs[0] != "apple" || dirs[1] != "mango" || dirs[2] != "zebra" {
		t.Errorf("dirs not sorted: %v", dirs)
	}
}

func TestBuildDirectoryBrowser_PageBounds(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "a"), 0o755)

	// Page -1 should clamp to 0
	_, _, dirs := buildDirectoryBrowser(dir, -1)
	if len(dirs) != 1 {
		t.Errorf("expected 1 dir, got %d", len(dirs))
	}

	// Page 999 should clamp to last page
	_, _, dirs = buildDirectoryBrowser(dir, 999)
	if len(dirs) != 1 {
		t.Errorf("expected 1 dir, got %d", len(dirs))
	}
}
