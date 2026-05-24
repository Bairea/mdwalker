package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bairea/mdwalker/internal/discover"
	"github.com/bairea/mdwalker/internal/search"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSlashActivatesFileSearchWithoutTypingSlash(t *testing.T) {
	m := New(t.TempDir())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.focus = focusFiles

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	if !m.search.Active {
		t.Fatal("search was not activated")
	}
	if m.search.Query != "" {
		t.Fatalf("slash key leaked into search query: %q", m.search.Query)
	}
}

func TestMouseClickInFilePaneOpensClickedFile(t *testing.T) {
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

	m.Update(tea.MouseMsg{
		X:      1,
		Y:      1,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	})

	if got := m.preview.FilePath(); got != "second.md" {
		t.Fatalf("mouse click opened %q, want second.md", got)
	}
	if m.focus != focusPreview {
		t.Fatalf("focus = %v, want preview after opening clicked file", m.focus)
	}
}

func TestFileCursorMovementPreviewsSelectedFileWithoutLeavingFilePane(t *testing.T) {
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

	m.Update(tea.KeyMsg{Type: tea.KeyDown})

	if got := m.preview.FilePath(); got != "second.md" {
		t.Fatalf("file movement previewed %q, want second.md", got)
	}
	if m.focus != focusFiles {
		t.Fatalf("focus = %v, want to remain in file pane", m.focus)
	}
}

func TestFileCursorMovementDoesNotInvokeExternalMermaidRenderer(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "plain.md"), []byte("# Plain"), 0644); err != nil {
		t.Fatal(err)
	}
	mermaidDoc := "```mermaid\nflowchart TD\nA-->B\n```\n"
	if err := os.WriteFile(filepath.Join(root, "diagram.md"), []byte(mermaidDoc), 0644); err != nil {
		t.Fatal(err)
	}
	mmdcPath := filepath.Join(root, "mmdc")
	if err := os.WriteFile(mmdcPath, []byte("#!/bin/sh\nprintf called >> \"$MMDC_CALLED_LOG\"\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "mmdc.log")
	t.Setenv("MDWALKER_MMDC", mmdcPath)
	t.Setenv("MMDC_CALLED_LOG", logPath)
	t.Setenv("HOME", filepath.Join(root, "home"))

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{
		{Path: "plain.md", ModTime: time.Now()},
		{Path: "diagram.md", ModTime: time.Now()},
	}})
	m.focus = focusFiles

	m.Update(tea.KeyMsg{Type: tea.KeyDown})

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("file cursor movement invoked external mermaid renderer; err=%v", err)
	}
	if got := m.preview.FilePath(); got != "diagram.md" {
		t.Fatalf("lightweight movement previewed %q, want diagram.md", got)
	}
}

func TestStatusBarShowsCurrentFocus(t *testing.T) {
	m := New(t.TempDir())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.focus = focusPreview

	if !strings.Contains(m.renderStatusBar(), "focus:preview") {
		t.Fatalf("status bar did not show current focus: %q", m.renderStatusBar())
	}
}

func TestSpaceFoldsCurrentPreviewSection(t *testing.T) {
	root := t.TempDir()
	content := "# First\nhidden body\nstill hidden\n# Second\nvisible body\n"
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{{Path: "doc.md", ModTime: time.Now()}}})
	m.openSelectedFile()

	m.Update(tea.KeyMsg{Type: tea.KeySpace})

	view := m.preview.View()
	if strings.Contains(view, "hidden body") || strings.Contains(view, "still hidden") {
		t.Fatalf("Space did not fold the current section; view=%q", view)
	}
	if !strings.Contains(view, "...") {
		t.Fatalf("fold marker missing after Space; view=%q", view)
	}
}

func TestClickPreviewHeadingThenSpaceFoldsThatHeadingOnly(t *testing.T) {
	root := t.TempDir()
	content := "# First\nfirst body\n# Second\nsecond body\n"
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{{Path: "doc.md", ModTime: time.Now()}}})
	m.openSelectedFile()

	m.Update(tea.MouseMsg{
		X:      m.filesWidth + 2,
		Y:      5,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	})
	m.Update(tea.KeyMsg{Type: tea.KeySpace})

	view := m.preview.View()
	if !strings.Contains(view, "first body") {
		t.Fatalf("clicking second heading folded the wrong section; view=%q", view)
	}
	if strings.Contains(view, "second body") {
		t.Fatalf("clicked heading body remained visible after Space; view=%q", view)
	}
}

func TestFileSearchTabOpensCandidateAndSwitchesToContentSearch(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"first.md", "second.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# "+name+"\nneedle\n"), 0644); err != nil {
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
	m.Update(tea.KeyMsg{Type: tea.KeyTab})

	if got := m.preview.FilePath(); got != "second.md" {
		t.Fatalf("Tab opened %q, want second.md", got)
	}
	if !m.search.Active || m.search.Mode != search.ModeContent {
		t.Fatalf("Tab should keep search active in content mode; active=%v mode=%v", m.search.Active, m.search.Mode)
	}
	if m.focus != focusSearch {
		t.Fatalf("focus = %v, want search", m.focus)
	}
	if m.search.Query != "" {
		t.Fatalf("content search query = %q, want empty after file selection", m.search.Query)
	}
}

func TestContentSearchEnterJumpsToCurrentMatch(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	for i := 0; i < 40; i++ {
		if i == 30 {
			b.WriteString("needle target\n")
		} else {
			b.WriteString("filler line\n")
		}
	}
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{{Path: "doc.md", ModTime: time.Now()}}})
	m.openSelectedFile()

	if strings.Contains(m.preview.View(), "needle target") {
		t.Fatalf("test setup invalid: needle is visible before searching")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "needle" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !strings.Contains(m.preview.View(), "needle target") {
		t.Fatalf("content search Enter did not jump to the match; view=%q", m.preview.View())
	}
}

func TestOpeningCodeblocksFixtureAndScrollingDoesNotPanic(t *testing.T) {
	m := New(filepath.Join("..", "..", "testdata"))
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{{Path: "codeblocks.md", ModTime: time.Now()}}})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("opening and scrolling codeblocks.md panicked: %v", r)
		}
	}()

	m.openSelectedFile()
	for i := 0; i < 25; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if got := m.preview.FilePath(); got != "codeblocks.md" {
		t.Fatalf("opened %q, want codeblocks.md", got)
	}
	if view := m.preview.View(); view == "" {
		t.Fatalf("codeblocks preview rendered empty view")
	}
}
