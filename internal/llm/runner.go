package llm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var (
	ErrModelOff      = errors.New("model is off, skipping AI")
	ErrBinaryMissing = errors.New("LLM binary not found in PATH")
	ErrAuthExpired   = errors.New("anthropic API key missing or invalid")
)

// Runner wraps Anthropic API calls via the official Go SDK.
type Runner struct {
	Model        string // "cloud", "local", "off", "mock"
	VaultPath    string
	MockResponse []byte // For testing: if set, Run returns this instead of calling API.
	client       anthropic.Client
	hasClient    bool
}

// NewRunner creates a Runner. For cloud mode, ANTHROPIC_API_KEY must be set.
func NewRunner(model, vaultPath string) *Runner {
	r := &Runner{
		Model:     model,
		VaultPath: vaultPath,
	}
	if model == "cloud" || model == "" {
		r.client = anthropic.NewClient()
		r.hasClient = true
	}
	return r
}

// NewMockRunner creates a Runner that returns a fixed response for testing.
func NewMockRunner(response []byte) *Runner {
	return &Runner{
		Model:        "mock",
		MockResponse: response,
	}
}

type runConfig struct {
	systemPrompt string
	maxTokens    int
	images       []string
}

// RunOption configures a Run call.
type RunOption func(*runConfig)

func WithSystemPrompt(prompt string) RunOption {
	return func(c *runConfig) { c.systemPrompt = prompt }
}

func WithMaxTokens(n int) RunOption {
	return func(c *runConfig) { c.maxTokens = n }
}

func WithImages(paths []string) RunOption {
	return func(c *runConfig) { c.images = paths }
}

// Run sends a prompt to the Anthropic API and returns the text response.
func (r *Runner) Run(ctx context.Context, prompt string, opts ...RunOption) ([]byte, error) {
	if r.Model == "off" {
		return nil, ErrModelOff
	}

	if r.Model == "mock" && r.MockResponse != nil {
		return r.MockResponse, nil
	}

	cfg := &runConfig{maxTokens: 4096}
	for _, o := range opts {
		o(cfg)
	}

	if !r.hasClient {
		return nil, ErrAuthExpired
	}

	// Build content blocks.
	var contentBlocks []anthropic.ContentBlockParamUnion

	// Add images first if present.
	for _, imgPath := range cfg.images {
		data, err := os.ReadFile(imgPath)
		if err != nil {
			continue
		}
		ext := strings.ToLower(filepath.Ext(imgPath))
		mediaType := "image/png"
		switch ext {
		case ".jpg", ".jpeg":
			mediaType = "image/jpeg"
		case ".gif":
			mediaType = "image/gif"
		case ".webp":
			mediaType = "image/webp"
		}
		contentBlocks = append(contentBlocks, anthropic.NewImageBlockBase64(mediaType, base64.StdEncoding.EncodeToString(data)))
	}

	// Add text prompt.
	contentBlocks = append(contentBlocks, anthropic.NewTextBlock(prompt))

	// Build params.
	params := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: int64(cfg.maxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(contentBlocks...),
		},
	}

	if cfg.systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: cfg.systemPrompt, Type: "text"},
		}
	}

	msg, err := r.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic api: %w", err)
	}

	// Extract text from response.
	var result strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			result.WriteString(block.Text)
		}
	}

	text := strings.TrimSpace(result.String())
	if text == "" {
		return nil, fmt.Errorf("anthropic api: empty response (stop_reason: %s)", msg.StopReason)
	}

	return []byte(text), nil
}

// CheckAuth verifies the API key is set and valid.
func (r *Runner) CheckAuth(ctx context.Context) error {
	if r.Model == "off" || r.Model == "local" || r.Model == "mock" {
		return nil
	}

	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return ErrAuthExpired
	}

	// Quick API call to verify key works.
	_, err := r.Run(ctx, "respond with ok", WithMaxTokens(10))
	if err != nil {
		return fmt.Errorf("auth check: %w", err)
	}
	return nil
}
