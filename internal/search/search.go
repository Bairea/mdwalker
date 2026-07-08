package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bairea/mdwalker/internal/discover"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SearchMode int

const (
	ModeContent SearchMode = iota
	ModeFileName
	ModeAllContent
)

type Model struct {
	input        textinput.Model
	Active       bool
	Mode         SearchMode
	Query        string
	Matches      []Match
	Current      int
	FileMatches  []FileMatch
	FileCurrent  int
	AllMatches   []AllContentMatch
	AllCurrent   int
	width        int
	height       int
}

type Match struct {
	Line int
	Text string
}

type FileMatch struct {
	Index int
	Entry discover.FileEntry
}

type AllContentMatch struct {
	Path string
	Line int
	Text string
}

var (
	countStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 80
	return Model{input: ti}
}

func (m *Model) Activate(mode SearchMode) {
	m.Active = true
	m.Mode = mode
	m.input.Focus()
	m.Query = ""
	m.Matches = nil
	m.FileMatches = nil
	m.AllMatches = nil
	m.Current = 0
	m.FileCurrent = 0
	m.AllCurrent = 0
	m.input.SetValue("")
	if mode == ModeFileName {
		m.input.Placeholder = "search files..."
	} else {
		m.input.Placeholder = "search..."
	}
}

func (m *Model) ToggleMode() {
	switch m.Mode {
	case ModeFileName:
		m.Mode = ModeContent
		m.input.Placeholder = "search..."
	case ModeContent:
		m.Mode = ModeAllContent
		m.input.Placeholder = "search..."
	case ModeAllContent:
		m.Mode = ModeFileName
		m.input.Placeholder = "search files..."
	}
	m.Matches = nil
	m.FileMatches = nil
	m.AllMatches = nil
	m.Current = 0
	m.FileCurrent = 0
	m.AllCurrent = 0
}

func (m *Model) Deactivate() {
	m.Active = false
	m.input.Blur()
	m.Query = ""
	m.Matches = nil
	m.FileMatches = nil
	m.AllMatches = nil
	m.Current = 0
	m.FileCurrent = 0
	m.AllCurrent = 0
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	if width <= 0 {
		m.input.Width = 20
		return
	}
	m.input.Width = modalContentWidth(width) - 2
	if m.input.Width < 12 {
		m.input.Width = 12
	}
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

func (m *Model) SearchFiles(entries []discover.FileEntry) {
	if m.Query == "" {
		m.FileMatches = nil
		return
	}
	m.FileMatches = nil
	lower := strings.ToLower(m.Query)
	for i, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Path), lower) {
			m.FileMatches = append(m.FileMatches, FileMatch{Index: i, Entry: entry})
		}
	}
	if m.FileCurrent >= len(m.FileMatches) {
		m.FileCurrent = 0
	}
}

func (m *Model) SearchAllContent(root string, entries []discover.FileEntry) {
	if m.Query == "" {
		m.AllMatches = nil
		return
	}
	m.AllMatches = nil
	lower := strings.ToLower(m.Query)
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(root, entry.Path))
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), lower) {
				m.AllMatches = append(m.AllMatches, AllContentMatch{
					Path: entry.Path,
					Line: i,
					Text: line,
				})
			}
		}
	}
	if m.AllCurrent >= len(m.AllMatches) {
		m.AllCurrent = 0
	}
}

func (m *Model) Next() {
	if m.Mode == ModeFileName {
		m.NextFile()
		return
	}
	if m.Mode == ModeAllContent {
		if len(m.AllMatches) > 0 {
			m.AllCurrent = (m.AllCurrent + 1) % len(m.AllMatches)
		}
		return
	}
	if len(m.Matches) > 0 {
		m.Current = (m.Current + 1) % len(m.Matches)
	}
}

func (m *Model) Prev() {
	if m.Mode == ModeFileName {
		m.PrevFile()
		return
	}
	if m.Mode == ModeAllContent {
		if len(m.AllMatches) > 0 {
			m.AllCurrent--
			if m.AllCurrent < 0 {
				m.AllCurrent = len(m.AllMatches) - 1
			}
		}
		return
	}
	if len(m.Matches) > 0 {
		m.Current--
		if m.Current < 0 {
			m.Current = len(m.Matches) - 1
		}
	}
}

func (m *Model) NextFile() {
	if len(m.FileMatches) > 0 {
		m.FileCurrent = (m.FileCurrent + 1) % len(m.FileMatches)
	}
}

func (m *Model) PrevFile() {
	if len(m.FileMatches) > 0 {
		m.FileCurrent--
		if m.FileCurrent < 0 {
			m.FileCurrent = len(m.FileMatches) - 1
		}
	}
}

func (m Model) CurrentLine() int {
	if m.Current < len(m.Matches) {
		return m.Matches[m.Current].Line
	}
	return 0
}

func (m Model) CurrentFileIndex() int {
	if m.FileCurrent < len(m.FileMatches) {
		return m.FileMatches[m.FileCurrent].Index
	}
	return -1
}

func (m Model) CurrentAllMatch() AllContentMatch {
	if m.AllCurrent < len(m.AllMatches) {
		return m.AllMatches[m.AllCurrent]
	}
	return AllContentMatch{}
}

func (m *Model) UpdateSearch(root string, files []discover.FileEntry, content string) {
	switch m.Mode {
	case ModeFileName:
		m.SearchFiles(files)
	case ModeContent:
		m.Search(content)
	case ModeAllContent:
		m.SearchAllContent(root, files)
	}
}

func (m Model) View() string {
	if !m.Active {
		return ""
	}
	panelWidth := modalPanelInnerWidth(m.width)
	contentWidth := panelWidth - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	modeLabel := "content"
	title := "Search content"
	if m.Mode == ModeFileName {
		modeLabel = "files"
		title = "Search files"
	} else if m.Mode == ModeAllContent {
		modeLabel = "all"
		title = "Search all files"
	}
	count := ""
	if m.Mode == ModeContent && len(m.Matches) > 0 {
		count = fmt.Sprintf("%d/%d", m.Current+1, len(m.Matches))
	} else if m.Mode == ModeFileName && len(m.FileMatches) > 0 {
		count = fmt.Sprintf("%d/%d", m.FileCurrent+1, len(m.FileMatches))
	} else if m.Mode == ModeAllContent && len(m.AllMatches) > 0 {
		count = fmt.Sprintf("%d/%d", m.AllCurrent+1, len(m.AllMatches))
	}
	meta := modeLabel
	if count != "" {
		meta += " · " + count
	}
	header := renderAlignedLine(title, meta, contentWidth)
	prompt := inputLineStyle.Render(padRight(truncateCells("/"+m.input.View(), contentWidth), contentWidth))
	help := countStyle.Render("Tab switch  Enter open  Esc cancel")

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(prompt)
	b.WriteString("\n")
	b.WriteString(help)

	if m.Mode == ModeFileName && len(m.FileMatches) > 0 {
		for i, match := range m.FileMatches {
			if i >= 8 {
				break
			}
			prefix := "  "
			if i == m.FileCurrent {
				prefix = "› "
			}
			line := padRight(prefix+truncateCells(match.Entry.Path, contentWidth-2), contentWidth)
			if i == m.FileCurrent {
				line = selectedCandidateStyle.Render(line)
			}
			b.WriteString("\n")
			b.WriteString(line)
		}
	}

	if m.Mode == ModeContent && len(m.Matches) > 0 {
		b.WriteString("\n")
		b.WriteString(countStyle.Render(strings.Repeat("─", contentWidth)))
		for i, match := range m.Matches {
			if i >= 8 {
				b.WriteString("\n")
				b.WriteString(countStyle.Render(fmt.Sprintf("  ... %d more matches", len(m.Matches)-8)))
				break
			}
			prefix := "  "
			if i == m.Current {
				prefix = "› "
			}
			text := strings.TrimSpace(match.Text)
			line := padRight(prefix+truncateCells(fmt.Sprintf("L%d %s", match.Line+1, text), contentWidth-2), contentWidth)
			if i == m.Current {
				line = selectedCandidateStyle.Render(line)
			}
			b.WriteString("\n")
			b.WriteString(line)
		}
	}

	if m.Mode == ModeAllContent && len(m.AllMatches) > 0 {
		b.WriteString("\n")
		b.WriteString(countStyle.Render(strings.Repeat("─", contentWidth)))
		for i, match := range m.AllMatches {
			if i >= 8 {
				b.WriteString("\n")
				b.WriteString(countStyle.Render(fmt.Sprintf("  ... %d more matches", len(m.AllMatches)-8)))
				break
			}
			prefix := "  "
			if i == m.AllCurrent {
				prefix = "› "
			}
			text := strings.TrimSpace(match.Text)
			line := padRight(prefix+truncateCells(fmt.Sprintf("%s:L%d %s", match.Path, match.Line+1, text), contentWidth-2), contentWidth)
			if i == m.AllCurrent {
				line = selectedCandidateStyle.Render(line)
			}
			b.WriteString("\n")
			b.WriteString(line)
		}
	}

	return searchPanelStyle.Width(panelWidth).Render(b.String())
}
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.Query = m.input.Value()
	return m, cmd
}

var selectedCandidateStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("62")).
	Foreground(lipgloss.Color("255"))

var (
	searchPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 2).
				Background(lipgloss.Color("235"))
	inputLineStyle = lipgloss.NewStyle().Background(lipgloss.Color("236"))
)

func modalContentWidth(width int) int {
	w := modalPanelInnerWidth(width) - 4
	if w < 20 {
		return 20
	}
	return w
}

func modalPanelInnerWidth(width int) int {
	if width <= 0 {
		return 54
	}
	w := width * 70 / 100
	if w < 44 {
		w = 44
	}
	if w > 72 {
		w = 72
	}
	if w > width-4 {
		w = width - 4
	}
	if w < 24 {
		w = 24
	}
	return w - 2
}

func renderAlignedLine(left, right string, width int) string {
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	if leftWidth+rightWidth >= width {
		maxLeft := width - rightWidth - 1
		if maxLeft < 1 {
			maxLeft = width
		}
		left = truncateCells(left, maxLeft)
		leftWidth = lipgloss.Width(left)
	}
	padding := width - leftWidth - rightWidth
	if padding < 1 {
		padding = 1
	}
	return left + strings.Repeat(" ", padding) + right
}

func truncateCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	var b strings.Builder
	for _, r := range s {
		next := b.String() + string(r)
		if lipgloss.Width(next)+1 > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "…"
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
