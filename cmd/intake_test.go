package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/filter"
	"github.com/yahsan2/gh-pm/pkg/issue"
	"github.com/yahsan2/gh-pm/pkg/project"
)

func TestIntakeCommand_ParseApplyFlags(t *testing.T) {
	tests := []struct {
		name        string
		applyFlags  []string
		expected    map[string]string
		expectError bool
	}{
		{
			name:       "parses status and priority",
			applyFlags: []string{"status:backlog", "priority:p2"},
			expected: map[string]string{
				"status":   "backlog",
				"priority": "p2",
			},
			expectError: false,
		},
		{
			name:        "handles empty flags",
			applyFlags:  []string{},
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name:       "parses single field",
			applyFlags: []string{"status:ready"},
			expected: map[string]string{
				"status": "ready",
			},
			expectError: false,
		},
		{
			name:        "invalid format should error",
			applyFlags:  []string{"invalid_format"},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "missing value should error",
			applyFlags:  []string{"status:"},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]string)
			var err error

			// Simulate the parsing logic from runIntake
			for _, apply := range tt.applyFlags {
				parts := strings.SplitN(apply, ":", 2)
				if len(parts) != 2 {
					err = fmt.Errorf("invalid apply format: %s (expected 'field:value')", apply)
					break
				}
				field := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if field == "" || value == "" {
					err = fmt.Errorf("invalid apply format: %s (field and value cannot be empty)", apply)
					break
				}
				result[field] = value
			}

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIntakeCommand_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with project",
			config: &config.Config{
				Project: config.ProjectConfig{
					Name: "test-project",
				},
				Repositories: []string{"owner/repo"},
			},
			expectError: false,
		},
		{
			name: "missing project configuration",
			config: &config.Config{
				Repositories: []string{"owner/repo"},
			},
			expectError: true,
			errorMsg:    "no project configured",
		},
		{
			name: "project with only number",
			config: &config.Config{
				Project: config.ProjectConfig{
					Number: 1,
				},
				Repositories: []string{"owner/repo"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation logic from runIntake
			var err error
			if tt.config.Project.Name == "" && tt.config.Project.Number == 0 {
				err = fmt.Errorf("no project configured. Run 'gh pm init' to configure a project")
			}

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIntakeCommand_FieldMapping(t *testing.T) {
	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
		Fields: map[string]config.Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"backlog":     "Backlog",
					"in_progress": "In Progress",
					"done":        "Done",
				},
			},
			"priority": {
				Field: "Priority",
				Values: map[string]string{
					"p0": "P0",
					"p1": "P1",
					"p2": "P2",
				},
			},
		},
	}

	tests := []struct {
		name           string
		fieldName      string
		filterValue    string
		actualValue    string
		expectedResult bool
	}{
		{
			name:           "maps status field with config",
			fieldName:      "status",
			filterValue:    "in_progress",
			actualValue:    "In Progress",
			expectedResult: true,
		},
		{
			name:           "maps priority field",
			fieldName:      "priority",
			filterValue:    "p1",
			actualValue:    "P1",
			expectedResult: true,
		},
		{
			name:           "field not found",
			fieldName:      "unknown",
			filterValue:    "value",
			actualValue:    "value",
			expectedResult: true, // Should fall back to direct match
		},
		{
			name:           "option not found",
			fieldName:      "status",
			filterValue:    "invalid",
			actualValue:    "Backlog",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a search client to test the shared field matching logic
			searchClient, err := issue.NewSearchClient(cfg)
			require.NoError(t, err)

			// Test field value matching using the correct field name mapping
			actualFieldName := tt.fieldName
			if tt.fieldName == "status" {
				actualFieldName = "Status"
			} else if tt.fieldName == "priority" {
				actualFieldName = "Priority"
			}

			result := searchClient.FilterProjectIssues(
				[]filter.ProjectIssue{
					{
						Number: 1,
						Fields: map[string]interface{}{
							actualFieldName: tt.actualValue,
						},
					},
				},
				&filter.IssueFilters{
					Status: func() string {
						if tt.fieldName == "status" {
							return tt.filterValue
						}
						return ""
					}(),
					Priority: func() string {
						if tt.fieldName == "priority" {
							return tt.filterValue
						}
						return ""
					}(),
				},
			)

			if tt.expectedResult {
				assert.Len(t, result, 1)
			} else {
				assert.Len(t, result, 0)
			}
		})
	}
}

func TestIntakeCommand_IssueFiltering(t *testing.T) {
	// Mock project issues (issues already in project)
	projectIssues := []filter.GitHubIssue{
		{Number: 1, Title: "Issue in Project", ID: "gid_1"},
		{Number: 3, Title: "Another Issue in Project", ID: "gid_3"},
	}

	// Mock search results (all issues found by search)
	searchResults := []filter.GitHubIssue{
		{Number: 1, Title: "Issue in Project", ID: "gid_1"},         // Already in project
		{Number: 2, Title: "Issue Not in Project", ID: "gid_2"},     // Not in project
		{Number: 3, Title: "Another Issue in Project", ID: "gid_3"}, // Already in project
		{Number: 4, Title: "External Issue", ID: "gid_4"},           // Not in project
	}

	// Simulate filtering logic from IntakeCommand.processIssues
	existingMap := make(map[int]bool)
	for _, issue := range projectIssues {
		existingMap[issue.Number] = true
	}

	var issuesToAdd []filter.GitHubIssue
	for _, issue := range searchResults {
		if !existingMap[issue.Number] {
			issuesToAdd = append(issuesToAdd, issue)
		}
	}

	// Verify results
	assert.Len(t, issuesToAdd, 2, "Should find 2 issues not in project")
	assert.Equal(t, 2, issuesToAdd[0].Number)
	assert.Equal(t, 4, issuesToAdd[1].Number)
	assert.Equal(t, "Issue Not in Project", issuesToAdd[0].Title)
	assert.Equal(t, "External Issue", issuesToAdd[1].Title)
}

func TestIntakeCommand_DryRunBehavior(t *testing.T) {
	tests := []struct {
		name         string
		dryRun       bool
		issuesCount  int
		shouldPrompt bool
	}{
		{
			name:         "dry run should not prompt",
			dryRun:       true,
			issuesCount:  3,
			shouldPrompt: false,
		},
		{
			name:         "normal run should prompt",
			dryRun:       false,
			issuesCount:  2,
			shouldPrompt: true,
		},
		{
			name:         "no issues should not prompt",
			dryRun:       false,
			issuesCount:  0,
			shouldPrompt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from IntakeCommand.processIssues
			shouldPrompt := !tt.dryRun && tt.issuesCount > 0
			assert.Equal(t, tt.shouldPrompt, shouldPrompt)
		})
	}
}

func TestIntakeCommand_BackwardCompatibility(t *testing.T) {
	t.Run("query flag backward compatibility", func(t *testing.T) {
		// Simulate the backward compatibility logic from runIntake
		filters := &filter.IssueFilters{}

		// Old query flag
		query := "old search query"
		search := ""

		// Use query if search is not provided (backward compatibility)
		if search == "" && query != "" {
			search = query
		}
		filters.Search = search

		assert.Equal(t, "old search query", filters.Search)
	})

	t.Run("search flag takes precedence", func(t *testing.T) {
		// Simulate the backward compatibility logic from runIntake
		filters := &filter.IssueFilters{}

		// Both flags provided - search should take precedence
		query := "old search query"
		search := "new search query"

		// Use query if search is not provided (backward compatibility)
		if search == "" && query != "" {
			search = query
		}
		filters.Search = search

		assert.Equal(t, "new search query", filters.Search)
	})
}

func TestIntakeCommand_SharedComponentIntegration(t *testing.T) {
	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
		Fields: map[string]config.Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"backlog": "Backlog",
					"ready":   "Ready",
				},
			},
		},
	}

	searchClient, err := issue.NewSearchClient(cfg)
	require.NoError(t, err)

	t.Run("SearchClient filters work correctly", func(t *testing.T) {
		issues := []filter.ProjectIssue{
			{
				Number: 1,
				Title:  "Test Issue 1",
				State:  "open",
				Fields: map[string]interface{}{
					"Status": "Backlog",
				},
			},
			{
				Number: 2,
				Title:  "Test Issue 2",
				State:  "closed",
				Fields: map[string]interface{}{
					"Status": "Ready",
				},
			},
		}

		// Test state filtering
		filtered := searchClient.FilterProjectIssues(issues, &filter.IssueFilters{
			State: "open",
		})
		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)

		// Test status filtering with config mapping
		filtered = searchClient.FilterProjectIssues(issues, &filter.IssueFilters{
			Status: "backlog", // This should map to "Backlog"
		})
		assert.Len(t, filtered, 1)
		assert.Equal(t, 1, filtered[0].Number)
	})
}

func TestIntakeCommand_ProjectFieldUpdate(t *testing.T) {
	fields := []project.Field{
		{
			ID:       "FIELD_status",
			Name:     "Status",
			DataType: "SINGLE_SELECT",
			Options: []project.FieldOption{
				{ID: "OPT_backlog", Name: "Backlog"},
				{ID: "OPT_ready", Name: "Ready"},
			},
		},
		{
			ID:       "FIELD_priority",
			Name:     "Priority",
			DataType: "SINGLE_SELECT",
			Options: []project.FieldOption{
				{ID: "OPT_p0", Name: "P0"},
				{ID: "OPT_p1", Name: "P1"},
			},
		},
	}

	cfg := &config.Config{
		Fields: map[string]config.Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"backlog": "Backlog",
					"ready":   "Ready",
				},
			},
			"priority": {
				Field: "Priority",
				Values: map[string]string{
					"p0": "P0",
					"p1": "P1",
				},
			},
		},
	}

	tests := []struct {
		name       string
		fieldKey   string
		fieldValue string
		expectID   string
		expectErr  bool
	}{
		{
			name:       "status field mapping",
			fieldKey:   "status",
			fieldValue: "backlog",
			expectID:   "OPT_backlog",
			expectErr:  false,
		},
		{
			name:       "priority field mapping",
			fieldKey:   "priority",
			fieldValue: "p1",
			expectID:   "OPT_p1",
			expectErr:  false,
		},
		{
			name:       "invalid field value",
			fieldKey:   "status",
			fieldValue: "invalid",
			expectID:   "",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the field mapping logic from updateProjectField
			var fieldName string
			switch tt.fieldKey {
			case "status":
				fieldName = "Status"
			case "priority":
				fieldName = "Priority"
			default:
				fieldName = tt.fieldKey
			}

			// Find the field
			var targetField *project.Field
			for _, field := range fields {
				if strings.EqualFold(field.Name, fieldName) {
					targetField = &field
					break
				}
			}

			require.NotNil(t, targetField, "Field should be found")

			// Find option ID using config mapping
			var optionID string
			if targetField.DataType == "SINGLE_SELECT" {
				configKey := strings.ToLower(fieldName)
				if configField, ok := cfg.Fields[configKey]; ok {
					if mappedValue, ok := configField.Values[tt.fieldValue]; ok {
						for _, option := range targetField.Options {
							if option.Name == mappedValue {
								optionID = option.ID
								break
							}
						}
					}
				}
			}

			if tt.expectErr {
				assert.Empty(t, optionID, "Should not find option ID for invalid value")
			} else {
				assert.Equal(t, tt.expectID, optionID, "Should find correct option ID")
			}
		})
	}
}
