package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var domainsCmd = &cobra.Command{
	Use:   "domains [domain]",
	Short: "List domains and clusters with note counts",
	Long:  "Without args: list all domains. With a domain arg: list clusters under that domain.",
	RunE:  domainsRun,
}

func domainsRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	knowledgeDir := filepath.Join(c.VaultPath, "knowledge")

	if len(args) == 0 {
		return listDomains(knowledgeDir)
	}

	return listClusters(knowledgeDir, args[0])
}

func listDomains(knowledgeDir string) error {
	entries, err := os.ReadDir(knowledgeDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("no domains yet")
			return nil
		}
		return err
	}

	type domainInfo struct {
		Name  string
		Count int
	}

	var domains []domainInfo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		count := countMDFiles(filepath.Join(knowledgeDir, e.Name()))
		domains = append(domains, domainInfo{Name: e.Name(), Count: count})
	}

	sort.Slice(domains, func(i, j int) bool {
		return domains[i].Count > domains[j].Count
	})

	if len(domains) == 0 {
		fmt.Println("no domains yet")
		return nil
	}

	for _, d := range domains {
		fmt.Printf("  %-30s %d notes\n", d.Name, d.Count)
	}
	return nil
}

func listClusters(knowledgeDir, domain string) error {
	domainDir := filepath.Join(knowledgeDir, domain)
	entries, err := os.ReadDir(domainDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("domain %q not found", domain)
		}
		return err
	}

	fmt.Printf("%s:\n", domain)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		count := countMDFiles(filepath.Join(domainDir, e.Name()))
		fmt.Printf("  %-30s %d notes\n", e.Name(), count)
	}
	return nil
}

func countMDFiles(dir string) int {
	count := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".md") {
			count++
		}
		return nil
	})
	return count
}
