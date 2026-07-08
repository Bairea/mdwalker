package markdown

import (
	"os/exec"
	"strings"
)

type Block struct {
	StartLine int
	EndLine   int
	Language  string
	Content   string
}

func ExtractBlocks(content string) []Block {
	var blocks []Block
	lines := strings.Split(content, "\n")
	inBlock := false
	var current Block

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") && !inBlock {
			inBlock = true
			current = Block{
				StartLine: i,
				Language:  strings.TrimPrefix(trimmed, "```"),
			}
		} else if strings.HasPrefix(trimmed, "```") && inBlock {
			inBlock = false
			current.EndLine = i
			current.Content = strings.Join(lines[current.StartLine+1:i], "\n")
			blocks = append(blocks, current)
		}
	}
	return blocks
}

func BlockAtLine(blocks []Block, line int) *Block {
	for i := range blocks {
		if line >= blocks[i].StartLine && line <= blocks[i].EndLine {
			return &blocks[i]
		}
	}
	return nil
}

func CopyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

