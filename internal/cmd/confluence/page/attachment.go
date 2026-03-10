package page

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// AttachmentOptions holds the options for the attachment command.
type AttachmentOptions struct {
	IO           *iostreams.IOStreams
	PageID       string
	AttachmentID string
	OutputDir    string
	UploadFiles  []string
	List         bool
	Download     bool
	DownloadAll  bool
	JSON         bool
}

// NewCmdAttachment creates the confluence page attachment command.
func NewCmdAttachment(ios *iostreams.IOStreams) *cobra.Command {
	opts := &AttachmentOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "attachment <page-id>",
		Short: "Manage attachments on a Confluence page",
		Long: `List, download, or upload attachments on a Confluence page.

Use this to manage files attached to pages, such as images,
documents, or other resources.`,
		Example: `  # List attachments on a page
  atl confluence page attachment 123456 --list

  # Download a specific attachment by ID
  atl confluence page attachment 123456 --download --id att789

  # Download all attachments from a page
  atl confluence page attachment 123456 --download-all

  # Download to a specific directory
  atl confluence page attachment 123456 --download-all --output ./downloads

  # Upload a file to a page
  atl confluence page attachment 123456 --upload ./diagram.png

  # Upload multiple files
  atl confluence page attachment 123456 --upload file1.pdf --upload file2.png

  # Output attachment list as JSON
  atl confluence page attachment 123456 --list --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PageID = args[0]

			if !opts.List && !opts.Download && !opts.DownloadAll && len(opts.UploadFiles) == 0 {
				opts.List = true // Default to list
			}

			if opts.Download && opts.AttachmentID == "" {
				return fmt.Errorf("--id is required when using --download")
			}

			return runAttachment(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List all attachments on the page")
	cmd.Flags().BoolVarP(&opts.Download, "download", "d", false, "Download a specific attachment (requires --id)")
	cmd.Flags().StringVar(&opts.AttachmentID, "id", "", "Attachment ID to download")
	cmd.Flags().BoolVarP(&opts.DownloadAll, "download-all", "a", false, "Download all attachments")
	cmd.Flags().StringVarP(&opts.OutputDir, "output", "o", ".", "Output directory for downloads")
	cmd.Flags().StringArrayVarP(&opts.UploadFiles, "upload", "u", nil, "File path(s) to upload (can be repeated)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// Attachment output structs

type attachmentOutput struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mimeType"`
	CreatedAt string `json:"created"`
}

type attachmentListOutput struct {
	PageID      string              `json:"page_id"`
	Attachments []*attachmentOutput `json:"attachments"`
	Total       int                 `json:"total"`
}

type attachmentDownloadOutput struct {
	PageID   string `json:"page_id"`
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Path     string `json:"path"`
}

type attachmentUploadOutput struct {
	PageID   string `json:"page_id"`
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
	confluence := api.NewConfluenceService(client)

	// Upload mode
	if len(opts.UploadFiles) > 0 {
		return uploadAttachments(opts, confluence, ctx)
	}

	// Get attachments for list/download operations
	attachments, err := confluence.GetPageAttachmentsAll(ctx, opts.PageID)
	if err != nil {
		return fmt.Errorf("failed to get attachments: %w", err)
	}

	if opts.List {
		return listAttachments(opts, attachments)
	}

	if opts.Download {
		return downloadAttachment(opts, confluence, ctx, attachments)
	}

	if opts.DownloadAll {
		return downloadAllAttachments(opts, confluence, ctx, attachments)
	}

	return nil
}

func listAttachments(opts *AttachmentOptions, attachments []*api.ConfluenceAttachment) error {
	listOut := &attachmentListOutput{
		PageID:      opts.PageID,
		Attachments: make([]*attachmentOutput, 0, len(attachments)),
		Total:       len(attachments),
	}

	for _, a := range attachments {
		created := ""
		if a.Version != nil {
			created = formatAttachmentTime(a.Version.CreatedAt)
		}
		listOut.Attachments = append(listOut.Attachments, &attachmentOutput{
			ID:        a.ID,
			Filename:  a.Title,
			Size:      a.FileSize,
			MimeType:  a.MediaType,
			CreatedAt: created,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, listOut)
	}

	if len(attachments) == 0 {
		fmt.Fprintf(opts.IO.Out, "No attachments on page %s\n", opts.PageID)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Attachments on page %s (%d total):\n\n", opts.PageID, listOut.Total)

	headers := []string{"ID", "FILENAME", "SIZE", "TYPE", "CREATED"}
	rows := make([][]string, 0, len(listOut.Attachments))

	for _, a := range listOut.Attachments {
		rows = append(rows, []string{
			a.ID,
			a.Filename,
			formatAttachmentSize(a.Size),
			a.MimeType,
			a.CreatedAt,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)

	fmt.Fprintf(opts.IO.Out, "\nTo download: atl confluence page attachment %s --download --id <ID>\n", opts.PageID)
	fmt.Fprintf(opts.IO.Out, "To download all: atl confluence page attachment %s --download-all\n", opts.PageID)

	return nil
}

func downloadAttachment(opts *AttachmentOptions, confluence *api.ConfluenceService, ctx context.Context, attachments []*api.ConfluenceAttachment) error {
	// Find the attachment
	var attachment *api.ConfluenceAttachment
	for _, a := range attachments {
		if a.ID == opts.AttachmentID {
			attachment = a
			break
		}
	}

	if attachment == nil {
		return fmt.Errorf("attachment %s not found on page %s", opts.AttachmentID, opts.PageID)
	}

	if attachment.DownloadLink == "" {
		return fmt.Errorf("attachment %s has no download link", opts.AttachmentID)
	}

	content, _, err := confluence.DownloadConfluenceAttachment(ctx, attachment.DownloadLink)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Sanitize filename to prevent path traversal
	safeFilename := filepath.Base(attachment.Title)
	outputPath := filepath.Join(opts.OutputDir, safeFilename)
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	dlOut := &attachmentDownloadOutput{
		PageID:   opts.PageID,
		ID:       attachment.ID,
		Filename: safeFilename,
		Size:     attachment.FileSize,
		Path:     outputPath,
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, dlOut)
	}

	fmt.Fprintf(opts.IO.Out, "Downloaded: %s (%s)\n", outputPath, formatAttachmentSize(attachment.FileSize))
	return nil
}

func downloadAllAttachments(opts *AttachmentOptions, confluence *api.ConfluenceService, ctx context.Context, attachments []*api.ConfluenceAttachment) error {
	if len(attachments) == 0 {
		fmt.Fprintf(opts.IO.Out, "No attachments to download on page %s\n", opts.PageID)
		return nil
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var downloads []*attachmentDownloadOutput
	var errors []string

	for _, a := range attachments {
		if a.DownloadLink == "" {
			errors = append(errors, fmt.Sprintf("%s: no download link", a.Title))
			continue
		}

		content, _, err := confluence.DownloadConfluenceAttachment(ctx, a.DownloadLink)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", a.Title, err))
			continue
		}

		// Sanitize filename to prevent path traversal
		safeFilename := filepath.Base(a.Title)
		outputPath := filepath.Join(opts.OutputDir, safeFilename)
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", a.Title, err))
			continue
		}

		downloads = append(downloads, &attachmentDownloadOutput{
			PageID:   opts.PageID,
			ID:       a.ID,
			Filename: safeFilename,
			Size:     a.FileSize,
			Path:     outputPath,
		})

		if !opts.JSON {
			fmt.Fprintf(opts.IO.Out, "Downloaded: %s (%s)\n", outputPath, formatAttachmentSize(a.FileSize))
		}
	}

	if opts.JSON {
		result := struct {
			PageID    string                      `json:"page_id"`
			Downloads []*attachmentDownloadOutput `json:"downloads"`
			Errors    []string                    `json:"errors,omitempty"`
		}{
			PageID:    opts.PageID,
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

func uploadAttachments(opts *AttachmentOptions, confluence *api.ConfluenceService, ctx context.Context) error {
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

	var uploads []*attachmentUploadOutput
	var errors []string

	for _, f := range opts.UploadFiles {
		result, err := confluence.UploadConfluenceAttachment(ctx, opts.PageID, f)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(f), err))
			continue
		}

		for _, a := range result.Results {
			mediaType := a.MediaType
			fileSize := a.FileSize
			if mediaType == "" && a.Extensions != nil {
				mediaType = a.Extensions.MediaType
			}
			if fileSize == 0 && a.Extensions != nil {
				fileSize = a.Extensions.FileSize
			}

			uploads = append(uploads, &attachmentUploadOutput{
				PageID:   opts.PageID,
				ID:       a.ID,
				Filename: a.Title,
				Size:     fileSize,
				MimeType: mediaType,
			})

			if !opts.JSON {
				fmt.Fprintf(opts.IO.Out, "Uploaded: %s (%s) [ID: %s]\n", a.Title, formatAttachmentSize(fileSize), a.ID)
			}
		}
	}

	if opts.JSON {
		result := struct {
			PageID  string                    `json:"page_id"`
			Uploads []*attachmentUploadOutput `json:"uploads"`
			Errors  []string                  `json:"errors,omitempty"`
		}{
			PageID:  opts.PageID,
			Uploads: uploads,
			Errors:  errors,
		}
		return output.JSON(opts.IO.Out, result)
	}

	if len(errors) > 0 {
		fmt.Fprintf(opts.IO.Out, "\nFailed to upload %d file(s):\n", len(errors))
		for _, e := range errors {
			fmt.Fprintf(opts.IO.Out, "  - %s\n", e)
		}
	}

	if len(opts.UploadFiles) > 1 || len(errors) > 0 {
		fmt.Fprintf(opts.IO.Out, "\nUploaded %d of %d files to page %s\n", len(uploads), len(opts.UploadFiles), opts.PageID)
	}

	return nil
}

// formatAttachmentSize formats a file size in human-readable form.
func formatAttachmentSize(bytes int64) string {
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

// formatAttachmentTime formats a timestamp for display.
func formatAttachmentTime(timeStr string) string {
	if timeStr == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", timeStr)
		if err != nil {
			return timeStr
		}
	}
	return t.Format("2006-01-02 15:04:05")
}
