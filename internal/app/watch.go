package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type fileEvent struct {
	Path string
	Op   string
}

type fileWatcher struct {
	w      *fsnotify.Watcher
	Events chan fileEvent
	Errors chan error
	root   string
}

func newFileWatcher(root string, dotDirs []string) (*fileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &fileWatcher{
		w:      w,
		Events: make(chan fileEvent, 100),
		Errors: make(chan error, 10),
		root:   root,
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && !isWhitelistedDotDir(name, dotDirs) {
				return filepath.SkipDir
			}
			return w.Add(path)
		}
		return nil
	})
	if err != nil {
		w.Close()
		return nil, err
	}

	go watcher.loop()
	return watcher, nil
}

func (w *fileWatcher) loop() {
	for {
		select {
		case event, ok := <-w.w.Events:
			if !ok {
				return
			}
			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}
			rel, _ := filepath.Rel(w.root, event.Name)
			op := "write"
			if event.Has(fsnotify.Create) {
				op = "create"
			} else if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				op = "remove"
			}
			w.Events <- fileEvent{Path: rel, Op: op}
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			w.Errors <- err
		}
	}
}

func (w *fileWatcher) close() {
	w.w.Close()
}

func isWhitelistedDotDir(name string, dotDirs []string) bool {
	for _, d := range dotDirs {
		if name == d {
			return true
		}
	}
	return false
}
