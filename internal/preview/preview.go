package preview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	mdimage "github.com/bairea/mdwalker/internal/image"
	"github.com/bairea/mdwalker/internal/mermaid"
)

type Heading struct {
	Level int
	Text  string
	Line  int
}

type Model struct {
	viewport    viewport.Model
	renderer    *glamour.TermRenderer
	root        string
	filePath    string
	content     string
	headings    []Heading
	foldStates  map[int]bool
	cursorLine  int
	rendered    string
	renderMedia bool
	width       int
	height      int
	ready       bool
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
	return m.loadFile(root, path, true)
}

func (m *Model) LoadFileLight(root, path string) error {
	return m.loadFile(root, path, false)
}

func (m *Model) loadFile(root, path string, renderMedia bool) error {
	m.root = root
	m.renderMedia = renderMedia
	fullPath := filepath.Join(root, path)
	if isImagePath(path) {
		m.filePath = path
		m.content = m.renderImage(fullPath, path)
		m.headings = nil
		m.foldStates = make(map[int]bool)
		m.cursorLine = 0
		m.renderFolded()
		return nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	m.filePath = path
	m.content = string(data)
	m.parseHeadings()
	m.cursorLine = 0
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
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		isFence := strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
		if isFence {
			if inFence {
				fenceChar := "`"
				if strings.HasPrefix(trimmed, "~~~") {
					fenceChar = "~"
				}
				rest := strings.TrimLeft(trimmed, fenceChar)
				if strings.TrimSpace(rest) == "" {
					inFence = false
				}
			} else {
				inFence = true
			}
			continue
		}
		if inFence {
			continue
		}
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
	headingIdx := -1
	for i := range m.headings {
		start, end := m.contentRange(i)
		if cursorLine >= start && cursorLine < end {
			headingIdx = i
		}
	}
	if headingIdx < 0 || !m.IsLineVisible(cursorLine) {
		return
	}
	line := m.headings[headingIdx].Line
	m.foldStates[line] = !m.foldStates[line]
	m.renderFolded()
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
	markdown, blocks := m.extractMediaBlocks(filtered)
	rendered, err := m.renderer.Render(markdown)
	if err != nil {
		m.rendered = filtered
		m.viewport.SetContent(filtered)
		return
	}
	for _, b := range blocks {
		lines := strings.Split(rendered, "\n")
		for i, line := range lines {
			if strings.Contains(line, b.placeholder) {
				outputLines := strings.Split(strings.Trim(b.output, "\n"), "\n")
				newLines := make([]string, 0, len(lines)+len(outputLines))
				newLines = append(newLines, lines[:i]...)
				newLines = append(newLines, outputLines...)
				newLines = append(newLines, lines[i+1:]...)
				lines = newLines
				break
			}
		}
		rendered = strings.Join(lines, "\n")
	}
	m.rendered = rendered
	m.viewport.SetContent(rendered)
}

func (m *Model) ScrollUp()           { m.viewport.LineUp(1) }
func (m *Model) ScrollDown()         { m.viewport.LineDown(1) }
func (m *Model) ScrollTop()          { m.viewport.GotoTop() }
func (m *Model) ScrollBottom()       { m.viewport.GotoBottom() }
func (m *Model) ScrollHalfPageUp()   { m.viewport.HalfViewUp() }
func (m *Model) ScrollHalfPageDown() { m.viewport.HalfViewDown() }

func (m Model) FilePath() string { return m.filePath }
func (m Model) Content() string  { return m.content }
func (m Model) CurrentLine() int { return m.cursorLine }
func (m *Model) ScrollToLine(line int) {
	m.cursorLine = line
	m.viewport.SetYOffset(m.renderedOffsetForLine(line))
}

func (m Model) renderedOffsetForLine(line int) int {
	sourceLines := strings.Split(m.content, "\n")
	if line < 0 || line >= len(sourceLines) {
		return 0
	}

	fullLine := strings.TrimSpace(sourceLines[line])
	if fullLine == "" {
		return line
	}

	headingText := strings.TrimSpace(strings.TrimLeft(fullLine, "#"))

	renderedLines := strings.Split(m.rendered, "\n")

	searchFrom := func(text string, start int) int {
		if start >= len(renderedLines) {
			start = len(renderedLines) - 1
		}
		if start < 0 {
			start = 0
		}
		for i := start; i < len(renderedLines); i++ {
			if text != "" && strings.Contains(renderedLines[i], text) {
				return i
			}
		}
		for i := 0; i < start; i++ {
			if text != "" && strings.Contains(renderedLines[i], text) {
				return i
			}
		}
		return -1
	}

	approxStart := line - 5
	if approxStart < 0 {
		approxStart = 0
	}

	if idx := searchFrom(fullLine, approxStart); idx >= 0 {
		return idx
	}
	if headingText != fullLine {
		if idx := searchFrom(headingText, approxStart); idx >= 0 {
			return idx
		}
	}
	return line
}

func (m *Model) SetCursorFromVisibleRow(row int) {
	if row < 0 {
		row = 0
	}
	line := m.viewport.YOffset + row
	if len(m.headings) > 0 {
		line = m.nearestHeadingLine(line)
	}
	m.cursorLine = line
}

func (m Model) nearestHeadingLine(visibleRow int) int {
	approxSourceLine := m.viewport.YOffset + max(0, (visibleRow-1)/2)
	best := approxSourceLine
	bestDistance := len(strings.Split(m.content, "\n")) + 1
	for _, h := range m.headings {
		distance := h.Line - approxSourceLine
		if distance < 0 {
			distance = -distance
		}
		if distance < bestDistance {
			bestDistance = distance
			best = h.Line
		}
	}
	return best
}

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func isImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}

type mediaBlock struct {
	placeholder string
	output      string
}

var mediaCounter int

func (m Model) extractMediaBlocks(content string) (string, []mediaBlock) {
	var blocks []mediaBlock
	content = m.replaceMermaidWithPlaceholders(content, &blocks)
	content = m.replaceImagesWithPlaceholders(content, &blocks)
	return content, blocks
}

func (m Model) replaceMermaidWithPlaceholders(content string, blocks *[]mediaBlock) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	var block []string
	inMermaid := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inMermaid && strings.HasPrefix(trimmed, "```mermaid") {
			inMermaid = true
			block = block[:0]
			continue
		}
		if inMermaid {
			if strings.HasPrefix(trimmed, "```") {
				output := m.renderMermaidBlock(strings.Join(block, "\n"))
				mediaCounter++
				key := fmt.Sprintf("%%MDWALKER_MERMAID_%d%%", mediaCounter)
				*blocks = append(*blocks, mediaBlock{key, output})
				out = append(out, key)
				inMermaid = false
				continue
			}
			block = append(block, line)
			continue
		}
		out = append(out, line)
	}
	if inMermaid {
		output := m.renderMermaidBlock(strings.Join(block, "\n"))
		mediaCounter++
		key := fmt.Sprintf("%%MDWALKER_MERMAID_%d%%", mediaCounter)
		*blocks = append(*blocks, mediaBlock{key, output})
		out = append(out, key)
	}
	return strings.Join(out, "\n")
}

func (m Model) replaceImagesWithPlaceholders(content string, blocks *[]mediaBlock) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		refs := mdimage.Extract(line)
		if len(refs) == 0 {
			continue
		}
		placeholders := make([]string, 0, len(refs))
		for _, ref := range refs {
			resolved := m.resolveAssetPath(ref.Path)
			output := m.renderImage(resolved, ref.Path)
			mediaCounter++
			placeholder := fmt.Sprintf("%%MDWALKER_IMG_%d%%", mediaCounter)
			*blocks = append(*blocks, mediaBlock{placeholder, output})
			placeholders = append(placeholders, placeholder)
		}
		lines[i] = strings.Join(placeholders, " ")
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderMermaidBlock(content string) string {
	if !m.renderMedia {
		return "[Mermaid: press Enter to render]"
	}
	path, err := mermaid.Render(content)
	if err != nil {
		return "[Mermaid: render unavailable: " + err.Error() + "]"
	}
	w, h := m.mediaWidth(), m.mediaHeight()
	if mdimage.TerminalSupportsImages() {
		rendered := mdimage.RenderInline(path, w, h)
		if strings.TrimSpace(rendered) != "" {
			return "\n" + rendered + "\n"
		}
	}
	rendered, err := mdimage.ToHalfblock(path, w, h)
	if err == nil && strings.TrimSpace(rendered) != "" {
		return "\n" + rendered + "\n"
	}
	return "[Mermaid: " + path + "]"
}

func (m Model) resolveAssetPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	baseDir := filepath.Dir(filepath.Join(m.root, m.filePath))
	return filepath.Clean(filepath.Join(baseDir, path))
}

func (m Model) renderImage(fullPath, displayPath string) string {
	if !m.renderMedia {
		return mdimage.RenderPlaceholder(mdimage.ImageRef{Path: displayPath})
	}
	w, h := m.mediaWidth(), m.mediaHeight()
	if mdimage.TerminalSupportsImages() {
		rendered := mdimage.RenderInline(fullPath, w, h)
		if strings.TrimSpace(rendered) != "" {
			return "\n" + rendered + "\n"
		}
	}
	rendered, err := mdimage.ToHalfblock(fullPath, w, h)
	if err == nil && strings.TrimSpace(rendered) != "" {
		return "\n" + rendered + "\n"
	}
	return mdimage.RenderPlaceholder(mdimage.ImageRef{Path: displayPath})
}

func (m Model) mediaWidth() int {
	width := m.width - 2
	if width <= 0 {
		return 80
	}
	if width > 120 {
		return 120
	}
	return width
}

func (m Model) mediaHeight() int {
	height := m.height - 2
	if height <= 0 {
		return 20
	}
	if height > 30 {
		return 30
	}
	return height
}
