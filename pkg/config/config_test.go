package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	assert.NotNil(t, cfg)
	assert.Equal(t, "medium", cfg.Defaults.Priority)
	assert.Equal(t, "Todo", cfg.Defaults.Status)
	assert.Contains(t, cfg.Defaults.Labels, "pm-tracked")
	assert.Equal(t, "table", cfg.Output.Format)
	assert.True(t, cfg.Output.Color)
	assert.Equal(t, "UTC", cfg.Output.Timezone)
	
	// Check field mappings
	assert.Contains(t, cfg.Fields, "priority")
	assert.Contains(t, cfg.Fields, "status")
	
	priorityField := cfg.Fields["priority"]
	assert.Equal(t, "Priority", priorityField.Field)
	assert.Contains(t, priorityField.Values, "low")
	assert.Contains(t, priorityField.Values, "medium")
	assert.Contains(t, priorityField.Values, "high")
	assert.Contains(t, priorityField.Values, "critical")
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	
	// Change to temp directory
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create config
	cfg := &Config{
		Project: ProjectConfig{
			Name:   "Test Project",
			Number: 123,
			Org:    "test-org",
		},
		Repositories: []string{"owner/repo1", "owner/repo2"},
		Defaults: DefaultsConfig{
			Priority: "high",
			Status:   "In Progress",
			Labels:   []string{"test", "automated"},
		},
		Fields: map[string]Field{
			"custom": {
				Field: "CustomField",
				Values: map[string]string{
					"a": "A",
					"b": "B",
				},
			},
		},
		Output: OutputConfig{
			Format:   "json",
			Color:    false,
			Timezone: "PST",
		},
	}
	
	// Save config
	configPath := filepath.Join(tmpDir, ConfigFileName)
	err = cfg.Save(configPath)
	require.NoError(t, err)
	
	// Verify file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
	
	// Load config
	loadedCfg, err := Load()
	require.NoError(t, err)
	
	// Verify loaded config matches original
	assert.Equal(t, cfg.Project.Name, loadedCfg.Project.Name)
	assert.Equal(t, cfg.Project.Number, loadedCfg.Project.Number)
	assert.Equal(t, cfg.Project.Org, loadedCfg.Project.Org)
	assert.Equal(t, cfg.Repositories, loadedCfg.Repositories)
	assert.Equal(t, cfg.Defaults.Priority, loadedCfg.Defaults.Priority)
	assert.Equal(t, cfg.Defaults.Status, loadedCfg.Defaults.Status)
	assert.Equal(t, cfg.Defaults.Labels, loadedCfg.Defaults.Labels)
	assert.Equal(t, cfg.Output.Format, loadedCfg.Output.Format)
	assert.Equal(t, cfg.Output.Color, loadedCfg.Output.Color)
	assert.Equal(t, cfg.Output.Timezone, loadedCfg.Output.Timezone)
}

func TestConfigExists(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	
	// Change to temp directory
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Should not exist initially
	assert.False(t, Exists())
	
	// Create config file
	cfg := DefaultConfig()
	err = cfg.Save(ConfigFileName)
	require.NoError(t, err)
	
	// Should exist now
	assert.True(t, Exists())
}

func TestConfigYAMLFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Project.Name = "My Project"
	cfg.Project.Org = "my-org"
	cfg.Repositories = []string{"my-org/repo1"}
	
	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	
	yamlStr := string(data)
	
	// Verify YAML structure
	assert.Contains(t, yamlStr, "project:")
	assert.Contains(t, yamlStr, "name: My Project")
	assert.Contains(t, yamlStr, "org: my-org")
	assert.Contains(t, yamlStr, "repositories:")
	assert.Contains(t, yamlStr, "- my-org/repo1")
	assert.Contains(t, yamlStr, "defaults:")
	assert.Contains(t, yamlStr, "priority: medium")
	assert.Contains(t, yamlStr, "fields:")
	assert.Contains(t, yamlStr, "output:")
}