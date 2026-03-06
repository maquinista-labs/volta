package render

import (
	"fmt"
	"strings"
)

// Sentinel markers for expandable quotes. These are replaced during MarkdownV2 conversion.
const (
	ExpQuoteStart = "\x02EXPQUOTE_START\x02"
	ExpQuoteEnd   = "\x02EXPQUOTE_END\x02"
)

// previewLines is how many content lines to show before truncating with "… +N lines".
const previewLines = 3

// FormatToolUse formats a tool_use block as the initial message (before result arrives).
func FormatToolUse(name, input string) string {
	return "● " + toolHeader(name, input)
}

// FormatToolResult formats a tool_result combined with its tool_use header.
// The result replaces the tool_use message, so it includes both the header and result.
func FormatToolResult(toolName, toolInput, content string, isError bool) string {
	header := "● " + toolHeader(toolName, toolInput)

	if isError {
		return header + "\n  ⎿ " + formatErrorBody(content)
	}

	body := formatResultBody(toolName, content)
	return header + "\n  ⎿ " + body
}

// FormatThinking formats a thinking block: truncate to 500 chars and wrap in expandable quote.
func FormatThinking(text string) string {
	truncated := text
	if len(truncated) > 500 {
		truncated = truncated[:500] + "..."
	}
	return formatExpandableQuote(truncated)
}

// FormatText strips system tags and returns clean text.
func FormatText(text string) string {
	return text
}

// toolHeader builds "**Name**(input)" or "**Name**()" for use in tool formatting.
func toolHeader(name, input string) string {
	if input != "" {
		return fmt.Sprintf("**%s**(%s)", name, input)
	}
	return fmt.Sprintf("**%s**()", name)
}

// formatResultBody produces the result body for a given tool type.
func formatResultBody(toolName, content string) string {
	if content == "" {
		return "(No output)"
	}

	lines := strings.Split(content, "\n")
	lineCount := len(lines)
	if lineCount > 0 && lines[lineCount-1] == "" {
		lineCount--
		lines = lines[:lineCount]
	}

	switch toolName {
	case "Read":
		return fmt.Sprintf("Read %d lines", lineCount)
	case "Write":
		return fmt.Sprintf("Wrote %d lines", lineCount)
	case "Edit":
		added, removed := countEditChanges(content)
		if added > 0 || removed > 0 {
			return fmt.Sprintf("Added %d, removed %d", added, removed)
		}
		// No diff — show first line (e.g. "The file ... has been updated successfully.")
		return firstLine(content)
	case "Bash":
		return formatPreview(lines, lineCount)
	case "Grep":
		matchCount := countNonEmpty(lines)
		summary := fmt.Sprintf("Found %d matches", matchCount)
		if matchCount > 0 {
			summary += "\n" + formatPreviewQuote(content)
		}
		return summary
	case "Glob":
		fileCount := countNonEmpty(lines)
		summary := fmt.Sprintf("Found %d files", fileCount)
		if fileCount > 0 {
			summary += "\n" + formatPreviewQuote(content)
		}
		return summary
	case "Task":
		return fmt.Sprintf("Agent output %d lines", lineCount)
	case "WebFetch":
		return fmt.Sprintf("Fetched %d characters", len(content))
	case "WebSearch":
		resultCount := countSearchResults(content)
		summary := fmt.Sprintf("%d search results", resultCount)
		if content != "" {
			summary += "\n" + formatPreviewQuote(content)
		}
		return summary
	default:
		return formatPreview(lines, lineCount)
	}
}

// formatPreview shows up to previewLines of content, then "… +N lines".
func formatPreview(lines []string, totalLines int) string {
	if totalLines == 0 {
		return "(No output)"
	}

	show := lines
	if len(show) > previewLines {
		show = show[:previewLines]
	}

	var b strings.Builder
	for i, line := range show {
		if i > 0 {
			b.WriteString("\n     ")
		}
		b.WriteString(line)
	}

	remaining := totalLines - len(show)
	if remaining > 0 {
		b.WriteString(fmt.Sprintf("\n     … +%d lines", remaining))
	}

	return b.String()
}

// formatPreviewQuote wraps content in an expandable quote, truncated.
func formatPreviewQuote(content string) string {
	return formatExpandableQuote(truncateContent(content, 3000))
}

// formatErrorBody formats error content for display after ⎿.
func formatErrorBody(content string) string {
	lines := strings.SplitN(content, "\n", 2)
	first := lines[0]
	if len(first) > 100 {
		first = first[:100] + "..."
	}
	result := "Error: " + first
	if len(lines) > 1 {
		result += "\n" + formatExpandableQuote(truncateContent(content, 3000))
	}
	return result
}

// formatExpandableQuote wraps text in expandable quote markers.
func formatExpandableQuote(text string) string {
	return ExpQuoteStart + text + ExpQuoteEnd
}

// truncateContent truncates content to maxLen characters.
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n... (truncated)"
}

// firstLine returns the first line of text.
func firstLine(text string) string {
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return text[:idx]
	}
	return text
}

// countNonEmpty counts non-empty lines.
func countNonEmpty(lines []string) int {
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// countEditChanges counts added and removed lines from a diff-like content.
func countEditChanges(content string) (added, removed int) {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removed++
		}
	}
	return
}

// countSearchResults counts the number of search results (lines starting with a number or bullet).
func countSearchResults(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && (len(trimmed) > 0 && (trimmed[0] >= '1' && trimmed[0] <= '9' || trimmed[0] == '-' || trimmed[0] == '*')) {
			count++
		}
	}
	if count == 0 {
		count = 1 // at least 1 if there's any content
	}
	return count
}
