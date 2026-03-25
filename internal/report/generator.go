package report

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/search"
	"github.com/ty-cooper/ngram/internal/vault"
)

// ReportGenerator orchestrates per-section LLM calls to build a report.
type ReportGenerator struct {
	VaultPath string
	BoxName   string
	Runner    *llm.Runner
	Search    *search.Client
	Template  *ReportTemplate
	Variables map[string]string
}

// GeneratedReport is the complete report output.
type GeneratedReport struct {
	Sections []ReportSection
	BoxName  string
}

// ReportSection is a single section of the report.
type ReportSection struct {
	Name    string
	Title   string
	Content string // markdown
}

// ReportFinding is a structured finding for the findings section.
type ReportFinding struct {
	Title       string
	Severity    string
	Description string
	Impact      string
	Evidence    []string
	Remediation string
}

// Generate builds the full report.
func (g *ReportGenerator) Generate(ctx context.Context) (*GeneratedReport, error) {
	// Collect all notes for this box.
	notes := g.collectNotes()
	if len(notes) == 0 {
		return nil, fmt.Errorf("no notes found for box %s", g.BoxName)
	}

	notesSummary := g.buildNotesSummary(notes)

	report := &GeneratedReport{BoxName: g.BoxName}

	sections := g.Template.Sections
	if len(sections) == 0 {
		sections = []string{"executive_summary", "scope_methodology", "findings", "remediation_summary"}
	}

	for _, section := range sections {
		prompt := g.sectionPrompt(section, notesSummary)
		if prompt == "" {
			continue
		}

		out, err := g.Runner.Run(ctx, prompt,
			llm.WithSystemPrompt("You are a professional penetration testing report writer. Write concise, technical, client-ready content. Use markdown formatting."),
			llm.WithMaxTokens(4096),
		)
		if err != nil {
			log.Printf("warn: report section %s failed: %v", section, err)
			continue
		}

		title := sectionTitle(section)
		report.Sections = append(report.Sections, ReportSection{
			Name:    section,
			Title:   title,
			Content: string(out),
		})
	}

	return report, nil
}

func (g *ReportGenerator) collectNotes() []string {
	var notes []string

	// Walk box directory.
	boxDir := filepath.Join(g.VaultPath, "boxes", g.BoxName)
	filepath.Walk(boxDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		notes = append(notes, string(data))
		return nil
	})

	// Also search Meilisearch for notes tagged with this box.
	if g.Search != nil {
		results, err := g.Search.FindSimilarFiltered("", 100, `box = "`+g.BoxName+`"`)
		if err == nil {
			for _, r := range results {
				path := vault.FindNoteByID(g.VaultPath, r.ID)
				if path != "" {
					data, err := os.ReadFile(path)
					if err == nil {
						notes = append(notes, string(data))
					}
				}
			}
		}
	}

	return notes
}

func (g *ReportGenerator) buildNotesSummary(notes []string) string {
	var b strings.Builder
	for i, note := range notes {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		// Truncate long notes.
		if len(note) > 2000 {
			note = note[:2000] + "\n...(truncated)"
		}
		b.WriteString(note)
	}
	return b.String()
}

func (g *ReportGenerator) sectionPrompt(section, notesSummary string) string {
	vars := g.variableContext()

	switch section {
	case "executive_summary":
		return fmt.Sprintf(`Write an executive summary for a penetration test report.
%s

Based on these engagement notes:

%s

Keep it to 2-3 paragraphs. Focus on overall risk posture, critical findings, and key recommendations.`, vars, notesSummary)

	case "scope_methodology":
		return fmt.Sprintf(`Write a Scope and Methodology section for a penetration test report.
%s

Based on these engagement notes:

%s

Include: target scope, testing methodology, tools used, and timeline.`, vars, notesSummary)

	case "findings":
		return fmt.Sprintf(`Write the Findings section for a penetration test report.
%s

Based on these engagement notes:

%s

For each finding include: title, severity (Critical/High/Medium/Low/Info), description, impact, evidence, and remediation. Sort by severity (critical first).`, vars, notesSummary)

	case "remediation_summary":
		return fmt.Sprintf(`Write a Remediation Summary section for a penetration test report.
%s

Based on these engagement notes:

%s

Provide prioritized remediation recommendations. Group by priority (immediate, short-term, long-term).`, vars, notesSummary)
	}
	return ""
}

func (g *ReportGenerator) variableContext() string {
	if len(g.Variables) == 0 {
		return ""
	}
	var parts []string
	for k, v := range g.Variables {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	return "Context: " + strings.Join(parts, ", ")
}

func sectionTitle(name string) string {
	switch name {
	case "executive_summary":
		return "Executive Summary"
	case "scope_methodology":
		return "Scope and Methodology"
	case "findings":
		return "Findings"
	case "remediation_summary":
		return "Remediation Summary"
	case "appendix":
		return "Appendix"
	default:
		return strings.Title(strings.ReplaceAll(name, "_", " "))
	}
}

// ToMarkdown renders the report as markdown.
func (r *GeneratedReport) ToMarkdown() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Penetration Test Report — %s\n\n", r.BoxName))

	for _, s := range r.Sections {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", s.Title, s.Content)
	}

	return b.String()
}
