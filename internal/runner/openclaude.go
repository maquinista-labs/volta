package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// OpenClaudeRunner implements AgentRunner for OpenClaude.
type OpenClaudeRunner struct {
	Model     string
	MaxBudget float64
}

func init() {
	Register("openclaude", &OpenClaudeRunner{})
}

func (o *OpenClaudeRunner) Name() string { return "openclaude" }

func (o *OpenClaudeRunner) LaunchCommand(cfg Config) string {
	return "IS_SANDBOX=1 openclaude --dangerously-skip-permissions"
}

func (o *OpenClaudeRunner) InteractiveCommand(prompt string, cfg Config) string {
	escaped := strings.ReplaceAll(prompt, "\"", "\\\"")
	return fmt.Sprintf("openclaude --dangerously-skip-permissions -p \"%s\"", escaped)
}

func (o *OpenClaudeRunner) PlannerCommand(systemPromptPath string, cfg Config) string {
	return fmt.Sprintf("openclaude --dangerously-skip-permissions --system-prompt \"$(cat %s)\"", systemPromptPath)
}

func (o *OpenClaudeRunner) NonInteractiveArgs(prompt string, cfg Config) []string {
	args := []string{
		"openclaude",
		"-p", prompt,
		"--output-format", "json",
		"--dangerously-skip-permissions",
		"--no-session-persistence",
	}
	model := o.Model
	if model == "" {
		model = "sonnet"
	}
	args = append(args, "--model", model)

	if o.MaxBudget > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", o.MaxBudget))
	}

	return args
}

func (o *OpenClaudeRunner) RunNonInteractive(ctx context.Context, prompt string, cfg Config) (*Result, error) {
	fullArgs := o.NonInteractiveArgs(prompt, cfg)
	// fullArgs[0] is "openclaude", rest are arguments
	cmd := exec.CommandContext(ctx, fullArgs[0], fullArgs[1:]...)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	for k, v := range o.EnvOverrides() {
		cmd.Env = append(cmd.Environ(), k+"="+v)
	}
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Environ(), k+"="+v)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("running openclaude: %w", err)
		}
	}

	return &Result{
		ExitCode: exitCode,
		Output:   stdout.String(),
	}, nil
}

func (o *OpenClaudeRunner) DetectInstallation() bool {
	_, err := exec.LookPath("openclaude")
	return err == nil
}

func (o *OpenClaudeRunner) EnvOverrides() map[string]string {
	return map[string]string{
		"CLAUDECODE": "",
	}
}

func (o *OpenClaudeRunner) HasSessionHook() bool {
	return true
}
