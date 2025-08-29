package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/issue"
	"github.com/yahsan2/gh-pm/pkg/output"
	"github.com/yahsan2/gh-pm/pkg/project"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue with project metadata",
	Long: `Create a new GitHub issue and automatically add it to the configured project
with metadata such as priority and status.

This command will:
- Create an issue in the configured repository
- Add the issue to the specified GitHub Project
- Set priority, status, and other custom fields
- Apply default labels from configuration`,
	Example: `  # Create an issue with a title
  gh pm create --title "Fix login bug"
  
  # Create an issue with title and body
  gh pm create --title "Add new feature" --body "Description of the feature"
  
  # Create with specific priority and status
  gh pm create --title "Critical issue" --priority high --status in_progress
  
  # Create from a file (batch mode)
  gh pm create --from-file issues.yml
  
  # Create from a template
  gh pm create --template bug
  
  # Interactive mode
  gh pm create --interactive`,
	RunE: runCreate,
}

// Command flags
var (
	createTitle       string
	createBody        string
	createLabels      []string
	createPriority    string
	createStatus      string
	createRepo        string
	createFromFile    string
	createTemplate    string
	createInteractive bool
	createQuiet       bool
	
	// Pass-through flags for gh issue create compatibility
	createAssignee    string
	createMilestone   string
	createProject     string
)

func init() {
	rootCmd.AddCommand(createCmd)
	
	// Basic issue flags
	createCmd.Flags().StringVarP(&createTitle, "title", "t", "", "Issue title")
	createCmd.Flags().StringVarP(&createBody, "body", "b", "", "Issue body content")
	createCmd.Flags().StringSliceVarP(&createLabels, "labels", "l", []string{}, "Comma-separated labels")
	
	// Project metadata flags
	createCmd.Flags().StringVar(&createPriority, "priority", "", "Issue priority (overrides default)")
	createCmd.Flags().StringVar(&createStatus, "status", "", "Issue status (overrides default)")
	
	// Repository selection
	createCmd.Flags().StringVarP(&createRepo, "repo", "r", "", "Repository (owner/repo format)")
	
	// Advanced features
	createCmd.Flags().StringVar(&createFromFile, "from-file", "", "Create issues from YAML/JSON file")
	createCmd.Flags().StringVar(&createTemplate, "template", "", "Use issue template")
	createCmd.Flags().BoolVarP(&createInteractive, "interactive", "i", false, "Interactive mode")
	
	// Output control
	createCmd.Flags().BoolVarP(&createQuiet, "quiet", "q", false, "Only output issue URL")
	
	// gh issue create compatibility flags
	createCmd.Flags().StringVarP(&createAssignee, "assignee", "a", "", "Assign to user")
	createCmd.Flags().StringVarP(&createMilestone, "milestone", "m", "", "Add to milestone")
	createCmd.Flags().StringVar(&createProject, "project", "", "Add to project (number or title)")
}

type CreateCommand struct {
	config     *config.Config
	client     *project.Client
	issueAPI   *issue.Client
	formatter  *output.Formatter
}

func runCreate(cmd *cobra.Command, args []string) error {
	// For backward compatibility, if args are provided without --title flag, use them as title
	if len(args) > 0 && createTitle == "" {
		createTitle = strings.Join(args, " ")
	}
	
	// Validate basic requirements
	if createTitle == "" && createFromFile == "" && createTemplate == "" && !createInteractive {
		return fmt.Errorf("issue title is required (use --title, --from-file, --template, or --interactive)")
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
	
	// Create clients
	projectClient, err := project.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create project client: %w", err)
	}
	
	issueClient, err := issue.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create issue client: %w", err)
	}
	
	// Create output formatter
	formatType := output.FormatTable // Default
	if createQuiet {
		formatType = output.FormatQuiet
	} else if outputFormat == "json" {
		formatType = output.FormatJSON
	} else if outputFormat == "csv" {
		formatType = output.FormatCSV
	}
	
	formatter := output.NewFormatter(formatType)
	
	// Create command executor
	command := &CreateCommand{
		config:     cfg,
		client:     projectClient,
		issueAPI:   issueClient,
		formatter:  formatter,
	}
	
	// Execute based on mode
	if createFromFile != "" {
		return command.ExecuteBatch(createFromFile)
	}
	
	if createTemplate != "" {
		return command.ExecuteTemplate(createTemplate)
	}
	
	if createInteractive {
		return command.ExecuteInteractive()
	}
	
	// Execute single issue creation
	return command.Execute(createTitle)
}

func (c *CreateCommand) Execute(title string) error {
	// Prepare issue data
	issueData := &issue.IssueData{
		Title:      title,
		Body:       createBody,
		Labels:     c.mergeLabels(),
		Repository: c.selectRepository(),
		Priority:   c.selectPriority(),
		Status:     c.selectStatus(),
	}
	
	// Validate issue data
	if err := issueData.Validate(); err != nil {
		return fmt.Errorf("invalid issue data: %w", err)
	}
	
	// Create issue
	createdIssue, err := c.issueAPI.CreateIssue(issueData)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}
	
	// Add to project if configured
	if c.config.Project.Name != "" || c.config.Project.Number > 0 {
		projectID := c.config.GetProjectID()
		if projectID == "" {
			// Fetch project ID if not cached
			var proj *project.Project
			var err error
			
			// Check if it's an organization or user project
			if c.config.Project.Org != "" {
				proj, err = c.client.GetProject(
					c.config.Project.Org,
					c.config.Project.Name,
					c.config.Project.Number,
				)
			} else {
				// Try to get as user project
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
		
		// Add issue to project
		itemID, err := c.issueAPI.AddToProject(createdIssue.ID, projectID)
		if err != nil {
			return fmt.Errorf("failed to add issue to project: %w", err)
		}
		
		// Get project fields to map field names to IDs
		fields, err := c.client.GetFieldsWithOptions(projectID)
		if err != nil {
			return fmt.Errorf("failed to get project fields: %w", err)
		}
		
		// Update Status field if configured
		if issueData.Status != "" {
			if err := c.updateProjectField(projectID, itemID, "Status", issueData.Status, fields); err != nil {
				// Log error but don't fail the whole operation
				fmt.Printf("Warning: failed to update status field: %v\n", err)
			}
		}
		
		// Update Priority field if configured
		if issueData.Priority != "" {
			if err := c.updateProjectField(projectID, itemID, "Priority", issueData.Priority, fields); err != nil {
				// Log error but don't fail the whole operation
				fmt.Printf("Warning: failed to update priority field: %v\n", err)
			}
		}
	}
	
	// Format and display output
	return c.formatter.FormatIssue(createdIssue)
}

func (c *CreateCommand) ExecuteBatch(filepath string) error {
	// Implementation will be added in task 14
	return fmt.Errorf("batch processing not yet implemented")
}

func (c *CreateCommand) ExecuteTemplate(templateName string) error {
	// Implementation will be added in task 17
	return fmt.Errorf("template processing not yet implemented")
}

func (c *CreateCommand) ExecuteInteractive() error {
	// Implementation will be added in task 11
	return fmt.Errorf("interactive mode not yet implemented")
}

func (c *CreateCommand) mergeLabels() []string {
	labels := make([]string, 0)
	
	// Start with default labels from config
	if len(c.config.Defaults.Labels) > 0 {
		labels = append(labels, c.config.Defaults.Labels...)
	}
	
	// Add command-line labels
	if len(createLabels) > 0 {
		labels = append(labels, createLabels...)
	}
	
	// Remove duplicates
	labelMap := make(map[string]bool)
	uniqueLabels := make([]string, 0)
	for _, label := range labels {
		if !labelMap[label] {
			labelMap[label] = true
			uniqueLabels = append(uniqueLabels, label)
		}
	}
	
	return uniqueLabels
}

func (c *CreateCommand) selectRepository() string {
	// Use command-line flag if provided
	if createRepo != "" {
		return createRepo
	}
	
	// Use first repository from config
	if len(c.config.Repositories) > 0 {
		return c.config.Repositories[0]
	}
	
	return ""
}

func (c *CreateCommand) selectPriority() string {
	// Use command-line flag if provided
	if createPriority != "" {
		return createPriority
	}
	
	// Use default from config
	return c.config.Defaults.Priority
}

func (c *CreateCommand) selectStatus() string {
	// Use command-line flag if provided
	if createStatus != "" {
		return createStatus
	}
	
	// Use default from config
	return c.config.Defaults.Status
}

// updateProjectField updates a single project field value
func (c *CreateCommand) updateProjectField(projectID, itemID, fieldName, value string, fields []project.Field) error {
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

func (c *CreateCommand) validateFlags() error {
	// Validate priority if provided
	if createPriority != "" {
		if field, ok := c.config.Fields["priority"]; ok {
			if _, exists := field.Values[createPriority]; !exists {
				validValues := make([]string, 0, len(field.Values))
				for k := range field.Values {
					validValues = append(validValues, k)
				}
				return fmt.Errorf("invalid priority '%s'. Valid values: %s", 
					createPriority, strings.Join(validValues, ", "))
			}
		}
	}
	
	// Validate status if provided
	if createStatus != "" {
		if field, ok := c.config.Fields["status"]; ok {
			if _, exists := field.Values[createStatus]; !exists {
				validValues := make([]string, 0, len(field.Values))
				for k := range field.Values {
					validValues = append(validValues, k)
				}
				return fmt.Errorf("invalid status '%s'. Valid values: %s", 
					createStatus, strings.Join(validValues, ", "))
			}
		}
	}
	
	return nil
}