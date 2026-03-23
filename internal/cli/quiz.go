package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/pipeline"
	"github.com/ty-cooper/ngram/internal/quiz"
	"github.com/ty-cooper/ngram/internal/retention"
	"github.com/ty-cooper/ngram/internal/search"
	"github.com/ty-cooper/ngram/internal/vault"
)

var (
	quizDomain string
	quizWeak   bool
	quizNew    bool
)

var quizCmd = &cobra.Command{
	Use:   "quiz",
	Short: "Start an interactive quiz session",
	RunE:  quizRun,
}

func init() {
	quizCmd.Flags().StringVar(&quizDomain, "domain", "", "filter by domain")
	quizCmd.Flags().BoolVar(&quizWeak, "weak", false, "only notes with score < 60")
	quizCmd.Flags().BoolVar(&quizNew, "new", false, "only unreviewed notes")
}

func quizRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	// Walk vault for knowledge notes with retention.
	files, err := search.WalkVault(c.VaultPath)
	if err != nil {
		return fmt.Errorf("walk vault: %w", err)
	}

	var notes []retention.Note
	for _, f := range files {
		doc, err := search.ParseNoteFile(f, c.VaultPath)
		if err != nil || doc.ContentType != "knowledge" {
			continue
		}

		// Parse retention from frontmatter.
		ret := parseRetentionFromFile(f)
		if ret == nil {
			continue
		}

		n := retention.Note{
			Path:         doc.FilePath,
			ContentType:  doc.ContentType,
			Domain:       doc.Domain,
			TopicCluster: doc.TopicCluster,
			Retention:    ret,
		}

		// Apply filters.
		if quizDomain != "" && n.Domain != quizDomain {
			continue
		}
		if quizWeak && ret.RetentionScore >= 60 {
			continue
		}
		if quizNew && ret.State != "new" {
			continue
		}

		notes = append(notes, n)
	}

	queue := retention.BuildDailyQueue(notes)
	if len(queue) == 0 {
		fmt.Println("No notes due for quiz today.")
		return nil
	}

	// Build quiz items.
	runner := &llm.Runner{
		BinaryPath: "claude",
		Model:      c.Model,
		VaultPath:  c.VaultPath,
	}

	items := make([]quiz.QuizItem, 0, len(queue))
	for _, n := range queue {
		notePath := filepath.Join(c.VaultPath, n.Path)
		body, _ := os.ReadFile(notePath)

		// Generate question.
		question := generateQuestion(runner, n, string(body))

		items = append(items, quiz.QuizItem{
			NotePath:     n.Path,
			NoteTitle:    filepath.Base(n.Path),
			Domain:       n.Domain,
			TopicCluster: n.TopicCluster,
			Question:     question,
			NoteBody:     string(body),
		})
	}

	start := time.Now()

	// Build TUI model.
	model := quiz.NewModel(items)
	model.OnGrade = func(item quiz.QuizItem, answer string) (*quiz.GradeResult, error) {
		return gradeAnswer(runner, item, answer)
	}
	model.OnSkip = func(item quiz.QuizItem) {
		// Grade 0 for skip.
	}

	// Run TUI.
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("quiz tui: %w", err)
	}

	// Log session.
	duration := time.Since(start)
	session := vault.QuizSession{
		SessionID:       start.UTC().Format(time.RFC3339),
		NotesQuizzed:    len(items),
		DomainScores:    make(map[string]int),
		DurationSeconds: int(duration.Seconds()),
	}
	vault.LogQuizSession(c.VaultPath, session)

	return nil
}

func generateQuestion(runner *llm.Runner, n retention.Note, body string) string {
	prompt := quiz.BuildQuestionPrompt(filepath.Base(n.Path), n.Domain, n.TopicCluster, body)
	out, err := runner.Run(context.Background(), prompt)
	if err != nil {
		return quiz.FallbackQuestion(filepath.Base(n.Path))
	}

	var result struct {
		Question string `json:"question"`
	}
	clean := pipeline.StripCodeFencesExported(out)
	if json.Unmarshal(clean, &result) != nil || result.Question == "" {
		return quiz.FallbackQuestion(filepath.Base(n.Path))
	}
	return result.Question
}

func gradeAnswer(runner *llm.Runner, item quiz.QuizItem, answer string) (*quiz.GradeResult, error) {
	prompt := quiz.BuildGradingPrompt(item.Question, answer, item.NoteBody, item.Domain)
	out, err := runner.Run(context.Background(), prompt)
	if err != nil {
		return nil, err
	}

	var result struct {
		Score         int      `json:"score"`
		CorrectPoints []string `json:"correct_points"`
		MissingPoints []string `json:"missing_points"`
		Feedback      string   `json:"feedback"`
	}
	clean := pipeline.StripCodeFencesExported(out)
	if err := json.Unmarshal(clean, &result); err != nil {
		return nil, err
	}

	sm2Grade := retention.LLMScoreToGrade(result.Score, answer)

	return &quiz.GradeResult{
		Score:         result.Score,
		CorrectPoints: result.CorrectPoints,
		MissingPoints: result.MissingPoints,
		Feedback:      result.Feedback,
		SM2Grade:      sm2Grade,
	}, nil
}

func parseRetentionFromFile(path string) *retention.Retention {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)

	// Simple frontmatter parse for retention fields.
	lines := strings.Split(content, "\n")
	inFM := false
	inRetention := false
	r := &retention.Retention{EaseFactor: 2.5}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFM {
				break
			}
			inFM = true
			continue
		}
		if !inFM {
			continue
		}

		if trimmed == "retention:" {
			inRetention = true
			continue
		}
		if inRetention && !strings.HasPrefix(line, "  ") && trimmed != "" {
			inRetention = false
		}

		if inRetention {
			kv := strings.SplitN(strings.TrimSpace(trimmed), ":", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(strings.Trim(kv[1], `"'`))

			switch key {
			case "state":
				r.State = val
			case "next_review":
				r.NextReview = val
			case "retention_score":
				fmt.Sscanf(val, "%d", &r.RetentionScore)
			case "lapse_count":
				fmt.Sscanf(val, "%d", &r.LapseCount)
			case "streak":
				fmt.Sscanf(val, "%d", &r.Streak)
			}
		}
	}

	if r.State == "" {
		return nil
	}
	return r
}
