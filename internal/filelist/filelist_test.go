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

func TestNarrowFlatViewWrapsFullBasenameInsteadOfTruncating(t *testing.T) {
	m := New()
	m.SetSize(18, 10)
	m.SetFiles([]discover.FileEntry{{
		Path:    "docs/deep/very-long-component-name-alpha.md",
		ModTime: time.Now(),
	}})

	view := m.View()
	for _, want := range []string{"very-long-compo", "nent-name-alpha", ".md"} {
		if !strings.Contains(view, want) {
			t.Fatalf("narrow file list did not wrap full basename chunk %q in view %q", want, view)
		}
	}
	if strings.Contains(view, "...") {
		t.Fatalf("narrow basename should wrap instead of truncating: %q", view)
	}
}

func TestNarrowFlatViewMarksWrappedSelectionOnlyOnFirstLine(t *testing.T) {
	m := New()
	m.SetSize(18, 10)
	m.SetFiles([]discover.FileEntry{{
		Path:    "docs/deep/very-long-component-name-alpha.md",
		ModTime: time.Now(),
	}})

	lines := strings.Split(m.View(), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected wrapped filename lines, got %q", m.View())
	}
	if !strings.HasPrefix(lines[0], "• ") {
		t.Fatalf("selected file should mark the first visual line: %q", lines[0])
	}
	for _, line := range lines[1:3] {
		if strings.HasPrefix(line, "• ") {
			t.Fatalf("wrapped continuation line should not repeat the selection marker: %q in view %q", line, m.View())
		}
	}
}
