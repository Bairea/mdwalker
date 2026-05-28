package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bairea/mdwalker/internal/config"
)

func TestScanIncludesMarkdownAndPreviewableImages(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"notes.md", "diagram.png", "photo.jpg", "ignored.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := Scan(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, entry := range entries {
		got[entry.Path] = true
	}
	for _, want := range []string{"notes.md", "diagram.png", "photo.jpg"} {
		if !got[want] {
			t.Fatalf("Scan() missing previewable file %q; got paths %#v", want, got)
		}
	}
	if got["ignored.txt"] {
		t.Fatalf("Scan() included unsupported file ignored.txt")
	}
}

func TestScanSkipSubdirs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	os.MkdirAll(filepath.Join(".claude", "skills", "golang-cli"), 0755)
	os.WriteFile(filepath.Join(".claude", "skills", "golang-cli", "SKILL.md"), []byte("# skill"), 0644)
	os.WriteFile(filepath.Join(".claude", "CLAUDE.md"), []byte("# claude config"), 0644)

	wl := &config.WhitelistConfig{
		Unignore: config.WhitelistUnignore{
			DotDirs: []string{".claude"},
		},
		SkipSubdirs: []string{"*/skills"},
	}

	entries, err := Scan(dir, wl)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if strings.Contains(e.Path, "/skills/") {
			t.Errorf("entry in skipped dir should not appear: %s", e.Path)
		}
	}

	foundClaudeMD := false
	for _, e := range entries {
		if e.Path == filepath.Join(".claude", "CLAUDE.md") {
			foundClaudeMD = true
		}
	}
	if !foundClaudeMD {
		t.Error("expected .claude/CLAUDE.md to be discovered")
	}
}

func TestShouldSkipDirMatchesSlashPatternsOnWindowsPaths(t *testing.T) {
	dir := filepath.Join(".claude", "skills")

	if !shouldSkipDir(dir, []string{"*/skills"}) {
		t.Fatalf("shouldSkipDir(%q, [*/skills]) = false, want true", dir)
	}
}

func TestScanDedup(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	os.MkdirAll("docs", 0755)
	os.WriteFile(filepath.Join("docs", "design.md"), []byte("# design"), 0644)

	wl := &config.WhitelistConfig{
		Unignore: config.WhitelistUnignore{
			Paths: []string{"docs"},
		},
	}

	entries, err := Scan(dir, wl)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for _, e := range entries {
		if e.Path == filepath.Join("docs", "design.md") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 entry for design.md, got %d (duplicate detected)", count)
	}
}
