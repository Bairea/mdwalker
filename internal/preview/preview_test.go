package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToggleFoldFoldsCurrentSectionWhenCursorIsInsideIt(t *testing.T) {
	root := t.TempDir()
	content := "# First\nintro\n## Child\nchild body\n# Second\nsecond body\n"
	if err := os.WriteFile(filepath.Join(root, "headings.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 20)
	if err := m.LoadFile(root, "headings.md"); err != nil {
		t.Fatal(err)
	}

	m.ToggleFold(1)

	for _, hidden := range []int{1, 2, 3} {
		if m.IsLineVisible(hidden) {
			t.Fatalf("line %d remained visible after folding the current section", hidden)
		}
	}
	if !m.IsLineVisible(4) {
		t.Fatalf("next top-level heading should remain visible")
	}
}

func TestScrollToLinePlacesRenderedHeadingAtTop(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("# Top\n\n")
	for i := 0; i < 20; i++ {
		b.WriteString("filler paragraph with enough text to render\n\n")
	}
	targetLine := strings.Count(b.String(), "\n")
	b.WriteString("## Target Heading\nbody\n")
	for i := 0; i < 20; i++ {
		b.WriteString("after target filler\n\n")
	}
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 8)
	if err := m.LoadFile(root, "doc.md"); err != nil {
		t.Fatal(err)
	}

	m.ScrollToLine(targetLine)

	firstLine := strings.Split(m.View(), "\n")[0]
	if !strings.Contains(firstLine, "Target Heading") {
		t.Fatalf("ScrollToLine did not place rendered heading at top; first line=%q view=%q", firstLine, m.View())
	}
}

func TestScrollToLineUsesRenderedOffsetNotSourceLineApproximation(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("# Top\n\n")
	for i := 0; i < 8; i++ {
		b.WriteString("- list item before target\n")
		b.WriteString("  - nested item before target\n")
	}
	targetLine := strings.Count(b.String(), "\n")
	b.WriteString("## Precise Target\n")
	b.WriteString("body that should not be the first visible line\n")
	for i := 0; i < 8; i++ {
		b.WriteString("after target filler\n\n")
	}
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 8)
	if err := m.LoadFile(root, "doc.md"); err != nil {
		t.Fatal(err)
	}

	m.ScrollToLine(targetLine)

	firstLine := strings.Split(m.View(), "\n")[0]
	if !strings.Contains(firstLine, "Precise Target") {
		t.Fatalf("ScrollToLine first visible line = %q, want target heading; view=%q", firstLine, m.View())
	}
	if strings.Contains(firstLine, "body that should not") {
		t.Fatalf("ScrollToLine skipped past target heading: %q", m.View())
	}
}
