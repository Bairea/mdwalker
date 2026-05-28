package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bairea/mdwalker/internal/discover"
	"github.com/bairea/mdwalker/internal/search"
	tea "github.com/charmbracelet/bubbletea"
)

type rendererProbeModel struct {
	view string
}

func (m rendererProbeModel) Init() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return tea.WindowSizeMsg{Width: 40, Height: 8} },
		func() tea.Msg { return tea.Quit() },
	)
}

func (m rendererProbeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		return m, nil
	}
	return m, nil
}

func (m rendererProbeModel) View() string {
	return m.view
}

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
		Y:      2,
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

func TestFileCursorMovementPreviewDoesNotLeakImagePlaceholders(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "plain.md"), []byte("# Plain"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "images.md"), []byte("# Images\n\n![screenshot](screenshot.png)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "screenshot.png"), []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("WEZTERM_EXECUTABLE", "")

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{
		{Path: "plain.md", ModTime: time.Now()},
		{Path: "images.md", ModTime: time.Now()},
	}})
	m.focus = focusFiles

	m.Update(tea.KeyMsg{Type: tea.KeyDown})

	view := m.View()
	if strings.Contains(view, "%MDWALKER_IMG_") {
		t.Fatalf("lightweight markdown preview leaked image placeholder token: %q", view)
	}
	if !strings.Contains(view, "[Image: screenshot.png]") {
		t.Fatalf("lightweight markdown preview did not render image placeholder: %q", view)
	}
}

func TestFileCursorMovementPreviewDoesNotLeakImagePlaceholdersForSubdirFile(t *testing.T) {
	root := filepath.Join("..", "..")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{
		{Path: filepath.Join("testdata", "semantic.md"), ModTime: time.Now()},
		{Path: filepath.Join("testdata", "images.md"), ModTime: time.Now()},
	}})
	m.focus = focusFiles

	m.Update(tea.KeyMsg{Type: tea.KeyDown})

	view := m.View()
	if strings.Contains(view, "%MDWALKER_IMG_") {
		t.Fatalf("subdir markdown preview leaked image placeholder token: %q", view)
	}
	if !strings.Contains(view, "\x1b]1337;File=") {
		t.Fatalf("subdir markdown preview did not render native referenced image via iTerm2: %q", view)
	}
}

func TestOpenSelectedMarkdownImageFileDoesNotLeakImagePlaceholders(t *testing.T) {
	root := filepath.Join("..", "..")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	entries, err := discover.Scan(root, &m.wl)
	if err != nil {
		t.Fatal(err)
	}
	m.Update(filesLoadedMsg{entries: entries})
	for i, entry := range m.files.Entries {
		if entry.Path == filepath.Join("testdata", "images.md") {
			m.files.Cursor = i
			m.files.UpdateViewport()
			break
		}
	}

	m.openSelectedFile()

	view := m.View()
	if strings.Contains(view, "%MDWALKER_IMG_") {
		t.Fatalf("opened markdown image file leaked image placeholder token: %q", view)
	}
	if !strings.Contains(view, "\x1b]1337;File=") {
		t.Fatalf("opened markdown image file did not render native image via iTerm2: %q", view)
	}
}

func TestMarkdownImagePreviewNeverLeaksPlaceholdersAcrossWidths(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, width := range []int{62, 80, 100, 120, 160} {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			t.Setenv("KITTY_WINDOW_ID", "")
			t.Setenv("TERM", "xterm-256color")
			t.Setenv("TERM_PROGRAM", "WezTerm")

			m := New(root)
			m.Update(tea.WindowSizeMsg{Width: width, Height: 16})
			m.Update(filesLoadedMsg{entries: []discover.FileEntry{
				{Path: filepath.Join("testdata", "semantic.md"), ModTime: time.Now()},
				{Path: filepath.Join("testdata", "images.md"), ModTime: time.Now()},
			}})
			m.focus = focusFiles

			m.Update(tea.KeyMsg{Type: tea.KeyDown})
			lightView := m.View()
			if strings.Contains(lightView, "%MDWALKER_IMG_") {
				t.Fatalf("light preview leaked image placeholder at width %d: %q", width, lightView)
			}

			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			fullView := m.View()
			if strings.Contains(fullView, "%MDWALKER_IMG_") {
				t.Fatalf("open preview leaked image placeholder at width %d: %q", width, fullView)
			}
		})
	}
}

func TestFileCursorMovementPreviewsImageFileNatively(t *testing.T) {
	root := filepath.Join("..", "..")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{
		{Path: filepath.Join("testdata", "semantic.md"), ModTime: time.Now()},
		{Path: filepath.Join("testdata", "screenshot.png"), ModTime: time.Now()},
	}})
	m.focus = focusFiles

	m.Update(tea.KeyMsg{Type: tea.KeyDown})

	view := m.View()
	if !strings.Contains(view, "\x1b]1337;File=") {
		t.Fatalf("image file cursor preview did not render native image via iTerm2: %q", view)
	}
}

func TestOpenCurrentImageResolvesRelativePathFromMarkdownFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(root, "docs", "doc.md")
	imagePath := filepath.Join(root, "docs", "assets", "pic.png")
	if err := os.WriteFile(docPath, []byte("# Doc\n\n![pic](assets/pic.png)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	var opened string
	oldOpenImage := openImage
	openImage = func(path string) error {
		opened = path
		return nil
	}
	defer func() { openImage = oldOpenImage }()

	m := New(root)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(filesLoadedMsg{entries: []discover.FileEntry{{Path: filepath.Join("docs", "doc.md"), ModTime: time.Now()}}})
	m.openSelectedFile()
	m.preview.ScrollToLine(2)

	m.openCurrentImage()

	if opened != imagePath {
		t.Fatalf("openCurrentImage opened %q, want %q", opened, imagePath)
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

func TestRenderPanePreservesITermImageSequence(t *testing.T) {
	image := "\x1b]1337;File=width=40;height=10;inline=1:abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789\x07"

	view := renderPane(image, 60, 12, true)

	if !strings.Contains(view, image) {
		t.Fatalf("renderPane corrupted iTerm image sequence: %q", view)
	}
}

func TestRenderPanePreservesKittyImageSequence(t *testing.T) {
	image := "\x1b_Ga=T,f=100,t=f,c=40,r=10;abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789\x1b\\"

	view := renderPane(image, 60, 12, true)

	if !strings.Contains(view, image) {
		t.Fatalf("renderPane corrupted kitty image sequence: %q", view)
	}
}

func TestBubbleTeaRendererPreservesKittyImageSequence(t *testing.T) {
	image := "\x1b_Ga=T,f=100,t=f,c=40,r=10;abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789\x1b\\"
	var out bytes.Buffer

	_, err := tea.NewProgram(
		rendererProbeModel{view: image},
		tea.WithInput(nil),
		tea.WithOutput(&out),
		tea.WithoutSignalHandler(),
		tea.WithoutCatchPanics(),
	).Run()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), image) {
		t.Fatalf("bubbletea renderer corrupted kitty image sequence: %q", out.String())
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
		Y:      6,
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
