package search

import (
	"fmt"
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
)

type Model struct {
	input       textinput.Model
	Active      bool
	Mode        SearchMode
	Query       string
	Matches     []Match
	Current     int
	FileMatches []FileMatch
	FileCurrent int
}

type Match struct {
	Line int
	Text string
}

type FileMatch struct {
	Index int
	Entry discover.FileEntry
}

var (
	barStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236"))
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
	m.Current = 0
	m.FileCurrent = 0
	m.input.SetValue("")
	if mode == ModeFileName {
		m.input.Placeholder = "search files..."
	} else {
		m.input.Placeholder = "search..."
	}
}

func (m *Model) ToggleMode() {
	if m.Mode == ModeContent {
		m.Mode = ModeFileName
		m.input.Placeholder = "search files..."
	} else {
		m.Mode = ModeContent
		m.input.Placeholder = "search..."
	}
	m.Matches = nil
	m.FileMatches = nil
	m.Current = 0
	m.FileCurrent = 0
}

func (m *Model) Deactivate() {
	m.Active = false
	m.input.Blur()
	m.Query = ""
	m.Matches = nil
	m.FileMatches = nil
	m.Current = 0
	m.FileCurrent = 0
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

func (m *Model) Next() {
	if m.Mode == ModeFileName {
		m.NextFile()
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

func (m *Model) UpdateSearch(files []discover.FileEntry, content string) {
	if m.Mode == ModeFileName {
		m.SearchFiles(files)
	} else {
		m.Search(content)
	}
}

func (m Model) View() string {
	if !m.Active {
		return ""
	}
	modeLabel := countStyle.Render("[content]")
	title := "Search content"
	if m.Mode == ModeFileName {
		modeLabel = countStyle.Render("[files]")
		title = "Search files"
	}
	count := ""
	if m.Mode == ModeContent && len(m.Matches) > 0 {
		count = countStyle.Render(fmt.Sprintf(" %d/%d", m.Current+1, len(m.Matches)))
	} else if m.Mode == ModeFileName && len(m.FileMatches) > 0 {
		count = countStyle.Render(fmt.Sprintf(" %d/%d", m.FileCurrent+1, len(m.FileMatches)))
	}
	header := barStyle.Render(" " + title + "\n /" + m.input.View() + " " + modeLabel + count + " Tab:switch Esc:cancel")
	if m.Mode != ModeFileName || len(m.FileMatches) == 0 {
		return header
	}

	var b strings.Builder
	b.WriteString(header)
	for i, match := range m.FileMatches {
		if i >= 8 {
			break
		}
		line := "  " + match.Entry.Path
		if i == m.FileCurrent {
			line = selectedCandidateStyle.Render(line)
		}
		b.WriteString("\n")
		b.WriteString(line)
	}
	return b.String()
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
