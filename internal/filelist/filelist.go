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
