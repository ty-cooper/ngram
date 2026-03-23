package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBoxRC(t *testing.T) {
	content := `BOX=optimum
IP=10.10.10.8
PHASE=exploit
ENGAGEMENT=htb-2026
MODEL=cloud
`
	path := writeTemp(t, ".boxrc", content)

	ctx, err := ParseBoxRC(path)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Box != "optimum" {
		t.Errorf("Box = %q, want optimum", ctx.Box)
	}
	if ctx.IP != "10.10.10.8" {
		t.Errorf("IP = %q, want 10.10.10.8", ctx.IP)
	}
	if ctx.Phase != "exploit" {
		t.Errorf("Phase = %q, want exploit", ctx.Phase)
	}
	if ctx.Engagement != "htb-2026" {
		t.Errorf("Engagement = %q, want htb-2026", ctx.Engagement)
	}
}

func TestParseBoxRC_CommentsAndBlanks(t *testing.T) {
	content := `# this is a comment
BOX=alpha

# another comment
IP=192.168.1.1
`
	path := writeTemp(t, ".boxrc", content)

	ctx, err := ParseBoxRC(path)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Box != "alpha" {
		t.Errorf("Box = %q, want alpha", ctx.Box)
	}
	if ctx.IP != "192.168.1.1" {
		t.Errorf("IP = %q, want 192.168.1.1", ctx.IP)
	}
}

func TestParseBoxRC_QuotedValues(t *testing.T) {
	content := `BOX="my box"
IP='10.10.10.8'
`
	path := writeTemp(t, ".boxrc", content)

	ctx, err := ParseBoxRC(path)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Box != "my box" {
		t.Errorf("Box = %q, want 'my box'", ctx.Box)
	}
	if ctx.IP != "10.10.10.8" {
		t.Errorf("IP = %q, want 10.10.10.8", ctx.IP)
	}
}

func TestFindBoxRC_ParentWalk(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a", "b", "c")
	os.MkdirAll(child, 0o755)

	content := "BOX=found\nPHASE=recon\n"
	os.WriteFile(filepath.Join(root, "a", ".boxrc"), []byte(content), 0o644)

	ctx, err := FindBoxRC(child)
	if err != nil {
		t.Fatal(err)
	}
	if ctx == nil {
		t.Fatal("expected to find .boxrc, got nil")
	}
	if ctx.Box != "found" {
		t.Errorf("Box = %q, want found", ctx.Box)
	}
}

func TestFindBoxRC_NotFound(t *testing.T) {
	dir := t.TempDir()
	ctx, err := FindBoxRC(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ctx != nil {
		t.Errorf("expected nil, got %+v", ctx)
	}
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
