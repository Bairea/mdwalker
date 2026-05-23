package discover

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileEntry struct {
	Path    string
	ModTime time.Time
	IsDir   bool
}

var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "target": true,
	".direnv": true, "__pycache__": true,
}

var priorityFiles = map[string]int{
	"AGENTS.md": 1,
	"CLAUDE.md": 2,
	"README.md": 3,
}

var priorityDirs = []string{".ai", ".claude", ".codex"}
var secondaryDirs = []string{"docs", "notes", "reports"}

func Scan(root string) ([]FileEntry, error) {
	entries, err := scanWithFD(root)
	if err != nil {
		entries, err = scanNative(root)
		if err != nil {
			return nil, err
		}
	}
	sortEntries(entries)
	return entries, nil
}

func scanWithFD(root string) ([]FileEntry, error) {
	_, err := exec.LookPath("fd")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("fd", "--type", "f", "--extension", "md", "--search-path", root)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var entries []FileEntry
	for _, line := range lines {
		if line == "" {
			continue
		}
		info, err := os.Stat(line)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, line)
		entries = append(entries, FileEntry{
			Path:    rel,
			ModTime: info.ModTime(),
		})
	}
	return entries, nil
}

func scanNative(root string) ([]FileEntry, error) {
	var entries []FileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			info, _ := d.Info()
			rel, _ := filepath.Rel(root, path)
			entries = append(entries, FileEntry{
				Path:    rel,
				ModTime: info.ModTime(),
			})
		}
		return nil
	})
	return entries, err
}

func sortEntries(entries []FileEntry) {
	now := time.Now()
	last24h := now.Add(-24 * time.Hour)

	sort.SliceStable(entries, func(i, j int) bool {
		pi := priority(entries[i], last24h)
		pj := priority(entries[j], last24h)
		if pi != pj {
			return pi < pj
		}
		return entries[i].ModTime.After(entries[j].ModTime)
	})
}

func priority(e FileEntry, last24h time.Time) int {
	base := filepath.Base(e.Path)
	if p, ok := priorityFiles[base]; ok {
		return p
	}
	dir := filepath.Dir(e.Path)
	for _, d := range priorityDirs {
		if strings.HasPrefix(dir, d) || strings.Contains(dir, "/"+d) {
			return 10
		}
	}
	if e.ModTime.After(last24h) {
		return 11
	}
	for _, d := range secondaryDirs {
		if strings.HasPrefix(dir, d) || strings.Contains(dir, "/"+d) {
			return 20
		}
	}
	return 30
}

func TimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
