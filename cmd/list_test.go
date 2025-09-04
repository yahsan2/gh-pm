package cmd

import (
	"testing"
)

func TestFilterIssues(t *testing.T) {
	// Note: This test is now deprecated since filtering logic
	// has been moved to the shared SearchClient in pkg/issue/search.go
	// New tests should be written for the SearchClient.FilterProjectIssues method

	// This test is kept for backward compatibility but marked as skipped
	t.Skip("FilterIssues logic has been moved to shared SearchClient")
}
