package main

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// table renders rows in aligned columns.
//
// It exists so every listing lines up the same way without each one measuring
// its own widths, and so a column's width comes from the rows rather than from
// a guess that a longer name will eventually break.
type table struct {
	header []string
	rows   [][]string
	// numeric marks columns that read better right-aligned.
	numeric map[int]bool
}

func newTable(header ...string) *table {
	return &table{header: header, numeric: map[int]bool{}}
}

// rightAlign marks a column as numeric.
func (t *table) rightAlign(columns ...int) *table {
	for _, column := range columns {
		t.numeric[column] = true
	}
	return t
}

func (t *table) add(cells ...string) { t.rows = append(t.rows, cells) }

func (t *table) render(out io.Writer) {
	// Widths are counted in runes, not bytes. A column holding an arrow would
	// otherwise be padded three times for one character and the whole table
	// would lean.
	widths := make([]int, len(t.header))
	for i, cell := range t.header {
		widths[i] = utf8.RuneCountInString(cell)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if width := utf8.RuneCountInString(cell); i < len(widths) && width > widths[i] {
				widths[i] = width
			}
		}
	}
	writeRow := func(cells []string) {
		var line strings.Builder
		for i, cell := range cells {
			if i > 0 {
				line.WriteString("  ")
			}
			pad := ""
			if i < len(widths) {
				if gap := widths[i] - utf8.RuneCountInString(cell); gap > 0 {
					pad = strings.Repeat(" ", gap)
				}
			}
			// The last column is never padded: trailing spaces on a free-text
			// column are noise in a terminal and in a golden file alike.
			switch {
			case i == len(widths)-1:
				line.WriteString(cell)
			case t.numeric[i]:
				line.WriteString(pad)
				line.WriteString(cell)
			default:
				line.WriteString(cell)
				line.WriteString(pad)
			}
		}
		fmt.Fprintln(out, strings.TrimRight(line.String(), " "))
	}
	writeRow(t.header)
	for _, row := range t.rows {
		writeRow(row)
	}
}
