package search

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	input   textinput.Model
	Active  bool
	Query   string
	Matches []Match
	Current int
}

type Match struct {
	Line int
	Text string
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
