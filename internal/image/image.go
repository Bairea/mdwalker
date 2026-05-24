package image

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type ImageRef struct {
	Alt  string
	Path string
	Line int
}

var imgRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

func Extract(content string) []ImageRef {
	var refs []ImageRef
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		matches := imgRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			refs = append(refs, ImageRef{
				Alt:  m[1],
				Path: m[2],
				Line: i,
			})
		}
	}
	return refs
}

func OpenWithDefault(path string) error {
	return exec.Command("open", path).Run()
}

func RenderPlaceholder(ref ImageRef) string {
	return "[Image: " + ref.Path + "]  press i to open"
}

func ToHalfblock(path string, width, height int) (string, error) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	cmd := exec.Command("chafa", "--symbols", "block", "--size", strconv.Itoa(width)+"x"+strconv.Itoa(height), path)
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("viu", "--width", strconv.Itoa(width), "--height", strconv.Itoa(height), path)
		out, err = cmd.Output()
	}
	return string(out), err
}

func TerminalSupportsImages() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	term := os.Getenv("TERM")
	if term == "ghostty" || term == "xterm-ghostty" || term == "wezterm" || strings.Contains(term, "kitty") {
		return true
	}
	prog := os.Getenv("TERM_PROGRAM")
	if prog == "iTerm.app" || prog == "WezTerm" {
		return true
	}
	return false
}

func RenderInline(path string, width, height int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(os.Getenv("TERM"), "kitty") ||
		os.Getenv("TERM") == "ghostty" || os.Getenv("TERM") == "xterm-ghostty" ||
		os.Getenv("TERM") == "wezterm" {
		return renderKitty(data, width, height)
	}

	return renderITerm2(data, width, height)
}

func renderITerm2(data []byte, width, height int) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("\x1b]1337;File=width=%d;height=%d;preserveAspectRatio=1;inline=1:%s\x07",
		width, height, encoded)
}

func renderKitty(data []byte, width, height int) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	const chunkSize = 4096
	pixelW := width * 10
	pixelH := height * 20

	var b strings.Builder
	for i := 0; i < len(encoded); {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		more := 0
		if end < len(encoded) {
			more = 1
		}
		if i == 0 {
			b.WriteString(fmt.Sprintf("\x1b_Ga=T,f=24,t=t,s=%d,v=%d,m=%d;%s\x1b\\",
				pixelW, pixelH, more, chunk))
		} else {
			b.WriteString(fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", more, chunk))
		}
		i = end
	}
	return b.String()
}
