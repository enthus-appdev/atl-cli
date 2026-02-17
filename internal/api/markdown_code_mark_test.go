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

	// "italic " should have em mark
	if para.Content[0].Text != "italic " {
		t.Errorf("expected 'italic ', got %q", para.Content[0].Text)
	}
	if len(para.Content[0].Marks) != 1 || para.Content[0].Marks[0].Type != "em" {
		t.Errorf("expected [em] marks, got %v", para.Content[0].Marks)
	}

	// "code" should have ONLY code mark
	if para.Content[1].Text != "code" {
		t.Errorf("expected 'code', got %q", para.Content[1].Text)
	}
	if len(para.Content[1].Marks) != 1 || para.Content[1].Marks[0].Type != "code" {
		t.Errorf("expected [code] marks only, got %v", para.Content[1].Marks)
	}

	// " here" should have em mark
	if para.Content[2].Text != " here" {
		t.Errorf("expected ' here', got %q", para.Content[2].Text)
	}
	if len(para.Content[2].Marks) != 1 || para.Content[2].Marks[0].Type != "em" {
		t.Errorf("expected [em] marks, got %v", para.Content[2].Marks)
	}
}

func TestMarkdownToADF_CodeInsideStrikethrough(t *testing.T) {
	adf := MarkdownToADF("~~deleted `code` here~~")

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

	// "deleted " should have strike mark
	if para.Content[0].Text != "deleted " {
		t.Errorf("expected 'deleted ', got %q", para.Content[0].Text)
	}
	if len(para.Content[0].Marks) != 1 || para.Content[0].Marks[0].Type != "strike" {
		t.Errorf("expected [strike] marks, got %v", para.Content[0].Marks)
	}

	// "code" should have ONLY code mark
	if para.Content[1].Text != "code" {
		t.Errorf("expected 'code', got %q", para.Content[1].Text)
	}
	if len(para.Content[1].Marks) != 1 || para.Content[1].Marks[0].Type != "code" {
		t.Errorf("expected [code] marks only, got %v", para.Content[1].Marks)
	}

	// " here" should have strike mark
	if para.Content[2].Text != " here" {
		t.Errorf("expected ' here', got %q", para.Content[2].Text)
	}
	if len(para.Content[2].Marks) != 1 || para.Content[2].Marks[0].Type != "strike" {
		t.Errorf("expected [strike] marks, got %v", para.Content[2].Marks)
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

	// Verify paragraph content
	para := adf.Content[0]
	if para.Type != "paragraph" {
		t.Fatalf("expected paragraph, got %s", para.Type)
	}
	if len(para.Content) != 3 {
		b, _ := json.MarshalIndent(para, "", "  ")
		t.Fatalf("expected 3 inline nodes in paragraph, got %d:\n%s", len(para.Content), b)
	}

	// "Migration: Falsche " should have strong mark
	if para.Content[0].Text != "Migration: Falsche " {
		t.Errorf("expected 'Migration: Falsche ', got %q", para.Content[0].Text)
	}
	if len(para.Content[0].Marks) != 1 || para.Content[0].Marks[0].Type != "strong" {
		t.Errorf("expected [strong] marks, got %v", para.Content[0].Marks)
	}

	// "s_action" should have ONLY code mark
	if para.Content[1].Text != "s_action" {
		t.Errorf("expected 's_action', got %q", para.Content[1].Text)
	}
	if len(para.Content[1].Marks) != 1 || para.Content[1].Marks[0].Type != "code" {
		t.Errorf("expected [code] marks only, got %v", para.Content[1].Marks)
	}

	// " korrigieren" should have strong mark
	if para.Content[2].Text != " korrigieren" {
		t.Errorf("expected ' korrigieren', got %q", para.Content[2].Text)
	}
	if len(para.Content[2].Marks) != 1 || para.Content[2].Marks[0].Type != "strong" {
		t.Errorf("expected [strong] marks, got %v", para.Content[2].Marks)
	}

	// Verify code block is present
	if adf.Content[1].Type != "codeBlock" {
		t.Errorf("expected codeBlock, got %s", adf.Content[1].Type)
	}
}
