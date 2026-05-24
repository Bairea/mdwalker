package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bairea/mdwalker/internal/discover"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestOutlineRendersAsSeparatePanel(t *testing.T) {
	root := t.TempDir()
	content := "# Title\n\n## Usage\nbody\n"
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{{Path: "doc.md", ModTime: time.Now()}}})
	m.openSelectedFile()
	withoutOutline := m.View()
	previewWidthWithoutOutline := lipgloss.Width(strings.Split(m.preview.View(), "\n")[0])

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	withOutline := m.View()

	if !strings.Contains(withOutline, "Outline") {
		t.Fatalf("outline panel was not rendered: %q", withOutline)
	}
	if strings.Count(withOutline, "Usage") <= strings.Count(withoutOutline, "Usage") {
		t.Fatalf("outline did not render as an additional panel; before=%q after=%q", withoutOutline, withOutline)
	}
	previewWidthWithOutline := lipgloss.Width(strings.Split(m.preview.View(), "\n")[0])
	if previewWidthWithOutline >= previewWidthWithoutOutline {
		t.Fatalf("outline did not reserve a separate panel width: preview width stayed at %d", previewWidthWithOutline)
	}
}

func TestSearchPanelRendersAboveSingleLineStatus(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"first.md", "second.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{
		{Path: "first.md", ModTime: time.Now()},
		{Path: "second.md", ModTime: time.Now()},
	}})
	m.focus = focusFiles
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "second" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	view := m.View()
	lines := strings.Split(view, "\n")
	if !strings.Contains(view, "second.md") {
		t.Fatalf("search candidate missing from view: %q", view)
	}
	if strings.Contains(lines[len(lines)-1], "second.md") {
		t.Fatalf("search candidates rendered inside the final status line: %q", lines[len(lines)-1])
	}
}

func TestFileSearchRendersCenteredModalWithKeyboardSelectableCandidates(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha-match.md", "beta-match.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{
		{Path: "alpha-match.md", ModTime: time.Now().Add(-time.Minute)},
		{Path: "beta-match.md", ModTime: time.Now()},
	}})
	m.focus = focusFiles

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "match" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	view := m.View()
	lines := strings.Split(view, "\n")
	titleLine := -1
	for i, line := range lines {
		if strings.Contains(line, "Search files") {
			titleLine = i
			break
		}
	}
	if titleLine < 8 || titleLine > 20 {
		t.Fatalf("file search did not render as a centered modal; title line=%d view=%q", titleLine, view)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got := m.preview.FilePath(); got != "beta-match.md" {
		t.Fatalf("down+enter opened %q, want beta-match.md", got)
	}
}

func TestMouseClickInOutlineJumpsPreviewToHeading(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("# Top\n")
	for i := 0; i < 30; i++ {
		b.WriteString("filler\n")
	}
	b.WriteString("## Target\nbody\n")
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 14})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{{Path: "doc.md", ModTime: time.Now()}}})
	m.openSelectedFile()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if strings.Contains(m.preview.View(), "Target") {
		t.Fatalf("test setup invalid: target heading is visible before outline click")
	}

	m.Update(tea.MouseMsg{
		X:      m.width - m.outlineWidth + 1,
		Y:      4,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	})

	if !strings.Contains(m.preview.View(), "Target") {
		t.Fatalf("outline click did not jump preview to target heading; view=%q", m.preview.View())
	}
	if m.focus != focusOutline {
		t.Fatalf("focus = %v, want outline after clicking outline", m.focus)
	}
}

func TestOutlineEnterKeepsOutlineInteractiveForNextSelection(t *testing.T) {
	root := t.TempDir()
	content := "# First\none\n## Second\ntwo\n## Third\nthree\n"
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 14})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{{Path: "doc.md", ModTime: time.Now()}}})
	m.openSelectedFile()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !m.outline.Visible || m.focus != focusOutline {
		t.Fatalf("outline should stay interactive after Enter; visible=%v focus=%v", m.outline.Visible, m.focus)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.outline.SelectedLine() != 4 {
		t.Fatalf("outline did not continue to the next heading after Enter; selected line=%d", m.outline.SelectedLine())
	}
}
