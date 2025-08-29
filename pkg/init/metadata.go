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
				key := normalizeOptionKey(opt.Name)
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
		Fields: config.FieldsMetadata{},
	}
	
	// Process fields to find Status and Priority
	for _, field := range fields {
		if field.DataType != "SINGLE_SELECT" || len(field.Options) == 0 {
			continue
		}
		
		// Build options mapping
		options := make(map[string]string)
		for _, opt := range field.Options {
			key := normalizeOptionKey(opt.Name)
			options[key] = opt.ID
		}
		
		fieldMeta := &config.FieldMetadata{
			ID:      field.ID,
			Options: options,
		}
		
		// Assign to appropriate field based on name
		switch {
		case strings.EqualFold(field.Name, "Status"):
			metadata.Fields.Status = fieldMeta
		case strings.EqualFold(field.Name, "Priority"):
			metadata.Fields.Priority = fieldMeta
		}
	}
	
	return metadata, nil
}

// normalizeOptionKey normalizes option names for consistent mapping
func normalizeOptionKey(name string) string {
	// Convert to lowercase and replace spaces with underscores
	key := strings.ToLower(name)
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "-", "_")
	
	// Map common variations to standard keys
	switch key {
	case "to_do", "todo", "backlog":
		return "todo"
	case "in_progress", "doing", "in_development":
		return "in_progress"
	case "in_review", "reviewing", "review":
		return "in_review"
	case "done", "completed", "complete", "closed":
		return "done"
	case "low", "p3", "p4":
		return "low"
	case "medium", "normal", "p2":
		return "medium"
	case "high", "p1":
		return "high"
	case "critical", "urgent", "p0":
		return "critical"
	default:
		return key
	}
}