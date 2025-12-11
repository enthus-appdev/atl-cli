package output

import (
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
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

	table := tablewriter.NewWriter(t.writer)

	// Configure table style for CLI
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)

	if !t.options.NoHeader && len(t.options.Header) > 0 {
		// Make headers uppercase
		headers := make([]string, len(t.options.Header))
		for i, h := range t.options.Header {
			headers[i] = strings.ToUpper(h)
		}
		table.SetHeader(headers)
	}

	table.AppendBulk(t.rows)
	table.Render()
}

// SimpleTable creates and renders a simple table in one call.
func SimpleTable(w io.Writer, headers []string, rows [][]string) {
	t := NewTable(w, TableOptions{Header: headers})
	for _, row := range rows {
		t.AddRow(row...)
	}
	t.Render()
}
