package preview

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPreviewImageFileUsesImagePlaceholder(t *testing.T) {
	disableNativeImageProtocol(t)
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
	disableNativeImageProtocol(t)
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
	disableNativeImageProtocol(t)
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

func TestPreviewMarkdownImageWithTitleRendersNativeImage(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "docs", "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "doc.md"), []byte("before\n\n![pic](assets/pic.png \"caption\")\n\nafter\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "assets", "pic.png"), []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New()
	m.SetSize(100, 20)
	if err := m.LoadFile(root, "docs/doc.md"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "\x1b]1337;File=") {
		t.Fatalf("markdown image with title did not render native image via iTerm2 protocol: %q", view)
	}
	if strings.Contains(view, "[Image: assets/pic.png") {
		t.Fatalf("markdown image with title fell back to placeholder: %q", view)
	}
}

func TestPreviewMarkdownFixtureRendersReferencedImageNatively(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New()
	m.SetSize(100, 20)
	if err := m.LoadFile(root, "images.md"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "\x1b]1337;File=") {
		t.Fatalf("testdata/images.md referenced image did not render natively via iTerm2 protocol: %q", view)
	}
	if !strings.Contains(view, "inline=1") {
		t.Fatalf("testdata/images.md referenced image did not include inline=1: %q", view)
	}
}

func TestLightPreviewRendersNativeImageWithoutExternalRenderer(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte("# Images\n\n![pic](pic.png)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pic.png"), []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "chafa.log")
	installFakeChafa(t, root, logPath)
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New()
	m.SetSize(100, 20)
	if err := m.LoadFileLight(root, "doc.md"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "\x1b]1337;File=") {
		t.Fatalf("light preview did not render native image via iTerm2 protocol: %q", view)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("light preview invoked external image renderer; err=%v", err)
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
	disableNativeImageProtocol(t)
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

func TestScrollToLinePlacesRenderedHeadingAtTop(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("# Top\n\n")
	for i := 0; i < 20; i++ {
		b.WriteString("filler paragraph with enough text to render\n\n")
	}
	targetLine := strings.Count(b.String(), "\n")
	b.WriteString("## Target Heading\nbody\n")
	for i := 0; i < 20; i++ {
		b.WriteString("after target filler\n\n")
	}
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 8)
	if err := m.LoadFile(root, "doc.md"); err != nil {
		t.Fatal(err)
	}

	m.ScrollToLine(targetLine)

	firstLine := strings.Split(m.View(), "\n")[0]
	if !strings.Contains(firstLine, "Target Heading") {
		t.Fatalf("ScrollToLine did not place rendered heading at top; first line=%q view=%q", firstLine, m.View())
	}
}

func TestScrollToLineUsesRenderedOffsetNotSourceLineApproximation(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("# Top\n\n")
	for i := 0; i < 8; i++ {
		b.WriteString("- list item before target\n")
		b.WriteString("  - nested item before target\n")
	}
	targetLine := strings.Count(b.String(), "\n")
	b.WriteString("## Precise Target\n")
	b.WriteString("body that should not be the first visible line\n")
	for i := 0; i < 8; i++ {
		b.WriteString("after target filler\n\n")
	}
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 8)
	if err := m.LoadFile(root, "doc.md"); err != nil {
		t.Fatal(err)
	}

	m.ScrollToLine(targetLine)

	firstLine := strings.Split(m.View(), "\n")[0]
	if !strings.Contains(firstLine, "Precise Target") {
		t.Fatalf("ScrollToLine first visible line = %q, want target heading; view=%q", firstLine, m.View())
	}
	if strings.Contains(firstLine, "body that should not") {
		t.Fatalf("ScrollToLine skipped past target heading: %q", m.View())
	}
}

func TestPreviewImageRendererUsesViewportBoundedSize(t *testing.T) {
	disableNativeImageProtocol(t)
	root := t.TempDir()
	imagePath := filepath.Join(root, "diagram.png")
	if err := os.WriteFile(imagePath, []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "chafa.log")
	installFakeChafa(t, root, logPath)

	m := New()
	m.SetSize(42, 12)
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

func TestPreviewImageFilePreservesNativeImageSequence(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "diagram.png"), []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New()
	m.SetSize(80, 20)
	if err := m.LoadFile(root, "diagram.png"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "\x1b]1337;File=") {
		t.Fatalf("preview corrupted native iTerm2 image sequence: %q", view)
	}
}

func TestPreviewImageFileUsesNativeImageWithoutExternalRenderer(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "diagram.png"), []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "chafa.log")
	installFakeChafa(t, root, logPath)
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New()
	m.SetSize(100, 20)
	if err := m.LoadFile(root, "diagram.png"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if !strings.Contains(view, "\x1b]1337;File=") {
		t.Fatalf("image file did not render through native iTerm2 protocol: %q", view)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("image file invoked external image renderer; err=%v", err)
	}
}

func TestImageSwitchClearsPreviousImageSequence(t *testing.T) {
	root := t.TempDir()
	img1 := filepath.Join(root, "alpha.png")
	img2 := filepath.Join(root, "beta.png")
	if err := os.WriteFile(img1, []byte("img1-data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(img2, []byte("img2-data"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New()
	m.SetSize(80, 20)

	if err := m.LoadFile(root, "alpha.png"); err != nil {
		t.Fatal(err)
	}
	view1 := m.View()
	if !strings.Contains(view1, "\x1b]1337;File=") {
		t.Fatalf("first image did not render iTerm2 sequence: %q", view1)
	}

	if err := m.LoadFile(root, "beta.png"); err != nil {
		t.Fatal(err)
	}
	view2 := m.View()
	if !strings.Contains(view2, "\x1b]1337;File=") {
		t.Fatalf("second image did not render iTerm2 sequence: %q", view2)
	}

	if strings.Contains(view2, "img1-data") {
		t.Fatalf("second view still contains first image data (residual): %q", view2)
	}
}

func TestMarkdownImageSwitchClearsPreviousInlineImage(t *testing.T) {
	root := t.TempDir()
	doc1 := filepath.Join(root, "doc1.md")
	doc2 := filepath.Join(root, "doc2.md")
	img1 := filepath.Join(root, "pic1.png")
	img2 := filepath.Join(root, "pic2.png")
	if err := os.WriteFile(doc1, []byte("# Doc1\n\n![pic1](pic1.png)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doc2, []byte("# Doc2\n\n![pic2](pic2.png)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(img1, []byte("img1-raw-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(img2, []byte("img2-raw-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	m := New()
	m.SetSize(100, 20)

	if err := m.LoadFile(root, "doc1.md"); err != nil {
		t.Fatal(err)
	}
	view1 := m.View()
	if !strings.Contains(view1, "\x1b]1337;File=") {
		t.Fatalf("doc1 image did not render iTerm2 sequence: %q", view1)
	}

	if err := m.LoadFile(root, "doc2.md"); err != nil {
		t.Fatal(err)
	}
	view2 := m.View()
	if !strings.Contains(view2, "\x1b]1337;File=") {
		t.Fatalf("doc2 image did not render iTerm2 sequence: %q", view2)
	}

	if strings.Contains(view2, "img1-raw-bytes") {
		t.Fatalf("doc2 view still contains doc1 image data (residual): %q", view2)
	}
}

func TestKittyImageSwitchEmitsDeleteAll(t *testing.T) {
	root := t.TempDir()
	img1 := filepath.Join(root, "alpha.png")
	img2 := filepath.Join(root, "beta.png")
	if err := os.WriteFile(img1, []byte("img1-data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(img2, []byte("img2-data"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("TERM_PROGRAM", "")

	m := New()
	m.SetSize(80, 20)

	if err := m.LoadFile(root, "alpha.png"); err != nil {
		t.Fatal(err)
	}
	view1 := m.View()
	if !strings.Contains(view1, "\x1b_G") {
		t.Fatalf("first image did not render Kitty sequence: %q", view1)
	}

	if err := m.LoadFile(root, "beta.png"); err != nil {
		t.Fatal(err)
	}
	view2 := m.View()
	if !strings.Contains(view2, "\x1b_G") {
		t.Fatalf("second image did not render Kitty sequence: %q", view2)
	}
	if !strings.Contains(view2, "\x1b_Ga=d,d=a\x1b\\") {
		t.Fatalf("Kitty view missing delete-all before new image: %q", view2)
	}
}

func TestGlamourPreservesMediaPlaceholder(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte("# Title\n\n![pic](pic.png)\n\nAfter.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pic.png"), []byte("not-real-image"), 0644); err != nil {
		t.Fatal(err)
	}
	disableNativeImageProtocol(t)

	m := New()
	m.SetSize(100, 20)
	if err := m.LoadFile(root, "doc.md"); err != nil {
		t.Fatal(err)
	}

	view := m.View()
	if strings.Contains(view, "![pic](pic.png)") {
		t.Fatalf("raw markdown image syntax leaked into view: %q", view)
	}
	if !strings.Contains(view, "[Image: pic.png]") {
		t.Fatalf("image placeholder not rendered: %q", view)
	}
}

func disableNativeImageProtocol(t *testing.T) {
	t.Helper()
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("WEZTERM_EXECUTABLE", "")
}

func installFakeChafa(t *testing.T, dir, logPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		writeExecutable(t, filepath.Join(dir, "chafa.cmd"), `@echo off
set last=
:loop
if "%~1"=="" goto done
set last=%~1
shift
goto loop
:done
if not "%FAKE_CHAFA_LOG%"=="" echo %last%>"%FAKE_CHAFA_LOG%"
echo IMAGE_RENDER:%last%
`)
		t.Setenv("FAKE_CHAFA_LOG", logPath)
		t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		return
	}
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
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "mmdc.cmd")
		writeExecutable(t, path, `@echo off
set out=
:loop
if "%~1"=="" goto done
if "%~1"=="-o" goto foundout
shift
goto loop
:foundout
shift
set out=%~1
shift
goto loop
:done
echo fake png>"%out%"
`)
		t.Setenv("MDWALKER_MMDC", path)
		return
	}
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
