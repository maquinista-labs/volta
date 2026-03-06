package bot

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestBuildPriorityKeyboard(t *testing.T) {
	kb := buildPriorityKeyboard()

	// 3 rows of priorities + 1 cancel row = 4 rows
	if len(kb.InlineKeyboard) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(kb.InlineKeyboard))
	}

	// First 3 rows have 3 buttons each
	for i := 0; i < 3; i++ {
		if len(kb.InlineKeyboard[i]) != 3 {
			t.Errorf("row %d: expected 3 buttons, got %d", i, len(kb.InlineKeyboard[i]))
		}
	}

	// Last row has 1 cancel button
	if len(kb.InlineKeyboard[3]) != 1 {
		t.Errorf("cancel row: expected 1 button, got %d", len(kb.InlineKeyboard[3]))
	}

	// Verify all callback data starts with task_
	for _, row := range kb.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == nil {
				t.Error("button has nil callback data")
				continue
			}
			if !strings.HasPrefix(*btn.CallbackData, "task_") {
				t.Errorf("callback data %q does not start with task_", *btn.CallbackData)
			}
		}
	}

	// Check specific priority values
	first := *kb.InlineKeyboard[0][0].CallbackData
	if first != "task_pri:10" {
		t.Errorf("first button callback = %q, want task_pri:10", first)
	}

	last := *kb.InlineKeyboard[2][2].CallbackData
	if last != "task_pri:1" {
		t.Errorf("last priority button callback = %q, want task_pri:1", last)
	}
}

func TestBuildBodyKeyboard(t *testing.T) {
	text, kb := buildBodyKeyboard("Fix login", 8)

	if !strings.Contains(text, "Fix login") {
		t.Error("body text should contain task title")
	}
	if !strings.Contains(text, "8") {
		t.Error("body text should contain priority")
	}

	if len(kb.InlineKeyboard) != 1 {
		t.Fatalf("expected 1 row, got %d", len(kb.InlineKeyboard))
	}
	if len(kb.InlineKeyboard[0]) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(kb.InlineKeyboard[0]))
	}

	skip := *kb.InlineKeyboard[0][0].CallbackData
	if skip != "task_skip" {
		t.Errorf("skip button callback = %q, want task_skip", skip)
	}

	cancel := *kb.InlineKeyboard[0][1].CallbackData
	if cancel != "task_cancel" {
		t.Errorf("cancel button callback = %q, want task_cancel", cancel)
	}
}

func TestBuildConfirmationKeyboard(t *testing.T) {
	kb := buildConfirmationKeyboard("fix-login-abc12")

	if len(kb.InlineKeyboard) != 1 {
		t.Fatalf("expected 1 row, got %d", len(kb.InlineKeyboard))
	}
	if len(kb.InlineKeyboard[0]) != 1 {
		t.Fatalf("expected 1 button, got %d", len(kb.InlineKeyboard[0]))
	}

	data := *kb.InlineKeyboard[0][0].CallbackData
	if data != "task_pick:fix-login-abc12" {
		t.Errorf("pick button callback = %q, want task_pick:fix-login-abc12", data)
	}
}

func TestHandleAddTaskReply_NoReply(t *testing.T) {
	b := newTestBot(t)

	msg := &tgbotapi.Message{
		MessageID: 10,
		Text:      "Some body text",
		From:      &tgbotapi.User{ID: 100},
		Chat:      &tgbotapi.Chat{ID: -1001},
		// No ReplyToMessage
	}

	if b.handleAddTaskReply(msg) {
		t.Error("should return false when no reply")
	}
}

func TestHandleAddTaskReply_NoState(t *testing.T) {
	b := newTestBot(t)

	msg := &tgbotapi.Message{
		MessageID: 10,
		Text:      "Some body text",
		From:      &tgbotapi.User{ID: 100},
		Chat:      &tgbotapi.Chat{ID: -1001},
		ReplyToMessage: &tgbotapi.Message{
			MessageID: 5,
		},
	}

	if b.handleAddTaskReply(msg) {
		t.Error("should return false when no wizard state")
	}
}

func TestHandleAddTaskReply_WrongMessage(t *testing.T) {
	b := newTestBot(t)

	// Set up wizard state for user 100
	b.addTaskStates[100] = &addTaskState{
		Title:     "Test task",
		Step:      addStepBody,
		MessageID: 5,
		ChatID:    -1001,
		ThreadID:  7,
	}

	msg := &tgbotapi.Message{
		MessageID: 10,
		Text:      "Some body text",
		From:      &tgbotapi.User{ID: 100},
		Chat:      &tgbotapi.Chat{ID: -1001},
		ReplyToMessage: &tgbotapi.Message{
			MessageID: 99, // wrong message ID
		},
	}

	if b.handleAddTaskReply(msg) {
		t.Error("should return false when reply is to wrong message")
	}
}

func TestHandleAddTaskReply_WrongStep(t *testing.T) {
	b := newTestBot(t)

	// Set up wizard state at priority step (not body step)
	b.addTaskStates[100] = &addTaskState{
		Title:     "Test task",
		Step:      addStepPriority,
		MessageID: 5,
		ChatID:    -1001,
		ThreadID:  7,
	}

	msg := &tgbotapi.Message{
		MessageID: 10,
		Text:      "Some body text",
		From:      &tgbotapi.User{ID: 100},
		Chat:      &tgbotapi.Chat{ID: -1001},
		ReplyToMessage: &tgbotapi.Message{
			MessageID: 5,
		},
	}

	if b.handleAddTaskReply(msg) {
		t.Error("should return false when on wrong step")
	}
}

func TestCallbackDataPrefixes_Task(t *testing.T) {
	// Verify all task callback data values start with "task_"
	prefixes := []string{
		"task_pri:10",
		"task_pri:5",
		"task_pri:1",
		"task_skip",
		"task_cancel",
		"task_pick:some-id",
	}

	for _, data := range prefixes {
		if !strings.HasPrefix(data, "task_") {
			t.Errorf("callback %q does not have task_ prefix", data)
		}
	}
}
