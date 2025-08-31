package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yahsan2/gh-pm/pkg/issue"
	"github.com/yahsan2/gh-pm/pkg/output"
)

var (
	splitFrom   string
	splitRepo   string
	splitDryRun bool
)

// splitCmd represents the split command
var splitCmd = &cobra.Command{
	Use:   "split [issue number] [tasks...]",
	Short: "Split an issue into sub-issues",
	Long: `Split a parent issue into sub-issues with automatic parent-child linking.
	
Examples:
  # Split from issue body checklist
  gh pm split 123 --from=body
  
  # Split from a file
  gh pm split 123 --from=./tasks.md
  
  # Split from stdin
  cat tasks.md | gh pm split 123
  
  # Split from JSON array
  gh pm split 123 '["Task 1", "Task 2", "Task 3"]'
  
  # Split from command arguments
  gh pm split 123 "Task 1" "Task 2" "Task 3"`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires at least 1 argument")
		}
		// Validate that the first argument is a valid issue number
		if _, err := strconv.Atoi(args[0]); err != nil {
			return fmt.Errorf("invalid issue number: %s", args[0])
		}
		return nil
	},
	RunE: runSplit,
}

func init() {
	rootCmd.AddCommand(splitCmd)
	splitCmd.Flags().StringVar(&splitFrom, "from", "", "Source of tasks: 'body' (issue body) or file path")
	splitCmd.Flags().StringVar(&splitRepo, "repo", "", "Repository (owner/repo format)")
	splitCmd.Flags().BoolVar(&splitDryRun, "dry-run", false, "Preview what would be created without making changes")
}

// isGhSubIssueInstalled checks if gh sub-issue extension is installed
func isGhSubIssueInstalled() bool {
	cmd := exec.Command("gh", "extension", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "gh-sub-issue")
}

// SubIssueInfo represents a sub-issue from gh sub-issue list
type SubIssueInfo struct {
	Number int
	State  string
	Title  string
}

// getExistingSubIssues gets the list of existing sub-issues for a parent issue
func getExistingSubIssues(parentIssueNum int, repo string) ([]SubIssueInfo, error) {
	args := []string{"sub-issue", "list", strconv.Itoa(parentIssueNum)}
	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var subIssues []SubIssueInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse format: "123	open	Title of issue"
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			number, _ := strconv.Atoi(parts[0])
			subIssues = append(subIssues, SubIssueInfo{
				Number: number,
				State:  parts[1],
				Title:  parts[2],
			})
		}
	}

	return subIssues, nil
}

// isTaskAlreadySubIssue checks if a task already exists as a sub-issue
func isTaskAlreadySubIssue(task string, existingSubIssues []SubIssueInfo) bool {
	taskLower := strings.ToLower(strings.TrimSpace(task))
	for _, subIssue := range existingSubIssues {
		titleLower := strings.ToLower(strings.TrimSpace(subIssue.Title))
		// Check for exact match or if the existing title contains the task
		if titleLower == taskLower || strings.Contains(titleLower, taskLower) {
			return true
		}
		// Also check if task contains the existing title (in case of slight variations)
		if strings.Contains(taskLower, titleLower) {
			return true
		}
	}
	return false
}

func runSplit(cmd *cobra.Command, args []string) error {
	// Check if gh sub-issue extension is installed (skip for dry-run)
	if !splitDryRun && !isGhSubIssueInstalled() {
		fmt.Println("❌ gh sub-issue extension is not installed.")
		fmt.Println("\nThis command requires the gh sub-issue extension to create properly linked sub-issues.")
		fmt.Println("Please install it by running:")
		fmt.Println("\n  gh extension install yahsan2/gh-sub-issue")
		fmt.Println("\nFor more information, visit: https://github.com/yahsan2/gh-sub-issue")
		return fmt.Errorf("gh sub-issue extension is required")
	}

	// Parse issue number
	issueNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}

	// Get tasks based on input method
	var tasks []string

	if splitFrom == "body" {
		// Extract from issue body
		tasks, err = extractTasksFromIssueBody(issueNum, splitRepo)
		if err != nil {
			return fmt.Errorf("failed to extract tasks from issue body: %w", err)
		}
	} else if splitFrom != "" {
		// Read from file
		tasks, err = extractTasksFromFile(splitFrom)
		if err != nil {
			return fmt.Errorf("failed to read tasks from file: %w", err)
		}
	} else if len(args) > 1 {
		// Check if first argument after issue number is JSON
		if strings.HasPrefix(args[1], "[") {
			err = json.Unmarshal([]byte(args[1]), &tasks)
			if err != nil {
				return fmt.Errorf("failed to parse JSON tasks: %w", err)
			}
		} else {
			// Use remaining arguments as tasks
			tasks = args[1:]
		}
	} else {
		// Check if stdin has data
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			tasks, err = extractTasksFromReader(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read tasks from stdin: %w", err)
			}
		} else {
			return fmt.Errorf("no tasks provided. Use --from flag, provide tasks as arguments, or pipe from stdin")
		}
	}

	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found to create sub-issues")
	}

	// Create or preview sub-issues
	if splitDryRun {
		fmt.Printf("🔍 DRY-RUN: Previewing sub-issues that would be created for issue #%d...\n\n", issueNum)
	} else {
		fmt.Printf("Checking for existing sub-issues and creating new ones for issue #%d...\n", issueNum)
	}

	client := issue.NewClient()
	parentIssue, err := client.GetIssueWithRepo(issueNum, splitRepo)
	if err != nil {
		return fmt.Errorf("failed to get parent issue: %w", err)
	}

	// Get existing sub-issues to avoid duplicates
	var existingSubIssues []SubIssueInfo
	if !splitDryRun {
		existingSubIssues, err = getExistingSubIssues(issueNum, splitRepo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get existing sub-issues: %v\n", err)
			// Continue anyway, but might create duplicates
		}

		if len(existingSubIssues) > 0 {
			fmt.Printf("Found %d existing sub-issues for issue #%d\n", len(existingSubIssues), issueNum)
		}
	}

	createdIssues := []issue.Issue{}
	skippedCount := 0
	wouldCreateCount := 0

	if splitDryRun {
		// Dry-run mode: show what would be created
		fmt.Println("Parent Issue:")
		fmt.Printf("  #%d: %s\n", parentIssue.Number, parentIssue.Title)
		if len(parentIssue.Labels) > 0 {
			fmt.Print("  Labels: ")
			for i, label := range parentIssue.Labels {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(label.Name)
			}
			fmt.Println()
		}
		if len(parentIssue.Assignees) > 0 {
			fmt.Printf("  Assignees: %s\n", strings.Join(parentIssue.Assignees, ", "))
		}
		if parentIssue.Milestone != "" {
			fmt.Printf("  Milestone: %s\n", parentIssue.Milestone)
		}
		fmt.Println("\nSub-issues that would be created:")
		fmt.Println("─────────────────────────────────")

		for _, task := range tasks {
			wouldCreateCount++
			fmt.Printf("  %d. %s\n", wouldCreateCount, task)

			// Show what would be inherited
			inheritedItems := []string{}
			if len(parentIssue.Labels) > 0 {
				labelCount := 0
				for _, label := range parentIssue.Labels {
					if label.Name != "epic" && label.Name != "parent" && label.Name != "sub-task" {
						labelCount++
					}
				}
				if labelCount > 0 {
					inheritedItems = append(inheritedItems, fmt.Sprintf("%d labels", labelCount))
				}
			}
			if len(parentIssue.Assignees) > 0 {
				inheritedItems = append(inheritedItems, fmt.Sprintf("%d assignees", len(parentIssue.Assignees)))
			}
			if parentIssue.Milestone != "" {
				inheritedItems = append(inheritedItems, "milestone")
			}

			if len(inheritedItems) > 0 {
				fmt.Printf("     → Inherits: %s\n", strings.Join(inheritedItems, ", "))
			}
		}

		fmt.Println("\n─────────────────────────────────")
		fmt.Printf("Total: %d sub-issues would be created\n", wouldCreateCount)
		fmt.Println("\nTo actually create these sub-issues, run without --dry-run")

	} else {
		// Normal mode: actually create sub-issues
		for i, task := range tasks {
			// Check if this task already exists as a sub-issue
			if isTaskAlreadySubIssue(task, existingSubIssues) {
				fmt.Printf("⏭️  Skipping (already exists): %s\n", task)
				skippedCount++
				continue
			}

			subIssue, err := createSubIssue(client, parentIssue, task, i+1, splitRepo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create sub-issue for task '%s': %v\n", task, err)
				continue
			}
			createdIssues = append(createdIssues, subIssue)
			fmt.Printf("✓ Created sub-issue #%d: %s\n", subIssue.Number, subIssue.Title)
		}

		if skippedCount > 0 {
			fmt.Printf("\nSkipped %d tasks that already have sub-issues\n", skippedCount)
		}
	}

	// Optional: Update parent issue body with sub-issue links
	// Note: GitHub's native sub-issue feature will show these automatically
	// Uncomment if you want to also add links in the issue body
	/*
		if len(createdIssues) > 0 {
			err = updateParentIssueWithSubIssues(client, parentIssue, createdIssues, splitRepo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to update parent issue with sub-issue links: %v\n", err)
			}
		}
	*/

	// Output summary (skip for dry-run in non-JSON format)
	if splitDryRun && outputFormat != "json" {
		// Already printed detailed preview above
		return nil
	}

	var formatType output.FormatType
	switch outputFormat {
	case "json":
		formatType = output.FormatJSON
	case "table":
		formatType = output.FormatTable
	case "csv":
		formatType = output.FormatCSV
	case "quiet":
		formatType = output.FormatQuiet
	default:
		formatType = output.FormatJSON
	}

	formatter := output.NewFormatter(formatType)

	if splitDryRun {
		// For dry-run JSON output
		summary := map[string]interface{}{
			"dry_run":            true,
			"parent_issue":       issueNum,
			"would_create_count": wouldCreateCount,
			"tasks":              tasks,
		}
		return formatter.Format(summary)
	} else {
		// Normal output
		summary := map[string]interface{}{
			"parent_issue":  issueNum,
			"created_count": len(createdIssues),
			"sub_issues":    createdIssues,
		}
		return formatter.Format(summary)
	}
}

func extractTasksFromIssueBody(issueNum int, repo string) ([]string, error) {
	client := issue.NewClient()
	parentIssue, err := client.GetIssueWithRepo(issueNum, repo)
	if err != nil {
		return nil, err
	}

	tasks := extractChecklistItems(parentIssue.Body)
	return tasks, nil
}

func extractTasksFromFile(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return extractTasksFromReader(file)
}

func extractTasksFromReader(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)

	// Try to detect if it's JSON
	var allContent strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		allContent.WriteString(line + "\n")
	}

	content := strings.TrimSpace(allContent.String())

	// Check if it's JSON array
	if strings.HasPrefix(content, "[") {
		var jsonTasks []string
		err := json.Unmarshal([]byte(content), &jsonTasks)
		if err == nil {
			return jsonTasks, nil
		}
	}

	// Otherwise, extract checklist items
	return extractChecklistItems(content), nil
}

func extractChecklistItems(text string) []string {
	tasks := []string{}

	// Match GitHub-style checkboxes: - [ ] or - [x]
	checkboxPattern := regexp.MustCompile(`^[\s]*[-*]\s*\[[ xX]\]\s*(.+)`)

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		matches := checkboxPattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			task := strings.TrimSpace(matches[1])
			if task != "" {
				tasks = append(tasks, task)
			}
		}
	}

	return tasks
}

func createSubIssue(client *issue.Client, parentIssue issue.Issue, task string, index int, repo string) (issue.Issue, error) {
	// Use gh sub-issue create to create a linked sub-issue
	args := []string{"sub-issue", "create",
		"--parent", strconv.Itoa(parentIssue.Number),
		"--title", task,
		"--body", fmt.Sprintf("## Task\n%s", task),
	}

	// Add labels from parent (except certain meta labels)
	for _, label := range parentIssue.Labels {
		if label.Name != "epic" && label.Name != "parent" && label.Name != "sub-task" {
			args = append(args, "--label", label.Name)
		}
	}

	// Add assignees from parent
	for _, assignee := range parentIssue.Assignees {
		args = append(args, "--assignee", assignee)
	}

	// Add milestone if present
	if parentIssue.Milestone != "" {
		args = append(args, "--milestone", parentIssue.Milestone)
	}

	// Add repo if specified
	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return issue.Issue{}, fmt.Errorf("failed to create sub-issue: %w\nstderr: %s", err, stderr.String())
	}

	// Parse the output to get issue number and URL
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return issue.Issue{}, fmt.Errorf("no output from gh sub-issue create")
	}

	// Extract issue number from URL (format: https://github.com/owner/repo/issues/123)
	parts := strings.Split(output, "/")
	if len(parts) < 2 {
		return issue.Issue{}, fmt.Errorf("unexpected output format: %s", output)
	}

	numberStr := parts[len(parts)-1]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return issue.Issue{}, fmt.Errorf("failed to parse issue number from URL: %s", output)
	}

	return issue.Issue{
		Number: number,
		Title:  task,
		URL:    output,
	}, nil
}

func updateParentIssueWithSubIssues(client *issue.Client, parentIssue issue.Issue, subIssues []issue.Issue, repo string) error {
	// Build sub-issues section
	subIssuesSection := "\n\n## Sub-Issues\n"
	for _, subIssue := range subIssues {
		subIssuesSection += fmt.Sprintf("- [ ] #%d %s\n", subIssue.Number, subIssue.Title)
	}

	// Update parent issue body
	updatedBody := parentIssue.Body

	// Check if sub-issues section already exists
	if strings.Contains(updatedBody, "## Sub-Issues") {
		// Replace existing section
		re := regexp.MustCompile(`(?s)## Sub-Issues.*?(?:##|$)`)
		updatedBody = re.ReplaceAllString(updatedBody, subIssuesSection)
	} else {
		// Append new section
		updatedBody += subIssuesSection
	}

	// Keep existing labels
	labels := []string{}
	for _, label := range parentIssue.Labels {
		labels = append(labels, label.Name)
	}

	return client.UpdateIssueWithRepo(parentIssue.Number, issue.IssueRequest{
		Title:  parentIssue.Title,
		Body:   updatedBody,
		Labels: labels,
	}, repo)
}
