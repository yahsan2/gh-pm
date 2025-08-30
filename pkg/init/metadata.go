package init

import (
	"fmt"
	"strings"

	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/project"
)

// MetadataManager handles project metadata fetching and management
type MetadataManager struct {
	client *project.Client
}

// NewMetadataManager creates a new MetadataManager instance
func NewMetadataManager(client *project.Client) *MetadataManager {
	return &MetadataManager{
		client: client,
	}
}

// FetchProjectMetadata fetches project metadata including node ID
func (m *MetadataManager) FetchProjectMetadata(projectID string) (*config.ProjectMetadata, error) {
	// The projectID here is already the node ID from the Project struct
	return &config.ProjectMetadata{
		ID: projectID,
	}, nil
}

// FetchFieldMetadata fetches field metadata including field ID and option IDs
func (m *MetadataManager) FetchFieldMetadata(projectID string, fieldName string) (*config.FieldMetadata, error) {
	// Get project fields
	fields, err := m.client.GetProjectFields(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project fields: %w", err)
	}
	
	// Find the specific field
	for _, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
			// Check if it's a single select field with options
			if field.DataType != "SINGLE_SELECT" || len(field.Options) == 0 {
				continue
			}
			
			// Build option ID mapping
			options := make(map[string]string)
			for _, opt := range field.Options {
				// Map option names to their IDs
				// Use lowercase keys for consistent mapping
				key := strings.ToLower(opt.Name)
				key = strings.ReplaceAll(key, " ", "_")
				options[key] = opt.ID
			}
			
			return &config.FieldMetadata{
				ID:      field.ID,
				Options: options,
			}, nil
		}
	}
	
	return nil, fmt.Errorf("field '%s' not found or is not a single select field", fieldName)
}

// BuildMetadata builds complete metadata structure for a project
func (m *MetadataManager) BuildMetadata(proj *project.Project, fields []project.Field) (*config.ConfigMetadata, error) {
	if proj == nil {
		return nil, fmt.Errorf("project is nil")
	}
	
	metadata := &config.ConfigMetadata{
		Project: config.ProjectMetadata{
			ID: proj.ID,
		},
		Fields: make([]config.FieldInfo, 0, len(fields)),
	}
	
	
	// Cache ALL fields information for future reference
	for _, field := range fields {
		// Store all field information
		fieldInfo := config.FieldInfo{
			Name:     field.Name,
			ID:       field.ID,
			DataType: field.DataType,
		}
		
		// If it's a single select field, also store the options
		if field.DataType == "SINGLE_SELECT" && len(field.Options) > 0 {
			fieldInfo.Options = make([]config.FieldOption, 0, len(field.Options))
			for _, opt := range field.Options {
				fieldInfo.Options = append(fieldInfo.Options, config.FieldOption{
					Name: opt.Name,
					ID:   opt.ID,
				})
			}
			
		}
		
		metadata.Fields = append(metadata.Fields, fieldInfo)
	}
	
	return metadata, nil
}

