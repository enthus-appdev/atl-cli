package issue

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// SprintOptions holds the options for the sprint command.
type SprintOptions struct {
	IO          *iostreams.IOStreams
	IssueKeys   []string
	SprintID    int
	SprintName  string
	BoardID     int
	Project     string
	ListSprints bool
	ListBoards  bool
	Backlog     bool
	JSON        bool
}

// NewCmdSprint creates the sprint command.
func NewCmdSprint(ios *iostreams.IOStreams) *cobra.Command {
	opts := &SprintOptions{
		IO: ios,
	}

	cmd := &cobra.Command{
		Use:   "sprint [issue-keys...]",
		Short: "Manage sprint assignments for issues",
		Long: `Move issues to a sprint or backlog.

Use --list-boards to find board IDs, then --list-sprints to find sprint IDs.`,
		Example: `  # List boards in a project
  atl issue sprint --list-boards --project PROJ

  # List sprints for a board
  atl issue sprint --list-sprints --board 123

  # Move issues to a sprint by ID
  atl issue sprint PROJ-1 PROJ-2 --sprint-id 456

  # Move issues to a sprint by name (requires --board)
  atl issue sprint PROJ-1 --sprint "Sprint 5" --board 123

  # Move issues to backlog
  atl issue sprint PROJ-1 --backlog`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.ListBoards {
				return runListBoards(opts)
			}
			if opts.ListSprints {
				if opts.BoardID == 0 {
					return fmt.Errorf("--board is required when listing sprints")
				}
				return runListSprints(opts)
			}

			if len(args) == 0 {
				return fmt.Errorf("at least one issue key is required")
			}
			opts.IssueKeys = args

			if opts.Backlog {
				return runMoveToBacklog(opts)
			}

			if opts.SprintID == 0 && opts.SprintName == "" {
				return fmt.Errorf("either --sprint-id or --sprint is required")
			}

			return runMoveSprint(opts)
		},
	}

	cmd.Flags().IntVar(&opts.SprintID, "sprint-id", 0, "Sprint ID to move issues to")
	cmd.Flags().StringVar(&opts.SprintName, "sprint", "", "Sprint name to move issues to (requires --board)")
	cmd.Flags().IntVar(&opts.BoardID, "board", 0, "Board ID (required for --list-sprints or --sprint)")
	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Project key (for --list-boards)")
	cmd.Flags().BoolVar(&opts.ListSprints, "list-sprints", false, "List available sprints for a board")
	cmd.Flags().BoolVar(&opts.ListBoards, "list-boards", false, "List available boards")
	cmd.Flags().BoolVar(&opts.Backlog, "backlog", false, "Move issues to backlog (remove from sprint)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output as JSON")

	return cmd
}

// BoardOutput represents a board in the output.
type BoardOutput struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Project string `json:"project,omitempty"`
}

// BoardsOutput represents the output for boards list.
type BoardsOutput struct {
	Boards []*BoardOutput `json:"boards"`
	Total  int            `json:"total"`
}

// SprintOutput represents a sprint in the output.
type SprintOutput struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

// SprintsOutput represents the output for sprints list.
type SprintsOutput struct {
	BoardID int             `json:"board_id"`
	Sprints []*SprintOutput `json:"sprints"`
	Total   int             `json:"total"`
}

// SprintMoveOutput represents the output for sprint move.
type SprintMoveOutput struct {
	Issues   []string `json:"issues"`
	SprintID int      `json:"sprint_id,omitempty"`
	Sprint   string   `json:"sprint,omitempty"`
	Action   string   `json:"action"`
}

func runListBoards(opts *SprintOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	boards, err := jira.GetBoards(ctx, opts.Project)
	if err != nil {
		return fmt.Errorf("failed to get boards: %w", err)
	}

	boardsOutput := &BoardsOutput{
		Boards: make([]*BoardOutput, 0, len(boards)),
		Total:  len(boards),
	}

	for _, b := range boards {
		project := ""
		if b.Location != nil {
			project = b.Location.ProjectKey
		}
		boardsOutput.Boards = append(boardsOutput.Boards, &BoardOutput{
			ID:      b.ID,
			Name:    b.Name,
			Type:    b.Type,
			Project: project,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, boardsOutput)
	}

	if boardsOutput.Total == 0 {
		fmt.Fprintln(opts.IO.Out, "No boards found")
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Found %d boards:\n\n", boardsOutput.Total)

	headers := []string{"ID", "NAME", "TYPE", "PROJECT"}
	rows := make([][]string, 0, len(boardsOutput.Boards))

	for _, b := range boardsOutput.Boards {
		rows = append(rows, []string{
			strconv.Itoa(b.ID),
			b.Name,
			b.Type,
			b.Project,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}

func runListSprints(opts *SprintOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	// Get active and future sprints
	sprints, err := jira.GetSprints(ctx, opts.BoardID, "active,future")
	if err != nil {
		return fmt.Errorf("failed to get sprints: %w", err)
	}

	sprintsOutput := &SprintsOutput{
		BoardID: opts.BoardID,
		Sprints: make([]*SprintOutput, 0, len(sprints)),
		Total:   len(sprints),
	}

	for _, s := range sprints {
		sprintsOutput.Sprints = append(sprintsOutput.Sprints, &SprintOutput{
			ID:        s.ID,
			Name:      s.Name,
			State:     s.State,
			StartDate: s.StartDate,
			EndDate:   s.EndDate,
		})
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, sprintsOutput)
	}

	if sprintsOutput.Total == 0 {
		fmt.Fprintf(opts.IO.Out, "No active or future sprints found for board %d\n", opts.BoardID)
		return nil
	}

	fmt.Fprintf(opts.IO.Out, "Sprints for board %d:\n\n", opts.BoardID)

	headers := []string{"ID", "NAME", "STATE", "START", "END"}
	rows := make([][]string, 0, len(sprintsOutput.Sprints))

	for _, s := range sprintsOutput.Sprints {
		startDate := ""
		if s.StartDate != "" {
			startDate = s.StartDate[:10] // Just date part
		}
		endDate := ""
		if s.EndDate != "" {
			endDate = s.EndDate[:10]
		}
		rows = append(rows, []string{
			strconv.Itoa(s.ID),
			s.Name,
			s.State,
			startDate,
			endDate,
		})
	}

	output.SimpleTable(opts.IO.Out, headers, rows)
	return nil
}

func runMoveSprint(opts *SprintOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	sprintID := opts.SprintID
	sprintName := ""

	// If sprint name provided, look it up
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
		for _, s := range sprints {
			if strings.ToLower(s.Name) == nameLower || strings.Contains(strings.ToLower(s.Name), nameLower) {
				found = s
				break
			}
		}

		if found == nil {
			return fmt.Errorf("sprint not found: %s\n\nUse 'atl issue sprint --list-sprints --board %d' to see available sprints", opts.SprintName, opts.BoardID)
		}

		sprintID = found.ID
		sprintName = found.Name
	}

	err = jira.MoveIssuesToSprint(ctx, sprintID, opts.IssueKeys)
	if err != nil {
		return fmt.Errorf("failed to move issues to sprint: %w", err)
	}

	moveOutput := &SprintMoveOutput{
		Issues:   opts.IssueKeys,
		SprintID: sprintID,
		Sprint:   sprintName,
		Action:   "moved_to_sprint",
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, moveOutput)
	}

	if sprintName != "" {
		fmt.Fprintf(opts.IO.Out, "Moved %d issue(s) to sprint '%s' (ID: %d)\n", len(opts.IssueKeys), sprintName, sprintID)
	} else {
		fmt.Fprintf(opts.IO.Out, "Moved %d issue(s) to sprint %d\n", len(opts.IssueKeys), sprintID)
	}
	return nil
}

func runMoveToBacklog(opts *SprintOptions) error {
	client, err := api.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()
	jira := api.NewJiraService(client)

	err = jira.RemoveIssuesFromSprint(ctx, opts.IssueKeys)
	if err != nil {
		return fmt.Errorf("failed to move issues to backlog: %w", err)
	}

	moveOutput := &SprintMoveOutput{
		Issues: opts.IssueKeys,
		Action: "moved_to_backlog",
	}

	if opts.JSON {
		return output.JSON(opts.IO.Out, moveOutput)
	}

	fmt.Fprintf(opts.IO.Out, "Moved %d issue(s) to backlog\n", len(opts.IssueKeys))
	return nil
}
