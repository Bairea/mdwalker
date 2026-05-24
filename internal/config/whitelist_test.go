package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWhitelistDefaults(t *testing.T) {
	wl := LoadWhitelist()

	if len(wl.Unignore.DotDirs) < 30 {
		t.Errorf("expected at least 30 built-in dot_dirs, got %d", len(wl.Unignore.DotDirs))
	}

	foundClaude := false
	foundAgents := false
	for _, d := range wl.Unignore.DotDirs {
		if d == ".claude" {
			foundClaude = true
		}
		if d == ".agents" {
			foundAgents = true
		}
	}
	if !foundClaude || !foundAgents {
		t.Error("expected .claude and .agents in defaults")
	}

	if len(wl.SkipSubdirs) != 1 || wl.SkipSubdirs[0] != "*/skills" {
		t.Errorf("expected default skip_subdirs [*/skills], got %v", wl.SkipSubdirs)
	}

	if len(wl.Unignore.Files) < 2 {
		t.Errorf("expected at least 2 built-in files, got %d", len(wl.Unignore.Files))
	}
	foundAGENTS := false
	for _, f := range wl.Unignore.Files {
		if f == "AGENTS.md" {
			foundAGENTS = true
		}
	}
	if !foundAGENTS {
		t.Error("expected AGENTS.md in default files")
	}
}

func TestLoadWhitelistMergeProject(t *testing.T) {
	dir := t.TempDir()
	os.Chdir(dir)

	projectYAML := `
unignore:
  dot_dirs:
    - .mycustom
  paths:
    - docs/custom
  files:
    - CUSTOM.md
skip_subdirs:
  - "*/custom"
`
	os.WriteFile("mdwalker-whitelist.yaml", []byte(projectYAML), 0644)

	wl := LoadWhitelist()

	hasCustom := false
	for _, d := range wl.Unignore.DotDirs {
		if d == ".mycustom" {
			hasCustom = true
			break
		}
	}
	if !hasCustom {
		t.Error("expected .mycustom from project-level whitelist")
	}

	hasCustomPath := false
	for _, p := range wl.Unignore.Paths {
		if p == "docs/custom" {
			hasCustomPath = true
			break
		}
	}
	if !hasCustomPath {
		t.Error("expected docs/custom from project-level whitelist")
	}

	foundCustom := false
	for _, s := range wl.SkipSubdirs {
		if s == "*/custom" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Error("expected */custom in skip_subdirs")
	}

	hasCustomFile := false
	for _, f := range wl.Unignore.Files {
		if f == "CUSTOM.md" {
			hasCustomFile = true
			break
		}
	}
	if !hasCustomFile {
		t.Error("expected CUSTOM.md from project-level whitelist")
	}
}

func TestLoadWhitelistMergeGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	os.Chdir(dir)

	configDir := filepath.Join(home, ".config", "mdwalker")
	os.MkdirAll(configDir, 0755)
	globalYAML := `
unignore:
  dot_dirs:
    - .globalonly
`
	os.WriteFile(filepath.Join(configDir, "whitelist.yaml"), []byte(globalYAML), 0644)

	wl := LoadWhitelist()

	hasGlobal := false
	for _, d := range wl.Unignore.DotDirs {
		if d == ".globalonly" {
			hasGlobal = true
			break
		}
	}
	if !hasGlobal {
		t.Error("expected .globalonly from global whitelist")
	}

	hasClaude := false
	for _, d := range wl.Unignore.DotDirs {
		if d == ".claude" {
			hasClaude = true
			break
		}
	}
	if !hasClaude {
		t.Error("expected .claude from built-in defaults")
	}
}
