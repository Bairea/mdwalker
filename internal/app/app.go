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

	root       string
	focus      focus
	history    []string
	filesWidth int
	width      int
	height     int
	ready      bool
	codeBlocks []codeblock.Block
}

type filesLoadedMsg struct {
	entries []discover.FileEntry
}

type watchEventMsg watch.Event

type watchErrorMsg error

func New(root string) *Model {
	mermaid.CleanCache()

	return &Model{
		cfg:     config.Load(),
		files:   filelist.New(),
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

func (m *Model) Init() tea.Cmd {
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// pass through for text selection

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit

		case "tab":
			if m.outline.Visible {
				m.focus = (m.focus + 1) % 3
			} else {
				m.focus = (m.focus + 1) % 2
			}

		case "h":
			if m.focus != focusFiles {
				m.focus = focusFiles
			}

		case "l":
			if m.focus != focusPreview && !m.outline.Visible {
				m.focus = focusPreview
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
				line := m.outline.SelectedLine()
				m.preview.ScrollToLine(line)
				m.focus = focusPreview
			}

		case "/":
			m.search.Activate()
			m.focus = focusSearch

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
			if m.search.Active {
				m.search.Next()
				line := m.search.CurrentLine()
				m.preview.ScrollToLine(line)
			}

		case "N":
			if m.search.Active {
				m.search.Prev()
				line := m.search.CurrentLine()
				m.preview.ScrollToLine(line)
			}

		case "y":
			m.copyCurrentBlock()

		case "i":
			m.openCurrentImage()

		case "b":
			if len(m.history) > 0 {
				prev := m.history[len(m.history)-1]
				m.history = m.history[:len(m.history)-1]
				m.preview.LoadFile(m.root, prev)
				m.outline.SetContent(m.preview.Content())
				m.codeBlocks = codeblock.Extract(m.preview.Content())
				m.focus = focusPreview
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
		cmds = append(cmds, func() tea.Msg {
			entries, _ := discover.Scan(m.root)
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

	if m.search.Active {
		m.search, cmd = m.search.Update(msg)
		m.search.Search(m.preview.Content())
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
	currentPath := m.preview.FilePath()
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
	m.filesWidth = filesWidth
	previewWidth := m.width - filesWidth
	if filesWidth > 0 {
		previewWidth--
	}
	bodyHeight := m.height - 1

	m.files.SetSize(filesWidth, bodyHeight)
	m.preview.SetSize(previewWidth, bodyHeight)
}

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}

	filesView := m.files.View()
	previewView := m.preview.View()

	_ = semantic.Scan(m.preview.Content())

	outlineView := m.outline.View()
	if outlineView != "" {
		outlineWidth := lipgloss.Width(outlineView)
		previewLines := strings.Split(previewView, "\n")
		for i := range previewLines {
			if i == 0 {
				if len(previewLines[i])+outlineWidth > m.width && m.width > outlineWidth {
					previewLines[i] = previewLines[i][:m.width-outlineWidth]
				}
			}
		}
		previewView = strings.Join(previewLines, "\n")
		previewView = lipgloss.JoinHorizontal(lipgloss.Top, previewView, outlineView)
	}

	var mainView string
	if m.filesWidth > 0 {
		mainView = lipgloss.JoinHorizontal(
			lipgloss.Top,
			filesView,
			previewView,
		)
	} else {
		mainView = previewView
	}

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

	var right string
	if m.search.Active {
		right = m.search.View()
	} else {
		right = " q:quit  o:outline  /:search  r:rescan "
	}

	used := lipgloss.Width(left) + lipgloss.Width(right)
	padding := m.width - used
	if padding < 0 {
		padding = 0
	}

	return bar.Render(left + strings.Repeat(" ", padding) + right)
}
