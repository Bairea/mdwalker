package filelist

import (
	"strings"
	"testing"
	"time"

	"github.com/bairea/mdwalker/internal/discover"
)

func TestNarrowFlatViewKeepsDistinctiveBasenameSuffix(t *testing.T) {
	m := New()
	m.SetSize(28, 10)
	m.SetFiles([]discover.FileEntry{{
		Path:    "docs/deep/nested/very-long-component-name-alpha.md",
		ModTime: time.Now(),
	}})

	view := m.View()
	if !strings.Contains(view, "alpha.md") {
		t.Fatalf("narrow file list lost the distinguishing basename suffix: %q", view)
	}
	if strings.Contains(view, "docs/deep/nested") {
		t.Fatalf("narrow file list should not spend pane width on parent directories: %q", view)
	}
}

func TestNarrowFlatViewShowsFileMarkerAndBasename(t *testing.T) {
	m := New()
	m.SetSize(28, 10)
	m.SetFiles([]discover.FileEntry{{
		Path:    "docs/deep/alpha.md",
		ModTime: time.Now(),
	}})

	view := m.View()
	if !strings.Contains(view, "• alpha.md") {
		t.Fatalf("narrow file list should show a file marker and basename: %q", view)
	}
	if strings.Contains(view, "docs/deep") {
		t.Fatalf("narrow file list should not spend pane width on parent directories: %q", view)
	}
}
