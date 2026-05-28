package markdown

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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

func ExtractImages(content string) []ImageRef {
	var refs []ImageRef
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		matches := imgRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			path := parseImageTarget(m[2])
			refs = append(refs, ImageRef{
				Alt:  m[1],
				Path: path,
				Line: i,
			})
		}
	}
	return refs
}

func parseImageTarget(target string) string {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "<") {
		if end := strings.Index(target, ">"); end >= 0 {
			return decodeImagePath(target[1:end])
		}
	}
	for i, r := range target {
		if r == ' ' || r == '\t' || r == '\n' {
			return decodeImagePath(target[:i])
		}
	}
	return decodeImagePath(target)
}

func decodeImagePath(path string) string {
	path = strings.TrimSpace(path)
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return decoded
}

func OpenImage(path string) error {
	return exec.Command("open", path).Run()
}

func RenderImagePlaceholder(ref ImageRef) string {
	return "[Image: " + ref.Path + "]  press i to open"
}

func HasTerminalImageSequence(s string) bool {
	return strings.Contains(s, "\x1b]1337;File=") || strings.Contains(s, "\x1b_G")
}

func ClearAllImages() string {
	switch imageProtocol() {
	case "kitty":
		return "\x1b_Ga=d,d=a\x1b\\"
	case "iterm2":
		return ""
	}
	return ""
}

func ImageToHalfblock(path string, width, height int) (string, error) {
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
	return imageProtocol() != ""
}

func RenderImageInline(path string, width, height int) string {
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}

	switch imageProtocol() {
	case "kitty":
		return renderKitty(path, width, height)
	case "iterm2":
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		return renderITerm2(data, width, height, isWezTerm())
	}

	return ""
}

func imageProtocol() string {
	if os.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(os.Getenv("TERM"), "kitty") {
		return "kitty"
	}
	if isWezTerm() {
		return "iterm2"
	}
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return "iterm2"
	}
	if os.Getenv("TERM") == "ghostty" || os.Getenv("TERM") == "xterm-ghostty" {
		return "kitty"
	}
	return ""
}

func isWezTerm() bool {
	return os.Getenv("TERM_PROGRAM") == "WezTerm" || os.Getenv("WEZTERM_EXECUTABLE") != "" || os.Getenv("TERM") == "wezterm"
}

func renderITerm2(data []byte, width, height int, stableCursor bool) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	params := fmt.Sprintf("width=%d;height=%d;preserveAspectRatio=1;inline=1", width, height)
	if stableCursor {
		params += ";doNotMoveCursor=1"
	}
	return fmt.Sprintf("\x1b]1337;File=%s:%s\x1b\\", params, encoded)
}

func renderKitty(path string, width, height int) string {
	return renderKittyFile(path, width, height)
}

func renderKittyFile(path string, width, height int) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(path))
	return fmt.Sprintf("\x1b_Ga=T,f=100,t=f,c=%d,r=%d,q=2;%s\x1b\\", width, height, encoded)
}
