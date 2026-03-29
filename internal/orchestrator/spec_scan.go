package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/otaviocarvalho/volta/internal/db"
	"github.com/otaviocarvalho/volta/internal/git"
	"github.com/otaviocarvalho/volta/internal/runner"
	"github.com/otaviocarvalho/volta/internal/spec"
	"github.com/otaviocarvalho/volta/internal/tmux"
)

// plannerTracker tracks which specs already have a planner agent dispatched.
var (
	plannerMu       sync.Mutex
	dispatchedSpecs = make(map[string]string) // specID → plannerAgentID
)

// specScan detects new spec files in .specs/ that have no corresponding task
// and spawns planner agents to decompose them.
func specScan(ctx context.Context, cfg Config) error {
	specsDir := cfg.SpecsDir
	if specsDir == "" {
		repoRoot, err := git.RepoRoot(".")
		if err != nil {
			return nil // not in a git repo, skip
		}
		specsDir = filepath.Join(repoRoot, ".specs")
	}

	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return nil // no .specs/ directory
	}

	specs, err := spec.ParseDir(specsDir)
	if err != nil {
		return fmt.Errorf("parsing specs dir: %w", err)
	}

	if len(specs) == 0 {
		return nil
	}

	for _, s := range specs {
		// Check if task already exists for this spec.
		// GetTask returns an error when no task is found (via ResolvePartialID),
		// which is the expected case for new specs that need planning.
		task, err := db.GetTask(cfg.Pool, s.ID)
		if err == nil && task != nil {
			continue // task already exists, skip
		}

		// Check if we already dispatched a planner for this spec
		plannerMu.Lock()
		if _, dispatched := dispatchedSpecs[s.ID]; dispatched {
			plannerMu.Unlock()
			continue
		}
		plannerMu.Unlock()

		// Spawn a planner agent
		if err := spawnPlannerForSpec(ctx, cfg, s); err != nil {
			log.Printf("Spec scan: error spawning planner for %s: %v", s.ID, err)
		}
	}

	return nil
}

// spawnPlannerForSpec creates a Telegram topic and planner agent for a spec file.
func spawnPlannerForSpec(ctx context.Context, cfg Config, s *spec.SpecFile) error {
	agentID := fmt.Sprintf("planner-%s", s.ID)

	// Create Telegram topic for the planner
	topicName := fmt.Sprintf("Plan: %s", s.Title)
	threadID, err := cfg.BotRef.CreatePlannerTopic(cfg.ChatID, topicName)
	if err != nil {
		return fmt.Errorf("creating planner topic: %w", err)
	}

	env := map[string]string{
		"DATABASE_URL":    cfg.DatabaseURL,
		"MINUANO_PROJECT": cfg.ProjectID,
	}

	runnerName := "claude"
	if cfg.Runner != nil {
		runnerName = cfg.Runner.Name()
	}

	// Register planner agent in DB (role=planner, not counted against maxAgents)
	if err := db.RegisterAgent(cfg.Pool, agentID, cfg.TmuxSession, agentID, nil, nil, runnerName, nil, "planner"); err != nil {
		return fmt.Errorf("registering planner agent: %w", err)
	}

	// Build merged env for tmux window
	mergedEnv := make(map[string]string, len(env))
	for k, v := range env {
		mergedEnv[k] = v
	}
	if cfg.Runner != nil {
		for k, v := range cfg.Runner.EnvOverrides() {
			mergedEnv[k] = v
		}
	}

	if err := tmux.NewWindowWithDir(cfg.TmuxSession, agentID, ".", mergedEnv); err != nil {
		db.DeleteAgent(cfg.Pool, agentID)
		return fmt.Errorf("creating planner tmux window: %w", err)
	}

	// Send bootstrap env exports
	scriptsDir := filepath.Join(filepath.Dir(cfg.ClaudeMDPath), "..", "scripts")
	absScripts, _ := filepath.Abs(scriptsDir)

	bootstrap := []string{
		fmt.Sprintf("export AGENT_ID=%q", agentID),
		fmt.Sprintf("export DATABASE_URL=%q", cfg.DatabaseURL),
		fmt.Sprintf("export MINUANO_PROJECT=%q", cfg.ProjectID),
		fmt.Sprintf("export PATH=\"$PATH:%s\"", absScripts),
	}

	// Use the runner's PlannerCommand with the planner system prompt
	plannerPrompt := cfg.PlannerPromptPath
	if plannerPrompt == "" {
		// Fall back to finding it relative to ClaudeMDPath
		plannerPrompt = filepath.Join(filepath.Dir(cfg.ClaudeMDPath), "planner-system-prompt.md")
	}

	if cfg.Runner != nil {
		bootstrap = append(bootstrap, cfg.Runner.PlannerCommand(plannerPrompt, runner.Config{Env: env}))
	} else {
		bootstrap = append(bootstrap, fmt.Sprintf("claude --dangerously-skip-permissions --system-prompt \"$(cat %s)\"", plannerPrompt))
	}

	for _, cmd := range bootstrap {
		tmux.SendKeysWithDelay(cfg.TmuxSession, agentID, cmd, 100)
	}

	// Track the dispatched planner
	plannerMu.Lock()
	dispatchedSpecs[s.ID] = agentID
	plannerMu.Unlock()

	// Write spec content to temp file and send as initial prompt
	prompt := fmt.Sprintf("Read the following spec and decompose it into tasks.\n\n---\n\nSpec: %s\nID: %s\nProject: %s\n\n%s",
		s.Title, s.ID, cfg.ProjectID, s.Body)

	cfg.BotRef.ReplyToThread(cfg.ChatID, threadID, fmt.Sprintf("Planner agent spawned: %s\nSpec: %s", agentID, s.Title))

	// Send prompt to the planner after it starts up
	sendPlannerPrompt(cfg.TmuxSession, agentID, prompt)

	log.Printf("Spec scan: spawned planner %s for spec %s in topic %d", agentID, s.ID, threadID)
	notify(cfg, fmt.Sprintf("Planner spawned: %s for spec %s", agentID, s.Title))

	return nil
}

// sendPlannerPrompt writes the spec prompt to a temp file and sends it to the planner's tmux window.
func sendPlannerPrompt(tmuxSession, windowID, prompt string) {
	tmpFile, err := os.CreateTemp("", "volta-planner-*.md")
	if err != nil {
		log.Printf("Error creating planner prompt file: %v", err)
		return
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(prompt); err != nil {
		log.Printf("Error writing planner prompt: %v", err)
		return
	}
	tmpFile.Close()

	ref := fmt.Sprintf("Please read and follow the instructions in %s", tmpFile.Name())
	tmux.SendKeysWithDelay(tmuxSession, windowID, ref, 500)
}

// ClearPlannerTracker removes a spec from the dispatched planners map.
// Called when a planner agent completes or is killed.
func ClearPlannerTracker(specID string) {
	plannerMu.Lock()
	delete(dispatchedSpecs, specID)
	plannerMu.Unlock()
}
