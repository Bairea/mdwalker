package preview

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type Heading struct {
	Level int
	Text  string
	Line  int
}

type Model struct {
	viewport   viewport.Model
	renderer   *glamour.TermRenderer
	filePath   string
	content    string
	headings   []Heading
	foldStates map[int]bool
	width      int
	height     int
	ready      bool
}

func New() Model {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	return Model{
		renderer:   r,
		foldStates: make(map[int]bool),
	}
}

func (m *Model) LoadFile(root, path string) error {
	fullPath := root + "/" + path
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	m.filePath = path
	m.content = string(data)
	m.parseHeadings()
	m.renderFolded()
	return nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.ready = true
	m.renderFolded()
}

func (m *Model) parseHeadings() {
	m.headings = nil
	lines := strings.Split(m.content, "\n")
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
				m.headings = append(m.headings, Heading{
					Level: level,
					Text:  strings.TrimSpace(line[level:]),
					Line:  i,
				})
			}
		}
	}
	m.foldStates = make(map[int]bool)
}

func (m *Model) contentRange(headingIdx int) (int, int) {
	startLine := m.headings[headingIdx].Line
	level := m.headings[headingIdx].Level
	endLine := len(strings.Split(m.content, "\n"))
	for j := headingIdx + 1; j < len(m.headings); j++ {
		if m.headings[j].Level <= level {
			endLine = m.headings[j].Line
			break
		}
	}
	return startLine, endLine
}

func (m *Model) ToggleFold(cursorLine int) {
	for i, h := range m.headings {
		start, end := m.contentRange(i)
		if cursorLine >= start && cursorLine < end {
			if cursorLine == start {
				m.foldStates[h.Line] = !m.foldStates[h.Line]
				m.renderFolded()
				return
			}
			// check if a parent heading is folded
			for j := i; j >= 0; j-- {
				if m.foldStates[m.headings[j].Line] {
					startJ, _ := m.contentRange(j)
					if cursorLine >= startJ {
						return // inside folded content, can't toggle
					}
				}
			}
		}
	}
}

func (m *Model) IsLineVisible(line int) bool {
	for i, h := range m.headings {
		if m.foldStates[h.Line] {
			_, end := m.contentRange(i)
			if line > h.Line && line < end {
				return false
			}
		}
	}
	return true
}

func (m *Model) HeadingAtLine(line int) *Heading {
	for i := range m.headings {
		if m.headings[i].Line == line {
			return &m.headings[i]
		}
	}
	return nil
}

func (m *Model) CurrentHeading(cursorLine int) *Heading {
	var current *Heading
	for i := range m.headings {
		if m.headings[i].Line <= cursorLine {
			current = &m.headings[i]
		}
	}
	return current
}

func (m *Model) renderFolded() {
	if !m.ready || m.content == "" {
		return
	}
	lines := strings.Split(m.content, "\n")
	var visible []string
	for i, line := range lines {
		if m.IsLineVisible(i) {
			h := m.HeadingAtLine(i)
			if h != nil && m.foldStates[i] {
				visible = append(visible, line+"  ...")
			} else {
				visible = append(visible, line)
			}
		}
	}
	filtered := strings.Join(visible, "\n")
	rendered, err := m.renderer.Render(filtered)
	if err != nil {
		m.viewport.SetContent(filtered)
		return
	}
	m.viewport.SetContent(rendered)
}

func (m *Model) render() {
	m.renderFolded()
}

func (m *Model) ScrollUp()           { m.viewport.LineUp(1) }
func (m *Model) ScrollDown()         { m.viewport.LineDown(1) }
func (m *Model) ScrollTop()          { m.viewport.GotoTop() }
func (m *Model) ScrollBottom()       { m.viewport.GotoBottom() }
func (m *Model) ScrollHalfPageUp()   { m.viewport.HalfViewUp() }
func (m *Model) ScrollHalfPageDown() { m.viewport.HalfViewDown() }

func (m Model) FilePath() string       { return m.filePath }
func (m Model) Content() string        { return m.content }
func (m Model) CurrentLine() int       { return m.viewport.YOffset + m.viewport.Height/2 }
func (m *Model) ScrollToLine(line int) { m.viewport.SetYOffset(line) }

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
