package args

import (
	"github.com/spf13/cobra"

	"github.com/yahsan2/gh-pm/pkg/filter"
)

// CommonFlags contains flag names used across commands
type CommonFlags struct {
	Label     string
	Assignee  string
	Author    string
	State     string
	Milestone string
	Search    string
	Limit     string
	Mention   string
	App       string
}

// DefaultFlags returns the default flag names
func DefaultFlags() *CommonFlags {
	return &CommonFlags{
		Label:     "label",
		Assignee:  "assignee",
		Author:    "author",
		State:     "state",
		Milestone: "milestone",
		Search:    "search",
		Limit:     "limit",
		Mention:   "mention",
		App:       "app",
	}
}

// AddCommonFlags adds common gh issue list compatible flags to the command
func AddCommonFlags(cmd *cobra.Command, flags *CommonFlags) {
	if flags == nil {
		flags = DefaultFlags()
	}

	// gh issue list compatible flags
	cmd.Flags().StringSliceP(flags.Label, "l", []string{}, "Filter by label")
	cmd.Flags().StringP(flags.Assignee, "a", "", "Filter by assignee")
	cmd.Flags().StringP(flags.Author, "A", "", "Filter by author")
	cmd.Flags().StringP(flags.State, "s", "open", "Filter by state: {open|closed|all}")
	cmd.Flags().StringP(flags.Milestone, "m", "", "Filter by milestone number or title")
	cmd.Flags().StringP(flags.Search, "S", "", "Search issues with query")
	cmd.Flags().IntP(flags.Limit, "L", 100, "Maximum number of issues to fetch")
	cmd.Flags().String(flags.Mention, "", "Filter by mention")
	cmd.Flags().String(flags.App, "", "Filter by GitHub App author")
}

// ParseCommonFlags extracts common filters from command flags
func ParseCommonFlags(cmd *cobra.Command, flags *CommonFlags) (*filter.IssueFilters, error) {
	if flags == nil {
		flags = DefaultFlags()
	}

	filters := filter.NewIssueFilters()

	// Parse flags
	var err error

	if filters.Labels, err = cmd.Flags().GetStringSlice(flags.Label); err != nil {
		return nil, err
	}

	if filters.Assignee, err = cmd.Flags().GetString(flags.Assignee); err != nil {
		return nil, err
	}

	if filters.Author, err = cmd.Flags().GetString(flags.Author); err != nil {
		return nil, err
	}

	if filters.State, err = cmd.Flags().GetString(flags.State); err != nil {
		return nil, err
	}

	if filters.Milestone, err = cmd.Flags().GetString(flags.Milestone); err != nil {
		return nil, err
	}

	if filters.Search, err = cmd.Flags().GetString(flags.Search); err != nil {
		return nil, err
	}

	if filters.Limit, err = cmd.Flags().GetInt(flags.Limit); err != nil {
		return nil, err
	}

	if filters.Mention, err = cmd.Flags().GetString(flags.Mention); err != nil {
		return nil, err
	}

	if filters.App, err = cmd.Flags().GetString(flags.App); err != nil {
		return nil, err
	}

	return filters, nil
}

// AddProjectFlags adds project-specific flags to the command
func AddProjectFlags(cmd *cobra.Command) {
	cmd.Flags().String("status", "", "Filter by project status field")
	cmd.Flags().String("priority", "", "Filter by project priority field")
}

// ParseProjectFlags extracts project-specific filters from command flags
func ParseProjectFlags(cmd *cobra.Command, filters *filter.IssueFilters) error {
	var err error

	if filters.Status, err = cmd.Flags().GetString("status"); err != nil {
		return err
	}

	if filters.Priority, err = cmd.Flags().GetString("priority"); err != nil {
		return err
	}

	return nil
}
