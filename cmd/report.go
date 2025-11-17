package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/mrbooshehri/qix-go/internal/models"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate reports",
	Long:  "Generate various reports: daily, project, KPI, WBS",
}

var reportDailyCmd = &cobra.Command{
	Use:   "daily [date]",
	Short: "Daily time report",
	Long:  "Show time entries for a specific date (defaults to today)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dateStr := time.Now().Format("2006-01-02")
		
		if len(args) > 0 {
			dateStr = args[0]
			// Validate date
			if _, err := time.Parse("2006-01-02", dateStr); err != nil {
				ui.PrintError("Invalid date format. Use: YYYY-MM-DD")
				return
			}
		}
		
		store := storage.Get()
		
		// Get all time entries for the date
		entriesByProject, err := store.GetTimeEntriesForDate(dateStr)
		if err != nil {
			ui.PrintError("Failed to get time entries: %v", err)
			return
		}
		
		// Calculate totals
		totalHours := 0.0
		for _, entries := range entriesByProject {
			for _, entry := range entries {
				totalHours += entry.Hours
			}
		}
		
		// Use the beautiful UI function
		ui.PrintDailyReport(dateStr, entriesByProject, totalHours)
		
		// Show active tracking session if today
		if dateStr == time.Now().Format("2006-01-02") {
			tracking, _ := store.IsTracking()
			if tracking {
				session, _ := store.GetActiveSession()
				elapsed := time.Since(session.StartTime)
				
				fmt.Println()
				ui.Yellow.Println("⏳ Active Session:")
				ui.Cyan.Printf("  Task: [%s] %s\n", session.TaskID, session.Path)
				ui.Green.Printf("  Elapsed: %s (%.2fh)\n", 
					ui.FormatDuration(elapsed), elapsed.Hours())
				ui.Dim.Println("  (Not yet logged - stop tracking to save)")
			}
		}
	},
}

var reportProjectCmd = &cobra.Command{
	Use:   "project <project> [from_date] [to_date]",
	Short: "Project performance report",
	Long:  "Generate a comprehensive project report for a date range",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		
		// Default date range: last 30 days
		endDate := time.Now().Format("2006-01-02")
		startDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
		
		if len(args) > 1 {
			startDate = args[1]
			if _, err := time.Parse("2006-01-02", startDate); err != nil {
				ui.PrintError("Invalid start date format. Use: YYYY-MM-DD")
				return
			}
		}
		
		if len(args) > 2 {
			endDate = args[2]
			if _, err := time.Parse("2006-01-02", endDate); err != nil {
				ui.PrintError("Invalid end date format. Use: YYYY-MM-DD")
				return
			}
		}
		
		store := storage.Get()
		
		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}
		
		// Use the beautiful UI function
		ui.PrintProjectReport(project, startDate, endDate)
		
		// Additional insights
		ui.PrintSubHeader("📈 Activity Breakdown")
		
		// Tasks completed in period
		completedInPeriod := 0
		for _, task := range project.GetAllTasks() {
			if task.Status == models.StatusDone {
				updatedDate := task.UpdatedAt.Format("2006-01-02")
				if updatedDate >= startDate && updatedDate <= endDate {
					completedInPeriod++
				}
			}
		}
		
		if completedInPeriod > 0 {
			ui.Green.Printf("Completed in period: %d tasks\n", completedInPeriod)
			
			// Calculate days in period
			start, _ := time.Parse("2006-01-02", startDate)
			end, _ := time.Parse("2006-01-02", endDate)
			days := int(end.Sub(start).Hours()/24) + 1
			
			if days > 0 {
				velocity := float64(completedInPeriod) / float64(days)
				ui.Cyan.Printf("Velocity: %.2f tasks/day\n", velocity)
			}
		}
		
		fmt.Println()
		
		// Top contributors (most time logged)
		ui.PrintSubHeader("⏱️  Most Time-Intensive Tasks")
		
		type taskHours struct {
			task  models.Task
			hours float64
		}
		
		var taskList []taskHours
		for _, task := range project.GetAllTasks() {
			hours := task.CalculateActualHours()
			if hours > 0 {
				taskList = append(taskList, taskHours{task, hours})
			}
		}
		
		// Sort by hours (simple bubble sort for small lists)
		for i := 0; i < len(taskList)-1; i++ {
			for j := 0; j < len(taskList)-i-1; j++ {
				if taskList[j].hours < taskList[j+1].hours {
					taskList[j], taskList[j+1] = taskList[j+1], taskList[j]
				}
			}
		}
		
		// Show top 5
		shown := 0
		for _, th := range taskList {
			if shown >= 5 {
				break
			}
			
			statusColor := ui.GetStatusColor(th.task.Status)
			statusColor.Printf("  %s [%s] %s\n", 
				ui.GetStatusIcon(th.task.Status), 
				th.task.ID, 
				th.task.Title)
			
			ui.Cyan.Printf("    └─ %s", ui.FormatHours(th.hours))
			
			if th.task.EstimatedHours > 0 {
				variance := th.hours - th.task.EstimatedHours
				if variance > 0 {
					ui.Red.Printf(" (+%s over)", ui.FormatHours(variance))
				} else {
					ui.Green.Printf(" (%s under)", ui.FormatHours(-variance))
				}
			}
			fmt.Println()
			
			shown++
		}
		
		if shown == 0 {
			ui.Dim.Println("  No time logged yet")
		}
	},
}

var reportKPICmd = &cobra.Command{
	Use:   "kpi <project>",
	Short: "KPI metrics report",
	Long:  "Generate Key Performance Indicators report with metrics and analysis",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		
		store := storage.Get()
		
		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}
		
		// Use the beautiful UI function
		ui.PrintKPIReport(project)
		
		// Additional KPIs
		ui.PrintSubHeader("📊 Additional Metrics")
		
		allTasks := project.GetAllTasks()
		
		// Tasks with dependencies
		withDeps := 0
		for _, task := range allTasks {
			if len(task.Dependencies) > 0 {
				withDeps++
			}
		}
		
		// Tasks with time entries
		withTime := 0
		for _, task := range allTasks {
			if len(task.TimeEntries) > 0 {
				withTime++
			}
		}
		
		// Recurring tasks
		recurring := 0
		for _, task := range allTasks {
			if task.IsRecurring() {
				recurring++
			}
		}
		
		table := ui.NewTableBuilder("Metric", "Count", "Percentage").
			Align(1, ui.AlignRight).
			Align(2, ui.AlignRight)
		
		if len(allTasks) > 0 {
			table.Row("Tasks with dependencies", 
				fmt.Sprintf("%d", withDeps),
				fmt.Sprintf("%.1f%%", float64(withDeps)/float64(len(allTasks))*100))
			
			table.Row("Tasks with time logged",
				fmt.Sprintf("%d", withTime),
				fmt.Sprintf("%.1f%%", float64(withTime)/float64(len(allTasks))*100))
			
			table.Row("Recurring tasks",
				fmt.Sprintf("%d", recurring),
				fmt.Sprintf("%.1f%%", float64(recurring)/float64(len(allTasks))*100))
		}
		
		table.PrintSimple()
		fmt.Println()
		
		// Health score
		ui.PrintSubHeader("💚 Project Health Score")
		
		score := 0.0
		maxScore := 0.0
		
		// Completion rate (30 points)
		completion := project.GetCompletionPercentage()
		score += (completion / 100.0) * 30.0
		maxScore += 30.0
		
		// Estimation accuracy (30 points)
		estimated := project.CalculateTotalEstimated()
		actual := project.CalculateTotalActual()
		if estimated > 0 {
			accuracy := 100.0
			variance := ((actual - estimated) / estimated) * 100
			if variance < 0 {
				accuracy = 100 + variance
			} else {
				accuracy = 100 - variance
			}
			if accuracy < 0 {
				accuracy = 0
			}
			score += (accuracy / 100.0) * 30.0
		}
		maxScore += 30.0
		
		// Task tracking adoption (20 points)
		if len(allTasks) > 0 {
			trackingRate := float64(withTime) / float64(len(allTasks)) * 100
			score += (trackingRate / 100.0) * 20.0
		}
		maxScore += 20.0
		
		// Active work (20 points) - balance between todo and doing
		counts := project.CountByStatus()
		active := counts[models.StatusDoing]
		if len(allTasks) > 0 {
			activeRate := float64(active) / float64(len(allTasks)) * 100
			// Optimal is around 20-40% active
			if activeRate >= 20 && activeRate <= 40 {
				score += 20.0
			} else if activeRate > 40 {
				score += 20.0 * (1.0 - (activeRate-40)/60.0)
			} else {
				score += 20.0 * (activeRate / 20.0)
			}
		}
		maxScore += 20.0
		
		healthScore := (score / maxScore) * 100
		
		fmt.Print("Health Score: ")
		ui.PrintProgressBar(healthScore, 50)
		
		if healthScore >= 80 {
			ui.Green.Printf(" %.1f%% - Excellent! 🎉\n", healthScore)
		} else if healthScore >= 60 {
			ui.Yellow.Printf(" %.1f%% - Good\n", healthScore)
		} else if healthScore >= 40 {
			ui.Magenta.Printf(" %.1f%% - Needs attention\n", healthScore)
		} else {
			ui.Red.Printf(" %.1f%% - Requires improvement\n", healthScore)
		}
		
		fmt.Println()
		
		// Recommendations
		if healthScore < 80 {
			ui.Yellow.Println("💡 Recommendations:")
			
			if completion < 20 {
				ui.Dim.Println("  • Focus on completing tasks to improve progress")
			}
			
			if withTime < len(allTasks)/2 {
				ui.Dim.Println("  • Track time more consistently for better insights")
			}
			
			if estimated > 0 && actual > estimated*1.5 {
				ui.Dim.Println("  • Review estimates - tasks are taking longer than expected")
			}
			
			if counts[models.StatusBlocked] > 0 {
				ui.Dim.Println("  • Address blocked tasks to maintain momentum")
			}
		}
	},
}

var reportWBSCmd = &cobra.Command{
	Use:   "wbs <project>",
	Short: "Work Breakdown Structure report",
	Long:  "Display the complete WBS with progress visualization",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		
		store := storage.Get()
		
		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}
		
		// Use the beautiful UI function
		ui.PrintWBSReport(project)
		
		// Show task relationships
		ui.PrintSubHeader("🔗 Task Dependencies")
		
		hasDependencies := false
		
		for _, task := range project.GetAllTasks() {
			if len(task.Dependencies) > 0 {
				hasDependencies = true
				
				statusColor := ui.GetStatusColor(task.Status)
				statusColor.Printf("  [%s] %s\n", task.ID, task.Title)
				
				for _, depID := range task.Dependencies {
					depTask, _, err := store.FindTask(projectName, depID)
					if err != nil {
						ui.Red.Printf("    ↳ [%s] (not found)\n", depID)
						continue
					}
					
					depColor := ui.GetStatusColor(depTask.Status)
					depColor.Printf("    ↳ %s [%s] %s\n", 
						ui.GetStatusIcon(depTask.Status),
						depID,
						depTask.Title)
				}
				fmt.Println()
			}
		}
		
		if !hasDependencies {
			ui.Dim.Println("  No task dependencies defined")
			fmt.Println()
		}
		
		// Show parent-child relationships
		ui.PrintSubHeader("👨‍👩‍👧 Task Hierarchy")
		
		hasHierarchy := false
		
		// Find root tasks (no parent)
		for _, task := range project.GetAllTasks() {
			if task.ParentID == "" {
				children, _ := store.GetChildTasks(projectName, task.ID)
				if len(children) > 0 {
					hasHierarchy = true
					
					statusColor := ui.GetStatusColor(task.Status)
					statusColor.Printf("  [%s] %s\n", task.ID, task.Title)
					
					for _, child := range children {
						childColor := ui.GetStatusColor(child.Status)
						childColor.Printf("    └─ %s [%s] %s\n",
							ui.GetStatusIcon(child.Status),
							child.ID,
							child.Title)
					}
					fmt.Println()
				}
			}
		}
		
		if !hasHierarchy {
			ui.Dim.Println("  No parent-child relationships defined")
			ui.Dim.Println("  Create with: qix task link <project> <child_id> <parent_id>")
			fmt.Println()
		}
	},
}

var reportCompareCmd = &cobra.Command{
	Use:   "compare <project1> <project2>",
	Short: "Compare two projects",
	Long:  "Side-by-side comparison of project metrics",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		project1Name := args[0]
		project2Name := args[1]
		
		store := storage.Get()
		
		project1, err := store.LoadProject(project1Name)
		if err != nil {
			ui.PrintError("Project not found: %s", project1Name)
			return
		}
		
		project2, err := store.LoadProject(project2Name)
		if err != nil {
			ui.PrintError("Project not found: %s", project2Name)
			return
		}
		
		ui.PrintHeader("📊 Project Comparison")
		
		// Build comparison table
		table := ui.NewTableBuilder("Metric", project1Name, project2Name).
			Align(1, ui.AlignRight).
			Align(2, ui.AlignRight)
		
		// Task counts
		counts1 := project1.CountByStatus()
		counts2 := project2.CountByStatus()
		
		table.Row("Total Tasks",
			fmt.Sprintf("%d", len(project1.GetAllTasks())),
			fmt.Sprintf("%d", len(project2.GetAllTasks())))
		
		table.Row("Completed",
			fmt.Sprintf("%d", counts1[models.StatusDone]),
			fmt.Sprintf("%d", counts2[models.StatusDone]))
		
		table.Row("In Progress",
			fmt.Sprintf("%d", counts1[models.StatusDoing]),
			fmt.Sprintf("%d", counts2[models.StatusDoing]))
		
		table.Row("Blocked",
			fmt.Sprintf("%d", counts1[models.StatusBlocked]),
			fmt.Sprintf("%d", counts2[models.StatusBlocked]))
		
		table.Row("", "", "")
		
		// Completion rates
		completion1 := project1.GetCompletionPercentage()
		completion2 := project2.GetCompletionPercentage()
		
		table.Row("Completion",
			fmt.Sprintf("%.1f%%", completion1),
			fmt.Sprintf("%.1f%%", completion2))
		
		table.Row("", "", "")
		
		// Time tracking
		est1 := project1.CalculateTotalEstimated()
		est2 := project2.CalculateTotalEstimated()
		act1 := project1.CalculateTotalActual()
		act2 := project2.CalculateTotalActual()
		
		table.Row("Estimated Hours",
			ui.FormatHours(est1),
			ui.FormatHours(est2))
		
		table.Row("Actual Hours",
			ui.FormatHours(act1),
			ui.FormatHours(act2))
		
		if est1 > 0 && est2 > 0 {
			eff1 := (est1 / act1) * 100
			eff2 := (est2 / act2) * 100
			
			table.Row("Efficiency",
				fmt.Sprintf("%.1f%%", eff1),
				fmt.Sprintf("%.1f%%", eff2))
		}
		
		table.Row("", "", "")
		
		// Structure
		table.Row("Modules",
			fmt.Sprintf("%d", len(project1.Modules)),
			fmt.Sprintf("%d", len(project2.Modules)))
		
		table.Row("Sprints",
			fmt.Sprintf("%d", len(project1.Sprints)),
			fmt.Sprintf("%d", len(project2.Sprints)))
		
		table.PrintSimple()
		
		fmt.Println()
		
		// Visual comparison
		ui.PrintSubHeader("📈 Completion Comparison")
		
		fmt.Printf("%-20s ", project1Name)
		ui.PrintProgressBar(completion1, 40)
		fmt.Printf(" %.1f%%\n", completion1)
		
		fmt.Printf("%-20s ", project2Name)
		ui.PrintProgressBar(completion2, 40)
		fmt.Printf(" %.1f%%\n", completion2)
	},
}

var reportTimelineCmd = &cobra.Command{
	Use:   "timeline <project> [days]",
	Short: "Activity timeline report",
	Long:  "Show task activity over time (default: last 14 days)",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		
		days := 14
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
		
		ui.PrintHeader(fmt.Sprintf("📅 Activity Timeline: %s (Last %d days)", projectName, days))
		
		// Collect activity by day
		endDate := time.Now()
		startDate := endDate.AddDate(0, 0, -days+1)
		
		// Track task updates by day
		type dayActivity struct {
			date      string
			completed int
			started   int
			updated   int
		}
		
		activities := make([]dayActivity, days)
		
		for i := 0; i < days; i++ {
			date := startDate.AddDate(0, 0, i)
			dateStr := date.Format("2006-01-02")
			
			activity := dayActivity{date: dateStr}
			
			for _, task := range project.GetAllTasks() {
				taskDate := task.UpdatedAt.Format("2006-01-02")
				
				if taskDate == dateStr {
					if task.Status == models.StatusDone {
						activity.completed++
					} else if task.Status == models.StatusDoing {
						activity.started++
					} else {
						activity.updated++
					}
				}
			}
			
			activities[i] = activity
		}
		
		// Display timeline
		for _, act := range activities {
			fmt.Printf("%s  ", ui.FormatDate(act.date))
			
			total := act.completed + act.started + act.updated
			
			if total > 0 {
				// Show activity bar
				bar := ""
				for i := 0; i < act.completed; i++ {
					bar += "●"
				}
				for i := 0; i < act.started; i++ {
					bar += "◐"
				}
				for i := 0; i < act.updated; i++ {
					bar += "○"
				}
				
				ui.Green.Print(bar)
				ui.Dim.Printf(" (%d)", total)
			} else {
				ui.Dim.Print("─")
			}
			
			fmt.Println()
		}
		
		fmt.Println()
		ui.Green.Print("● Completed  ")
		ui.Cyan.Print("◐ Started  ")
		ui.Yellow.Println("○ Updated")
	},
}

func init() {
	// Add subcommands
	reportCmd.AddCommand(reportDailyCmd)
	reportCmd.AddCommand(reportProjectCmd)
	reportCmd.AddCommand(reportKPICmd)
	reportCmd.AddCommand(reportWBSCmd)
	reportCmd.AddCommand(reportCompareCmd)
	reportCmd.AddCommand(reportTimelineCmd)
}