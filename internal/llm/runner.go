package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/567-labs/instructor-go/pkg/instructor"
	anthropic "github.com/liushuangls/go-anthropic/v2"
	langsmith "github.com/ty-cooper/langsmith-go"
)

var (
	ErrModelOff    = errors.New("model is off, skipping AI")
	ErrAuthExpired = errors.New("anthropic API key missing or invalid")
)

const (
	defaultModel  = "claude-sonnet-4-6"
	callTimeout   = 90 * time.Second // Max time per LLM call
)

// Usage tracks token consumption from the last API call.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Runner wraps Anthropic API calls via instructor-go for type-safe structured output
// and raw client for free-text responses.
type Runner struct {
	Model        string // "cloud", "off", "mock"
	VaultPath    string
	MockResponse []byte // For testing: if set, returns this instead of calling API.
	LastUsage    Usage  // Token counts from last API call.
	Tracer       *langsmith.Client // Optional LangSmith tracing — nil = no-op.
	instructor   *instructor.InstructorAnthropic
	rawClient    *anthropic.Client
	hasClient    bool
}

// NewRunner creates a Runner. For cloud mode, ANTHROPIC_API_KEY must be set.
// If LANGCHAIN_API_KEY is set, LangSmith tracing is enabled automatically.
func NewRunner(model, vaultPath string) *Runner {
	r := &Runner{
		Model:     model,
		VaultPath: vaultPath,
	}
	if model == "cloud" || model == "" {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return r
		}
		raw := anthropic.NewClient(apiKey)
		r.rawClient = raw
		r.instructor = instructor.FromAnthropic(
			raw,
			instructor.WithMode(instructor.ModeJSONSchema),
			instructor.WithMaxRetries(3),
		)
		r.hasClient = true
	}

	// Initialize LangSmith tracing if API key is available.
	if lsClient, err := langsmith.NewClient(
		langsmith.WithProject("ngram"),
	); err == nil {
		r.Tracer = lsClient
	}

	return r
}

// Close shuts down the runner, flushing any pending LangSmith traces.
func (r *Runner) Close() {
	if r.Tracer != nil {
		r.Tracer.Close()
	}
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

// RunOption configures a Run or Instruct call.
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

// Instruct sends a prompt and unmarshals the response into a typed struct.
// Uses instructor-go for automatic schema generation, validation, and retries.
func (r *Runner) Instruct(ctx context.Context, prompt string, result any, opts ...RunOption) error {
	if r.Model == "off" {
		return ErrModelOff
	}
	if r.Model == "mock" && r.MockResponse != nil {
		return json.Unmarshal(r.MockResponse, result)
	}
	if !r.hasClient {
		return ErrAuthExpired
	}

	cfg := &runConfig{maxTokens: 4096}
	for _, o := range opts {
		o(cfg)
	}

	// Enforce call timeout.
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	content := buildMessageContent(prompt, cfg.images)

	req := anthropic.MessagesRequest{
		Model:     anthropic.Model(defaultModel),
		MaxTokens: cfg.maxTokens,
		Messages: []anthropic.Message{
			{Role: anthropic.RoleUser, Content: content},
		},
	}

	if cfg.systemPrompt != "" {
		req.System = cfg.systemPrompt
	}

	// Trace the call if LangSmith is enabled.
	var rt *langsmith.RunTree
	if r.Tracer != nil {
		parent := langsmith.RunTreeFromContext(ctx)
		if parent != nil {
			rt = parent.CreateChild("instruct", langsmith.RunTypeLLM,
				langsmith.WithRunTreeClient(r.Tracer),
			)
		} else {
			rt = langsmith.NewRunTree("instruct", langsmith.RunTypeLLM,
				langsmith.WithRunTreeClient(r.Tracer),
			)
		}
		rt.SetInputs(map[string]any{
			"model":  defaultModel,
			"prompt": truncateForTrace(prompt),
		})
		rt.PostRun()
	}

	_, err := r.instructor.CreateMessages(ctx, req, result)
	if err != nil {
		if rt != nil {
			rt.End(langsmith.WithEndError(err.Error()))
		}
		return fmt.Errorf("instructor: %w", err)
	}

	if rt != nil {
		rt.End(langsmith.WithEndOutputs(map[string]any{"result": "ok"}))
	}
	return nil
}

// Run sends a prompt and returns the raw text response.
// Use for free-text responses (report sections, RAG synthesis).
func (r *Runner) Run(ctx context.Context, prompt string, opts ...RunOption) ([]byte, error) {
	if r.Model == "off" {
		return nil, ErrModelOff
	}
	if r.Model == "mock" && r.MockResponse != nil {
		return r.MockResponse, nil
	}
	if !r.hasClient {
		return nil, ErrAuthExpired
	}

	cfg := &runConfig{maxTokens: 4096}
	for _, o := range opts {
		o(cfg)
	}

	// Enforce call timeout.
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	content := buildMessageContent(prompt, cfg.images)

	req := anthropic.MessagesRequest{
		Model:     anthropic.Model(defaultModel),
		MaxTokens: cfg.maxTokens,
		Messages: []anthropic.Message{
			{Role: anthropic.RoleUser, Content: content},
		},
	}

	if cfg.systemPrompt != "" {
		req.System = cfg.systemPrompt
	}

	// Trace the call if LangSmith is enabled.
	var rt *langsmith.RunTree
	if r.Tracer != nil {
		parent := langsmith.RunTreeFromContext(ctx)
		if parent != nil {
			rt = parent.CreateChild("run", langsmith.RunTypeLLM,
				langsmith.WithRunTreeClient(r.Tracer),
			)
		} else {
			rt = langsmith.NewRunTree("run", langsmith.RunTypeLLM,
				langsmith.WithRunTreeClient(r.Tracer),
			)
		}
		rt.SetInputs(map[string]any{
			"model":  defaultModel,
			"prompt": truncateForTrace(prompt),
		})
		rt.PostRun()
	}

	resp, err := r.rawClient.CreateMessages(ctx, req)
	if err != nil {
		if rt != nil {
			rt.End(langsmith.WithEndError(err.Error()))
		}
		return nil, fmt.Errorf("anthropic api: %w", err)
	}

	// Track token usage.
	r.LastUsage = Usage{
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}

	var result strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			result.WriteString(*block.Text)
		}
	}

	text := strings.TrimSpace(result.String())
	if text == "" {
		if rt != nil {
			rt.End(langsmith.WithEndError("empty response"))
		}
		return nil, fmt.Errorf("anthropic api: empty response (stop_reason: %s)", resp.StopReason)
	}

	if rt != nil {
		rt.End(langsmith.WithEndOutputs(map[string]any{
			"output":        truncateForTrace(text),
			"input_tokens":  resp.Usage.InputTokens,
			"output_tokens": resp.Usage.OutputTokens,
		}))
	}
	return []byte(text), nil
}

// buildMessageContent creates Anthropic message content blocks with optional images.
func buildMessageContent(prompt string, images []string) []anthropic.MessageContent {
	var content []anthropic.MessageContent

	for _, imgPath := range images {
		data, mediaType, err := readImage(imgPath)
		if err != nil {
			continue
		}
		content = append(content, anthropic.NewImageMessageContent(
			anthropic.MessageContentSource{
				Type:      "base64",
				MediaType: mediaType,
				Data:      base64.StdEncoding.EncodeToString(data),
			},
		))
	}

	content = append(content, anthropic.NewTextMessageContent(prompt))
	return content
}

const maxImageBytes = 5 * 1024 * 1024 // 5 MB — resize if larger

// readImage reads an image file, converting HEIC/HEIF to JPEG and resizing large images.
func readImage(imgPath string) ([]byte, string, error) {
	ext := strings.ToLower(filepath.Ext(imgPath))

	if ext == ".heic" || ext == ".heif" {
		if runtime.GOOS != "darwin" {
			return nil, "", fmt.Errorf("HEIC/HEIF conversion requires macOS (sips). Convert %s to JPEG manually", filepath.Base(imgPath))
		}
		tmpFile := imgPath + ".converted.jpg"
		defer os.Remove(tmpFile)

		cmd := execCommand("sips", "-s", "format", "jpeg", imgPath, "--out", tmpFile)
		if err := cmd.Run(); err != nil {
			return nil, "", fmt.Errorf("convert HEIC via sips: %w", err)
		}
		data, err := os.ReadFile(tmpFile)
		if err != nil {
			return nil, "", err
		}
		data, err = resizeIfNeeded(tmpFile, data)
		if err != nil {
			return nil, "", err
		}
		return data, "image/jpeg", nil
	}

	data, err := os.ReadFile(imgPath)
	if err != nil {
		return nil, "", err
	}

	mediaType := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mediaType = "image/jpeg"
	case ".gif":
		mediaType = "image/gif"
	case ".webp":
		mediaType = "image/webp"
	}

	data, err = resizeIfNeeded(imgPath, data)
	if err != nil {
		return nil, "", err
	}
	return data, mediaType, nil
}

// resizeIfNeeded uses macOS sips to downsample images exceeding maxImageBytes.
func resizeIfNeeded(imgPath string, data []byte) ([]byte, error) {
	if len(data) <= maxImageBytes {
		return data, nil
	}
	if runtime.GOOS != "darwin" {
		// Can't resize without sips — send original.
		return data, nil
	}

	tmpOut := imgPath + ".resized.jpg"
	defer os.Remove(tmpOut)

	// Resize to max 2048px on longest side and convert to JPEG for size.
	cmd := execCommand("sips", "-Z", "2048", "-s", "format", "jpeg", imgPath, "--out", tmpOut)
	if err := cmd.Run(); err != nil {
		// If sips fails, return original — don't lose the image.
		return data, nil
	}
	resized, err := os.ReadFile(tmpOut)
	if err != nil {
		return data, nil
	}
	return resized, nil
}

var execCommand = newExecCommand

func newExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// truncateForTrace limits prompt/output size in trace payloads.
func truncateForTrace(s string) string {
	if len(s) > 2000 {
		return s[:2000] + "...(truncated)"
	}
	return s
}

// CheckAuth verifies the API key is set and valid.
func (r *Runner) CheckAuth(ctx context.Context) error {
	if r.Model == "off" || r.Model == "local" || r.Model == "mock" {
		return nil
	}
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return ErrAuthExpired
	}
	_, err := r.Run(ctx, "respond with ok", WithMaxTokens(10))
	if err != nil {
		return fmt.Errorf("auth check: %w", err)
	}
	return nil
}
