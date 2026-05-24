package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bairea/mdwalker/internal/app"
	"github.com/bairea/mdwalker/internal/config"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	args := config.ParseArgs()

	if *showVersion {
		fmt.Printf("mdwalker %s\n", version)
		os.Exit(0)
	}

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
