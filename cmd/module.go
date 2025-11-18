package cmd

import (
	"fmt"
	"strings"

	"github.com/mrbooshehri/qix-go/internal/models"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
	"github.com/spf13/cobra"
)

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Manage project modules",
	Long:  "Create, list, and manage modules within projects",
}

var moduleCreateCmd = &cobra.Command{
	Use:   "create <project/module> [description]",
	Short: "Create a new module",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		description := ""
		if len(args) > 1 {
			description = strings.Join(args[1:], " ")
		}

		// Parse path
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			ui.PrintError("Invalid path format. Use: <project>/<module>")
			return
		}

		projectName := parts[0]
		moduleName := parts[1]

		tags, _ := cmd.Flags().GetStringSlice("tags")

		store := storage.Get()

		module := models.Module{
			Name:        moduleName,
			Description: description,
			Tags:        tags,
		}

		if err := store.AddModule(projectName, module); err != nil {
			ui.PrintError("Failed to create module: %v", err)
			return
		}

		ui.PrintSuccess("Module '%s' created in project '%s'", moduleName, projectName)
		if description != "" {
			ui.Dim.Printf("  Description: %s\n", description)
		}
	},
}

var moduleListCmd = &cobra.Command{
	Use:   "list <project>",
	Short: "List modules in a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		store := storage.Get()

		project, err := store.LoadProject(projectName)
		if err != nil {
			ui.PrintError("Project not found: %s", projectName)
			return
		}

		if len(project.Modules) == 0 {
			ui.PrintEmptyState(
				fmt.Sprintf("No modules in project '%s'", projectName),
				fmt.Sprintf("Create one with: qix module create %s/<module_name>", projectName),
			)
			return
		}

		ui.PrintHeader(fmt.Sprintf("📦 Modules in '%s'", projectName))

		for _, module := range project.Modules {
			ui.BoldCyan.Printf("\n• %s\n", module.Name)

			if module.Description != "" {
				ui.Blue.Printf("  %s\n", module.Description)
			}

			taskCount := len(module.Tasks)
			ui.Yellow.Printf("  Tasks: %d\n", taskCount)

			if taskCount > 0 {
				// Calculate completion
				done := 0
				for _, task := range module.Tasks {
					if task.Status == models.StatusDone {
						done++
					}
				}

				completion := float64(done) / float64(taskCount) * 100
				ui.Cyan.Printf("  Progress: ")
				ui.PrintProgressBar(completion, 30)
				fmt.Printf(" %.1f%%\n", completion)
			}

			if len(module.Tags) > 0 {
				ui.Dim.Printf("  Tags: %s\n", strings.Join(module.Tags, ", "))
			}
		}
		fmt.Println()
	},
}

var moduleShowCmd = &cobra.Command{
	Use:   "show <project/module>",
	Short: "Show module details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		// Parse path
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			ui.PrintError("Invalid path format. Use: <project>/<module>")
			return
		}

		projectName := parts[0]
		moduleName := parts[1]

		store := storage.Get()

		module, err := store.GetModule(projectName, moduleName)
		if err != nil {
			ui.PrintError("Module not found: %v", err)
			return
		}

		ui.PrintHeader(fmt.Sprintf("📦 %s", module.Name))

		if module.Description != "" {
			ui.Blue.Println(module.Description)
			fmt.Println()
		}

		// Statistics
		taskCount := len(module.Tasks)
		done := 0
		totalEst := 0.0
		totalAct := 0.0

		for _, task := range module.Tasks {
			if task.Status == models.StatusDone {
				done++
			}
			totalEst += task.EstimatedHours
			totalAct += task.CalculateActualHours()
		}

		table := ui.NewTableBuilder("Metric", "Value").
			Row("Total Tasks", fmt.Sprintf("%d", taskCount)).
			Row("Completed", fmt.Sprintf("%d", done))

		if taskCount > 0 {
			completion := float64(done) / float64(taskCount) * 100
			table.Row("Completion", fmt.Sprintf("%.1f%%", completion))
		}

		if totalEst > 0 {
			table.Row("", "").
				Row("Estimated", ui.FormatHours(totalEst)).
				Row("Actual", ui.FormatHours(totalAct))
		}

		table.Align(1, ui.AlignRight).PrintSimple()
		fmt.Println()

		// Show tasks
		if len(module.Tasks) > 0 {
			ui.PrintSubHeader("📋 Tasks")

			for _, task := range module.Tasks {
				ui.PrintTask(task, "  ")
			}
			fmt.Println()
		}

		// Tags
		if len(module.Tags) > 0 {
			ui.PrintSubHeader("🏷️  Tags")
			ui.PrintList(module.Tags, "•")
		}
	},
}

var moduleRemoveCmd = &cobra.Command{
	Use:   "remove <project/module>",
	Short: "Remove a module",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		// Parse path
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			ui.PrintError("Invalid path format. Use: <project>/<module>")
			return
		}

		projectName := parts[0]
		moduleName := parts[1]

		store := storage.Get()

		// Check if module exists
		module, err := store.GetModule(projectName, moduleName)
		if err != nil {
			ui.PrintError("Module not found: %v", err)
			return
		}

		// Confirmation
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("⚠️  This will delete module '%s' and its %d task(s).\n",
				moduleName, len(module.Tasks))
			fmt.Print("Type the module name to confirm: ")

			var confirm string
			fmt.Scanln(&confirm)

			if confirm != moduleName {
				ui.PrintInfo("Deletion cancelled")
				return
			}
		}

		if err := store.RemoveModule(projectName, moduleName); err != nil {
			ui.PrintError("Failed to remove module: %v", err)
			return
		}

		ui.PrintSuccess("Module '%s' removed from project '%s'", moduleName, projectName)
	},
}

var moduleEditCmd = &cobra.Command{
	Use:   "edit <project/module>",
	Short: "Edit module details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		// Parse path
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			ui.PrintError("Invalid path format. Use: <project>/<module>")
			return
		}

		projectName := parts[0]
		moduleName := parts[1]

		newName, _ := cmd.Flags().GetString("name")
		newDesc, _ := cmd.Flags().GetString("description")

		if newName == "" && newDesc == "" {
			ui.PrintError("Specify at least --name or --description")
			return
		}

		store := storage.Get()

		err := store.UpdateModule(projectName, moduleName, func(m *models.Module) error {
			if newName != "" {
				m.Name = newName
			}
			if newDesc != "" {
				m.Description = newDesc
			}
			return nil
		})

		if err != nil {
			ui.PrintError("Failed to update module: %v", err)
			return
		}

		if newName != "" && newName != moduleName {
			ui.PrintSuccess("Module renamed: %s → %s", moduleName, newName)
		} else {
			ui.PrintSuccess("Module '%s' updated", moduleName)
		}

		if newDesc != "" {
			ui.Dim.Printf("  Description: %s\n", newDesc)
		}
	},
}

func init() {
	// module create flags
	moduleCreateCmd.Flags().StringSliceP("tags", "t", []string{}, "Tags for the module")
	moduleCreateCmd.ValidArgsFunction = moduleCreateArgCompletion

	// module remove flags
	moduleRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	moduleRemoveCmd.ValidArgsFunction = modulePathArgCompletion

	// module edit flags
	moduleEditCmd.Flags().StringP("name", "n", "", "New module name")
	moduleEditCmd.Flags().StringP("description", "d", "", "New module description")
	moduleEditCmd.ValidArgsFunction = modulePathArgCompletion

	moduleListCmd.ValidArgsFunction = projectArgCompletion
	moduleShowCmd.ValidArgsFunction = modulePathArgCompletion

	// Add subcommands
	moduleCmd.AddCommand(moduleCreateCmd)
	moduleCmd.AddCommand(moduleListCmd)
	moduleCmd.AddCommand(moduleShowCmd)
	moduleCmd.AddCommand(moduleRemoveCmd)
	moduleCmd.AddCommand(moduleEditCmd)
}
