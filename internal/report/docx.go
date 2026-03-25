package report

import (
	"os"
	"strings"

	docx "github.com/fumiama/go-docx"
)

// WriteDocx generates a .docx file from the report.
func WriteDocx(report *GeneratedReport, outPath string) error {
	doc := docx.New()

	// Title.
	p := doc.AddParagraph()
	p.AddText("Penetration Test Report").Bold().Size("48")

	p2 := doc.AddParagraph()
	p2.AddText(report.BoxName).Size("28")

	// Sections.
	for _, section := range report.Sections {
		// Section heading.
		hp := doc.AddParagraph()
		hp.AddText(section.Title).Bold().Size("32")

		// Section content — split by paragraphs.
		paragraphs := strings.Split(section.Content, "\n\n")
		for _, text := range paragraphs {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}

			// Handle markdown sub-headings.
			if strings.HasPrefix(text, "### ") {
				sp := doc.AddParagraph()
				sp.AddText(strings.TrimPrefix(text, "### ")).Bold().Size("24")
				continue
			}

			// Handle bullet points.
			if strings.HasPrefix(text, "- ") || strings.HasPrefix(text, "* ") {
				lines := strings.Split(text, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					line = strings.TrimPrefix(line, "- ")
					line = strings.TrimPrefix(line, "* ")
					bp := doc.AddParagraph()
					bp.AddText("• " + line)
				}
				continue
			}

			// Handle code blocks.
			if strings.HasPrefix(text, "```") {
				code := strings.TrimPrefix(text, "```")
				if idx := strings.Index(code, "\n"); idx >= 0 {
					code = code[idx+1:]
				}
				code = strings.TrimSuffix(code, "```")
				cp := doc.AddParagraph()
				cp.AddText(strings.TrimSpace(code)).Size("18")
				continue
			}

			// Regular paragraph.
			rp := doc.AddParagraph()
			rp.AddText(text)
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = doc.WriteTo(f)
	return err
}
