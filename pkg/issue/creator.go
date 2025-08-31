package issue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Creator handles issue creation and project management
type Creator struct {
	client *Client
}

// NewCreator creates a new issue creator
func NewCreator(client *Client) *Creator {
	return &Creator{
		client: client,
	}
}

// CreateIssue creates a new GitHub issue
func (c *Creator) CreateIssue(data *IssueData) (*Issue, error) {
	// Validate issue data
	if err := data.Validate(); err != nil {
		return nil, NewValidationError("invalid issue data", err)
	}

	// Prepare the request
	requestData := data.ToCreateRequest()

	// Convert request to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, NewAPIError("failed to marshal request", err)
	}

	// Create the issue using REST API
	path := fmt.Sprintf("repos/%s/%s/issues", data.GetOwner(), data.GetRepo())

	var response map[string]interface{}
	err = c.client.rest.Post(path, bytes.NewReader(jsonData), &response)
	if err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			return nil, NewPermissionError("failed to create issue", err)
		}
		if strings.Contains(err.Error(), "404") {
			return nil, NewNotFoundError(fmt.Sprintf("repository %s", data.Repository))
		}
		return nil, NewAPIError("failed to create issue", err)
	}

	// Parse the response
	issue := &Issue{
		Repository: data.Repository,
	}

	// Extract fields from response
	if id, ok := response["node_id"].(string); ok {
		issue.ID = id
	}
	if number, ok := response["number"].(float64); ok {
		issue.Number = int(number)
	}
	if title, ok := response["title"].(string); ok {
		issue.Title = title
	}
	if url, ok := response["html_url"].(string); ok {
		issue.URL = url
	}
	if state, ok := response["state"].(string); ok {
		issue.State = state
	}

	// Parse timestamps
	if createdAt, ok := response["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			issue.CreatedAt = t
		}
	}
	if updatedAt, ok := response["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			issue.UpdatedAt = t
		}
	}

	// Parse labels
	if labels, ok := response["labels"].([]interface{}); ok {
		issue.Labels = make([]Label, 0, len(labels))
		for _, l := range labels {
			if labelMap, ok := l.(map[string]interface{}); ok {
				label := Label{}
				if name, ok := labelMap["name"].(string); ok {
					label.Name = name
				}
				if color, ok := labelMap["color"].(string); ok {
					label.Color = color
				}
				issue.Labels = append(issue.Labels, label)
			}
		}
	}

	return issue, nil
}

// AddToProject adds an issue to a GitHub Project v2
func (c *Creator) AddToProject(issueID, projectID string) error {
	if issueID == "" {
		return NewValidationError("issue ID is required", nil)
	}
	if projectID == "" {
		return NewValidationError("project ID is required", nil)
	}

	// GraphQL mutation to add item to project
	mutation := `
		mutation($projectId: ID!, $contentId: ID!) {
			addProjectV2ItemById(input: {
				projectId: $projectId
				contentId: $contentId
			}) {
				item {
					id
				}
			}
		}`

	variables := map[string]interface{}{
		"projectId": projectID,
		"contentId": issueID,
	}

	var result struct {
		AddProjectV2ItemById struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"addProjectV2ItemById"`
	}

	err := c.client.gql.Do(mutation, variables, &result)
	if err != nil {
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			return NewPermissionError("failed to add issue to project", err)
		}
		if strings.Contains(err.Error(), "NOT_FOUND") {
			return NewNotFoundError("project or issue")
		}
		return NewAPIError("failed to add issue to project", err)
	}

	return nil
}

// UpdateFields updates project field values for an issue
func (c *Creator) UpdateFields(itemID string, fields map[string]interface{}) error {
	if itemID == "" {
		return NewValidationError("item ID is required", nil)
	}
	if len(fields) == 0 {
		return nil // Nothing to update
	}

	// For each field, we need to update it with the appropriate mutation
	for fieldName, value := range fields {
		if err := c.updateSingleField(itemID, fieldName, value); err != nil {
			return WrapError(err, fmt.Sprintf("failed to update field '%s'", fieldName))
		}
	}

	return nil
}

// updateSingleField updates a single project field
func (c *Creator) updateSingleField(itemID string, fieldName string, value interface{}) error {
	// This is a simplified implementation
	// In a real implementation, we would need to:
	// 1. Get the field ID and type from project metadata
	// 2. Use the appropriate mutation based on field type (text, single_select, etc.)
	// 3. Convert the value to the appropriate format

	// For now, we'll return a placeholder error
	return fmt.Errorf("field update not fully implemented - requires project metadata")
}

// GetProjectItemID retrieves the project item ID for an issue
func (c *Creator) GetProjectItemID(issueID, projectID string) (string, error) {
	query := `
		query($issueId: ID!, $projectId: ID!) {
			node(id: $issueId) {
				... on Issue {
					projectItems(first: 100) {
						nodes {
							id
							project {
								id
							}
						}
					}
				}
			}
		}`

	variables := map[string]interface{}{
		"issueId":   issueID,
		"projectId": projectID,
	}

	var result struct {
		Node struct {
			ProjectItems struct {
				Nodes []struct {
					ID      string `json:"id"`
					Project struct {
						ID string `json:"id"`
					} `json:"project"`
				} `json:"nodes"`
			} `json:"projectItems"`
		} `json:"node"`
	}

	err := c.client.gql.Do(query, variables, &result)
	if err != nil {
		return "", NewAPIError("failed to get project item", err)
	}

	// Find the item for the specified project
	for _, item := range result.Node.ProjectItems.Nodes {
		if item.Project.ID == projectID {
			return item.ID, nil
		}
	}

	return "", NewNotFoundError("project item")
}
