package discover

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bairea/mdwalker/internal/config"
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

var priorityDirs = []string{".ai", ".claude", ".codex", ".agents", ".pi", ".trae", ".omx"}
var secondaryDirs = []string{"docs", "notes", "reports"}
var previewableExts = map[string]bool{
	".md": true,
}

func Scan(root string, wl *config.WhitelistConfig) ([]FileEntry, error) {
	entries, err := scanWithFD(root)
	if err != nil {
		entries, err = scanNative(root)
		if err != nil {
			return nil, err
		}
	}
	if wl != nil {
		unignoreEntries := scanUnignoreDirs(root, wl)
		entries = append(entries, unignoreEntries...)
		for _, f := range wl.Unignore.Files {
			fullPath := filepath.Join(root, f)
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				entries = append(entries, FileEntry{Path: f, ModTime: info.ModTime()})
			}
		}
	}
	entries = filterSkipSubdirs(entries, wl)
	entries = dedupEntries(entries)
	sortEntries(entries)
	return entries, nil
}

func scanWithFD(root string) ([]FileEntry, error) {
	_, err := exec.LookPath("fd")
	if err != nil {
		return nil, err
	}
	args := []string{"--type", "f", "-H", "--extension", "md", "--search-path", root}
	cmd := exec.Command("fd", args...)
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
		if isPreviewableFile(d.Name()) {
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

func scanUnignoreDirs(root string, wl *config.WhitelistConfig) []FileEntry {
	var entries []FileEntry
	scanDir := func(dir string) {
		filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				rel, _ := filepath.Rel(root, path)
				if shouldSkipDir(rel, wl.SkipSubdirs) {
					return fs.SkipDir
				}
				return nil
			}
			if isPreviewableFile(d.Name()) {
				info, _ := d.Info()
				rel, _ := filepath.Rel(root, path)
				entries = append(entries, FileEntry{
					Path:    rel,
					ModTime: info.ModTime(),
				})
			}
			return nil
		})
	}
	for _, d := range wl.Unignore.DotDirs {
		scanDir(d)
	}
	for _, p := range wl.Unignore.Paths {
		scanDir(p)
	}
	return entries
}

func shouldSkipDir(relPath string, skipPatterns []string) bool {
	relPath = filepath.ToSlash(relPath)
	for _, pat := range skipPatterns {
		pat = filepath.ToSlash(pat)
		for dir := relPath; dir != "." && dir != "/"; {
			if matched, _ := path.Match(pat, dir); matched {
				return true
			}
			dir = path.Dir(dir)
		}
	}
	return false
}

func filterSkipSubdirs(entries []FileEntry, wl *config.WhitelistConfig) []FileEntry {
	if wl == nil || len(wl.SkipSubdirs) == 0 {
		return entries
	}
	var filtered []FileEntry
	for _, e := range entries {
		if shouldSkipDir(filepath.Dir(e.Path), wl.SkipSubdirs) {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

func dedupEntries(entries []FileEntry) []FileEntry {
	seen := make(map[string]bool, len(entries))
	var out []FileEntry
	for _, e := range entries {
		if !seen[e.Path] {
			seen[e.Path] = true
			out = append(out, e)
		}
	}
	return out
}

func isPreviewableFile(name string) bool {
	return previewableExts[strings.ToLower(filepath.Ext(name))]
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
