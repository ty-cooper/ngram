package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/search"
)

var reportCmd = &cobra.Command{
	Use:   "report <box>",
	Short: "Generate a draft pentest report for a box",
	Args:  cobra.ExactArgs(1),
	RunE:  reportRun,
}

func reportRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	boxName := args[0]
	boxDir := filepath.Join(c.VaultPath, "boxes", boxName)

	if _, err := os.Stat(boxDir); os.IsNotExist(err) {
		return fmt.Errorf("box %q not found at %s", boxName, boxDir)
	}

	// Collect all notes for this box.
	var notes []noteForReport
	phases := []string{"_recon", "_enum", "_exploit", "_post", "_loot"}

	for _, phase := range phases {
		phaseDir := filepath.Join(boxDir, phase)
		entries, err := os.ReadDir(phaseDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(phaseDir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			notes = append(notes, noteForReport{
				Phase:   strings.TrimPrefix(phase, "_"),
				Path:    path,
				Content: string(data),
			})
		}
	}

	// Also search Meilisearch for box-tagged notes.
	client, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err == nil {
		resp, err := client.Search("", search.SearchOptions{
			Filter: fmt.Sprintf(`box = "%s"`, boxName),
			Limit:  100,
		})
		if err == nil {
			for _, r := range resp.Results {
				path := filepath.Join(c.VaultPath, r.FilePath)
				data, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				notes = append(notes, noteForReport{
					Phase:   "indexed",
					Path:    path,
					Content: string(data),
				})
			}
		}
	}

	if len(notes) == 0 {
		return fmt.Errorf("no notes found for box %q", boxName)
	}

	fmt.Printf("[+] compiling notes for: %s\n", boxName)
	fmt.Printf("[+] %d notes across %d phases\n", len(notes), countPhases(notes))

	// Build report via LLM.
	runner := llm.NewRunner(c.Model, c.VaultPath)

	prompt := buildReportPrompt(boxName, notes)
	out, err := runner.Run(context.Background(), prompt)
	if err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	// Write report.
	reportsDir := filepath.Join(c.VaultPath, "reports")
	os.MkdirAll(reportsDir, 0o755)

	reportFile := filepath.Join(reportsDir, fmt.Sprintf("%s-draft.md", boxName))
	if err := os.WriteFile(reportFile, out, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	fmt.Printf("[+] report saved: reports/%s-draft.md\n", boxName)
	return nil
}

type noteForReport struct {
	Phase   string
	Path    string
	Content string
}

func buildReportPrompt(boxName string, notes []noteForReport) string {
	var b strings.Builder

	b.WriteString("Generate a draft pentest report for the following engagement.\n\n")
	fmt.Fprintf(&b, "TARGET: %s\n", boxName)
	fmt.Fprintf(&b, "DATE: %s\n\n", time.Now().Format("2006-01-02"))

	b.WriteString("REPORT STRUCTURE:\n")
	b.WriteString("1. Executive Summary\n")
	b.WriteString("2. Scope & Methodology\n")
	b.WriteString("3. Findings (sorted by severity: critical, high, medium, low, info)\n")
	b.WriteString("   - Each finding: title, description, impact, evidence, remediation\n")
	b.WriteString("4. Appendix (command log, credentials found)\n\n")

	b.WriteString("NOTES FROM ENGAGEMENT:\n\n")

	for _, n := range notes {
		fmt.Fprintf(&b, "--- [%s] %s ---\n%s\n\n",
			n.Phase, filepath.Base(n.Path), truncateReport(n.Content, 2000))
	}

	b.WriteString("Generate the complete report in markdown format.")
	return b.String()
}

func countPhases(notes []noteForReport) int {
	seen := make(map[string]bool)
	for _, n := range notes {
		seen[n.Phase] = true
	}
	return len(seen)
}

func truncateReport(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n[...truncated]"
}
