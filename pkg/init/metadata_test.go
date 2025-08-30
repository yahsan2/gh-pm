package init

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/project"
)


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
	
	// Check all fields are in metadata
	assert.Len(t, metadata.Fields, 3) // Status, Priority, and Title
	
	// Check Status field metadata
	var statusField *config.FieldInfo
	for _, field := range metadata.Fields {
		if field.Name == "Status" {
			statusField = &field
			break
		}
	}
	assert.NotNil(t, statusField)
	assert.Equal(t, "FIELD_status", statusField.ID)
	assert.Len(t, statusField.Options, 3)
	
	// Check Priority field metadata
	var priorityField *config.FieldInfo
	for _, field := range metadata.Fields {
		if field.Name == "Priority" {
			priorityField = &field
			break
		}
	}
	assert.NotNil(t, priorityField)
	assert.Equal(t, "FIELD_priority", priorityField.ID)
	assert.Len(t, priorityField.Options, 3)
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
	assert.Len(t, metadata.Fields, 2) // Title and Body fields
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
	
	// Check that the metadata contains the Status field
	var statusField *config.FieldInfo
	for _, field := range metadata.Fields {
		if field.Name == "Status" {
			statusField = &field
			break
		}
	}
	assert.NotNil(t, statusField)
	
	// Check that non-standard status names exist in options
	optionNames := make(map[string]bool)
	for _, opt := range statusField.Options {
		optionNames[opt.Name] = true
	}
	assert.True(t, optionNames["New"])
	assert.True(t, optionNames["Active"])
	assert.True(t, optionNames["Closed"]) // "Closed" remains as-is, not mapped to "Done"
}

func TestConfigMetadata_Integration(t *testing.T) {
	// Test that the metadata integrates properly with config
	metadata := &config.ConfigMetadata{
		Project: config.ProjectMetadata{
			ID: "PVT_integration",
		},
		Fields: []config.FieldInfo{
			{
				Name:     "Status",
				ID:       "FIELD_status_int",
				DataType: "SINGLE_SELECT",
				Options: []config.FieldOption{
					{Name: "Todo", ID: "opt_todo_int"},
					{Name: "Done", ID: "opt_done_int"},
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
	
	// Verify field metadata can be retrieved from metadata
	fieldMeta := cfg.GetFieldMetadata("Status")
	assert.NotNil(t, fieldMeta)
	assert.Equal(t, "opt_todo_int", fieldMeta.Options["todo"])
}