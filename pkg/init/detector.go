package init

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/yahsan2/gh-pm/pkg/project"
)

// ProjectDetector handles project detection and listing
type ProjectDetector struct {
	client *project.Client
}

// NewProjectDetector creates a new ProjectDetector instance
func NewProjectDetector() (*ProjectDetector, error) {
	client, err := project.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create project client: %w", err)
	}
	
	return &ProjectDetector{
		client: client,
	}, nil
}

// DetectCurrentRepo detects the current repository using GitHub CLI
func (d *ProjectDetector) DetectCurrentRepo() (org, repo string, err error) {
	// Use gh CLI to get current repository information
	r, err := repository.Current()
	if err != nil {
		return "", "", fmt.Errorf("failed to detect current repository: %w", err)
	}
	
	return r.Owner, r.Name, nil
}

// ListRepoProjects lists all projects associated with a repository
func (d *ProjectDetector) ListRepoProjects(org, repo string) ([]project.Project, error) {
	projects, err := d.client.GetRepoProjects(org, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to list repository projects: %w", err)
	}
	
	return projects, nil
}

// ListOrgProjects lists all projects in an organization
func (d *ProjectDetector) ListOrgProjects(org string) ([]project.Project, error) {
	projects, err := d.client.ListProjects(org)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization projects: %w", err)
	}
	
	return projects, nil
}