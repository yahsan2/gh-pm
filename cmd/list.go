package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/yahsan2/gh-pm/pkg/args"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/filter"
	"github.com/yahsan2/gh-pm/pkg/issue"
	"github.com/yahsan2/gh-pm/pkg/project"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List issues in the project",
	Long: `List issues in the configured project with filtering options.

This command provides a gh issue list compatible interface for viewing issues
within your project, with additional project-specific field filtering.`,
	Example: `  # List all open issues in project
  gh pm list

  # Filter by status
  gh pm list --status "in_progress"

  # Filter by priority
  gh pm list --priority "p0,p1"

  # Filter by assignee
  gh pm list --assignee @me

  # Filter by labels
  gh pm list --label bug --label enhancement

  # Search with query
  gh pm list --search "authentication"

  # Search with GitHub Projects date expressions
  gh pm list --search "updated:@today"
  gh pm list --search "created:@today-1w"
  gh pm list --search "state:open created:>@today-30d"

  # JSON output with specific fields
  gh pm list --json number,title,status,priority

  # Open in web browser
  gh pm list --web`,
	RunE: runList,
}

func init() {
	// Add common gh issue list compatible flags
	flags := args.DefaultFlags()
	flags.Limit = "limit" // Override default limit for list command
	args.AddCommonFlags(listCmd, flags)

	// Override limit default for list command
	listCmd.Flags().Lookup("limit").DefValue = "30"

	// Add project-specific flags
	args.AddProjectFlags(listCmd)

	// Output flags
	listCmd.Flags().String("json", "", "Output JSON with the specified fields")
	listCmd.Flags().StringP("jq", "q", "", "Filter JSON output using a jq expression")
	listCmd.Flags().StringP("template", "t", "", "Format JSON output using a Go template")
	listCmd.Flags().BoolP("web", "w", false, "List issues in the web browser")

	rootCmd.AddCommand(listCmd)
}

type ListCommand struct {
	config    *config.Config
	client    *project.Client
	issueAPI  *issue.Client
	searchAPI *issue.SearchClient
}

func runList(cmd *cobra.Command, cmdArgs []string) error {
	// Parse common flags using shared argument parser
	filters, err := args.ParseCommonFlags(cmd, nil)
	if err != nil {
		return fmt.Errorf("failed to parse common flags: %w", err)
	}

	// Parse project-specific flags
	if err := args.ParseProjectFlags(cmd, filters); err != nil {
		return fmt.Errorf("failed to parse project flags: %w", err)
	}

	// Override limit default if not set
	if filters.Limit == 0 {
		filters.Limit = 30
	}

	// Output flags
	jsonFields, _ := cmd.Flags().GetString("json")
	jqExpr, _ := cmd.Flags().GetString("jq")
	template, _ := cmd.Flags().GetString("template")
	webMode, _ := cmd.Flags().GetBool("web")

	// Handle web mode
	if webMode {
		return openProjectInBrowser()
	}

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
	command := &ListCommand{
		config:    cfg,
		client:    projectClient,
		issueAPI:  issueClient,
		searchAPI: searchClient,
	}

	// Get project ID
	projectID := cfg.GetProjectID()
	if projectID == "" {
		// Fetch project ID if not cached
		var proj *project.Project
		if cfg.Project.Org != "" {
			proj, err = projectClient.GetProject(
				cfg.Project.Org,
				cfg.Project.Name,
				cfg.Project.Number,
			)
		} else {
			proj, err = projectClient.GetCurrentUserProject(
				cfg.Project.Name,
				cfg.Project.Number,
			)
		}

		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}
		projectID = proj.ID
		cfg.SetProjectID(projectID)
	}

	// Fetch project items with filters using shared search client
	issues, err := command.searchAPI.FetchProjectIssues(projectID, filters.Limit)
	if err != nil {
		return fmt.Errorf("failed to fetch project issues: %w", err)
	}

	// Apply local filters using shared filtering logic
	filtered := command.searchAPI.FilterProjectIssues(issues, filters)

	// Handle JSON output
	if jsonFields != "" {
		return command.outputJSON(filtered, jsonFields, jqExpr, template)
	}

	// Default table output
	return command.outputTable(filtered)
}

func (c *ListCommand) fetchProjectIssues(projectID string, limit int) ([]filter.ProjectIssue, error) {
	query := `
		query($projectId: ID!, $endCursor: String, $limit: Int!) {
			node(id: $projectId) {
				... on ProjectV2 {
					items(first: $limit, after: $endCursor) {
						pageInfo {
							hasNextPage
							endCursor
						}
						nodes {
							id
							databaseId
							fieldValues(first: 20) {
								nodes {
									... on ProjectV2ItemFieldTextValue {
										field {
											... on ProjectV2Field {
												name
											}
										}
										text
									}
									... on ProjectV2ItemFieldNumberValue {
										field {
											... on ProjectV2Field {
												name
											}
										}
										number
									}
									... on ProjectV2ItemFieldDateValue {
										field {
											... on ProjectV2Field {
												name
											}
										}
										date
									}
									... on ProjectV2ItemFieldSingleSelectValue {
										field {
											... on ProjectV2SingleSelectField {
												name
											}
										}
										name
									}
									... on ProjectV2ItemFieldIterationValue {
										field {
											... on ProjectV2IterationField {
												name
											}
										}
										title
									}
								}
							}
							content {
								... on Issue {
									id
									number
									title
									state
									url
									body
									createdAt
									updatedAt
									closedAt
									author {
										login
									}
									assignees(first: 10) {
										nodes {
											login
										}
									}
									labels(first: 20) {
										nodes {
											name
										}
									}
									milestone {
										title
									}
									comments {
										totalCount
									}
								}
							}
						}
					}
				}
			}
		}`

	var allIssues []filter.ProjectIssue
	var endCursor *string

	for {
		variables := map[string]interface{}{
			"projectId": projectID,
			"limit":     limit,
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
						ID          string `json:"id"`
						DatabaseID  int    `json:"databaseId"`
						FieldValues struct {
							Nodes []interface{} `json:"nodes"`
						} `json:"fieldValues"`
						Content struct {
							ID        string `json:"id"`
							Number    int    `json:"number"`
							Title     string `json:"title"`
							State     string `json:"state"`
							URL       string `json:"url"`
							Body      string `json:"body"`
							CreatedAt string `json:"createdAt"`
							UpdatedAt string `json:"updatedAt"`
							ClosedAt  string `json:"closedAt"`
							Author    struct {
								Login string `json:"login"`
							} `json:"author"`
							Assignees struct {
								Nodes []struct {
									Login string `json:"login"`
								} `json:"nodes"`
							} `json:"assignees"`
							Labels struct {
								Nodes []struct {
									Name string `json:"name"`
								} `json:"nodes"`
							} `json:"labels"`
							Milestone struct {
								Title string `json:"title"`
							} `json:"milestone"`
							Comments struct {
								TotalCount int `json:"totalCount"`
							} `json:"comments"`
						} `json:"content"`
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
			if item.Content.Number == 0 {
				continue // Skip non-issues
			}

			issue := filter.ProjectIssue{
				Number:    item.Content.Number,
				Title:     item.Content.Title,
				State:     strings.ToLower(item.Content.State),
				URL:       item.Content.URL,
				ID:        item.Content.ID,
				Body:      item.Content.Body,
				Author:    item.Content.Author.Login,
				Milestone: item.Content.Milestone.Title,
				CreatedAt: item.Content.CreatedAt,
				UpdatedAt: item.Content.UpdatedAt,
				ClosedAt:  item.Content.ClosedAt,
				Comments:  item.Content.Comments.TotalCount,
				Fields:    make(map[string]interface{}),
			}

			// Add assignees
			for _, assignee := range item.Content.Assignees.Nodes {
				issue.Assignees = append(issue.Assignees, assignee.Login)
			}

			// Add labels
			for _, label := range item.Content.Labels.Nodes {
				issue.Labels = append(issue.Labels, label.Name)
			}

			// Parse field values
			for _, fieldValue := range item.FieldValues.Nodes {
				if fv, ok := fieldValue.(map[string]interface{}); ok {
					if field, ok := fv["field"].(map[string]interface{}); ok {
						if fieldName, ok := field["name"].(string); ok {
							// Get the value based on type
							if text, ok := fv["text"].(string); ok {
								issue.Fields[fieldName] = text
							} else if number, ok := fv["number"].(float64); ok {
								issue.Fields[fieldName] = number
							} else if date, ok := fv["date"].(string); ok {
								issue.Fields[fieldName] = date
							} else if name, ok := fv["name"].(string); ok {
								issue.Fields[fieldName] = name
							} else if title, ok := fv["title"].(string); ok {
								issue.Fields[fieldName] = title
							}
						}
					}
				}
			}

			// Generate project URL
			urlBuilder := project.NewURLBuilder(c.config, c.client)
			issue.ProjectURL = urlBuilder.GetProjectItemURL(item.DatabaseID)

			allIssues = append(allIssues, issue)
		}

		// Check if we've fetched enough or there are no more pages
		if len(allIssues) >= limit || !result.Node.Items.PageInfo.HasNextPage {
			break
		}
		endCursor = &result.Node.Items.PageInfo.EndCursor
	}

	// Trim to limit
	if len(allIssues) > limit {
		allIssues = allIssues[:limit]
	}

	return allIssues, nil
}

// The filtering logic is now handled by the shared SearchClient

func (c *ListCommand) outputTable(issues []filter.ProjectIssue) error {
	if len(issues) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintf(w, "#\tTITLE\tSTATUS\tPRIORITY\tASSIGNEES\tLABELS\n")

	// Rows
	for _, issue := range issues {
		status := ""
		if s, ok := issue.Fields["Status"].(string); ok {
			status = s
		}

		priority := ""
		if p, ok := issue.Fields["Priority"].(string); ok {
			priority = p
		}

		assignees := strings.Join(issue.Assignees, ", ")
		if assignees == "" {
			assignees = "-"
		}

		labels := strings.Join(issue.Labels, ", ")
		if labels == "" {
			labels = "-"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			issue.Number,
			truncate(issue.Title, 50),
			status,
			priority,
			assignees,
			labels,
		)
	}

	return w.Flush()
}

func (c *ListCommand) outputJSON(issues []filter.ProjectIssue, fields, jqExpr, template string) error {
	// If specific fields requested, filter the output
	var output interface{}

	if fields != "" {
		requestedFields := strings.Split(fields, ",")
		var filtered []map[string]interface{}

		for _, issue := range issues {
			item := make(map[string]interface{})

			// Marshal to map for field selection
			data, _ := json.Marshal(issue)
			var fullItem map[string]interface{}
			if err := json.Unmarshal(data, &fullItem); err != nil {
				continue
			}

			// Select requested fields
			for _, field := range requestedFields {
				field = strings.TrimSpace(field)
				if value, ok := fullItem[field]; ok {
					item[field] = value
				}
			}

			filtered = append(filtered, item)
		}
		output = filtered
	} else {
		output = issues
	}

	// Apply jq filter if specified
	if jqExpr != "" {
		// This would require jq integration
		// For now, just output the JSON
		fmt.Fprintln(os.Stderr, "Warning: jq filtering not yet implemented")
	}

	// Apply Go template if specified
	if template != "" {
		// This would require template processing
		// For now, just output the JSON
		fmt.Fprintln(os.Stderr, "Warning: template formatting not yet implemented")
	}

	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func openProjectInBrowser() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	projectClient, err := project.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create project client: %w", err)
	}

	urlBuilder := project.NewURLBuilder(cfg, projectClient)
	projectURL := urlBuilder.GetProjectURL()

	cmd := exec.Command("gh", "browse", projectURL)
	return cmd.Run()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
