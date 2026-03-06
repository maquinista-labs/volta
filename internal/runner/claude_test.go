package runner

import (
	"strings"
	"testing"
)

func TestClaudeRunner_Name(t *testing.T) {
	c := &ClaudeRunner{}
	if c.Name() != "claude" {
		t.Errorf("Name() = %q", c.Name())
	}
}

func TestClaudeRunner_InteractiveCommand(t *testing.T) {
	c := &ClaudeRunner{}
	cmd := c.InteractiveCommand("do something", Config{})
	if !strings.Contains(cmd, "--dangerously-skip-permissions") {
		t.Error("missing --dangerously-skip-permissions")
	}
	if !strings.Contains(cmd, "do something") {
		t.Error("missing prompt")
	}
}

func TestClaudeRunner_NonInteractiveArgs(t *testing.T) {
	c := &ClaudeRunner{Model: "opus", MaxBudget: 5.0}
	args := c.NonInteractiveArgs("test prompt", Config{})

	hasFlag := func(flag string) bool {
		for _, a := range args {
			if a == flag {
				return true
			}
		}
		return false
	}

	if !hasFlag("--output-format") {
		t.Error("missing --output-format")
	}
	if !hasFlag("--dangerously-skip-permissions") {
		t.Error("missing --dangerously-skip-permissions")
	}
	if !hasFlag("--no-session-persistence") {
		t.Error("missing --no-session-persistence")
	}
	if !hasFlag("--model") {
		t.Error("missing --model")
	}
	if !hasFlag("--max-budget-usd") {
		t.Error("missing --max-budget-usd")
	}

	// Check model value follows --model flag
	for i, a := range args {
		if a == "--model" && i+1 < len(args) {
			if args[i+1] != "opus" {
				t.Errorf("model = %q, want opus", args[i+1])
			}
		}
	}
}

func TestClaudeRunner_NonInteractiveArgs_DefaultModel(t *testing.T) {
	c := &ClaudeRunner{}
	args := c.NonInteractiveArgs("test", Config{})
	for i, a := range args {
		if a == "--model" && i+1 < len(args) {
			if args[i+1] != "sonnet" {
				t.Errorf("default model = %q, want sonnet", args[i+1])
			}
		}
	}
}

func TestClaudeRunner_EnvOverrides(t *testing.T) {
	c := &ClaudeRunner{}
	env := c.EnvOverrides()
	if _, ok := env["CLAUDECODE"]; !ok {
		t.Error("missing CLAUDECODE env override")
	}
}

func TestClaudeRunner_Registered(t *testing.T) {
	r, err := Get("claude")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name() != "claude" {
		t.Errorf("registered runner name = %q", r.Name())
	}
}
