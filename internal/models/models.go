package models

import "time"

// Project represents a QIX project
type Project struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Modules     []Module  `json:"modules"`
	Tasks       []Task    `json:"tasks"`
	Sprints     []Sprint  `json:"sprints"`
	CreatedAt   time.Time `json:"created_at"`
}

// Module represents a sub-component of a project
type Module struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Tasks       []Task    `json:"tasks"`
	CreatedAt   time.Time `json:"created_at"`
}

// Task represents a work item
type Task struct {
	ID             string      `json:"id"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Status         TaskStatus  `json:"status"`
	Priority       Priority    `json:"priority"`
	EstimatedHours float64     `json:"estimated_hours"`
	Tags           []string    `json:"tags"`
	Dependencies   []string    `json:"dependencies"`
	JiraIssue      string      `json:"jira_issue,omitempty"`
	ParentID       string      `json:"parent_id,omitempty"`
	TimeEntries    []TimeEntry `json:"time_entries"`
	Recurrence     *Recurrence `json:"recurrence,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// TaskStatus represents the state of a task
type TaskStatus string

const (
	StatusTodo    TaskStatus = "todo"
	StatusDoing   TaskStatus = "doing"
	StatusDone    TaskStatus = "done"
	StatusBlocked TaskStatus = "blocked"
)

// Priority represents task priority
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// TimeEntry represents logged work time
type TimeEntry struct {
	Date     string    `json:"date"`
	Hours    float64   `json:"hours"`
	LoggedAt time.Time `json:"logged_at"`
}

// Recurrence represents recurring task configuration
type Recurrence struct {
	Type          RecurrenceType `json:"type"`
	Value         string         `json:"value"`
	NextDue       string         `json:"next_due"`
	LastCompleted string         `json:"last_completed,omitempty"`
	Enabled       bool           `json:"enabled"`
}

// RecurrenceType defines how often a task repeats
type RecurrenceType string

const (
	RecurDaily    RecurrenceType = "daily"
	RecurWeekly   RecurrenceType = "weekly"
	RecurMonthly  RecurrenceType = "monthly"
	RecurInterval RecurrenceType = "interval"
)

// Sprint represents a time-boxed work period
type Sprint struct {
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	TaskIDs   []string  `json:"task_ids"`
	CreatedAt time.Time `json:"created_at"`
}

// TrackingSession represents an active time tracking session
type TrackingSession struct {
	Path      string    `json:"path"`
	TaskID    string    `json:"task_id"`
	StartTime time.Time `json:"start"`
}

// TrackingData stores all tracking sessions
type TrackingData struct {
	ActiveSession *TrackingSession `json:"active_session"`
	Sessions      []interface{}    `json:"sessions"` // Historical sessions
}

// TaskIndex maps task IDs to their locations for fast lookup
type TaskIndex map[string]TaskLocation

// TaskLocation describes where a task is stored
type TaskLocation struct {
	Project  string `json:"project"`
	Location string `json:"location"` // "project" or "module:<name>"
}

// CalculateActualHours returns total hours from time entries
func (t *Task) CalculateActualHours() float64 {
	total := 0.0
	for _, entry := range t.TimeEntries {
		total += entry.Hours
	}
	return total
}

// IsOverBudget checks if task exceeded estimated hours
func (t *Task) IsOverBudget() bool {
	if t.EstimatedHours == 0 {
		return false
	}
	return t.CalculateActualHours() > t.EstimatedHours
}

// GetVariance returns the difference between actual and estimated hours
func (t *Task) GetVariance() float64 {
	return t.CalculateActualHours() - t.EstimatedHours
}

// GetVariancePercentage returns variance as a percentage
func (t *Task) GetVariancePercentage() float64 {
	if t.EstimatedHours == 0 {
		return 0
	}
	return (t.GetVariance() / t.EstimatedHours) * 100
}

// IsRecurring checks if task has recurrence configured
func (t *Task) IsRecurring() bool {
	return t.Recurrence != nil && t.Recurrence.Enabled
}

// GetAllTasks returns all tasks from project (including modules)
func (p *Project) GetAllTasks() []Task {
	tasks := make([]Task, 0, len(p.Tasks))
	tasks = append(tasks, p.Tasks...)

	for _, module := range p.Modules {
		tasks = append(tasks, module.Tasks...)
	}

	return tasks
}

// CountByStatus returns task counts grouped by status
func (p *Project) CountByStatus() map[TaskStatus]int {
	counts := make(map[TaskStatus]int)
	counts[StatusTodo] = 0
	counts[StatusDoing] = 0
	counts[StatusDone] = 0
	counts[StatusBlocked] = 0

	for _, task := range p.GetAllTasks() {
		counts[task.Status]++
	}

	return counts
}

// CalculateTotalEstimated returns sum of all estimated hours
func (p *Project) CalculateTotalEstimated() float64 {
	total := 0.0
	for _, task := range p.GetAllTasks() {
		total += task.EstimatedHours
	}
	return total
}

// CalculateTotalActual returns sum of all actual hours
func (p *Project) CalculateTotalActual() float64 {
	total := 0.0
	for _, task := range p.GetAllTasks() {
		total += task.CalculateActualHours()
	}
	return total
}

// GetCompletionPercentage returns percentage of completed tasks
func (p *Project) GetCompletionPercentage() float64 {
	counts := p.CountByStatus()
	total := len(p.GetAllTasks())

	if total == 0 {
		return 0
	}

	return (float64(counts[StatusDone]) / float64(total)) * 100
}
