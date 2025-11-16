package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Table represents a formatted table
type Table struct {
	Headers []string
	Rows    [][]string
	Colors  [][]color.Color // Optional colors for cells
	Align   []Alignment     // Column alignment
}

// Alignment defines text alignment in table cells
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignRight
	AlignCenter
)

// NewTable creates a new table
func NewTable(headers []string) *Table {
	return &Table{
		Headers: headers,
		Rows:    make([][]string, 0),
		Colors:  make([][]color.Color, 0),
		Align:   make([]Alignment, len(headers)),
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	t.Rows = append(t.Rows, cells)
}

// AddColoredRow adds a row with specific colors
func (t *Table) AddColoredRow(cells []string, colors []color.Color) {
	t.Rows = append(t.Rows, cells)
	t.Colors = append(t.Colors, colors)
}

// SetColumnAlignment sets alignment for a specific column
func (t *Table) SetColumnAlignment(col int, align Alignment) {
	if col >= 0 && col < len(t.Align) {
		t.Align[col] = align
	}
}

// Print prints the table to stdout
func (t *Table) Print() {
	if len(t.Headers) == 0 {
		return
	}
	
	// Calculate column widths
	widths := t.calculateColumnWidths()
	
	// Print top border
	t.printBorder(widths, "┌", "┬", "┐")
	
	// Print headers
	t.printRow(t.Headers, widths, true, nil)
	
	// Print header separator
	t.printBorder(widths, "├", "┼", "┤")
	
	// Print rows
	for i, row := range t.Rows {
		var rowColors []color.Color
		if i < len(t.Colors) {
			rowColors = t.Colors[i]
		}
		t.printRow(row, widths, false, rowColors)
	}
	
	// Print bottom border
	t.printBorder(widths, "└", "┴", "┘")
}

// PrintSimple prints a simple table without borders
func (t *Table) PrintSimple() {
	if len(t.Headers) == 0 {
		return
	}
	
	widths := t.calculateColumnWidths()
	
	// Print headers
	for i, header := range t.Headers {
		BoldCyan.Print(t.padCell(header, widths[i], t.Align[i]))
		if i < len(t.Headers)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()
	
	// Print separator
	for i, width := range widths {
		fmt.Print(strings.Repeat("─", width))
		if i < len(widths)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()
	
	// Print rows
	for i, row := range t.Rows {
		for j, cell := range row {
			if j < len(widths) {
				padded := t.padCell(cell, widths[j], t.Align[j])
				
				// Apply color if specified
				if i < len(t.Colors) && j < len(t.Colors[i]) {
					t.Colors[i][j].Print(padded)
				} else {
					fmt.Print(padded)
				}
				
				if j < len(row)-1 {
					fmt.Print("  ")
				}
			}
		}
		fmt.Println()
	}
}

// PrintCompact prints a very compact table
func (t *Table) PrintCompact() {
	if len(t.Headers) == 0 {
		return
	}
	
	widths := t.calculateColumnWidths()
	
	// Print headers
	for i, header := range t.Headers {
		BoldCyan.Print(t.padCell(header, widths[i], t.Align[i]))
		if i < len(t.Headers)-1 {
			fmt.Print(" ")
		}
	}
	fmt.Println()
	
	// Print rows
	for i, row := range t.Rows {
		for j, cell := range row {
			if j < len(widths) {
				padded := t.padCell(cell, widths[j], t.Align[j])
				
				if i < len(t.Colors) && j < len(t.Colors[i]) {
					t.Colors[i][j].Print(padded)
				} else {
					fmt.Print(padded)
				}
				
				if j < len(row)-1 {
					fmt.Print(" ")
				}
			}
		}
		fmt.Println()
	}
}

// calculateColumnWidths calculates the width of each column
func (t *Table) calculateColumnWidths() []int {
	widths := make([]int, len(t.Headers))
	
	// Start with header widths
	for i, header := range t.Headers {
		widths[i] = len(header)
	}
	
	// Check row widths
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) {
				cellLen := len(stripAnsiCodes(cell))
				if cellLen > widths[i] {
					widths[i] = cellLen
				}
			}
		}
	}
	
	return widths
}

// printBorder prints a horizontal border
func (t *Table) printBorder(widths []int, left, mid, right string) {
	fmt.Print(left)
	for i, width := range widths {
		fmt.Print(strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			fmt.Print(mid)
		}
	}
	fmt.Println(right)
}

// printRow prints a table row
func (t *Table) printRow(cells []string, widths []int, isHeader bool, colors []color.Color) {
	fmt.Print("│")
	for i, cell := range cells {
		if i < len(widths) {
			fmt.Print(" ")
			padded := t.padCell(cell, widths[i], t.Align[i])
			
			if isHeader {
				BoldCyan.Print(padded)
			} else if colors != nil && i < len(colors) {
				colors[i].Print(padded)
			} else {
				fmt.Print(padded)
			}
			
			fmt.Print(" │")
		}
	}
	fmt.Println()
}

// padCell pads a cell to the specified width with alignment
func (t *Table) padCell(cell string, width int, align Alignment) string {
	cellLen := len(stripAnsiCodes(cell))
	
	if cellLen >= width {
		return cell
	}
	
	padding := width - cellLen
	
	switch align {
	case AlignRight:
		return strings.Repeat(" ", padding) + cell
	case AlignCenter:
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + cell + strings.Repeat(" ", rightPad)
	default: // AlignLeft
		return cell + strings.Repeat(" ", padding)
	}
}

// stripAnsiCodes removes ANSI color codes for length calculation
func stripAnsiCodes(s string) string {
	// Simple implementation - in production you'd use a regex
	// For now, assume no color codes in the string itself
	return s
}

// PrintKeyValue prints a key-value table
func PrintKeyValue(pairs map[string]string) {
	maxKeyLen := 0
	for key := range pairs {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}
	
	for key, value := range pairs {
		BoldBlue.Print(key)
		fmt.Print(strings.Repeat(" ", maxKeyLen-len(key)+2))
		fmt.Println(value)
	}
}

// PrintColumns prints data in columns
func PrintColumns(items []string, columns int) {
	if len(items) == 0 || columns <= 0 {
		return
	}
	
	// Calculate column width
	maxWidth := 0
	for _, item := range items {
		if len(item) > maxWidth {
			maxWidth = len(item)
		}
	}
	columnWidth := maxWidth + 2
	
	// Print in columns
	for i, item := range items {
		padded := item + strings.Repeat(" ", columnWidth-len(item))
		fmt.Print(padded)
		
		if (i+1)%columns == 0 {
			fmt.Println()
		}
	}
	
	// Final newline if needed
	if len(items)%columns != 0 {
		fmt.Println()
	}
}

// PrintGrid prints items in a grid layout
func PrintGrid(items []string, columns int, cellWidth int) {
	if len(items) == 0 || columns <= 0 {
		return
	}
	
	for i := 0; i < len(items); i += columns {
		end := i + columns
		if end > len(items) {
			end = len(items)
		}
		
		for j := i; j < end; j++ {
			cell := items[j]
			if len(cell) > cellWidth {
				cell = cell[:cellWidth-3] + "..."
			}
			
			padded := cell + strings.Repeat(" ", cellWidth-len(cell))
			Cyan.Print("│ ")
			fmt.Print(padded)
			fmt.Print(" ")
		}
		Cyan.Println("│")
	}
}

// TableBuilder is a fluent interface for building tables
type TableBuilder struct {
	table *Table
}

// NewTableBuilder creates a new table builder
func NewTableBuilder(headers ...string) *TableBuilder {
	return &TableBuilder{
		table: NewTable(headers),
	}
}

// Row adds a row
func (tb *TableBuilder) Row(cells ...string) *TableBuilder {
	tb.table.AddRow(cells...)
	return tb
}

// ColoredRow adds a colored row
func (tb *TableBuilder) ColoredRow(cells []string, colors []color.Color) *TableBuilder {
	tb.table.AddColoredRow(cells, colors)
	return tb
}

// Align sets column alignment
func (tb *TableBuilder) Align(col int, align Alignment) *TableBuilder {
	tb.table.SetColumnAlignment(col, align)
	return tb
}

// Build returns the table
func (tb *TableBuilder) Build() *Table {
	return tb.table
}

// Print prints the table
func (tb *TableBuilder) Print() {
	tb.table.Print()
}

// PrintSimple prints simple format
func (tb *TableBuilder) PrintSimple() {
	tb.table.PrintSimple()
}