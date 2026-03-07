package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/otaviocarvalho/volta/internal/agent"
	"github.com/otaviocarvalho/volta/internal/db"
	"github.com/otaviocarvalho/volta/internal/git"
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
		// Check if task already exists for this spec
		task, err := db.GetTask(cfg.Pool, s.ID)
		if err != nil {
			log.Printf("Spec scan: error checking task for spec %s: %v", s.ID, err)
			continue
		}
		if task != nil {
			continue // task already exists
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
		"DATABASE_URL": cfg.DatabaseURL,
	}

	// Spawn planner agent (role=planner, not counted against maxAgents)
	a, err := agent.Spawn(cfg.Pool, cfg.TmuxSession, agentID, cfg.ClaudeMDPath, env, cfg.Runner, "planner")
	if err != nil {
		return fmt.Errorf("spawning planner agent: %w", err)
	}

	// Track the dispatched planner
	plannerMu.Lock()
	dispatchedSpecs[s.ID] = a.ID
	plannerMu.Unlock()

	// Send spec content as prompt to the planner
	prompt := fmt.Sprintf("Read the following spec and decompose it into tasks. "+
		"Present a plan for approval.\n\n---\n\nSpec: %s\nID: %s\n\n%s",
		s.Title, s.ID, s.Body)

	cfg.BotRef.ReplyToThread(cfg.ChatID, threadID, fmt.Sprintf("Planner agent spawned: %s\nSpec: %s", a.ID, s.Title))

	// Send prompt to the tmux window via the planner's window
	sendPlannerPrompt(cfg.TmuxSession, agentID, prompt)

	log.Printf("Spec scan: spawned planner %s for spec %s in topic %d", a.ID, s.ID, threadID)
	notify(cfg, fmt.Sprintf("Planner spawned: %s for spec %s", a.ID, s.Title))

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
