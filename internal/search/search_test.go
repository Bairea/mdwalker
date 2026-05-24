package search

import (
	"strings"
	"testing"
	"time"

	"github.com/bairea/mdwalker/internal/discover"
)

func TestFileSearchViewShowsCandidatePaths(t *testing.T) {
	m := New()
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
