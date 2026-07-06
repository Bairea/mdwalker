package markdown

import (
	"net/url"
	"regexp"
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

func RenderImagePlaceholder(ref ImageRef) string {
	return "[Image: " + ref.Path + "]"
}
