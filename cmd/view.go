package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/issue"
	"github.com/yahsan2/gh-pm/pkg/output"
	"github.com/yahsan2/gh-pm/pkg/project"
)

var viewCmd = &cobra.Command{
	Use:   "view [issue-number]",
	Short: "View an issue with project metadata",
	Long: `Display detailed information about a GitHub issue including its project metadata.

This command shows:
- Basic issue information (number, title, state, labels)
- Project status and custom fields (priority, status, etc.)
- Project board URL for quick access`,
	Example: `  # View an issue by number
  gh pm view 123
  
  # View an issue in a specific repository
  gh pm view 456 --repo owner/repo
  
  # View in JSON format
  gh pm view 789 --output json
  
  # View with quiet output (URLs only)
  gh pm view 101 --quiet`,
	Args: cobra.ExactArgs(1),
	RunE: runView,
}

// Command flags
var (
	viewRepo      string
	viewQuiet     bool
	viewWeb       bool
	viewComments  bool
)

func init() {
	rootCmd.AddCommand(viewCmd)
	
	// Repository selection
	viewCmd.Flags().StringVarP(&viewRepo, "repo", "r", "", "Repository (owner/repo format)")
	
	// Output options
	viewCmd.Flags().BoolVarP(&viewQuiet, "quiet", "q", false, "Only output URLs")
	viewCmd.Flags().BoolVarP(&viewWeb, "web", "w", false, "Open in web browser")
	viewCmd.Flags().BoolVar(&viewComments, "comments", false, "Include comments")
}

type ViewCommand struct {
	config     *config.Config
	client     *project.Client
	issueAPI   *issue.Client
	formatter  *output.Formatter
	urlBuilder *project.URLBuilder
}

func runView(cmd *cobra.Command, args []string) error {
	// Parse issue number
	issueNumber, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}
	
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		// Configuration is optional for view command
		cfg = &config.Config{}
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
	if viewQuiet {
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
	command := &ViewCommand{
		config:     cfg,
		client:     projectClient,
		issueAPI:   issueClient,
		formatter:  formatter,
		urlBuilder: urlBuilder,
	}
	
	return command.Execute(issueNumber)
}

func (c *ViewCommand) Execute(issueNumber int) error {
	// Determine repository
	repo := viewRepo
	if repo == "" && len(c.config.Repositories) > 0 {
		repo = c.config.Repositories[0]
	}
	
	// If web flag is set, open in browser
	if viewWeb {
		return c.openInBrowser(issueNumber, repo)
	}
	
	// Get issue details
	issueDetails, err := c.getIssueDetails(issueNumber, repo)
	if err != nil {
		return fmt.Errorf("failed to get issue details: %w", err)
	}
	
	// Get project metadata if configured
	if c.config.Project.Name != "" || c.config.Project.Number > 0 {
		projectData, err := c.getProjectMetadata(issueDetails)
		if err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: could not get project metadata: %v\n", err)
		} else {
			issueDetails.ProjectItem = projectData
			
			// Build project URL
			if projectData != nil && projectData.DatabaseID > 0 {
				issueDetails.ProjectURL = c.urlBuilder.GetProjectItemURL(projectData.DatabaseID)
			}
		}
	}
	
	// Get comments if requested
	if viewComments {
		comments, err := c.getIssueComments(issueNumber, repo)
		if err != nil {
			fmt.Printf("Warning: could not get comments: %v\n", err)
		} else {
			issueDetails.Comments = comments
		}
	}
	
	// Format and display output
	return c.formatter.FormatIssueView(issueDetails)
}

func (c *ViewCommand) getIssueDetails(issueNumber int, repo string) (*issue.Issue, error) {
	// Use gh issue view to get detailed information
	args := []string{"issue", "view", strconv.Itoa(issueNumber), "--json", 
		"id,number,title,body,url,state,labels,assignees,milestone,createdAt,updatedAt,closedAt,author"}
	
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	
	cmd := exec.Command("gh", args...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}
	
	// Parse the JSON output
	// For now, we'll use the simpler GetIssueDetails function
	return issue.GetIssueDetails(issueNumber, repo)
}

func (c *ViewCommand) getProjectMetadata(issueDetails *issue.Issue) (*issue.ProjectItem, error) {
	// Get project ID
	projectID := c.config.GetProjectID()
	if projectID == "" {
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
			return nil, err
		}
		projectID = proj.ID
		// Cache the project ID for future use
		c.config.SetProjectID(projectID)
	}
	
	// Get project item for this issue
	itemData, err := c.client.GetProjectItemForIssue(projectID, issueDetails.ID)
	if err != nil {
		return nil, err
	}
	
	// Get project fields
	fields, err := c.client.GetFieldsWithOptions(projectID)
	if err != nil {
		return nil, err
	}
	
	// Map field values
	projectItem := &issue.ProjectItem{
		ID:         itemData.ID,
		DatabaseID: itemData.DatabaseID,
		Fields:     make(map[string]interface{}),
	}
	
	// Extract field values from item data
	for _, field := range fields {
		if field.DataType == "SINGLE_SELECT" {
			// For single select fields, get the selected option
			if fieldValue, ok := itemData.FieldValues[field.ID]; ok {
				if optionID, ok := fieldValue.(string); ok {
					// Find the option name
					for _, option := range field.Options {
						if option.ID == optionID {
							projectItem.Fields[field.Name] = option.Name
							break
						}
					}
				}
			}
		} else {
			// For other field types, store the value directly
			if fieldValue, ok := itemData.FieldValues[field.ID]; ok {
				projectItem.Fields[field.Name] = fieldValue
			}
		}
	}
	
	return projectItem, nil
}

func (c *ViewCommand) getIssueComments(issueNumber int, repo string) ([]issue.Comment, error) {
	args := []string{"issue", "view", strconv.Itoa(issueNumber), "--comments", "--json", "comments"}
	
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	
	cmd := exec.Command("gh", args...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	
	// Parse comments from output
	// This is a simplified implementation
	return []issue.Comment{}, nil
}

func (c *ViewCommand) openInBrowser(issueNumber int, repo string) error {
	// Get issue details first to get the URL
	issueDetails, err := c.getIssueDetails(issueNumber, repo)
	if err != nil {
		return err
	}
	
	// Determine which URL to open
	urlToOpen := issueDetails.URL
	
	// If project URL is available and configured, prefer that
	if c.config.Project.Number > 0 {
		projectData, err := c.getProjectMetadata(issueDetails)
		if err == nil && projectData != nil && projectData.DatabaseID > 0 {
			projectURL := c.urlBuilder.GetProjectItemURL(projectData.DatabaseID)
			if projectURL != "" {
				urlToOpen = projectURL
			}
		}
	}
	
	// Print the URL we're opening
	if _, err := fmt.Fprintf(os.Stderr, "Opening %s in your browser.\n", urlToOpen); err != nil {
		// Ignore error, just for user feedback
	}
	
	// Open URL in browser based on OS
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", urlToOpen)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", urlToOpen)
	default: // linux, bsd, etc.
		// Try xdg-open first, then fallback to other options
		if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command("xdg-open", urlToOpen)
		} else if _, err := exec.LookPath("sensible-browser"); err == nil {
			cmd = exec.Command("sensible-browser", urlToOpen)
		} else if _, err := exec.LookPath("x-www-browser"); err == nil {
			cmd = exec.Command("x-www-browser", urlToOpen)
		} else {
			return fmt.Errorf("could not find a suitable browser command")
		}
	}
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}
	
	return nil
}