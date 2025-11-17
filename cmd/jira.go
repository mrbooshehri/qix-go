package cmd

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mrbooshehri/qix-go/internal/config"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
)

var jiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Jira integration helpers",
}

var jiraOpenCmd = &cobra.Command{
	Use:   "open <project> <task_id>",
	Short: "Open the Jira issue linked to a task",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		taskID := args[1]

		store := storage.Get()
		task, _, err := store.FindTask(projectName, taskID)
		if err != nil {
			ui.PrintError("Task not found: %v", err)
			return
		}

		issueID := strings.TrimSpace(task.JiraIssue)
		if issueID == "" {
			ui.PrintError("Task [%s] has no Jira issue linked. Use 'qix task edit %s %s --jira-issue <ID>' to set one.", taskID, projectName, taskID)
			return
		}

		cfg := config.Get()
		baseURL := strings.TrimSpace(cfg.JiraBaseURL)
		if baseURL == "" {
			ui.PrintError("Jira base URL not configured. Set 'jira_base_url' in %s or export JIRA_BASE_URL.", cfg.ConfigFile)
			return
		}

		issueURL := strings.TrimRight(baseURL, "/") + "/" + issueID
		if err := openInBrowser(issueURL); err != nil {
			ui.PrintError("Failed to open Jira issue: %v", err)
			ui.Dim.Printf("URL: %s\n", issueURL)
			return
		}

		ui.PrintSuccess("Opening Jira issue: %s", issueURL)
	},
}

func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func init() {
	jiraOpenCmd.ValidArgsFunction = jiraOpenCompletion
	jiraCmd.AddCommand(jiraOpenCmd)
}

func jiraOpenCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeProjectNames(toComplete)
	case 1:
		return completeTaskIDs(args[0], toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
