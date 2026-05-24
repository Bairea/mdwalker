package markdown

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Highlight struct {
	Line  int
	Style lipgloss.Style
	Label string
}

var (
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	todoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("120")).Bold(true)
	decisionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

	patterns = []struct {
		re    *regexp.Regexp
		style lipgloss.Style
		label string
	}{
		{regexp.MustCompile(`^(Error:|\[ERROR\]|❌)`), errorStyle, "ERR"},
		{regexp.MustCompile(`^(Warning:|\[WARN\]|⚠️)`), warnStyle, "WRN"},
		{regexp.MustCompile(`^(TODO:|\[TODO\])`), todoStyle, "TODO"},
		{regexp.MustCompile(`^(Next Steps:|Decision:)`), decisionStyle, "DEC"},
	}
)

func ScanSemantic(content string) map[int]Highlight {
	results := make(map[int]Highlight)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		for _, p := range patterns {
			if p.re.MatchString(strings.TrimSpace(line)) {
				results[i] = Highlight{Line: i, Style: p.style, Label: p.label}
				break
			}
		}
	}
	return results
}

func ApplySemanticLine(line string, h *Highlight) string {
	if h == nil {
		return line
	}
	prefix := h.Style.Render("[" + h.Label + "] ")
	return prefix + line
}
