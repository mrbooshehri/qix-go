package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mrbooshehri/qix-go/internal/models"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  "Create, inspect, and remove projects",
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name> [description]",
	Short: "Create a new project",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		description := ""
		if len(args) > 1 {
			description = strings.Join(args[1:], " ")
		}

		tags, _ := cmd.Flags().GetStringSlice("tags")

		store := storage.Get()
		project, err := store.CreateProject(name, description, tags)
		if err != nil {
			ui.PrintError("Failed to create project: %v", err)
			return
		}

		ui.PrintSuccess("Project '%s' created", project.Name)
		if project.Description != "" {
			ui.Dim.Printf("  Description: %s\n", project.Description)
		}
		if len(project.Tags) > 0 {
			ui.Dim.Printf("  Tags: %s\n", strings.Join(project.Tags, ", "))
		}
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing projects",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		store := storage.Get()
		names, err := store.ListProjects()
		if err != nil {
			ui.PrintError("Failed to list projects: %v", err)
			return
		}

		if len(names) == 0 {
			ui.PrintEmptyState(
				"No projects found",
				"Create one with: qix project create <name>",
			)
			return
		}

		sort.Strings(names)
		ui.PrintHeader("📁 Projects")

		for _, name := range names {
			project, err := store.LoadProject(name)
			if err != nil {
				ui.PrintError("Failed to load project %s: %v", name, err)
				continue
			}

			printProjectSummary(project)
			fmt.Println()
		}
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		store := storage.Get()
		project, err := store.LoadProject(name)
		if err != nil {
			ui.PrintError("Project not found: %v", err)
			return
		}

		ui.PrintHeader(fmt.Sprintf("📁 %s", project.Name))
		if project.Description != "" {
			ui.Blue.Println(project.Description)
			fmt.Println()
		}

		if len(project.Tags) > 0 {
			ui.PrintSubHeader("🏷️  Tags")
			ui.PrintList(project.Tags, "•")
			fmt.Println()
		}

		printProjectStats(project)
		fmt.Println()

		// Show modules
		if len(project.Modules) > 0 {
			ui.PrintSubHeader("🧩 Modules")
			for _, module := range project.Modules {
				done := 0
				for _, task := range module.Tasks {
					if task.Status == models.StatusDone {
						done++
					}
				}

				completion := 0.0
				if len(module.Tasks) > 0 {
					completion = (float64(done) / float64(len(module.Tasks))) * 100
				}

				ui.BoldCyan.Printf("\n• %s\n", module.Name)
				if module.Description != "" {
					ui.Blue.Printf("  %s\n", module.Description)
				}
				ui.Dim.Printf("  Tasks: %d\n", len(module.Tasks))
				ui.Cyan.Printf("  Progress: ")
				ui.PrintProgressBar(completion, 25)
				fmt.Printf(" %.1f%%\n", completion)
			}
			fmt.Println()
		}

		// Show top-level tasks
		if len(project.Tasks) > 0 {
			ui.PrintSubHeader("🗒️  Project Tasks")
			for _, task := range project.Tasks {
				ui.PrintTask(task, "  ")
			}
			fmt.Println()
		}

		if len(project.Sprints) > 0 {
			ui.PrintSubHeader("🏃  Sprints")
			for _, sprint := range project.Sprints {
				ui.BoldGreen.Printf("\n• %s\n", sprint.Name)
				ui.Dim.Printf("  Tasks: %d\n", len(sprint.TaskIDs))
				ui.Dim.Printf("  Duration: %s → %s\n", sprint.StartDate, sprint.EndDate)
			}
			fmt.Println()
		}
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		store := storage.Get()
		project, err := store.LoadProject(name)
		if err != nil {
			ui.PrintError("Project not found: %v", err)
			return
		}

		if !force {
			fmt.Printf("⚠️  This will delete project '%s' and all its data.\n", name)
			fmt.Print("Type the project name to confirm: ")

			var confirm string
			fmt.Scanln(&confirm)
			if confirm != name {
				ui.PrintInfo("Deletion cancelled")
				return
			}
		}

		if err := store.DeleteProject(name); err != nil {
			ui.PrintError("Failed to delete project: %v", err)
			return
		}

		ui.PrintSuccess("Project '%s' deleted", project.Name)
	},
}

var projectStatsCmd = &cobra.Command{
	Use:   "stats <name>",
	Short: "Show project KPIs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		store := storage.Get()
		project, err := store.LoadProject(name)
		if err != nil {
			ui.PrintError("Project not found: %v", err)
			return
		}

		ui.PrintHeader(fmt.Sprintf("📊 Project KPIs • %s", project.Name))
		printProjectStats(project)
		fmt.Println()

		data := map[string]float64{
			"Todo":    float64(project.CountByStatus()[models.StatusTodo]),
			"Doing":   float64(project.CountByStatus()[models.StatusDoing]),
			"Blocked": float64(project.CountByStatus()[models.StatusBlocked]),
			"Done":    float64(project.CountByStatus()[models.StatusDone]),
		}
		ui.PrintChart(data, 30, true)
	},
}

func printProjectSummary(project *models.Project) {
	counts := project.CountByStatus()
	totalTasks := len(project.GetAllTasks())

	ui.BoldCyan.Printf("• %s\n", project.Name)
	if project.Description != "" {
		ui.Blue.Printf("  %s\n", project.Description)
	}

	ui.Dim.Printf("  Modules: %d | Tasks: %d\n", len(project.Modules), totalTasks)
	ui.Dim.Printf("  Status: %d todo • %d in progress • %d done • %d blocked\n",
		counts[models.StatusTodo],
		counts[models.StatusDoing],
		counts[models.StatusDone],
		counts[models.StatusBlocked],
	)

	ui.Cyan.Printf("  Progress: ")
	ui.PrintProgressBar(project.GetCompletionPercentage(), 25)
	fmt.Printf(" %.1f%%\n", project.GetCompletionPercentage())
}

func printProjectStats(project *models.Project) {
	counts := project.CountByStatus()

	table := ui.NewTableBuilder("Metric", "Value").
		Row("Total Tasks", fmt.Sprintf("%d", len(project.GetAllTasks()))).
		Row("Todo", fmt.Sprintf("%d", counts[models.StatusTodo])).
		Row("Doing", fmt.Sprintf("%d", counts[models.StatusDoing])).
		Row("Done", fmt.Sprintf("%d", counts[models.StatusDone])).
		Row("Blocked", fmt.Sprintf("%d", counts[models.StatusBlocked])).
		Row("Modules", fmt.Sprintf("%d", len(project.Modules))).
		Row("Sprints", fmt.Sprintf("%d", len(project.Sprints))).
		Row("Estimated", ui.FormatHours(project.CalculateTotalEstimated())).
		Row("Actual", ui.FormatHours(project.CalculateTotalActual())).
		Row("Completion", fmt.Sprintf("%.1f%%", project.GetCompletionPercentage()))

	table.Align(1, ui.AlignRight).PrintSimple()
}

func init() {
	projectCreateCmd.Flags().StringSliceP("tags", "t", []string{}, "Tags for the project")
	projectDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectStatsCmd)
}
