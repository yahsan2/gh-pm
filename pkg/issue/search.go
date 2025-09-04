package issue

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/filter"
	"github.com/yahsan2/gh-pm/pkg/project"
	"github.com/yahsan2/gh-pm/pkg/utils"
)

// SearchClient handles issue searching and filtering operations
type SearchClient struct {
	client  *Client
	config  *config.Config
	projCli *project.Client
}

// NewSearchClient creates a new SearchClient
func NewSearchClient(cfg *config.Config) (*SearchClient, error) {
	issueClient := NewClient()

	projectClient, err := project.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create project client: %w", err)
	}

	return &SearchClient{
		client:  issueClient,
		config:  cfg,
		projCli: projectClient,
	}, nil
}

// SearchIssues searches for issues using GitHub API with the provided filters
func (s *SearchClient) SearchIssues(filters *filter.IssueFilters) ([]filter.GitHubIssue, error) {
	var args []string

	// Use configured repository
	if len(s.config.Repositories) > 0 {
		args = append(args, "issue", "list", "--repo", s.config.Repositories[0])
	} else {
		args = append(args, "issue", "list")
	}

	// Add state filter
	if filters.State != "" {
		args = append(args, "--state", filters.State)
	}

	// Add label filters
	for _, label := range filters.Labels {
		args = append(args, "--label", label)
	}

	// Add assignee filter
	if filters.Assignee != "" {
		args = append(args, "--assignee", filters.Assignee)
	}

	// Add author filter
	if filters.Author != "" {
		args = append(args, "--author", filters.Author)
	}

	// Add milestone filter
	if filters.Milestone != "" {
		args = append(args, "--milestone", filters.Milestone)
	}

	// Add mention filter
	if filters.Mention != "" {
		args = append(args, "--mention", filters.Mention)
	}

	// Add app filter
	if filters.App != "" {
		args = append(args, "--app", filters.App)
	}

	// Convert and add search filter
	if filters.Search != "" {
		convertedSearch, err := utils.ConvertSearchQuery(filters.Search)
		if err != nil {
			// If conversion fails, use original search query and log warning
			fmt.Fprintf(os.Stderr, "Warning: Failed to convert date expressions in search query: %v\n", err)
			convertedSearch = filters.Search
		}
		args = append(args, "--search", convertedSearch)
	}

	// Add limit
	if filters.Limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", filters.Limit))
	}

	// Add JSON output
	args = append(args, "--json", "number,title,url,id")

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w\nOutput: %s", err, string(output))
	}

	var issues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		URL    string `json:"url"`
		ID     string `json:"id"`
	}

	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	// Convert to GitHubIssue format
	var result []filter.GitHubIssue
	for _, issue := range issues {
		result = append(result, filter.GitHubIssue{
			Number: issue.Number,
			Title:  issue.Title,
			ID:     issue.ID,
			URL:    issue.URL,
		})
	}

	return result, nil
}

// GetProjectIssues fetches all issues in the specified project
func (s *SearchClient) GetProjectIssues(projectID string) ([]filter.GitHubIssue, error) {
	// Use GraphQL to get all issues in the project
	query := `
		query($projectId: ID!, $endCursor: String) {
			node(id: $projectId) {
				... on ProjectV2 {
					items(first: 100, after: $endCursor) {
						pageInfo {
							hasNextPage
							endCursor
						}
						nodes {
							content {
								... on Issue {
									id
									number
									title
									url
								}
							}
						}
					}
				}
			}
		}`

	var allIssues []filter.GitHubIssue
	var endCursor *string

	for {
		variables := map[string]interface{}{
			"projectId": projectID,
		}
		if endCursor != nil {
			variables["endCursor"] = *endCursor
		}

		var result struct {
			Node struct {
				Items struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						Content struct {
							ID     string `json:"id"`
							Number int    `json:"number"`
							Title  string `json:"title"`
							URL    string `json:"url"`
						} `json:"content"`
					} `json:"nodes"`
				} `json:"items"`
			} `json:"node"`
		}

		err := s.client.GetGraphQLClient().Do(query, variables, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch project items: %w", err)
		}

		// Process items
		for _, item := range result.Node.Items.Nodes {
			if item.Content.Number > 0 { // Skip non-issues
				allIssues = append(allIssues, filter.GitHubIssue{
					Number: item.Content.Number,
					Title:  item.Content.Title,
					ID:     item.Content.ID,
					URL:    item.Content.URL,
				})
			}
		}

		// Check if there are more pages
		if !result.Node.Items.PageInfo.HasNextPage {
			break
		}
		endCursor = &result.Node.Items.PageInfo.EndCursor
	}

	return allIssues, nil
}

// FetchProjectIssues fetches project issues with field values and applies filtering
func (s *SearchClient) FetchProjectIssues(projectID string, limit int) ([]filter.ProjectIssue, error) {
	query := `
		query($projectId: ID!, $endCursor: String, $limit: Int!) {
			node(id: $projectId) {
				... on ProjectV2 {
					items(first: $limit, after: $endCursor) {
						pageInfo {
							hasNextPage
							endCursor
						}
						nodes {
							id
							databaseId
							fieldValues(first: 20) {
								nodes {
									... on ProjectV2ItemFieldTextValue {
										field {
											... on ProjectV2Field {
												name
											}
										}
										text
									}
									... on ProjectV2ItemFieldNumberValue {
										field {
											... on ProjectV2Field {
												name
											}
										}
										number
									}
									... on ProjectV2ItemFieldDateValue {
										field {
											... on ProjectV2Field {
												name
											}
										}
										date
									}
									... on ProjectV2ItemFieldSingleSelectValue {
										field {
											... on ProjectV2SingleSelectField {
												name
											}
										}
										name
									}
									... on ProjectV2ItemFieldIterationValue {
										field {
											... on ProjectV2IterationField {
												name
											}
										}
										title
									}
								}
							}
							content {
								... on Issue {
									id
									number
									title
									state
									url
									body
									createdAt
									updatedAt
									closedAt
									author {
										login
									}
									assignees(first: 10) {
										nodes {
											login
										}
									}
									labels(first: 20) {
										nodes {
											name
										}
									}
									milestone {
										title
									}
									comments {
										totalCount
									}
								}
							}
						}
					}
				}
			}
		}`

	var allIssues []filter.ProjectIssue
	var endCursor *string

	for {
		variables := map[string]interface{}{
			"projectId": projectID,
			"limit":     limit,
		}
		if endCursor != nil {
			variables["endCursor"] = *endCursor
		}

		var result struct {
			Node struct {
				Items struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID          string `json:"id"`
						DatabaseID  int    `json:"databaseId"`
						FieldValues struct {
							Nodes []interface{} `json:"nodes"`
						} `json:"fieldValues"`
						Content struct {
							ID        string `json:"id"`
							Number    int    `json:"number"`
							Title     string `json:"title"`
							State     string `json:"state"`
							URL       string `json:"url"`
							Body      string `json:"body"`
							CreatedAt string `json:"createdAt"`
							UpdatedAt string `json:"updatedAt"`
							ClosedAt  string `json:"closedAt"`
							Author    struct {
								Login string `json:"login"`
							} `json:"author"`
							Assignees struct {
								Nodes []struct {
									Login string `json:"login"`
								} `json:"nodes"`
							} `json:"assignees"`
							Labels struct {
								Nodes []struct {
									Name string `json:"name"`
								} `json:"nodes"`
							} `json:"labels"`
							Milestone struct {
								Title string `json:"title"`
							} `json:"milestone"`
							Comments struct {
								TotalCount int `json:"totalCount"`
							} `json:"comments"`
						} `json:"content"`
					} `json:"nodes"`
				} `json:"items"`
			} `json:"node"`
		}

		err := s.client.GetGraphQLClient().Do(query, variables, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch project items: %w", err)
		}

		// Process items
		for _, item := range result.Node.Items.Nodes {
			if item.Content.Number == 0 {
				continue // Skip non-issues
			}

			issue := filter.ProjectIssue{
				Number:    item.Content.Number,
				Title:     item.Content.Title,
				State:     strings.ToLower(item.Content.State),
				URL:       item.Content.URL,
				ID:        item.Content.ID,
				Body:      item.Content.Body,
				Author:    item.Content.Author.Login,
				Milestone: item.Content.Milestone.Title,
				CreatedAt: item.Content.CreatedAt,
				UpdatedAt: item.Content.UpdatedAt,
				ClosedAt:  item.Content.ClosedAt,
				Comments:  item.Content.Comments.TotalCount,
				Fields:    make(map[string]interface{}),
			}

			// Add assignees
			for _, assignee := range item.Content.Assignees.Nodes {
				issue.Assignees = append(issue.Assignees, assignee.Login)
			}

			// Add labels
			for _, label := range item.Content.Labels.Nodes {
				issue.Labels = append(issue.Labels, label.Name)
			}

			// Parse field values
			for _, fieldValue := range item.FieldValues.Nodes {
				if fv, ok := fieldValue.(map[string]interface{}); ok {
					if field, ok := fv["field"].(map[string]interface{}); ok {
						if fieldName, ok := field["name"].(string); ok {
							// Get the value based on type
							if text, ok := fv["text"].(string); ok {
								issue.Fields[fieldName] = text
							} else if number, ok := fv["number"].(float64); ok {
								issue.Fields[fieldName] = number
							} else if date, ok := fv["date"].(string); ok {
								issue.Fields[fieldName] = date
							} else if name, ok := fv["name"].(string); ok {
								issue.Fields[fieldName] = name
							} else if title, ok := fv["title"].(string); ok {
								issue.Fields[fieldName] = title
							}
						}
					}
				}
			}

			// Generate project URL
			if s.projCli != nil {
				urlBuilder := project.NewURLBuilder(s.config, s.projCli)
				issue.ProjectURL = urlBuilder.GetProjectItemURL(item.DatabaseID)
			}

			allIssues = append(allIssues, issue)
		}

		// Check if we've fetched enough or there are no more pages
		if len(allIssues) >= limit || !result.Node.Items.PageInfo.HasNextPage {
			break
		}
		endCursor = &result.Node.Items.PageInfo.EndCursor
	}

	// Trim to limit
	if len(allIssues) > limit {
		allIssues = allIssues[:limit]
	}

	return allIssues, nil
}

// FilterProjectIssues applies local filtering to project issues
func (s *SearchClient) FilterProjectIssues(issues []filter.ProjectIssue, filters *filter.IssueFilters) []filter.ProjectIssue {
	var filtered []filter.ProjectIssue

	for _, issue := range issues {
		// State filter
		if filters.State != "" && filters.State != "all" {
			if filters.State != issue.State {
				continue
			}
		}

		// Label filter
		if len(filters.Labels) > 0 {
			hasLabel := false
			for _, filterLabel := range filters.Labels {
				for _, issueLabel := range issue.Labels {
					if strings.EqualFold(issueLabel, filterLabel) {
						hasLabel = true
						break
					}
				}
				if hasLabel {
					break
				}
			}
			if !hasLabel {
				continue
			}
		}

		// Assignee filter
		if filters.Assignee != "" {
			hasAssignee := false
			targetAssignee := filters.Assignee
			if targetAssignee == "@me" {
				// Get current user
				cmd := exec.Command("gh", "api", "user", "--jq", ".login")
				output, err := cmd.Output()
				if err == nil {
					targetAssignee = strings.TrimSpace(string(output))
				}
			}

			for _, assignee := range issue.Assignees {
				if strings.EqualFold(assignee, targetAssignee) {
					hasAssignee = true
					break
				}
			}
			if !hasAssignee {
				continue
			}
		}

		// Author filter
		if filters.Author != "" && !strings.EqualFold(issue.Author, filters.Author) {
			continue
		}

		// Milestone filter
		if filters.Milestone != "" && !strings.EqualFold(issue.Milestone, filters.Milestone) {
			continue
		}

		// Search filter (basic text search in title and body)
		if filters.Search != "" {
			searchLower := strings.ToLower(filters.Search)
			if !strings.Contains(strings.ToLower(issue.Title), searchLower) &&
				!strings.Contains(strings.ToLower(issue.Body), searchLower) {
				continue
			}
		}

		// Status filter (project field)
		if filters.Status != "" {
			statusValue, ok := issue.Fields["Status"].(string)
			if !ok || !s.matchesFieldValue("status", filters.Status, statusValue) {
				continue
			}
		}

		// Priority filter (project field)
		if filters.Priority != "" {
			priorityValue, ok := issue.Fields["Priority"].(string)
			if !ok {
				continue
			}

			// Support comma-separated values
			priorityFilters := strings.Split(filters.Priority, ",")
			matched := false
			for _, pf := range priorityFilters {
				if s.matchesFieldValue("priority", strings.TrimSpace(pf), priorityValue) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		filtered = append(filtered, issue)
	}

	return filtered
}

// matchesFieldValue checks if a filter value matches the actual field value
func (s *SearchClient) matchesFieldValue(fieldName, filterValue, actualValue string) bool {
	// Direct match
	if strings.EqualFold(filterValue, actualValue) {
		return true
	}

	// Check config field mappings
	if field, ok := s.config.Fields[fieldName]; ok {
		// Check if filter value is a mapped key
		if mappedValue, ok := field.Values[strings.ToLower(filterValue)]; ok {
			return strings.EqualFold(mappedValue, actualValue)
		}
		// Check reverse mapping
		for key, value := range field.Values {
			if strings.EqualFold(value, filterValue) && strings.EqualFold(key, actualValue) {
				return true
			}
		}
	}

	return false
}
