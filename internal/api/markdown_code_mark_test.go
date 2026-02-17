package api

import (
	"encoding/json"
	"testing"
)

func TestMarkdownToADF_CodeInsideBold(t *testing.T) {
	adf := MarkdownToADF("**Bold with `code` inside**")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(adf.Content))
	}

	para := adf.Content[0]
	if para.Type != "paragraph" {
		t.Fatalf("expected paragraph, got %s", para.Type)
	}

	if len(para.Content) != 3 {
		b, _ := json.MarshalIndent(adf, "", "  ")
		t.Fatalf("expected 3 inline nodes, got %d:\n%s", len(para.Content), b)
	}

	// "Bold with " should have strong mark
	if para.Content[0].Text != "Bold with " {
		t.Errorf("expected 'Bold with ', got %q", para.Content[0].Text)
	}
	if len(para.Content[0].Marks) != 1 || para.Content[0].Marks[0].Type != "strong" {
		t.Errorf("expected [strong] marks, got %v", para.Content[0].Marks)
	}

	// "code" should have ONLY code mark (not strong+code)
	if para.Content[1].Text != "code" {
		t.Errorf("expected 'code', got %q", para.Content[1].Text)
	}
	if len(para.Content[1].Marks) != 1 || para.Content[1].Marks[0].Type != "code" {
		t.Errorf("expected [code] marks only, got %v", para.Content[1].Marks)
	}

	// " inside" should have strong mark
	if para.Content[2].Text != " inside" {
		t.Errorf("expected ' inside', got %q", para.Content[2].Text)
	}
	if len(para.Content[2].Marks) != 1 || para.Content[2].Marks[0].Type != "strong" {
		t.Errorf("expected [strong] marks, got %v", para.Content[2].Marks)
	}
}

func TestMarkdownToADF_CodeInsideItalic(t *testing.T) {
	adf := MarkdownToADF("*italic `code` here*")

	para := adf.Content[0]
	for _, c := range para.Content {
		if hasCodeMark(c) {
			if len(c.Marks) != 1 {
				t.Errorf("code node should have only code mark, got %v", c.Marks)
			}
		}
	}
}

func TestMarkdownToADF_CodeInsideStrikethrough(t *testing.T) {
	adf := MarkdownToADF("~~deleted `code` here~~")

	para := adf.Content[0]
	for _, c := range para.Content {
		if hasCodeMark(c) {
			if len(c.Marks) != 1 {
				t.Errorf("code node should have only code mark, got %v", c.Marks)
			}
		}
	}
}

func TestMarkdownToADF_BoldWithCodeAndCodeBlock(t *testing.T) {
	// This was the exact combination that caused INVALID_INPUT from Jira
	md := "**Migration: Falsche `s_action` korrigieren**\n\n```sql\nSELECT 1\n```"
	adf := MarkdownToADF(md)

	if len(adf.Content) != 2 {
		b, _ := json.MarshalIndent(adf, "", "  ")
		t.Fatalf("expected 2 blocks (paragraph + codeBlock), got %d:\n%s", len(adf.Content), b)
	}

	// Verify no code node has additional marks
	para := adf.Content[0]
	for _, c := range para.Content {
		if hasCodeMark(c) && len(c.Marks) > 1 {
			t.Errorf("code node should have only code mark, got %v", c.Marks)
		}
	}

	// Verify code block is present
	if adf.Content[1].Type != "codeBlock" {
		t.Errorf("expected codeBlock, got %s", adf.Content[1].Type)
	}
}
