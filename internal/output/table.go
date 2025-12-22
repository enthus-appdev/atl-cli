package output

import (
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// TableOptions configures table output.
type TableOptions struct {
	Header    []string
	NoHeader  bool
	Separator string
}

// Table renders data as a table.
type Table struct {
	writer  io.Writer
	options TableOptions
	rows    [][]string
}

// NewTable creates a new table writer.
func NewTable(w io.Writer, opts TableOptions) *Table {
	return &Table{
		writer:  w,
		options: opts,
		rows:    make([][]string, 0),
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(row ...string) {
	t.rows = append(t.rows, row)
}

// Render writes the table to the output.
func (t *Table) Render() {
	if len(t.rows) == 0 {
		return
	}

	// Configure table style for CLI: no borders, no separators, left-aligned
	table := tablewriter.NewTable(t.writer,
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Separators: tw.SeparatorsNone,
				Lines:      tw.LinesNone,
			},
		}),
		tablewriter.WithHeaderAutoFormat(tw.On),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithPadding(tw.Padding{Left: "", Right: "  ", Overwrite: true}),
		tablewriter.WithTrimSpace(tw.On),
	)

	if !t.options.NoHeader && len(t.options.Header) > 0 {
		// Make headers uppercase
		headers := make([]any, len(t.options.Header))
		for i, h := range t.options.Header {
			headers[i] = strings.ToUpper(h)
		}
		table.Header(headers...)
	}

	_ = table.Bulk(t.rows)
	_ = table.Render()
}

// SimpleTable creates and renders a simple table in one call.
func SimpleTable(w io.Writer, headers []string, rows [][]string) {
	t := NewTable(w, TableOptions{Header: headers})
	for _, row := range rows {
		t.AddRow(row...)
	}
	t.Render()
}
