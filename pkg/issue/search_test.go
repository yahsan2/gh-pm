package issue

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/filter"
)

func TestNewSearchClient(t *testing.T) {
	// Skip this test if no GitHub token is available (local testing)
	if os.Getenv("GITHUB_TOKEN") == "" && os.Getenv("CI") == "" {
		t.Skip("Skipping test: requires GitHub authentication")
	}

	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
	}

	client, err := NewSearchClient(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.NotNil(t, client.config)
	assert.NotNil(t, client.projCli)
}

func TestFilterProjectIssues(t *testing.T) {
	// Skip this test if no GitHub token is available (local testing)
	if os.Getenv("GITHUB_TOKEN") == "" && os.Getenv("CI") == "" {
		t.Skip("Skipping test: requires GitHub authentication")
	}

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

	client, err := NewSearchClient(cfg)
	require.NoError(t, err)

	issues := []filter.ProjectIssue{
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
		filters := &filter.IssueFilters{State: "open"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 3, filtered[1].Number)
	})

	t.Run("Filter by labels", func(t *testing.T) {
		filters := &filter.IssueFilters{Labels: []string{"bug"}}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 3, filtered[1].Number)
	})

	t.Run("Filter by multiple labels", func(t *testing.T) {
		filters := &filter.IssueFilters{Labels: []string{"bug", "documentation"}}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 3) // Should match issues with ANY of the labels
	})

	t.Run("Filter by assignee", func(t *testing.T) {
		filters := &filter.IssueFilters{Assignee: "user1"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)
	})

	t.Run("Filter by author", func(t *testing.T) {
		filters := &filter.IssueFilters{Author: "author1"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 3, filtered[1].Number)
	})

	t.Run("Filter by milestone", func(t *testing.T) {
		filters := &filter.IssueFilters{Milestone: "v1.0"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)
	})

	t.Run("Filter by search", func(t *testing.T) {
		filters := &filter.IssueFilters{Search: "another"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 1)
		assert.Equal(t, 2, filtered[0].Number)
	})

	t.Run("Filter by search in title", func(t *testing.T) {
		filters := &filter.IssueFilters{Search: "authentication"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 1)
		assert.Equal(t, 3, filtered[0].Number)
	})

	t.Run("Filter by status (project field)", func(t *testing.T) {
		filters := &filter.IssueFilters{Status: "in_progress"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)
	})

	t.Run("Filter by priority (project field)", func(t *testing.T) {
		filters := &filter.IssueFilters{Priority: "p2"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 1)
		assert.Equal(t, 3, filtered[0].Number)
	})

	t.Run("Filter by multiple priorities", func(t *testing.T) {
		filters := &filter.IssueFilters{Priority: "p0,p1"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 2, filtered[1].Number)
	})

	t.Run("No matches", func(t *testing.T) {
		filters := &filter.IssueFilters{State: "unknown"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 0)
	})

	t.Run("Combined filters", func(t *testing.T) {
		filters := &filter.IssueFilters{
			State:  "open",
			Author: "author1",
		}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 2)
		assert.Equal(t, 1, filtered[0].Number)
		assert.Equal(t, 3, filtered[1].Number)
	})
}

func TestMatchesFieldValue(t *testing.T) {
	// Skip this test if no GitHub token is available (local testing)
	if os.Getenv("GITHUB_TOKEN") == "" && os.Getenv("CI") == "" {
		t.Skip("Skipping test: requires GitHub authentication")
	}

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

	client, err := NewSearchClient(cfg)
	require.NoError(t, err)

	t.Run("Direct match", func(t *testing.T) {
		result := client.matchesFieldValue("status", "Backlog", "Backlog")
		assert.True(t, result)
	})

	t.Run("Case insensitive direct match", func(t *testing.T) {
		result := client.matchesFieldValue("status", "backlog", "Backlog")
		assert.True(t, result)
	})

	t.Run("Config mapping match", func(t *testing.T) {
		result := client.matchesFieldValue("status", "in_progress", "In Progress")
		assert.True(t, result)
	})

	t.Run("Reverse mapping match", func(t *testing.T) {
		result := client.matchesFieldValue("status", "In Progress", "in_progress")
		assert.True(t, result)
	})

	t.Run("No match", func(t *testing.T) {
		result := client.matchesFieldValue("status", "invalid", "Backlog")
		assert.False(t, result)
	})

	t.Run("Field not in config", func(t *testing.T) {
		result := client.matchesFieldValue("unknown", "value", "value")
		assert.True(t, result) // Should fall back to direct match
	})
}

func TestSearchClientEdgeCases(t *testing.T) {
	// Skip this test if no GitHub token is available (local testing)
	if os.Getenv("GITHUB_TOKEN") == "" && os.Getenv("CI") == "" {
		t.Skip("Skipping test: requires GitHub authentication")
	}

	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
	}

	client, err := NewSearchClient(cfg)
	require.NoError(t, err)

	t.Run("Empty issues list", func(t *testing.T) {
		filters := &filter.IssueFilters{State: "open"}
		filtered := client.FilterProjectIssues([]filter.ProjectIssue{}, filters)

		assert.Len(t, filtered, 0)
	})

	t.Run("Filter with all state", func(t *testing.T) {
		issues := []filter.ProjectIssue{
			{Number: 1, State: "open"},
			{Number: 2, State: "closed"},
		}
		filters := &filter.IssueFilters{State: "all"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 2)
	})

	t.Run("Empty search filter", func(t *testing.T) {
		issues := []filter.ProjectIssue{
			{Number: 1, Title: "Test", Body: "Content"},
		}
		filters := &filter.IssueFilters{Search: ""}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 1)
	})

	t.Run("No assignees", func(t *testing.T) {
		issues := []filter.ProjectIssue{
			{Number: 1, Assignees: []string{}},
		}
		filters := &filter.IssueFilters{Assignee: "user1"}
		filtered := client.FilterProjectIssues(issues, filters)

		assert.Len(t, filtered, 0)
	})
}
