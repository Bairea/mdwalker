package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanIncludesMarkdownAndPreviewableImages(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"notes.md", "diagram.png", "photo.jpg", "ignored.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := Scan(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, entry := range entries {
		got[entry.Path] = true
	}
	for _, want := range []string{"notes.md", "diagram.png", "photo.jpg"} {
		if !got[want] {
			t.Fatalf("Scan() missing previewable file %q; got paths %#v", want, got)
		}
	}
	if got["ignored.txt"] {
		t.Fatalf("Scan() included unsupported file ignored.txt")
	}
}
