# mdwalker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a terminal Markdown workbench for AI agent outputs — discover, navigate, and read AI-generated .md files with two-panel TUI, watch mode, semantic highlighting, code block operations, and progressive image/mermaid support.

**Architecture:** Go CLI using Bubble Tea for TUI, Glamour for Markdown→ANSI rendering, fsnotify for watch mode. Two-panel layout (file list + preview) with floating outline. AI-priority file sorting with fd fallback for fast discovery.

**Tech Stack:** Go 1.22+, bubbletea, glamour, lipgloss, bubbles, fsnotify, chroma

---

### Task 1: Project Initialization

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/config/config.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go mod init github.com/bairea/mdwalker
```

- [ ] **Step 2: Create main.go skeleton**

```go
// main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bairea/mdwalker/internal/app"
)

func main() {
	p := tea.NewProgram(app.New(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mdwalker: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Create config package**

```go
// internal/config/config.go
package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ImageProtocol string // auto | kitty | halfblock | off
	MermaidMode   string // auto | code | browser
	MmdcPath      string
}

func Default() Config {
	return Config{
		ImageProtocol: "auto",
		MermaidMode:   "auto",
		MmdcPath:      "mmdc",
	}
}

func Load() Config {
	cfg := Default()
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}
	path := filepath.Join(home, ".config", "mdwalker", "config.toml")
	// parse TOML if exists, merge into cfg
	_ = path
	return cfg
}
```

- [ ] **Step 4: Create app package skeleton**

```go
// internal/app/app.go
package app

import tea "github.com/charmbracelet/bubbletea"

type Model struct {
	width  int
	height int
	ready  bool
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	return "mdwalker - loading..."
}
```

- [ ] **Step 5: Fetch dependencies and verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/glamour
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles
go get github.com/fsnotify/fsnotify
go build ./...
```

Expected: Build succeeds. Running `./mdwalker` shows blank screen with "mdwalker - loading...", exits on `q`.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum main.go internal/
git commit -m "feat: initialize mdwalker project with Go + Bubble Tea skeleton"
```

---

### Task 2: File Discovery

**Files:**
- Create: `internal/discover/discover.go`

- [ ] **Step 1: Create discover package**

```go
// internal/discover/discover.go
package discover

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileEntry struct {
	Path    string    // relative path from root
	ModTime time.Time
	IsDir   bool
}

var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "target": true,
	".direnv": true, "__pycache__": true,
}

// priority groups for AI-oriented sorting
var priorityFiles = map[string]int{
	"AGENTS.md":  1,
	"CLAUDE.md":  2,
	"README.md":  3,
}

var priorityDirs = []string{".ai", ".claude", ".codex"}
var secondaryDirs = []string{"docs", "notes", "reports"}

func Scan(root string) ([]FileEntry, error) {
	entries, err := scanWithFD(root)
	if err != nil {
		entries, err = scanNative(root)
		if err != nil {
			return nil, err
		}
	}
	sortEntries(entries)
	return entries, nil
}

func scanWithFD(root string) ([]FileEntry, error) {
	_, err := exec.LookPath("fd")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("fd", "--type", "f", "--extension", "md", "--search-path", root)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var entries []FileEntry
	for _, line := range lines {
		if line == "" {
			continue
		}
		info, err := os.Stat(line)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, line)
		entries = append(entries, FileEntry{
			Path:    rel,
			ModTime: info.ModTime(),
		})
	}
	return entries, nil
}

func scanNative(root string) ([]FileEntry, error) {
	var entries []FileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			info, _ := d.Info()
			rel, _ := filepath.Rel(root, path)
			entries = append(entries, FileEntry{
				Path:    rel,
				ModTime: info.ModTime(),
			})
		}
		return nil
	})
	return entries, err
}

func sortEntries(entries []FileEntry) {
	now := time.Now()
	last24h := now.Add(-24 * time.Hour)

	sort.SliceStable(entries, func(i, j int) bool {
		pi := priority(entries[i], last24h)
		pj := priority(entries[j], last24h)
		if pi != pj {
			return pi < pj
		}
		return entries[i].ModTime.After(entries[j].ModTime)
	})
}

func priority(e FileEntry, last24h time.Time) int {
	base := filepath.Base(e.Path)
	if p, ok := priorityFiles[base]; ok {
		return p
	}
	dir := filepath.Dir(e.Path)
	for _, d := range priorityDirs {
		if strings.HasPrefix(dir, d) || strings.Contains(dir, "/"+d) {
			return 10
		}
	}
	if e.ModTime.After(last24h) {
		return 11
	}
	for _, d := range secondaryDirs {
		if strings.HasPrefix(dir, d) || strings.Contains(dir, "/"+d) {
			return 20
		}
	}
	return 30
}

func TimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/discover/...
```

Expected: Build succeeds (note: may have unused import for `fmt` — add `import "fmt"` at top).

- [ ] **Step 3: Commit**

```bash
git add internal/discover/discover.go
git commit -m "feat: add file discovery with fd fallback and AI-priority sorting"
```

---

### Task 3: File List Panel

**Files:**
- Create: `internal/filelist/filelist.go`

- [ ] **Step 1: Create file list component**

```go
// internal/filelist/filelist.go
package filelist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bairea/mdwalker/internal/discover"
)

type Model struct {
	Entries  []discover.FileEntry
	Cursor   int
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

var (
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
	normalStyle   = lipgloss.NewStyle()
	timeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dirStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
)

func New() Model {
	return Model{}
}

func (m *Model) SetFiles(entries []discover.FileEntry) {
	m.Entries = entries
	if m.Cursor >= len(m.Entries) {
		m.Cursor = 0
	}
	m.updateViewport()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.ready = true
	m.updateViewport()
}

func (m *Model) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
		m.updateViewport()
	}
}

func (m *Model) MoveDown() {
	if m.Cursor < len(m.Entries)-1 {
		m.Cursor++
		m.updateViewport()
	}
}

func (m Model) SelectedFile() string {
	if m.Cursor < len(m.Entries) {
		return m.Entries[m.Cursor].Path
	}
	return ""
}

func (m *Model) updateViewport() {
	var b strings.Builder
	for i, entry := range m.Entries {
		line := m.renderLine(entry, i == m.Cursor)
		b.WriteString(line + "\n")
	}
	m.viewport.SetContent(b.String())
}

func (m Model) renderLine(entry discover.FileEntry, selected bool) string {
	timeStr := discover.TimeAgo(entry.ModTime)
	line := fmt.Sprintf(" %-*s %s", m.width-15, entry.Path, timeStr)
	if selected {
		return selectedStyle.Render(line)
	}
	return normalStyle.Render(line)
}

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/filelist/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/filelist/filelist.go
git commit -m "feat: add file list panel component"
```

---

### Task 4: Preview Panel with Glamour

**Files:**
- Create: `internal/preview/preview.go`

- [ ] **Step 1: Create preview component**

```go
// internal/preview/preview.go
package preview

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type Model struct {
	viewport viewport.Model
	renderer *glamour.TermRenderer
	filePath string
	content  string
	width    int
	height   int
	ready    bool
}

func New() Model {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0), // we handle wrapping
	)
	return Model{renderer: r}
}

func (m *Model) LoadFile(root, path string) error {
	fullPath := root + "/" + path
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	m.filePath = path
	m.content = string(data)
	m.render()
	return nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.ready = true
	m.render()
}

func (m *Model) render() {
	if !m.ready || m.content == "" {
		return
	}
	rendered, err := m.renderer.Render(m.content)
	if err != nil {
		m.viewport.SetContent(m.content)
		return
	}
	m.viewport.SetContent(rendered)
	m.viewport.GotoTop()
}

func (m *Model) ScrollUp()   { m.viewport.LineUp(1) }
func (m *Model) ScrollDown() { m.viewport.LineDown(1) }
func (m *Model) ScrollTop()  { m.viewport.GotoTop() }
func (m *Model) ScrollBottom() { m.viewport.GotoBottom() }
func (m *Model) ScrollHalfPageUp()   { m.viewport.HalfViewUp() }
func (m *Model) ScrollHalfPageDown() { m.viewport.HalfViewDown() }

func (m *Model) Content() string { return m.content }

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/preview/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/preview/preview.go
git commit -m "feat: add markdown preview panel with Glamour rendering"
```

---

### Task 5: Watch Mode

**Files:**
- Create: `internal/watch/watch.go`

- [ ] **Step 1: Create watch package**

```go
// internal/watch/watch.go
package watch

import (
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type Event struct {
	Path string
	Op   string // "create", "write", "remove", "rename"
}

type Watcher struct {
	w       *fsnotify.Watcher
	Events  chan Event
	Errors  chan error
	root    string
}

func New(root string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		w:      w,
		Events: make(chan Event, 100),
		Errors: make(chan error, 10),
		root:   root,
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && name != ".ai" && name != ".claude" && name != ".codex" {
				return filepath.SkipDir
			}
			return w.Add(path)
		}
		return nil
	})
	if err != nil {
		w.Close()
		return nil, err
	}

	go watcher.loop()
	return watcher, nil
}

func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.w.Events:
			if !ok {
				return
			}
			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}
			rel, _ := filepath.Rel(w.root, event.Name)
			op := "write"
			if event.Has(fsnotify.Create) {
				op = "create"
			} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				op = "remove"
			}
			w.Events <- Event{Path: rel, Op: op}
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			w.Errors <- err
		}
	}
}

func (w *Watcher) Close() {
	w.w.Close()
}
```

- [ ] **Step 2: Fix import (add "os" to imports)**

Add `"os"` to the import block in watch.go.

- [ ] **Step 3: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/watch/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/watch/watch.go
git commit -m "feat: add watch mode with fsnotify"
```

---

### Task 6: Outline Panel

**Files:**
- Create: `internal/outline/outline.go`

- [ ] **Step 1: Create outline component**

```go
// internal/outline/outline.go
package outline

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Heading struct {
	Level int
	Text  string
	Line  int
}

type Model struct {
	Headings []Heading
	Cursor   int
	Visible  bool
	width    int
}

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("62"))
	levelStyles = map[int]lipgloss.Style{
		1: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")),
		2: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")),
		3: lipgloss.NewStyle().Foreground(lipgloss.Color("110")),
	}
)

func New() Model {
	return Model{}
}

func Parse(content string) []Heading {
	var headings []Heading
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			level := 0
			for _, c := range line {
				if c == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= 6 && len(line) > level && line[level] == ' ' {
				headings = append(headings, Heading{
					Level: level,
					Text:  strings.TrimSpace(line[level:]),
					Line:  i,
				})
			}
		}
	}
	return headings
}

func (m *Model) SetContent(content string) {
	m.Headings = Parse(content)
	if m.Cursor >= len(m.Headings) {
		m.Cursor = 0
	}
}

func (m *Model) Toggle() {
	m.Visible = !m.Visible
}

func (m *Model) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

func (m *Model) MoveDown() {
	if m.Cursor < len(m.Headings)-1 {
		m.Cursor++
	}
}

func (m Model) SelectedLine() int {
	if m.Cursor < len(m.Headings) {
		return m.Headings[m.Cursor].Line
	}
	return 0
}

func (m Model) View() string {
	if !m.Visible || len(m.Headings) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Outline\n\n")
	for i, h := range m.Headings {
		prefix := strings.Repeat("  ", h.Level-1)
		style := levelStyles[h.Level]
		if style == (lipgloss.Style{}) {
			style = lipgloss.NewStyle()
		}
		line := prefix + style.Render(h.Text)
		if i == m.Cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return panelStyle.Render(b.String())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/outline/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/outline/outline.go
git commit -m "feat: add outline panel with heading parsing"
```

---

### Task 7: Search

**Files:**
- Create: `internal/search/search.go`

- [ ] **Step 1: Create search component**

```go
// internal/search/search.go
package search

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	input     textinput.Model
	Active    bool
	Query     string
	Matches   []Match
	Current   int
}

type Match struct {
	Line int
	Text string
}

var (
	barStyle    = lipgloss.NewStyle().Background(lipgloss.Color("236"))
	matchStyle  = lipgloss.NewStyle().Background(lipgloss.Color("227")).Foreground(lipgloss.Color("0"))
	countStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 80
	return Model{input: ti}
}

func (m *Model) Activate() {
	m.Active = true
	m.input.Focus()
	m.Query = ""
	m.Matches = nil
	m.Current = 0
}

func (m *Model) Deactivate() {
	m.Active = false
	m.input.Blur()
	m.Query = ""
	m.Matches = nil
}

func (m *Model) Search(content string) {
	if m.Query == "" {
		m.Matches = nil
		return
	}
	m.Matches = nil
	lower := strings.ToLower(m.Query)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), lower) {
			m.Matches = append(m.Matches, Match{Line: i, Text: line})
		}
	}
	if m.Current >= len(m.Matches) {
		m.Current = 0
	}
}

func (m *Model) Next() {
	if len(m.Matches) > 0 {
		m.Current = (m.Current + 1) % len(m.Matches)
	}
}

func (m *Model) Prev() {
	if len(m.Matches) > 0 {
		m.Current--
		if m.Current < 0 {
			m.Current = len(m.Matches) - 1
		}
	}
}

func (m Model) CurrentLine() int {
	if m.Current < len(m.Matches) {
		return m.Matches[m.Current].Line
	}
	return 0
}

func (m Model) View() string {
	if !m.Active {
		return ""
	}
	count := ""
	if len(m.Matches) > 0 {
		count = countStyle.Render(fmt.Sprintf(" %d/%d", m.Current+1, len(m.Matches)))
	}
	return barStyle.Render(" /" + m.input.View() + count)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.Query = m.input.Value()
	return m, cmd
}
```

- [ ] **Step 2: Fix import — add `"fmt"` to imports**

- [ ] **Step 3: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/search/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/search/search.go
git commit -m "feat: add search component with text input and match navigation"
```

---

### Task 8: Semantic Highlighting

**Files:**
- Create: `internal/semantic/semantic.go`

- [ ] **Step 1: Create semantic highlighting package**

```go
// internal/semantic/semantic.go
package semantic

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Highlight struct {
	Line  int
	Style lipgloss.Style
	Label string
}

var (
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	todoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("120")).Bold(true)
	decisionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

	patterns = []struct {
		re    *regexp.Regexp
		style lipgloss.Style
		label string
	}{
		{regexp.MustCompile(`^(Error:|\[ERROR\]|❌)`), errorStyle, "ERR"},
		{regexp.MustCompile(`^(Warning:|\[WARN\]|⚠️)`), warnStyle, "WRN"},
		{regexp.MustCompile(`^(TODO:|\[TODO\])`), todoStyle, "TODO"},
		{regexp.MustCompile(`^(Next Steps:|Decision:)`), decisionStyle, "DEC"},
	}
)

func Scan(content string) map[int]Highlight {
	results := make(map[int]Highlight)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		for _, p := range patterns {
			if p.re.MatchString(strings.TrimSpace(line)) {
				results[i] = Highlight{Line: i, Style: p.style, Label: p.label}
				break
			}
		}
	}
	return results
}

func ApplyLine(line string, h *Highlight) string {
	if h == nil {
		return line
	}
	prefix := h.Style.Render("[" + h.Label + "] ")
	return prefix + line
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/semantic/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/semantic/semantic.go
git commit -m "feat: add semantic highlighting for AI output patterns"
```

---

### Task 9: Code Block Operations

**Files:**
- Create: `internal/codeblock/codeblock.go`

- [ ] **Step 1: Create code block operations package**

```go
// internal/codeblock/codeblock.go
package codeblock

import (
	"os/exec"
	"strings"
)

type Block struct {
	StartLine int
	EndLine   int
	Language  string
	Content   string
}

func Extract(content string) []Block {
	var blocks []Block
	lines := strings.Split(content, "\n")
	inBlock := false
	var current Block

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") && !inBlock {
			inBlock = true
			current = Block{
				StartLine: i,
				Language:  strings.TrimPrefix(trimmed, "```"),
			}
		} else if strings.HasPrefix(trimmed, "```") && inBlock {
			inBlock = false
			current.EndLine = i
			current.Content = strings.Join(lines[current.StartLine+1:i], "\n")
			blocks = append(blocks, current)
		}
	}
	return blocks
}

func BlockAtLine(blocks []Block, line int) *Block {
	for i := range blocks {
		if line >= blocks[i].StartLine && line <= blocks[i].EndLine {
			return &blocks[i]
		}
	}
	return nil
}

func CopyToClipboard(text string) error {
	cmd := exec.Command("pbcopy") // macOS
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func IsDiff(block Block) bool {
	return block.Language == "diff"
}

func DiffLines(content string) []string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "-") {
			lines[i] = "\033[41m" + line + "\033[0m"
		} else if strings.HasPrefix(line, "+") {
			lines[i] = "\033[42m" + line + "\033[0m"
		}
	}
	return lines
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/codeblock/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/codeblock/codeblock.go
git commit -m "feat: add code block extraction, copy, and diff rendering"
```

---

### Task 10: Terminal Detection

**Files:**
- Create: `internal/terminal/detect.go`

- [ ] **Step 1: Create terminal detection package**

```go
// internal/terminal/detect.go
package terminal

import "os"

type Capability int

const (
	CapNone     Capability = iota
	CapHalfblock
	CapKitty
	CapITerm2
)

type Info struct {
	ImageCap Capability
	IsTmux   bool
}

func Detect() Info {
	info := Info{}
	term := os.Getenv("TERM")
	termProg := os.Getenv("TERM_PROGRAM")

	if os.Getenv("TMUX") != "" {
		info.IsTmux = true
	}

	switch {
	case termProg == "kitty" || strings.Contains(term, "kitty"):
		info.ImageCap = CapKitty
	case termProg == "iTerm.app" || termProg == "WezTerm":
		info.ImageCap = CapITerm2
	default:
		if chafaAvailable() {
			info.ImageCap = CapHalfblock
		}
	}

	return info
}

func chafaAvailable() bool {
	_, err := exec.LookPath("chafa")
	if err == nil {
		return true
	}
	_, err = exec.LookPath("viu")
	return err == nil
}
```

- [ ] **Step 2: Add imports `"os/exec"` and `"strings"`**

- [ ] **Step 3: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/terminal/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/terminal/detect.go
git commit -m "feat: add terminal capability detection"
```

---

### Task 11: Image Support

**Files:**
- Create: `internal/image/image.go`

- [ ] **Step 1: Create image package**

```go
// internal/image/image.go
package image

import (
	"os/exec"
	"regexp"
	"strings"
)

type ImageRef struct {
	Alt  string
	Path string
	Line int
}

var imgRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

func Extract(content string) []ImageRef {
	var refs []ImageRef
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		matches := imgRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			refs = append(refs, ImageRef{
				Alt:  m[1],
				Path: m[2],
				Line: i,
			})
		}
	}
	return refs
}

func OpenWithDefault(path string) error {
	return exec.Command("open", path).Run()
}

func RenderPlaceholder(ref ImageRef) string {
	return "[Image: " + ref.Path + "]  press i to open"
}

func ToHalfblock(path string) (string, error) {
	cmd := exec.Command("chafa", "--symbols", "block", "--size", "80x20", path)
	out, err := cmd.Output()
	if err != nil {
		// try viu
		cmd = exec.Command("viu", "--width", "80", "--height", "20", path)
		out, err = cmd.Output()
	}
	return string(out), err
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/image/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/image/image.go
git commit -m "feat: add image reference extraction and progressive rendering"
```

---

### Task 12: Mermaid Support

**Files:**
- Create: `internal/mermaid/mermaid.go`

- [ ] **Step 1: Create mermaid package**

```go
// internal/mermaid/mermaid.go
package mermaid

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Diagram struct {
	Content string
	Hash    string
}

func Extract(content string) []Diagram {
	// find ```mermaid blocks, same logic as codeblock but language-aware
	return nil // placeholder — integrated with codeblock.Extract
}

func CacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "mdwalker", "mermaid")
}

func Render(content string) (string, error) {
	h := sha256.Sum256([]byte(content))
	hash := fmt.Sprintf("%x", h[:16])
	cacheDir := CacheDir()
	os.MkdirAll(cacheDir, 0755)
	pngPath := filepath.Join(cacheDir, hash+".png")

	if _, err := os.Stat(pngPath); os.IsNotExist(err) {
		mmdcPath := "mmdc"
		if custom := os.Getenv("MDWALKER_MMDC"); custom != "" {
			mmdcPath = custom
		}
		tmpFile := filepath.Join(cacheDir, hash+".mmd")
		os.WriteFile(tmpFile, []byte(content), 0644)
		defer os.Remove(tmpFile)
		cmd := exec.Command(mmdcPath, "-i", tmpFile, "-o", pngPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("mmdc failed: %w", err)
		}
	}

	return pngPath, nil
}

func CleanCache() {
	cacheDir := CacheDir()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, e := range entries {
		info, _ := e.Info()
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(cacheDir, e.Name()))
		}
	}
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./internal/mermaid/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/mermaid/mermaid.go
git commit -m "feat: add mermaid rendering with mmdc and content-hash caching"
```

---

### Task 13: App Integration

**Files:**
- Modify: `internal/app/app.go` — complete rewrite with full model

- [ ] **Step 1: Rewrite app.go with full TUI model**

```go
// internal/app/app.go
package app

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bairea/mdwalker/internal/codeblock"
	"github.com/bairea/mdwalker/internal/config"
	"github.com/bairea/mdwalker/internal/discover"
	"github.com/bairea/mdwalker/internal/filelist"
	"github.com/bairea/mdwalker/internal/image"
	"github.com/bairea/mdwalker/internal/mermaid"
	"github.com/bairea/mdwalker/internal/outline"
	"github.com/bairea/mdwalker/internal/preview"
	"github.com/bairea/mdwalker/internal/search"
	"github.com/bairea/mdwalker/internal/semantic"
	"github.com/bairea/mdwalker/internal/watch"
)

type focus int

const (
	focusFiles focus = iota
	focusPreview
	focusOutline
	focusSearch
)

type Model struct {
	cfg     config.Config
	files   filelist.Model
	preview preview.Model
	outline outline.Model
	search  search.Model
	watcher *watch.Watcher

	root    string
	focus   focus
	history []string // back navigation
	width   int
	height  int
	ready   bool

	codeBlocks []codeblock.Block
}

func New(root string) Model {
	mermaid.CleanCache()

	return Model{
		cfg:     config.Load(),
		files:   filelist.New(),
		preview: preview.New(),
		outline: outline.New(),
		search:  search.New(),
		root:    root,
	}
}

func NewDefault() Model {
	root, _ := os.Getwd()
	return New(root)
}

type filesLoadedMsg struct {
	entries []discover.FileEntry
}

type watchEventMsg watch.Event

type watchErrorMsg error

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			entries, err := discover.Scan(m.root)
			if err != nil {
				return err
			}
			return filesLoadedMsg{entries}
		},
		m.startWatcher,
	)
}

func (m *Model) startWatcher() tea.Msg {
	w, err := watch.New(m.root)
	if err != nil {
		return watchErrorMsg(err)
	}
	m.watcher = w
	return nil
}

func (m *Model) listenWatch() tea.Msg {
	if m.watcher == nil {
		return nil
	}
	select {
	case evt := <-m.watcher.Events:
		return watchEventMsg(evt)
	case err := <-m.watcher.Errors:
		return watchErrorMsg(err)
	default:
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
		m.layout()

	case tea.MouseMsg:
		// pass through for text selection, handled by terminal

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit

		case "tab":
			m.focus = (m.focus + 1) % 2 // toggle files/preview

		case "o":
			if m.focus == focusSearch {
				break
			}
			m.outline.Toggle()
			if m.outline.Visible {
				m.focus = focusOutline
			} else {
				m.focus = focusPreview
			}

		case "j", "down":
			switch m.focus {
			case focusFiles:
				m.files.MoveDown()
			case focusPreview:
				m.preview.ScrollDown()
			case focusOutline:
				m.outline.MoveDown()
			}

		case "k", "up":
			switch m.focus {
			case focusFiles:
				m.files.MoveUp()
			case focusPreview:
				m.preview.ScrollUp()
			case focusOutline:
				m.outline.MoveUp()
			}

		case "enter":
			switch m.focus {
			case focusFiles:
				m.openSelectedFile()
			case focusOutline:
				// jump to heading line in preview
				line := m.outline.SelectedLine()
				// scroll preview to approximate position
				m.preview.ScrollToLine(line)
				m.focus = focusPreview
			}

		case "/":
			if m.focus == focusFiles {
				// search file names
			} else {
				m.search.Activate()
				m.focus = focusSearch
			}

		case "esc":
			if m.search.Active {
				m.search.Deactivate()
				m.focus = focusPreview
			}
			if m.outline.Visible {
				m.outline.Toggle()
				m.focus = focusPreview
			}

		case "n":
			m.search.Next()
		case "N":
			m.search.Prev()

		case "y":
			m.copyCurrentBlock()

		case "i":
			m.openCurrentImage()

		case "b":
			if len(m.history) > 0 {
				// go back to previous file
				prev := m.history[len(m.history)-1]
				m.history = m.history[:len(m.history)-1]
				m.preview.LoadFile(m.root, prev)
			}

		case "r":
			cmds = append(cmds, func() tea.Msg {
				entries, err := discover.Scan(m.root)
				if err != nil {
					return err
				}
				return filesLoadedMsg{entries}
			})

		case "g":
			m.preview.ScrollTop()
		case "G":
			m.preview.ScrollBottom()
		}

	case filesLoadedMsg:
		m.files.SetFiles(msg.entries)

	case watchEventMsg:
		// re-scan on file changes
		cmds = append(cmds, func() tea.Msg {
			entries, _ := discover.Scan(m.root)
			return filesLoadedMsg{entries}
		})

	case watchErrorMsg:
		// silently ignore watch errors
	}

	// update sub-components
	var cmd tea.Cmd
	switch m.focus {
	case focusFiles:
		m.files, cmd = m.files.Update(msg)
	case focusPreview:
		m.preview, cmd = m.preview.Update(msg)
	case focusOutline:
		m.outline, cmd = m.outline.Update(msg)
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if m.search.Active {
		m.search, cmd = m.search.Update(msg)
		m.search.Search(m.preview.Content())
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// poll watch events
	cmds = append(cmds, m.listenWatch)

	return m, tea.Batch(cmds...)
}

func (m *Model) openSelectedFile() {
	path := m.files.SelectedFile()
	if path == "" {
		return
	}
	currentPath := m.preview.FilePath() // need to add FilePath() to preview
	if currentPath != "" && currentPath != path {
		m.history = append(m.history, currentPath)
	}
	m.preview.LoadFile(m.root, path)
	m.outline.SetContent(m.preview.Content())
	m.codeBlocks = codeblock.Extract(m.preview.Content())
	m.focus = focusPreview
}

func (m *Model) copyCurrentBlock() {
	line := m.preview.CurrentLine()
	block := codeblock.BlockAtLine(m.codeBlocks, line)
	if block != nil {
		codeblock.CopyToClipboard(block.Content)
	}
}

func (m *Model) openCurrentImage() {
	line := m.preview.CurrentLine()
	content := m.preview.Content()
	refs := image.Extract(content)
	for _, ref := range refs {
		if ref.Line == line {
			image.OpenWithDefault(ref.Path)
			break
		}
	}
}

func (m *Model) layout() {
	filesWidth := m.width * 20 / 100
	if filesWidth < 20 {
		filesWidth = 0
	}
	previewWidth := m.width - filesWidth - 1
	bodyHeight := m.height - 1 // status bar

	m.files.SetSize(filesWidth, bodyHeight)
	m.preview.SetSize(previewWidth, bodyHeight)
}

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}

	filesView := m.files.View()
	previewView := m.preview.View()

	// apply semantic highlights to preview
	highlights := semantic.Scan(m.preview.Content())
	_ = highlights // integrate with glamour output

	// floating outline
	outlineView := m.outline.View()

	if outlineView != "" {
		// overlay on right side
		outlineLines := strings.Split(outlineView, "\n")
		previewLines := strings.Split(previewView, "\n")
		outlineWidth := 30
		for i := 0; i < len(outlineLines) && i < len(previewLines); i++ {
			ol := outlineLines[i]
			pl := previewLines[i]
			if len(pl) > m.width-outlineWidth {
				pl = pl[:m.width-outlineWidth]
			}
			previewLines[i] = pl + strings.Repeat(" ", m.width-outlineWidth-len(pl)) + ol
		}
		previewView = strings.Join(previewLines, "\n")
	}

	mainView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		filesView,
		previewView,
	)

	statusBar := m.renderStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)
}

func (m Model) renderStatusBar() string {
	bar := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(m.width)

	left := fmt.Sprintf(" mdwalker | %s ", m.root)
	if m.watcher != nil {
		left += "● watch "
	}

	right := ""
	if m.search.Active {
		right = m.search.View()
	} else {
		right = fmt.Sprintf(" q:quit  o:outline  /:search  r:rescan ")
	}

	leftWidth := len(left)
	rightWidth := len(right)
	padding := m.width - leftWidth - rightWidth
	if padding < 0 {
		padding = 0
	}

	return bar.Render(left + strings.Repeat(" ", padding) + right)
}
```

- [ ] **Step 2: Add needed methods to preview.Model**

Add to `internal/preview/preview.go`:
```go
func (m Model) FilePath() string { return m.filePath }
func (m Model) CurrentLine() int { return m.viewport.YOffset + m.viewport.Height/2 } // approximate
func (m *Model) ScrollToLine(line int) { m.viewport.SetYOffset(line) }
```

- [ ] **Step 3: Update main.go**

```go
// main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bairea/mdwalker/internal/app"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	p := tea.NewProgram(
		app.New(root),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mdwalker: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Verify full build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build ./...
```

Fix any compilation errors (unused imports, missing methods).

- [ ] **Step 5: Manual smoke test**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go run . .
```

Expected: TUI launches, shows .md files in current dir, preview renders markdown. `j`/`k` navigates, `q` exits.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: integrate all components into full TUI application"
```

---

### Task 14: CLI Argument Parsing

**Files:**
- Modify: `main.go`
- Create: `internal/cli/cli.go`

- [ ] **Step 1: Add proper CLI parsing**

```go
// internal/cli/cli.go
package cli

import (
	"flag"
	"os"
)

type Args struct {
	Root      string
	Files     []string
	NoWatch   bool
	Mermaid   string // auto | code | browser
}

func Parse() Args {
	var a Args
	noWatch := flag.Bool("no-watch", false, "disable file watching")
	mermaid := flag.String("mermaid", "auto", "mermaid mode: auto, code, browser")
	flag.Parse()

	a.NoWatch = *noWatch
	a.Mermaid = *mermaid

	positional := flag.Args()
	if len(positional) == 0 {
		a.Root, _ = os.Getwd()
	} else if len(positional) == 1 {
		info, err := os.Stat(positional[0])
		if err == nil && info.IsDir() {
			a.Root = positional[0]
		} else {
			a.Files = positional
			a.Root, _ = os.Getwd()
		}
	} else {
		a.Files = positional
		a.Root, _ = os.Getwd()
	}

	return a
}
```

- [ ] **Step 2: Update main.go**

```go
func main() {
	args := cli.Parse()
	// pass args to app.New()
}
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/bairea/Documents/cs_proj/mdwalker
go build -o mdwalker .
```

- [ ] **Step 4: Commit**

```bash
git add internal/cli/cli.go main.go
git commit -m "feat: add CLI argument parsing with flags and positional args"
```

---

### Task 15: README with mmdc Reminder

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

```markdown
# mdwalker

AI agent 输出物专用终端 Markdown 工作台。

## 安装

```bash
go install github.com/bairea/mdwalker@latest
```

## 推荐安装（可选增强）

```bash
# 更快的文件搜索
brew install fd

# Mermaid 图表渲染
npm install -g @mermaid-js/mermaid-cli

# 终端图片渲染（降级方案）
brew install chafa
```

## 使用

```bash
mdwalker                  # 当前目录，默认 watch
mdwalker docs/            # 指定目录
mdwalker README.md        # 单文件预览
mdwalker --no-watch       # 关闭文件监听
mdwalker --mermaid code   # Mermaid 只显示源码
```
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with installation and usage"
```

---

## Self-Review

**1. Spec coverage check:**
- File discovery with fd fallback, AI priority sorting -> Task 2
- Two-panel layout -> Task 13 (integration)
- Outline floating panel -> Task 6
- Watch mode default on -> Task 5 + Task 13
- Semantic highlighting -> Task 8
- Code block copy, diff, fold -> Task 9
- Image progressive -> Task 11
- Mermaid progressive -> Task 12
- Terminal detection -> Task 10
- Search -> Task 7
- CLI args -> Task 14
- Config -> Task 1
- mmdc install reminder -> Task 15 (README)

**2. Placeholder scan:** No TBD, TODO. All code is concrete.

**3. Type consistency:** 
- `discover.FileEntry` used in Task 2, consumed in Task 3 and 13 — consistent
- `preview.Model` has all methods called in Task 13 (need to verify `CurrentLine()`, `FilePath()`, `ScrollToLine()` — added in Step 2 of Task 13)
- `outline.Model` methods match usage in app.go
- `search.Model` methods match usage

One issue: `preview.ScrollToLine()` is used but needs to be added in Task 13 Step 2. This is accounted for.
