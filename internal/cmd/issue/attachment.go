package issue

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// AttachmentOptions holds the options for the attachment command.
type AttachmentOptions struct {
	IO           *iostreams.IOStreams
	IssueKey     string
	AttachmentID string
	OutputDir    string
	UploadFiles  []string
	List         bool
	Download     bool
	DownloadAll  bool
	JSON         bool
}

// NewCmdAttachment creates the attachment command.
func NewCmdAttachment(ios *iostreams.IOStreams) *cobra.Command {
	opts := &AttachmentOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "attachment <issue-key>",
		Short: "Manage attachments on a Jira issue",
		Long: `List, download, or upload attachments on a Jira issue.

Use this to manage files attached to tickets, such as error logs,
screenshots, or documents.`,
		Example: `  # List attachments on an issue
  atl issue attachment PROJ-123 --list

  # Download a specific attachment by ID
  atl issue attachment PROJ-123 --download --id 12345

  # Download all attachments from an issue
  atl issue attachment PROJ-123 --download-all

  # Download to a specific directory
  atl issue attachment PROJ-123 --download-all --output ./downloads

  # Upload a file to an issue
  atl issue attachment PROJ-123 --upload ./screenshot.png

  # Upload multiple files
  atl issue attachment PROJ-123 --upload file1.pdf --upload file2.png

  # Output attachment list as JSON
  atl issue attachment PROJ-123 --list --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKey = args[0]

			if !opts.List && !opts.Download && !opts.DownloadAll && len(opts.UploadFiles) == 0 {
				opts.List = true // Default to list
			}

			if opts.Download && opts.AttachmentID == "" {
				return fmt.Errorf("--id is required when using --download")
			}

			return runAttachment(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List all attachments on the issue")
	cmd.Flags().BoolVarP(&opts.Download, "download", "d", false, "Download a specific attachment (requires --id)")
	cmd.Flags().StringVar(&opts.AttachmentID, "id", "", "Attachment ID to download")
	cmd.Flags().BoolVarP(&opts.DownloadAll, "download-all", "a", false, "Download all attachments")
	cmd.Flags().StringVarP(&opts.OutputDir, "output", "o", ".", "Output directory for downloads")
	cmd.Flags().StringArrayVarP(&opts.UploadFiles, "upload", "u", nil, "File path(s) to upload (can be repeated)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// AttachmentOutput represents an attachment in output.
type AttachmentOutput struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Author   string `json:"author,omitempty"`
	Created  string `json:"created"`
}

// AttachmentListOutput represents the list output.
type AttachmentListOutput struct {
	IssueKey    string              `json:"issue_key"`
	Attachments []*AttachmentOutput `json:"attachments"`
	Total       int                 `json:"total"`
}

// DownloadOutput represents a download result.
type DownloadOutput struct {
	IssueKey string `json:"issue_key"`
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Path     string `json:"path"`
}

// UploadOutput represents an upload result.
type UploadOutput struct {
	IssueKey string `json:"issue_key"`
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
}

func runAttachment(opts *AttachmentOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Upload mode - doesn't need to fetch the issue first
	if len(opts.UploadFiles) > 0 {
		return uploadAttachments(opts, jira, ctx)
	}

	// Get the issue to get attachment list
	issue, err := jira.GetIssue(ctx, opts.IssueKey)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	attachments := issue.Fields.Attachment
	if attachments == nil {
		attachments = []*api.Attachment{}
	}

	// List mode
	if opts.List {
		return listAttachments(opts, attachments)
	}

	// Download single attachment
	if opts.Download {
		return downloadAttachment(opts, jira, ctx, attachments)
	}

	// Download all attachments
	if opts.DownloadAll {
		return downloadAllAttachments(opts, jira, ctx, attachments)
	}

	return nil
}

func listAttachments(opts *AttachmentOptions, attachments []*api.Attachment) error {
	listOutput := &AttachmentListOutput{
		IssueKey:    opts.IssueKey,
		Attachments: make([]*AttachmentOutput, 0, len(attachments)),
		Total:       len(attachments),
	}

	for _, a := range attachments {
		author := ""
		if a.Author != nil {
			author = a.Author.DisplayName
		}
		listOutput.Attachments = append(listOutput.Attachments, &AttachmentOutput{
			ID:       a.ID,
			Filename: a.Filename,
			Size:     a.Size,
			MimeType: a.MimeType,
			Author:   author,
			Created:  formatTime(a.Created),
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOutput)
	}

	if len(attachments) == 0 {
		fmt.Fprintf(opts.IO.Out, "No attachments on %s\n", opts.IssueKey)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Attachments on %s (%d total):\n\n", opts.IssueKey, listOutput.Total)

	headers := []string{"ID", "FILENAME", "SIZE", "TYPE", "CREATED"}
	rows := make([][]string, 0, len(listOutput.Attachments))

	for _, a := range listOutput.Attachments {
		rows = append(rows, []string{
			a.ID,
			a.Filename,
			formatSize(a.Size),
			a.MimeType,
			a.Created,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	fmt.Fprintf(opts.IO.Out, "\nTo download: atl issue attachment %s --download --id <ID>\n", opts.IssueKey)
	fmt.Fprintf(opts.IO.Out, "To download all: atl issue attachment %s --download-all\n", opts.IssueKey)

	return nil
}

func downloadAttachment(opts *AttachmentOptions, jira *api.JiraService, ctx context.Context, attachments []*api.Attachment) error {
	// Find the attachment
	var attachment *api.Attachment
	for _, a := range attachments {
		if a.ID == opts.AttachmentID {
			attachment = a
			break
		}
	}

	if attachment == nil {
		return fmt.Errorf("attachment %s not found on issue %s", opts.AttachmentID, opts.IssueKey)
	}

	// Download the content
	content, _, err := jira.DownloadAttachment(ctx, opts.AttachmentID)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}

	// Create output directory if needed
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(opts.OutputDir, attachment.Filename)
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	downloadOutput := &DownloadOutput{
		IssueKey: opts.IssueKey,
		ID:       attachment.ID,
		Filename: attachment.Filename,
		Size:     int64(len(content)),
		Path:     outputPath,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, downloadOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Downloaded: %s (%s)\n", outputPath, formatSize(int64(len(content))))

	return nil
}

func downloadAllAttachments(opts *AttachmentOptions, jira *api.JiraService, ctx context.Context, attachments []*api.Attachment) error {
	if len(attachments) == 0 {
		fmt.Fprintf(opts.IO.Out, "No attachments to download on %s\n", opts.IssueKey)
		return nil
	}

	// Create output directory if needed
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var downloads []*DownloadOutput
	var errors []string

	for _, a := range attachments {
		content, _, err := jira.DownloadAttachment(ctx, a.ID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", a.Filename, err))
			continue
		}

		outputPath := filepath.Join(opts.OutputDir, a.Filename)
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", a.Filename, err))
			continue
		}

		downloads = append(downloads, &DownloadOutput{
			IssueKey: opts.IssueKey,
			ID:       a.ID,
			Filename: a.Filename,
			Size:     int64(len(content)),
			Path:     outputPath,
		})

		if !opts.JSON {
			fmt.Fprintf(opts.IO.Out, "Downloaded: %s (%s)\n", outputPath, formatSize(int64(len(content))))
		}
	}

	if opts.JSON {
		result := struct {
			IssueKey  string            `json:"issue_key"`
			Downloads []*DownloadOutput `json:"downloads"`
			Errors    []string          `json:"errors,omitempty"`
		}{
			IssueKey:  opts.IssueKey,
			Downloads: downloads,
			Errors:    errors,
		}
		return output.JSON(opts.IO.Out, result)
	}

	if len(errors) > 0 {
		fmt.Fprintf(opts.IO.Out, "\nFailed to download %d file(s):\n", len(errors))
		for _, e := range errors {
			fmt.Fprintf(opts.IO.Out, "  - %s\n", e)
		}
	}

	fmt.Fprintf(opts.IO.Out, "\nDownloaded %d of %d attachments to %s\n", len(downloads), len(attachments), opts.OutputDir)

	return nil
}

func uploadAttachments(opts *AttachmentOptions, jira *api.JiraService, ctx context.Context) error {
	// Validate all files exist before uploading
	for _, f := range opts.UploadFiles {
		info, err := os.Stat(f)
		if err != nil {
			return fmt.Errorf("file not found: %s", f)
		}
		if info.IsDir() {
			return fmt.Errorf("cannot upload a directory: %s", f)
		}
	}

	var uploads []*UploadOutput
	var errors []string

	for _, f := range opts.UploadFiles {
		attachments, err := jira.UploadAttachment(ctx, opts.IssueKey, f)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(f), err))
			continue
		}

		for _, a := range attachments {
			uploads = append(uploads, &UploadOutput{
				IssueKey: opts.IssueKey,
				ID:       a.ID,
				Filename: a.Filename,
				Size:     a.Size,
				MimeType: a.MimeType,
			})

			if !opts.JSON {
				fmt.Fprintf(opts.IO.Out, "Uploaded: %s (%s) [ID: %s]\n", a.Filename, formatSize(a.Size), a.ID)
			}
		}
	}

	if opts.JSON {
		result := struct {
			IssueKey string          `json:"issue_key"`
			Uploads  []*UploadOutput `json:"uploads"`
			Errors   []string        `json:"errors,omitempty"`
		}{
			IssueKey: opts.IssueKey,
			Uploads:  uploads,
			Errors:   errors,
		}
		return output.JSON(opts.IO.Out, result)
	}

	if len(errors) > 0 {
		fmt.Fprintf(opts.IO.Out, "\nFailed to upload %d file(s):\n", len(errors))
		for _, e := range errors {
			fmt.Fprintf(opts.IO.Out, "  - %s\n", e)
		}
	}

	if len(opts.UploadFiles) > 1 {
		fmt.Fprintf(opts.IO.Out, "\nUploaded %d of %d files to %s\n", len(uploads), len(opts.UploadFiles), opts.IssueKey)
	}

	return nil
}

// formatSize formats a file size in human-readable form.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
