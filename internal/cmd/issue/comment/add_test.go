package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

func TestBuildReplyADF_MentionsAuthorAndLinksOriginalWithoutQuoting(t *testing.T) {
	body := &api.ADF{
		Type:    "doc",
		Version: 1,
		Content: []api.ADFContent{{
			Type:    "paragraph",
			Content: []api.ADFContent{{Type: "text", Text: "The actual reply"}},
		}},
	}
	author := &api.User{AccountID: "account-123", DisplayName: "Alex Example"}

	doc := buildReplyADF("example.atlassian.net", "PROJ-42", "987", author, body)

	if len(doc.Content) != 2 {
		t.Fatalf("expected reply header and body only, got %d top-level nodes", len(doc.Content))
	}
	if doc.Content[0].Type != "paragraph" {
		t.Fatalf("expected reply header paragraph, got %q", doc.Content[0].Type)
	}
	if len(doc.Content[0].Content) != 3 {
		t.Fatalf("expected mention, separator, and link, got %+v", doc.Content[0].Content)
	}

	mention := doc.Content[0].Content[0]
	if mention.Type != "mention" || mention.Attrs == nil {
		t.Fatalf("expected real ADF mention, got %+v", mention)
	}
	if mention.Attrs.ID != "account-123" || mention.Attrs.Text != "@Alex Example" {
		t.Errorf("unexpected mention attrs: %+v", mention.Attrs)
	}

	link := doc.Content[0].Content[2]
	if link.Text != "Replying to comment 987" || len(link.Marks) != 1 || link.Marks[0].Type != "link" {
		t.Fatalf("expected focused-comment link, got %+v", link)
	}
	if got := link.Marks[0].Attrs.Href; got != "https://example.atlassian.net/browse/PROJ-42?focusedCommentId=987" {
		t.Errorf("unexpected reply URL: %q", got)
	}
	if got := doc.Content[1].Content[0].Text; got != "The actual reply" {
		t.Errorf("expected reply body unchanged, got %q", got)
	}
	for _, node := range doc.Content {
		if node.Type == "blockquote" {
			t.Fatal("reply must not copy the original comment into a blockquote")
		}
	}
}

func TestBuildReplyADF_StillLinksWhenAuthorUnavailable(t *testing.T) {
	doc := buildReplyADF("example.atlassian.net", "PROJ-42", "987", nil, nil)

	if len(doc.Content) != 1 || len(doc.Content[0].Content) != 1 {
		t.Fatalf("expected link-only header, got %+v", doc.Content)
	}
	link := doc.Content[0].Content[0]
	if link.Type != "text" || len(link.Marks) != 1 || link.Marks[0].Type != "link" {
		t.Fatalf("expected focused-comment link without a mention, got %+v", link)
	}
}

func TestNewCmdAdd_BodyFileFlag(t *testing.T) {
	ios := iostreams.Test()
	cmd := NewCmdAdd(ios)

	t.Run("body-file flag exists", func(t *testing.T) {
		f := cmd.Flags().Lookup("body-file")
		if f == nil {
			t.Fatal("expected --body-file flag to exist")
		}
	})

	t.Run("body flag shorthand", func(t *testing.T) {
		f := cmd.Flags().ShorthandLookup("b")
		if f == nil {
			t.Fatal("expected -b shorthand for --body")
		}
	})
}

func TestNewCmdAdd_Validation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		flags   map[string]string
		wantErr string
	}{
		{
			name:    "body and body-file mutually exclusive",
			args:    []string{"PROJ-1"},
			flags:   map[string]string{"body": "text", "body-file": "file.md"},
			wantErr: "--body and --body-file are mutually exclusive",
		},
		{
			name:    "neither body nor body-file",
			args:    []string{"PROJ-1"},
			flags:   map[string]string{},
			wantErr: "--body or --body-file is required",
		},
		{
			name:    "body-file does not exist",
			args:    []string{"PROJ-1"},
			flags:   map[string]string{"body-file": "/nonexistent/path/file.md"},
			wantErr: "failed to read body file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios := iostreams.Test()
			cmd := NewCmdAdd(ios)

			for k, v := range tt.flags {
				if err := cmd.Flags().Set(k, v); err != nil {
					t.Fatalf("failed to set flag %s: %v", k, err)
				}
			}

			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNewCmdAdd_BodyFileReadsContent(t *testing.T) {
	tmpDir := t.TempDir()
	bodyFile := filepath.Join(tmpDir, "comment.md")
	content := "Hello **world** from file"
	if err := os.WriteFile(bodyFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	ios := iostreams.Test()
	cmd := NewCmdAdd(ios)

	if err := cmd.Flags().Set("body-file", bodyFile); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	cmd.SetArgs([]string{"PROJ-1"})

	// The command will fail at the API call (no config), but body-file reading happens first.
	// If it fails with "failed to read body file", the file reading is broken.
	err := cmd.Execute()
	if err == nil {
		t.Skip("command succeeded unexpectedly (API config available)")
	}
	if strings.Contains(err.Error(), "failed to read body file") {
		t.Errorf("body file should have been read successfully, got: %v", err)
	}
}

func TestNewCmdEdit_Validation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		flags   map[string]string
		wantErr string
	}{
		{
			name:    "id required",
			args:    []string{"PROJ-1"},
			flags:   map[string]string{"body": "text"},
			wantErr: "--id is required",
		},
		{
			name:    "body and body-file mutually exclusive",
			args:    []string{"PROJ-1"},
			flags:   map[string]string{"id": "123", "body": "text", "body-file": "file.md"},
			wantErr: "--body and --body-file are mutually exclusive",
		},
		{
			name:    "neither body nor body-file",
			args:    []string{"PROJ-1"},
			flags:   map[string]string{"id": "123"},
			wantErr: "--body or --body-file is required",
		},
		{
			name:    "body-file does not exist",
			args:    []string{"PROJ-1"},
			flags:   map[string]string{"id": "123", "body-file": "/nonexistent/path/file.md"},
			wantErr: "failed to read body file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios := iostreams.Test()
			cmd := NewCmdEdit(ios)

			for k, v := range tt.flags {
				if err := cmd.Flags().Set(k, v); err != nil {
					t.Fatalf("failed to set flag %s: %v", k, err)
				}
			}

			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNewCmdEdit_BodyFileReadsContent(t *testing.T) {
	tmpDir := t.TempDir()
	bodyFile := filepath.Join(tmpDir, "comment.md")
	content := "Updated comment from file"
	if err := os.WriteFile(bodyFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	ios := iostreams.Test()
	cmd := NewCmdEdit(ios)

	if err := cmd.Flags().Set("id", "12345"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := cmd.Flags().Set("body-file", bodyFile); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	cmd.SetArgs([]string{"PROJ-1"})

	err := cmd.Execute()
	if err == nil {
		t.Skip("command succeeded unexpectedly (API config available)")
	}
	if strings.Contains(err.Error(), "failed to read body file") {
		t.Errorf("body file should have been read successfully, got: %v", err)
	}
}
