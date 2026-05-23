package terminal

import (
	"os"
	"os/exec"
	"strings"
)

type Capability int

const (
	CapNone     Capability = iota
	CapHalfblock
	CapKitty
	CapITerm2
)

type Info struct {
	ImageCap Capability
	IsTmux   bool
}

func Detect() Info {
	info := Info{}
	term := os.Getenv("TERM")
	termProg := os.Getenv("TERM_PROGRAM")

	if os.Getenv("TMUX") != "" {
		info.IsTmux = true
	}

	switch {
	case termProg == "kitty" || strings.Contains(term, "kitty"):
		info.ImageCap = CapKitty
	case termProg == "iTerm.app" || termProg == "WezTerm":
		info.ImageCap = CapITerm2
	default:
		if chafaAvailable() {
			info.ImageCap = CapHalfblock
		}
	}

	return info
}

func chafaAvailable() bool {
	_, err := exec.LookPath("chafa")
	if err == nil {
		return true
	}
	_, err = exec.LookPath("viu")
	return err == nil
}
