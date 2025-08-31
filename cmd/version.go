package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build information set at compile time via ldflags
var (
	Commit = "none"
	Date   = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display the version information for gh-pm including version number, commit hash, and build date.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gh-pm version %s\n", Version)
		if verbose {
			fmt.Printf("  commit: %s\n", Commit)
			fmt.Printf("  built:  %s\n", Date)
		}
	},
}

var verbose bool

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed version information")
}
