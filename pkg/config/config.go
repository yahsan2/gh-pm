package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const ConfigFileName = ".gh-pm.yml"

// Config represents the project configuration
type Config struct {
	Project      ProjectConfig             `yaml:"project"`
	Repositories []string                  `yaml:"repositories"`
	Defaults     DefaultsConfig            `yaml:"defaults"`
	Fields       map[string]Field          `yaml:"fields"`
	Triage       map[string]TriageConfig   `yaml:"triage,omitempty"`
	Metadata     *ConfigMetadata           `yaml:"metadata,omitempty"`
}

// ProjectConfig represents project settings
type ProjectConfig struct {
	Name   string `yaml:"name"`
	Number int    `yaml:"number,omitempty"`
	Org    string `yaml:"org,omitempty"`
	Owner  string `yaml:"owner,omitempty"` // Project owner username (for URL generation)
}

// DefaultsConfig represents default values
type DefaultsConfig struct {
	Priority string   `yaml:"priority"`
	Status   string   `yaml:"status"`
	Labels   []string `yaml:"labels"`
}

// Field represents a custom field mapping
type Field struct {
	Field  string            `yaml:"field"`
	Values map[string]string `yaml:"values"`
}

// TriageConfig represents a triage configuration
type TriageConfig struct {
	Query       string            `yaml:"query"`
	Apply       TriageApply       `yaml:"apply"`
	Interactive TriageInteractive `yaml:"interactive,omitempty"`
}

// TriageApply represents what to apply during triage
type TriageApply struct {
	Labels []string             `yaml:"labels,omitempty"`
	Fields map[string]string    `yaml:"fields,omitempty"`
}

// TriageInteractive represents interactive options for triage
type TriageInteractive struct {
	Status   bool `yaml:"status,omitempty"`
	Estimate bool `yaml:"estimate,omitempty"`
}


// ConfigMetadata represents cached project metadata
type ConfigMetadata struct {
	Project ProjectMetadata `yaml:"project"`
	Fields  FieldsMetadata  `yaml:"fields"`
}

// ProjectMetadata represents cached project IDs
type ProjectMetadata struct {
	ID string `yaml:"id"` // Node ID (e.g., "PVT_kwHOAAlRwM4A8arc")
}

// FieldsMetadata represents cached field metadata as a dynamic map
type FieldsMetadata map[string]*FieldMetadata

// FieldMetadata represents cached field IDs and options
type FieldMetadata struct {
	ID      string            `yaml:"id"`      // Field ID
	Options map[string]string `yaml:"options"` // name -> option ID
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Project: ProjectConfig{
			Name: "",
		},
		Repositories: []string{},
		Defaults: DefaultsConfig{
			Priority: "medium",
			Status:   "todo",
			Labels:   []string{"pm-tracked"},
		},
		Fields: map[string]Field{
			"priority": {
				Field: "Priority",
				Values: map[string]string{
					"low":      "Low",
					"medium":   "Medium",
					"high":     "High",
					"critical": "Critical",
				},
			},
			"status": {
				Field: "Status",
				Values: map[string]string{
					"todo":        "Todo",
					"in_progress": "In Progress",
					"in_review":   "In Review",
					"done":        "Done",
				},
			},
		},
		Triage: map[string]TriageConfig{
			"tracked": {
				Query: "is:issue is:open -label:pm-tracked",
				Apply: TriageApply{
					Labels: []string{"pm-tracked"},
					Fields: map[string]string{
						"status":   "backlog",
						"priority": "p1",
					},
				},
				Interactive: TriageInteractive{
					Status: true,
				},
			},
			"estimate": {
				Query: "is:issue is:open -has:estimate",
				Apply: TriageApply{},
				Interactive: TriageInteractive{
					Estimate: true,
				},
			},
		},
	}
}

// Load loads configuration from file
func Load() (*Config, error) {
	configPath := findConfigFile()
	if configPath == "" {
		return nil, fmt.Errorf("configuration file %s not found", ConfigFileName)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// findConfigFile searches for config file in current and parent directories
func findConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// Exists checks if configuration file exists
func Exists() bool {
	return findConfigFile() != ""
}

// SaveWithMetadata saves configuration with metadata to file
func (c *Config) SaveWithMetadata(path string) error {
	// Use the same implementation as Save since metadata is already part of Config
	return c.Save(path)
}

// LoadMetadata loads just the metadata section from configuration
func (c *Config) LoadMetadata() (*ConfigMetadata, error) {
	if c.Metadata == nil {
		return nil, fmt.Errorf("no metadata found in configuration")
	}
	return c.Metadata, nil
}

// LoadConfig loads configuration from file with enhanced error handling
func LoadConfig() (*Config, error) {
	configPath := findConfigFile()
	if configPath == "" {
		return nil, fmt.Errorf("configuration file %s not found in current or parent directories", ConfigFileName)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check required fields
	if c.Project.Name == "" && c.Project.Number == 0 {
		return fmt.Errorf("project name or number is required")
	}
	
	if len(c.Repositories) == 0 {
		return fmt.Errorf("at least one repository must be configured")
	}
	
	// Validate repository format
	for _, repo := range c.Repositories {
		if !isValidRepository(repo) {
			return fmt.Errorf("invalid repository format '%s': must be 'owner/repo'", repo)
		}
	}
	
	// Validate field mappings
	if c.Fields != nil {
		for name, field := range c.Fields {
			if field.Field == "" {
				return fmt.Errorf("field name is required for '%s'", name)
			}
			if len(field.Values) == 0 {
				return fmt.Errorf("at least one value mapping is required for field '%s'", name)
			}
		}
	}
	
	// Validate defaults against field mappings
	if c.Defaults.Priority != "" {
		if priority, ok := c.Fields["priority"]; ok {
			if _, exists := priority.Values[c.Defaults.Priority]; !exists {
				return fmt.Errorf("default priority '%s' is not defined in field mappings", c.Defaults.Priority)
			}
		}
	}
	
	if c.Defaults.Status != "" {
		if status, ok := c.Fields["status"]; ok {
			if _, exists := status.Values[c.Defaults.Status]; !exists {
				return fmt.Errorf("default status '%s' is not defined in field mappings", c.Defaults.Status)
			}
		}
	}
	
	return nil
}

// GetProjectID returns the cached project ID if available
func (c *Config) GetProjectID() string {
	if c.Metadata != nil && c.Metadata.Project.ID != "" {
		return c.Metadata.Project.ID
	}
	return ""
}

// SetProjectID sets the project ID in metadata
func (c *Config) SetProjectID(id string) {
	if c.Metadata == nil {
		c.Metadata = &ConfigMetadata{}
	}
	c.Metadata.Project.ID = id
}

// GetFieldMetadata returns metadata for a specific field
func (c *Config) GetFieldMetadata(fieldName string) *FieldMetadata {
	if c.Metadata == nil || c.Metadata.Fields == nil {
		return nil
	}
	
	if fieldMeta, exists := c.Metadata.Fields[fieldName]; exists {
		return fieldMeta
	}
	
	return nil
}

// isValidRepository checks if a repository string is in the correct format
func isValidRepository(repo string) bool {
	parts := strings.Split(repo, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// FindConfigPath returns the path to the configuration file
func FindConfigPath() string {
	return findConfigFile()
}