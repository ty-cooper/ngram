package llm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

var (
	ErrModelOff      = errors.New("model is off, skipping AI")
	ErrBinaryMissing = errors.New("LLM binary not found in PATH")
	ErrAuthExpired   = errors.New("claude auth expired, run: claude login")
)

// Runner wraps exec.Command calls to the claude or ollama binary.
type Runner struct {
	BinaryPath string // "claude" (default), "ollama", or path to mock script
	Model      string // "cloud", "local", "off", "mock"
	VaultPath  string // working directory for invocations
}

type runConfig struct {
	systemPrompt string
	jsonSchema   string
	maxTokens    int
}

// RunOption configures a Run call.
type RunOption func(*runConfig)

func WithSystemPrompt(prompt string) RunOption {
	return func(c *runConfig) { c.systemPrompt = prompt }
}

func WithMaxTokens(n int) RunOption {
	return func(c *runConfig) { c.maxTokens = n }
}

func WithJSONSchema(schema string) RunOption {
	return func(c *runConfig) { c.jsonSchema = schema }
}

// Run executes the LLM binary with a prompt and returns stdout.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...RunOption) ([]byte, error) {
	if r.Model == "off" {
		return nil, ErrModelOff
	}

	cfg := &runConfig{}
	for _, o := range opts {
		o(cfg)
	}

	binary := r.BinaryPath
	if binary == "" {
		binary = "claude"
	}

	var args []string
	switch r.Model {
	case "cloud", "mock", "":
		args = r.buildClaudeArgs(prompt, cfg)
	case "local":
		args = r.buildOllamaArgs(cfg)
	default:
		return nil, fmt.Errorf("unknown model: %q", r.Model)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = r.VaultPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// For local/mock, pass prompt via stdin.
	if r.Model == "local" || r.Model == "mock" {
		cmd.Stdin = bytes.NewReader([]byte(prompt))
	}

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, ErrBinaryMissing
		}
		return nil, fmt.Errorf("llm %s: %w: %s", binary, err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func (r *Runner) buildClaudeArgs(prompt string, cfg *runConfig) []string {
	args := []string{"-p", prompt, "--output-format", "json"}
	if cfg.systemPrompt != "" {
		args = append(args, "--system-prompt", cfg.systemPrompt)
	}
	if cfg.jsonSchema != "" {
		args = append(args, "--json-schema", cfg.jsonSchema)
	}
	if cfg.maxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", cfg.maxTokens))
	}
	return args
}

// CheckAuth verifies the claude binary exists and auth is valid.
// Returns nil if model is "off", "local", or "mock".
func (r *Runner) CheckAuth(ctx context.Context) error {
	if r.Model == "off" || r.Model == "local" || r.Model == "mock" {
		return nil
	}

	binary := r.BinaryPath
	if binary == "" {
		binary = "claude"
	}

	if _, err := exec.LookPath(binary); err != nil {
		return ErrBinaryMissing
	}

	cmd := exec.CommandContext(ctx, binary, "-p", "respond with ok", "--output-format", "text", "--max-turns", "1")
	cmd.Dir = r.VaultPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if bytes.Contains([]byte(msg), []byte("auth")) || bytes.Contains([]byte(msg), []byte("login")) || bytes.Contains([]byte(msg), []byte("token")) || bytes.Contains([]byte(msg), []byte("expired")) {
			return ErrAuthExpired
		}
		return fmt.Errorf("claude check failed: %w: %s", err, msg)
	}
	return nil
}

func (r *Runner) buildOllamaArgs(cfg *runConfig) []string {
	model := "llama3"
	args := []string{"run", model, "--format", "json"}
	return args
}
