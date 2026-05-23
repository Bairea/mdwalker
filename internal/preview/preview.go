package preview

import (
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type Model struct {
	viewport viewport.Model
	renderer *glamour.TermRenderer
	filePath string
	content  string
	width    int
	height   int
	ready    bool
}

func New() Model {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	return Model{renderer: r}
}

func (m *Model) LoadFile(root, path string) error {
	fullPath := root + "/" + path
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	m.filePath = path
	m.content = string(data)
	m.render()
	return nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.ready = true
	m.render()
}

func (m *Model) render() {
	if !m.ready || m.content == "" {
		return
	}
	rendered, err := m.renderer.Render(m.content)
	if err != nil {
		m.viewport.SetContent(m.content)
		return
	}
	m.viewport.SetContent(rendered)
	m.viewport.GotoTop()
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
