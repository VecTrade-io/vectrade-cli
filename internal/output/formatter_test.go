package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  Format
	}{
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"csv", FormatCSV},
		{"CSV", FormatCSV},
		{"table", FormatTable},
		{"TABLE", FormatTable},
		{"", FormatTable},
		{"unknown", FormatTable},
	}
	for _, tc := range tests {
		if got := ParseFormat(tc.input); got != tc.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestTable_RenderTable(t *testing.T) {
	t.Parallel()
	tbl := NewTable("Name", "Age")
	tbl.AddRow("Alice", "30")
	tbl.AddRow("Bob", "25")

	var buf bytes.Buffer
	if err := tbl.Render(&buf, FormatTable); err != nil {
		t.Fatalf("Render(table) error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Name") {
		t.Error("table output missing header 'Name'")
	}
	if !strings.Contains(out, "Alice") {
		t.Error("table output missing 'Alice'")
	}
	if !strings.Contains(out, "Bob") {
		t.Error("table output missing 'Bob'")
	}
	// Check separator line exists
	if !strings.Contains(out, "─") {
		t.Error("table output missing separator")
	}
}

func TestTable_RenderJSON(t *testing.T) {
	t.Parallel()
	tbl := NewTable("Symbol", "Price")
	tbl.AddRow("AAPL", "198.50")
	tbl.AddRow("MSFT", "415.00")

	var buf bytes.Buffer
	if err := tbl.Render(&buf, FormatJSON); err != nil {
		t.Fatalf("Render(json) error: %v", err)
	}

	var records []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &records); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["Symbol"] != "AAPL" {
		t.Errorf("expected AAPL, got %s", records[0]["Symbol"])
	}
	if records[1]["Price"] != "415.00" {
		t.Errorf("expected 415.00, got %s", records[1]["Price"])
	}
}

func TestTable_RenderCSV(t *testing.T) {
	t.Parallel()
	tbl := NewTable("ID", "Label")
	tbl.AddRow("1", "test-key")
	tbl.AddRow("2", "prod-key")

	var buf bytes.Buffer
	if err := tbl.Render(&buf, FormatCSV); err != nil {
		t.Fatalf("Render(csv) error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d", len(lines))
	}
	if lines[0] != "ID,Label" {
		t.Errorf("expected header 'ID,Label', got %q", lines[0])
	}
}

func TestTable_EmptyTable(t *testing.T) {
	t.Parallel()
	tbl := NewTable("Col1", "Col2")

	var buf bytes.Buffer
	if err := tbl.Render(&buf, FormatJSON); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	var records []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &records); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records for empty table, got %d", len(records))
	}
}

func TestTable_ColumnWidthAlignment(t *testing.T) {
	t.Parallel()
	tbl := NewTable("X", "LongColumnName")
	tbl.AddRow("a", "b")

	var buf bytes.Buffer
	if err := tbl.Render(&buf, FormatTable); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	lines := strings.Split(buf.String(), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	// Header and data rows should be same length (padded)
	if len(lines[0]) != len(lines[2]) {
		t.Errorf("header and data row should be same length: %d vs %d", len(lines[0]), len(lines[2]))
	}
}

func TestTable_RowWithFewerColumns(t *testing.T) {
	t.Parallel()
	tbl := NewTable("A", "B", "C")
	tbl.AddRow("only-one")

	var buf bytes.Buffer
	// Should not panic
	if err := tbl.Render(&buf, FormatTable); err != nil {
		t.Fatalf("Render error: %v", err)
	}
}

func TestTable_RenderCSV_SpecialChars(t *testing.T) {
	t.Parallel()
	tbl := NewTable("Name", "Value")
	tbl.AddRow("field,with,commas", "has \"quotes\"")

	var buf bytes.Buffer
	if err := tbl.Render(&buf, FormatCSV); err != nil {
		t.Fatalf("Render(csv) error: %v", err)
	}
	// CSV should properly escape these
	if !strings.Contains(buf.String(), "\"field,with,commas\"") {
		t.Error("CSV should quote fields containing commas")
	}
}

type errWriter struct{}

func (e *errWriter) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("write error")
}

type errAfterNWriter struct {
	n       int
	written int
}

func (e *errAfterNWriter) Write(p []byte) (int, error) {
	e.written++
	if e.written > e.n {
		return 0, fmt.Errorf("write error after %d writes", e.n)
	}
	return len(p), nil
}

func TestTable_RenderCSV_HeaderWriteError(t *testing.T) {
	t.Parallel()
	tbl := NewTable("A", "B")
	tbl.AddRow("1", "2")

	w := &errWriter{}
	err := tbl.Render(w, FormatCSV)
	if err == nil {
		t.Error("expected write error")
	}
}

func TestTable_RenderCSV_RowWriteError(t *testing.T) {
	t.Parallel()
	tbl := NewTable("A", "B")
	// Add many rows to overflow the bufio buffer (4096 bytes)
	longVal := strings.Repeat("x", 500)
	for i := 0; i < 20; i++ {
		tbl.AddRow(longVal, longVal)
	}

	// Succeed on first write (header flush), fail on subsequent
	w := &errAfterNWriter{n: 1}
	err := tbl.Render(w, FormatCSV)
	if err == nil {
		t.Error("expected write error on row")
	}
}
