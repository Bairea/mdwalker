package app

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bairea/mdwalker/internal/config"
	"github.com/bairea/mdwalker/internal/discover"
	"github.com/bairea/mdwalker/internal/filelist"
	"github.com/bairea/mdwalker/internal/markdown"
	"github.com/bairea/mdwalker/internal/outline"
	"github.com/bairea/mdwalker/internal/preview"
	"github.com/bairea/mdwalker/internal/search"
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
	wl      config.WhitelistConfig
	files   filelist.Model
	preview preview.Model
	outline outline.Model
	search  search.Model
	watcher *fileWatcher

	root         string
	focus        focus
	history      []string
	filesWidth   int
	outlineWidth int
	width        int
	height       int
	ready        bool
	codeBlocks   []markdown.Block
}

type filesLoadedMsg struct {
	entries []discover.FileEntry
}

type watchEventMsg fileEvent

type watchErrorMsg error

var (
	activePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	inactivePaneStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
)

func New(root string) *Model {
	cfg := config.Load()
	wl := config.LoadWhitelist()
	files := filelist.New()
	files.ShowTime = cfg.ShowTime

	return &Model{
		cfg:     cfg,
		wl:      wl,
		files:   files,
		preview: preview.New(),
		outline: outline.New(),
		search:  search.New(),
		root:    root,
	}
}

func NewDefault() *Model {
	root, _ := os.Getwd()
	return New(root)
}

func (m *Model) SetShowTime(v bool) {
	m.files.ShowTime = v
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			entries, err := discover.Scan(m.root, &m.wl)
			if err != nil {
				return err
			}
			return filesLoadedMsg{entries}
		},
		m.startWatcher,
	)
}

func (m *Model) startWatcher() tea.Msg {
	w, err := newFileWatcher(m.root, m.wl.Unignore.DotDirs)
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	skipSearchInput := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
		m.layout()

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if m.filesWidth > 0 && msg.X < m.filesWidth {
				m.focus = focusFiles
				if m.files.SelectVisibleRow(msg.Y - 1) {
					m.openSelectedFile()
				}
			} else if m.outline.Visible && m.outlineWidth > 0 && msg.X >= m.width-m.outlineWidth {
				m.focus = focusOutline
				if m.outline.SelectVisibleRow(msg.Y) {
					m.preview.ScrollToLine(m.outline.SelectedLine())
				}
			} else {
				m.focus = focusPreview
				m.preview.SetCursorFromVisibleRow(msg.Y - 1)
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.watcher != nil {
				m.watcher.close()
			}
			return m, tea.Quit

		case "tab":
			if m.search.Active {
				m.search.ToggleMode()
				m.search.UpdateSearch(m.root, m.files.Entries, m.preview.Content())
				skipSearchInput = true
			} else if m.outline.Visible {
				m.focus = (m.focus + 1) % 3
			} else {
				m.focus = (m.focus + 1) % 2
			}

		case "h", "left":
			m.moveFocusLeft()

		case "l", "right":
			m.moveFocusRight()

		case " ":
			if m.focus == focusPreview && !m.search.Active {
				line := m.preview.CurrentLine()
				m.preview.ToggleFold(line)
			}

		case "t":
			if m.focus == focusFiles {
				m.files.ToggleTreeMode()
			}

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
			m.layout()

		case "j", "down":
			if m.search.Active {
				m.search.Next()
				skipSearchInput = true
				break
			}
			switch m.focus {
			case focusFiles:
				m.files.MoveDown()
				m.previewSelectedFile()
			case focusPreview:
				m.preview.ScrollDown()
			case focusOutline:
				m.outline.MoveDown()
			}

		case "k", "up":
			if m.search.Active {
				m.search.Prev()
				skipSearchInput = true
				break
			}
			switch m.focus {
			case focusFiles:
				m.files.MoveUp()
				m.previewSelectedFile()
			case focusPreview:
				m.preview.ScrollUp()
			case focusOutline:
				m.outline.MoveUp()
			}

		case "enter":
			if m.search.Active && m.search.Mode == search.ModeFileName {
				idx := m.search.CurrentFileIndex()
				if idx >= 0 && idx < len(m.files.Entries) {
					m.openFile(m.files.Entries[idx].Path)
					m.search.Deactivate()
					m.focus = focusPreview
				}
				skipSearchInput = true
				break
			}
			if m.search.Active && m.search.Mode == search.ModeContent {
				if len(m.search.Matches) > 0 {
					m.preview.ScrollToLine(m.search.CurrentLine())
					m.search.Deactivate()
					m.focus = focusPreview
				}
				skipSearchInput = true
				break
			}
			if m.search.Active && m.search.Mode == search.ModeAllContent {
				match := m.search.CurrentAllMatch()
				if match.Path != "" {
					m.openFile(match.Path)
					m.preview.ScrollToLine(match.Line)
					m.search.Deactivate()
					m.focus = focusPreview
				}
				skipSearchInput = true
				break
			}
			switch m.focus {
			case focusFiles:
				m.openSelectedFile()
			case focusOutline:
				line := m.outline.SelectedLine()
				m.preview.ScrollToLine(line)
			}

		case "/":
			if m.focus == focusFiles {
				m.search.Activate(search.ModeFileName)
			} else {
				m.search.Activate(search.ModeContent)
			}
			m.focus = focusSearch
			skipSearchInput = true

		case "esc":
			if m.search.Active {
				m.search.Deactivate()
				m.focus = focusPreview
				skipSearchInput = true
			}
			if m.outline.Visible {
				m.outline.Toggle()
				m.focus = focusPreview
				m.layout()
			}

		case "n":
			if m.search.Active && m.search.Mode == search.ModeFileName {
				m.search.Next()
				idx := m.search.CurrentFileIndex()
				if idx >= 0 && idx < len(m.files.Entries) {
					m.files.Cursor = idx
					m.files.UpdateViewport()
				}
			} else if m.search.Active {
				m.search.Next()
				line := m.search.CurrentLine()
				m.preview.ScrollToLine(line)
			}

		case "N":
			if m.search.Active && m.search.Mode == search.ModeFileName {
				m.search.Prev()
				idx := m.search.CurrentFileIndex()
				if idx >= 0 && idx < len(m.files.Entries) {
					m.files.Cursor = idx
					m.files.UpdateViewport()
				}
			} else if m.search.Active {
				m.search.Prev()
				line := m.search.CurrentLine()
				m.preview.ScrollToLine(line)
			}

		case "y":
			m.copyCurrentBlock()

		case "b":
			if len(m.history) > 0 {
				prev := m.history[len(m.history)-1]
				m.history = m.history[:len(m.history)-1]
				m.preview.LoadFile(m.root, prev)
				m.outline.SetContent(m.preview.Content())
				m.codeBlocks = markdown.ExtractBlocks(m.preview.Content())
				m.focus = focusPreview
			}

		case "r":
			cmds = append(cmds, func() tea.Msg {
				entries, err := discover.Scan(m.root, &m.wl)
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
		cmds = append(cmds, func() tea.Msg {
			entries, _ := discover.Scan(m.root, &m.wl)
			return filesLoadedMsg{entries}
		})

	case watchErrorMsg:
		// silently ignore watch errors
	}

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

	if m.search.Active && !skipSearchInput {
		m.search, cmd = m.search.Update(msg)
		m.search.UpdateSearch(m.root, m.files.Entries, m.preview.Content())
		if m.search.Mode == search.ModeContent && len(m.search.Matches) > 0 {
			m.preview.ScrollToLine(m.search.CurrentLine())
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	cmds = append(cmds, m.listenWatch)

	return m, tea.Batch(cmds...)
}

func (m *Model) openSelectedFile() {
	path := m.files.SelectedFile()
	if path == "" {
		return
	}
	m.loadFile(path, true)
	m.focus = focusPreview
}

func (m *Model) openFile(path string) {
	m.loadFile(path, true)
}

func (m *Model) previewSelectedFile() {
	path := m.files.SelectedFile()
	if path == "" {
		return
	}
	m.loadFile(path, false)
}

func (m *Model) loadFile(path string, recordHistory bool) {
	currentPath := m.preview.FilePath()
	if recordHistory && currentPath != "" && currentPath != path {
		m.history = append(m.history, currentPath)
	}
	m.preview.LoadFile(m.root, path)
	m.outline.SetContent(m.preview.Content())
	m.codeBlocks = markdown.ExtractBlocks(m.preview.Content())
}

func (m *Model) copyCurrentBlock() {
	line := m.preview.CurrentLine()
	block := markdown.BlockAtLine(m.codeBlocks, line)
	if block != nil {
		markdown.CopyToClipboard(block.Content)
	}
}

func (m *Model) layout() {
	filesWidth := m.width * 20 / 100
	if filesWidth < 20 {
		filesWidth = 0
	}
	m.filesWidth = filesWidth
	outlineWidth := 0
	if m.outline.Visible {
		outlineWidth = m.width * 25 / 100
		if outlineWidth < 24 {
			outlineWidth = 24
		}
		if outlineWidth > 36 {
			outlineWidth = 36
		}
		if m.width-filesWidth-outlineWidth < 30 {
			outlineWidth = 0
		}
	}
	m.outlineWidth = outlineWidth

	previewWidth := m.width - filesWidth - outlineWidth
	bodyHeight := m.height - 1

	m.files.SetSize(innerPaneWidth(filesWidth), innerPaneHeight(bodyHeight))
	m.preview.SetSize(innerPaneWidth(previewWidth), innerPaneHeight(bodyHeight))
	m.search.SetSize(m.width, bodyHeight)
}

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}

	filesView := m.files.View()
	previewView := m.preview.View()
	outlineView := m.outline.View()
	if m.filesWidth > 0 {
		filesView = renderPane(filesView, m.filesWidth, m.height-1, m.focus == focusFiles)
	}
	previewWidth := m.width - m.filesWidth - m.outlineWidth
	if previewWidth > 0 {
		previewView = renderPane(previewView, previewWidth, m.height-1, m.focus == focusPreview)
	}

	var mainView string
	if m.filesWidth > 0 && m.outlineWidth > 0 && outlineView != "" {
		mainView = lipgloss.JoinHorizontal(
			lipgloss.Top,
			filesView,
			previewView,
			outlineView,
		)
	} else if m.filesWidth > 0 {
		mainView = lipgloss.JoinHorizontal(
			lipgloss.Top,
			filesView,
			previewView,
		)
	} else if m.outlineWidth > 0 && outlineView != "" {
		mainView = lipgloss.JoinHorizontal(
			lipgloss.Top,
			previewView,
			outlineView,
		)
	} else {
		mainView = previewView
	}

	if m.search.Active {
		mainView = overlayCentered(mainView, m.search.View(), m.width, m.height-1)
	}

	statusBar := m.renderStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)
}

func (m Model) renderStatusBar() string {
	bar := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(m.width)

	left := fmt.Sprintf(" mdwalker | focus:%s | %s ", m.focusName(), m.root)
	if m.watcher != nil {
		left += "● watch "
	}

	var right string
	if m.search.Active {
		mode := "content"
		if m.search.Mode == search.ModeFileName {
			mode = "files"
		} else if m.search.Mode == search.ModeAllContent {
			mode = "all"
		}
		right = fmt.Sprintf(" search:%s  Enter:open  Tab:switch  Esc:cancel ", mode)
	} else {
		right = " q:quit  o:outline  /:search  r:rescan "
	}

	maxLeft := m.width - lipgloss.Width(right)
	if maxLeft < 0 {
		maxLeft = 0
	}
	left = truncateWidth(left, maxLeft)

	used := lipgloss.Width(left) + lipgloss.Width(right)
	padding := m.width - used
	if padding < 0 {
		padding = 0
	}

	return bar.Render(left + strings.Repeat(" ", padding) + right)
}

func (m Model) focusName() string {
	switch m.focus {
	case focusFiles:
		return "files"
	case focusPreview:
		return "preview"
	case focusOutline:
		return "outline"
	case focusSearch:
		return "search"
	default:
		return "unknown"
	}
}

func truncateWidth(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 3 {
		return strings.Repeat(".", max)
	}

	var b strings.Builder
	for _, r := range s {
		next := b.String() + string(r)
		if lipgloss.Width(next)+3 > max {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "..."
}

func overlayCentered(base, panel string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	for len(baseLines) < height {
		baseLines = append(baseLines, strings.Repeat(" ", width))
	}
	panelLines := strings.Split(panel, "\n")
	top := (height - len(panelLines)) / 2
	if top < 0 {
		top = 0
	}
	for i, line := range panelLines {
		target := top + i
		if target >= len(baseLines) {
			break
		}
		lineWidth := lipgloss.Width(line)
		left := (width - lineWidth) / 2
		if left < 0 {
			left = 0
		}
		right := width - left - lineWidth
		if right < 0 {
			right = 0
		}
		baseLines[target] = strings.Repeat(" ", left) + line + strings.Repeat(" ", right)
	}
	return strings.Join(baseLines, "\n")
}

func renderPane(content string, width, height int, focused bool) string {
	if width <= 1 || height <= 1 {
		return ""
	}
	innerWidth := innerPaneWidth(width)
	innerHeight := innerPaneHeight(height)
	content = fitBlock(content, innerWidth, innerHeight)
	style := inactivePaneStyle
	if focused {
		style = activePaneStyle
	}
	return style.Width(innerWidth).Render(content)
}

func innerPaneWidth(width int) int {
	if width <= 2 {
		return 0
	}
	return width - 2
}

func innerPaneHeight(height int) int {
	if height <= 2 {
		return 0
	}
	return height - 2
}

func fitBlock(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = padLine(truncateWidth(line, width), width)
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func padLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func (m *Model) moveFocusLeft() {
	if m.outline.Visible {
		switch m.focus {
		case focusPreview:
			m.focus = focusOutline
		case focusOutline:
			m.focus = focusFiles
		}
	} else {
		if m.focus == focusPreview {
			m.focus = focusFiles
		}
	}
}

func (m *Model) moveFocusRight() {
	if m.outline.Visible {
		switch m.focus {
		case focusFiles:
			m.focus = focusOutline
		case focusOutline:
			m.focus = focusPreview
		}
	} else {
		if m.focus == focusFiles {
			m.focus = focusPreview
		}
	}
}
