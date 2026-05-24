package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewImageFileUsesImagePlaceholder(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "diagram.png"), []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 20)
	if err := m.LoadFile(root, "diagram.png"); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(m.View(), "[Image: diagram.png]") {
		t.Fatalf("image preview did not render an image placeholder: %q", m.View())
	}
}

func TestPreviewImageFileRendersTerminalImageWhenRendererAvailable(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "diagram.png")
	if err := os.WriteFile(imagePath, []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "chafa.log")
	installFakeChafa(t, root, logPath)

	m := New()
	m.SetSize(80, 20)
	if err := m.LoadFile(root, "diagram.png"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "IMAGE_RENDER:") {
		t.Fatalf("image file did not render through terminal image renderer: %q", view)
	}
	if got := strings.TrimSpace(readFile(t, logPath)); got != imagePath {
		t.Fatalf("image renderer received path %q, want %q", got, imagePath)
	}
}

func TestPreviewMarkdownImageResolvesRelativePathAndRenders(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "docs", "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(root, "docs", "doc.md")
	imagePath := filepath.Join(root, "docs", "assets", "pic.png")
	if err := os.WriteFile(docPath, []byte("before\n\n![pic](assets/pic.png)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "chafa.log")
	installFakeChafa(t, root, logPath)

	m := New()
	m.SetSize(100, 20)
	if err := m.LoadFile(root, "docs/doc.md"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "IMAGE_RENDER:") {
		t.Fatalf("markdown image did not resolve and render relative image path %q: %q", imagePath, view)
	}
	if got := strings.TrimSpace(readFile(t, logPath)); got != imagePath {
		t.Fatalf("image renderer received path %q, want resolved path %q", got, imagePath)
	}
}

func TestPreviewReplacesMermaidFenceWithRenderablePlaceholder(t *testing.T) {
	t.Setenv("MDWALKER_MMDC", "definitely-missing-mmdc")
	root := t.TempDir()
	t.Setenv("HOME", root)
	content := "before\n\n```mermaid\ngraph TD\nA-->MissingMMDC\n```\n\nafter\n"
	if err := os.WriteFile(filepath.Join(root, "diagram.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 20)
	if err := m.LoadFile(root, "diagram.md"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "[Mermaid:") {
		t.Fatalf("mermaid fence was not replaced with render feedback: %q", view)
	}
	if strings.Contains(view, "```mermaid") {
		t.Fatalf("mermaid source fence leaked into rendered preview: %q", view)
	}
}

func TestPreviewRendersMermaidThroughDiagramAndImagePipeline(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	installFakeChafa(t, root, filepath.Join(root, "chafa.log"))
	installFakeMMDC(t, root)

	content := "before\n\n```mermaid\nflowchart TD\nA-->B\n```\n\nafter\n"
	if err := os.WriteFile(filepath.Join(root, "diagram.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(100, 20)
	if err := m.LoadFile(root, "diagram.md"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "IMAGE_RENDER:") {
		t.Fatalf("mermaid diagram was not rendered through the image pipeline: %q", view)
	}
	if strings.Contains(view, "flowchart TD") || strings.Contains(view, "```mermaid") {
		t.Fatalf("mermaid source leaked into rendered preview: %q", view)
	}
}

func TestToggleFoldFoldsCurrentSectionWhenCursorIsInsideIt(t *testing.T) {
	root := t.TempDir()
	content := "# First\nintro\n## Child\nchild body\n# Second\nsecond body\n"
	if err := os.WriteFile(filepath.Join(root, "headings.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 20)
	if err := m.LoadFile(root, "headings.md"); err != nil {
		t.Fatal(err)
	}

	m.ToggleFold(1)

	for _, hidden := range []int{1, 2, 3} {
		if m.IsLineVisible(hidden) {
			t.Fatalf("line %d remained visible after folding the current section", hidden)
		}
	}
	if !m.IsLineVisible(4) {
		t.Fatalf("next top-level heading should remain visible")
	}
}

func installFakeChafa(t *testing.T, dir, logPath string) {
	t.Helper()
	writeExecutable(t, filepath.Join(dir, "chafa"), `#!/bin/sh
last=""
for arg do
  last="$arg"
done
if [ -n "$FAKE_CHAFA_LOG" ]; then
  printf '%s\n' "$last" > "$FAKE_CHAFA_LOG"
fi
printf 'IMAGE_RENDER:%s\n' "$last"
`)
	t.Setenv("FAKE_CHAFA_LOG", logPath)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func installFakeMMDC(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "mmdc")
	writeExecutable(t, path, `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    out="$1"
  fi
  shift
done
printf 'fake png' > "$out"
`)
	t.Setenv("MDWALKER_MMDC", path)
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
