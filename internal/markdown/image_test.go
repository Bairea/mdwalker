package markdown

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderImageInlineUsesKittyPNGProtocolForKitty(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "diagram.png")
	if err := os.WriteFile(imagePath, []byte("fake png data"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("TERM_PROGRAM", "")

	got := RenderImageInline(imagePath, 40, 10)

	if !strings.Contains(got, "\x1b_Ga=T,f=100,t=f") {
		t.Fatalf("kitty image sequence did not use file transfer graphics protocol: %q", got)
	}
	if strings.Contains(got, "f=24") {
		t.Fatalf("kitty image sequence sent encoded file bytes as raw RGB: %q", got)
	}
	if strings.Contains(got, "ZmFrZSBwbmcgZGF0YQ==") {
		t.Fatalf("kitty image sequence embedded file contents instead of file path: %q", got)
	}
}

func TestRenderImageInlineUsesAbsolutePathForKittyFileTransfer(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "diagram.png")
	if err := os.WriteFile(imagePath, []byte("fake png data"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("TERM_PROGRAM", "")

	got := RenderImageInline("diagram.png", 40, 10)

	encodedAbs := base64.StdEncoding.EncodeToString([]byte(imagePath))
	if !strings.Contains(got, encodedAbs) {
		t.Fatalf("kitty image sequence did not encode absolute path %q: %q", imagePath, got)
	}
}

func TestRenderImageInlineUsesFileTransferForRealKittyImage(t *testing.T) {
	imagePath := filepath.Join("..", "..", "testdata", "screenshot.png")
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("TERM_PROGRAM", "")

	got := RenderImageInline(imagePath, 40, 10)

	if !strings.Contains(got, "\x1b_Ga=T,f=100,t=f") {
		t.Fatalf("real kitty image did not use file transfer protocol: %q", got)
	}
	if strings.Contains(got, "t=t") {
		t.Fatalf("real kitty image used temp file transfer instead of file path: %q", got)
	}
	if !strings.Contains(got, "a=T") {
		t.Fatalf("real kitty image was transmitted but not displayed: %q", got)
	}
}

func TestRenderImageInlineUsesITermProtocolForWezTerm(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "diagram.png")
	if err := os.WriteFile(imagePath, []byte("fake png data"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	got := RenderImageInline(imagePath, 40, 10)

	if !strings.Contains(got, "\x1b]1337;File=") || !strings.Contains(got, "inline=1") {
		t.Fatalf("wezterm image sequence did not use iTerm2 inline image protocol: %q", got)
	}
	if !strings.Contains(got, "doNotMoveCursor=1") {
		t.Fatalf("wezterm image sequence did not include stable cursor parameter: %q", got)
	}
	if !strings.HasSuffix(got, "\x1b\\") {
		t.Fatalf("wezterm image sequence did not use ST terminator: %q", got)
	}
}

func TestRenderImageInlineUsesITermProtocolForITerm(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "diagram.png")
	if err := os.WriteFile(imagePath, []byte("fake png data"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	t.Setenv("WEZTERM_EXECUTABLE", "")

	got := RenderImageInline(imagePath, 40, 10)

	if !strings.Contains(got, "\x1b]1337;File=") || !strings.Contains(got, "inline=1") {
		t.Fatalf("iterm image sequence did not use iTerm inline image protocol: %q", got)
	}
}

func TestExtractImagesStripsTitleFromTarget(t *testing.T) {
	refs := ExtractImages(`![pic](assets/pic%201.png "caption")`)

	if len(refs) != 1 {
		t.Fatalf("ExtractImages() returned %d refs, want 1", len(refs))
	}
	if refs[0].Path != "assets/pic 1.png" {
		t.Fatalf("ExtractImages() path = %q, want decoded path without title", refs[0].Path)
	}
}

func TestClearAllImagesKittyDeletesAll(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("TERM_PROGRAM", "")

	got := ClearAllImages()
	if got != "\x1b_Ga=d,d=a\x1b\\" {
		t.Fatalf("ClearAllImages() for kitty = %q, want delete-all sequence", got)
	}
}

func TestClearAllImagesITerm2ReturnsEmpty(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "WezTerm")
	t.Setenv("WEZTERM_EXECUTABLE", "")

	got := ClearAllImages()
	if got != "" {
		t.Fatalf("ClearAllImages() for iterm2 = %q, want empty (images are cell-tied and cleared by overwriting)", got)
	}
}

func TestClearAllImagesNoProtocolReturnsEmpty(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("WEZTERM_EXECUTABLE", "")

	got := ClearAllImages()
	if got != "" {
		t.Fatalf("ClearAllImages() for no protocol = %q, want empty", got)
	}
}
