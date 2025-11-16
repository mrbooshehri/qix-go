package storage

import (
	"fmt"
	"time"

	"github.com/mrbooshehri/qix-go/internal/models"
)

// LoadProject loads a project from disk (with caching)
func (s *Storage) LoadProject(projectName string) (*models.Project, error) {
	// Check cache first
	if project, exists := s.GetFromCache(projectName); exists {
		return project, nil
	}
	
	// Load from disk
	path := s.config.GetProjectPath(projectName)
	
	var project models.Project
	if err := readJSONFile(path, &project); err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}
	
	// Cache it
	s.PutInCache(projectName, &project)
	
	return &project, nil
}

// SaveProject saves a project to disk
func (s *Storage) SaveProject(projectName string, project *models.Project) error {
	// Validate JSON before writing
	if err := validateJSON(project); err != nil {
		return fmt.Errorf("invalid project data: %w", err)
	}
	
	path := s.config.GetProjectPath(projectName)
	
	if err := writeJSONFile(path, project); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}
	
	// Update cache
	s.PutInCache(projectName, project)
	s.ClearDirty(projectName)
	
	// Update index
	return s.indexProject(projectName, project)
}

// CreateProject creates a new project
func (s *Storage) CreateProject(name, description string, tags []string) (*models.Project, error) {
	if s.ProjectExists(name) {
		return nil, fmt.Errorf("project '%s' already exists", name)
	}
	
	project := &models.Project{
		Name:        name,
		Description: description,
		Tags:        tags,
		Modules:     make([]models.Module, 0),
		Tasks:       make([]models.Task, 0),
		Sprints:     make([]models.Sprint, 0),
		CreatedAt:   time.Now(),
	}
	
	if err := s.SaveProject(name, project); err != nil {
		return nil, err
	}
	
	return project, nil
}

// UpdateProject updates an existing project
func (s *Storage) UpdateProject(projectName string, updater func(*models.Project) error) error {
	project, err := s.LoadProject(projectName)
	if err != nil {
		return err
	}
	
	if err := updater(project); err != nil {
		return err
	}
	
	return s.SaveProject(projectName, project)
}

// AddModule adds a module to a project
func (s *Storage) AddModule(projectName string, module models.Module) error {
	return s.UpdateProject(projectName, func(p *models.Project) error {
		// Check for duplicate module names
		for _, m := range p.Modules {
			if m.Name == module.Name {
				return fmt.Errorf("module '%s' already exists", module.Name)
			}
		}
		
		module.CreatedAt = time.Now()
		module.Tasks = make([]models.Task, 0)
		p.Modules = append(p.Modules, module)
		return nil
	})
}

// RemoveModule removes a module from a project
func (s *Storage) RemoveModule(projectName, moduleName string) error {
	return s.UpdateProject(projectName, func(p *models.Project) error {
		for i, m := range p.Modules {
			if m.Name == moduleName {
				// Remove module
				p.Modules = append(p.Modules[:i], p.Modules[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("module '%s' not found", moduleName)
	})
}

// GetModule retrieves a specific module
func (s *Storage) GetModule(projectName, moduleName string) (*models.Module, error) {
	project, err := s.LoadProject(projectName)
	if err != nil {
		return nil, err
	}
	
	for _, m := range project.Modules {
		if m.Name == moduleName {
			return &m, nil
		}
	}
	
	return nil, fmt.Errorf("module '%s' not found", moduleName)
}

// UpdateModule updates a specific module
func (s *Storage) UpdateModule(projectName, moduleName string, updater func(*models.Module) error) error {
	return s.UpdateProject(projectName, func(p *models.Project) error {
		for i := range p.Modules {
			if p.Modules[i].Name == moduleName {
				return updater(&p.Modules[i])
			}
		}
		return fmt.Errorf("module '%s' not found", moduleName)
	})
}

// AddSprint adds a sprint to a project
func (s *Storage) AddSprint(projectName string, sprint models.Sprint) error {
	return s.UpdateProject(projectName, func(p *models.Project) error {
		// Check for duplicate sprint names
		for _, sp := range p.Sprints {
			if sp.Name == sprint.Name {
				return fmt.Errorf("sprint '%s' already exists", sprint.Name)
			}
		}
		
		sprint.CreatedAt = time.Now()
		sprint.TaskIDs = make([]string, 0)
		p.Sprints = append(p.Sprints, sprint)
		return nil
	})
}

// GetSprint retrieves a specific sprint
func (s *Storage) GetSprint(projectName, sprintName string) (*models.Sprint, error) {
	project, err := s.LoadProject(projectName)
	if err != nil {
		return nil, err
	}
	
	for _, sp := range project.Sprints {
		if sp.Name == sprintName {
			return &sp, nil
		}
	}
	
	return nil, fmt.Errorf("sprint '%s' not found", sprintName)
}

// AssignTaskToSprint assigns a task ID to a sprint
func (s *Storage) AssignTaskToSprint(projectName, sprintName, taskID string) error {
	return s.UpdateProject(projectName, func(p *models.Project) error {
		for i := range p.Sprints {
			if p.Sprints[i].Name == sprintName {
				// Check if already assigned
				for _, id := range p.Sprints[i].TaskIDs {
					if id == taskID {
						return nil // Already assigned
					}
				}
				p.Sprints[i].TaskIDs = append(p.Sprints[i].TaskIDs, taskID)
				return nil
			}
		}
		return fmt.Errorf("sprint '%s' not found", sprintName)
	})
}

// GetAllProjects loads all projects (useful for reports)
func (s *Storage) GetAllProjects() ([]*models.Project, error) {
	names, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	
	projects := make([]*models.Project, 0, len(names))
	for _, name := range names {
		project, err := s.LoadProject(name)
		if err != nil {
			// Skip corrupted projects but log
			continue
		}
		projects = append(projects, project)
	}
	
	return projects, nil
}

// GetProjectStats returns statistics for a project
func (s *Storage) GetProjectStats(projectName string) (map[string]interface{}, error) {
	project, err := s.LoadProject(projectName)
	if err != nil {
		return nil, err
	}
	
	counts := project.CountByStatus()
	allTasks := project.GetAllTasks()
	
	stats := map[string]interface{}{
		"total_tasks":       len(allTasks),
		"todo":              counts[models.StatusTodo],
		"doing":             counts[models.StatusDoing],
		"done":              counts[models.StatusDone],
		"blocked":           counts[models.StatusBlocked],
		"total_estimated":   project.CalculateTotalEstimated(),
		"total_actual":      project.CalculateTotalActual(),
		"completion_pct":    project.GetCompletionPercentage(),
		"module_count":      len(project.Modules),
		"sprint_count":      len(project.Sprints),
	}
	
	return stats, nil
}