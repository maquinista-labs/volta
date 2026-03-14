package monitor

import (
	"fmt"
	"sync"
)

// ActiveSession represents a discovered active agent session.
type ActiveSession struct {
	Key      string // unique key for offset tracking
	WindowID string // tmux window ID
}

// TranscriptSource defines the interface for pluggable transcript sources.
// Each source knows how to discover sessions, read new entries, and detect
// UI state for a specific agent runtime (Claude Code, OpenCode, etc.).
type TranscriptSource interface {
	// Name returns the source's identifier (e.g. "claude", "opencode").
	Name() string

	// DiscoverSessions returns all currently active sessions for this source.
	DiscoverSessions() []ActiveSession

	// ReadNewEntries reads new transcript entries from a session starting at the given offset.
	// Returns parsed entries, the new offset, and any error.
	ReadNewEntries(session ActiveSession, lastOffset int64) ([]ParsedEntry, int64, error)

	// ExtractStatusLine detects the agent's status from terminal output.
	// Returns the status text and whether a status was found.
	ExtractStatusLine(paneText string) (string, bool)

	// IsInteractiveUI returns true if the pane text contains an interactive UI prompt.
	IsInteractiveUI(paneText string) bool
}

var (
	sourceMu sync.RWMutex
	sources  = make(map[string]TranscriptSource)
)

// RegisterSource adds a TranscriptSource to the global registry.
func RegisterSource(name string, src TranscriptSource) {
	sourceMu.Lock()
	defer sourceMu.Unlock()
	sources[name] = src
}

// GetSource returns a registered TranscriptSource by name.
func GetSource(name string) (TranscriptSource, error) {
	sourceMu.RLock()
	defer sourceMu.RUnlock()
	src, ok := sources[name]
	if !ok {
		return nil, fmt.Errorf("unknown transcript source: %q", name)
	}
	return src, nil
}
