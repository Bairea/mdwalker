package filelist

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bairea/mdwalker/internal/discover"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Entries   []discover.FileEntry
	Cursor    int
	viewport  viewport.Model
	width     int
	height    int
	ready     bool
	TreeMode  bool
	ShowTime  bool
	rowIndex  []int
	treeOrder []int
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
	if m.TreeMode && len(m.treeOrder) > 0 {
		m.moveTree(-1)
		return
	}
	if m.Cursor > 0 {
		m.Cursor--
		m.UpdateViewport()
	}
}

func (m *Model) MoveDown() {
	if m.TreeMode && len(m.treeOrder) > 0 {
		m.moveTree(1)
		return
	}
	if m.Cursor < len(m.Entries)-1 {
		m.Cursor++
		m.UpdateViewport()
	}
}

func (m *Model) moveTree(delta int) {
	for i, idx := range m.treeOrder {
		if idx == m.Cursor {
			next := i + delta
			if next >= 0 && next < len(m.treeOrder) {
				m.Cursor = m.treeOrder[next]
				m.UpdateViewport()
			}
			return
		}
	}
}

func (m Model) SelectedFile() string {
	if m.Cursor < len(m.Entries) {
		return m.Entries[m.Cursor].Path
	}
	return ""
}

func (m *Model) SelectVisibleRow(row int) bool {
	if row < 0 {
		return false
	}
	line := m.viewport.YOffset + row
	if line < 0 || line >= len(m.rowIndex) {
		return false
	}
	idx := m.rowIndex[line]
	if idx < 0 || idx >= len(m.Entries) {
		return false
	}
	m.Cursor = idx
	m.UpdateViewport()
	return true
}

func (m *Model) ToggleTreeMode() {
	m.TreeMode = !m.TreeMode
	m.UpdateViewport()
}

func (m *Model) UpdateViewport() {
	var content string
	var cursorRow int
	if m.TreeMode {
		content, cursorRow = m.buildTreeView()
	} else {
		content, cursorRow = m.buildFlatView()
	}
	m.viewport.SetContent(content)
	m.keepCursorVisible(cursorRow)
}

func (m *Model) buildFlatView() (string, int) {
	var b strings.Builder
	m.rowIndex = m.rowIndex[:0]
	cursorRow := 0
	for i, entry := range m.Entries {
		if i == m.Cursor {
			cursorRow = len(m.rowIndex)
		}
		for _, line := range m.renderEntryLines(entry, i == m.Cursor) {
			m.rowIndex = append(m.rowIndex, i)
			b.WriteString(line + "\n")
		}
	}
	return b.String(), cursorRow
}

func (m Model) renderEntryLines(entry discover.FileEntry, selected bool) []string {
	if m.width <= 32 {
		return m.renderWrappedBasename(entry, selected)
	}

	if m.ShowTime {
		return m.renderEntryWithTime(entry, selected)
	}

	name := entry.Path
	nameWidth := m.width - 4
	if nameWidth < 4 {
		nameWidth = 4
	}
	name = truncateStart(name, nameWidth)
	label := "• " + name
	line := fmt.Sprintf(" %-*s", m.width-2, label)
	if selected {
		return []string{selectedStyle.Render(line)}
	}
	return []string{normalStyle.Render(line)}
}

func (m Model) renderEntryWithTime(entry discover.FileEntry, selected bool) []string {
	timeStr := dimTimeStyle.Render(discover.TimeAgo(entry.ModTime))
	availWidth := m.width - 15
	if availWidth < 10 {
		availWidth = 10
	}
	name := entry.Path
	nameWidth := availWidth - 2
	if nameWidth < 1 {
		nameWidth = 1
	}
	name = truncateStart(name, nameWidth)
	label := "• " + name
	line := fmt.Sprintf(" %-*s %s", availWidth, label, timeStr)
	if selected {
		return []string{selectedStyle.Render(line)}
	}
	return []string{normalStyle.Render(line)}
}

func (m Model) renderWrappedBasename(entry discover.FileEntry, selected bool) []string {
	name := filepath.Base(entry.Path)
	nameWidth := m.width - 3
	if nameWidth < 1 {
		nameWidth = 1
	}
	parts := wrapCells(name, nameWidth)
	lines := make([]string, 0, len(parts))
	for i, part := range parts {
		prefix := "  "
		if i == 0 {
			prefix = "• "
		}
		line := prefix + part
		if selected && i == 0 {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, normalStyle.Render(line))
	}
	return lines
}

func truncateStart(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[len(s)-width:]
	}
	return "..." + s[len(s)-(width-3):]
}

func (m *Model) buildTreeView() (string, int) {
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
	cursorRow := 0
	m.rowIndex = m.rowIndex[:0]
	m.treeOrder = m.treeOrder[:0]
	var b strings.Builder
	for di, dir := range dirs {
		isLastDir := di == len(dirs)-1
		if dir != "." {
			dirPrefix := "├─ "
			if isLastDir {
				dirPrefix = "└─ "
			}
			line := dirPrefix + dirStyle.Render(dir+"/")
				m.rowIndex = append(m.rowIndex, -1)
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
			line := prefix + name
			if m.ShowTime {
				line += dimTimeStyle.Render(" " + discover.TimeAgo(e.ModTime))
			}
			entryIdx := entryIndex(m.Entries, e.Path)
			if entryIdx == m.Cursor {
				line = selectedStyle.Render(line)
				cursorRow = len(m.rowIndex)
			}
			m.rowIndex = append(m.rowIndex, entryIdx)
				m.treeOrder = append(m.treeOrder, entryIdx)
				b.WriteString(line + "\n")
				lineIdx++
		}
	}
	return b.String(), cursorRow
}

func (m *Model) keepCursorVisible(cursorRow int) {
	if m.height <= 0 {
		return
	}
	if cursorRow < m.viewport.YOffset {
		m.viewport.SetYOffset(cursorRow)
		return
	}
	bottom := m.viewport.YOffset + m.height
	if cursorRow >= bottom {
		m.viewport.SetYOffset(cursorRow - m.height + 1)
	}
}

func wrapCells(s string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	if lipgloss.Width(s) <= width {
		return []string{s}
	}
	var parts []string
	var b strings.Builder
	currentWidth := 0
	for _, r := range s {
		w := lipgloss.Width(string(r))
		if currentWidth > 0 && currentWidth+w > width {
			parts = append(parts, b.String())
			b.Reset()
			currentWidth = 0
		}
		b.WriteRune(r)
		currentWidth += w
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	return parts
}

func entryIndex(entries []discover.FileEntry, path string) int {
	for i, entry := range entries {
		if entry.Path == path {
			return i
		}
	}
	return -1
}

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
