package render

import (
	"strings"
	"testing"
)

func TestFormatToolUse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Read", "main.go", "● **Read**(main.go)"},
		{"Bash", "git status", "● **Bash**(git status)"},
		{"Task", "", "● **Task**()"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatToolUse(tt.name, tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatToolResult_Read(t *testing.T) {
	content := "line1\nline2\nline3\n"
	got := FormatToolResult("Read", "main.go", content, false)
	if !strings.Contains(got, "● **Read**(main.go)") {
		t.Errorf("missing header in %q", got)
	}
	if !strings.Contains(got, "⎿ Read 3 lines") {
		t.Errorf("missing result in %q", got)
	}
}

func TestFormatToolResult_Write(t *testing.T) {
	content := "a\nb\n"
	got := FormatToolResult("Write", "file.go", content, false)
	if !strings.Contains(got, "⎿ Wrote 2 lines") {
		t.Errorf("got %q, want 'Wrote 2 lines'", got)
	}
}

func TestFormatToolResult_Bash(t *testing.T) {
	content := "file1\nfile2\nfile3"
	got := FormatToolResult("Bash", "ls", content, false)
	if !strings.Contains(got, "● **Bash**(ls)") {
		t.Errorf("missing header in %q", got)
	}
	if !strings.Contains(got, "⎿ file1") {
		t.Errorf("missing preview in %q", got)
	}
}

func TestFormatToolResult_BashLong(t *testing.T) {
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "output line"
	}
	content := strings.Join(lines, "\n")
	got := FormatToolResult("Bash", "make", content, false)
	if !strings.Contains(got, "… +7 lines") {
		t.Errorf("should show truncation, got %q", got)
	}
}

func TestFormatToolResult_BashEmpty(t *testing.T) {
	got := FormatToolResult("Bash", "go build", "", false)
	if !strings.Contains(got, "⎿ (No output)") {
		t.Errorf("got %q, want '(No output)'", got)
	}
}

func TestFormatToolResult_Grep(t *testing.T) {
	content := "src/main.go:10: TODO fix this\nsrc/lib.go:20: TODO refactor\n"
	got := FormatToolResult("Grep", "TODO", content, false)
	if !strings.Contains(got, "Found 2 matches") {
		t.Errorf("got %q, should contain 'Found 2 matches'", got)
	}
}

func TestFormatToolResult_Glob(t *testing.T) {
	content := "main.go\nutil.go\n"
	got := FormatToolResult("Glob", "*.go", content, false)
	if !strings.Contains(got, "Found 2 files") {
		t.Errorf("got %q, should contain 'Found 2 files'", got)
	}
}

func TestFormatToolResult_Edit(t *testing.T) {
	content := "--- a/file.go\n+++ b/file.go\n-old line\n+new line\n+another new line"
	got := FormatToolResult("Edit", "file.go", content, false)
	if !strings.Contains(got, "Added 2, removed 1") {
		t.Errorf("got %q, should contain 'Added 2, removed 1'", got)
	}
}

func TestFormatToolResult_EditSuccess(t *testing.T) {
	content := "The file /path/to/file.go has been updated successfully."
	got := FormatToolResult("Edit", "file.go", content, false)
	if !strings.Contains(got, "The file /path/to/file.go has been updated successfully.") {
		t.Errorf("got %q, should show success message", got)
	}
	if strings.Contains(got, "Added 0") {
		t.Errorf("should not show 'Added 0'")
	}
}

func TestFormatToolResult_Task(t *testing.T) {
	content := "Searching...\nFound 3 results\nDone."
	got := FormatToolResult("Task", "explore", content, false)
	if !strings.Contains(got, "Agent output 3 lines") {
		t.Errorf("got %q, should contain 'Agent output 3 lines'", got)
	}
}

func TestFormatToolResult_WebFetch(t *testing.T) {
	content := "some html content here"
	got := FormatToolResult("WebFetch", "https://example.com", content, false)
	if !strings.Contains(got, "Fetched 22 characters") {
		t.Errorf("got %q, should contain 'Fetched 22 characters'", got)
	}
}

func TestFormatToolResult_WebSearch(t *testing.T) {
	content := "1. First result\n2. Second result\n"
	got := FormatToolResult("WebSearch", "test query", content, false)
	if !strings.Contains(got, "2 search results") {
		t.Errorf("got %q, should contain '2 search results'", got)
	}
}

func TestFormatToolResult_Error(t *testing.T) {
	content := "command not found: xyz"
	got := FormatToolResult("Bash", "xyz", content, true)
	if !strings.Contains(got, "Error: command not found: xyz") {
		t.Errorf("got %q, should contain error", got)
	}
}

func TestFormatToolResult_ErrorMultiline(t *testing.T) {
	content := "error line 1\nerror line 2\nerror line 3"
	got := FormatToolResult("Bash", "cmd", content, true)
	if !strings.Contains(got, ExpQuoteStart) {
		t.Error("multiline error should have expandable quote")
	}
}

func TestFormatThinking(t *testing.T) {
	text := "Let me think about this problem..."
	got := FormatThinking(text)
	if !strings.Contains(got, ExpQuoteStart) {
		t.Error("thinking should be wrapped in expandable quote")
	}
	if !strings.Contains(got, "Let me think") {
		t.Error("should contain the thinking text")
	}
}

func TestFormatThinking_Truncation(t *testing.T) {
	long := strings.Repeat("x", 600)
	got := FormatThinking(long)
	// Extract content between markers
	content := strings.TrimPrefix(got, ExpQuoteStart)
	content = strings.TrimSuffix(content, ExpQuoteEnd)
	if len(content) > 504 { // 500 + "..."
		t.Errorf("thinking not truncated: %d chars", len(content))
	}
}

func TestTruncateContent(t *testing.T) {
	short := "hello"
	if truncateContent(short, 100) != "hello" {
		t.Error("short content should not be truncated")
	}

	long := strings.Repeat("x", 200)
	got := truncateContent(long, 100)
	if !strings.HasSuffix(got, "... (truncated)") {
		t.Error("long content should be truncated")
	}
}

func TestCountNonEmpty(t *testing.T) {
	lines := []string{"a", "", "b", "  ", "c"}
	got := countNonEmpty(lines)
	if got != 3 {
		t.Errorf("countNonEmpty = %d, want 3", got)
	}
}

func TestCountEditChanges(t *testing.T) {
	diff := "--- a/file\n+++ b/file\n-removed\n+added1\n+added2\n context"
	added, removed := countEditChanges(diff)
	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
}

func TestFormatToolResult_EmptyContent(t *testing.T) {
	got := FormatToolResult("Read", "file.go", "", false)
	if !strings.Contains(got, "⎿ (No output)") {
		t.Errorf("got %q, want '(No output)'", got)
	}
}

func TestFormatPreview(t *testing.T) {
	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	got := formatPreview(lines, 5)
	if !strings.Contains(got, "line1") {
		t.Error("should include first line")
	}
	if !strings.Contains(got, "… +2 lines") {
		t.Errorf("should show truncation, got %q", got)
	}
	if strings.Contains(got, "line4") {
		t.Error("should not include line4")
	}
}
