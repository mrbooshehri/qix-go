package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  "Create, list, view, and manage projects",
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
		if description != "" {
			ui.Dim.Printf("  Description: %s\n", description)
		}
		if len(tags) > 0 {
			ui.Dim.Printf("  Tags: %s\n", strings.Join(tags, ", "))
		}
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Run: func(cmd *cobra.Command, args []string) {
		store := storage.Get()
		
		projects, err := store.ListProjects()
		if err != nil {
			ui.PrintError("Failed to list projects: %v", err)
			return
		}
		
		if len(projects) == 0 {
			ui.PrintEmptyState(
				"No projects found",
				"Create one with: qix project create <name>",
			)
			return
		}
		
		ui.PrintHeader("📂 Projects")
		
		for _, name := range projects {
			project, err := store.LoadProject(name)
			if err != nil {
				ui.PrintWarning("Could not load project: %s", name)
				continue
			}
			
			ui.BoldMagenta.Printf("\n📁 %s\n", project.Name)
			
			if project.Description != "" {
				ui.Blue.Printf("   %s\n", project.Description)
			}
			
			taskCount := len(project.GetAllTasks())
			moduleCount := len(project.Modules)
			
			ui.Yellow.Printf("   Tasks: %d | Modules: %d\n", taskCount, moduleCount)
			
			// Quick stats
			counts := project.CountByStatus()
			if taskCount > 0 {
				completion := project.GetCompletionPercentage()
				ui.Cyan.Printf("   Completion: ")
				ui.PrintProgressBar(completion, 30)
				fmt.Printf(" %.1f%% (%d done)\n", completion, counts["done"])
			}
		}
		fmt.Println()
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show <project>",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		store := storage.Get()
		
		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}
		
		ui.PrintProjectSummary(project)
		
		// Show modules
		if len(project.Modules) > 0 {
			ui.PrintSubHeader("📦 Modules")
			
			for _, module := range project.Modules {
				ui.PrintModuleSummary(&module)
			}
			fmt.Println()
		}
		
		// Show project-level tasks
		if len(project.Tasks) > 0 {
			ui.PrintSubHeader("📋 Project-Level Tasks")
			
			for _, task := range project.Tasks {
				ui.PrintTask(task, "  ")
			}
			fmt.Println()
		}
		
		// Show sprints
		if len(project.Sprints) > 0 {
			ui.PrintSubHeader("🏃 Sprints")
			
			for _, sprint := range project.Sprints {
				ui.Cyan.Printf("  • %s", sprint.Name)
				ui.Dim.Printf(" (%s → %s)", 
					ui.FormatDate(sprint.StartDate), 
					ui.FormatDate(sprint.EndDate))
				fmt.Printf(" - %d tasks\n", len(sprint.TaskIDs))
			}
			fmt.Println()
		}
	},
}

var projectStatsCmd = &cobra.Command{
	Use:   "stats <project>",
	Short: "Show project statistics",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		store := storage.Get()
		
		stats, err := store.GetProjectStats(projectName)
		if err != nil {
			ui.PrintError("Failed to get stats: %v", err)
			return
		}
		
		ui.PrintHeader(fmt.Sprintf("📊 Statistics: %s", projectName))
		
		// Task counts
		table := ui.NewTableBuilder("Metric", "Value").
			Row("Total Tasks", fmt.Sprintf("%v", stats["total_tasks"])).
			Row("", "").
			Row("⭕ Todo", fmt.Sprintf("%v", stats["todo"])).
			Row("🔄 Doing", fmt.Sprintf("%v", stats["doing"])).
			Row("✅ Done", fmt.Sprintf("%v", stats["done"])).
			Row("🚫 Blocked", fmt.Sprintf("%v", stats["blocked"])).
			Row("", "").
			Row("Modules", fmt.Sprintf("%v", stats["module_count"])).
			Row("Sprints", fmt.Sprintf("%v", stats["sprint_count"])).
			Align(1, ui.AlignRight)
		
		table.PrintSimple()
		fmt.Println()
		
		// Time statistics
		ui.PrintSubHeader("⏱️  Time Tracking")
		
		totalEst := stats["total_estimated"].(float64)
		totalAct := stats["total_actual"].(float64)
		
		fmt.Printf("Estimated: %s\n", ui.FormatHours(totalEst))
		fmt.Printf("Actual:    %s\n", ui.FormatHours(totalAct))
		
		if totalEst > 0 {
			variance := totalAct - totalEst
			varPct := (variance / totalEst) * 100
			
			fmt.Print("Variance:  ")
			if variance > 0 {
				ui.Red.Printf("+%s (%.1f%% over)\n", ui.FormatHours(variance), varPct)
			} else {
				ui.Green.Printf("%s (%.1f%% under)\n", ui.FormatHours(variance), -varPct)
			}
		}
		
		fmt.Println()
		
		// Completion
		completion := stats["completion_pct"].(float64)
		fmt.Print("Completion: ")
		ui.PrintProgressBar(completion, 50)
		fmt.Printf(" %.1f%%\n", completion)
	},
}

var projectRemoveCmd = &cobra.Command{
	Use:   "remove <project>",
	Short: "Remove a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		store := storage.Get()
		
		if !store.ProjectExists(projectName) {
			ui.PrintError("Project not found: %s", projectName)
			return
		}
		
		// Confirmation
		force, _ := cmd.Flags().GetBool("force")
		
		if !force {
			fmt.Printf("⚠️  This will permanently delete project '%s' and all its data.\n", projectName)
			fmt.Print("Type the project name to confirm: ")
			
			var confirm string
			fmt.Scanln(&confirm)
			
			if confirm != projectName {
				ui.PrintInfo("Deletion cancelled")
				return
			}
		}
		
		if err := store.DeleteProject(projectName); err != nil {
			ui.PrintError("Failed to delete project: %v", err)
			return
		}
		
		ui.PrintSuccess("Project '%s' deleted", projectName)
	},
}

var projectEditCmd = &cobra.Command{
	Use:   "edit <project>",
	Short: "Edit project details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		store := storage.Get()
		
		newName, _ := cmd.Flags().GetString("name")
		newDesc, _ := cmd.Flags().GetString("description")
		
		if newName == "" && newDesc == "" {
			ui.PrintError("Specify at least --name or --description")
			return
		}
		
		err := store.UpdateProject(projectName, func(p *models.Project) error {
			if newName != "" {
				p.Name = newName
			}
			if newDesc != "" {
				p.Description = newDesc
			}
			return nil
		})
		
		if err != nil {
			ui.PrintError("Failed to update project: %v", err)
			return
		}
		
		// Handle rename
		if newName != "" && newName != projectName {
			// Delete old file and save with new name
			oldPath := store.GetProjectPath(projectName)
			newPath := store.GetProjectPath(newName)
			
			if err := os.Rename(oldPath, newPath); err != nil {
				ui.PrintError("Failed to rename project file: %v", err)
				return
			}
			
			store.InvalidateCache(projectName)
			
			ui.PrintSuccess("Project renamed: %s → %s", projectName, newName)
		} else {
			ui.PrintSuccess("Project '%s' updated", projectName)
		}
		
		if newDesc != "" {
			ui.Dim.Printf("  Description: %s\n", newDesc)
		}
	},
}

func init() {
	// project create flags
	projectCreateCmd.Flags().StringSliceP("tags", "t", []string{}, "Tags for the project")
	
	// project remove flags
	projectRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	
	// project edit flags
	projectEditCmd.Flags().StringP("name", "n", "", "New project name")
	projectEditCmd.Flags().StringP("description", "d", "", "New project description")
	
	// Add subcommands
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectStatsCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectEditCmd)
}