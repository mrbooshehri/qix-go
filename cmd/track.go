package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrbooshehri/qix-go/internal/models"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
)

var trackCmd = &cobra.Command{
	Use:   "track",
	Short: "Time tracking",
	Long:  "Start, stop, and manage time tracking sessions",
}

var trackStartCmd = &cobra.Command{
	Use:   "start <project[/module]> <task_id>",
	Short: "Start time tracking for a task",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		taskID := args[1]

		projectName, moduleName := parsePath(path)

		store := storage.Get()

		// Check if already tracking
		tracking, err := store.IsTracking()
		if err != nil {
			ui.PrintError("Failed to check tracking status: %v", err)
			return
		}

		if tracking {
			session, _ := store.GetActiveSession()
			ui.PrintWarning("Already tracking task: %s", session.TaskID)
			ui.Dim.Printf("  Path: %s\n", session.Path)
			ui.Dim.Printf("  Started: %s\n", ui.FormatDateTime(session.StartTime))

			fmt.Println()
			fmt.Print("Stop current session and start new one? (y/N): ")

			var confirm string
			fmt.Scanln(&confirm)

			if confirm != "y" && confirm != "Y" {
				ui.PrintInfo("Tracking not changed")
				return
			}

			// Stop current session
			elapsed, oldPath, oldTaskID, err := store.StopTracking()
			if err != nil {
				ui.PrintError("Failed to stop current session: %v", err)
				return
			}

			ui.PrintSuccess("Stopped tracking: %s [%s]", oldPath, oldTaskID)
			ui.Cyan.Printf("  Duration: %s (%.2fh)\n", ui.FormatDuration(elapsed), elapsed.Hours())
			fmt.Println()
		}

		// Verify task exists
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		// Start tracking
		if err := store.StartTracking(projectName, moduleName, taskID); err != nil {
			ui.PrintError("Failed to start tracking: %v", err)
			return
		}

		ui.PrintSuccess("⏱️  Tracking started")
		ui.BoldCyan.Printf("  Task: [%s] %s\n", taskID, task.Title)

		if moduleName != "" {
			ui.Dim.Printf("  Path: %s/%s\n", projectName, moduleName)
		} else {
			ui.Dim.Printf("  Path: %s\n", projectName)
		}

		ui.Dim.Printf("  Started: %s\n", time.Now().Format("15:04:05"))

		fmt.Println()
		ui.Yellow.Println("💡 Tip: Use 'qix track stop' when done")
	},
}

var trackStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop active time tracking",
	Run: func(cmd *cobra.Command, args []string) {
		store := storage.Get()

		// Check if tracking
		tracking, err := store.IsTracking()
		if err != nil {
			ui.PrintError("Failed to check tracking status: %v", err)
			return
		}

		if !tracking {
			ui.PrintWarning("No active tracking session")
			return
		}

		// Get session details before stopping
		session, err := store.GetActiveSession()
		if err != nil {
			ui.PrintError("Failed to get session: %v", err)
			return
		}

		// Get task details
		projectName, _ := parsePath(session.Path)
		task, _, err := store.FindTask(projectName, session.TaskID)
		if err != nil {
			ui.PrintWarning("Could not load task details")
		}

		// Stop tracking
		elapsed, path, taskID, err := store.StopTracking()
		if err != nil {
			ui.PrintError("Failed to stop tracking: %v", err)
			return
		}

		hours := elapsed.Hours()

		ui.PrintSuccess("⏹️  Tracking stopped")

		if task != nil {
			ui.BoldCyan.Printf("  Task: [%s] %s\n", taskID, task.Title)
		} else {
			ui.BoldCyan.Printf("  Task: [%s]\n", taskID)
		}

		ui.Cyan.Printf("  Path: %s\n", path)
		ui.Green.Printf("  Duration: %s\n", ui.FormatDuration(elapsed))
		ui.Yellow.Printf("  Logged: %.2fh\n", hours)
		ui.Dim.Printf("  Date: %s\n", time.Now().Format("2006-01-02"))

		// Show updated totals if we have task
		if task != nil {
			newActual := task.CalculateActualHours() + hours

			fmt.Println()
			ui.Blue.Println("  Time Summary:")

			if task.EstimatedHours > 0 {
				ui.Dim.Printf("    Estimated: %s\n", ui.FormatHours(task.EstimatedHours))
				ui.Cyan.Printf("    Actual:    %s\n", ui.FormatHours(newActual))

				variance := newActual - task.EstimatedHours
				if variance > 0 {
					ui.Red.Printf("    Variance:  +%s (%.1f%% over)\n",
						ui.FormatHours(variance),
						(variance/task.EstimatedHours)*100)
				} else if variance < 0 {
					ui.Green.Printf("    Variance:  %s (%.1f%% under)\n",
						ui.FormatHours(variance),
						(-variance/task.EstimatedHours)*100)
				}
			} else {
				ui.Cyan.Printf("    Total logged: %s\n", ui.FormatHours(newActual))
			}
		}
	},
}

var trackStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current tracking status",
	Run: func(cmd *cobra.Command, args []string) {
		store := storage.Get()

		tracking, err := store.IsTracking()
		if err != nil {
			ui.PrintError("Failed to check tracking status: %v", err)
			return
		}

		if !tracking {
			ui.Blue.Println("🟢 No active tracking session")
			fmt.Println()
			ui.Dim.Println("Start tracking with: qix track start <project> <task_id>")
			return
		}

		session, err := store.GetActiveSession()
		if err != nil {
			ui.PrintError("Failed to get session: %v", err)
			return
		}

		elapsed := time.Since(session.StartTime)

		// Get task details
		projectName, moduleName := parsePath(session.Path)
		task, _, err := store.FindTask(projectName, session.TaskID)

		ui.PrintHeader("⏳ Active Tracking Session")

		if task != nil {
			ui.BoldGreen.Printf("Task:     [%s] %s\n", session.TaskID, task.Title)

			statusColor := ui.GetStatusColor(task.Status)
			statusColor.Printf("Status:   %s %s\n", ui.GetStatusIcon(task.Status), task.Status)

			if task.Priority != "" {
				priorityColor := ui.GetPriorityColor(task.Priority)
				priorityColor.Printf("Priority: %s %s\n", ui.GetPriorityIcon(task.Priority), task.Priority)
			}
		} else {
			ui.BoldGreen.Printf("Task:     [%s]\n", session.TaskID)
		}

		fmt.Println()

		if moduleName != "" {
			ui.Blue.Printf("Path:     %s/%s\n", projectName, moduleName)
		} else {
			ui.Blue.Printf("Path:     %s\n", projectName)
		}

		ui.Cyan.Printf("Started:  %s\n", ui.FormatDateTime(session.StartTime))
		ui.Yellow.Printf("Elapsed:  %s (%.2fh)\n", ui.FormatDuration(elapsed), elapsed.Hours())

		// Show time info if we have task
		if task != nil && task.EstimatedHours > 0 {
			fmt.Println()
			ui.Blue.Println("Time Tracking:")

			currentActual := task.CalculateActualHours()
			projectedActual := currentActual + elapsed.Hours()

			ui.Dim.Printf("  Estimated:       %s\n", ui.FormatHours(task.EstimatedHours))
			ui.Cyan.Printf("  Logged so far:   %s\n", ui.FormatHours(currentActual))
			ui.Yellow.Printf("  This session:    %.2fh\n", elapsed.Hours())
			ui.Green.Printf("  Projected total: %s\n", ui.FormatHours(projectedActual))

			remaining := task.EstimatedHours - projectedActual
			if remaining > 0 {
				ui.Blue.Printf("  Remaining:       %s\n", ui.FormatHours(remaining))
			} else {
				ui.Red.Printf("  Over budget by:  %s\n", ui.FormatHours(-remaining))
			}
		}

		fmt.Println()
		ui.Dim.Println("Stop tracking with: qix track stop")
	},
}

var trackLogCmd = &cobra.Command{
	Use:   "log <project[/module]> <task_id> <hours>",
	Short: "Manually log time to a task",
	Long:  "Log time without starting/stopping a tracking session",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		taskID := args[1]
		hoursStr := args[2]

		// Parse hours
		var hours float64
		if _, err := fmt.Sscanf(hoursStr, "%f", &hours); err != nil {
			ui.PrintError("Invalid hours format: %s", hoursStr)
			return
		}

		if hours <= 0 {
			ui.PrintError("Hours must be positive")
			return
		}

		projectName, _ := parsePath(path)

		store := storage.Get()

		// Verify task exists
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		// Get date flag
		dateStr, _ := cmd.Flags().GetString("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		} else {
			// Validate date format
			if _, err := time.Parse("2006-01-02", dateStr); err != nil {
				ui.PrintError("Invalid date format. Use: YYYY-MM-DD")
				return
			}
		}

		// Log time
		entry := models.TimeEntry{
			Date:  dateStr,
			Hours: hours,
		}

		if err := store.AddTimeEntry(projectName, taskID, entry); err != nil {
			ui.PrintError("Failed to log time: %v", err)
			return
		}

		ui.PrintSuccess("Time logged")
		ui.Cyan.Printf("  Task: [%s] %s\n", taskID, task.Title)
		ui.Yellow.Printf("  Hours: %s\n", ui.FormatHours(hours))
		ui.Blue.Printf("  Date: %s\n", ui.FormatDate(dateStr))

		// Show updated totals
		newActual := task.CalculateActualHours() + hours

		fmt.Println()
		ui.Blue.Println("  Updated Totals:")
		if task.EstimatedHours > 0 {
			ui.Dim.Printf("    Estimated: %s\n", ui.FormatHours(task.EstimatedHours))
			ui.Cyan.Printf("    Actual:    %s\n", ui.FormatHours(newActual))

			variance := newActual - task.EstimatedHours
			if variance > 0 {
				ui.Red.Printf("    Variance:  +%s\n", ui.FormatHours(variance))
			} else if variance < 0 {
				ui.Green.Printf("    Variance:  %s\n", ui.FormatHours(variance))
			}
		} else {
			ui.Cyan.Printf("    Total: %s\n", ui.FormatHours(newActual))
		}
	},
}

var trackSwitchCmd = &cobra.Command{
	Use:   "switch <project[/module]> <task_id>",
	Short: "Stop current tracking and start tracking a different task",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		taskID := args[1]

		projectName, moduleName := parsePath(path)

		store := storage.Get()

		// Check if currently tracking
		tracking, err := store.IsTracking()
		if err != nil {
			ui.PrintError("Failed to check tracking status: %v", err)
			return
		}

		if tracking {
			session, _ := store.GetActiveSession()
			oldElapsed := time.Since(session.StartTime)

			// Stop current
			elapsed, oldPath, oldTaskID, err := store.StopTracking()
			if err != nil {
				ui.PrintError("Failed to stop current session: %v", err)
				return
			}

			ui.PrintSuccess("⏹️  Stopped: [%s] %s", oldTaskID, oldPath)
			ui.Cyan.Printf("  Duration: %s (%.2fh logged)\n",
				ui.FormatDuration(oldElapsed), elapsed.Hours())
			fmt.Println()
		}

		// Verify new task exists
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		// Start new session
		if err := store.StartTracking(projectName, moduleName, taskID); err != nil {
			ui.PrintError("Failed to start tracking: %v", err)
			return
		}

		ui.PrintSuccess("▶️  Started: [%s] %s", taskID, task.Title)

		if moduleName != "" {
			ui.Dim.Printf("  Path: %s/%s\n", projectName, moduleName)
		} else {
			ui.Dim.Printf("  Path: %s\n", projectName)
		}

		ui.Dim.Printf("  Time: %s\n", time.Now().Format("15:04:05"))
	},
}

var trackListCmd = &cobra.Command{
	Use:   "list <project> [date]",
	Short: "List time entries for a date",
	Long:  "Show all time entries for a specific date (defaults to today)",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]

		dateStr := time.Now().Format("2006-01-02")
		if len(args) > 1 {
			dateStr = args[1]
			// Validate date
			if _, err := time.Parse("2006-01-02", dateStr); err != nil {
				ui.PrintError("Invalid date format. Use: YYYY-MM-DD")
				return
			}
		}

		store := storage.Get()

		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}

		ui.PrintHeader(fmt.Sprintf("⏱️  Time Entries: %s", ui.FormatDate(dateStr)))
		ui.Dim.Printf("Project: %s\n\n", projectName)

		totalHours := 0.0
		entryCount := 0

		// Check all tasks
		for _, task := range project.GetAllTasks() {
			taskHours := 0.0

			for _, entry := range task.TimeEntries {
				if entry.Date == dateStr {
					taskHours += entry.Hours
					entryCount++
				}
			}

			if taskHours > 0 {
				totalHours += taskHours

				statusColor := ui.GetStatusColor(task.Status)
				statusColor.Printf("  %s [%s] %s\n",
					ui.GetStatusIcon(task.Status), task.ID, task.Title)
				ui.Cyan.Printf("    └─ %s\n", ui.FormatHours(taskHours))
			}
		}

		if entryCount == 0 {
			ui.PrintEmptyState(
				fmt.Sprintf("No time entries on %s", dateStr),
				"Log time with: qix track log <project> <task_id> <hours>",
			)
			return
		}

		fmt.Println()
		ui.PrintSeparator()
		ui.BoldGreen.Printf("Total: %s (%d entries)\n", ui.FormatHours(totalHours), entryCount)
	},
}

var trackSummaryCmd = &cobra.Command{
	Use:   "summary <project> [days]",
	Short: "Show time tracking summary",
	Long:  "Show time tracking summary for the last N days (default: 7)",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]

		days := 7
		if len(args) > 1 {
			if _, err := fmt.Sscanf(args[1], "%d", &days); err != nil || days <= 0 {
				ui.PrintError("Invalid days: %s", args[1])
				return
			}
		}

		store := storage.Get()

		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}

		ui.PrintHeader(fmt.Sprintf("⏱️  Time Summary: %s (Last %d days)", projectName, days))

		// Calculate date range
		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -days+1)

		// Collect daily totals
		dailyTotals := make(map[string]float64)

		for _, task := range project.GetAllTasks() {
			for _, entry := range task.TimeEntries {
				entryDate, err := time.Parse("2006-01-02", entry.Date)
				if err != nil {
					continue
				}

				if entryDate.After(startDate.AddDate(0, 0, -1)) && entryDate.Before(endDate.AddDate(0, 0, 1)) {
					dailyTotals[entry.Date] += entry.Hours
				}
			}
		}

		if len(dailyTotals) == 0 {
			ui.PrintEmptyState("No time entries in this period", "")
			return
		}

		// Create table
		table := ui.NewTableBuilder("Date", "Hours", "Bar").
			Align(1, ui.AlignRight)

		grandTotal := 0.0
		maxHours := 0.0

		// Find max for scaling
		for _, hours := range dailyTotals {
			if hours > maxHours {
				maxHours = hours
			}
			grandTotal += hours
		}

		// Add rows for each day
		current := startDate
		for current.Before(endDate.AddDate(0, 0, 1)) {
			dateStr := current.Format("2006-01-02")
			hours := dailyTotals[dateStr]

			// Create bar
			barWidth := 20
			filled := 0
			if maxHours > 0 {
				filled = int((hours / maxHours) * float64(barWidth))
			}

			bar := ""
			for i := 0; i < barWidth; i++ {
				if i < filled {
					bar += "█"
				} else {
					bar += "░"
				}
			}

			table.Row(
				ui.FormatDate(dateStr),
				ui.FormatHours(hours),
				bar,
			)

			current = current.AddDate(0, 0, 1)
		}

		table.PrintSimple()

		fmt.Println()
		ui.BoldGreen.Printf("Total: %s\n", ui.FormatHours(grandTotal))

		if days > 0 {
			avgPerDay := grandTotal / float64(days)
			ui.Cyan.Printf("Average: %s/day\n", ui.FormatHours(avgPerDay))
		}
	},
}

func init() {
	// track log flags
	trackLogCmd.Flags().StringP("date", "d", "", "Date for time entry (YYYY-MM-DD, defaults to today)")

	// Add subcommands
	trackCmd.AddCommand(trackStartCmd)
	trackCmd.AddCommand(trackStopCmd)
	trackCmd.AddCommand(trackStatusCmd)
	trackCmd.AddCommand(trackLogCmd)
	trackCmd.AddCommand(trackSwitchCmd)
	trackCmd.AddCommand(trackListCmd)
	trackCmd.AddCommand(trackSummaryCmd)
}
