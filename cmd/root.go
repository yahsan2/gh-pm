package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "gh-pm",
	Short: "GitHub CLI extension for project management",
	Long: `A GitHub CLI extension for project management with GitHub Projects (v2) and Issues.

This extension allows you to:
- Manage GitHub Projects v2 directly from CLI
- Create, update, and track issues with rich metadata
- Break down issues into manageable sub-tasks
- Set and track priorities across issues
- Monitor task completion and project status`,
	Version: Version,
}

// Global flags
var (
	projectName  string
	orgName      string
	repoNames    []string
	outputFormat string
)

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&projectName, "project", "p", "", "Target project name or ID")
	rootCmd.PersistentFlags().StringVar(&orgName, "org", "", "Organization name")
	rootCmd.PersistentFlags().StringSliceVar(&repoNames, "repo", []string{}, "Repository (owner/repo format, can be specified multiple times)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, csv)")
}

func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
