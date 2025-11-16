package storage

import (
	"fmt"
	"os"
	"time"

	"github.com/mrbooshehri/qix-go/internal/models"
)

// LoadTrackingData loads the tracking session data
func (s *Storage) LoadTrackingData() (*models.TrackingData, error) {
	// Check if file exists
	if _, err := os.Stat(s.config.TrackFile); os.IsNotExist(err) {
		// Create empty tracking data
		return &models.TrackingData{
			ActiveSession: nil,
			Sessions:      make([]interface{}, 0),
		}, nil
	}
	
	var data models.TrackingData
	if err := readJSONFile(s.config.TrackFile, &data); err != nil {
		return nil, fmt.Errorf("failed to load tracking data: %w", err)
	}
	
	return &data, nil
}

// SaveTrackingData saves the tracking session data
func (s *Storage) SaveTrackingData(data *models.TrackingData) error {
	return writeJSONFile(s.config.TrackFile, data)
}

// StartTracking starts a new tracking session
func (s *Storage) StartTracking(projectName, moduleName, taskID string) error {
	// Verify task exists
	if _, _, err := s.FindTask(projectName, taskID); err != nil {
		return fmt.Errorf("task not found: %w", err)
	}
	
	data, err := s.LoadTrackingData()
	if err != nil {
		return err
	}
	
	// Check for existing session
	if data.ActiveSession != nil {
		return fmt.Errorf("active session already exists for task %s", data.ActiveSession.TaskID)
	}
	
	// Create path
	path := projectName
	if moduleName != "" {
		path = fmt.Sprintf("%s/%s", projectName, moduleName)
	}
	
	// Create new session
	data.ActiveSession = &models.TrackingSession{
		Path:      path,
		TaskID:    taskID,
		StartTime: time.Now(),
	}
	
	return s.SaveTrackingData(data)
}

// StopTracking stops the current tracking session and logs time
func (s *Storage) StopTracking() (time.Duration, string, string, error) {
	data, err := s.LoadTrackingData()
	if err != nil {
		return 0, "", "", err
	}
	
	if data.ActiveSession == nil {
		return 0, "", "", fmt.Errorf("no active tracking session")
	}
	
	session := data.ActiveSession
	elapsed := time.Since(session.StartTime)
	hours := elapsed.Hours()
	
	// Parse path to get project and module
	projectName := session.Path
	moduleName := ""
	
	// Check if path contains module
	// This is a simple split, you might want to use the parse_path logic
	// For now, assume format is "project" or "project/module"
	
	// Add time entry to task
	entry := models.TimeEntry{
		Date:     time.Now().Format("2006-01-02"),
		Hours:    hours,
		LoggedAt: time.Now(),
	}
	
	if err := s.AddTimeEntry(projectName, session.TaskID, entry); err != nil {
		return 0, "", "", fmt.Errorf("failed to log time: %w", err)
	}
	
	// Clear active session
	taskID := session.TaskID
	path := session.Path
	data.ActiveSession = nil
	
	if err := s.SaveTrackingData(data); err != nil {
		return 0, "", "", err
	}
	
	return elapsed, path, taskID, nil
}

// GetActiveSession returns the current active session if any
func (s *Storage) GetActiveSession() (*models.TrackingSession, error) {
	data, err := s.LoadTrackingData()
	if err != nil {
		return nil, err
	}
	
	return data.ActiveSession, nil
}

// IsTracking checks if there's an active tracking session
func (s *Storage) IsTracking() (bool, error) {
	session, err := s.GetActiveSession()
	if err != nil {
		return false, err
	}
	
	return session != nil, nil
}

// GetElapsedTime returns the elapsed time for the active session
func (s *Storage) GetElapsedTime() (time.Duration, error) {
	session, err := s.GetActiveSession()
	if err != nil {
		return 0, err
	}
	
	if session == nil {
		return 0, fmt.Errorf("no active session")
	}
	
	return time.Since(session.StartTime), nil
}

// SwitchTracking stops current session and starts a new one
func (s *Storage) SwitchTracking(projectName, moduleName, taskID string) error {
	// Stop current session if exists
	tracking, err := s.IsTracking()
	if err != nil {
		return err
	}
	
	if tracking {
		if _, _, _, err := s.StopTracking(); err != nil {
			return fmt.Errorf("failed to stop current session: %w", err)
		}
	}
	
	// Start new session
	return s.StartTracking(projectName, moduleName, taskID)
}

// GetTimeEntriesForDate returns all time entries for a specific date
func (s *Storage) GetTimeEntriesForDate(date string) (map[string][]models.TimeEntry, error) {
	projects, err := s.GetAllProjects()
	if err != nil {
		return nil, err
	}
	
	entriesByProject := make(map[string][]models.TimeEntry)
	
	for _, project := range projects {
		entries := make([]models.TimeEntry, 0)
		
		for _, task := range project.GetAllTasks() {
			for _, entry := range task.TimeEntries {
				if entry.Date == date {
					entries = append(entries, entry)
				}
			}
		}
		
		if len(entries) > 0 {
			entriesByProject[project.Name] = entries
		}
	}
	
	return entriesByProject, nil
}

// GetTimeEntriesInRange returns all time entries within a date range
func (s *Storage) GetTimeEntriesInRange(projectName, startDate, endDate string) ([]models.TimeEntry, error) {
	project, err := s.LoadProject(projectName)
	if err != nil {
		return nil, err
	}
	
	entries := make([]models.TimeEntry, 0)
	
	for _, task := range project.GetAllTasks() {
		for _, entry := range task.TimeEntries {
			if entry.Date >= startDate && entry.Date <= endDate {
				entries = append(entries, entry)
			}
		}
	}
	
	return entries, nil
}

// CalculateTotalHoursForDate returns total hours logged on a specific date
func (s *Storage) CalculateTotalHoursForDate(date string) (float64, error) {
	entriesByProject, err := s.GetTimeEntriesForDate(date)
	if err != nil {
		return 0, err
	}
	
	total := 0.0
	for _, entries := range entriesByProject {
		for _, entry := range entries {
			total += entry.Hours
		}
	}
	
	return total, nil
}