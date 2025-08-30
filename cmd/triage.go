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
	Long: `Execute a predefined triage configuration from .gh-pm.yml or run ad-hoc triage.
	
This command will:
- Execute the GitHub search query defined in the triage configuration or provided via --query
- Apply labels, status, and priority updates to matching issues
- Update project fields for issues that are part of the configured project`,
	Example: `  # Run the hogehoge triage configuration
  gh pm triage hogehoge
  
  # List issues that would be affected without making changes
  gh pm triage hogehoge --list
  
  # Same as --list (dry-run mode)
  gh pm triage hogehoge --dry-run
  
  # Ad-hoc triage with query and apply
  gh pm triage --query="status:backlog -has:estimate" --apply="status:in_progress"
  
  # Ad-hoc triage with interactive mode for specific fields
  gh pm triage --query="status:backlog" --interactive="status,estimate"
  gh pm triage --query="-has:priority" --interactive="priority"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTriage,
}

func init() {
	triageCmd.Flags().BoolP("list", "l", false, "List matching issues without applying changes")
	triageCmd.Flags().Bool("dry-run", false, "Show what would be changed without making changes (alias for --list)")
	triageCmd.Flags().String("query", "", "Query to filter issues (required when not using a named configuration)")
	triageCmd.Flags().StringSlice("apply", []string{}, "Fields to apply (e.g., 'status:in-progress', 'label:bug')")
	triageCmd.Flags().StringSlice("interactive", []string{}, "Fields to prompt for interactively (e.g., 'status', 'estimate', 'priority')")
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

// IssueUpdate holds the updates to be applied to an issue
type IssueUpdate struct {
	Issue          GitHubIssue
	ItemID         string
	StatusChoice   *string           // nil means skip
	EstimateChoice *string           // nil means skip
	FieldChoices   map[string]string // field name -> selected value
}

func runTriage(cmd *cobra.Command, args []string) error {
	// Parse flags
	listOnly, _ := cmd.Flags().GetBool("list")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	queryFlag, _ := cmd.Flags().GetString("query")
	applyFlags, _ := cmd.Flags().GetStringSlice("apply")
	interactiveFields, _ := cmd.Flags().GetStringSlice("interactive")
	
	// If either --list or --dry-run is specified, enable list-only mode
	if dryRun {
		listOnly = true
	}
	
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nRun 'gh pm init' to create a configuration file", err)
	}
	
	var triageConfig config.TriageConfig
	
	// Check if using ad-hoc mode or named configuration
	if queryFlag != "" {
		// Ad-hoc mode: --query is required, --apply or --interactive is required
		if len(applyFlags) == 0 && len(interactiveFields) == 0 {
			return fmt.Errorf("--query requires either --apply or --interactive flag")
		}
		
		// Build triage config from flags
		triageConfig = config.TriageConfig{
			Query: queryFlag,
			Apply: config.TriageApply{
				Fields: make(map[string]string),
				Labels: []string{},
			},
			Interactive: config.TriageInteractive{},
			InteractiveFields: make(map[string]bool),
		}
		
		// Parse apply flags
		for _, apply := range applyFlags {
			parts := strings.SplitN(apply, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid apply format: %s (expected 'field:value')", apply)
			}
			field := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			if field == "label" {
				triageConfig.Apply.Labels = append(triageConfig.Apply.Labels, value)
			} else {
				triageConfig.Apply.Fields[field] = value
			}
		}
		
		// Set interactive fields
		for _, field := range interactiveFields {
			field = strings.ToLower(strings.TrimSpace(field))
			// For backward compatibility, handle status and estimate specially
			if field == "status" {
				triageConfig.Interactive.Status = true
			} else if field == "estimate" {
				triageConfig.Interactive.Estimate = true
			} else {
				// Store other fields in the new map
				triageConfig.InteractiveFields[field] = true
			}
		}
	} else if len(args) > 0 {
		// Named configuration mode
		triageName := args[0]
		var exists bool
		triageConfig, exists = cfg.Triage[triageName]
		if !exists {
			return fmt.Errorf("triage configuration '%s' not found in .gh-pm.yml", triageName)
		}
	} else {
		return fmt.Errorf("either provide a triage name or use --query with --apply/--interactive")
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
	
	// Display instruction if configured
	if triageConfig.Instruction != "" {
		// Use dim cyan for instruction
		fmt.Printf("\n\033[36m%s\033[0m\n\n", triageConfig.Instruction)
	}
	
	// Get project ID if needed for field updates or interactive features
	var projectID string
	if len(triageConfig.Apply.Fields) > 0 || triageConfig.Interactive.Status || triageConfig.Interactive.Estimate || len(triageConfig.InteractiveFields) > 0 {
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
	}
	
	// Get project fields if we need to update them or handle interactive features
	var fields []project.Field
	if projectID != "" && (len(triageConfig.Apply.Fields) > 0 || triageConfig.Interactive.Status || triageConfig.Interactive.Estimate || len(triageConfig.InteractiveFields) > 0) {
		// Try to use cached fields first
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
	
	// Validate interactive fields are supported
	if len(triageConfig.InteractiveFields) > 0 && len(fields) > 0 {
		unsupportedFields := []string{}
		for fieldName := range triageConfig.InteractiveFields {
			var fieldFound bool
			for _, field := range fields {
				if strings.EqualFold(field.Name, fieldName) {
					fieldFound = true
					// Check if field type is supported
					if field.DataType != "SINGLE_SELECT" && field.DataType != "TEXT" && field.DataType != "NUMBER" {
						unsupportedFields = append(unsupportedFields, fmt.Sprintf("%s (%s)", fieldName, field.DataType))
					}
					break
				}
			}
			if !fieldFound {
				unsupportedFields = append(unsupportedFields, fmt.Sprintf("%s (not found)", fieldName))
			}
		}
		
		if len(unsupportedFields) > 0 {
			fmt.Printf("\nWarning: The following fields cannot be used interactively:\n")
			for _, field := range unsupportedFields {
				fmt.Printf("  - %s\n", field)
			}
			fmt.Printf("\nCurrently supported field types: SINGLE_SELECT, TEXT, NUMBER\n\n")
		}
	}
	
	// Phase 1: Collect all interactive choices first
	updates := make([]IssueUpdate, 0, len(issues))
	
	hasInteractive := triageConfig.Interactive.Status || triageConfig.Interactive.Estimate || len(triageConfig.InteractiveFields) > 0
	if hasInteractive {
		fmt.Println("\n=== Interactive Selection Phase ===")
		reader := bufio.NewReader(os.Stdin)
		
		for _, issue := range issues {
			update := IssueUpdate{Issue: issue}
			
			// Get project item ID if needed
			if projectID != "" {
				itemID, _, err := c.issueAPI.AddToProjectWithDatabaseID(issue.ID, projectID)
				if err != nil {
					fmt.Printf("Warning: failed to add issue #%d to project: %v\n", issue.Number, err)
					continue
				}
				update.ItemID = itemID
			}
			
			// Collect interactive status choice
			if triageConfig.Interactive.Status && update.ItemID != "" {
				statusChoice := c.collectStatusChoice(issue, reader, fields)
				update.StatusChoice = statusChoice
			}
			
			// Collect interactive estimate choice
			if triageConfig.Interactive.Estimate {
				estimateChoice := c.collectEstimateChoice(issue, reader)
				update.EstimateChoice = estimateChoice
			}
			
			// Collect other interactive fields
			if len(triageConfig.InteractiveFields) > 0 && update.ItemID != "" {
				if update.FieldChoices == nil {
					update.FieldChoices = make(map[string]string)
				}
				for fieldName := range triageConfig.InteractiveFields {
					choice := c.collectFieldChoice(issue, reader, fieldName, fields)
					if choice != nil {
						update.FieldChoices[fieldName] = *choice
					}
				}
			}
			
			updates = append(updates, update)
		}
		
		fmt.Println("\n=== Applying Updates ===")
	} else {
		// No interactive fields, just prepare updates
		for _, issue := range issues {
			update := IssueUpdate{Issue: issue}
			
			if projectID != "" {
				itemID, _, err := c.issueAPI.AddToProjectWithDatabaseID(issue.ID, projectID)
				if err != nil {
					fmt.Printf("Warning: failed to add issue #%d to project: %v\n", issue.Number, err)
					continue
				}
				update.ItemID = itemID
			}
			
			updates = append(updates, update)
		}
	}
	
	// Phase 2: Apply all changes
	for _, update := range updates {
		fmt.Printf("Processing issue #%d: %s\n", update.Issue.Number, update.Issue.Title)
		
		// Apply labels
		if len(triageConfig.Apply.Labels) > 0 {
			if err := c.applyLabels(update.Issue.Number, triageConfig.Apply.Labels); err != nil {
				fmt.Printf("Warning: failed to apply labels to issue #%d: %v\n", update.Issue.Number, err)
			}
		}
		
		// Apply project field updates
		if projectID != "" && update.ItemID != "" {
			// Apply configuration fields
			for fieldKey, fieldValue := range triageConfig.Apply.Fields {
				var fieldName string
				switch fieldKey {
				case "status":
					fieldName = "Status"
				case "priority":
					fieldName = "Priority"
				default:
					fieldName = fieldKey
				}
				
				if err := c.updateProjectField(projectID, update.ItemID, fieldName, fieldValue, fields); err != nil {
					fmt.Printf("Warning: failed to update %s field for issue #%d: %v\n", fieldName, update.Issue.Number, err)
				}
			}
			
				// Apply interactive status choice
				if update.StatusChoice != nil {
					if err := c.updateProjectField(projectID, update.ItemID, "Status", *update.StatusChoice, fields); err != nil {
						fmt.Printf("Warning: failed to update status for issue #%d: %v\n", update.Issue.Number, err)
					} else {
						fmt.Printf("✓ Updated status to '%s' for issue #%d\n", *update.StatusChoice, update.Issue.Number)
					}
				}
				
				// Apply interactive estimate choice
				if update.EstimateChoice != nil {
					if err := c.updateEstimateField(projectID, update.ItemID, *update.EstimateChoice, fields); err != nil {
						fmt.Printf("Warning: failed to update estimate for issue #%d: %v\n", update.Issue.Number, err)
					} else {
						fmt.Printf("✓ Set estimate '%s' for issue #%d\n", *update.EstimateChoice, update.Issue.Number)
					}
				}
				
				// Apply other interactive field choices
				for fieldName, fieldValue := range update.FieldChoices {
					// Capitalize field name for consistency
					displayFieldName := strings.Title(fieldName)
					if err := c.updateProjectField(projectID, update.ItemID, displayFieldName, fieldValue, fields); err != nil {
						fmt.Printf("Warning: failed to update %s for issue #%d: %v\n", fieldName, update.Issue.Number, err)
					} else {
						fmt.Printf("✓ Updated %s to '%s' for issue #%d\n", fieldName, fieldValue, update.Issue.Number)
					}
				}
		}
	}
	
	fmt.Printf("Triage completed for %d issues\n", len(issues))
	return nil
}

func (c *TriageCommand) searchIssues(query string) ([]GitHubIssue, error) {
	// Parse query to extract field filters
	fieldFilters := make(map[string]string) // field name -> filter value
	fieldExcludes := make(map[string]bool) // field name -> true if should be empty/unset
	var labelExcludes []string
	
	// Get available field names from metadata
	availableFields := make(map[string]bool)
	if c.config.Metadata != nil && c.config.Metadata.Fields != nil {
		for _, field := range c.config.Metadata.Fields {
			availableFields[field.Name] = true
		}
	}
	
	parts := strings.Split(query, " ")
	for _, part := range parts {
		// Check for label exclusions
		if strings.HasPrefix(part, "-label:") {
			labelExcludes = append(labelExcludes, strings.TrimPrefix(part, "-label:"))
			continue
		}
		
		// Check for field exclusions (-has:fieldname)
		if strings.HasPrefix(part, "-has:") {
			fieldName := strings.TrimPrefix(part, "-has:")
			fieldFound := false
			
			// Try to find the field in metadata (case-insensitive)
			for availField := range availableFields {
				if strings.EqualFold(availField, fieldName) {
					fieldExcludes[availField] = true
					fieldFound = true
					break
				}
			}
			
			// If not found in metadata, check config field names
			if !fieldFound && c.config.Fields != nil {
				// Check if it's a config field name (like "status", "priority", "estimate")
				if configField, ok := c.config.Fields[strings.ToLower(fieldName)]; ok {
					// Use the actual field name from config
					actualFieldName := configField.Field
					if actualFieldName != "" {
						// Check if this field exists in metadata
						for availField := range availableFields {
							if strings.EqualFold(availField, actualFieldName) {
								fieldExcludes[availField] = true
								break
							}
						}
					}
				} else {
					// Special case for "estimate" which might be "Estimate" field
					for availField := range availableFields {
						if strings.EqualFold(availField, fieldName) {
							fieldExcludes[availField] = true
							break
						}
					}
				}
			}
			continue
		}
		
		// Check for field filters dynamically
		if strings.Contains(part, ":") {
			colonIdx := strings.Index(part, ":")
			fieldName := part[:colonIdx]
			fieldValue := part[colonIdx+1:]
			
			// Try to find field in metadata (case-insensitive) or config
			fieldFound := false
			
			// First check metadata fields (case-insensitive)
			for availField := range availableFields {
				if strings.EqualFold(availField, fieldName) {
					fieldFilters[availField] = fieldValue
					fieldFound = true
					break
				}
			}
			
			// If not found in metadata, check config field names
			if !fieldFound && c.config.Fields != nil {
				// Check if it's a config field name (like "status", "priority")
				if configField, ok := c.config.Fields[strings.ToLower(fieldName)]; ok {
					// Use the actual field name from config
					actualFieldName := configField.Field
					if actualFieldName != "" {
						fieldFilters[actualFieldName] = fieldValue
					}
				}
			}
		}
	}
	
	// If we have field filters/excludes and project metadata, use optimized GraphQL query
	if (len(fieldFilters) > 0 || len(fieldExcludes) > 0) && c.config.Metadata != nil && c.config.Metadata.Project.ID != "" {
		return c.searchIssuesWithProjectFields(fieldFilters, fieldExcludes, labelExcludes)
	}
	
	// Fallback to original implementation for label-only filters
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
	
	// Filter based on query (for label-only filtering)
	var filteredIssues []GitHubIssue
	for _, issue := range allIssues {
		// Check label exclusions
		skipItem := false
		for _, excludeLabel := range labelExcludes {
			for _, label := range issue.Labels {
				if label.Name == excludeLabel {
					skipItem = true
					break
				}
			}
			if skipItem {
				break
			}
		}
		if !skipItem {
			filteredIssues = append(filteredIssues, GitHubIssue{
				Number: issue.Number,
				Title:  issue.Title,
				ID:     issue.ID,
				URL:    issue.URL,
			})
		}
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
	// Find the field by name (case-insensitive)
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
		
		// First try to use metadata for dynamic field lookup
		if c.config.Metadata != nil && c.config.Metadata.Fields != nil {
			// Find field metadata dynamically (case-insensitive)
			for _, fieldMeta := range c.config.Metadata.Fields {
				if strings.EqualFold(fieldMeta.Name, fieldName) {
					// Try to find the option ID directly from metadata
					for _, opt := range fieldMeta.Options {
						if opt.Name == value {
							optionID = opt.ID
							break
						}
					}
					break
				}
			}
		}
		
		// If not found in metadata, fall back to config field mappings
		if optionID == "" {
			// Convert field name to config key (e.g., "Status" -> "status")
			configKey := strings.ToLower(fieldName)
			
			if configField, ok := c.config.Fields[configKey]; ok {
				// Use the configured mapping
				if mappedValue, ok := configField.Values[value]; ok {
					// Find option with the mapped value
					for _, option := range targetField.Options {
						if option.Name == mappedValue {
							optionID = option.ID
							break
						}
					}
				}
			} else {
				// Direct match as last resort
				for _, option := range targetField.Options {
					if option.Name == value {
						optionID = option.ID
						break
					}
				}
			}
		}
		
		if optionID == "" {
			return fmt.Errorf("option '%s' not found for field '%s'", value, fieldName)
		}
		
		return c.issueAPI.UpdateProjectItemField(projectID, itemID, targetField.ID, optionID)
	}
	
	// For TEXT fields
	if targetField.DataType == "TEXT" {
		gql := c.issueAPI.GetGraphQLClient()
		mutation := `
			mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $text: String!) {
				updateProjectV2ItemFieldValue(
					input: {
						projectId: $projectId,
						itemId: $itemId,
						fieldId: $fieldId,
						value: { text: $text }
					}
				) { projectV2Item { id } }
			}`
		variables := map[string]interface{}{
			"projectId": projectID,
			"itemId":    itemID,
			"fieldId":   targetField.ID,
			"text":      value,
		}
		var result map[string]interface{}
		if err := gql.Do(mutation, variables, &result); err != nil {
			return fmt.Errorf("failed to set %s text: %w", fieldName, err)
		}
		return nil
	}
	
	// For NUMBER fields
	if targetField.DataType == "NUMBER" {
		// Try to parse the value as a number
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			// If not a pure number, try to extract number from strings like "2h", "3pts"
			// This is a simple implementation - could be enhanced
			for i, ch := range value {
				if !('0' <= ch && ch <= '9' || ch == '.') {
					if i > 0 {
						num, err = strconv.ParseFloat(value[:i], 64)
						if err == nil {
							break
						}
					}
					return fmt.Errorf("invalid numeric value '%s' for field '%s'", value, fieldName)
				}
			}
		}
		
		gql := c.issueAPI.GetGraphQLClient()
		mutation := `
			mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $number: Float!) {
				updateProjectV2ItemFieldValue(
					input: {
						projectId: $projectId,
						itemId: $itemId,
						fieldId: $fieldId,
						value: { number: $number }
					}
				) { projectV2Item { id } }
			}`
		variables := map[string]interface{}{
			"projectId": projectID,
			"itemId":    itemID,
			"fieldId":   targetField.ID,
			"number":    num,
		}
		var result map[string]interface{}
		if err := gql.Do(mutation, variables, &result); err != nil {
			return fmt.Errorf("failed to set %s number: %w", fieldName, err)
		}
		return nil
	}
	
	// For other field types, we'd need different handling
	return fmt.Errorf("unsupported field type '%s' for field '%s'", targetField.DataType, fieldName)
}

// updateEstimateField updates the Estimate field (TEXT or NUMBER) on a project item
func (c *TriageCommand) updateEstimateField(projectID, itemID, estimate string, fields []project.Field) error {
    // Try to use the generic updateProjectField function first
    // It now supports TEXT and NUMBER fields dynamically
    
    // Try "Estimate" field name first
    err := c.updateProjectField(projectID, itemID, "Estimate", estimate, fields)
    if err == nil {
        return nil
    }
    
    // If not found, try lowercase "estimate" as fallback
    if strings.Contains(err.Error(), "not found") {
        // Try with lowercase
        return c.updateProjectField(projectID, itemID, "estimate", estimate, fields)
    }
    
    return err
}

func (c *TriageCommand) collectStatusChoice(issue GitHubIssue, reader *bufio.Reader, fields []project.Field) *string {
	// Find Status field
	var statusField *project.Field
	for _, field := range fields {
		if field.Name == "Status" {
			statusField = &field
			break
		}
	}
	
	if statusField == nil {
		fmt.Printf("Status field not found in project for issue #%d\n", issue.Number)
		return nil
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
		fmt.Printf("Failed to read input: %v\n", err)
		return nil
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
	return &selectedStatus
}

func (c *TriageCommand) collectEstimateChoice(issue GitHubIssue, reader *bufio.Reader) *string {
	fmt.Printf("\nEnter estimate for issue #%d: %s\n", issue.Number, issue.Title)
	fmt.Print("Estimate (e.g., '2h', '1d', '3pts', or press Enter to skip): ")
	
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read input: %v\n", err)
		return nil
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		fmt.Printf("Skipped estimate for issue #%d\n", issue.Number)
		return nil
	}
	
	return &input
}

func (c *TriageCommand) collectFieldChoice(issue GitHubIssue, reader *bufio.Reader, fieldName string, fields []project.Field) *string {
	// Find the target field
	var targetField *project.Field
	for _, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
			targetField = &field
			break
		}
	}
	
	if targetField == nil {
		fmt.Printf("Field '%s' not found in project for issue #%d\n", fieldName, issue.Number)
		return nil
	}
	
	fmt.Printf("\nSelect %s for issue #%d: %s\n", fieldName, issue.Number, issue.Title)
	
	// Handle different field types
	switch targetField.DataType {
	case "SINGLE_SELECT":
		// Get available options
		var availableOptions []string
		var configMapping map[string]string
		
		// Check if there's a config mapping for this field
		if fieldConfig, ok := c.config.Fields[strings.ToLower(fieldName)]; ok {
			configMapping = fieldConfig.Values
			for key := range configMapping {
				availableOptions = append(availableOptions, key)
			}
		} else {
			// Use field options directly
			for _, option := range targetField.Options {
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
			fmt.Printf("Failed to read input: %v\n", err)
			return nil
		}
		
		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 0 || choice > len(availableOptions) {
			fmt.Printf("Invalid choice, skipping %s update for issue #%d\n", fieldName, issue.Number)
			return nil
		}
		
		if choice == 0 {
			fmt.Printf("Skipped %s update for issue #%d\n", fieldName, issue.Number)
			return nil
		}
		
		selectedValue := availableOptions[choice-1]
		return &selectedValue
		
	case "TEXT", "NUMBER":
		// For text or number fields, accept free-form input
		fmt.Printf("Enter %s value (or press Enter to skip): ", fieldName)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Failed to read input: %v\n", err)
			return nil
		}
		
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Printf("Skipped %s for issue #%d\n", fieldName, issue.Number)
			return nil
		}
		
		return &input
		
	default:
		fmt.Printf("Field '%s' has type '%s' which is not yet supported for interactive mode\n", fieldName, targetField.DataType)
		fmt.Printf("Currently supported types: SINGLE_SELECT, TEXT, NUMBER\n")
		return nil
	}
}


func (c *TriageCommand) searchIssuesWithProjectFields(fieldFilters map[string]string, fieldExcludes map[string]bool, labelExcludes []string) ([]GitHubIssue, error) {
	projectID := c.config.Metadata.Project.ID
	
	// Build GraphQL query to get all project items with field values
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
							id
							databaseId
							content {
								... on Issue {
									id
									number
									title
									url
									state
									labels(first: 100) {
										nodes {
											name
										}
									}
								}
							}
							fieldValues(first: 20) {
								nodes {
									... on ProjectV2ItemFieldSingleSelectValue {
										field {
											... on ProjectV2SingleSelectField {
												id
												name
											}
										}
										optionId
										name
									}
									... on ProjectV2ItemFieldTextValue {
										field {
											... on ProjectV2Field {
												id
												name
											}
										}
										text
									}
									... on ProjectV2ItemFieldNumberValue {
										field {
											... on ProjectV2Field {
												id
												name
											}
										}
										number
									}
								}
							}
						}
					}
				}
			}
		}`
	
	var allItems []GitHubIssue
	var endCursor *string
	
	// Prepare field option IDs to filter by
	filterOptionIDs := make(map[string]string) // fieldID -> optionID
	
	// Map field filters to option IDs using metadata dynamically
	for fieldName, filterValue := range fieldFilters {
		if fieldMeta := c.config.GetFieldMetadata(fieldName); fieldMeta != nil && filterValue != "" {
			if optionID, ok := fieldMeta.Options[filterValue]; ok {
				filterOptionIDs[fieldMeta.ID] = optionID
			}
		}
	}
	
	fmt.Printf("Fetching project items with field values...\n")
	
	// Paginate through all project items
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
						ID         string `json:"id"`
						DatabaseID int    `json:"databaseId"`
						Content    struct {
							ID     string `json:"id"`
							Number int    `json:"number"`
							Title  string `json:"title"`
							URL    string `json:"url"`
							State  string `json:"state"`
							Labels struct {
								Nodes []struct {
									Name string `json:"name"`
								} `json:"nodes"`
							} `json:"labels"`
						} `json:"content"`
						FieldValues struct {
							Nodes []map[string]interface{} `json:"nodes"`
						} `json:"fieldValues"`
					} `json:"nodes"`
				} `json:"items"`
			} `json:"node"`
		}
		
		err := c.issueAPI.GetGraphQLClient().Do(query, variables, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch project items: %w", err)
		}
		
		// Process items
		for _, item := range result.Node.Items.Nodes {
			// Skip if not an issue or closed
			if item.Content.Number == 0 || item.Content.State != "OPEN" {
				continue
			}
			
			// Check label exclusions
			skipItem := false
			for _, excludeLabel := range labelExcludes {
				for _, label := range item.Content.Labels.Nodes {
					if label.Name == excludeLabel {
						skipItem = true
						break
					}
				}
				if skipItem {
					break
				}
			}
			if skipItem {
				continue
			}
			
			// Build a map of field values for this item
			itemFieldValues := make(map[string]interface{})
			for _, fieldValueNode := range item.FieldValues.Nodes {
				if fieldData, ok := fieldValueNode["field"].(map[string]interface{}); ok {
					if fieldID, ok := fieldData["id"].(string); ok {
						if fieldName, ok := fieldData["name"].(string); ok {
							// Check for different field value types
							if optionID, ok := fieldValueNode["optionId"].(string); ok {
								// Single select field
								itemFieldValues[fieldID] = optionID
							} else if text, ok := fieldValueNode["text"].(string); ok {
								// Text field
								itemFieldValues[fieldName] = text
							} else if number, ok := fieldValueNode["number"].(float64); ok {
								// Number field
								itemFieldValues[fieldName] = number
							}
						}
					}
				}
			}
			
			// Check field filters
			matchesAllFilters := true
			for fieldID, requiredOptionID := range filterOptionIDs {
				if value, exists := itemFieldValues[fieldID]; exists {
					if value != requiredOptionID {
						matchesAllFilters = false
						break
					}
				} else {
					matchesAllFilters = false
					break
				}
			}
			
			// Check field exclusions (e.g., -has:estimate)
			if matchesAllFilters {
				for excludeFieldName := range fieldExcludes {
					// Check if this field has any value
					hasValue := false
					for fieldName, value := range itemFieldValues {
						if strings.EqualFold(fieldName, excludeFieldName) {
							// Check if the value is not empty
							switch v := value.(type) {
							case string:
								if v != "" {
									hasValue = true
								}
							case float64:
								hasValue = true // any number means it has a value
							default:
								if v != nil {
									hasValue = true
								}
							}
							break
						}
					}
					if hasValue {
						matchesAllFilters = false
						break
					}
				}
			}
			
			if matchesAllFilters {
				allItems = append(allItems, GitHubIssue{
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
	
	fmt.Printf("Found %d issues matching criteria\n", len(allItems))
	return allItems, nil
}


func (c *TriageCommand) displayIssuesList(issues []GitHubIssue, triageConfig config.TriageConfig) error {
	// Display instruction if configured
	if triageConfig.Instruction != "" {
		// Use dim cyan for instruction
		fmt.Printf("\033[36m%s\033[0m\n\n", triageConfig.Instruction)
	}
	
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
	if triageConfig.Interactive.Status || triageConfig.Interactive.Estimate || len(triageConfig.InteractiveFields) > 0 {
		fmt.Printf("- Interactive fields:\n")
		if triageConfig.Interactive.Status {
			fmt.Printf("  - Status (will prompt for each issue)\n")
		}
		if triageConfig.Interactive.Estimate {
			fmt.Printf("  - Estimate (will prompt for each issue)\n")
		}
		for fieldName := range triageConfig.InteractiveFields {
			fmt.Printf("  - %s (will prompt for each issue)\n", strings.Title(fieldName))
		}
	}
	
	if len(triageConfig.Apply.Labels) == 0 && len(triageConfig.Apply.Fields) == 0 && 
		!triageConfig.Interactive.Status && !triageConfig.Interactive.Estimate && 
		len(triageConfig.InteractiveFields) == 0 {
		fmt.Printf("- No changes configured\n")
	}
	
	fmt.Printf("\nTo execute these changes, run without --list or --dry-run flags.\n")
	
	return nil
}
