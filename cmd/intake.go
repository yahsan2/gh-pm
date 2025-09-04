package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yahsan2/gh-pm/pkg/args"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/filter"
	"github.com/yahsan2/gh-pm/pkg/issue"
	"github.com/yahsan2/gh-pm/pkg/project"
)

var intakeCmd = &cobra.Command{
	Use:   "intake",
	Short: "List and add issues not in project",
	Long: `List issues that are not in the project and optionally add them.

This command will:
- List issues based on filters (similar to gh issue list)
- Filter out issues already in the project
- Optionally add remaining issues to the project`,
	Example: `  # List all open issues not in project
  gh pm intake

  # Filter by label
  gh pm intake --label bug --label enhancement

  # Filter by assignee
  gh pm intake --assignee @me

  # Search with query
  gh pm intake --search "authentication"

  # Search with GitHub Projects date expressions
  gh pm intake --search "created:@today-1w"
  gh pm intake --search "updated:>@today-30d"

  # Preview what would be added without making changes
  gh pm intake --dry-run

  # Add issues and set project fields
  gh pm intake --apply "status:backlog,priority:p2"`,
	RunE: runIntake,
}

func init() {
	// Add common gh issue list compatible flags
	args.AddCommonFlags(intakeCmd, nil)

	// Deprecated but kept for compatibility
	intakeCmd.Flags().String("query", "", "GitHub search query (deprecated, use --search)")
	_ = intakeCmd.Flags().MarkDeprecated("query", "use --search instead")

	// intake specific flags
	intakeCmd.Flags().Bool("dry-run", false, "Show what would be added without making changes")
	intakeCmd.Flags().StringSlice("apply", []string{}, "Fields to apply when adding (e.g., 'status:backlog', 'priority:p2')")

	rootCmd.AddCommand(intakeCmd)
}

type IntakeCommand struct {
	config    *config.Config
	client    *project.Client
	issueAPI  *issue.Client
	searchAPI *issue.SearchClient
}

func runIntake(cmd *cobra.Command, cmdArgs []string) error {
	// Parse common flags using shared argument parser
	filters, err := args.ParseCommonFlags(cmd, nil)
	if err != nil {
		return fmt.Errorf("failed to parse common flags: %w", err)
	}

	// Handle deprecated query flag for backward compatibility
	query, _ := cmd.Flags().GetString("query")
	if filters.Search == "" && query != "" {
		filters.Search = query
	}

	// Parse intake-specific flags
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	applyFlags, _ := cmd.Flags().GetStringSlice("apply")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nRun 'gh pm init' to create a configuration file", err)
	}

	// Check if project is configured
	if cfg.Project.Name == "" && cfg.Project.Number == 0 {
		return fmt.Errorf("no project configured. Run 'gh pm init' to configure a project")
	}

	// Create clients
	projectClient, err := project.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create project client: %w", err)
	}

	issueClient := issue.NewClient()

	// Create search client
	searchClient, err := issue.NewSearchClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create search client: %w", err)
	}

	// Create command executor
	command := &IntakeCommand{
		config:    cfg,
		client:    projectClient,
		issueAPI:  issueClient,
		searchAPI: searchClient,
	}

	// Parse apply flags
	applyFields := make(map[string]string)
	for _, apply := range applyFlags {
		parts := strings.SplitN(apply, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid apply format: %s (expected 'field:value')", apply)
		}
		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		applyFields[field] = value
	}

	return command.ExecuteWithFilters(filters, dryRun, applyFields)
}

func (c *IntakeCommand) ExecuteWithFilters(filters *filter.IssueFilters, dryRun bool, applyFields map[string]string) error {
	// Search for issues using shared search client
	issues, err := c.searchAPI.SearchIssues(filters)
	if err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found matching the filters")
		return nil
	}

	fmt.Printf("Found %d issues from search\n", len(issues))

	// Continue with existing logic
	return c.processIssues(issues, dryRun, applyFields)
}

func (c *IntakeCommand) processIssues(issues []filter.GitHubIssue, dryRun bool, applyFields map[string]string) error {
	// Get project ID
	var projectID string
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
			// Cache the project ID for future use
			c.config.SetProjectID(projectID)
		}
	}

	// Get issues already in project using shared search client
	existingIssues, err := c.searchAPI.GetProjectIssues(projectID)
	if err != nil {
		return fmt.Errorf("failed to get existing project issues: %w", err)
	}

	// Create a map for quick lookup
	existingMap := make(map[int]bool)
	for _, issue := range existingIssues {
		existingMap[issue.Number] = true
	}

	// Filter out issues already in project
	var issuesToAdd []filter.GitHubIssue
	for _, issue := range issues {
		if !existingMap[issue.Number] {
			issuesToAdd = append(issuesToAdd, issue)
		}
	}

	if len(issuesToAdd) == 0 {
		fmt.Println("All matching issues are already in the project")
		return nil
	}

	fmt.Printf("\nFound %d issues not in project:\n", len(issuesToAdd))
	for _, issue := range issuesToAdd {
		fmt.Printf("  #%d: %s\n", issue.Number, issue.Title)
	}

	if dryRun {
		fmt.Println("\n[DRY RUN] Would add these issues to the project")
		if len(applyFields) > 0 {
			fmt.Println("Would apply the following fields:")
			for field, value := range applyFields {
				fmt.Printf("  - %s: %s\n", field, value)
			}
		}
		return nil
	}

	// Confirm before adding
	fmt.Printf("\nAdd %d issues to project? (y/N): ", len(issuesToAdd))
	var response string
	_, _ = fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Cancelled")
		return nil
	}

	// Get project fields if we need to apply field values
	var fields []project.Field
	if len(applyFields) > 0 {
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
	}

	// Add issues to project
	successCount := 0
	for _, issue := range issuesToAdd {
		fmt.Printf("Adding issue #%d to project... ", issue.Number)

		itemID, _, err := c.issueAPI.AddToProjectWithDatabaseID(issue.ID, projectID)
		if err != nil {
			fmt.Printf("failed: %v\n", err)
			continue
		}

		// Apply field values if specified
		if len(applyFields) > 0 && itemID != "" {
			for fieldKey, fieldValue := range applyFields {
				var fieldName string
				switch fieldKey {
				case "status":
					fieldName = "Status"
				case "priority":
					fieldName = "Priority"
				default:
					fieldName = fieldKey
				}

				if err := c.updateProjectField(projectID, itemID, fieldName, fieldValue, fields); err != nil {
					fmt.Printf("\n  Warning: failed to set %s: %v", fieldName, err)
				}
			}
		}

		fmt.Println("✓")
		successCount++
	}

	fmt.Printf("\nSuccessfully added %d/%d issues to project\n", successCount, len(issuesToAdd))
	return nil
}

// Search and project issue retrieval logic is now handled by shared SearchClient

func (c *IntakeCommand) updateProjectField(projectID, itemID, fieldName, value string, fields []project.Field) error {
	// Find the field by name
	var targetField *project.Field
	for _, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
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

		// Check config field mappings
		configKey := strings.ToLower(fieldName)
		if configField, ok := c.config.Fields[configKey]; ok {
			if mappedValue, ok := configField.Values[value]; ok {
				// Find option with the mapped value
				for _, option := range targetField.Options {
					if option.Name == mappedValue {
						optionID = option.ID
						break
					}
				}
			}
		}

		// Direct match as fallback
		if optionID == "" {
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

	return fmt.Errorf("field type '%s' not supported", targetField.DataType)
}
