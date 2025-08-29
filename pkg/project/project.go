package project

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Project represents a GitHub Project v2
type Project struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Owner  struct {
		Login string `json:"login"`
		Type  string `json:"__typename"`
	} `json:"owner"`
}

// Field represents a project field
type Field struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	DataType string      `json:"dataType"`
	Options  []FieldOption `json:"options,omitempty"`
}

// FieldOption represents an option for a single-select field
type FieldOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Client is a wrapper around GitHub API client
type Client struct {
	rest *api.RESTClient
	gql  *api.GraphQLClient
}

// NewClient creates a new project client
func NewClient() (*Client, error) {
	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}
	
	gqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL client: %w", err)
	}
	
	return &Client{
		rest: restClient,
		gql:  gqlClient,
	}, nil
}

// GetProject fetches a project by name or number
func (c *Client) GetProject(org string, projectName string, projectNumber int) (*Project, error) {
	query := `
		query($org: String!, $projectNumber: Int!) {
			organization(login: $org) {
				projectV2(number: $projectNumber) {
					id
					number
					title
					url
					owner {
						__typename
						... on Organization {
							login
						}
						... on User {
							login
						}
					}
				}
			}
		}`

	if projectNumber > 0 {
		variables := map[string]interface{}{
			"org":           org,
			"projectNumber": projectNumber,
		}

		var result struct {
			Organization struct {
				ProjectV2 Project `json:"projectV2"`
			} `json:"organization"`
		}

		err := c.graphQL(query, variables, &result)
		if err != nil {
			return nil, err
		}

		return &result.Organization.ProjectV2, nil
	}

	// If projectNumber is not provided, list all projects and find by name
	projects, err := c.ListProjects(org)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if strings.EqualFold(p.Title, projectName) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("project '%s' not found in organization '%s'", projectName, org)
}

// ListProjects lists all projects in an organization
func (c *Client) ListProjects(org string) ([]Project, error) {
	query := `
		query($org: String!, $cursor: String) {
			organization(login: $org) {
				projectsV2(first: 100, after: $cursor) {
					nodes {
						id
						number
						title
						url
						owner {
							__typename
							... on Organization {
								login
							}
							... on User {
								login
							}
						}
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		}`

	var allProjects []Project
	var cursor *string

	for {
		variables := map[string]interface{}{
			"org":    org,
			"cursor": cursor,
		}

		var result struct {
			Organization struct {
				ProjectsV2 struct {
					Nodes    []Project `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"projectsV2"`
			} `json:"organization"`
		}

		err := c.graphQL(query, variables, &result)
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, result.Organization.ProjectsV2.Nodes...)

		if !result.Organization.ProjectsV2.PageInfo.HasNextPage {
			break
		}
		cursor = &result.Organization.ProjectsV2.PageInfo.EndCursor
	}

	return allProjects, nil
}

// GetProjectFields fetches all fields for a project
func (c *Client) GetProjectFields(projectID string) ([]Field, error) {
	query := `
		query($projectId: ID!) {
			node(id: $projectId) {
				... on ProjectV2 {
					fields(first: 100) {
						nodes {
							... on ProjectV2Field {
								id
								name
								dataType
							}
							... on ProjectV2SingleSelectField {
								id
								name
								dataType
								options {
									id
									name
								}
							}
						}
					}
				}
			}
		}`

	variables := map[string]interface{}{
		"projectId": projectID,
	}

	var result struct {
		Node struct {
			Fields struct {
				Nodes []json.RawMessage `json:"nodes"`
			} `json:"fields"`
		} `json:"node"`
	}

	err := c.graphQL(query, variables, &result)
	if err != nil {
		return nil, err
	}

	var fields []Field
	for _, node := range result.Node.Fields.Nodes {
		var field Field
		if err := json.Unmarshal(node, &field); err != nil {
			continue
		}
		fields = append(fields, field)
	}

	return fields, nil
}

// GetRepoProjects fetches all projects associated with a repository
func (c *Client) GetRepoProjects(owner, repo string) ([]Project, error) {
	query := `
		query($owner: String!, $repo: String!, $cursor: String) {
			repository(owner: $owner, name: $repo) {
				projectsV2(first: 100, after: $cursor) {
					nodes {
						id
						number
						title
						url
						owner {
							__typename
							... on Organization {
								login
							}
							... on User {
								login
							}
						}
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		}`

	var allProjects []Project
	var cursor *string

	for {
		variables := map[string]interface{}{
			"owner":  owner,
			"repo":   repo,
			"cursor": cursor,
		}

		var result struct {
			Repository struct {
				ProjectsV2 struct {
					Nodes    []Project `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"projectsV2"`
			} `json:"repository"`
		}

		err := c.graphQL(query, variables, &result)
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, result.Repository.ProjectsV2.Nodes...)

		if !result.Repository.ProjectsV2.PageInfo.HasNextPage {
			break
		}
		cursor = &result.Repository.ProjectsV2.PageInfo.EndCursor
	}

	return allProjects, nil
}

// GetProjectNodeID returns the node ID for a project
func (c *Client) GetProjectNodeID(org string, projectNumber int) (string, error) {
	proj, err := c.GetProject(org, "", projectNumber)
	if err != nil {
		return "", err
	}
	return proj.ID, nil
}

// GetFieldsWithOptions fetches fields with their options including IDs
func (c *Client) GetFieldsWithOptions(projectID string) ([]Field, error) {
	// This is the same as GetProjectFields but with a more explicit name
	return c.GetProjectFields(projectID)
}

// graphQL executes a GraphQL query
func (c *Client) graphQL(query string, variables map[string]interface{}, result interface{}) error {
	return c.gql.Do(query, variables, result)
}