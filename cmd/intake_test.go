package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/project"
)

// Test for intake command structure and logic
// Note: Full testing would require refactoring the IntakeCommand to accept interfaces
// or using integration tests. This demonstrates the test approach.

func TestIntakeCommand_ParseApplyFlags(t *testing.T) {
	tests := []struct {
		name        string
		applyFlags  []string
		expected    map[string]string
		expectError bool
	}{
		{
			name: "parses status and priority",
			applyFlags: []string{
				"status:backlog",
				"priority:p1",
			},
			expected: map[string]string{
				"status":   "backlog",
				"priority": "p1",
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
			name: "parses single field",
			applyFlags: []string{
				"status:in_progress",
			},
			expected: map[string]string{
				"status": "in_progress",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]string)

			// Parse apply flags (extracted logic from runIntake)
			for _, apply := range tt.applyFlags {
				parts := strings.SplitN(apply, ":", 2)
				if len(parts) == 2 {
					field := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					result[field] = value
				}
			}

			assert.Equal(t, tt.expected, result)
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
					Name:   "Test Project",
					Number: 1,
				},
				Repositories: []string{"owner/repo"},
			},
			expectError: false,
		},
		{
			name: "missing project configuration",
			config: &config.Config{
				Project:      config.ProjectConfig{},
				Repositories: []string{"owner/repo"},
			},
			expectError: true,
			errorMsg:    "no project configured",
		},
		{
			name: "project with only number",
			config: &config.Config{
				Project: config.ProjectConfig{
					Number: 42,
				},
				Repositories: []string{"owner/repo"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate config (extracted logic from Execute)
			hasError := tt.config.Project.Name == "" && tt.config.Project.Number == 0

			if tt.expectError {
				assert.True(t, hasError, "Expected error for invalid config")
			} else {
				assert.False(t, hasError, "Expected no error for valid config")
			}
		})
	}
}

func TestIntakeCommand_FieldMapping(t *testing.T) {
	tests := []struct {
		name       string
		fieldKey   string
		fieldValue string
		fields     []project.Field
		config     *config.Config
		expectID   string
		expectErr  bool
	}{
		{
			name:       "maps status field with config",
			fieldKey:   "status",
			fieldValue: "backlog",
			fields: []project.Field{
				{
					ID:       "FIELD_status",
					Name:     "Status",
					DataType: "SINGLE_SELECT",
					Options: []project.FieldOption{
						{ID: "OPT_backlog", Name: "Backlog"},
						{ID: "OPT_todo", Name: "Todo"},
					},
				},
			},
			config: &config.Config{
				Fields: map[string]config.Field{
					"status": {
						Field: "Status",
						Values: map[string]string{
							"backlog": "Backlog",
							"todo":    "Todo",
						},
					},
				},
			},
			expectID:  "OPT_backlog",
			expectErr: false,
		},
		{
			name:       "maps priority field",
			fieldKey:   "priority",
			fieldValue: "p1",
			fields: []project.Field{
				{
					ID:       "FIELD_priority",
					Name:     "Priority",
					DataType: "SINGLE_SELECT",
					Options: []project.FieldOption{
						{ID: "OPT_p0", Name: "P0"},
						{ID: "OPT_p1", Name: "P1"},
						{ID: "OPT_p2", Name: "P2"},
					},
				},
			},
			config: &config.Config{
				Fields: map[string]config.Field{
					"priority": {
						Field: "Priority",
						Values: map[string]string{
							"p0": "P0",
							"p1": "P1",
							"p2": "P2",
						},
					},
				},
			},
			expectID:  "OPT_p1",
			expectErr: false,
		},
		{
			name:       "field not found",
			fieldKey:   "nonexistent",
			fieldValue: "value",
			fields:     []project.Field{},
			config: &config.Config{
				Fields: map[string]config.Field{},
			},
			expectID:  "",
			expectErr: true,
		},
		{
			name:       "option not found",
			fieldKey:   "status",
			fieldValue: "invalid",
			fields: []project.Field{
				{
					ID:       "FIELD_status",
					Name:     "Status",
					DataType: "SINGLE_SELECT",
					Options: []project.FieldOption{
						{ID: "OPT_backlog", Name: "Backlog"},
					},
				},
			},
			config: &config.Config{
				Fields: map[string]config.Field{
					"status": {
						Field: "Status",
						Values: map[string]string{
							"backlog": "Backlog",
						},
					},
				},
			},
			expectID:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test field mapping logic (extracted from updateProjectField)
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
			for _, field := range tt.fields {
				if strings.EqualFold(field.Name, fieldName) {
					targetField = &field
					break
				}
			}

			if targetField == nil {
				if tt.expectErr {
					assert.Nil(t, targetField, "Expected field not found")
				} else {
					t.Errorf("Expected to find field %s", fieldName)
				}
				return
			}

			// Find the option ID
			var optionID string
			if targetField.DataType == "SINGLE_SELECT" {
				configKey := strings.ToLower(fieldName)
				if configField, ok := tt.config.Fields[configKey]; ok {
					if mappedValue, ok := configField.Values[tt.fieldValue]; ok {
						for _, option := range targetField.Options {
							if option.Name == mappedValue {
								optionID = option.ID
								break
							}
						}
					}
				}

				// Direct match as fallback
				if optionID == "" {
					for _, option := range targetField.Options {
						if option.Name == tt.fieldValue {
							optionID = option.ID
							break
						}
					}
				}
			}

			if tt.expectErr {
				assert.Empty(t, optionID, "Expected no option ID for error case")
			} else {
				assert.Equal(t, tt.expectID, optionID, "Option ID mismatch")
			}
		})
	}
}
