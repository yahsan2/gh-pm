package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ConfigFileName = ".gh-pm.yml"

// Config represents the project configuration
type Config struct {
	Project      ProjectConfig      `yaml:"project"`
	Repositories []string           `yaml:"repositories"`
	Defaults     DefaultsConfig     `yaml:"defaults"`
	Fields       map[string]Field   `yaml:"fields"`
	Output       OutputConfig       `yaml:"output"`
}

// ProjectConfig represents project settings
type ProjectConfig struct {
	Name   string `yaml:"name"`
	Number int    `yaml:"number,omitempty"`
	Org    string `yaml:"org,omitempty"`
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

// OutputConfig represents output preferences
type OutputConfig struct {
	Format   string `yaml:"format"`
	Color    bool   `yaml:"color"`
	Timezone string `yaml:"timezone"`
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
			Status:   "Todo",
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
		Output: OutputConfig{
			Format:   "table",
			Color:    true,
			Timezone: "UTC",
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