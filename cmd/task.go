package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mrbooshehri/qix-go/internal/models"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long:  "Create, list, update, and manage tasks within projects and modules",
}

var taskCreateCmd = &cobra.Command{
	Use:   "create <project[/module]> <title>",
	Short: "Create a new task",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		title := strings.Join(args[1:], " ")

		// Parse path
		projectName, moduleName := parsePath(path)

		// Get flags
		description, _ := cmd.Flags().GetString("description")
		status, _ := cmd.Flags().GetString("status")
		priority, _ := cmd.Flags().GetString("priority")
		estimated, _ := cmd.Flags().GetFloat64("estimated")
		jiraIssue, _ := cmd.Flags().GetString("jira-issue")
		tags, _ := cmd.Flags().GetStringSlice("tags")
		interactive, _ := cmd.Flags().GetBool("interactive")

		// Validate status
		taskStatus := models.StatusTodo
		if status != "" {
			switch status {
			case "todo":
				taskStatus = models.StatusTodo
			case "doing":
				taskStatus = models.StatusDoing
			case "done":
				taskStatus = models.StatusDone
			case "blocked":
				taskStatus = models.StatusBlocked
			default:
				ui.PrintError("Invalid status. Use: todo, doing, done, blocked")
				return
			}
		}

		// Validate priority
		taskPriority := models.PriorityMedium
		if priority != "" {
			switch priority {
			case "low":
				taskPriority = models.PriorityLow
			case "medium":
				taskPriority = models.PriorityMedium
			case "high":
				taskPriority = models.PriorityHigh
			default:
				ui.PrintError("Invalid priority. Use: low, medium, high")
				return
			}
		}

		// Create task
		task := models.Task{
			Title:          title,
			Description:    description,
			Status:         taskStatus,
			Priority:       taskPriority,
			EstimatedHours: estimated,
			Tags:           tags,
			JiraIssue:      strings.TrimSpace(jiraIssue),
		}

		if interactive {
			if err := runInteractiveTaskCreate(&task); err != nil {
				ui.PrintError("Failed to gather task details: %v", err)
				return
			}
		}

		if task.ID == "" {
			task.ID = storage.GenerateTaskID()
		}

		store := storage.Get()

		if err := store.AddTask(projectName, moduleName, task); err != nil {
			ui.PrintError("Failed to create task: %v", err)
			return
		}

		ui.PrintSuccess("Task created with ID: %s", task.ID)
		ui.Dim.Printf("  Title: %s\n", title)

		if moduleName != "" {
			ui.Dim.Printf("  Location: %s/%s\n", projectName, moduleName)
		} else {
			ui.Dim.Printf("  Location: %s (project level)\n", projectName)
		}

		ui.Dim.Printf("  Status: %s | Priority: %s\n", taskStatus, taskPriority)

		if estimated > 0 {
			ui.Dim.Printf("  Estimated: %s\n", ui.FormatHours(estimated))
		}
		if jiraIssue != "" {
			ui.Dim.Printf("  Jira: %s\n", jiraIssue)
		}
	},
}

var taskListCmd = &cobra.Command{
	Use:   "list <project[/module]>",
	Short: "List tasks",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		projectName, moduleName := parsePath(path)

		all, _ := cmd.Flags().GetBool("all")
		status, _ := cmd.Flags().GetString("status")

		store := storage.Get()

		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}

		var tasks []models.Task

		if moduleName != "" {
			// List tasks in specific module
			moduleTasks, err := store.ListTasksInModule(projectName, moduleName)
			if err != nil {
				ui.PrintError("Module not found: %v", err)
				return
			}
			tasks = moduleTasks

			ui.PrintHeader(fmt.Sprintf("📋 Tasks in %s/%s", projectName, moduleName))
		} else if all {
			// List all tasks recursively
			tasks = project.GetAllTasks()
			ui.PrintHeader(fmt.Sprintf("📋 All Tasks in %s", projectName))
		} else {
			// List project-level tasks only
			tasks = project.Tasks
			ui.PrintHeader(fmt.Sprintf("📋 Project-Level Tasks in %s", projectName))
		}

		// Filter by status if specified
		if status != "" {
			var filtered []models.Task
			for _, task := range tasks {
				if string(task.Status) == status {
					filtered = append(filtered, task)
				}
			}
			tasks = filtered
		}

		if len(tasks) == 0 {
			msg := fmt.Sprintf("No tasks found in %s", path)
			if status != "" {
				msg = fmt.Sprintf("No %s tasks found in %s", status, path)
			}
			ui.PrintEmptyState(msg, fmt.Sprintf("Create one with: qix task create %s <title>", path))
			return
		}

		// Group by status
		byStatus := make(map[models.TaskStatus][]models.Task)
		for _, task := range tasks {
			byStatus[task.Status] = append(byStatus[task.Status], task)
		}

		// Print by status
		statusOrder := []models.TaskStatus{
			models.StatusDoing,
			models.StatusTodo,
			models.StatusBlocked,
			models.StatusDone,
		}

		for _, st := range statusOrder {
			if len(byStatus[st]) == 0 {
				continue
			}

			statusColor := ui.GetStatusColor(st)
			statusIcon := ui.GetStatusIcon(st)

			fmt.Println()
			statusColor.Printf("%s %s (%d)\n", statusIcon, st, len(byStatus[st]))
			ui.PrintSeparator()

			for _, task := range byStatus[st] {
				ui.PrintTask(task, "  ")
			}
		}

		fmt.Println()
	},
}

var taskShowCmd = &cobra.Command{
	Use:   "show <project> <task_id>",
	Short: "Show task details",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]

		store := storage.Get()

		task, location, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		ui.PrintTaskDetailed(*task, formatTaskLocation(projectName, location))

		// Show parent task if exists
		if task.ParentID != "" {
			parentTask, _, err := store.FindTask(projectName, task.ParentID)
			if err == nil {
				fmt.Println()
				ui.BoldBlue.Println("👨‍👩‍👧 Parent Task:")
				ui.Magenta.Printf("   [%s] %s\n", parentTask.ID, parentTask.Title)
			}
		}

		// Show child tasks
		children, err := store.GetChildTasks(projectName, taskID)
		if err == nil && len(children) > 0 {
			fmt.Println()
			ui.BoldBlue.Println("👶 Child Tasks:")
			for _, child := range children {
				statusColor := ui.GetStatusColor(child.Status)
				statusColor.Printf("   %s [%s] %s [%s]\n",
					ui.GetStatusIcon(child.Status),
					child.ID,
					child.Title,
					child.Status)
			}
		}

		// Show dependent tasks
		dependents, err := store.GetDependentTasks(projectName, taskID)
		if err == nil && len(dependents) > 0 {
			fmt.Println()
			ui.BoldBlue.Println("🔒 Blocking Tasks:")
			for _, dep := range dependents {
				ui.Red.Printf("   🔒 [%s] %s\n", dep.ID, dep.Title)
			}
		}
	},
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update <project> <task_id> <status>",
	Short: "Update task status",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]
		statusStr := args[2]

		// Validate status
		var status models.TaskStatus
		switch statusStr {
		case "todo":
			status = models.StatusTodo
		case "doing":
			status = models.StatusDoing
		case "done":
			status = models.StatusDone
		case "blocked":
			status = models.StatusBlocked
		default:
			ui.PrintError("Invalid status. Use: todo, doing, done, blocked")
			return
		}

		store := storage.Get()

		// Get task first to show before/after
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		oldStatus := task.Status

		if err := store.UpdateTaskStatus(projectName, taskID, status); err != nil {
			ui.PrintError("Failed to update task: %v", err)
			return
		}

		ui.PrintSuccess("Task status updated")
		ui.Cyan.Printf("  [%s] %s\n", taskID, task.Title)

		oldColor := ui.GetStatusColor(oldStatus)
		newColor := ui.GetStatusColor(status)

		fmt.Print("  ")
		oldColor.Printf("%s %s", ui.GetStatusIcon(oldStatus), oldStatus)
		fmt.Print(" → ")
		newColor.Printf("%s %s\n", ui.GetStatusIcon(status), status)
	},
}

var taskEditCmd = &cobra.Command{
	Use:   "edit <project> <task_id>",
	Short: "Edit task details",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]

		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		status, _ := cmd.Flags().GetString("status")
		priority, _ := cmd.Flags().GetString("priority")
		estimated, _ := cmd.Flags().GetFloat64("estimated")
		jiraIssue, _ := cmd.Flags().GetString("jira-issue")
		jiraIssueChanged := cmd.Flags().Changed("jira-issue")

		if title == "" && description == "" && status == "" && priority == "" && estimated == 0 && !jiraIssueChanged {
			if err := runInteractiveTaskEdit(projectName, taskID); err != nil {
				ui.PrintError("Failed to update task: %v", err)
			}
			return
		}

		store := storage.Get()

		err := store.UpdateTask(projectName, taskID, func(t *models.Task) error {
			if title != "" {
				t.Title = title
			}
			if description != "" {
				t.Description = description
			}
			if status != "" {
				switch status {
				case "todo":
					t.Status = models.StatusTodo
				case "doing":
					t.Status = models.StatusDoing
				case "done":
					t.Status = models.StatusDone
				case "blocked":
					t.Status = models.StatusBlocked
				default:
					return fmt.Errorf("invalid status: %s", status)
				}
			}
			if priority != "" {
				switch priority {
				case "low":
					t.Priority = models.PriorityLow
				case "medium":
					t.Priority = models.PriorityMedium
				case "high":
					t.Priority = models.PriorityHigh
				default:
					return fmt.Errorf("invalid priority: %s", priority)
				}
			}
			if estimated > 0 {
				t.EstimatedHours = estimated
			}
			if jiraIssueChanged {
				t.JiraIssue = strings.TrimSpace(jiraIssue)
			}
			return nil
		})

		if err != nil {
			ui.PrintError("Failed to update task: %v", err)
			return
		}

		ui.PrintSuccess("Task updated: %s", taskID)
	},
}

var taskRemoveCmd = &cobra.Command{
	Use:   "remove <project> <task_id>",
	Short: "Remove a task",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]

		store := storage.Get()

		// Get task details first
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		// Confirmation
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("⚠️  Delete task '%s' [%s]?\n", task.Title, taskID)
			fmt.Print("Type 'yes' to confirm: ")

			var confirm string
			fmt.Scanln(&confirm)

			if confirm != "yes" {
				ui.PrintInfo("Deletion cancelled")
				return
			}
		}

		if err := store.RemoveTask(projectName, taskID); err != nil {
			ui.PrintError("Failed to remove task: %v", err)
			return
		}

		ui.PrintSuccess("Task removed: [%s] %s", taskID, task.Title)
	},
}

var taskLinkCmd = &cobra.Command{
	Use:   "link <project> <child_id> <parent_id>",
	Short: "Link a task as child of another",
	Long:  "Create a parent-child relationship between tasks",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		childID := args[1]
		parentID := args[2]

		store := storage.Get()

		// Get task details
		childTask, _, err := store.FindTask(projectName, childID)
		if err != nil {
			ui.PrintError("Child task not found: %v", err)
			return
		}

		parentTask, _, err := store.FindTask(projectName, parentID)
		if err != nil {
			ui.PrintError("Parent task not found: %v", err)
			return
		}

		if err := store.LinkTaskAsChild(projectName, childID, parentID); err != nil {
			ui.PrintError("Failed to link tasks: %v", err)
			return
		}

		ui.PrintSuccess("Task linked successfully")
		ui.Cyan.Printf("  Child:  [%s] %s\n", childID, childTask.Title)
		ui.Magenta.Printf("  Parent: [%s] %s\n", parentID, parentTask.Title)
	},
}

var taskDependCmd = &cobra.Command{
	Use:   "depend <project> <task_id> <depends_on_id>",
	Short: "Add a task dependency",
	Long:  "Make a task depend on another (task_id will be blocked until depends_on_id is done)",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]
		dependsOnID := args[2]

		store := storage.Get()

		// Get task details
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		depTask, _, err := store.FindTask(projectName, dependsOnID)
		if err != nil {
			ui.PrintError("Dependency task not found: %v", err)
			return
		}

		if err := store.AddTaskDependency(projectName, taskID, dependsOnID); err != nil {
			ui.PrintError("Failed to add dependency: %v", err)
			return
		}

		ui.PrintSuccess("Dependency added")
		ui.Yellow.Printf("  [%s] %s\n", taskID, task.Title)
		ui.Cyan.Print("  ↓ depends on\n")
		ui.Green.Printf("  [%s] %s\n", dependsOnID, depTask.Title)

		if depTask.Status != models.StatusDone {
			ui.PrintWarning("Note: [%s] is not done yet (%s)", dependsOnID, depTask.Status)
		}
	},
}

var taskRecurCmd = &cobra.Command{
	Use:   "recur <project> <task_id> <pattern>",
	Short: "Set task as recurring",
	Long: `Set a task to recur automatically.

Patterns:
  daily                    - Every day
  weekly:<day>             - Every week (monday, tuesday, etc.)
  monthly:<day>            - Every month (1-31)
  interval:<days>          - Every N days

Examples:
  qix task recur myproject task123 daily
  qix task recur myproject task456 weekly:friday
  qix task recur myproject task789 monthly:15
  qix task recur myproject taskabc interval:3`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]
		pattern := args[2]

		// Parse pattern
		recurrence, err := parseRecurrencePattern(pattern)
		if err != nil {
			ui.PrintError("Invalid pattern: %v", err)
			return
		}

		store := storage.Get()

		// Get task
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		if err := store.SetTaskRecurrence(projectName, taskID, *recurrence); err != nil {
			ui.PrintError("Failed to set recurrence: %v", err)
			return
		}

		ui.PrintSuccess("Recurring schedule set")
		ui.Cyan.Printf("  Task: [%s] %s\n", taskID, task.Title)
		ui.Yellow.Printf("  Pattern: %s\n", pattern)
		ui.Green.Printf("  Next due: %s\n", ui.FormatDate(recurrence.NextDue))
	},
}

var taskUnrecurCmd = &cobra.Command{
	Use:   "unrecur <project> <task_id>",
	Short: "Remove recurrence from task",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]

		store := storage.Get()

		if err := store.RemoveTaskRecurrence(projectName, taskID); err != nil {
			ui.PrintError("Failed to remove recurrence: %v", err)
			return
		}

		ui.PrintSuccess("Recurrence removed from task: %s", taskID)
	},
}

var taskDueCmd = &cobra.Command{
	Use:   "due [project]",
	Short: "Show recurring tasks due today",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		today := time.Now().Format("2006-01-02")

		store := storage.Get()

		var projects []string
		var err error

		if len(args) > 0 {
			projects = []string{args[0]}
		} else {
			projects, err = store.ListProjects()
			if err != nil {
				ui.PrintError("Failed to list projects: %v", err)
				return
			}
		}

		ui.PrintHeader(fmt.Sprintf("🔔 Tasks Due Today - %s", ui.FormatDate(today)))

		found := false

		for _, projectName := range projects {
			tasks, err := store.GetRecurringTasksDue(projectName, today)
			if err != nil {
				continue
			}

			if len(tasks) > 0 {
				found = true
				ui.PrintSubHeader(fmt.Sprintf("📁 %s", projectName))

				for _, task := range tasks {
					ui.Yellow.Printf("  🔔 [%s] %s\n", task.ID, task.Title)

					if task.Recurrence != nil {
						pattern := string(task.Recurrence.Type)
						if task.Recurrence.Value != "" {
							pattern += ":" + task.Recurrence.Value
						}
						ui.Cyan.Printf("     📅 %s\n", pattern)
					}
				}
				fmt.Println()
			}
		}

		if !found {
			ui.PrintEmptyState("No recurring tasks due today", "")
		}
	},
}

var taskCompleteCmd = &cobra.Command{
	Use:   "complete <project> <task_id>",
	Short: "Complete a recurring task",
	Long:  "Mark a recurring task as done and schedule the next occurrence",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]

		store := storage.Get()

		// Get task
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		// Check if recurring
		if !task.IsRecurring() {
			// Just update status
			if err := store.UpdateTaskStatus(projectName, taskID, models.StatusDone); err != nil {
				ui.PrintError("Failed to complete task: %v", err)
				return
			}

			ui.PrintSuccess("Task completed: [%s] %s", taskID, task.Title)
			return
		}

		// Handle recurring task
		today := time.Now().Format("2006-01-02")

		// Calculate next occurrence
		nextDue := calculateNextOccurrence(task.Recurrence.Type, task.Recurrence.Value)

		// Update task
		err = store.UpdateTask(projectName, taskID, func(t *models.Task) error {
			t.Status = models.StatusDone
			if t.Recurrence != nil {
				t.Recurrence.LastCompleted = today
				t.Recurrence.NextDue = nextDue
			}
			return nil
		})

		if err != nil {
			ui.PrintError("Failed to complete task: %v", err)
			return
		}

		ui.PrintSuccess("Recurring task completed")
		ui.Cyan.Printf("  Task: [%s] %s\n", taskID, task.Title)
		ui.Green.Printf("  Completed: %s\n", ui.FormatDate(today))
		ui.Yellow.Printf("  Next due: %s\n", ui.FormatDate(nextDue))
	},
}

// Helper functions

func taskPathCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeProjectModulePaths(toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func taskDueCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeProjectNames(toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func runInteractiveTaskEdit(projectName, taskID string) error {
	store := storage.Get()
	task, _, err := store.FindTask(projectName, taskID)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println()
	ui.PrintHeader("Interactive Task Editor")
	fmt.Printf("Editing [%s] %s\n", task.ID, task.Title)
	fmt.Println("Press Enter to keep the current value. Type '-' to clear Jira Issue.")
	fmt.Println()

	edited := *task

	edited.Title = promptWithDefault(reader, "Title", task.Title)
	edited.Description = promptWithDefault(reader, "Description", task.Description)
	edited.Status = promptStatus(reader, task.Status)
	edited.Priority = promptPriority(reader, task.Priority)
	edited.EstimatedHours = promptEstimated(reader, task.EstimatedHours)
	edited.JiraIssue = promptJira(reader, task.JiraIssue)

	if err := store.UpdateTask(projectName, taskID, func(t *models.Task) error {
		t.Title = edited.Title
		t.Description = edited.Description
		t.Status = edited.Status
		t.Priority = edited.Priority
		t.EstimatedHours = edited.EstimatedHours
		t.JiraIssue = edited.JiraIssue
		return nil
	}); err != nil {
		return err
	}

	ui.PrintSuccess("Task updated: %s", taskID)
	return nil
}

func runInteractiveTaskCreate(task *models.Task) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println()
	ui.PrintHeader("Interactive Task Creator")
	fmt.Println("Provide values for the following fields (press Enter to keep defaults).")
	fmt.Println()

	task.Description = promptWithDefault(reader, "Description", task.Description)
	task.EstimatedHours = promptEstimated(reader, task.EstimatedHours)
	task.Status = promptStatus(reader, task.Status)
	task.Priority = promptPriority(reader, task.Priority)
	task.Tags = promptTags(reader, task.Tags)
	task.JiraIssue = promptJira(reader, task.JiraIssue)

	return nil
}

func promptWithDefault(reader *bufio.Reader, label, current string) string {
	display := current
	if display == "" {
		display = "<empty>"
	}
	fmt.Printf("%s [%s]: ", label, display)
	input, _ := reader.ReadString('\n')
	value := strings.TrimSpace(input)
	if value == "" {
		return current
	}
	return value
}

func promptStatus(reader *bufio.Reader, current models.TaskStatus) models.TaskStatus {
	for {
		fmt.Printf("Status [%s] (todo/doing/done/blocked): ", current)
		input, _ := reader.ReadString('\n')
		value := strings.TrimSpace(strings.ToLower(input))
		if value == "" {
			return current
		}
		switch value {
		case "todo":
			return models.StatusTodo
		case "doing":
			return models.StatusDoing
		case "done":
			return models.StatusDone
		case "blocked":
			return models.StatusBlocked
		default:
			fmt.Println("Invalid status. Use todo, doing, done, or blocked.")
		}
	}
}

func promptPriority(reader *bufio.Reader, current models.Priority) models.Priority {
	for {
		fmt.Printf("Priority [%s] (low/medium/high): ", current)
		input, _ := reader.ReadString('\n')
		value := strings.TrimSpace(strings.ToLower(input))
		if value == "" {
			return current
		}
		switch value {
		case "low":
			return models.PriorityLow
		case "medium":
			return models.PriorityMedium
		case "high":
			return models.PriorityHigh
		default:
			fmt.Println("Invalid priority. Use low, medium, or high.")
		}
	}
}

func promptEstimated(reader *bufio.Reader, current float64) float64 {
	for {
		fmt.Printf("Estimated Hours [%.2f]: ", current)
		input, _ := reader.ReadString('\n')
		value := strings.TrimSpace(input)
		if value == "" {
			return current
		}
		val, err := strconv.ParseFloat(value, 64)
		if err != nil || val < 0 {
			fmt.Println("Enter a positive number (e.g., 1.5).")
			continue
		}
		return val
	}
}

func promptJira(reader *bufio.Reader, current string) string {
	display := current
	if display == "" {
		display = "<none>"
	}
	for {
		fmt.Printf("Jira Issue ID [%s] (use '-' to clear): ", display)
		input, _ := reader.ReadString('\n')
		value := strings.TrimSpace(input)
		switch value {
		case "":
			return current
		case "-":
			return ""
		default:
			return value
		}
	}
}

func promptTags(reader *bufio.Reader, current []string) []string {
	display := "<none>"
	if len(current) > 0 {
		display = strings.Join(current, ", ")
	}

	fmt.Printf("Tags (comma-separated) [%s]: ", display)
	input, _ := reader.ReadString('\n')
	value := strings.TrimSpace(input)
	if value == "" {
		return current
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

func parsePath(path string) (project, module string) {
	parts := strings.SplitN(path, "/", 2)
	project = parts[0]
	if len(parts) > 1 {
		module = parts[1]
	}
	return
}

func formatTaskLocation(projectName, location string) string {
	if location == "project" {
		return fmt.Sprintf("%s (project level)", projectName)
	}
	moduleName := strings.TrimPrefix(location, "module:")
	return fmt.Sprintf("%s/%s", projectName, moduleName)
}

func parseRecurrencePattern(pattern string) (*models.Recurrence, error) {
	parts := strings.SplitN(pattern, ":", 2)
	recType := parts[0]
	recValue := ""
	if len(parts) > 1 {
		recValue = parts[1]
	}

	var rType models.RecurrenceType

	switch recType {
	case "daily":
		rType = models.RecurDaily
	case "weekly":
		rType = models.RecurWeekly
		if recValue == "" {
			return nil, fmt.Errorf("weekly pattern requires day (e.g., weekly:monday)")
		}
	case "monthly":
		rType = models.RecurMonthly
		if recValue == "" {
			return nil, fmt.Errorf("monthly pattern requires day number (e.g., monthly:15)")
		}
		day, err := strconv.Atoi(recValue)
		if err != nil || day < 1 || day > 31 {
			return nil, fmt.Errorf("monthly day must be 1-31")
		}
	case "interval":
		rType = models.RecurInterval
		if recValue == "" {
			return nil, fmt.Errorf("interval pattern requires number of days (e.g., interval:3)")
		}
		days, err := strconv.Atoi(recValue)
		if err != nil || days < 1 {
			return nil, fmt.Errorf("interval must be a positive number")
		}
	default:
		return nil, fmt.Errorf("unknown pattern type: %s (use: daily, weekly, monthly, interval)", recType)
	}

	nextDue := calculateNextOccurrence(rType, recValue)

	return &models.Recurrence{
		Type:    rType,
		Value:   recValue,
		NextDue: nextDue,
		Enabled: true,
	}, nil
}

func calculateNextOccurrence(recType models.RecurrenceType, value string) string {
	now := time.Now()

	switch recType {
	case models.RecurDaily:
		return now.AddDate(0, 0, 1).Format("2006-01-02")

	case models.RecurWeekly:
		// Find next occurrence of the specified day
		targetDay := value
		daysOfWeek := map[string]time.Weekday{
			"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
			"wednesday": time.Wednesday, "thursday": time.Thursday,
			"friday": time.Friday, "saturday": time.Saturday,
		}

		target, ok := daysOfWeek[strings.ToLower(targetDay)]
		if !ok {
			return now.Format("2006-01-02")
		}

		daysUntil := (int(target) - int(now.Weekday()) + 7) % 7
		if daysUntil == 0 {
			daysUntil = 7 // Next week
		}

		return now.AddDate(0, 0, daysUntil).Format("2006-01-02")

	case models.RecurMonthly:
		day, _ := strconv.Atoi(value)
		nextMonth := now.AddDate(0, 1, 0)

		// Handle months with fewer days
		lastDay := time.Date(nextMonth.Year(), nextMonth.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		if day > lastDay {
			day = lastDay
		}

		return time.Date(nextMonth.Year(), nextMonth.Month(), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	case models.RecurInterval:
		days, _ := strconv.Atoi(value)
		return now.AddDate(0, 0, days).Format("2006-01-02")
	}

	return now.Format("2006-01-02")
}

func init() {
	// task create flags
	taskCreateCmd.Flags().StringP("description", "d", "", "Task description")
	taskCreateCmd.Flags().StringP("status", "s", "todo", "Task status (todo/doing/done/blocked)")
	taskCreateCmd.Flags().StringP("priority", "p", "medium", "Task priority (low/medium/high)")
	taskCreateCmd.Flags().Float64P("estimated", "e", 0, "Estimated hours")
	taskCreateCmd.Flags().StringSliceP("tags", "t", []string{}, "Task tags")
	taskCreateCmd.Flags().String("jira-issue", "", "Jira issue ID")
	taskCreateCmd.Flags().BoolP("interactive", "i", false, "Interactive mode to enter task details")
	taskCreateCmd.ValidArgsFunction = taskPathCompletion

	// task list flags
	taskListCmd.Flags().BoolP("all", "a", false, "Show all tasks recursively")
	taskListCmd.Flags().StringP("status", "s", "", "Filter by status")
	taskListCmd.ValidArgsFunction = taskPathCompletion

	taskShowCmd.ValidArgsFunction = projectTaskArgCompletion
	taskUpdateCmd.ValidArgsFunction = projectTaskArgCompletion
	taskEditCmd.ValidArgsFunction = projectTaskArgCompletion
	taskRemoveCmd.ValidArgsFunction = projectTaskArgCompletion
	taskRecurCmd.ValidArgsFunction = projectTaskArgCompletion
	taskUnrecurCmd.ValidArgsFunction = projectTaskArgCompletion
	taskDueCmd.ValidArgsFunction = taskDueCompletion
	taskCompleteCmd.ValidArgsFunction = projectTaskArgCompletion
	taskLinkCmd.ValidArgsFunction = projectTwoTaskArgCompletion
	taskDependCmd.ValidArgsFunction = projectTwoTaskArgCompletion

	// task edit flags
	taskEditCmd.Flags().String("title", "", "New title")
	taskEditCmd.Flags().StringP("description", "d", "", "New description")
	taskEditCmd.Flags().StringP("status", "s", "", "New status")
	taskEditCmd.Flags().StringP("priority", "p", "", "New priority")
	taskEditCmd.Flags().Float64P("estimated", "e", 0, "New estimated hours")
	taskEditCmd.Flags().String("jira-issue", "", "Set Jira issue ID (use empty string to clear)")

	// task remove flags
	taskRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")

	// Add subcommands
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskEditCmd)
	taskCmd.AddCommand(taskRemoveCmd)
	taskCmd.AddCommand(taskLinkCmd)
	taskCmd.AddCommand(taskDependCmd)
	taskCmd.AddCommand(taskRecurCmd)
	taskCmd.AddCommand(taskUnrecurCmd)
	taskCmd.AddCommand(taskDueCmd)
	taskCmd.AddCommand(taskCompleteCmd)
}
