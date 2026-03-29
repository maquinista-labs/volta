package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/otaviocarvalho/volta/internal/agent"
	"github.com/otaviocarvalho/volta/internal/db"
	"github.com/otaviocarvalho/volta/internal/listener"
	"github.com/otaviocarvalho/volta/internal/orchestrator"
	"github.com/otaviocarvalho/volta/internal/runner"
	"github.com/otaviocarvalho/volta/internal/tmux"
)

// handleOrchestStartCommand starts the orchestrator as an embedded goroutine.
func (b *Bot) handleOrchestStartCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	threadID := getThreadID(msg)

	pool := b.getPool()
	if pool == nil {
		b.reply(chatID, threadID, "Database not available. Set DATABASE_URL to use orchestration.")
		return
	}

	b.orchMu.Lock()
	if b.orchCancel != nil {
		b.orchMu.Unlock()
		b.reply(chatID, threadID, "Orchestrator already running. Use /orchest_stop first.")
		return
	}
	b.orchMu.Unlock()

	// Parse arguments: /orchest_start [project] [--agents N] [--runner R] [--worktrees]
	args := strings.Fields(msg.CommandArguments())
	project := ""
	maxAgents := 3
	runnerName := "claude"
	useWorktrees := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agents":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					maxAgents = n
				}
			}
		case "--runner":
			if i+1 < len(args) {
				i++
				runnerName = args[i]
			}
		case "--worktrees":
			useWorktrees = true
		default:
			if project == "" {
				project = args[i]
			}
		}
	}

	if project == "" {
		project = b.config.DefaultProject
	}
	if project == "" {
		b.reply(chatID, threadID, "Usage: /orchest_start <project> [--agents N]")
		return
	}

	r, err := runner.Get(runnerName)
	if err != nil {
		b.reply(chatID, threadID, fmt.Sprintf("Unknown runner %q: %v", runnerName, err))
		return
	}

	claudeMD, err := findClaudeMDFromBot()
	if err != nil {
		b.reply(chatID, threadID, fmt.Sprintf("Cannot find CLAUDE.md: %v", err))
		return
	}

	if err := tmux.EnsureSession(b.config.TmuxSessionName); err != nil {
		b.reply(chatID, threadID, fmt.Sprintf("Error ensuring tmux session: %v", err))
		return
	}

	el := listener.New(b.config.DatabaseURL)
	ctx, cancel := context.WithCancel(context.Background())

	go el.Start(ctx)
	notifyCh := orchestrator.NotifyBridge(ctx, el.TaskEvents)

	orchCfg := orchestrator.Config{
		Pool:         pool,
		Runner:       r,
		TmuxSession:  b.config.TmuxSessionName,
		ProjectID:    project,
		MaxAgents:    maxAgents,
		PollInterval: 10 * time.Second,
		UseWorktrees: useWorktrees,
		ClaudeMDPath: claudeMD,
		DatabaseURL:  b.config.DatabaseURL,
		NotifyCh:     notifyCh,
		NotifyFunc: func(message string) {
			log.Printf("Orchestrator: %s", message)
			b.reply(chatID, threadID, message)
		},
		ChatID:            chatID,
		BotRef:            b,
		PlannerPromptPath: b.config.PlannerPromptPath,
	}

	b.orchMu.Lock()
	b.orchCancel = cancel
	b.orchConfig = &orchCfg
	b.orchMu.Unlock()

	go func() {
		if err := orchestrator.Run(ctx, orchCfg); err != nil {
			log.Printf("Orchestrator error: %v", err)
		}
	}()

	b.reply(chatID, threadID, fmt.Sprintf("Orchestrator started: project=%s maxAgents=%d runner=%s worktrees=%v",
		project, maxAgents, runnerName, useWorktrees))
}

// handleOrchestStopCommand stops the orchestrator.
func (b *Bot) handleOrchestStopCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	threadID := getThreadID(msg)

	b.orchMu.Lock()
	cancel := b.orchCancel
	b.orchCancel = nil
	b.orchConfig = nil
	b.orchMu.Unlock()

	if cancel == nil {
		b.reply(chatID, threadID, "Orchestrator is not running.")
		return
	}

	cancel()

	// Offer to kill all agents
	if pool := b.getPool(); pool != nil {
		agents, err := db.ListAgents(pool)
		if err == nil && len(agents) > 0 {
			kb := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(
						fmt.Sprintf("Kill all %d agents", len(agents)), "agent_killall_confirm"),
					tgbotapi.NewInlineKeyboardButtonData("Keep agents", "noop"),
				),
			)
			b.sendMessageWithKeyboard(chatID, threadID,
				fmt.Sprintf("Orchestrator stopped. Kill %d running agents?", len(agents)), kb)
			return
		}
	}

	b.reply(chatID, threadID, "Orchestrator stopped.")
}

// handleOrchestStatusCommand shows orchestrator status.
func (b *Bot) handleOrchestStatusCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	threadID := getThreadID(msg)

	b.orchMu.Lock()
	orchCfg := b.orchConfig
	b.orchMu.Unlock()

	pool := b.getPool()
	if pool == nil {
		running := "stopped"
		if orchCfg != nil {
			running = fmt.Sprintf("running (project=%s, maxAgents=%d)", orchCfg.ProjectID, orchCfg.GetMaxAgents())
		}
		b.reply(chatID, threadID, fmt.Sprintf("Orchestrator: %s\n(Database not available — no task details)", running))
		return
	}

	var projectID *string
	if orchCfg != nil {
		projectID = &orchCfg.ProjectID
	}

	status, err := orchestrator.Status(pool, projectID)
	if err != nil {
		b.reply(chatID, threadID, fmt.Sprintf("Error: %v", err))
		return
	}

	running := "stopped"
	if orchCfg != nil {
		running = fmt.Sprintf("running (project=%s, maxAgents=%d)", orchCfg.ProjectID, orchCfg.GetMaxAgents())
	}

	text := fmt.Sprintf("Orchestrator: %s\n%s", running, status.String())
	b.reply(chatID, threadID, text)
}

// handleOrchestScaleCommand updates the max agents for the running orchestrator.
func (b *Bot) handleOrchestScaleCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	threadID := getThreadID(msg)

	b.orchMu.Lock()
	orchCfg := b.orchConfig
	b.orchMu.Unlock()

	if orchCfg == nil {
		b.reply(chatID, threadID, "Orchestrator is not running. Start it with /orchest_start.")
		return
	}

	nStr := strings.TrimSpace(msg.CommandArguments())
	if nStr == "" {
		b.reply(chatID, threadID, fmt.Sprintf("Current max agents: %d\nUsage: /orchest_scale <N>", orchCfg.GetMaxAgents()))
		return
	}

	n, err := strconv.Atoi(nStr)
	if err != nil || n < 1 {
		b.reply(chatID, threadID, "Invalid number. Usage: /orchest_scale <N>")
		return
	}

	old := orchCfg.GetMaxAgents()
	orchCfg.SetMaxAgents(n)
	b.reply(chatID, threadID, fmt.Sprintf("Max agents updated: %d -> %d", old, n))
}

// findClaudeMDFromBot locates claude/agent-loop.md relative to CWD.
func findClaudeMDFromBot() (string, error) {
	path := "claude/agent-loop.md"
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("claude/agent-loop.md not found")
	}
	return filepath.Abs(path)
}

// handleOrchestStopCallback handles orchestrator stop confirmation callbacks.
func (b *Bot) handleOrchestStopCallback(cq *tgbotapi.CallbackQuery, data string) {
	chatID := cq.Message.Chat.ID
	threadID := getThreadID(cq.Message)

	if data == "orchest_stop_killall" {
		pool := b.getPool()
		if pool == nil {
			return
		}
		if err := agent.KillAll(pool, b.config.TmuxSessionName); err != nil {
			b.reply(chatID, threadID, fmt.Sprintf("Error: %v", err))
			return
		}
		b.editMessageText(chatID, cq.Message.MessageID, "Orchestrator stopped. All agents killed.")
	}
}
