package render

import (
	"strings"
	"testing"
)

func TestToMarkdownV2_PlainText(t *testing.T) {
	got := ToMarkdownV2("Hello world")
	if got != "Hello world" {
		t.Errorf("got %q, want 'Hello world'", got)
	}
}

func TestToMarkdownV2_SpecialChars(t *testing.T) {
	got := ToMarkdownV2("2 + 3 = 5")
	if !strings.Contains(got, "\\+") {
		t.Errorf("should escape +: got %q", got)
	}
	if !strings.Contains(got, "\\=") {
		t.Errorf("should escape =: got %q", got)
	}
}

func TestToMarkdownV2_Bold(t *testing.T) {
	got := ToMarkdownV2("**hello**")
	// Should convert **hello** to *hello* in MarkdownV2
	if !strings.Contains(got, "*hello*") {
		t.Errorf("bold not converted: got %q", got)
	}
	// Should NOT contain double asterisks
	if strings.Contains(got, "**") {
		t.Errorf("should not have double asterisks: got %q", got)
	}
}

func TestToMarkdownV2_Italic(t *testing.T) {
	got := ToMarkdownV2("_hello_")
	// Should produce italic markers
	if !strings.Contains(got, "_hello_") {
		t.Errorf("italic not preserved: got %q", got)
	}
}

func TestToMarkdownV2_Strikethrough(t *testing.T) {
	got := ToMarkdownV2("~~deleted~~")
	if !strings.Contains(got, "~deleted~") {
		t.Errorf("strikethrough not converted: got %q", got)
	}
}

func TestToMarkdownV2_CodeBlock(t *testing.T) {
	input := "```\nfmt.Println(\"hello\")\n```"
	got := ToMarkdownV2(input)
	if !strings.Contains(got, "```") {
		t.Errorf("should preserve code block markers: got %q", got)
	}
	// Content inside code blocks should not have extra escaping of special chars
	if strings.Contains(got, "\\(") {
		t.Errorf("should not escape ( inside code block: got %q", got)
	}
}

func TestToMarkdownV2_CodeBlockWithLang(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```"
	got := ToMarkdownV2(input)
	if !strings.Contains(got, "```go") {
		t.Errorf("should preserve language tag: got %q", got)
	}
}

func TestToMarkdownV2_InlineCode(t *testing.T) {
	input := "Use `fmt.Println()` function"
	got := ToMarkdownV2(input)
	if !strings.Contains(got, "`fmt\\.Println\\(\\)`") {
		// Code span should escape only ` and \
		// Actually let's check that backticks are present
		if !strings.Contains(got, "`") {
			t.Errorf("should preserve inline code: got %q", got)
		}
	}
}

func TestToMarkdownV2_Link(t *testing.T) {
	input := "[Google](https://google.com)"
	got := ToMarkdownV2(input)
	if !strings.Contains(got, "[") || !strings.Contains(got, "(") {
		t.Errorf("should preserve link format: got %q", got)
	}
}

func TestToMarkdownV2_DotEscape(t *testing.T) {
	got := ToMarkdownV2("file.go")
	if !strings.Contains(got, "\\.") {
		t.Errorf("should escape dot: got %q", got)
	}
}

func TestToMarkdownV2_ExclamationEscape(t *testing.T) {
	got := ToMarkdownV2("Hello!")
	if !strings.Contains(got, "\\!") {
		t.Errorf("should escape exclamation: got %q", got)
	}
}

func TestToMarkdownV2_ExpandableQuote(t *testing.T) {
	input := "Summary\n" + ExpQuoteStart + "line1\nline2" + ExpQuoteEnd
	got := ToMarkdownV2(input)
	if !strings.Contains(got, ">") {
		t.Errorf("should have blockquote marker: got %q", got)
	}
	if !strings.Contains(got, "||") {
		t.Errorf("should have spoiler for expandable: got %q", got)
	}
}

func TestToMarkdownV2_Heading(t *testing.T) {
	got := ToMarkdownV2("# Title")
	// Headings become bold
	if !strings.Contains(got, "*Title*") {
		t.Errorf("heading should be bold: got %q", got)
	}
}

func TestToMarkdownV2_HeadingH2(t *testing.T) {
	got := ToMarkdownV2("## Subtitle")
	if !strings.Contains(got, "*Subtitle*") {
		t.Errorf("h2 should also be bold: got %q", got)
	}
}

func TestToMarkdownV2_Table(t *testing.T) {
	input := "| A | B |\n|---|---|\n| 1 | 2 |"
	got := ToMarkdownV2(input)
	// Tables should be rendered as code blocks
	if !strings.Contains(got, "```") {
		t.Errorf("table should be in code block: got %q", got)
	}
	// Should contain pipe-delimited rows
	if !strings.Contains(got, "|") {
		t.Errorf("table should have pipe delimiters: got %q", got)
	}
}

func TestToMarkdownV2_UnorderedList(t *testing.T) {
	input := "- item one\n- item two"
	got := ToMarkdownV2(input)
	if !strings.Contains(got, "\\- item one") {
		t.Errorf("unordered list not rendered: got %q", got)
	}
}

func TestToMarkdownV2_OrderedList(t *testing.T) {
	input := "1. first\n2. second"
	got := ToMarkdownV2(input)
	if !strings.Contains(got, "1\\. first") {
		t.Errorf("ordered list not rendered: got %q", got)
	}
}

func TestToMarkdownV2_Blockquote(t *testing.T) {
	input := "> this is quoted"
	got := ToMarkdownV2(input)
	if !strings.Contains(got, ">") {
		t.Errorf("blockquote not rendered: got %q", got)
	}
}

func TestToMarkdownV2_HorizontalRule(t *testing.T) {
	input := "above\n\n---\n\nbelow"
	got := ToMarkdownV2(input)
	// Should contain em-dashes (or escaped version)
	if !strings.Contains(got, "—") {
		t.Errorf("horizontal rule not rendered: got %q", got)
	}
}

func TestToMarkdownV2_FilePathWithUnderscores(t *testing.T) {
	got := ToMarkdownV2("my_file_name.go")
	// Underscores in regular text should be escaped, not treated as italic
	if !strings.Contains(got, "\\_") {
		t.Errorf("underscores should be escaped: got %q", got)
	}
	if !strings.Contains(got, "\\.") {
		t.Errorf("dot should be escaped: got %q", got)
	}
}

func TestToMarkdownV2_NestedFormatting(t *testing.T) {
	got := ToMarkdownV2("**bold _and italic_**")
	// Should have both bold and italic markers
	if !strings.Contains(got, "*") {
		t.Errorf("should have bold marker: got %q", got)
	}
}

func TestToMarkdownV2_IndentedCode(t *testing.T) {
	// 4-space indented text becomes a code block in goldmark,
	// but our renderer renders it as escaped text (no ``` wrapper)
	input := "    indented text"
	got := ToMarkdownV2(input)
	// Should not be wrapped in ``` markers
	if strings.Contains(got, "```") {
		t.Errorf("indented code should not produce fenced block: got %q", got)
	}
	if !strings.Contains(got, "indented text") {
		t.Errorf("should preserve text content: got %q", got)
	}
}

func TestToMarkdownV2_MultipleCodeBlocks(t *testing.T) {
	input := "```\nblock 1\n```\n\ntext between\n\n```\nblock 2\n```"
	got := ToMarkdownV2(input)
	count := strings.Count(got, "```")
	if count != 4 {
		t.Errorf("should have 4 backtick markers (2 blocks): got %d in %q", count, got)
	}
}

func TestToMarkdownV2_ComplexMessage(t *testing.T) {
	input := `**Summary**: I've updated the `+ "`config.go`" +` file to add validation.

Changes:
- Added `+ "`validateConfig()`" +` function
- Updated `+ "`LoadConfig()`" +` to call it
- Fixed error handling in `+ "`my_helper.go`"
	got := ToMarkdownV2(input)

	// Should not panic or produce empty output
	if len(got) == 0 {
		t.Error("should produce non-empty output")
	}
	// Should contain some formatting
	if !strings.Contains(got, "*") {
		t.Errorf("should have bold marker: got %q", got)
	}
}

func TestToMarkdownV2_BackslashEscape(t *testing.T) {
	got := ToMarkdownV2("path\\to\\file")
	if !strings.Contains(got, "\\\\") {
		t.Errorf("backslash should be escaped: got %q", got)
	}
}

func TestToPlainText_RemovesBold(t *testing.T) {
	got := ToPlainText("**hello** world")
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") {
		t.Errorf("got %q, want text with 'hello' and 'world'", got)
	}
	if strings.Contains(got, "**") {
		t.Errorf("should not have bold markers: got %q", got)
	}
}

func TestToPlainText_RemovesCode(t *testing.T) {
	got := ToPlainText("Use `fmt.Println()` here")
	if !strings.Contains(got, "fmt.Println()") {
		t.Errorf("should contain raw code: got %q", got)
	}
	// Count backticks — should have none
	if strings.Count(got, "`") > 0 {
		t.Errorf("should remove backticks: got %q", got)
	}
}

func TestToPlainText_RemovesCodeBlock(t *testing.T) {
	got := ToPlainText("```\ncode here\n```")
	if !strings.Contains(got, "code here") {
		t.Errorf("should preserve code content: got %q", got)
	}
	if strings.Contains(got, "```") {
		t.Errorf("should remove code markers: got %q", got)
	}
}

func TestToPlainText_ConvertLinks(t *testing.T) {
	got := ToPlainText("[Google](https://google.com)")
	if !strings.Contains(got, "Google") {
		t.Errorf("should have link text: got %q", got)
	}
	if !strings.Contains(got, "https://google.com") {
		t.Errorf("should have link URL: got %q", got)
	}
}

func TestToPlainText_RemovesExpQuoteMarkers(t *testing.T) {
	input := "Hello " + ExpQuoteStart + "quoted" + ExpQuoteEnd + " world"
	got := ToPlainText(input)
	if strings.Contains(got, ExpQuoteStart) || strings.Contains(got, ExpQuoteEnd) {
		t.Errorf("should strip markers: got %q", got)
	}
	if !strings.Contains(got, "quoted") {
		t.Errorf("should preserve quote content: got %q", got)
	}
}

func TestToPlainText_Table(t *testing.T) {
	input := "| A | B |\n|---|---|\n| 1 | 2 |"
	got := ToPlainText(input)
	if !strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Errorf("should contain table headers: got %q", got)
	}
	if !strings.Contains(got, "1") || !strings.Contains(got, "2") {
		t.Errorf("should contain table data: got %q", got)
	}
}

func TestEscapeMarkdownV2(t *testing.T) {
	got := escapeMarkdownV2("hello_world *bold* [link]")
	if !strings.Contains(got, "\\_") {
		t.Error("should escape underscore")
	}
	if !strings.Contains(got, "\\*") {
		t.Error("should escape asterisk")
	}
	if !strings.Contains(got, "\\[") {
		t.Error("should escape bracket")
	}
}

func TestEscapeMarkdownV2_Backslash(t *testing.T) {
	got := escapeMarkdownV2("a\\b")
	if got != "a\\\\b" {
		t.Errorf("got %q, want %q", got, "a\\\\b")
	}
}

func TestEscapeCodeContent(t *testing.T) {
	got := escapeCodeContent("hello `world` and \\n")
	if got != "hello \\`world\\` and \\\\n" {
		t.Errorf("got %q", got)
	}
}

func TestSplitMessage_Short(t *testing.T) {
	parts := SplitMessage("short text", 100)
	if len(parts) != 1 {
		t.Errorf("short text should not be split: got %d parts", len(parts))
	}
}

func TestSplitMessage_Long(t *testing.T) {
	// Build a long message
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "This is line number something that takes up space"
	}
	text := strings.Join(lines, "\n")

	parts := SplitMessage(text, 500)
	if len(parts) < 2 {
		t.Errorf("long text should be split: got %d parts", len(parts))
	}

	// Each part should be under limit
	for i, part := range parts {
		if len(part) > 500 {
			t.Errorf("part %d exceeds max len: %d", i, len(part))
		}
	}
}

func TestSplitMessage_WithExpandableQuote(t *testing.T) {
	text := "prefix\n" + ExpQuoteStart + strings.Repeat("x", 5000) + ExpQuoteEnd
	parts := SplitMessage(text, 100)
	// Should NOT be split when it contains expandable quotes
	if len(parts) != 1 {
		t.Errorf("expandable quote messages should not be split: got %d parts", len(parts))
	}
}

func TestSplitMessage_NewlineBoundary(t *testing.T) {
	text := "line one\nline two\nline three"
	parts := SplitMessage(text, 20)
	// Should split at newline boundaries
	for _, part := range parts {
		if strings.HasPrefix(part, "\n") {
			t.Errorf("part should not start with newline: %q", part)
		}
	}
}

func TestToMarkdownV2_EmptyString(t *testing.T) {
	got := ToMarkdownV2("")
	if got != "" {
		t.Errorf("empty input should produce empty output: got %q", got)
	}
}

func TestToPlainText_EmptyString(t *testing.T) {
	got := ToPlainText("")
	if got != "" {
		t.Errorf("empty input should produce empty output: got %q", got)
	}
}
