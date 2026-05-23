package image

import (
	"os/exec"
	"regexp"
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

func ToHalfblock(path string) (string, error) {
	cmd := exec.Command("chafa", "--symbols", "block", "--size", "80x20", path)
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("viu", "--width", "80", "--height", "20", path)
		out, err = cmd.Output()
	}
	return string(out), err
}
