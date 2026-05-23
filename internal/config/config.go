package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ImageProtocol string // auto | kitty | halfblock | off
	MermaidMode   string // auto | code | browser
	MmdcPath      string
}

func Default() Config {
	return Config{
		ImageProtocol: "auto",
		MermaidMode:   "auto",
		MmdcPath:      "mmdc",
	}
}

func Load() Config {
	cfg := Default()
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}
	path := filepath.Join(home, ".config", "mdwalker", "config.toml")
	_ = path
	return cfg
}
