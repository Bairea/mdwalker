package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type WhitelistUnignore struct {
	DotDirs []string `yaml:"dot_dirs"`
	Paths   []string `yaml:"paths"`
}

type WhitelistConfig struct {
	Unignore    WhitelistUnignore `yaml:"unignore"`
	SkipSubdirs []string          `yaml:"skip_subdirs"`
}

func defaultWhitelist() WhitelistConfig {
	return WhitelistConfig{
		Unignore: WhitelistUnignore{
			DotDirs: []string{
				".claude", ".agents", ".pi", ".trae", ".omx",
				".adal", ".augment", ".bob", ".codebuddy", ".commandcode",
				".continue", ".cortex", ".crush", ".factory", ".goose",
				".iflow", ".junie", ".kilocode", ".kiro", ".kode",
				".mcpjam", ".mux", ".neovate", ".openhands", ".pochi",
				".qoder", ".qwen", ".roo", ".vibe", ".windsurf", ".zencoder",
			},
			Paths: []string{"docs/superpowers"},
		},
		SkipSubdirs: []string{"*/skills"},
	}
}

func dedupStrings(v []string) []string {
	seen := make(map[string]bool, len(v))
	var out []string
	for _, s := range v {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func mergeWhitelist(base, overlay WhitelistConfig) WhitelistConfig {
	base.Unignore.DotDirs = append(base.Unignore.DotDirs, overlay.Unignore.DotDirs...)
	base.Unignore.Paths = append(base.Unignore.Paths, overlay.Unignore.Paths...)
	base.SkipSubdirs = append(base.SkipSubdirs, overlay.SkipSubdirs...)
	base.Unignore.DotDirs = dedupStrings(base.Unignore.DotDirs)
	base.Unignore.Paths = dedupStrings(base.Unignore.Paths)
	base.SkipSubdirs = dedupStrings(base.SkipSubdirs)
	return base
}

func loadYAMLWhitelist(path string) (WhitelistConfig, error) {
	var wl WhitelistConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return wl, err
	}
	err = yaml.Unmarshal(data, &wl)
	return wl, err
}

func LoadWhitelist() WhitelistConfig {
	wl := defaultWhitelist()

	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".config", "mdwalker", "whitelist.yaml")
		if global, err := loadYAMLWhitelist(globalPath); err == nil {
			wl = mergeWhitelist(wl, global)
		}
	}

	if project, err := loadYAMLWhitelist("mdwalker-whitelist.yaml"); err == nil {
		wl = mergeWhitelist(wl, project)
	}

	return wl
}
