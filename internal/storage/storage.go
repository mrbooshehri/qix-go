package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mrbooshehri/qix-go/internal/config"
	"github.com/mrbooshehri/qix-go/internal/models"
)

// Storage handles all data persistence operations
type Storage struct {
	config *config.Config
	cache  *Cache
}

// Cache stores frequently accessed data in memory
type Cache struct {
	mu       sync.RWMutex
	projects map[string]*models.Project
	index    models.TaskIndex
	dirty    map[string]bool // Tracks which projects need saving
}

var globalStorage *Storage

// Init initializes the global storage instance
func Init() error {
	cfg := config.Get()
	
	globalStorage = &Storage{
		config: cfg,
		cache: &Cache{
			projects: make(map[string]*models.Project),
			index:    make(models.TaskIndex),
			dirty:    make(map[string]bool),
		},
	}
	
	// Load index on startup
	if err := globalStorage.LoadIndex(); err != nil {
		// Index doesn't exist or is corrupted, will rebuild on first access
		globalStorage.cache.index = make(models.TaskIndex)
	}
	
	return nil
}

// Get returns the global storage instance
func Get() *Storage {
	if globalStorage == nil {
		if err := Init(); err != nil {
			panic(err)
		}
	}
	return globalStorage
}

// readJSONFile reads and unmarshals a JSON file
func readJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, v)
}

// writeJSONFile marshals and writes a JSON file atomically
func writeJSONFile(path string, v interface{}) error {
	// Marshal with indentation for readability
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	
	// Write to temp file first
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}
	
	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // Cleanup on failure
		return err
	}
	
	return nil
}

// validateJSON checks if data is valid JSON before writing
func validateJSON(v interface{}) error {
	_, err := json.Marshal(v)
	return err
}

// GetFromCache retrieves a project from cache
func (s *Storage) GetFromCache(projectName string) (*models.Project, bool) {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()
	
	project, exists := s.cache.projects[projectName]
	return project, exists
}

// PutInCache stores a project in cache
func (s *Storage) PutInCache(projectName string, project *models.Project) {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	
	s.cache.projects[projectName] = project
}

// MarkDirty marks a project as needing to be saved
func (s *Storage) MarkDirty(projectName string) {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	
	s.cache.dirty[projectName] = true
}

// IsDirty checks if a project has unsaved changes
func (s *Storage) IsDirty(projectName string) bool {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()
	
	return s.cache.dirty[projectName]
}

// ClearDirty marks a project as saved
func (s *Storage) ClearDirty(projectName string) {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	
	delete(s.cache.dirty, projectName)
}

// InvalidateCache removes a project from cache
func (s *Storage) InvalidateCache(projectName string) {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	
	delete(s.cache.projects, projectName)
	delete(s.cache.dirty, projectName)
}

// FlushAll saves all dirty projects to disk
func (s *Storage) FlushAll() error {
	s.cache.mu.Lock()
	dirtyProjects := make([]string, 0, len(s.cache.dirty))
	for name := range s.cache.dirty {
		dirtyProjects = append(dirtyProjects, name)
	}
	s.cache.mu.Unlock()
	
	for _, name := range dirtyProjects {
		if project, exists := s.GetFromCache(name); exists {
			if err := s.SaveProject(name, project); err != nil {
				return fmt.Errorf("failed to save project %s: %w", name, err)
			}
		}
	}
	
	return nil
}

// ClearCache removes all cached data
func (s *Storage) ClearCache() {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	
	s.cache.projects = make(map[string]*models.Project)
	s.cache.dirty = make(map[string]bool)
}

// GetCacheStats returns statistics about cache usage
func (s *Storage) GetCacheStats() map[string]interface{} {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()
	
	return map[string]interface{}{
		"cached_projects": len(s.cache.projects),
		"dirty_projects":  len(s.cache.dirty),
		"index_entries":   len(s.cache.index),
	}
}

// ListProjects returns all project names
func (s *Storage) ListProjects() ([]string, error) {
	return s.config.ListProjectFiles()
}

// ProjectExists checks if a project exists
func (s *Storage) ProjectExists(projectName string) bool {
	return s.config.ProjectExists(projectName)
}

// DeleteProject removes a project file and clears cache
func (s *Storage) DeleteProject(projectName string) error {
	path := s.config.GetProjectPath(projectName)
	
	// Remove from cache first
	s.InvalidateCache(projectName)
	
	// Delete file
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete project file: %w", err)
	}
	
	// Rebuild index
	return s.RebuildIndex()
}

// CreateBackup creates a timestamped backup of all data
func (s *Storage) CreateBackup() (string, error) {
	timestamp := filepath.Base(s.config.QixDir)
	// You can use time.Now().Format("20060102_150405") for timestamp
	
	// Implementation would use tar or zip
	// For now, return placeholder
	return filepath.Join(s.config.BackupDir, "backup_"+timestamp+".tar.gz"), nil
}

// CleanupOldBackups removes backups older than retention period
func (s *Storage) CleanupOldBackups() error {
	// Implementation would check file modification times
	// and delete files older than config.BackupRetentionDays
	return nil
}