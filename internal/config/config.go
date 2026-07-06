package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ShowTime bool
}

func Default() Config {
	return Config{
		ShowTime: false,
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
