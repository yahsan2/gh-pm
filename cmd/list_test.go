package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	
	"github.com/yahsan2/gh-pm/pkg/config"
)

func TestFilterIssues(t *testing.T) {
	cfg := &config.Config{
		Fields: map[string]config.Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"backlog":     "Backlog",
					"in_progress": "In Progress",
					"done":        "Done",
				},
			},
			"priority": {
				Field: "Priority",
				Values: map[string]string{
					"p0": "P0",
					"p1": "P1",
					"p2": "P2",
				},
			},
		},
	}

	command := &ListCommand{
		config: cfg,
	}

	issues := []ProjectIssue{
		{
			Number:    1,
			Title:     "Test Issue 1",
			State:     "open",
			Labels:    []string{"bug", "enhancement"},
			Assignees: []string{"user1"},
			Author:    "author1",
			Milestone: "v1.0",
			Body:      "This is a test issue",
			Fields: map[string]interface{}{
				"Status":   "In Progress",
				"Priority": "P0",
			},
		},
		{
			Number:    2,
			Title:     "Test Issue 2",
			State:     "closed",
			Labels:    []string{"documentation"},
			Assignees: []string{"user2"},
			Author:    "author2",
			Milestone: "v2.0",
			Body:      "Another test issue",
			Fields: map[string]interface{}{
				"Status":   "Done",
				"Priority": "P1",
			},
		},
		{
			Number:    3,
			Title:     "Test Issue 3",
			State:     "open",
			Labels:    []string{"bug"},
			Assignees: []string{},
			Author:    "author1",
			Milestone: "",
			Body:      "Authentication error in login",
			Fields: map[string]interface{}{
				"Status":   "Backlog",
				"Priority": "P2",
			},
		},
	}

	t.Run("Filter by state", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			State: "open",
		})
		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 3, filtered[1].Number)
	})

	t.Run("Filter by labels", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Labels: []string{"bug"},
		})
		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 3, filtered[1].Number)
	})

	t.Run("Filter by multiple labels", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Labels: []string{"bug", "documentation"},
		})
		assert.Len(t, filtered, 3)
	})

	t.Run("Filter by assignee", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Assignee: "user1",
		})
		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)
	})

	t.Run("Filter by author", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Author: "author1",
		})
		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 3, filtered[1].Number)
	})

	t.Run("Filter by milestone", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Milestone: "v1.0",
		})
		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)
	})

	t.Run("Filter by search", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Search: "authentication",
		})
		assert.Len(t, filtered, 1)
		assert.Equal(t, 3, filtered[0].Number)
	})

	t.Run("Filter by status using config key", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Status: "in_progress",
		})
		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)
	})

	t.Run("Filter by status using actual value", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Status: "Done",
		})
		assert.Len(t, filtered, 1)
		assert.Equal(t, 2, filtered[0].Number)
	})

	t.Run("Filter by priority", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Priority: "p0",
		})
		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)
	})

	t.Run("Filter by multiple priorities", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			Priority: "p0,p1",
		})
		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 2, filtered[1].Number)
	})

	t.Run("Combined filters", func(t *testing.T) {
		filtered := command.filterIssues(issues, FilterOptions{
			State:    "open",
			Labels:   []string{"bug"},
			Priority: "p0,p2",
		})
		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 3, filtered[1].Number)
	})
}

func TestMatchesFieldValue(t *testing.T) {
	cfg := &config.Config{
		Fields: map[string]config.Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"backlog":     "Backlog",
					"in_progress": "In Progress",
					"done":        "Done",
				},
			},
		},
	}

	tests := []struct {
		name        string
		fieldName   string
		filterValue string
		actualValue string
		expected    bool
	}{
		{
			name:        "Direct match",
			fieldName:   "status",
			filterValue: "Backlog",
			actualValue: "Backlog",
			expected:    true,
		},
		{
			name:        "Case insensitive match",
			fieldName:   "status",
			filterValue: "backlog",
			actualValue: "Backlog",
			expected:    true,
		},
		{
			name:        "Config key match",
			fieldName:   "status",
			filterValue: "in_progress",
			actualValue: "In Progress",
			expected:    true,
		},
		{
			name:        "No match",
			fieldName:   "status",
			filterValue: "todo",
			actualValue: "Backlog",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesFieldValue(cfg, tt.fieldName, tt.filterValue, tt.actualValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "Short string",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "Exact length",
			input:    "Hello World",
			maxLen:   11,
			expected: "Hello World",
		},
		{
			name:     "Long string",
			input:    "This is a very long string that needs truncation",
			maxLen:   20,
			expected: "This is a very lo...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
