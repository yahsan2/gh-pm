package args

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCommonFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddCommonFlags(cmd, nil)

	// Check that flags were added
	assert.NotNil(t, cmd.Flags().Lookup("label"))
	assert.NotNil(t, cmd.Flags().Lookup("assignee"))
	assert.NotNil(t, cmd.Flags().Lookup("author"))
	assert.NotNil(t, cmd.Flags().Lookup("state"))
	assert.NotNil(t, cmd.Flags().Lookup("milestone"))
	assert.NotNil(t, cmd.Flags().Lookup("search"))
	assert.NotNil(t, cmd.Flags().Lookup("limit"))
	assert.NotNil(t, cmd.Flags().Lookup("mention"))
	assert.NotNil(t, cmd.Flags().Lookup("app"))

	// Check short flags
	assert.Equal(t, "l", cmd.Flags().Lookup("label").Shorthand)
	assert.Equal(t, "a", cmd.Flags().Lookup("assignee").Shorthand)
	assert.Equal(t, "A", cmd.Flags().Lookup("author").Shorthand)
	assert.Equal(t, "s", cmd.Flags().Lookup("state").Shorthand)
	assert.Equal(t, "m", cmd.Flags().Lookup("milestone").Shorthand)
	assert.Equal(t, "S", cmd.Flags().Lookup("search").Shorthand)
	assert.Equal(t, "L", cmd.Flags().Lookup("limit").Shorthand)
}

func TestAddProjectFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddProjectFlags(cmd)

	// Check that project-specific flags were added
	assert.NotNil(t, cmd.Flags().Lookup("status"))
	assert.NotNil(t, cmd.Flags().Lookup("priority"))
}

func TestParseCommonFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddCommonFlags(cmd, nil)

	// Set some flag values
	err := cmd.Flags().Set("state", "closed")
	require.NoError(t, err)
	err = cmd.Flags().Set("limit", "50")
	require.NoError(t, err)
	err = cmd.Flags().Set("assignee", "@me")
	require.NoError(t, err)

	filters, err := ParseCommonFlags(cmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "closed", filters.State)
	assert.Equal(t, 50, filters.Limit)
	assert.Equal(t, "@me", filters.Assignee)
}

func TestParseProjectFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddCommonFlags(cmd, nil)
	AddProjectFlags(cmd)

	// Set project flag values
	err := cmd.Flags().Set("status", "in_progress")
	require.NoError(t, err)
	err = cmd.Flags().Set("priority", "p1")
	require.NoError(t, err)

	filters, err := ParseCommonFlags(cmd, nil)
	require.NoError(t, err)

	err = ParseProjectFlags(cmd, filters)
	require.NoError(t, err)

	assert.Equal(t, "in_progress", filters.Status)
	assert.Equal(t, "p1", filters.Priority)
}

func TestDefaultFlags(t *testing.T) {
	flags := DefaultFlags()

	assert.Equal(t, "label", flags.Label)
	assert.Equal(t, "assignee", flags.Assignee)
	assert.Equal(t, "author", flags.Author)
	assert.Equal(t, "state", flags.State)
	assert.Equal(t, "milestone", flags.Milestone)
	assert.Equal(t, "search", flags.Search)
	assert.Equal(t, "limit", flags.Limit)
	assert.Equal(t, "mention", flags.Mention)
	assert.Equal(t, "app", flags.App)
}
