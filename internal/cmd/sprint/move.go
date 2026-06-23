package sprint

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

type moveOptions struct {
	IO         *iostreams.IOStreams
	IssueKeys  []string
	SprintID   int
	SprintName string
	BoardID    int
	JSON       bool
}

type moveOutput struct {
	Issues   []string `json:"issues"`
	SprintID int      `json:"sprint_id"`
	Sprint   string   `json:"sprint,omitempty"`
}

func newCmdMove(ios *iostreams.IOStreams) *cobra.Command {
	opts := &moveOptions{IO: ios}
	cmd := &cobra.Command{
		Use:   "move <issue-keys...>",
		Short: "Move issues into a sprint",
		Args:  cobra.MinimumNArgs(1),
		Example: `  atl jira sprint move NX-1 NX-2 --to 123
  atl jira sprint move NX-1 --sprint "Sprint 30" --board 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueKeys = args
			if opts.SprintID == 0 && opts.SprintName == "" {
				return fmt.Errorf("either --to <sprint-id> or --sprint <name> is required")
			}
			if opts.SprintID != 0 && opts.SprintName != "" {
				return fmt.Errorf("cannot specify both --to and --sprint; choose one")
			}
			return runMove(opts)
		},
	}
	cmd.Flags().IntVar(&opts.SprintID, "to", 0, "Target sprint ID")
	cmd.Flags().StringVar(&opts.SprintName, "sprint", "", "Target sprint name (requires --board)")
	cmd.Flags().IntVar(&opts.BoardID, "board", 0, "Board ID (required with --sprint)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")
	return cmd
}

func runMove(opts *moveOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	jira := api.NewJiraService(client)

	sprintID := opts.SprintID
	sprintName := ""
	if opts.SprintName != "" {
		if opts.BoardID == 0 {
			return fmt.Errorf("--board is required when using --sprint by name")
		}
		sprints, err := jira.GetSprints(ctx, opts.BoardID, "active,future")
		if err != nil {
			return fmt.Errorf("failed to get sprints: %w", err)
		}
		var found *api.Sprint
		nameLower := strings.ToLower(opts.SprintName)
		// Prefer an exact (case-insensitive) name match so a substring match
		// never shadows it; fall back to substring only when no exact match
		// exists (consistent with `atl jira issue sprint`).
		for _, s := range sprints {
			if s != nil && strings.ToLower(s.Name) == nameLower {
				found = s
				break
			}
		}
		if found == nil {
			for _, s := range sprints {
				if s != nil && strings.Contains(strings.ToLower(s.Name), nameLower) {
					found = s
					break
				}
			}
		}
		if found == nil {
			return fmt.Errorf("sprint not found: %s\n\nUse 'atl jira sprint list --board %d' to see available sprints", opts.SprintName, opts.BoardID)
		}
		sprintID = found.ID
		sprintName = found.Name
	}

	if err := jira.MoveIssuesToSprint(ctx, sprintID, opts.IssueKeys); err != nil {
		return fmt.Errorf("failed to move issues to sprint: %w", err)
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, &moveOutput{Issues: opts.IssueKeys, SprintID: sprintID, Sprint: sprintName})
	}
	if sprintName != "" {
		fmt.Fprintf(opts.IO.Out, "Moved %d issue(s) to sprint '%s' (ID: %d)\n", len(opts.IssueKeys), sprintName, sprintID)
	} else {
		fmt.Fprintf(opts.IO.Out, "Moved %d issue(s) to sprint %d\n", len(opts.IssueKeys), sprintID)
	}
	return nil
}
