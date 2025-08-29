package issue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"
	
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

// GetGraphQLClient returns the GraphQL client for direct use
func (c *Client) GetGraphQLClient() *api.GraphQLClient {
	return c.gql
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

// GetProjectItemID gets the project item ID for an issue if it exists in the project
func (c *Client) GetProjectItemID(issueID, projectID string) (string, int, error) {
	query := `
		query($issueId: ID!) {
			node(id: $issueId) {
				... on Issue {
					projectItems(first: 100) {
						nodes {
							id
							databaseId
							project {
								id
							}
						}
					}
				}
			}
		}`
	
	variables := map[string]interface{}{
		"issueId": issueID,
	}
	
	var result struct {
		Node struct {
			ProjectItems struct {
				Nodes []struct {
					ID         string `json:"id"`
					DatabaseID int    `json:"databaseId"`
					Project    struct {
						ID string `json:"id"`
					} `json:"project"`
				} `json:"nodes"`
			} `json:"projectItems"`
		} `json:"node"`
	}
	
	err := c.gql.Do(query, variables, &result)
	if err != nil {
		return "", 0, NewAPIError("failed to get project item", err)
	}
	
	// Find the item for the specified project
	for _, item := range result.Node.ProjectItems.Nodes {
		if item.Project.ID == projectID {
			return item.ID, item.DatabaseID, nil
		}
	}
	
	return "", 0, nil // Not found in project
}

// GetIssueDetails fetches issue details using gh issue view command
func GetIssueDetails(number int, repo string) (*Issue, error) {
	args := []string{"issue", "view", strconv.Itoa(number), "--json", "id,number,title,body,url,state,createdAt,updatedAt,labels"}
	
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	
	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get issue details: %w\nstderr: %s", err, stderr.String())
	}
	
	var result struct {
		ID        string    `json:"id"`
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		Body      string    `json:"body"`
		URL       string    `json:"url"`
		State     string    `json:"state"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
		Labels    []struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"labels"`
	}
	
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}
	
	// Convert labels
	labels := make([]Label, len(result.Labels))
	for i, l := range result.Labels {
		labels[i] = Label{
			Name:  l.Name,
			Color: l.Color,
		}
	}
	
	return &Issue{
		ID:         result.ID,
		Number:     result.Number,
		Title:      result.Title,
		Body:       result.Body,
		URL:        result.URL,
		State:      result.State,
		Repository: repo,
		Labels:     labels,
		CreatedAt:  result.CreatedAt,
		UpdatedAt:  result.UpdatedAt,
	}, nil
}