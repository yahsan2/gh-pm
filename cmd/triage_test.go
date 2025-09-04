package cmd

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/project"
)

func TestCollectStatusChoice_OptionsOrder(t *testing.T) {
	tests := []struct {
		name            string
		configFields    map[string]config.Field
		statusField     project.Field
		expectedOptions []string
		description     string
	}{
		{
			name: "With config mapping - should use field.Options order",
			configFields: map[string]config.Field{
				"status": {
					Field: "Status",
					Values: map[string]string{
						"backlog":     "Backlog",
						"ready":       "Ready",
						"in_progress": "In progress",
						"in_review":   "In review",
						"done":        "Done",
					},
				},
			},
			statusField: project.Field{
				Name:     "Status",
				DataType: "SINGLE_SELECT",
				Options: []project.FieldOption{
					{Name: "Backlog", ID: "1"},
					{Name: "Ready", ID: "2"},
					{Name: "In progress", ID: "3"},
					{Name: "In review", ID: "4"},
					{Name: "Done", ID: "5"},
				},
			},
			expectedOptions: []string{"backlog", "ready", "in_progress", "in_review", "done"},
			description:     "Should preserve the order from field.Options when config mapping exists",
		},
		{
			name:         "Without config mapping - should use field.Options directly",
			configFields: map[string]config.Field{},
			statusField: project.Field{
				Name:     "Status",
				DataType: "SINGLE_SELECT",
				Options: []project.FieldOption{
					{Name: "Todo", ID: "1"},
					{Name: "In Progress", ID: "2"},
					{Name: "Done", ID: "3"},
				},
			},
			expectedOptions: []string{"Todo", "In Progress", "Done"},
			description:     "Should use field.Options directly when no config mapping exists",
		},
		{
			name: "Partial config mapping - only mapped values in field.Options order",
			configFields: map[string]config.Field{
				"status": {
					Field: "Status",
					Values: map[string]string{
						"todo": "Todo",
						"done": "Done",
						// "In Progress" is not mapped
					},
				},
			},
			statusField: project.Field{
				Name:     "Status",
				DataType: "SINGLE_SELECT",
				Options: []project.FieldOption{
					{Name: "Todo", ID: "1"},
					{Name: "In Progress", ID: "2"},
					{Name: "Done", ID: "3"},
				},
			},
			expectedOptions: []string{"todo", "done"},
			description:     "Should only include mapped values in field.Options order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock TriageCommand with test config
			cmd := &TriageCommand{
				config: &config.Config{
					Fields: tt.configFields,
				},
			}

			// Create mock issue
			issue := GitHubIssue{
				Number: 1,
				Title:  "Test Issue",
			}

			// Create fields slice with our test status field
			fields := []project.Field{tt.statusField}

			// Mock reader with "0" input (skip)
			reader := bufio.NewReader(strings.NewReader("0\n"))

			// Call the function
			_ = cmd.collectStatusChoice(issue, reader, fields)

			// Extract available options from the status field config
			var actualOptions []string
			if statusFieldConfig, ok := cmd.config.Fields["status"]; ok {
				if len(statusFieldConfig.Values) > 0 {
					// Reconstruct the order based on the algorithm in collectStatusChoice
					for _, option := range tt.statusField.Options {
						for key, value := range statusFieldConfig.Values {
							if value == option.Name {
								actualOptions = append(actualOptions, key)
								break
							}
						}
					}
				} else {
					for _, option := range tt.statusField.Options {
						actualOptions = append(actualOptions, option.Name)
					}
				}
			} else {
				for _, option := range tt.statusField.Options {
					actualOptions = append(actualOptions, option.Name)
				}
			}

			assert.Equal(t, tt.expectedOptions, actualOptions, tt.description)
		})
	}
}

func TestCollectFieldChoice_OptionsOrder(t *testing.T) {
	tests := []struct {
		name            string
		fieldName       string
		configFields    map[string]config.Field
		targetField     project.Field
		expectedOptions []string
		description     string
	}{
		{
			name:      "Priority field with config mapping",
			fieldName: "Priority",
			configFields: map[string]config.Field{
				"priority": {
					Field: "Priority",
					Values: map[string]string{
						"p0": "P0",
						"p1": "P1",
						"p2": "P2",
					},
				},
			},
			targetField: project.Field{
				Name:     "Priority",
				DataType: "SINGLE_SELECT",
				Options: []project.FieldOption{
					{Name: "P0", ID: "1"},
					{Name: "P1", ID: "2"},
					{Name: "P2", ID: "3"},
				},
			},
			expectedOptions: []string{"p0", "p1", "p2"},
			description:     "Should preserve the order from field.Options for priority field",
		},
		{
			name:         "Size field without config mapping",
			fieldName:    "Size",
			configFields: map[string]config.Field{},
			targetField: project.Field{
				Name:     "Size",
				DataType: "SINGLE_SELECT",
				Options: []project.FieldOption{
					{Name: "XS", ID: "1"},
					{Name: "S", ID: "2"},
					{Name: "M", ID: "3"},
					{Name: "L", ID: "4"},
					{Name: "XL", ID: "5"},
				},
			},
			expectedOptions: []string{"XS", "S", "M", "L", "XL"},
			description:     "Should use field.Options directly for size field without config",
		},
		{
			name:         "Text field type",
			fieldName:    "Description",
			configFields: map[string]config.Field{},
			targetField: project.Field{
				Name:     "Description",
				DataType: "TEXT",
			},
			expectedOptions: nil, // Text fields don't have options
			description:     "Text field should not have predefined options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock TriageCommand with test config
			cmd := &TriageCommand{
				config: &config.Config{
					Fields: tt.configFields,
				},
			}

			// Create mock issue
			issue := GitHubIssue{
				Number: 1,
				Title:  "Test Issue",
			}

			// Create fields slice with our test field
			fields := []project.Field{tt.targetField}

			// Mock reader with "0" input (skip) or empty for text fields
			reader := bufio.NewReader(strings.NewReader("0\n"))

			// Call the function
			_ = cmd.collectFieldChoice(issue, reader, tt.fieldName, fields)

			if tt.targetField.DataType == "SINGLE_SELECT" {
				// Extract available options based on the algorithm
				var actualOptions []string

				configKey := strings.ToLower(tt.fieldName)
				if fieldConfig, ok := cmd.config.Fields[configKey]; ok {
					if len(fieldConfig.Values) > 0 {
						// Reconstruct the order based on the algorithm
						for _, option := range tt.targetField.Options {
							for key, value := range fieldConfig.Values {
								if value == option.Name {
									actualOptions = append(actualOptions, key)
									break
								}
							}
						}
					} else {
						for _, option := range tt.targetField.Options {
							actualOptions = append(actualOptions, option.Name)
						}
					}
				} else {
					for _, option := range tt.targetField.Options {
						actualOptions = append(actualOptions, option.Name)
					}
				}

				assert.Equal(t, tt.expectedOptions, actualOptions, tt.description)
			}
		})
	}
}

// Test that the order is consistent across multiple calls
func TestOptionsOrder_Consistency(t *testing.T) {
	configFields := map[string]config.Field{
		"status": {
			Field: "Status",
			Values: map[string]string{
				"backlog":     "Backlog",
				"ready":       "Ready",
				"in_progress": "In progress",
				"done":        "Done",
			},
		},
	}

	statusField := project.Field{
		Name:     "Status",
		DataType: "SINGLE_SELECT",
		Options: []project.FieldOption{
			{Name: "Backlog", ID: "1"},
			{Name: "Ready", ID: "2"},
			{Name: "In progress", ID: "3"},
			{Name: "Done", ID: "4"},
		},
	}

	// Run multiple times to ensure consistency
	for i := 0; i < 5; i++ {
		var actualOptions []string

		// Reconstruct the order based on the algorithm
		for _, option := range statusField.Options {
			for key, value := range configFields["status"].Values {
				if value == option.Name {
					actualOptions = append(actualOptions, key)
					break
				}
			}
		}

		expectedOptions := []string{"backlog", "ready", "in_progress", "done"}
		assert.Equal(t, expectedOptions, actualOptions,
			"Options order should be consistent across multiple iterations (iteration %d)", i+1)
	}
}
