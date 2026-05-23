package watch

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type Event struct {
	Path string
	Op   string
}

type Watcher struct {
	w      *fsnotify.Watcher
	Events chan Event
	Errors chan error
	root   string
}

func New(root string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		w:      w,
		Events: make(chan Event, 100),
		Errors: make(chan error, 10),
		root:   root,
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && name != ".ai" && name != ".claude" && name != ".codex" {
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

func (w *Watcher) loop() {
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
			w.Events <- Event{Path: rel, Op: op}
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			w.Errors <- err
		}
	}
}

func (w *Watcher) Close() {
	w.w.Close()
}
