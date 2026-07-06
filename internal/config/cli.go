package config

import (
	"flag"
	"os"
)

type Args struct {
	Root     string
	Files    []string
	NoWatch  bool
	ShowTime bool
}

func ParseArgs() Args {
	var a Args
	noWatch := flag.Bool("no-watch", false, "disable file watching")
	showTime := flag.Bool("show-time", false, "show file modification times in file list")
	flag.Parse()

	a.NoWatch = *noWatch
	a.ShowTime = *showTime

	positional := flag.Args()
	if len(positional) == 0 {
		a.Root, _ = os.Getwd()
	} else if len(positional) == 1 {
		info, err := os.Stat(positional[0])
		if err == nil && info.IsDir() {
			a.Root = positional[0]
		} else {
			a.Files = positional
			a.Root, _ = os.Getwd()
		}
	} else {
		a.Files = positional
		a.Root, _ = os.Getwd()
	}

	return a
}
