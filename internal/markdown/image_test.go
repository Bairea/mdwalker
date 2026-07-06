package markdown

import (
	"testing"
)

func TestExtractImagesStripsTitleFromTarget(t *testing.T) {
	refs := ExtractImages(`![pic](assets/pic%201.png "caption")`)

	if len(refs) != 1 {
		t.Fatalf("ExtractImages() returned %d refs, want 1", len(refs))
	}
	if refs[0].Path != "assets/pic 1.png" {
		t.Fatalf("ExtractImages() path = %q, want decoded path without title", refs[0].Path)
	}
}
