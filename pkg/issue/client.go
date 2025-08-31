package issue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client is a wrapper around GitHub API client for issue operations
type Client struct {
	rest *api.RESTClient
	gql  *api.GraphQLClient
}

// NewClient creates a new issue client
func NewClient() *Client {
	restClient, _ := api.DefaultRESTClient()
	gqlClient, _ := api.DefaultGraphQLClient()

	return &Client{
		rest: restClient,
		gql:  gqlClient,
	}
}

// GetGraphQLClient returns the GraphQL client for direct use
func (c *Client) GetGraphQLClient() *api.GraphQLClient {
	return c.gql
}

// CreateIssueWithData creates a new issue - delegates to Creator
func (c *Client) CreateIssueWithData(data *IssueData) (*Issue, error) {
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

// GetIssue fetches an issue by number
func (c *Client) GetIssue(number int) (Issue, error) {
	return c.GetIssueWithRepo(number, "")
}

// GetIssueWithRepo fetches an issue by number from a specific repo
func (c *Client) GetIssueWithRepo(number int, repo string) (Issue, error) {
	args := []string{"issue", "view", strconv.Itoa(number), "--json", "id,number,title,body,url,state,labels,assignees,milestone"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Issue{}, fmt.Errorf("failed to get issue: %w\nstderr: %s", err, stderr.String())
	}

	var result struct {
		ID     string `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		URL    string `json:"url"`
		State  string `json:"state"`
		Labels []struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"labels"`
		Assignees []struct {
			Login string `json:"login"`
		} `json:"assignees"`
		Milestone struct {
			Title string `json:"title"`
		} `json:"milestone"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return Issue{}, fmt.Errorf("failed to parse issue: %w", err)
	}

	labels := make([]Label, len(result.Labels))
	for i, l := range result.Labels {
		labels[i] = Label{
			Name:  l.Name,
			Color: l.Color,
		}
	}

	assignees := make([]string, len(result.Assignees))
	for i, a := range result.Assignees {
		assignees[i] = a.Login
	}

	return Issue{
		ID:        result.ID,
		Number:    result.Number,
		Title:     result.Title,
		Body:      result.Body,
		URL:       result.URL,
		State:     result.State,
		Labels:    labels,
		Assignees: assignees,
		Milestone: result.Milestone.Title,
	}, nil
}

// CreateIssueFromRequest creates a new issue from IssueRequest
func (c *Client) CreateIssue(req IssueRequest) (Issue, error) {
	return c.CreateIssueWithRepo(req, "")
}

// CreateIssueWithRepo creates a new issue in a specific repo
func (c *Client) CreateIssueWithRepo(req IssueRequest, repo string) (Issue, error) {
	args := []string{"issue", "create", "--title", req.Title}

	if req.Body != "" {
		args = append(args, "--body", req.Body)
	}

	for _, label := range req.Labels {
		args = append(args, "--label", label)
	}

	for _, assignee := range req.Assignees {
		args = append(args, "--assignee", assignee)
	}

	if req.Milestone != "" {
		args = append(args, "--milestone", req.Milestone)
	}

	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Issue{}, fmt.Errorf("failed to create issue: %w\nstderr: %s", err, stderr.String())
	}

	// Parse the URL from output
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return Issue{}, fmt.Errorf("no output from gh issue create")
	}

	// Extract issue number from URL (format: https://github.com/owner/repo/issues/123)
	parts := strings.Split(output, "/")
	if len(parts) < 2 {
		return Issue{}, fmt.Errorf("unexpected output format: %s", output)
	}

	numberStr := parts[len(parts)-1]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return Issue{}, fmt.Errorf("failed to parse issue number from URL: %s", output)
	}

	return Issue{
		Number: number,
		Title:  req.Title,
		URL:    output,
	}, nil
}

// UpdateIssue updates an existing issue
func (c *Client) UpdateIssue(number int, req IssueRequest) error {
	return c.UpdateIssueWithRepo(number, req, "")
}

// UpdateIssueWithRepo updates an existing issue in a specific repo
func (c *Client) UpdateIssueWithRepo(number int, req IssueRequest, repo string) error {
	args := []string{"issue", "edit", strconv.Itoa(number)}

	if req.Title != "" {
		args = append(args, "--title", req.Title)
	}

	if req.Body != "" {
		args = append(args, "--body", req.Body)
	}

	if len(req.Labels) > 0 {
		// Clear existing labels and add new ones
		args = append(args, "--remove-label", "*")
		for _, label := range req.Labels {
			args = append(args, "--add-label", label)
		}
	}

	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.Command("gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update issue: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

