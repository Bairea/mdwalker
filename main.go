package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bairea/mdwalker/internal/app"
	"github.com/bairea/mdwalker/internal/cli"
)

func main() {
	args := cli.Parse()

	root := args.Root
	if len(args.Files) > 0 {
		root = args.Files[0]
	}

	p := tea.NewProgram(
		app.New(root),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mdwalker: %v\n", err)
		os.Exit(1)
	}
}
