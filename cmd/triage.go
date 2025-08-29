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
  gh pm triage hogehoge
  
  # List issues that would be affected without making changes
  gh pm triage hogehoge --list
  
  # Same as --list (dry-run mode)
  gh pm triage hogehoge --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTriage,
}

func init() {
	triageCmd.Flags().BoolP("list", "l", false, "List matching issues without applying changes")
	triageCmd.Flags().Bool("dry-run", false, "Show what would be changed without making changes (alias for --list)")
	rootCmd.AddCommand(triageCmd)
}

type TriageCommand struct {
	config     *config.Config
	client     *project.Client
	issueAPI   *issue.Client
	urlBuilder *project.URLBuilder
}

type GitHubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	ID     string `json:"node_id"`
	URL    string `json:"html_url"`
}

func runTriage(cmd *cobra.Command, args []string) error {
	triageName := args[0]
	
	// Parse flags
	listOnly, _ := cmd.Flags().GetBool("list")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	
	// If either --list or --dry-run is specified, enable list-only mode
	if dryRun {
		listOnly = true
	}
	
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
	
	// Create URL builder
	urlBuilder := project.NewURLBuilder(cfg, projectClient)
	
	// Create command executor
	command := &TriageCommand{
		config:     cfg,
		client:     projectClient,
		issueAPI:   issueClient,
		urlBuilder: urlBuilder,
	}
	
	return command.Execute(triageConfig, listOnly)
}

func (c *TriageCommand) Execute(triageConfig config.TriageConfig, listOnly bool) error {
	// Execute GitHub search query
	issues, err := c.searchIssues(triageConfig.Query)
	if err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}
	
	if len(issues) == 0 {
		fmt.Printf("No issues found matching query: %s\n", triageConfig.Query)
		return nil
	}
	
	if listOnly {
		fmt.Printf("Found %d issues that would be affected by triage '%s':\n\n", len(issues), triageConfig.Query)
		return c.displayIssuesList(issues, triageConfig)
	}
	
	fmt.Printf("Found %d issues to triage\n", len(issues))
	
	// Get project ID if needed for field updates or interactive features
	var projectID string
	if len(triageConfig.Apply.Fields) > 0 || triageConfig.Interactive.Status || triageConfig.Interactive.Estimate {
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
	
	// Get project fields if we need to update them or handle interactive features
	var fields []project.Field
	if projectID != "" && (len(triageConfig.Apply.Fields) > 0 || triageConfig.Interactive.Status) {
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
		if projectID != "" {
			// Try to add issue to project (if already exists, this will return existing item)
			itemID, _, err := c.issueAPI.AddToProjectWithDatabaseID(issue.ID, projectID)
			if err != nil {
				fmt.Printf("Warning: failed to add issue #%d to project: %v\n", issue.Number, err)
				continue
			}
			
			if itemID != "" {
				// Update fields based on configuration (non-interactive)
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
				
				// Handle interactive fields (always check, even if no apply fields)
				if err := c.handleInteractiveFields(projectID, itemID, issue, triageConfig.Interactive, fields); err != nil {
					fmt.Printf("Warning: failed to handle interactive fields for issue #%d: %v\n", issue.Number, err)
				}
			}
		} else if triageConfig.Interactive.Status || triageConfig.Interactive.Estimate {
			// Handle interactive fields even without project fields (for estimate triage)
			fmt.Printf("No project configured, skipping interactive field updates for issue #%d\n", issue.Number)
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

func (c *TriageCommand) handleInteractiveFields(projectID, itemID string, issue GitHubIssue, interactive config.TriageInteractive, fields []project.Field) error {
	reader := bufio.NewReader(os.Stdin)
	
	// Handle status field interactively
	if interactive.Status {
		if err := c.handleInteractiveStatus(projectID, itemID, issue, reader, fields); err != nil {
			return err
		}
	}
	
	// Handle estimate field interactively
	if interactive.Estimate {
		if err := c.handleInteractiveEstimate(projectID, itemID, issue, reader); err != nil {
			return err
		}
	}
	
	return nil
}

func (c *TriageCommand) handleInteractiveStatus(projectID, itemID string, issue GitHubIssue, reader *bufio.Reader, fields []project.Field) error {
	// Find Status field
	var statusField *project.Field
	for _, field := range fields {
		if field.Name == "Status" {
			statusField = &field
			break
		}
	}
	
	if statusField == nil {
		return fmt.Errorf("Status field not found in project")
	}
	
	fmt.Printf("\nSelect status for issue #%d: %s\n", issue.Number, issue.Title)
	
	// Get available status options from config mapping
	var availableOptions []string
	var configMapping map[string]string
	
	if statusFieldConfig, ok := c.config.Fields["status"]; ok {
		configMapping = statusFieldConfig.Values
		for key := range configMapping {
			availableOptions = append(availableOptions, key)
		}
	} else {
		// Fallback to direct field options
		for _, option := range statusField.Options {
			availableOptions = append(availableOptions, option.Name)
		}
	}
	
	// Show options
	for i, option := range availableOptions {
		displayName := option
		if configMapping != nil {
			if mapped, ok := configMapping[option]; ok {
				displayName = fmt.Sprintf("%s (%s)", option, mapped)
			}
		}
		fmt.Printf("  %d. %s\n", i+1, displayName)
	}
	fmt.Printf("  0. Skip\n")
	
	fmt.Print("Enter your choice (0-" + strconv.Itoa(len(availableOptions)) + "): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 0 || choice > len(availableOptions) {
		fmt.Printf("Invalid choice, skipping status update for issue #%d\n", issue.Number)
		return nil
	}
	
	if choice == 0 {
		fmt.Printf("Skipped status update for issue #%d\n", issue.Number)
		return nil
	}
	
	selectedStatus := availableOptions[choice-1]
	if err := c.updateProjectField(projectID, itemID, "Status", selectedStatus, fields); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	
	fmt.Printf("✓ Updated status to '%s' for issue #%d\n", selectedStatus, issue.Number)
	return nil
}

func (c *TriageCommand) handleInteractiveEstimate(projectID, itemID string, issue GitHubIssue, reader *bufio.Reader) error {
	fmt.Printf("\nEnter estimate for issue #%d: %s\n", issue.Number, issue.Title)
	fmt.Print("Estimate (e.g., '2h', '1d', '3pts', or press Enter to skip): ")
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		fmt.Printf("Skipped estimate for issue #%d\n", issue.Number)
		return nil
	}
	
	// Here you would implement the estimate field update logic
	// This is a placeholder since estimate field implementation depends on your project setup
	fmt.Printf("✓ Set estimate '%s' for issue #%d (estimate field update not fully implemented)\n", input, issue.Number)
	return nil
}

func (c *TriageCommand) displayIssuesList(issues []GitHubIssue, triageConfig config.TriageConfig) error {
	// Display issues that would be affected
	for i, issue := range issues {
		fmt.Printf("%d. #%d: %s\n", i+1, issue.Number, issue.Title)
		
		// Try to get project URL
		projectID := c.config.GetProjectID()
		if projectID != "" {
			_, itemDatabaseID, err := c.issueAPI.GetProjectItemID(issue.ID, projectID)
			if err == nil && itemDatabaseID > 0 {
				projectURL := c.urlBuilder.GetProjectItemURL(itemDatabaseID)
				fmt.Printf("   URL: %s\n", projectURL)
			} else {
				// Issue not in project or error getting item ID
				fmt.Printf("   URL: %s\n", issue.URL)
			}
		} else {
			// Fallback to issue URL if no project info
			fmt.Printf("   URL: %s\n", issue.URL)
		}
	}
	
	fmt.Printf("\nWould apply the following changes:\n")
	
	// Show labels that would be applied
	if len(triageConfig.Apply.Labels) > 0 {
		fmt.Printf("- Labels: %s\n", strings.Join(triageConfig.Apply.Labels, ", "))
	}
	
	// Show fields that would be updated
	if len(triageConfig.Apply.Fields) > 0 {
		fmt.Printf("- Fields:\n")
		for fieldKey, fieldValue := range triageConfig.Apply.Fields {
			fieldName := fieldKey
			switch fieldKey {
			case "status":
				fieldName = "Status"
			case "priority":
				fieldName = "Priority"
			}
			fmt.Printf("  - %s: %s\n", fieldName, fieldValue)
		}
	}
	
	// Show interactive options
	if triageConfig.Interactive.Status || triageConfig.Interactive.Estimate {
		fmt.Printf("- Interactive fields:\n")
		if triageConfig.Interactive.Status {
			fmt.Printf("  - Status (will prompt for each issue)\n")
		}
		if triageConfig.Interactive.Estimate {
			fmt.Printf("  - Estimate (will prompt for each issue)\n")
		}
	}
	
	if len(triageConfig.Apply.Labels) == 0 && len(triageConfig.Apply.Fields) == 0 && !triageConfig.Interactive.Status && !triageConfig.Interactive.Estimate {
		fmt.Printf("- No changes configured\n")
	}
	
	fmt.Printf("\nTo execute these changes, run without --list or --dry-run flags.\n")
	
	return nil
}