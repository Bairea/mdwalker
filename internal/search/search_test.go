package search

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bairea/mdwalker/internal/discover"
)

func TestFileSearchViewShowsCandidatePaths(t *testing.T) {
	m := New()
	m.SetSize(100, 28)
	m.Activate(ModeFileName)
	m.Query = "prd"
	m.SearchFiles([]discover.FileEntry{
		{Path: "README.md", ModTime: time.Now()},
		{Path: "prd3.md", ModTime: time.Now()},
		{Path: "docs/prd-notes.md", ModTime: time.Now()},
	})

	view := m.View()
	for _, want := range []string{"prd3.md", "docs/prd-notes.md"} {
		if !strings.Contains(view, want) {
			t.Fatalf("file search view missing candidate %q in %q", want, view)
		}
	}
}

func TestFileSearchModalKeepsHeaderAndCandidatesOnStableLines(t *testing.T) {
	m := New()
	m.SetSize(100, 28)
	m.Activate(ModeFileName)
	m.Query = "li"
	m.SearchFiles([]discover.FileEntry{
		{Path: "testdata/links.md", ModTime: time.Now()},
		{Path: "testdata/lists.md", ModTime: time.Now()},
	})

	lines := strings.Split(m.View(), "\n")
	headerLine := ""
	for _, line := range lines {
		plain := stripANSI(line)
		if strings.Contains(plain, "Search files") {
			headerLine = plain
			break
		}
	}
	if !strings.Contains(headerLine, "files") || !strings.Contains(headerLine, "1/2") {
		t.Fatalf("search header should keep mode/count on one line, header=%q view=%q", headerLine, m.View())
	}

	selectedLine := -1
	for i, line := range lines {
		if strings.Contains(stripANSI(line), "testdata/links.md") {
			selectedLine = i
			break
		}
	}
	if selectedLine < 0 {
		t.Fatalf("selected candidate missing from search view: %q", m.View())
	}
	if selectedLine+1 >= len(lines) || !strings.Contains(stripANSI(lines[selectedLine+1]), "testdata/lists.md") {
		t.Fatalf("selected candidate wrapped or inserted a filler line before next candidate: %q", m.View())
	}
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;:]*[A-Za-z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestAllContentSearchViewShowsMatches(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "api.md"), []byte("# API\nGET /users\nPOST /users\n"), 0644)
	os.WriteFile(filepath.Join(root, "guide.md"), []byte("# Guide\nGetting users is easy\n"), 0644)

	m := New()
	m.SetSize(100, 28)
	m.Activate(ModeAllContent)
	m.Query = "users"
	m.SearchAllContent(root, []discover.FileEntry{
		{Path: "api.md", ModTime: time.Now()},
		{Path: "guide.md", ModTime: time.Now()},
	})

	view := m.View()
	for _, want := range []string{"api.md", "guide.md"} {
		if !strings.Contains(view, want) {
			t.Fatalf("all-content search view missing file %q in %q", want, view)
		}
	}
}

func TestToggleModeCyclesThroughThreeModes(t *testing.T) {
	m := New()
	m.Activate(ModeFileName)

	m.ToggleMode()
	if m.Mode != ModeContent {
		t.Fatalf("after Tab from FileName: mode=%v, want ModeContent", m.Mode)
	}

	m.ToggleMode()
	if m.Mode != ModeAllContent {
		t.Fatalf("after Tab from Content: mode=%v, want ModeAllContent", m.Mode)
	}

	m.ToggleMode()
	if m.Mode != ModeFileName {
		t.Fatalf("after Tab from AllContent: mode=%v, want ModeFileName", m.Mode)
	}
}

func TestAllContentSearchEmptyQuery(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "test.md"), []byte("hello\n"), 0644)

	m := New()
	m.SetSize(100, 28)
	m.Activate(ModeAllContent)
	m.Query = ""
	m.SearchAllContent(root, []discover.FileEntry{{Path: "test.md", ModTime: time.Now()}})

	if len(m.AllMatches) != 0 {
		t.Fatalf("empty query should produce no matches, got %d", len(m.AllMatches))
	}
}

func TestAllContentSearchNoMatch(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "test.md"), []byte("hello world\n"), 0644)

	m := New()
	m.SetSize(100, 28)
	m.Activate(ModeAllContent)
	m.Query = "nonexistent"
	m.SearchAllContent(root, []discover.FileEntry{{Path: "test.md", ModTime: time.Now()}})

	if len(m.AllMatches) != 0 {
		t.Fatalf("no match expected, got %d", len(m.AllMatches))
	}
}
