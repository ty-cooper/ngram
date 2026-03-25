package report

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ReportTemplate defines the structure and style of a report.
type ReportTemplate struct {
	Name     string            `yaml:"name"`
	Sections []string          `yaml:"sections"`
	Vars     map[string]string `yaml:"variables"`
}

// DefaultTemplate returns the built-in report template.
func DefaultTemplate() *ReportTemplate {
	return &ReportTemplate{
		Name:     "default",
		Sections: []string{"executive_summary", "scope_methodology", "findings", "remediation_summary"},
	}
}

// LoadTemplate loads a report template from _templates/report/.
func LoadTemplate(vaultPath, name string) (*ReportTemplate, error) {
	path := filepath.Join(vaultPath, "_templates", "report", name+".yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tmpl ReportTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}
