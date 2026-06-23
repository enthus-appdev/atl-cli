package sprint

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/atl-cli/internal/api"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
	"github.com/enthus-appdev/atl-cli/internal/output"
)

// NewCmdSprint creates the sprint command group.
func NewCmdSprint(ios *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sprint",
		Short: "Manage Jira sprints",
		Long: `Create, edit, start, close, and list sprints, and move issues in and out.

Use 'atl jira board list --project PROJ' to find a board ID, then
'atl jira sprint list --board <id>' to find sprint IDs.`,
	}

	cmd.AddCommand(newCmdCreate(ios))
	cmd.AddCommand(newCmdEdit(ios))
	cmd.AddCommand(newCmdStart(ios))
	cmd.AddCommand(newCmdClose(ios))
	cmd.AddCommand(newCmdList(ios))
	cmd.AddCommand(newCmdMove(ios))
	cmd.AddCommand(newCmdBacklog(ios))

	return cmd
}

// sprintDetail is the JSON shape for a single sprint.
type sprintDetail struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	Goal      string `json:"goal,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	BoardID   int    `json:"board_id,omitempty"`
}

func newSprintDetail(s *api.Sprint) *sprintDetail {
	if s == nil {
		return nil
	}
	return &sprintDetail{
		ID:        s.ID,
		Name:      s.Name,
		State:     s.State,
		Goal:      s.Goal,
		StartDate: s.StartDate,
		EndDate:   s.EndDate,
		BoardID:   s.OriginBoardID,
	}
}

// printSprint renders a single sprint as JSON or a short text block.
func printSprint(ios *iostreams.IOStreams, s *api.Sprint, asJSON bool) error {
	if s == nil {
		return fmt.Errorf("no sprint returned")
	}
	d := newSprintDetail(s)
	if asJSON {
		return output.JSON(ios.Out, d)
	}
	fmt.Fprintf(ios.Out, "Sprint %d: %s\n", d.ID, d.Name)
	fmt.Fprintf(ios.Out, "  State: %s\n", d.State)
	if d.Goal != "" {
		fmt.Fprintf(ios.Out, "  Goal:  %s\n", d.Goal)
	}
	if d.StartDate != "" {
		fmt.Fprintf(ios.Out, "  Start: %s\n", dateOnly(d.StartDate))
	}
	if d.EndDate != "" {
		fmt.Fprintf(ios.Out, "  End:   %s\n", dateOnly(d.EndDate))
	}
	return nil
}
