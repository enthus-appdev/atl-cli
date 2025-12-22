package board

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// RankOptions holds the options for the rank command.
type RankOptions struct {
	IO        *iostreams.IOStreams
	IssueKeys []string
	Before    string
	After     string
	Top       bool
	BoardID   int
	JSON      bool
}

// NewCmdRank creates the rank command.
func NewCmdRank(ios *iostreams.IOStreams) *cobra.Command {
	opts := &RankOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "rank <issue-key> [issue-key...]",
		Short: "Rank/reorder issues on a board",
		Long: `Rank issues to change their order on a Jira board.

Issues can be ranked before or after a target issue, or moved to the top
of the backlog. When ranking multiple issues, they will be placed in the
order specified.`,
		Example: `  # Rank an issue before another
  atl board rank PROJ-123 --before PROJ-456

  # Rank an issue after another
  atl board rank PROJ-123 --after PROJ-456

  # Rank multiple issues in order before a target
  atl board rank PROJ-123 PROJ-124 PROJ-125 --before PROJ-456

  # Move issues to top of backlog (requires board ID)
  atl board rank PROJ-123 PROJ-124 --top --board-id 42

  # Output as JSON
  atl board rank PROJ-123 --before PROJ-456 --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKeys = args

			// Validate flags
			flagCount := 0
			if opts.Before != "" {
				flagCount++
			}
			if opts.After != "" {
				flagCount++
			}
			if opts.Top {
				flagCount++
			}

			if flagCount == 0 {
				return fmt.Errorf("one of --before, --after, or --top is required")
			}
			if flagCount > 1 {
				return fmt.Errorf("only one of --before, --after, or --top can be specified")
			}

			if opts.Top && opts.BoardID == 0 {
				return fmt.Errorf("--board-id is required when using --top")
			}

			return runRank(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Before, "before", "", "Rank issues before this issue key")
	cmd.Flags().StringVar(&opts.After, "after", "", "Rank issues after this issue key")
	cmd.Flags().BoolVar(&opts.Top, "top", false, "Rank issues to top of backlog")
	cmd.Flags().IntVar(&opts.BoardID, "board-id", 0, "Board ID (required for --top)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// RankOutput represents the rank result.
type RankOutput struct {
	Issues   []string `json:"issues"`
	Position string   `json:"position"`
	Target   string   `json:"target,omitempty"`
	Success  bool     `json:"success"`
}

func runRank(opts *RankOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	var rankOutput *RankOutput

	if opts.Before != "" {
		err = jira.RankIssuesBefore(ctx, opts.IssueKeys, opts.Before)
		if err != nil {
			return fmt.Errorf("failed to rank issues: %w", err)
		}
		rankOutput = &RankOutput{
			Issues:   opts.IssueKeys,
			Position: "before",
			Target:   opts.Before,
			Success:  true,
		}
	} else if opts.After != "" {
		err = jira.RankIssuesAfter(ctx, opts.IssueKeys, opts.After)
		if err != nil {
			return fmt.Errorf("failed to rank issues: %w", err)
		}
		rankOutput = &RankOutput{
			Issues:   opts.IssueKeys,
			Position: "after",
			Target:   opts.After,
			Success:  true,
		}
	} else if opts.Top {
		err = jira.RankIssuesToTop(ctx, opts.IssueKeys, opts.BoardID)
		if err != nil {
			return fmt.Errorf("failed to rank issues: %w", err)
		}
		rankOutput = &RankOutput{
			Issues:   opts.IssueKeys,
			Position: "top",
			Success:  true,
		}
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, rankOutput)
	}

	if len(opts.IssueKeys) == 1 {
		if opts.Top {
			fmt.Fprintf(opts.IO.Out, "Ranked %s to top of backlog\n", opts.IssueKeys[0])
		} else {
			fmt.Fprintf(opts.IO.Out, "Ranked %s %s %s\n", opts.IssueKeys[0], rankOutput.Position, rankOutput.Target)
		}
	} else {
		if opts.Top {
			fmt.Fprintf(opts.IO.Out, "Ranked %d issues to top of backlog\n", len(opts.IssueKeys))
		} else {
			fmt.Fprintf(opts.IO.Out, "Ranked %d issues %s %s\n", len(opts.IssueKeys), rankOutput.Position, rankOutput.Target)
		}
		for _, key := range opts.IssueKeys {
			fmt.Fprintf(opts.IO.Out, "  - %s\n", key)
		}
	}

	return nil
}
