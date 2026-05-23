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
	b.WriteString(" Outline\n\n")
	for i, h := range m.Headings {
		prefix := strings.Repeat("  ", h.Level-1)
		style, ok := levelStyles[h.Level]
		if !ok {
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
