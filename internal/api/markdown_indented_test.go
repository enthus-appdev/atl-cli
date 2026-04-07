package api

import (
	"testing"
	"time"
)

// TestMarkdownToADF_IndentedCodeFenceInList reproduces a hang where an indented
// fenced code block inside an ordered/bulleted list caused parseParagraph to
// return consumed=0, leading to an infinite loop in parseBlocks.
//
// The fix has two parts:
//  1. parseBlocks must trim before checking for "```" so indented fences are
//     parsed as code blocks (not as paragraphs).
//  2. parseParagraph must guarantee consumed >= 1 as a defensive backstop.
func TestMarkdownToADF_IndentedCodeFenceInList(t *testing.T) {
	input := "1. Verify the token works:\n" +
		"    ```\n" +
		"    curl https://api.example.com/v1/me\n" +
		"    ```\n" +
		"2. Next step.\n"

	done := make(chan *ADF, 1)
	go func() {
		done <- MarkdownToADF(input)
	}()

	select {
	case adf := <-done:
		if adf == nil || adf.Type != "doc" {
			t.Fatalf("expected doc ADF, got %+v", adf)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("MarkdownToADF hung on indented code fence inside ordered list")
	}
}

// TestMarkdownToADF_IndentedCodeFenceTopLevel verifies that an indented code
// fence at the top level (4-space-indented ```) does not hang. Even if the
// dispatcher does not recognize it as a code block, parseParagraph must
// consume at least one line.
func TestMarkdownToADF_IndentedCodeFenceTopLevel(t *testing.T) {
	input := "Header\n\n    ```\n    code\n    ```\n\nFooter\n"

	done := make(chan *ADF, 1)
	go func() {
		done <- MarkdownToADF(input)
	}()

	select {
	case adf := <-done:
		if adf == nil || adf.Type != "doc" {
			t.Fatalf("expected doc ADF, got %+v", adf)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("MarkdownToADF hung on indented code fence")
	}
}

// TestParseParagraph_NeverReturnsZero is a defensive guarantee that
// parseParagraph always consumes at least one line. Otherwise the parseBlocks
// loop can spin forever.
func TestParseParagraph_NeverReturnsZero(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"indented backticks", "    ```"},
		{"indented heading marker", "    # not a heading"},
		{"indented quote", "    > not a quote"},
		{"indented bullet", "    - not a bullet"},
		{"indented ordered", "    1. not a list"},
		{"indented hr", "    ---"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := []string{tt.input}
			_, consumed := parseParagraph(lines, 0)
			if consumed < 1 {
				t.Errorf("parseParagraph returned consumed=%d for %q (must be >= 1 to prevent infinite loops)", consumed, tt.input)
			}
		})
	}
}
