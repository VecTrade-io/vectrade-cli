// Package output provides formatters for CLI output (table, JSON, CSV).
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Format specifies the output format.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
)

// ParseFormat converts a string to a Format, defaulting to table.
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "csv":
		return FormatCSV
	default:
		return FormatTable
	}
}

// Table writes tabular data to the writer.
type Table struct {
	headers []string
	rows    [][]string
}

// NewTable creates a table with the given column headers.
func NewTable(headers ...string) *Table {
	return &Table{headers: headers}
}

// AddRow appends a row to the table.
func (t *Table) AddRow(values ...string) {
	t.rows = append(t.rows, values)
}

// Render writes the table to the writer in the specified format.
func (t *Table) Render(w io.Writer, format Format) error {
	switch format {
	case FormatJSON:
		return t.renderJSON(w)
	case FormatCSV:
		return t.renderCSV(w)
	default:
		return t.renderTable(w)
	}
}

func (t *Table) renderTable(w io.Writer) error {
	// Calculate column widths
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range t.headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprintf(w, "%-*s", widths[i], h)
	}
	fmt.Fprintln(w)

	// Print separator
	for i, width := range widths {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprint(w, strings.Repeat("─", width))
	}
	fmt.Fprintln(w)

	// Print rows
	for _, row := range t.rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "  ")
			}
			if i < len(widths) {
				fmt.Fprintf(w, "%-*s", widths[i], cell)
			}
		}
		fmt.Fprintln(w)
	}

	return nil
}

func (t *Table) renderJSON(w io.Writer) error {
	records := make([]map[string]string, 0, len(t.rows))
	for _, row := range t.rows {
		record := make(map[string]string, len(t.headers))
		for i, h := range t.headers {
			if i < len(row) {
				record[h] = row[i]
			}
		}
		records = append(records, record)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}

func (t *Table) renderCSV(w io.Writer) error {
	csvw := csv.NewWriter(w)
	if err := csvw.Write(t.headers); err != nil {
		return err
	}
	for _, row := range t.rows {
		if err := csvw.Write(row); err != nil {
			return err
		}
	}
	csvw.Flush()
	return csvw.Error()
}
