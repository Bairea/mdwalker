package outline

import "testing"

func TestParseIgnoresHashHeadingsInsideFencedCodeBlocks(t *testing.T) {
	content := "# Real\n\n```go\n# not a markdown heading\nfmt.Println(\"# nope\")\n```\n\n## Child\n"

	headings := Parse(content)

	if len(headings) != 2 {
		t.Fatalf("Parse() headings = %#v, want only real markdown headings", headings)
	}
	if headings[0].Text != "Real" || headings[1].Text != "Child" {
		t.Fatalf("Parse() included code fence content: %#v", headings)
	}
}
