package cmd

import (
	"fmt"
	"time"

	"github.com/mrbooshehri/qix-go/internal/models"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
	"github.com/spf13/cobra"
)

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Manage sprints",
	Long:  "Create, list, and manage sprint cycles",
}

var sprintCreateCmd = &cobra.Command{
	Use:   "create <project> <name> <start_date> <end_date>",
	Short: "Create a new sprint",
	Long:  "Create a sprint with start and end dates (format: YYYY-MM-DD)",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		sprintName := args[1]
		startDate := args[2]
		endDate := args[3]

		// Validate dates
		start, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			ui.PrintError("Invalid start date format. Use: YYYY-MM-DD")
			return
		}

		end, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			ui.PrintError("Invalid end date format. Use: YYYY-MM-DD")
			return
		}

		if end.Before(start) {
			ui.PrintError("End date must be after start date")
			return
		}

		store := storage.Get()

		sprint := models.Sprint{
			Name:      sprintName,
			StartDate: startDate,
			EndDate:   endDate,
		}

		if err := store.AddSprint(projectName, sprint); err != nil {
			ui.PrintError("Failed to create sprint: %v", err)
			return
		}

		duration := int(end.Sub(start).Hours() / 24)

		ui.PrintSuccess("Sprint '%s' created", sprintName)
		ui.Cyan.Printf("  Project: %s\n", projectName)
		ui.Blue.Printf("  Period:  %s → %s\n", ui.FormatDate(startDate), ui.FormatDate(endDate))
		ui.Yellow.Printf("  Duration: %d days\n", duration)

		// Show status
		today := time.Now().Format("2006-01-02")
		if today < startDate {
			ui.Cyan.Println("  Status:  📅 Upcoming")
		} else if today > endDate {
			ui.Green.Println("  Status:  ✅ Completed")
		} else {
			daysLeft := int(end.Sub(time.Now()).Hours() / 24)
			ui.Yellow.Printf("  Status:  🔄 Active (%d days remaining)\n", daysLeft)
		}
	},
}

var sprintListCmd = &cobra.Command{
	Use:   "list <project>",
	Short: "List all sprints",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]

		store := storage.Get()

		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}

		if len(project.Sprints) == 0 {
			ui.PrintEmptyState(
				fmt.Sprintf("No sprints in project '%s'", projectName),
				fmt.Sprintf("Create one with: qix sprint create %s <name> <start> <end>", projectName),
			)
			return
		}

		ui.PrintHeader(fmt.Sprintf("🏃 Sprints in '%s'", projectName))

		today := time.Now().Format("2006-01-02")

		// Group sprints by status
		var upcoming, active, completed []models.Sprint

		for _, sprint := range project.Sprints {
			if today < sprint.StartDate {
				upcoming = append(upcoming, sprint)
			} else if today > sprint.EndDate {
				completed = append(completed, sprint)
			} else {
				active = append(active, sprint)
			}
		}

		// Show active first
		if len(active) > 0 {
			ui.PrintSubHeader("🔄 Active")
			for _, sprint := range active {
				printSprintSummary(sprint, project, store)
			}
		}

		// Then upcoming
		if len(upcoming) > 0 {
			ui.PrintSubHeader("📅 Upcoming")
			for _, sprint := range upcoming {
				printSprintSummary(sprint, project, store)
			}
		}

		// Finally completed
		if len(completed) > 0 {
			ui.PrintSubHeader("✅ Completed")
			for _, sprint := range completed {
				printSprintSummary(sprint, project, store)
			}
		}
	},
}

var sprintAssignCmd = &cobra.Command{
	Use:   "assign <project> <sprint_name> <task_id>",
	Short: "Assign a task to a sprint",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		sprintName := args[1]
		taskID := args[2]

		store := storage.Get()

		// Verify sprint exists
		sprint, err := store.GetSprint(projectName, sprintName)
		if err != nil {
			ui.PrintError("Sprint not found: %v", err)
			return
		}

		// Verify task exists
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		// Assign task
		if err := store.AssignTaskToSprint(projectName, sprintName, taskID); err != nil {
			ui.PrintError("Failed to assign task: %v", err)
			return
		}

		ui.PrintSuccess("Task assigned to sprint")
		ui.Cyan.Printf("  Sprint: %s\n", sprintName)
		ui.Yellow.Printf("  Task:   [%s] %s\n", taskID, task.Title)
		ui.Blue.Printf("  Period: %s → %s\n",
			ui.FormatDate(sprint.StartDate),
			ui.FormatDate(sprint.EndDate))
	},
}

var sprintReportCmd = &cobra.Command{
	Use:   "report <project> <sprint_name>",
	Short: "Generate sprint report",
	Long:  "Show detailed sprint progress and metrics",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		sprintName := args[1]

		store := storage.Get()

		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}

		sprint, err := store.GetSprint(projectName, sprintName)
		if err != nil {
			ui.PrintError("Sprint not found: %v", err)
			return
		}

		// Use the beautiful UI function
		ui.PrintSprintReport(project, sprint)

		// Additional burndown info
		if len(sprint.TaskIDs) > 0 {
			ui.PrintSubHeader("📉 Burndown Analysis")

			// Calculate ideal vs actual
			start, _ := time.Parse("2006-01-02", sprint.StartDate)
			end, _ := time.Parse("2006-01-02", sprint.EndDate)
			totalDays := int(end.Sub(start).Hours()/24) + 1

			today := time.Now()
			daysPassed := int(today.Sub(start).Hours() / 24)
			if daysPassed < 0 {
				daysPassed = 0
			}
			if daysPassed > totalDays {
				daysPassed = totalDays
			}

			// Count completed tasks
			done := 0
			for _, taskID := range sprint.TaskIDs {
				task, _, err := store.FindTask(projectName, taskID)
				if err == nil && task.Status == models.StatusDone {
					done++
				}
			}

			total := len(sprint.TaskIDs)
			remaining := total - done

			// Ideal remaining
			idealRemaining := total - int(float64(total)*float64(daysPassed)/float64(totalDays))

			ui.Cyan.Printf("Days passed:      %d / %d\n", daysPassed, totalDays)
			ui.Green.Printf("Tasks completed:  %d / %d\n", done, total)
			ui.Yellow.Printf("Tasks remaining:  %d\n", remaining)
			ui.Blue.Printf("Ideal remaining:  %d\n", idealRemaining)

			fmt.Println()

			if remaining > idealRemaining {
				deficit := remaining - idealRemaining
				ui.Red.Printf("⚠️  Behind schedule by %d task(s)\n", deficit)
			} else if remaining < idealRemaining {
				ahead := idealRemaining - remaining
				ui.Green.Printf("✨ Ahead of schedule by %d task(s)\n", ahead)
			} else {
				ui.Green.Println("✅ On track!")
			}
		}
	},
}

var sprintRemoveCmd = &cobra.Command{
	Use:   "remove <project> <sprint_name>",
	Short: "Remove a sprint",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		sprintName := args[1]

		store := storage.Get()

		// Verify sprint exists
		sprint, err := store.GetSprint(projectName, sprintName)
		if err != nil {
			ui.PrintError("Sprint not found: %v", err)
			return
		}

		// Confirmation
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("⚠️  Delete sprint '%s' (%d tasks assigned)?\n",
				sprintName, len(sprint.TaskIDs))
			fmt.Print("Type 'yes' to confirm: ")

			var confirm string
			fmt.Scanln(&confirm)

			if confirm != "yes" {
				ui.PrintInfo("Deletion cancelled")
				return
			}
		}

		// Remove sprint
		err = store.UpdateProject(projectName, func(p *models.Project) error {
			for i, s := range p.Sprints {
				if s.Name == sprintName {
					p.Sprints = append(p.Sprints[:i], p.Sprints[i+1:]...)
					return nil
				}
			}
			return fmt.Errorf("sprint not found")
		})

		if err != nil {
			ui.PrintError("Failed to remove sprint: %v", err)
			return
		}

		ui.PrintSuccess("Sprint '%s' removed", sprintName)
		ui.Dim.Printf("  Note: Tasks were not deleted, only unassigned from sprint\n")
	},
}

var sprintUnassignCmd = &cobra.Command{
	Use:   "unassign <project> <sprint_name> <task_id>",
	Short: "Unassign a task from a sprint",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		sprintName := args[1]
		taskID := args[2]

		store := storage.Get()

		err := store.UpdateProject(projectName, func(p *models.Project) error {
			for i := range p.Sprints {
				if p.Sprints[i].Name == sprintName {
					// Remove task ID from list
					for j, id := range p.Sprints[i].TaskIDs {
						if id == taskID {
							p.Sprints[i].TaskIDs = append(
								p.Sprints[i].TaskIDs[:j],
								p.Sprints[i].TaskIDs[j+1:]...)
							return nil
						}
					}
					return fmt.Errorf("task not assigned to this sprint")
				}
			}
			return fmt.Errorf("sprint not found")
		})

		if err != nil {
			ui.PrintError("Failed to unassign task: %v", err)
			return
		}

		ui.PrintSuccess("Task [%s] unassigned from sprint '%s'", taskID, sprintName)
	},
}

// Helper function
func printSprintSummary(sprint models.Sprint, project *models.Project, store *storage.Storage) {
	ui.BoldCyan.Printf("\n• %s\n", sprint.Name)
	ui.Blue.Printf("  %s → %s",
		ui.FormatDate(sprint.StartDate),
		ui.FormatDate(sprint.EndDate))

	// Status indicator
	today := time.Now().Format("2006-01-02")
	end, _ := time.Parse("2006-01-02", sprint.EndDate)

	if today < sprint.StartDate {
		start, _ := time.Parse("2006-01-02", sprint.StartDate)
		daysUntil := int(start.Sub(time.Now()).Hours() / 24)
		ui.Cyan.Printf(" (starts in %d days)\n", daysUntil)
	} else if today > sprint.EndDate {
		ui.Green.Println(" (completed)")
	} else {
		daysLeft := int(end.Sub(time.Now()).Hours() / 24)
		ui.Yellow.Printf(" (%d days remaining)\n", daysLeft)
	}

	// Task stats
	taskCount := len(sprint.TaskIDs)
	ui.Dim.Printf("  Tasks: %d", taskCount)

	if taskCount > 0 {
		done := 0
		for _, taskID := range sprint.TaskIDs {
			task, _, err := store.FindTask(project.Name, taskID)
			if err == nil && task.Status == models.StatusDone {
				done++
			}
		}

		completion := float64(done) / float64(taskCount) * 100
		fmt.Print(" | Progress: ")
		ui.PrintProgressBar(completion, 20)
		fmt.Printf(" %d/%d\n", done, taskCount)
	} else {
		fmt.Println()
	}
}

func init() {
	// sprint remove flags
	sprintRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")

	sprintCreateCmd.ValidArgsFunction = projectArgCompletion
	sprintListCmd.ValidArgsFunction = projectArgCompletion
	sprintAssignCmd.ValidArgsFunction = sprintProjectSprintTaskArgCompletion
	sprintReportCmd.ValidArgsFunction = sprintProjectSprintArgCompletion
	sprintRemoveCmd.ValidArgsFunction = sprintProjectSprintArgCompletion
	sprintUnassignCmd.ValidArgsFunction = sprintProjectSprintTaskArgCompletion

	// Add subcommands
	sprintCmd.AddCommand(sprintCreateCmd)
	sprintCmd.AddCommand(sprintListCmd)
	sprintCmd.AddCommand(sprintAssignCmd)
	sprintCmd.AddCommand(sprintReportCmd)
	sprintCmd.AddCommand(sprintRemoveCmd)
	sprintCmd.AddCommand(sprintUnassignCmd)
}
