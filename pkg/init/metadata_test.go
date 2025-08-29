package init

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/project"
)

func TestNormalizeOptionKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"todo lowercase", "todo", "todo"},
		{"todo uppercase", "TODO", "todo"},
		{"todo with space", "To Do", "todo"},
		{"todo with underscore", "to_do", "todo"},
		{"backlog", "Backlog", "todo"},
		{"in progress", "In Progress", "in_progress"},
		{"doing", "Doing", "in_progress"},
		{"in development", "In Development", "in_progress"},
		{"in review", "In Review", "in_review"},
		{"reviewing", "Reviewing", "in_review"},
		{"done", "Done", "done"},
		{"completed", "Completed", "done"},
		{"low priority", "Low", "low"},
		{"p3 priority", "P3", "low"},
		{"medium priority", "Medium", "medium"},
		{"normal priority", "Normal", "medium"},
		{"p2 priority", "P2", "medium"},
		{"high priority", "High", "high"},
		{"p1 priority", "P1", "high"},
		{"critical priority", "Critical", "critical"},
		{"urgent priority", "Urgent", "critical"},
		{"p0 priority", "P0", "critical"},
		{"custom status", "Custom Status", "custom_status"},
		{"with hyphens", "in-progress", "in_progress"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeOptionKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetadataManager_BuildMetadata(t *testing.T) {
	proj := &project.Project{
		ID:     "PVT_test123",
		Number: 1,
		Title:  "Test Project",
	}
	
	fields := []project.Field{
		{
			ID:       "FIELD_status",
			Name:     "Status",
			DataType: "SINGLE_SELECT",
			Options: []project.FieldOption{
				{ID: "opt_todo", Name: "Todo"},
				{ID: "opt_progress", Name: "In Progress"},
				{ID: "opt_done", Name: "Done"},
			},
		},
		{
			ID:       "FIELD_priority",
			Name:     "Priority",
			DataType: "SINGLE_SELECT",
			Options: []project.FieldOption{
				{ID: "opt_low", Name: "Low"},
				{ID: "opt_medium", Name: "Medium"},
				{ID: "opt_high", Name: "High"},
			},
		},
		{
			ID:       "FIELD_text",
			Name:     "Description",
			DataType: "TEXT",
			Options:  nil,
		},
	}
	
	manager := &MetadataManager{}
	metadata, err := manager.BuildMetadata(proj, fields)
	
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "PVT_test123", metadata.Project.ID)
	
	// Check Status field metadata
	assert.NotNil(t, metadata.Fields.Status)
	assert.Equal(t, "FIELD_status", metadata.Fields.Status.ID)
	assert.Equal(t, "opt_todo", metadata.Fields.Status.Options["todo"])
	assert.Equal(t, "opt_progress", metadata.Fields.Status.Options["in_progress"])
	assert.Equal(t, "opt_done", metadata.Fields.Status.Options["done"])
	
	// Check Priority field metadata
	assert.NotNil(t, metadata.Fields.Priority)
	assert.Equal(t, "FIELD_priority", metadata.Fields.Priority.ID)
	assert.Equal(t, "opt_low", metadata.Fields.Priority.Options["low"])
	assert.Equal(t, "opt_medium", metadata.Fields.Priority.Options["medium"])
	assert.Equal(t, "opt_high", metadata.Fields.Priority.Options["high"])
}

func TestMetadataManager_BuildMetadata_NoSingleSelectFields(t *testing.T) {
	proj := &project.Project{
		ID:     "PVT_test456",
		Number: 2,
		Title:  "Test Project 2",
	}
	
	fields := []project.Field{
		{
			ID:       "FIELD_text",
			Name:     "Description",
			DataType: "TEXT",
		},
		{
			ID:       "FIELD_number",
			Name:     "Count",
			DataType: "NUMBER",
		},
	}
	
	manager := &MetadataManager{}
	metadata, err := manager.BuildMetadata(proj, fields)
	
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "PVT_test456", metadata.Project.ID)
	assert.Nil(t, metadata.Fields.Status)
	assert.Nil(t, metadata.Fields.Priority)
}

func TestMetadataManager_BuildMetadata_NilProject(t *testing.T) {
	manager := &MetadataManager{}
	metadata, err := manager.BuildMetadata(nil, []project.Field{})
	
	assert.Error(t, err)
	assert.Nil(t, metadata)
	assert.Contains(t, err.Error(), "project is nil")
}

func TestMetadataManager_FetchProjectMetadata(t *testing.T) {
	manager := &MetadataManager{}
	
	// Test with a sample project ID
	projectID := "PVT_sample789"
	metadata, err := manager.FetchProjectMetadata(projectID)
	
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, projectID, metadata.ID)
}

func TestMetadataManager_BuildMetadata_PartialFields(t *testing.T) {
	proj := &project.Project{
		ID:     "PVT_partial",
		Number: 3,
		Title:  "Partial Project",
	}
	
	// Only Status field, no Priority
	fields := []project.Field{
		{
			ID:       "FIELD_status_only",
			Name:     "Status",
			DataType: "SINGLE_SELECT",
			Options: []project.FieldOption{
				{ID: "opt_new", Name: "New"},
				{ID: "opt_active", Name: "Active"},
				{ID: "opt_closed", Name: "Closed"},
			},
		},
	}
	
	manager := &MetadataManager{}
	metadata, err := manager.BuildMetadata(proj, fields)
	
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.NotNil(t, metadata.Fields.Status)
	assert.Nil(t, metadata.Fields.Priority)
	
	// Check that non-standard status names are normalized appropriately
	assert.Equal(t, "opt_new", metadata.Fields.Status.Options["new"])
	assert.Equal(t, "opt_active", metadata.Fields.Status.Options["active"])
	// "Closed" gets normalized to "done" by normalizeOptionKey
	assert.Equal(t, "opt_closed", metadata.Fields.Status.Options["done"])
}

func TestConfigMetadata_Integration(t *testing.T) {
	// Test that the metadata integrates properly with config
	metadata := &config.ConfigMetadata{
		Project: config.ProjectMetadata{
			ID: "PVT_integration",
		},
		Fields: config.FieldsMetadata{
			Status: &config.FieldMetadata{
				ID: "FIELD_status_int",
				Options: map[string]string{
					"todo": "opt_todo_int",
					"done": "opt_done_int",
				},
			},
		},
	}
	
	cfg := config.DefaultConfig()
	cfg.Metadata = metadata
	
	// Verify metadata is accessible
	loadedMetadata, err := cfg.LoadMetadata()
	assert.NoError(t, err)
	assert.NotNil(t, loadedMetadata)
	assert.Equal(t, "PVT_integration", loadedMetadata.Project.ID)
	assert.Equal(t, "opt_todo_int", loadedMetadata.Fields.Status.Options["todo"])
}