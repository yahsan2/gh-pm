package issue

import (
	"fmt"
	
	"github.com/cli/go-gh/v2/pkg/api"
)

// Client is a wrapper around GitHub API client for issue operations
type Client struct {
	rest *api.RESTClient
	gql  *api.GraphQLClient
}

// NewClient creates a new issue client
func NewClient() (*Client, error) {
	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, NewAPIError("failed to create REST client", err)
	}
	
	gqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, NewAPIError("failed to create GraphQL client", err)
	}
	
	return &Client{
		rest: restClient,
		gql:  gqlClient,
	}, nil
}

// CreateIssue creates a new issue - delegates to Creator
func (c *Client) CreateIssue(data *IssueData) (*Issue, error) {
	creator := NewCreator(c)
	return creator.CreateIssue(data)
}

// AddToProject adds an issue to a project and returns the project item ID
func (c *Client) AddToProject(issueID, projectID string) (string, error) {
	// Add issue to project using GraphQL
	mutation := `
		mutation($projectId: ID!, $contentId: ID!) {
			addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
				item {
					id
					databaseId
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
				ID         string `json:"id"`
				DatabaseID int    `json:"databaseId"`
			} `json:"item"`
		} `json:"addProjectV2ItemById"`
	}
	
	err := c.gql.Do(mutation, variables, &result)
	if err != nil {
		return "", NewAPIError("failed to add issue to project", err)
	}
	
	return result.AddProjectV2ItemById.Item.ID, nil
}

// AddToProjectWithDatabaseID adds an issue to a project and returns both IDs
func (c *Client) AddToProjectWithDatabaseID(issueID, projectID string) (string, int, error) {
	// Add issue to project using GraphQL
	mutation := `
		mutation($projectId: ID!, $contentId: ID!) {
			addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
				item {
					id
					databaseId
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
				ID         string `json:"id"`
				DatabaseID int    `json:"databaseId"`
			} `json:"item"`
		} `json:"addProjectV2ItemById"`
	}
	
	err := c.gql.Do(mutation, variables, &result)
	if err != nil {
		return "", 0, NewAPIError("failed to add issue to project", err)
	}
	
	return result.AddProjectV2ItemById.Item.ID, result.AddProjectV2ItemById.Item.DatabaseID, nil
}

// UpdateProjectItemField updates a single field value for a project item
func (c *Client) UpdateProjectItemField(projectID, itemID, fieldID, optionID string) error {
	mutation := `
		mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
			updateProjectV2ItemFieldValue(
				input: {
					projectId: $projectId
					itemId: $itemId
					fieldId: $fieldId
					value: { singleSelectOptionId: $optionId }
				}
			) {
				projectV2Item {
					id
				}
			}
		}`
	
	variables := map[string]interface{}{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   fieldID,
		"optionId":  optionID,
	}
	
	var result struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID string `json:"id"`
			} `json:"projectV2Item"`
		} `json:"updateProjectV2ItemFieldValue"`
	}
	
	if err := c.gql.Do(mutation, variables, &result); err != nil {
		return NewAPIError(fmt.Sprintf("failed to update field %s", fieldID), err)
	}
	
	return nil
}