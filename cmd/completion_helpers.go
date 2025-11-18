package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/mrbooshehri/qix-go/internal/config"
	"github.com/mrbooshehri/qix-go/internal/logging"
	"github.com/mrbooshehri/qix-go/internal/storage"
)

var (
	completionInitOnce sync.Once
	completionInitErr  error
)

func ensureCompletionReady() error {
	completionInitOnce.Do(func() {
		if err := config.Init(); err != nil {
			completionInitErr = err
			return
		}
		cfg := config.Get()
		if err := logging.Init(cfg.LogFile); err != nil {
			completionInitErr = err
			return
		}
		logging.SetLevel(cfg.LogLevel)
		logging.Debugf("Completion config initialized (projects: %s)", cfg.ProjectsDir)
		completionInitErr = storage.Init()
	})
	return completionInitErr
}

func completeProjectNames(toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := ensureCompletionReady(); err != nil {
		logging.Errorf("Project completion init failed: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	store := storage.Get()
	names, err := store.ListProjects()
	if err != nil {
		logging.Errorf("Failed to list projects for completion: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	matches := make([]string, 0, len(names))
	for _, name := range names {
		if toComplete == "" || strings.HasPrefix(name, toComplete) {
			matches = append(matches, name)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

func completeTaskIDs(projectName, toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := ensureCompletionReady(); err != nil {
		logging.Errorf("Task completion init failed: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	store := storage.Get()
	project, err := store.LoadProject(projectName)
	if err != nil {
		logging.Warnf("Project '%s' not found during completion: %v", projectName, err)
		return nil, cobra.ShellCompDirectiveError
	}

	matches := make([]string, 0, len(project.GetAllTasks()))
	filter := strings.ToLower(toComplete)

	for _, task := range project.GetAllTasks() {
		idMatch := toComplete == "" || strings.HasPrefix(task.ID, toComplete)
		nameMatch := filter != "" && strings.Contains(strings.ToLower(task.Title), filter)

		if idMatch || nameMatch {
			matches = append(matches, fmt.Sprintf("%s\t%s", task.ID, task.Title))
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

func completeProjectModulePaths(toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := ensureCompletionReady(); err != nil {
		logging.Errorf("Project/module completion init failed: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	store := storage.Get()
	names, err := store.ListProjects()
	if err != nil {
		logging.Errorf("Failed to list projects for path completion: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	lowerPrefix := strings.ToLower(toComplete)
	matches := make([]string, 0, len(names))

	for _, name := range names {
		if lowerPrefix == "" || strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			matches = append(matches, escapeCompletion(name))
		}

		project, err := store.LoadProject(name)
		if err != nil {
			logging.Warnf("Unable to load project '%s' for completion: %v", name, err)
			continue
		}

		for _, module := range project.Modules {
			path := fmt.Sprintf("%s/%s", name, module.Name)
			if lowerPrefix == "" || strings.HasPrefix(strings.ToLower(path), lowerPrefix) {
				matches = append(matches, escapeCompletion(path))
			}
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

func projectArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeProjectNames(toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func projectTaskArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeProjectNames(toComplete)
	case 1:
		return completeTaskIDs(args[0], toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func projectTwoTaskArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeProjectNames(toComplete)
	case 1, 2:
		return completeTaskIDs(args[0], toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func moduleCreateArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names, dir := completeProjectNames(toComplete)
	if dir == cobra.ShellCompDirectiveError {
		return names, dir
	}

	matches := make([]string, len(names))
	for i, name := range names {
		matches[i] = fmt.Sprintf("%s/", name)
	}

	return matches, dir | cobra.ShellCompDirectiveNoSpace
}

func modulePathArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeModulePaths(toComplete)
}

func completeModulePaths(toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := ensureCompletionReady(); err != nil {
		logging.Errorf("Module completion init failed: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	store := storage.Get()
	names, err := store.ListProjects()
	if err != nil {
		logging.Errorf("Failed to list projects for module completion: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	lowerPrefix := strings.ToLower(toComplete)
	matches := make([]string, 0)

	for _, name := range names {
		project, err := store.LoadProject(name)
		if err != nil {
			logging.Warnf("Unable to load project '%s' for module completion: %v", name, err)
			continue
		}

		for _, module := range project.Modules {
			path := fmt.Sprintf("%s/%s", name, module.Name)
			if lowerPrefix == "" || strings.HasPrefix(strings.ToLower(path), lowerPrefix) {
				matches = append(matches, escapeCompletion(path))
			}
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

func projectFromPath(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.SplitN(path, "/", 2)
	return parts[0]
}

func trackPathTaskArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeProjectModulePaths(toComplete)
	case 1:
		projectName := projectFromPath(args[0])
		if projectName == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeTaskIDs(projectName, toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func twoProjectArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0, 1:
		return completeProjectNames(toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func escapeCompletion(value string) string {
	if value == "" {
		return value
	}
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, " ", `\ `)
	return value
}

func completeSprintNames(projectName, toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := ensureCompletionReady(); err != nil {
		logging.Errorf("Sprint completion init failed: %v", err)
		return nil, cobra.ShellCompDirectiveError
	}

	store := storage.Get()
	project, err := store.LoadProject(projectName)
	if err != nil {
		logging.Warnf("Project '%s' not found during sprint completion: %v", projectName, err)
		return nil, cobra.ShellCompDirectiveError
	}

	filter := strings.ToLower(toComplete)
	matches := make([]string, 0, len(project.Sprints))
	for _, sprint := range project.Sprints {
		if filter == "" || strings.HasPrefix(strings.ToLower(sprint.Name), filter) {
			matches = append(matches, sprint.Name)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

func sprintProjectSprintArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeProjectNames(toComplete)
	case 1:
		return completeSprintNames(args[0], toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func sprintProjectSprintTaskArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeProjectNames(toComplete)
	case 1:
		return completeSprintNames(args[0], toComplete)
	case 2:
		return completeTaskIDs(args[0], toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
