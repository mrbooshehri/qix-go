package ui

import (
	"fmt"
	"strings"
)

// ProgressBarStyle defines the characters used in a progress bar
type ProgressBarStyle struct {
	LeftBracket  string
	RightBracket string
	Filled       string
	Empty        string
	Partial      []string
}

var (
	// DefaultProgressBarStyle is the default style for progress bars
	DefaultProgressBarStyle = ProgressBarStyle{
		LeftBracket:  "[",
		RightBracket: "]",
		Filled:       "█",
		Empty:        "░",
		Partial:      []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"},
	}
	
	// RoundedProgressBarStyle uses rounded brackets
	RoundedProgressBarStyle = ProgressBarStyle{
		LeftBracket:  "(",
		RightBracket: ")",
		Filled:       "●",
		Empty:        "○",
		Partial:      []string{"◔", "◑", "◕"},
	}
	
	// BlockProgressBarStyle uses block characters
	BlockProgressBarStyle = ProgressBarStyle{
		LeftBracket:  "⟦",
		RightBracket: "⟧",
		Filled:       "■",
		Empty:        "□",
		Partial:      []string{"▪"},
	}
)

// PrintProgressBar prints a progress bar with percentage
func PrintProgressBar(percentage float64, width int) {
	PrintProgressBarWithStyle(percentage, width, DefaultProgressBarStyle)
}

// PrintProgressBarWithStyle prints a progress bar with custom style
func PrintProgressBarWithStyle(percentage float64, width int, style ProgressBarStyle) {
	// Clamp percentage to 0-100
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}
	
	// Calculate filled width
	filledWidth := (percentage / 100.0) * float64(width)
	filledBlocks := int(filledWidth)
	partialBlock := filledWidth - float64(filledBlocks)
	
	// Build the bar
	bar := style.LeftBracket
	
	// Add filled blocks
	for i := 0; i < filledBlocks && i < width; i++ {
		bar += style.Filled
	}
	
	// Add partial block if needed
	if filledBlocks < width && partialBlock > 0 && len(style.Partial) > 0 {
		partialIndex := int(partialBlock * float64(len(style.Partial)))
		if partialIndex >= len(style.Partial) {
			partialIndex = len(style.Partial) - 1
		}
		bar += style.Partial[partialIndex]
		filledBlocks++
	}
	
	// Add empty blocks
	for i := filledBlocks; i < width; i++ {
		bar += style.Empty
	}
	
	bar += style.RightBracket
	
	// Color the bar based on percentage
	if percentage >= 80 {
		Green.Print(bar)
	} else if percentage >= 50 {
		Yellow.Print(bar)
	} else if percentage >= 25 {
		Magenta.Print(bar)
	} else {
		Red.Print(bar)
	}
}

// PrintSpinner prints a spinner character (for animations)
func PrintSpinner(frame int) {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	Cyan.Print(spinners[frame%len(spinners)])
}

// PrintLoadingBar prints a loading bar animation
func PrintLoadingBar(current, total int, width int) {
	percentage := (float64(current) / float64(total)) * 100
	
	fmt.Printf("\r")
	PrintProgressBar(percentage, width)
	fmt.Printf(" %d/%d (%.1f%%)", current, total, percentage)
	
	if current >= total {
		fmt.Println()
	}
}

// PrintTree prints a tree structure
func PrintTree(nodes []TreeNode, indent string, isLast bool) {
	for i, node := range nodes {
		isLastChild := i == len(nodes)-1
		
		// Print connector
		if indent == "" {
			// Root level
			if isLastChild {
				fmt.Print("└── ")
			} else {
				fmt.Print("├── ")
			}
		} else {
			fmt.Print(indent)
			if isLastChild {
				fmt.Print("└── ")
			} else {
				fmt.Print("├── ")
			}
		}
		
		// Print node
		node.Print()
		
		// Print children
		if len(node.Children) > 0 {
			var childIndent string
			if indent == "" {
				if isLastChild {
					childIndent = "    "
				} else {
					childIndent = "│   "
				}
			} else {
				if isLastChild {
					childIndent = indent + "    "
				} else {
					childIndent = indent + "│   "
				}
			}
			PrintTree(node.Children, childIndent, isLastChild)
		}
	}
}

// TreeNode represents a node in a tree structure
type TreeNode struct {
	Label    string
	Color    *color.Color
	Children []TreeNode
	Data     interface{}
}

// Print prints the tree node
func (n TreeNode) Print() {
	if n.Color != nil {
		n.Color.Println(n.Label)
	} else {
		fmt.Println(n.Label)
	}
}

// PrintChart prints a simple horizontal bar chart
func PrintChart(data map[string]float64, width int, showValues bool) {
	if len(data) == 0 {
		return
	}
	
	// Find max value for scaling
	maxValue := 0.0
	maxLabelLen := 0
	for label, value := range data {
		if value > maxValue {
			maxValue = value
		}
		if len(label) > maxLabelLen {
			maxLabelLen = len(label)
		}
	}
	
	// Print bars
	for label, value := range data {
		// Pad label
		paddedLabel := label + strings.Repeat(" ", maxLabelLen-len(label))
		fmt.Printf("%s: ", paddedLabel)
		
		// Calculate bar width
		barWidth := int((value / maxValue) * float64(width))
		
		// Print bar
		if value > 0 {
			Cyan.Print(strings.Repeat("█", barWidth))
		}
		
		// Print value
		if showValues {
			fmt.Printf(" %.1f", value)
		}
		
		fmt.Println()
	}
}

// PrintSparkline prints a sparkline chart
func PrintSparkline(values []float64) {
	if len(values) == 0 {
		return
	}
	
	// Find min and max
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	
	// Sparkline characters (8 levels)
	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	
	// Normalize and print
	for _, v := range values {
		var index int
		if max == min {
			index = len(chars) / 2
		} else {
			normalized := (v - min) / (max - min)
			index = int(normalized * float64(len(chars)-1))
		}
		
		// Color based on value
		if v >= max*0.75 {
			Green.Print(chars[index])
		} else if v >= max*0.5 {
			Yellow.Print(chars[index])
		} else {
			Red.Print(chars[index])
		}
	}
	fmt.Println()
}

// PrintGauge prints a gauge/meter display
func PrintGauge(value, min, max float64, width int) {
	// Clamp value
	if value < min {
		value = min
	}
	if value > max {
		value = max
	}
	
	// Calculate position
	percentage := ((value - min) / (max - min)) * 100
	position := int((percentage / 100.0) * float64(width))
	
	// Build gauge
	gauge := "["
	for i := 0; i < width; i++ {
		if i == position {
			gauge += "█"
		} else if i < position {
			gauge += "─"
		} else {
			gauge += "·"
		}
	}
	gauge += "]"
	
	// Color based on percentage
	if percentage >= 75 {
		Red.Print(gauge)
	} else if percentage >= 50 {
		Yellow.Print(gauge)
	} else {
		Green.Print(gauge)
	}
	
	fmt.Printf(" %.1f/%.1f", value, max)
	fmt.Println()
}

// PrintHeatmap prints a simple text-based heatmap
func PrintHeatmap(data [][]float64, labels []string) {
	if len(data) == 0 {
		return
	}
	
	// Find max value for normalization
	maxValue := 0.0
	for _, row := range data {
		for _, val := range row {
			if val > maxValue {
				maxValue = val
			}
		}
	}
	
	// Heat characters (from cold to hot)
	chars := []string{" ", "·", "∘", "○", "◐", "●", "◉", "⬤"}
	
	for i, row := range data {
		// Print row label if provided
		if i < len(labels) {
			fmt.Printf("%s: ", labels[i])
		}
		
		for _, val := range row {
			var index int
			if maxValue == 0 {
				index = 0
			} else {
				normalized := val / maxValue
				index = int(normalized * float64(len(chars)-1))
			}
			
			// Color based on intensity
			if val >= maxValue*0.75 {
				Red.Print(chars[index])
			} else if val >= maxValue*0.5 {
				Yellow.Print(chars[index])
			} else if val >= maxValue*0.25 {
				Cyan.Print(chars[index])
			} else {
				Blue.Print(chars[index])
			}
		}
		fmt.Println()
	}
}

// PrintBadge prints a colored badge
func PrintBadge(text string, badgeColor *color.Color) {
	if badgeColor == nil {
		badgeColor = Cyan
	}
	
	badgeColor.Printf(" %s ", text)
}

// PrintStatusBadge prints a status badge with icon
func PrintStatusBadge(status string, isSuccess bool) {
	if isSuccess {
		Green.Printf(" ✓ %s ", status)
	} else {
		Red.Printf(" ✗ %s ", status)
	}
}