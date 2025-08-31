package project

import (
	"fmt"

	"github.com/yahsan2/gh-pm/pkg/config"
)

// URLBuilder helps build GitHub project URLs
type URLBuilder struct {
	config *config.Config
	client *Client
}

// NewURLBuilder creates a new URL builder
func NewURLBuilder(cfg *config.Config, client *Client) *URLBuilder {
	return &URLBuilder{
		config: cfg,
		client: client,
	}
}

// GetProjectURL returns the base project URL
func (b *URLBuilder) GetProjectURL() string {
	projectNumber := b.config.Project.Number
	if projectNumber == 0 {
		return ""
	}

	// For organization projects
	if b.config.Project.Org != "" {
		return fmt.Sprintf("https://github.com/orgs/%s/projects/%d", b.config.Project.Org, projectNumber)
	}

	// For user projects
	owner := b.config.Project.Owner
	if owner == "" {
		return ""
	}

	return fmt.Sprintf("https://github.com/users/%s/projects/%d", owner, projectNumber)
}

// GetProjectItemURL returns the URL for a specific project item (issue/PR)
func (b *URLBuilder) GetProjectItemURL(itemDatabaseID int) string {
	baseURL := b.GetProjectURL()
	if baseURL == "" {
		return ""
	}

	return fmt.Sprintf("%s?pane=issue&itemId=%d", baseURL, itemDatabaseID)
}

// GetProjectItemURLFromIssueID returns the project URL for an issue given its node ID
func (b *URLBuilder) GetProjectItemURLFromIssueID(issueNodeID string, issueClient interface {
	GetProjectItemID(string, string) (string, int, error)
}) string {
	projectID := b.config.GetProjectID()
	if projectID == "" {
		return ""
	}

	_, itemDatabaseID, err := issueClient.GetProjectItemID(issueNodeID, projectID)
	if err != nil || itemDatabaseID == 0 {
		return ""
	}

	return b.GetProjectItemURL(itemDatabaseID)
}
