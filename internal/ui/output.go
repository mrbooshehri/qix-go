package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mrbooshehri/qix-go/internal/config"
	"github.com/mrbooshehri/qix-go/internal/models"
)

var (
	// Color definitions
	Red     = color.New(color.FgRed)
	Green   = color.New(color.FgGreen)
	Yellow  = color.New(color.FgYellow)
	Blue    = color.New(color.FgBlue)
	Cyan    = color.New(color.FgCyan)
	Magenta = color.New(color.FgMagenta)
	White   = color.New(color.FgWhite)
	
	// Bold variants
	BoldRed     = color.New(color.FgRed, color.Bold)
	BoldGreen   = color.New(color.FgGreen, color.Bold)
	BoldYellow  = color.New(color.FgYellow, color.Bold)
	BoldBlue    = color.New(color.FgBlue, color.Bold)
	BoldCyan    = color.New(color.FgCyan, color.Bold)
	BoldMagenta = color.New(color.FgMagenta, color.Bold)
	
	// Dim
	Dim = color.New(color.Faint)
)

// Init initializes the UI system
func Init() {
	cfg := config.Get()
	color.NoColor = !cfg.ColorOutput
}

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	Green.Printf("✓ "+format+"\n", args...)
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	Red.Printf("✗ "+format+"\n", args...)
}

// PrintWarning prints a warning message
func PrintWarning(format string, args ...interface{}) {
	Yellow.Printf("⚠ "+format+"\n", args...)
}

// PrintInfo prints an info message
func PrintInfo(format string, args ...interface{}) {
	Blue.Printf("ℹ "+format+"\n", args...)
}

// PrintHeader prints a section header
func PrintHeader(text string) {
	BoldCyan.Println("\n" + text)
	BoldCyan.Println(strings.Repeat("═", len(text)))
}

// PrintSubHeader prints a subsection header
func PrintSubHeader(text string) {
	BoldBlue.Println("\n" + text)
}

// PrintBox prints text in a bordered box
func PrintBox(title string, lines []string) {
	width := len(title) + 4
	for _, line := range lines {
		if len(line) > width-4 {
			width = len(line) + 4
		}
	}
	
	Cyan.Println("╔" + strings.Repeat("═", width-2) + "╗")
	Cyan.Print("║ ")
	BoldCyan.Print(title)
	Cyan.Println(strings.Repeat(" ", width-len(title)-3) + "║")
	Cyan.Println("╠" + strings.Repeat("═", width-2) + "╣")
	
	for _, line := range lines {
		Cyan.Print("║ ")
		fmt.Print(line)
		Cyan.Println(strings.Repeat(" ", width-len(line)-3) + "║")
	}
	
	Cyan.Println("╚" + strings.Repeat("═", width-2) + "╝")
}

// FormatDuration formats a duration in human-readable format
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// FormatHours formats hours with 2 decimal places
func FormatHours(hours float64) string {
	return fmt.Sprintf("%.2fh", hours)
}

// FormatPercentage formats a percentage with 1 decimal place
func FormatPercentage(pct float64) string {
	return fmt.Sprintf("%.1f%%", pct)
}

// FormatDate formats a date string
func FormatDate(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("Jan 02, 2006")
}

// FormatDateTime formats a datetime string
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// GetStatusIcon returns an icon for a task status
func GetStatusIcon(status models.TaskStatus) string {
	switch status {
	case models.StatusTodo:
		return "⭕"
	case models.StatusDoing:
		return "🔄"
	case models.StatusDone:
		return "✅"
	case models.StatusBlocked:
		return "🚫"
	default:
		return "❓"
	}
}

// GetStatusColor returns the color for a task status
func GetStatusColor(status models.TaskStatus) *color.Color {
	switch status {
	case models.StatusTodo:
		return Yellow
	case models.StatusDoing:
		return Cyan
	case models.StatusDone:
		return Green
	case models.StatusBlocked:
		return Red
	default:
		return White
	}
}

// GetPriorityIcon returns an icon for a priority level
func GetPriorityIcon(priority models.Priority) string {
	switch priority {
	case models.PriorityHigh:
		return "🔴"
	case models.PriorityMedium:
		return "🟡"
	case models.PriorityLow:
		return "🟢"
	default:
		return "⚪"
	}
}

// GetPriorityColor returns the color for a priority level
func GetPriorityColor(priority models.Priority) *color.Color {
	switch priority {
	case models.PriorityHigh:
		return Red
	case models.PriorityMedium:
		return Yellow
	case models.PriorityLow:
		return Green
	default:
		return White
	}
}

// PrintTask prints a task in a formatted way
func PrintTask(task models.Task, indent string) {
	statusColor := GetStatusColor(task.Status)
	statusIcon := GetStatusIcon(task.Status)
	
	// Task line
	statusColor.Printf("%s%s [%s] %s", indent, statusIcon, task.ID, task.Title)
	
	// Priority badge
	if task.Priority != "" {
		priorityColor := GetPriorityColor(task.Priority)
		priorityColor.Printf(" [%s]", task.Priority)
	}
	
	// Status badge
	statusColor.Printf(" [%s]\n", task.Status)
	
	// Time info
	if task.EstimatedHours > 0 {
		actual := task.CalculateActualHours()
		variance := actual - task.EstimatedHours
		
		fmt.Printf("%s   ⏱️  Est: %s | Act: %s", 
			indent, 
			FormatHours(task.EstimatedHours), 
			FormatHours(actual))
		
		if variance != 0 {
			if variance > 0 {
				Red.Printf(" | +%s over", FormatHours(variance))
			} else {
				Green.Printf(" | %s under", FormatHours(-variance))
			}
		}
		fmt.Println()
	}
	
	// Tags
	if len(task.Tags) > 0 {
		Dim.Printf("%s   🏷️  %s\n", indent, strings.Join(task.Tags, ", "))
	}
}

// PrintTaskDetailed prints a task with full details
func PrintTaskDetailed(task models.Task) {
	PrintBox(task.Title, []string{})
	
	// Basic info
	fmt.Println()
	BoldBlue.Print("ID:          ")
	fmt.Println(task.ID)
	
	BoldBlue.Print("Status:      ")
	statusColor := GetStatusColor(task.Status)
	statusColor.Printf("%s %s\n", GetStatusIcon(task.Status), task.Status)
	
	BoldBlue.Print("Priority:    ")
	priorityColor := GetPriorityColor(task.Priority)
	priorityColor.Printf("%s %s\n", GetPriorityIcon(task.Priority), task.Priority)
	
	if task.Description != "" {
		BoldBlue.Print("Description: ")
		fmt.Println(task.Description)
	}
	
	// Time tracking
	fmt.Println()
	BoldBlue.Println("⏱️  Time Tracking:")
	fmt.Printf("   Estimated:  %s\n", FormatHours(task.EstimatedHours))
	
	actual := task.CalculateActualHours()
	fmt.Printf("   Actual:     %s\n", FormatHours(actual))
	
	if task.EstimatedHours > 0 {
		variance := task.GetVariance()
		varPct := task.GetVariancePercentage()
		
		fmt.Print("   Variance:   ")
		if variance > 0 {
			Red.Printf("+%s (%.1f%% over)\n", FormatHours(variance), varPct)
		} else if variance < 0 {
			Green.Printf("%s (%.1f%% under)\n", FormatHours(variance), -varPct)
		} else {
			fmt.Println("On target")
		}
	}
	
	// Time entries
	if len(task.TimeEntries) > 0 {
		fmt.Println()
		BoldBlue.Println("📅 Time Entries:")
		for _, entry := range task.TimeEntries {
			Cyan.Printf("   %s: %s\n", entry.Date, FormatHours(entry.Hours))
		}
	}
	
	// Recurrence
	if task.Recurrence != nil && task.Recurrence.Enabled {
		fmt.Println()
		BoldBlue.Println("🔁 Recurrence:")
		fmt.Printf("   Pattern:    %s", task.Recurrence.Type)
		if task.Recurrence.Value != "" {
			fmt.Printf(" (%s)", task.Recurrence.Value)
		}
		fmt.Println()
		fmt.Printf("   Next Due:   %s\n", FormatDate(task.Recurrence.NextDue))
		if task.Recurrence.LastCompleted != "" {
			fmt.Printf("   Last Done:  %s\n", FormatDate(task.Recurrence.LastCompleted))
		}
	}
	
	// Dependencies
	if len(task.Dependencies) > 0 {
		fmt.Println()
		BoldBlue.Println("🔗 Dependencies:")
		for _, depID := range task.Dependencies {
			Yellow.Printf("   → Depends on: %s\n", depID)
		}
	}
	
	// Parent
	if task.ParentID != "" {
		fmt.Println()
		BoldBlue.Println("👨‍👩‍👧 Hierarchy:")
		Magenta.Printf("   Parent: %s\n", task.ParentID)
	}
	
	// Tags
	if len(task.Tags) > 0 {
		fmt.Println()
		BoldBlue.Println("🏷️  Tags:")
		fmt.Printf("   %s\n", strings.Join(task.Tags, ", "))
	}
	
	// Timestamps
	fmt.Println()
	Dim.Printf("Created: %s\n", FormatDateTime(task.CreatedAt))
	Dim.Printf("Updated: %s\n", FormatDateTime(task.UpdatedAt))
}

// PrintProjectSummary prints a project summary
func PrintProjectSummary(project *models.Project) {
	PrintHeader(project.Name)
	
	if project.Description != "" {
		Dim.Println(project.Description)
	}
	
	fmt.Println()
	
	// Statistics
	counts := project.CountByStatus()
	total := len(project.GetAllTasks())
	
	fmt.Printf("📊 Tasks: %d total\n", total)
	Yellow.Printf("   ⭕ Todo:    %d\n", counts[models.StatusTodo])
	Cyan.Printf("   🔄 Doing:   %d\n", counts[models.StatusDoing])
	Green.Printf("   ✅ Done:    %d\n", counts[models.StatusDone])
	Red.Printf("   🚫 Blocked: %d\n", counts[models.StatusBlocked])
	
	fmt.Println()
	
	// Time stats
	estimated := project.CalculateTotalEstimated()
	actual := project.CalculateTotalActual()
	
	fmt.Println("⏱️  Time:")
	fmt.Printf("   Estimated: %s\n", FormatHours(estimated))
	fmt.Printf("   Actual:    %s\n", FormatHours(actual))
	
	if estimated > 0 {
		variance := actual - estimated
		if variance > 0 {
			Red.Printf("   Variance:  +%s (%.1f%% over)\n", 
				FormatHours(variance), 
				(variance/estimated)*100)
		} else if variance < 0 {
			Green.Printf("   Variance:  %s (%.1f%% under)\n", 
				FormatHours(variance), 
				(-variance/estimated)*100)
		}
	}
	
	fmt.Println()
	
	// Completion
	completion := project.GetCompletionPercentage()
	fmt.Print("📈 Completion: ")
	PrintProgressBar(completion, 40)
	fmt.Printf(" %s\n", FormatPercentage(completion))
	
	fmt.Println()
	fmt.Printf("📦 Modules: %d\n", len(project.Modules))
	fmt.Printf("🏃 Sprints: %d\n", len(project.Sprints))
}

// PrintModuleSummary prints a module summary
func PrintModuleSummary(module *models.Module) {
	PrintSubHeader("📦 " + module.Name)
	
	if module.Description != "" {
		Dim.Println("   " + module.Description)
	}
	
	fmt.Printf("   Tasks: %d\n", len(module.Tasks))
	
	// Count by status
	statusCounts := make(map[models.TaskStatus]int)
	for _, task := range module.Tasks {
		statusCounts[task.Status]++
	}
	
	if len(module.Tasks) > 0 {
		done := statusCounts[models.StatusDone]
		completion := float64(done) / float64(len(module.Tasks)) * 100
		fmt.Print("   Progress: ")
		PrintProgressBar(completion, 30)
		fmt.Printf(" %s\n", FormatPercentage(completion))
	}
}

// PrintList prints a simple bulleted list
func PrintList(items []string, bullet string) {
	for _, item := range items {
		fmt.Printf("%s %s\n", bullet, item)
	}
}

// PrintSeparator prints a horizontal line
func PrintSeparator() {
	Dim.Println(strings.Repeat("─", 80))
}

// PrintEmptyState prints a message when no data exists
func PrintEmptyState(message string, suggestion string) {
	fmt.Println()
	Yellow.Println("ℹ️  " + message)
	if suggestion != "" {
		Dim.Println("   💡 " + suggestion)
	}
	fmt.Println()
}