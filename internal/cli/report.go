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
	"github.com/ty-cooper/ngram/internal/report"
	"github.com/ty-cooper/ngram/internal/search"
)

var (
	reportFormat   string
	reportTemplate string
	reportVars     []string
)

var reportCmd = &cobra.Command{
	Use:   "report <box>",
	Short: "Generate a draft pentest report for a box",
	Args:  cobra.ExactArgs(1),
	RunE:  reportRun,
}

func init() {
	reportCmd.Flags().StringVar(&reportFormat, "format", "md", "output format: md or docx")
	reportCmd.Flags().StringVar(&reportTemplate, "template", "default", "report template name")
	reportCmd.Flags().StringSliceVar(&reportVars, "var", nil, "template variables (KEY=VALUE)")
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

	// Load template.
	var tmpl *report.ReportTemplate
	if reportTemplate != "default" {
		tmpl, err = report.LoadTemplate(c.VaultPath, reportTemplate)
		if err != nil {
			return fmt.Errorf("load template %q: %w", reportTemplate, err)
		}
	} else {
		tmpl = report.DefaultTemplate()
	}

	// Parse variables.
	vars := parseVars(reportVars)

	// Connect to search (optional).
	var client *search.Client
	sc, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err == nil {
		client = sc
	}

	runner := llm.NewRunner(c.Model, c.VaultPath)

	gen := &report.ReportGenerator{
		VaultPath: c.VaultPath,
		BoxName:   boxName,
		Runner:    runner,
		Search:    client,
		Template:  tmpl,
		Variables: vars,
	}

	fmt.Printf("[+] generating report for: %s\n", boxName)

	generated, err := gen.Generate(context.Background())
	if err != nil {
		return err
	}

	// Output.
	reportsDir := filepath.Join(c.VaultPath, "reports")
	os.MkdirAll(reportsDir, 0o755)

	switch reportFormat {
	case "docx":
		outPath := filepath.Join(reportsDir, fmt.Sprintf("%s-draft.docx", boxName))
		if err := report.WriteDocx(generated, outPath); err != nil {
			return fmt.Errorf("write docx: %w", err)
		}
		fmt.Printf("[+] report saved: reports/%s-draft.docx\n", boxName)

	default: // "md"
		outPath := filepath.Join(reportsDir, fmt.Sprintf("%s-draft.md", boxName))
		md := generated.ToMarkdown()
		if err := os.WriteFile(outPath, []byte(md), 0o644); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		fmt.Printf("[+] report saved: reports/%s-draft.md\n", boxName)
	}

	return nil
}

func parseVars(vars []string) map[string]string {
	result := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// Legacy helpers kept for backward compat.
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
