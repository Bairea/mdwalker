package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bairea/mdwalker/internal/app"
	"github.com/bairea/mdwalker/internal/config"
)

func main() {
	args := config.ParseArgs()

	root := args.Root
	if len(args.Files) > 0 {
		root = args.Files[0]
	}

	m := app.New(root)
	m.SetShowTime(args.ShowTime)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mdwalker: %v\n", err)
		os.Exit(1)
	}
}
