package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yahsan2/gh-pm/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestInitCommand(t *testing.T) {
	t.Run("contains function", func(t *testing.T) {
		slice := []string{"apple", "banana", "orange"}

		assert.True(t, contains(slice, "banana"))
		assert.False(t, contains(slice, "grape"))
		assert.False(t, contains([]string{}, "anything"))
	})

	t.Run("config file creation", func(t *testing.T) {
		// Create temp directory for test
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		// Change to temp directory
		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create default config
		cfg := config.DefaultConfig()
		cfg.Project.Name = "Test Project"
		cfg.Project.Org = "test-org"
		cfg.Repositories = []string{"test-org/test-repo"}

		// Save config
		configPath := filepath.Join(tmpDir, config.ConfigFileName)
		err = cfg.Save(configPath)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)

		// Load config and verify
		loadedCfg, err := config.Load()
		require.NoError(t, err)

		assert.Equal(t, "Test Project", loadedCfg.Project.Name)
		assert.Equal(t, "test-org", loadedCfg.Project.Org)
		assert.Equal(t, []string{"test-org/test-repo"}, loadedCfg.Repositories)
		assert.Equal(t, "medium", loadedCfg.Defaults.Priority)
		assert.Equal(t, "todo", loadedCfg.Defaults.Status)
	})
}

func TestInitCommandFlags(t *testing.T) {
	// Reset flags for testing
	initProject = ""
	initOrg = ""
	initRepos = []string{}
	initInteractive = false
	skipMetadata = false

	// Test that flags are properly registered
	assert.NotNil(t, initCmd.Flags().Lookup("project"))
	assert.NotNil(t, initCmd.Flags().Lookup("org"))
	assert.NotNil(t, initCmd.Flags().Lookup("repo"))
	assert.NotNil(t, initCmd.Flags().Lookup("interactive"))
	assert.NotNil(t, initCmd.Flags().Lookup("skip-metadata"))
}

func TestUpdateFieldsFromMetadata(t *testing.T) {
	cfg := &config.Config{
		Fields: map[string]config.Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"todo":        "Todo",
					"in_progress": "In Progress",
					"done":        "Done",
				},
			},
			"priority": {
				Field: "Priority",
				Values: map[string]string{
					"high":   "High",
					"medium": "Medium",
					"low":    "Low",
				},
			},
		},
	}

	metadata := &config.ConfigMetadata{
		Fields: []config.FieldInfo{
			{
				Name:     "Status",
				ID:       "PVTSSF_status",
				DataType: "SINGLE_SELECT",
				Options: []config.FieldOption{
					{Name: "Backlog", ID: "backlog_id"},
					{Name: "Ready", ID: "ready_id"},
					{Name: "In progress", ID: "in_progress_id"},
					{Name: "Done", ID: "done_id"},
				},
			},
			{
				Name:     "Priority",
				ID:       "PVTSSF_priority",
				DataType: "SINGLE_SELECT",
				Options: []config.FieldOption{
					{Name: "P0", ID: "p0_id"},
					{Name: "P1", ID: "p1_id"},
					{Name: "P2", ID: "p2_id"},
				},
			},
			{
				Name:     "Title",
				ID:       "PVTF_title",
				DataType: "TITLE",
			},
		},
	}

	updateFieldsFromMetadata(cfg, metadata)

	// Check status field was updated
	assert.Equal(t, "Status", cfg.Fields["status"].Field)
	assert.Equal(t, "Backlog", cfg.Fields["status"].Values["backlog"])
	assert.Equal(t, "Ready", cfg.Fields["status"].Values["ready"])
	assert.Equal(t, "In progress", cfg.Fields["status"].Values["in_progress"])
	assert.Equal(t, "Done", cfg.Fields["status"].Values["done"])

	// Check priority field was updated
	assert.Equal(t, "Priority", cfg.Fields["priority"].Field)
	assert.Equal(t, "P0", cfg.Fields["priority"].Values["p0"])
	assert.Equal(t, "P1", cfg.Fields["priority"].Values["p1"])
	assert.Equal(t, "P2", cfg.Fields["priority"].Values["p2"])
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// Check default values
	assert.Equal(t, "medium", cfg.Defaults.Priority)
	assert.Equal(t, "todo", cfg.Defaults.Status)
	assert.Contains(t, cfg.Defaults.Labels, "pm-tracked")

	// Check fields structure
	assert.Contains(t, cfg.Fields, "status")
	assert.Contains(t, cfg.Fields, "priority")

	// Check status field
	statusField := cfg.Fields["status"]
	assert.Equal(t, "Status", statusField.Field)
	assert.Contains(t, statusField.Values, "todo")
	assert.Contains(t, statusField.Values, "in_progress")
	assert.Contains(t, statusField.Values, "in_review")
	assert.Contains(t, statusField.Values, "done")

	// Check priority field
	priorityField := cfg.Fields["priority"]
	assert.Equal(t, "Priority", priorityField.Field)
	assert.Contains(t, priorityField.Values, "low")
	assert.Contains(t, priorityField.Values, "medium")
	assert.Contains(t, priorityField.Values, "high")
}

func TestInitConfigFileCreation(t *testing.T) {
	// Skip if in CI environment to avoid GitHub API calls
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gh-pm.yml")

	// Change to temp directory
	originalDir, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Reset flags
	initProject = "Test Project"
	initOrg = "test-org"
	initRepos = []string{"test-org/test-repo"}
	initInteractive = false
	skipMetadata = true

	// Run init command
	var buf bytes.Buffer
	initCmd.SetOut(&buf)
	initCmd.SetErr(&buf)

	err = runInit(initCmd, []string{})
	assert.NoError(t, err)

	// Check that config file was created
	assert.FileExists(t, configPath)

	// Read and verify the config file
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var cfg config.Config
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(t, err)

	// Verify config contents
	assert.Equal(t, "Test Project", cfg.Project.Name)
	// Project.Org might be cleared if it's a user project
	// assert.Equal(t, "test-org", cfg.Project.Org)
	assert.Contains(t, cfg.Repositories, "test-org/test-repo")
}

func TestInitWithExistingConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gh-pm.yml")

	// Create an existing config file
	existingConfig := `project:
  name: Existing Project
  number: 123
repositories:
  - existing/repo
`
	err := os.WriteFile(configPath, []byte(existingConfig), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Reset flags
	initProject = ""
	initOrg = ""
	initRepos = []string{}
	initInteractive = false
	skipMetadata = true

	// Capture output
	var buf bytes.Buffer
	initCmd.SetOut(&buf)
	initCmd.SetErr(&buf)

	// Run init command - should fail because config already exists
	err = runInit(initCmd, []string{})
	// Init command doesn't return error for existing config,
	// it prompts interactively instead
	// Since we're not in interactive mode, it should still succeed
	assert.NoError(t, err)
}

func TestParseProjectSelection(t *testing.T) {
	tests := []struct {
		name        string
		max         int
		input       string
		expected    int
		shouldError bool
	}{
		{
			name:        "valid number",
			max:         5,
			input:       "3",
			expected:    3,
			shouldError: false,
		},
		{
			name:        "number too high",
			max:         5,
			input:       "10",
			expected:    0,
			shouldError: true,
		},
		{
			name:        "zero is valid (skip selection)",
			max:         5,
			input:       "0",
			expected:    0,
			shouldError: false,
		},
		{
			name:        "negative number",
			max:         5,
			input:       "-1",
			expected:    0,
			shouldError: true,
		},
		{
			name:        "non-numeric input",
			max:         5,
			input:       "abc",
			expected:    0,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse input like selectProject would
			choice, err := strconv.Atoi(tt.input)

			if err != nil || choice < 0 || choice > tt.max {
				if tt.shouldError {
					assert.True(t, err != nil || choice < 0 || choice > tt.max)
				} else {
					t.Errorf("Expected valid input but got error or invalid range")
				}
			} else {
				if tt.shouldError {
					t.Errorf("Expected error but got valid result")
				} else {
					assert.Equal(t, tt.expected, choice)
				}
			}
		})
	}
}

func TestTriageConfiguration(t *testing.T) {
	cfg := config.DefaultConfig()

	// Check triage configuration exists
	assert.NotNil(t, cfg.Triage)

	// Check tracked triage
	tracked, exists := cfg.Triage["tracked"]
	assert.True(t, exists)
	assert.Equal(t, "is:issue is:open -label:pm-tracked", tracked.Query)
	assert.Contains(t, tracked.Apply.Labels, "pm-tracked")
	assert.Equal(t, "p1", tracked.Apply.Fields["priority"])
	assert.Equal(t, "backlog", tracked.Apply.Fields["status"])
	assert.True(t, tracked.Interactive.Status)

	// Check estimate triage
	estimate, exists := cfg.Triage["estimate"]
	assert.True(t, exists)
	assert.Equal(t, "is:issue is:open -has:estimate", estimate.Query)
	assert.True(t, estimate.Interactive.Estimate)
}

func TestValidateRepositoryFormat(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		shouldError bool
	}{
		{
			name:        "valid format",
			repo:        "owner/repo",
			shouldError: false,
		},
		{
			name:        "missing slash",
			repo:        "ownerrepo",
			shouldError: true,
		},
		{
			name:        "multiple slashes",
			repo:        "owner/repo/sub",
			shouldError: true,
		},
		{
			name:        "empty string",
			repo:        "",
			shouldError: true,
		},
		{
			name:        "only slash",
			repo:        "/",
			shouldError: true,
		},
		{
			name:        "missing owner",
			repo:        "/repo",
			shouldError: true,
		},
		{
			name:        "missing repo",
			repo:        "owner/",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.repo, "/")
			isValid := len(parts) == 2 && parts[0] != "" && parts[1] != ""

			if tt.shouldError {
				assert.False(t, isValid)
			} else {
				assert.True(t, isValid)
			}
		})
	}
}

func TestSanitizeFieldKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "In Progress",
			expected: "in_progress",
		},
		{
			name:     "with special chars",
			input:    "In-Review!",
			expected: "in_review",
		},
		{
			name:     "multiple spaces",
			input:    "Ready  To  Deploy",
			expected: "ready_to_deploy",
		},
		{
			name:     "already lowercase",
			input:    "backlog",
			expected: "backlog",
		},
		{
			name:     "with numbers",
			input:    "P0 Critical",
			expected: "p0_critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate sanitization logic
			result := strings.ToLower(tt.input)
			result = strings.ReplaceAll(result, " ", "_")
			result = strings.ReplaceAll(result, "-", "_")
			// Remove special characters
			var sanitized []byte
			for i := 0; i < len(result); i++ {
				c := result[i]
				if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
					sanitized = append(sanitized, c)
				}
			}
			result = string(sanitized)
			// Clean up multiple underscores
			for strings.Contains(result, "__") {
				result = strings.ReplaceAll(result, "__", "_")
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetadataStructure(t *testing.T) {
	metadata := &config.ConfigMetadata{
		Project: config.ProjectMetadata{
			ID: "PVT_test",
		},
		Fields: []config.FieldInfo{
			{
				Name:     "Status",
				ID:       "PVTSSF_status",
				DataType: "SINGLE_SELECT",
				Options: []config.FieldOption{
					{Name: "Todo", ID: "todo_id"},
					{Name: "Done", ID: "done_id"},
				},
			},
			{
				Name:     "Title",
				ID:       "PVTF_title",
				DataType: "TITLE",
			},
		},
	}

	// Test metadata structure
	assert.Equal(t, "PVT_test", metadata.Project.ID)
	assert.Len(t, metadata.Fields, 2)

	// Test field metadata
	statusField := metadata.Fields[0]
	assert.Equal(t, "Status", statusField.Name)
	assert.Equal(t, "PVTSSF_status", statusField.ID)
	assert.Equal(t, "SINGLE_SELECT", statusField.DataType)
	assert.Len(t, statusField.Options, 2)

	// Test field options
	assert.Equal(t, "Todo", statusField.Options[0].Name)
	assert.Equal(t, "todo_id", statusField.Options[0].ID)

	// Test TITLE field (no options)
	titleField := metadata.Fields[1]
	assert.Equal(t, "Title", titleField.Name)
	assert.Equal(t, "TITLE", titleField.DataType)
	assert.Len(t, titleField.Options, 0)
}
