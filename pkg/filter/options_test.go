package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIssueFilters(t *testing.T) {
	filters := NewIssueFilters()

	assert.Equal(t, "open", filters.State)
	assert.Equal(t, 100, filters.Limit)
	assert.Empty(t, filters.Labels)
	assert.Empty(t, filters.Search)
}

func TestProjectIssueFields(t *testing.T) {
	issue := ProjectIssue{
		Number: 1,
		Title:  "Test Issue",
		State:  "open",
		Fields: make(map[string]interface{}),
	}

	issue.Fields["Status"] = "In Progress"
	issue.Fields["Priority"] = "P1"

	assert.Equal(t, "In Progress", issue.Fields["Status"])
	assert.Equal(t, "P1", issue.Fields["Priority"])
}

func TestGitHubIssueBasic(t *testing.T) {
	issue := GitHubIssue{
		Number: 42,
		Title:  "Sample Issue",
		ID:     "gid_123",
		URL:    "https://github.com/owner/repo/issues/42",
	}

	assert.Equal(t, 42, issue.Number)
	assert.Equal(t, "Sample Issue", issue.Title)
	assert.Equal(t, "gid_123", issue.ID)
	assert.Equal(t, "https://github.com/owner/repo/issues/42", issue.URL)
}
