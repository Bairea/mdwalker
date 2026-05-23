package filelist

import (
	"fmt"
	"path/filepath"
	"sort"
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
	TreeMode bool
}

var (
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
	normalStyle   = lipgloss.NewStyle()
	dimTimeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("60"))
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
	m.UpdateViewport()
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.ready = true
	m.UpdateViewport()
}

func (m *Model) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
		m.UpdateViewport()
	}
}

func (m *Model) MoveDown() {
	if m.Cursor < len(m.Entries)-1 {
		m.Cursor++
		m.UpdateViewport()
	}
}

func (m Model) SelectedFile() string {
	if m.Cursor < len(m.Entries) {
		return m.Entries[m.Cursor].Path
	}
	return ""
}

func (m *Model) ToggleTreeMode() {
	m.TreeMode = !m.TreeMode
	m.UpdateViewport()
}

func (m *Model) UpdateViewport() {
	var content string
	if m.TreeMode {
		content = m.buildTreeView()
	} else {
		content = m.buildFlatView()
	}
	m.viewport.SetContent(content)
}

func (m Model) buildFlatView() string {
	var b strings.Builder
	for i, entry := range m.Entries {
		line := m.renderLine(entry, i == m.Cursor)
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m Model) renderLine(entry discover.FileEntry, selected bool) string {
	timeStr := dimTimeStyle.Render(discover.TimeAgo(entry.ModTime))
	availWidth := m.width - 15
	if availWidth < 10 {
		availWidth = 10
	}
	path := entry.Path
	if len(path) > availWidth {
		path = path[:availWidth]
	}
	line := fmt.Sprintf(" %-*s %s", availWidth, path, timeStr)
	if selected {
		return selectedStyle.Render(line)
	}
	return normalStyle.Render(line)
}

func (m Model) buildTreeView() string {
	groups := make(map[string][]discover.FileEntry)
	var dirs []string
	for _, e := range m.Entries {
		dir := filepath.Dir(e.Path)
		if _, ok := groups[dir]; !ok {
			dirs = append(dirs, dir)
		}
		groups[dir] = append(groups[dir], e)
	}
	sort.Strings(dirs)

	lineIdx := 0
	var b strings.Builder
	for di, dir := range dirs {
		isLastDir := di == len(dirs)-1
		if dir != "." {
			dirPrefix := "├─ "
			if isLastDir {
				dirPrefix = "└─ "
			}
			line := dirPrefix + dirStyle.Render(dir+"/")
			if lineIdx == m.Cursor {
				line = selectedStyle.Render(line)
			}
			b.WriteString(line + "\n")
			lineIdx++
		}
		entries := groups[dir]
		for ei, e := range entries {
			isLastEntry := isLastDir && ei == len(entries)-1
			var prefix string
			if dir == "." {
				if isLastEntry {
					prefix = "└─ "
				} else {
					prefix = "├─ "
				}
			} else {
				if isLastEntry {
					prefix = "  └─ "
				} else {
					prefix = "  ├─ "
				}
			}
			name := filepath.Base(e.Path)
			timeStr := dimTimeStyle.Render(" " + discover.TimeAgo(e.ModTime))
			line := prefix + name + timeStr
			if lineIdx == m.Cursor {
				line = selectedStyle.Render(line)
			}
			b.WriteString(line + "\n")
			lineIdx++
		}
	}
	return b.String()
}

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
