package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/issue"
	"github.com/yahsan2/gh-pm/pkg/output"
	"github.com/yahsan2/gh-pm/pkg/project"
)

var moveCmd = &cobra.Command{
	Use:   "move <issue_number>",
	Short: "Move an issue by updating its project fields",
	Long: `Move an issue within a project by updating fields such as status and priority.

This command allows you to quickly update project fields for an issue:
- Change status (e.g., "todo", "in_progress", "done")  
- Change priority (e.g., "low", "medium", "high", "critical")
- Update multiple fields in a single operation

The issue must already be added to the configured project.`,
	Example: `  # Change issue status to ready
  gh pm move 15 --status ready
  
  # Change priority to high
  gh pm move 123 --priority high
  
  # Update both status and priority
  gh pm move 42 --status in_progress --priority critical`,
	Args: cobra.ExactArgs(1),
	RunE: runMove,
}

// Command flags
var (
	moveStatus   string
	movePriority string
	moveRepo     string
	moveQuiet    bool
)

func init() {
	rootCmd.AddCommand(moveCmd)
	
	// Field update flags
	moveCmd.Flags().StringVar(&moveStatus, "status", "", "New status for the issue")
	moveCmd.Flags().StringVar(&movePriority, "priority", "", "New priority for the issue")
	
	// Repository selection
	moveCmd.Flags().StringVarP(&moveRepo, "repo", "r", "", "Repository (owner/repo format)")
	
	// Output control
	moveCmd.Flags().BoolVarP(&moveQuiet, "quiet", "q", false, "Only output essential information")
}

type MoveCommand struct {
	config      *config.Config
	projectClient *project.Client
	issueClient *issue.Client
	formatter   *output.Formatter
}

func runMove(cmd *cobra.Command, args []string) error {
	// Parse issue number
	issueNumber, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number '%s': must be a number", args[0])
	}
	
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nRun 'gh pm init' to create a configuration file", err)
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Check that at least one field update was specified
	if moveStatus == "" && movePriority == "" {
		return fmt.Errorf("no field updates specified. Use --status or --priority flags")
	}
	
	// Create clients
	projectClient, err := project.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create project client: %w", err)
	}
	
	issueClient := issue.NewClient()
	
	// Create output formatter
	formatType := output.FormatTable // Default
	if moveQuiet {
		formatType = output.FormatQuiet
	} else if outputFormat == "json" {
		formatType = output.FormatJSON
	} else if outputFormat == "csv" {
		formatType = output.FormatCSV
	}
	
	formatter := output.NewFormatter(formatType)
	
	// Create command executor
	command := &MoveCommand{
		config:        cfg,
		projectClient: projectClient,
		issueClient:   issueClient,
		formatter:     formatter,
	}
	
	// Execute the move operation
	return command.Execute(issueNumber)
}

func (c *MoveCommand) Execute(issueNumber int) error {
	// Select repository
	repo := c.selectRepository()
	
	// Get issue details
	currentIssue, err := issue.GetIssueDetails(issueNumber, repo)
	if err != nil {
		return fmt.Errorf("failed to get issue details: %w", err)
	}
	
	// Get project ID
	projectID := c.config.GetProjectID()
	if projectID == "" {
		// Fetch project ID if not cached
		var proj *project.Project
		var err error
		
		// Check if it's an organization or user project
		if c.config.Project.Org != "" {
			proj, err = c.projectClient.GetProject(
				c.config.Project.Org,
				c.config.Project.Name,
				c.config.Project.Number,
			)
		} else {
			// Try to get as user project
			proj, err = c.projectClient.GetCurrentUserProject(
				c.config.Project.Name,
				c.config.Project.Number,
			)
		}
		
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}
		projectID = proj.ID
		
		// Cache the project ID
		c.config.SetProjectID(projectID)
	}
	
	// Get project item for this issue
	projectItem, err := c.projectClient.GetProjectItemForIssue(projectID, currentIssue.ID)
	if err != nil {
		return fmt.Errorf("failed to find issue in project (make sure issue %d is added to the project): %w", issueNumber, err)
	}
	
	// Get project fields to map field names to IDs
	fields, err := c.projectClient.GetFieldsWithOptions(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project fields: %w", err)
	}
	
	// Track changes made
	var updatesApplied []string
	
	// Update Status field if specified
	if moveStatus != "" {
		if err := c.updateProjectField(projectID, projectItem.ID, "Status", moveStatus, fields); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
		updatesApplied = append(updatesApplied, fmt.Sprintf("Status → %s", moveStatus))
	}
	
	// Update Priority field if specified
	if movePriority != "" {
		if err := c.updateProjectField(projectID, projectItem.ID, "Priority", movePriority, fields); err != nil {
			return fmt.Errorf("failed to update priority: %w", err)
		}
		updatesApplied = append(updatesApplied, fmt.Sprintf("Priority → %s", movePriority))
	}
	
	// Prepare success output
	if !moveQuiet {
		fmt.Printf("✓ Updated issue #%d: %s\n", issueNumber, currentIssue.Title)
		for _, update := range updatesApplied {
			fmt.Printf("  • %s\n", update)
		}
		fmt.Printf("🔗 %s\n", currentIssue.URL)
	} else {
		fmt.Printf("Updated issue #%d\n", issueNumber)
	}
	
	return nil
}

func (c *MoveCommand) selectRepository() string {
	// Use command-line flag if provided
	if moveRepo != "" {
		return moveRepo
	}
	
	// Use first repository from config
	if len(c.config.Repositories) > 0 {
		return c.config.Repositories[0]
	}
	
	return ""
}

// updateProjectField updates a single project field value using existing logic from create.go
func (c *MoveCommand) updateProjectField(projectID, itemID, fieldName, value string, fields []project.Field) error {
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
		
		return c.issueClient.UpdateProjectItemField(projectID, itemID, targetField.ID, optionID)
	}
	
	// For other field types, we'd need different handling
	return fmt.Errorf("unsupported field type '%s' for field '%s'", targetField.DataType, fieldName)
}

func (c *MoveCommand) validateFlags() error {
	// Validate status if provided
	if moveStatus != "" {
		if field, ok := c.config.Fields["status"]; ok {
			if _, exists := field.Values[moveStatus]; !exists {
				validValues := make([]string, 0, len(field.Values))
				for k := range field.Values {
					validValues = append(validValues, k)
				}
				return fmt.Errorf("invalid status '%s'. Valid values: %v", moveStatus, validValues)
			}
		}
	}
	
	// Validate priority if provided
	if movePriority != "" {
		if field, ok := c.config.Fields["priority"]; ok {
			if _, exists := field.Values[movePriority]; !exists {
				validValues := make([]string, 0, len(field.Values))
				for k := range field.Values {
					validValues = append(validValues, k)
				}
				return fmt.Errorf("invalid priority '%s'. Valid values: %v", movePriority, validValues)
			}
		}
	}
	
	return nil
}