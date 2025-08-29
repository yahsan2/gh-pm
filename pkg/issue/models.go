package issue

import (
	"fmt"
	"strings"
	"time"
)

// IssueData represents the data needed to create an issue
type IssueData struct {
	Title        string            `yaml:"title" json:"title"`
	Body         string            `yaml:"body" json:"body"`
	Labels       []string          `yaml:"labels" json:"labels"`
	Repository   string            `yaml:"repository" json:"repository"`
	Priority     string            `yaml:"priority" json:"priority"`
	Status       string            `yaml:"status" json:"status"`
	CustomFields map[string]string `yaml:"custom_fields" json:"custom_fields"`
	
	// Pass-through fields for gh issue create compatibility
	Assignee  string `yaml:"assignee" json:"assignee"`
	Milestone string `yaml:"milestone" json:"milestone"`
}

// Issue represents a created GitHub issue with project metadata
type Issue struct {
	ID          string       `json:"id"`
	Number      int          `json:"number"`
	Title       string       `json:"title"`
	URL         string       `json:"url"`
	State       string       `json:"state"`
	Repository  string       `json:"repository"`
	Labels      []Label      `json:"labels"`
	ProjectItem *ProjectItem `json:"project_item,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Label represents a GitHub issue label
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ProjectItem represents an issue's connection to a project
type ProjectItem struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	Fields    map[string]interface{} `json:"fields"`
}

// Validate checks if the issue data is valid
func (d *IssueData) Validate() error {
	// Check required fields
	if d.Title == "" {
		return fmt.Errorf("issue title is required")
	}
	
	if len(d.Title) > 256 {
		return fmt.Errorf("issue title must be 256 characters or less")
	}
	
	if d.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	
	// Validate repository format (owner/repo)
	parts := strings.Split(d.Repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("repository must be in 'owner/repo' format")
	}
	
	// Validate labels
	for _, label := range d.Labels {
		if label == "" {
			return fmt.Errorf("empty label is not allowed")
		}
		if len(label) > 50 {
			return fmt.Errorf("label '%s' exceeds maximum length of 50 characters", label)
		}
	}
	
	return nil
}

// GetOwner returns the repository owner
func (d *IssueData) GetOwner() string {
	parts := strings.Split(d.Repository, "/")
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}

// GetRepo returns the repository name
func (d *IssueData) GetRepo() string {
	parts := strings.Split(d.Repository, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// ToCreateRequest converts IssueData to a format suitable for GitHub API
func (d *IssueData) ToCreateRequest() map[string]interface{} {
	req := map[string]interface{}{
		"title": d.Title,
	}
	
	if d.Body != "" {
		req["body"] = d.Body
	}
	
	if len(d.Labels) > 0 {
		req["labels"] = d.Labels
	}
	
	if d.Assignee != "" {
		req["assignees"] = []string{d.Assignee}
	}
	
	if d.Milestone != "" {
		req["milestone"] = d.Milestone
	}
	
	return req
}

// GetFieldUpdates returns project field updates
func (d *IssueData) GetFieldUpdates() map[string]interface{} {
	fields := make(map[string]interface{})
	
	if d.Priority != "" {
		fields["priority"] = d.Priority
	}
	
	if d.Status != "" {
		fields["status"] = d.Status
	}
	
	// Add custom fields
	for key, value := range d.CustomFields {
		fields[key] = value
	}
	
	return fields
}