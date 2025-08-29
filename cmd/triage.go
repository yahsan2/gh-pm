package cmd

import (
	"fmt"
	"strings"
	"os/exec"
	"encoding/json"
	"bufio"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/project"
	"github.com/yahsan2/gh-pm/pkg/issue"
)

var triageCmd = &cobra.Command{
	Use:   "triage [triage-name]",
	Short: "Execute a triage configuration to update issues based on query",
	Long: `Execute a predefined triage configuration from .gh-pm.yml.
	
This command will:
- Execute the GitHub search query defined in the triage configuration
- Apply labels, status, and priority updates to matching issues
- Update project fields for issues that are part of the configured project`,
	Example: `  # Run the hogehoge triage configuration
  gh pm triage hogehoge`,
	Args: cobra.ExactArgs(1),
	RunE: runTriage,
}

func init() {
	rootCmd.AddCommand(triageCmd)
}

type TriageCommand struct {
	config     *config.Config
	client     *project.Client
	issueAPI   *issue.Client
}

type GitHubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	ID     string `json:"node_id"`
	URL    string `json:"html_url"`
}

func runTriage(cmd *cobra.Command, args []string) error {
	triageName := args[0]
	
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nRun 'gh pm init' to create a configuration file", err)
	}
	
	// Get triage configuration
	triageConfig, exists := cfg.Triage[triageName]
	if !exists {
		return fmt.Errorf("triage configuration '%s' not found in .gh-pm.yml", triageName)
	}
	
	// Create clients
	projectClient, err := project.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create project client: %w", err)
	}
	
	issueClient, err := issue.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create issue client: %w", err)
	}
	
	// Create command executor
	command := &TriageCommand{
		config:   cfg,
		client:   projectClient,
		issueAPI: issueClient,
	}
	
	return command.Execute(triageConfig)
}

func (c *TriageCommand) Execute(triageConfig config.TriageConfig) error {
	// Execute GitHub search query
	issues, err := c.searchIssues(triageConfig.Query)
	if err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}
	
	if len(issues) == 0 {
		fmt.Printf("No issues found matching query: %s\n", triageConfig.Query)
		return nil
	}
	
	fmt.Printf("Found %d issues to triage\n", len(issues))
	
	// Get project ID if needed for field updates
	var projectID string
	if len(triageConfig.Apply.Fields) > 0 {
		if c.config.Project.Name != "" || c.config.Project.Number > 0 {
			projectID = c.config.GetProjectID()
			if projectID == "" {
				// Fetch project ID if not cached
				var proj *project.Project
				var err error
				
				if c.config.Project.Org != "" {
					proj, err = c.client.GetProject(
						c.config.Project.Org,
						c.config.Project.Name,
						c.config.Project.Number,
					)
				} else {
					proj, err = c.client.GetCurrentUserProject(
						c.config.Project.Name,
						c.config.Project.Number,
					)
				}
				
				if err != nil {
					return fmt.Errorf("failed to get project: %w", err)
				}
				projectID = proj.ID
			}
		}
	}
	
	// Get project fields if we need to update them
	var fields []project.Field
	if projectID != "" && len(triageConfig.Apply.Fields) > 0 {
		fields, err = c.client.GetFieldsWithOptions(projectID)
		if err != nil {
			return fmt.Errorf("failed to get project fields: %w", err)
		}
	}
	
	// Apply changes to each issue
	for _, issue := range issues {
		fmt.Printf("Processing issue #%d: %s\n", issue.Number, issue.Title)
		
		// Apply labels
		if len(triageConfig.Apply.Labels) > 0 {
			if err := c.applyLabels(issue.Number, triageConfig.Apply.Labels); err != nil {
				fmt.Printf("Warning: failed to apply labels to issue #%d: %v\n", issue.Number, err)
			}
		}
		
		// Apply project field updates if issue is in project
		if projectID != "" && len(triageConfig.Apply.Fields) > 0 {
			// Try to add issue to project (if already exists, this will return existing item)
			itemID, _, err := c.issueAPI.AddToProjectWithDatabaseID(issue.ID, projectID)
			if err != nil {
				fmt.Printf("Warning: failed to add issue #%d to project: %v\n", issue.Number, err)
				continue
			}
			
			if itemID != "" {
				// Update fields based on configuration
				for fieldKey, fieldValue := range triageConfig.Apply.Fields {
					var fieldName string
					switch fieldKey {
					case "status":
						fieldName = "Status"
					case "priority":
						fieldName = "Priority"
					default:
						fieldName = fieldKey // Use as-is for other fields
					}
					
					if err := c.updateProjectField(projectID, itemID, fieldName, fieldValue, fields); err != nil {
						fmt.Printf("Warning: failed to update %s field for issue #%d: %v\n", fieldName, issue.Number, err)
					}
				}
			}
		}
	}
	
	fmt.Printf("Triage completed for %d issues\n", len(issues))
	return nil
}

func (c *TriageCommand) searchIssues(query string) ([]GitHubIssue, error) {
	// For now, use gh issue list to get issues and filter locally
	// This works around the search API permission issues
	var repo string
	if len(c.config.Repositories) > 0 {
		repo = c.config.Repositories[0]
	}
	
	fmt.Printf("Fetching issues from repository: %s\n", repo)
	
	// Get all open issues with labels
	args := []string{"issue", "list", "--state=open", "--json", "number,title,id,url,labels"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	
	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %w, output: %s", err, string(output))
	}
	
	// Parse JSON output
	var allIssues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		ID     string `json:"id"`
		URL    string `json:"url"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	
	if err := json.Unmarshal(output, &allIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}
	
	// Filter based on query (basic implementation for -label:pm-tracked)
	var filteredIssues []GitHubIssue
	for _, issue := range allIssues {
		// Check if query excludes pm-tracked label
		if strings.Contains(query, "-label:pm-tracked") {
			hasLabel := false
			for _, label := range issue.Labels {
				if label.Name == "pm-tracked" {
					hasLabel = true
					break
				}
			}
			// Skip if issue has pm-tracked label
			if hasLabel {
				continue
			}
		}
		
		filteredIssues = append(filteredIssues, GitHubIssue{
			Number: issue.Number,
			Title:  issue.Title,
			ID:     issue.ID,
			URL:    issue.URL,
		})
	}
	
	fmt.Printf("Found %d open issues, %d match query criteria\n", len(allIssues), len(filteredIssues))
	return filteredIssues, nil
}

func (c *TriageCommand) applyLabels(issueNumber int, labels []string) error {
	// Get current repository from config
	var repo string
	if len(c.config.Repositories) > 0 {
		repo = c.config.Repositories[0]
	}
	
	// Build gh command to add labels
	args := []string{"issue", "edit", fmt.Sprintf("%d", issueNumber), "--add-label", strings.Join(labels, ",")}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	
	cmd := exec.Command("gh", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply labels: %w", err)
	}
	
	return nil
}

func (c *TriageCommand) updateProjectField(projectID, itemID, fieldName, value string, fields []project.Field) error {
	// Find the field by name
	var targetField *project.Field
	for _, field := range fields {
		if field.Name == fieldName {
			targetField = &field
			break
		}
	}
	
	if targetField == nil {
		return fmt.Errorf("field '%s' not found in project", fieldName)
	}
	
	// For single select fields, find the option ID
	if targetField.DataType == "SINGLE_SELECT" {
		var optionID string
		// Look for matching option based on config mapping
		if fieldName == "Status" {
			if statusField, ok := c.config.Fields["status"]; ok {
				// Use the configured mapping
				if mappedValue, ok := statusField.Values[value]; ok {
					// Find option with the mapped value
					for _, option := range targetField.Options {
						if option.Name == mappedValue {
							optionID = option.ID
							break
						}
					}
				}
			}
		} else if fieldName == "Priority" {
			if priorityField, ok := c.config.Fields["priority"]; ok {
				// Use the configured mapping
				if mappedValue, ok := priorityField.Values[value]; ok {
					// Find option with the mapped value
					for _, option := range targetField.Options {
						if option.Name == mappedValue {
							optionID = option.ID
							break
						}
					}
				}
			}
		} else {
			// Direct match
			for _, option := range targetField.Options {
				if option.Name == value {
					optionID = option.ID
					break
				}
			}
		}
		
		if optionID == "" {
			return fmt.Errorf("option '%s' not found for field '%s'", value, fieldName)
		}
		
		return c.issueAPI.UpdateProjectItemField(projectID, itemID, targetField.ID, optionID)
	}
	
	// For other field types, we'd need different handling
	return fmt.Errorf("unsupported field type '%s' for field '%s'", targetField.DataType, fieldName)
}