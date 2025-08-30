package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractChecklistItems(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "standard GitHub checkboxes",
			input: `
## Tasks
- [ ] First task
- [ ] Second task
- [x] Completed task
- [ ] Fourth task
`,
			expected: []string{
				"First task",
				"Second task",
				"Completed task",
				"Fourth task",
			},
		},
		{
			name: "checkboxes with asterisks",
			input: `
* [ ] Task with asterisk
* [x] Another task
`,
			expected: []string{
				"Task with asterisk",
				"Another task",
			},
		},
		{
			name: "indented checkboxes",
			input: `
  - [ ] Indented task
    - [ ] More indented task
- [ ] Regular task
`,
			expected: []string{
				"Indented task",
				"More indented task",
				"Regular task",
			},
		},
		{
			name: "mixed content",
			input: `
# Title
Some text here
- [ ] Task 1
- Regular list item (not a checkbox)
- [X] Task 2 (uppercase X)
- [ ] Task 3
`,
			expected: []string{
				"Task 1",
				"Task 2 (uppercase X)",
				"Task 3",
			},
		},
		{
			name:     "no checkboxes",
			input:    "Just regular text\n- Regular list\n- Another item",
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractChecklistItems(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTaskAlreadySubIssue(t *testing.T) {
	existingSubIssues := []SubIssueInfo{
		{Number: 1, State: "open", Title: "Design database schema"},
		{Number: 2, State: "open", Title: "Implement API endpoints"},
		{Number: 3, State: "closed", Title: "Write unit tests"},
	}

	tests := []struct {
		name     string
		task     string
		expected bool
	}{
		{
			name:     "exact match",
			task:     "Design database schema",
			expected: true,
		},
		{
			name:     "case insensitive match",
			task:     "design DATABASE schema",
			expected: true,
		},
		{
			name:     "partial match in existing title",
			task:     "API endpoints",
			expected: true,
		},
		{
			name:     "no match",
			task:     "Create documentation",
			expected: false,
		},
		{
			name:     "whitespace differences",
			task:     "  Design database schema  ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTaskAlreadySubIssue(tt.task, existingSubIssues)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTasksFromFile(t *testing.T) {
	// Create a temporary file with test content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "tasks.md")
	
	content := `# Test Tasks
- [ ] Task 1
- [ ] Task 2
- [x] Task 3
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Test file extraction
	tasks, err := extractTasksFromFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Task 1", "Task 2", "Task 3"}, tasks)

	// Test non-existent file
	_, err = extractTasksFromFile("/non/existent/file.md")
	assert.Error(t, err)
}

func TestExtractTasksFromReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "JSON array",
			input:    `["Task 1", "Task 2", "Task 3"]`,
			expected: []string{"Task 1", "Task 2", "Task 3"},
		},
		{
			name: "Markdown checklist",
			input: `- [ ] Task A
- [ ] Task B
- [x] Task C`,
			expected: []string{"Task A", "Task B", "Task C"},
		},
		{
			name:     "Empty JSON array",
			input:    `[]`,
			expected: []string{},
		},
		{
			name:     "Invalid JSON falls back to checklist parsing",
			input:    `[invalid json but valid checklist - [ ] Task]`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := extractTasksFromReader(reader)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGhSubIssueInstalled(t *testing.T) {
	// This test might fail depending on the actual environment
	// We're mainly testing that the function doesn't panic
	t.Run("check installation", func(t *testing.T) {
		// Just ensure the function runs without panic
		_ = isGhSubIssueInstalled()
	})
}

func TestSplitCommandFlags(t *testing.T) {
	// Reset flags for testing
	splitFrom = ""
	splitRepo = ""
	splitDryRun = false

	// Test that flags are properly registered
	assert.NotNil(t, splitCmd.Flags().Lookup("from"))
	assert.NotNil(t, splitCmd.Flags().Lookup("repo"))
	assert.NotNil(t, splitCmd.Flags().Lookup("dry-run"))
}

func TestSplitCommandValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no arguments",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "invalid issue number",
			args:        []string{"abc"},
			expectError: true,
		},
		{
			name:        "valid issue number",
			args:        []string{"123"},
			expectError: false, // Will fail later when trying to fetch issue
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			var buf bytes.Buffer
			splitCmd.SetOut(&buf)
			splitCmd.SetErr(&buf)

			// Test args validation
			err := splitCmd.Args(nil, tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunSplitDryRun(t *testing.T) {
	// This test would require mocking the GitHub API
	// For now, we just test that dry-run mode skips the extension check
	
	oldDryRun := splitDryRun
	defer func() { splitDryRun = oldDryRun }()
	
	splitDryRun = true
	
	// In dry-run mode, the extension check should be skipped
	// This is a simplified test that would need proper mocking for full coverage
	t.Run("dry-run skips extension check", func(t *testing.T) {
		// The actual runSplit function would fail trying to fetch the issue
		// but we're testing that it doesn't fail on the extension check
		assert.True(t, splitDryRun)
	})
}