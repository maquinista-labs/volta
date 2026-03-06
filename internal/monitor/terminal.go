package monitor

import (
	"strings"
	"unicode/utf8"
)

// Spinner characters used by Claude Code's status line.
const spinnerChars = "·✻✽✶✳✢"

// StripPaneChrome removes Claude Code's bottom chrome (separator, prompt, status bar)
// from captured pane text. Returns the text above the separator.
func StripPaneChrome(paneText string) string {
	lines := strings.Split(paneText, "\n")
	sepIdx := findChromeSeparator(lines)
	if sepIdx < 0 {
		return paneText
	}
	return strings.Join(lines[:sepIdx], "\n")
}

// ExtractStatusLine detects Claude's spinner/status from the terminal output.
// Returns the status text and whether a status was found.
// Matches CCBot logic: find topmost separator, then search above it for a line
// whose first character is a spinner. Stops at the first non-empty non-spinner line.
func ExtractStatusLine(paneText string) (string, bool) {
	lines := strings.Split(paneText, "\n")
	sepIdx := findChromeSeparator(lines)
	if sepIdx < 0 {
		return "", false
	}

	// Check lines above separator (skip blanks, up to 5 lines above)
	minIdx := sepIdx - 5
	if minIdx < 0 {
		minIdx = -1
	}
	for i := sepIdx - 1; i > minIdx; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(line)
		if strings.ContainsRune(spinnerChars, r) {
			return strings.TrimSpace(line[size:]), true
		}
		// First non-empty non-spinner line → no status
		return "", false
	}
	return "", false
}

// findChromeSeparator finds the line index of the topmost chrome separator
// (a line of ─ chars) in the last 10 lines. Searches top-down to find the
// first separator, which sits just below the status line in Claude Code's layout.
func findChromeSeparator(lines []string) int {
	start := len(lines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines); i++ {
		if isChromeSeparator(lines[i]) {
			return i
		}
	}
	return -1
}

// isChromeSeparator checks if a line is a chrome separator.
// All characters must be ─ or ━, with at least 20 runes (matches CCBot's strict check).
func isChromeSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if utf8.RuneCountInString(trimmed) < 20 {
		return false
	}
	for _, r := range trimmed {
		if r != '─' && r != '━' {
			return false
		}
	}
	return true
}

// extractAfterSpinner extracts the text after the first spinner character.
func extractAfterSpinner(line string) string {
	for i, r := range line {
		if strings.ContainsRune(spinnerChars, r) {
			rest := strings.TrimSpace(line[i+utf8.RuneLen(r):])
			return rest
		}
	}
	return ""
}

// UIPattern defines markers for detecting interactive UI elements.
type UIPattern struct {
	Name       string
	TopMarkers []string
	BotMarkers []string // empty = use last non-empty line
}

// UIContent holds extracted interactive content.
type UIContent struct {
	Name    string
	Content string
}

var uiPatterns = []UIPattern{
	{
		Name:       "ExitPlanMode",
		TopMarkers: []string{"Would you like to proceed?", "Claude has written up a plan"},
		BotMarkers: []string{"ctrl-g to edit", "Esc to"},
	},
	{
		Name:       "AskUserQuestion_multi",
		TopMarkers: []string{"← "},
		BotMarkers: nil, // last non-empty line
	},
	{
		Name:       "AskUserQuestion_single",
		TopMarkers: []string{"☐", "✔", "☒"},
		BotMarkers: []string{"Enter to select"},
	},
	{
		Name:       "PermissionPrompt",
		TopMarkers: []string{"Do you want to proceed?"},
		BotMarkers: []string{"Esc to cancel"},
	},
	{
		Name:       "RestoreCheckpoint",
		TopMarkers: []string{"Restore the code"},
		BotMarkers: []string{"Enter to continue"},
	},
	{
		Name:       "Settings",
		TopMarkers: []string{"Settings:"},
		BotMarkers: []string{"Esc to cancel", "Type to filter"},
	},
}

// IsInteractiveUI returns true if the pane text contains an interactive UI prompt.
func IsInteractiveUI(paneText string) bool {
	_, ok := ExtractInteractiveContent(paneText)
	return ok
}

// ExtractInteractiveContent extracts the interactive UI content from pane text.
// Returns the UI content and true if found.
func ExtractInteractiveContent(paneText string) (UIContent, bool) {
	stripped := StripPaneChrome(paneText)
	lines := strings.Split(stripped, "\n")

	for _, pattern := range uiPatterns {
		content, ok := tryExtract(lines, pattern)
		if ok {
			return content, true
		}
	}
	return UIContent{}, false
}

func tryExtract(lines []string, pattern UIPattern) (UIContent, bool) {
	// Find top marker
	topIdx := -1
	for i, line := range lines {
		for _, marker := range pattern.TopMarkers {
			if strings.Contains(line, marker) {
				topIdx = i
				break
			}
		}
		if topIdx >= 0 {
			break
		}
	}

	if topIdx < 0 {
		return UIContent{}, false
	}

	// Find bottom marker
	botIdx := -1
	if len(pattern.BotMarkers) == 0 {
		// Use last non-empty line
		for i := len(lines) - 1; i > topIdx; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				botIdx = i
				break
			}
		}
	} else {
		for i := topIdx + 1; i < len(lines); i++ {
			for _, marker := range pattern.BotMarkers {
				if strings.Contains(lines[i], marker) {
					botIdx = i
					break
				}
			}
			if botIdx >= 0 {
				break
			}
		}
	}

	if botIdx < 0 {
		return UIContent{}, false
	}

	// Extract content between markers
	content := strings.Join(lines[topIdx:botIdx+1], "\n")
	return UIContent{
		Name:    pattern.Name,
		Content: content,
	}, true
}

// ExtractBashOutput extracts ! command output from a captured tmux pane.
// Searches from the bottom for the "! <command>" echo line, then returns
// that line and everything below it. Returns empty string if not found.
func ExtractBashOutput(paneText, command string) string {
	stripped := StripPaneChrome(paneText)
	lines := strings.Split(stripped, "\n")

	// Match on the first 10 chars of the command to handle terminal truncation
	matchPrefix := command
	if len(matchPrefix) > 10 {
		matchPrefix = matchPrefix[:10]
	}

	// Search from bottom for the "! <command>" echo line
	cmdIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "! "+matchPrefix) || strings.HasPrefix(trimmed, "!"+matchPrefix) {
			cmdIdx = i
			break
		}
	}

	if cmdIdx < 0 {
		return ""
	}

	// Include the command echo line and everything after
	output := lines[cmdIdx:]

	// Strip trailing empty lines
	for len(output) > 0 && strings.TrimSpace(output[len(output)-1]) == "" {
		output = output[:len(output)-1]
	}

	if len(output) == 0 {
		return ""
	}

	return strings.Join(output, "\n")
}

// ShortenSeparators replaces long ─ lines with a shorter version for display.
func ShortenSeparators(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if isChromeSeparator(line) {
			lines[i] = "─────"
		}
	}
	return strings.Join(lines, "\n")
}
