package project

import (
	"fmt"
	"strings"
)

// GetUserProject fetches a project by name or number for a user
func (c *Client) GetUserProject(username string, projectName string, projectNumber int) (*Project, error) {
	if projectNumber > 0 {
		// Use number-based query for user projects
		query := `
			query($username: String!, $projectNumber: Int!) {
				user(login: $username) {
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

		variables := map[string]interface{}{
			"username":      username,
			"projectNumber": projectNumber,
		}

		var result struct {
			User struct {
				ProjectV2 Project `json:"projectV2"`
			} `json:"user"`
		}

		err := c.graphQL(query, variables, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to get user project: %w", err)
		}

		if result.User.ProjectV2.ID == "" {
			return nil, fmt.Errorf("project #%d not found for user %s", projectNumber, username)
		}

		return &result.User.ProjectV2, nil
	}

	// If projectNumber is not provided, list all user projects and find by name
	projects, err := c.ListUserProjects(username)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if strings.EqualFold(p.Title, projectName) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("project '%s' not found for user '%s'", projectName, username)
}

// ListUserProjects lists all projects for a user
func (c *Client) ListUserProjects(username string) ([]Project, error) {
	query := `
		query($username: String!, $cursor: String) {
			user(login: $username) {
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
			"username": username,
		}
		if cursor != nil {
			variables["cursor"] = *cursor
		}

		var result struct {
			User struct {
				ProjectsV2 struct {
					Nodes    []Project `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"projectsV2"`
			} `json:"user"`
		}

		err := c.graphQL(query, variables, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to list user projects: %w", err)
		}

		allProjects = append(allProjects, result.User.ProjectsV2.Nodes...)

		if !result.User.ProjectsV2.PageInfo.HasNextPage {
			break
		}
		cursor = &result.User.ProjectsV2.PageInfo.EndCursor
	}

	return allProjects, nil
}

// GetCurrentUserProject fetches a project for the current authenticated user
func (c *Client) GetCurrentUserProject(projectName string, projectNumber int) (*Project, error) {
	// Get current user
	username, err := c.GetCurrentUsername()
	if err != nil {
		return nil, err
	}

	return c.GetUserProject(username, projectName, projectNumber)
}

// GetCurrentUsername gets the current authenticated user's username
func (c *Client) GetCurrentUsername() (string, error) {
	query := `
		query {
			viewer {
				login
			}
		}`

	var result struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
	}

	err := c.graphQL(query, nil, &result)
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	return result.Viewer.Login, nil
}

// GetCurrentUser is an alias for GetCurrentUsername for compatibility
func (c *Client) GetCurrentUser() (string, error) {
	return c.GetCurrentUsername()
}
