package bot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const filesPerPage = 8

type fileBrowseEntry struct {
	Name  string
	IsDir bool
}

// FileBrowseState holds per-user file browser state.
type FileBrowseState struct {
	CurrentPath string
	Page        int
	Entries     []fileBrowseEntry // cached for index-based callbacks
	MessageID   int
	ChatID      int64
	ThreadID    int
}

// showFileBrowser sends the file browser keyboard to the user.
func (b *Bot) showFileBrowser(chatID int64, threadID int, userID int64, startPath string) {
	text, keyboard, entries := buildFileBrowser(startPath, 0)

	msg, err := b.sendMessageWithKeyboard(chatID, threadID, text, keyboard)
	if err != nil {
		log.Printf("Error sending file browser: %v", err)
		return
	}

	b.mu.Lock()
	b.fileBrowseStates[userID] = &FileBrowseState{
		CurrentPath: startPath,
		Page:        0,
		Entries:     entries,
		MessageID:   msg.MessageID,
		ChatID:      chatID,
		ThreadID:    threadID,
	}
	b.mu.Unlock()
}

// buildFileBrowser builds the inline keyboard for file browsing.
// Returns the display text, keyboard markup, and cached entries.
func buildFileBrowser(currentPath string, page int) (string, tgbotapi.InlineKeyboardMarkup, []fileBrowseEntry) {
	dirEntries, err := os.ReadDir(currentPath)
	if err != nil {
		return fmt.Sprintf("Error reading %s", shortenPath(currentPath)), tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("..", "get_up"),
				tgbotapi.NewInlineKeyboardButtonData("Cancel", "get_cancel"),
			),
		), nil
	}

	// Separate dirs and files, skip hidden entries
	var dirs, files []fileBrowseEntry
	for _, e := range dirEntries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// Follow symlinks to determine if target is a directory
		info, err := os.Stat(filepath.Join(currentPath, e.Name()))
		if err != nil {
			continue
		}
		entry := fileBrowseEntry{Name: e.Name(), IsDir: info.IsDir()}
		if info.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	// Sort each group alphabetically
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	// Directories first, then files
	entries := append(dirs, files...)

	totalPages := (len(entries) + filesPerPage - 1) / filesPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	if page < 0 {
		page = 0
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	// Entry buttons (2 per row)
	start := page * filesPerPage
	end := start + filesPerPage
	if end > len(entries) {
		end = len(entries)
	}

	for i := start; i < end; i += 2 {
		var row []tgbotapi.InlineKeyboardButton
		label := entries[i].Name
		if entries[i].IsDir {
			label = "\U0001F4C1 " + truncateName(label, 11)
		} else {
			label = truncateName(label, 13)
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			label,
			fmt.Sprintf("get_sel:%d", i),
		))
		if i+1 < end {
			label2 := entries[i+1].Name
			if entries[i+1].IsDir {
				label2 = "\U0001F4C1 " + truncateName(label2, 11)
			} else {
				label2 = truncateName(label2, 13)
			}
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(
				label2,
				fmt.Sprintf("get_sel:%d", i+1),
			))
		}
		rows = append(rows, row)
	}

	// Pagination row
	if totalPages > 1 {
		var paginationRow []tgbotapi.InlineKeyboardButton
		if page > 0 {
			paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData(
				"\u25C0", fmt.Sprintf("get_page:%d", page-1),
			))
		}
		paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d/%d", page+1, totalPages),
			"get_noop",
		))
		if page < totalPages-1 {
			paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData(
				"\u25B6", fmt.Sprintf("get_page:%d", page+1),
			))
		}
		rows = append(rows, paginationRow)
	}

	// Action row: .. | Cancel
	actionRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("..", "get_up"),
		tgbotapi.NewInlineKeyboardButtonData("Cancel", "get_cancel"),
	}
	rows = append(rows, actionRow)

	displayPath := shortenPath(currentPath)
	headerText := fmt.Sprintf("Browse files:\n%s (%d dirs, %d files)", displayPath, len(dirs), len(files))
	if len(entries) == 0 {
		headerText = fmt.Sprintf("Browse files:\n%s (empty directory)", displayPath)
	}

	return headerText, tgbotapi.NewInlineKeyboardMarkup(rows...), entries
}

// processFileBrowserCallback handles file browser callback queries.
func (b *Bot) processFileBrowserCallback(cq *tgbotapi.CallbackQuery) {
	userID := cq.From.ID
	data := cq.Data

	b.mu.RLock()
	fs, ok := b.fileBrowseStates[userID]
	b.mu.RUnlock()

	if !ok {
		return
	}

	// Verify topic match
	threadID := getThreadID(cq.Message)
	if threadID != fs.ThreadID {
		return
	}

	switch {
	case strings.HasPrefix(data, "get_sel:"):
		b.handleGetSelect(cq, fs, userID)
	case strings.HasPrefix(data, "get_page:"):
		b.handleGetPage(cq, fs, userID)
	case data == "get_up":
		b.handleGetUp(cq, fs, userID)
	case data == "get_cancel":
		b.handleGetCancel(cq, fs, userID)
	case data == "get_noop":
		// Do nothing — page indicator button
	}
}

func (b *Bot) handleGetSelect(cq *tgbotapi.CallbackQuery, fs *FileBrowseState, userID int64) {
	idxStr := strings.TrimPrefix(cq.Data, "get_sel:")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(fs.Entries) {
		return
	}

	entry := fs.Entries[idx]
	fullPath := filepath.Join(fs.CurrentPath, entry.Name)

	if entry.IsDir {
		// Navigate into directory
		text, keyboard, entries := buildFileBrowser(fullPath, 0)
		b.editMessageWithKeyboard(fs.ChatID, fs.MessageID, text, keyboard)

		b.mu.Lock()
		fs.CurrentPath = fullPath
		fs.Page = 0
		fs.Entries = entries
		b.mu.Unlock()
		return
	}

	// It's a file — stat for size check
	info, err := os.Stat(fullPath)
	if err != nil {
		b.showFileBrowserError(fs, fmt.Sprintf("Error: %v", err))
		return
	}

	const maxFileSize = 50 * 1024 * 1024 // 50MB
	if info.Size() > maxFileSize {
		b.showFileBrowserError(fs, fmt.Sprintf("File too large: %s (%d MB limit is 50 MB)",
			entry.Name, info.Size()/(1024*1024)))
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		b.showFileBrowserError(fs, fmt.Sprintf("Error reading file: %v", err))
		return
	}

	// Send file as document
	_, err = b.sendDocumentInThread(fs.ChatID, fs.ThreadID, data, entry.Name, tgbotapi.InlineKeyboardMarkup{})
	if err != nil {
		b.showFileBrowserError(fs, fmt.Sprintf("Error sending file: %v", err))
		return
	}

	// Success — edit browser message and clean up state
	b.editMessageText(fs.ChatID, fs.MessageID, fmt.Sprintf("Sent: %s", entry.Name))

	b.mu.Lock()
	delete(b.fileBrowseStates, userID)
	b.mu.Unlock()
}

// showFileBrowserError shows an error in the browser message but keeps state alive.
func (b *Bot) showFileBrowserError(fs *FileBrowseState, errMsg string) {
	text, keyboard, entries := buildFileBrowser(fs.CurrentPath, fs.Page)
	// Prepend error to the header text
	text = errMsg + "\n\n" + text
	b.editMessageWithKeyboard(fs.ChatID, fs.MessageID, text, keyboard)

	b.mu.Lock()
	fs.Entries = entries
	b.mu.Unlock()
}

func (b *Bot) handleGetPage(cq *tgbotapi.CallbackQuery, fs *FileBrowseState, userID int64) {
	pageStr := strings.TrimPrefix(cq.Data, "get_page:")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return
	}

	text, keyboard, entries := buildFileBrowser(fs.CurrentPath, page)
	b.editMessageWithKeyboard(fs.ChatID, fs.MessageID, text, keyboard)

	b.mu.Lock()
	fs.Page = page
	fs.Entries = entries
	b.mu.Unlock()
}

func (b *Bot) handleGetUp(cq *tgbotapi.CallbackQuery, fs *FileBrowseState, userID int64) {
	parent := filepath.Dir(fs.CurrentPath)
	if parent == fs.CurrentPath {
		return // already at root
	}

	text, keyboard, entries := buildFileBrowser(parent, 0)
	b.editMessageWithKeyboard(fs.ChatID, fs.MessageID, text, keyboard)

	b.mu.Lock()
	fs.CurrentPath = parent
	fs.Page = 0
	fs.Entries = entries
	b.mu.Unlock()
}

func (b *Bot) handleGetCancel(cq *tgbotapi.CallbackQuery, fs *FileBrowseState, userID int64) {
	b.mu.Lock()
	delete(b.fileBrowseStates, userID)
	b.mu.Unlock()

	b.editMessageText(fs.ChatID, fs.MessageID, "Cancelled.")
}
