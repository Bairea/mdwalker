package mermaid

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Diagram struct {
	Content string
	Hash    string
}

func CacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "mdwalker", "mermaid")
}

func Render(content string) (string, error) {
	h := sha256.Sum256([]byte(content))
	hash := fmt.Sprintf("%x", h[:16])
	cacheDir := CacheDir()
	os.MkdirAll(cacheDir, 0755)
	pngPath := filepath.Join(cacheDir, hash+".png")

	if _, err := os.Stat(pngPath); os.IsNotExist(err) {
		mmdcPath := "mmdc"
		if custom := os.Getenv("MDWALKER_MMDC"); custom != "" {
			mmdcPath = custom
		}
		tmpFile := filepath.Join(cacheDir, hash+".mmd")
		os.WriteFile(tmpFile, []byte(content), 0644)
		defer os.Remove(tmpFile)
		cmd := exec.Command(mmdcPath, "-i", tmpFile, "-o", pngPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("mmdc failed: %w", err)
		}
	}

	return pngPath, nil
}

func CleanCache() {
	cacheDir := CacheDir()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, e := range entries {
		info, _ := e.Info()
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(cacheDir, e.Name()))
		}
	}
}
