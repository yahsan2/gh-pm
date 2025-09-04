package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
	createAssignee  string
	createMilestone string
	createProject   string
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
	urlBuilder *project.URLBuilder
}

func runCreate(cmd *cobra.Command, args []string) error {
	// For backward compatibility, if args are provided without --title flag, use them as title
	if len(args) > 0 && createTitle == "" {
		createTitle = strings.Join(args, " ")
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

	issueClient := issue.NewClient()

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

	// Create URL builder
	urlBuilder := project.NewURLBuilder(cfg, projectClient)

	// Create command executor
	command := &CreateCommand{
		config:     cfg,
		client:     projectClient,
		issueAPI:   issueClient,
		formatter:  formatter,
		urlBuilder: urlBuilder,
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

// CallGHIssueCreate calls gh issue create command and returns the created issue information
func CallGHIssueCreate(title, body, repo string, labels []string) (*issue.Issue, error) {
	// Build gh issue create command
	args := []string{"issue", "create"}

	// If title is empty, gh will use interactive mode
	if title == "" {
		// Interactive mode - capture output to get issue URL
		if body != "" {
			args = append(args, "--body", body)
		}
		if repo != "" {
			args = append(args, "--repo", repo)
		}
		for _, label := range labels {
			args = append(args, "--label", label)
		}

		// Execute gh command with interactive mode
		cmd := exec.Command("gh", args...)
		cmd.Stdin = os.Stdin // Connect stdin for interactive prompt

		// Use pipe to capture output while still showing it to the user
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to create issue: %w", err)
		}

		// Print the output to the user
		fmt.Print(string(output))

		// Parse the output to find the issue URL and extract the number
		lines := strings.Split(string(output), "\n")
		var issueNumber int
		for _, line := range lines {
			// Look for the issue URL in the output
			if strings.Contains(line, "github.com") && strings.Contains(line, "/issues/") {
				// Extract issue number from URL
				parts := strings.Split(line, "/issues/")
				if len(parts) == 2 {
					// Clean the number part and parse it
					numStr := strings.TrimSpace(parts[1])
					// Remove any trailing characters
					for i, ch := range numStr {
						if ch < '0' || ch > '9' {
							numStr = numStr[:i]
							break
						}
					}
					if num, err := strconv.Atoi(numStr); err == nil {
						issueNumber = num
						break
					}
				}
			}
		}

		if issueNumber == 0 {
			return nil, fmt.Errorf("could not extract issue number from output")
		}

		return issue.GetIssueDetails(issueNumber, repo)
	}

	// Non-interactive mode
	if title != "" {
		args = append(args, "--title", title)
	}

	if body != "" {
		args = append(args, "--body", body)
	}

	if repo != "" {
		args = append(args, "--repo", repo)
	}

	for _, label := range labels {
		args = append(args, "--label", label)
	}

	// Execute gh command
	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w\noutput: %s", err, string(output))
	}

	// Parse the output to find the issue URL and extract the number
	lines := strings.Split(string(output), "\n")
	var issueNumber int
	for _, line := range lines {
		// Look for the issue URL in the output
		if strings.Contains(line, "github.com") && strings.Contains(line, "/issues/") {
			// Extract issue number from URL
			parts := strings.Split(line, "/issues/")
			if len(parts) == 2 {
				// Clean the number part and parse it
				numStr := strings.TrimSpace(parts[1])
				// Remove any trailing characters
				for i, ch := range numStr {
					if ch < '0' || ch > '9' {
						numStr = numStr[:i]
						break
					}
				}
				if num, err := strconv.Atoi(numStr); err == nil {
					issueNumber = num
					break
				}
			}
		}
	}

	if issueNumber == 0 {
		return nil, fmt.Errorf("could not extract issue number from output: %s", string(output))
	}

	// Get full issue details
	return issue.GetIssueDetails(issueNumber, repo)
}

func (c *CreateCommand) Execute(title string) error {
	// Prepare issue data
	repo := c.selectRepository()
	labels := c.mergeLabels()

	// Call gh issue create command
	createdIssue, err := CallGHIssueCreate(title, createBody, repo, labels)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	// Set these for project field updates
	priority := c.selectPriority()
	status := c.selectStatus()

	// Add to project if configured
	var projectURL string
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
			// Cache the project ID for future use
			c.config.SetProjectID(projectID)
		}

		// Add issue to project
		itemID, databaseID, err := c.issueAPI.AddToProjectWithDatabaseID(createdIssue.ID, projectID)
		if err != nil {
			return fmt.Errorf("failed to add issue to project: %w", err)
		}

		// Build the project URL with the numeric database ID
		projectURL = c.urlBuilder.GetProjectItemURL(databaseID)

		// Try to use cached fields first
		var fields []project.Field
		if c.config.HasCachedFields() {
			// Convert cached fields to project.Field format
			cachedFields := c.config.GetAllFields()
			fields = make([]project.Field, 0, len(cachedFields))
			for _, cf := range cachedFields {
				field := project.Field{
					ID:       cf.ID,
					Name:     cf.Name,
					DataType: cf.DataType,
				}
				if cf.Options != nil {
					field.Options = make([]project.FieldOption, 0, len(cf.Options))
					for _, opt := range cf.Options {
						field.Options = append(field.Options, project.FieldOption{
							ID:   opt.ID,
							Name: opt.Name,
						})
					}
				}
				fields = append(fields, field)
			}
		} else {
			// Fallback to API call if no cache
			fields, err = c.client.GetFieldsWithOptions(projectID)
			if err != nil {
				return fmt.Errorf("failed to get project fields: %w", err)
			}
		}

		// Update Status field if configured
		if status != "" {
			if err := c.updateProjectField(projectID, itemID, "Status", status, fields); err != nil {
				// Log error but don't fail the whole operation
				fmt.Printf("Warning: failed to update status field: %v\n", err)
			}
		}

		// Update Priority field if configured
		if priority != "" {
			if err := c.updateProjectField(projectID, itemID, "Priority", priority, fields); err != nil {
				// Log error but don't fail the whole operation
				fmt.Printf("Warning: failed to update priority field: %v\n", err)
			}
		}
	}

	// Add project URL to issue if available
	if projectURL != "" {
		createdIssue.ProjectURL = projectURL
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
