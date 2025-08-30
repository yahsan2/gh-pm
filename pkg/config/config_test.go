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
	assert.Equal(t, "todo", cfg.Defaults.Status)
	assert.Contains(t, cfg.Defaults.Labels, "pm-tracked")
	
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
}

func TestSaveWithMetadata(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	
	// Change to temp directory
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create config with metadata
	cfg := DefaultConfig()
	cfg.Project.Name = "Test Project"
	cfg.Project.Org = "test-org"
	cfg.Metadata = &ConfigMetadata{
		Project: ProjectMetadata{
			ID: "PVT_kwHOAAlRwM4A8arc",
		},
		Fields: []FieldInfo{
			{
				Name:     "Status",
				ID:       "PVTSSF_lAHOAAlRwM4A8arczgwbDH4",
				DataType: "SINGLE_SELECT",
				Options: []FieldOption{
					{Name: "Todo", ID: "f75ad846"},
					{Name: "In progress", ID: "47fc9ee4"},
					{Name: "Done", ID: "98236657"},
				},
			},
			{
				Name:     "Priority",
				ID:       "PVTSSF_lAHOAAlRwM4A8arczgwbDH8",
				DataType: "SINGLE_SELECT",
				Options: []FieldOption{
					{Name: "Low", ID: "abc12345"},
					{Name: "Medium", ID: "def67890"},
					{Name: "High", ID: "ghi13579"},
					{Name: "Critical", ID: "jkl24680"},
				},
			},
		},
	}
	
	// Save with metadata
	configPath := filepath.Join(tmpDir, ConfigFileName)
	err = cfg.SaveWithMetadata(configPath)
	require.NoError(t, err)
	
	// Load and verify
	loadedCfg, err := Load()
	require.NoError(t, err)
	
	// Verify metadata was saved and loaded correctly
	assert.NotNil(t, loadedCfg.Metadata)
	assert.Equal(t, "PVT_kwHOAAlRwM4A8arc", loadedCfg.Metadata.Project.ID)
	assert.Len(t, loadedCfg.Metadata.Fields, 2)
	
	// Find Status field
	var statusField *FieldInfo
	for _, field := range loadedCfg.Metadata.Fields {
		if field.Name == "Status" {
			statusField = &field
			break
		}
	}
	assert.NotNil(t, statusField)
	assert.Equal(t, "PVTSSF_lAHOAAlRwM4A8arczgwbDH4", statusField.ID)
	assert.Len(t, statusField.Options, 3)
	
	// Find Priority field
	var priorityField *FieldInfo
	for _, field := range loadedCfg.Metadata.Fields {
		if field.Name == "Priority" {
			priorityField = &field
			break
		}
	}
	assert.NotNil(t, priorityField)
	assert.Equal(t, "PVTSSF_lAHOAAlRwM4A8arczgwbDH8", priorityField.ID)
	assert.Len(t, priorityField.Options, 4)
}

func TestConfigWithoutMetadata(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	
	// Change to temp directory
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create config without metadata
	cfg := DefaultConfig()
	cfg.Project.Name = "Test Project"
	
	// Save without metadata
	configPath := filepath.Join(tmpDir, ConfigFileName)
	err = cfg.Save(configPath)
	require.NoError(t, err)
	
	// Load and verify
	loadedCfg, err := Load()
	require.NoError(t, err)
	
	// Verify metadata is nil
	assert.Nil(t, loadedCfg.Metadata)
	
	// Verify other fields are intact
	assert.Equal(t, "Test Project", loadedCfg.Project.Name)
}

func TestLoadMetadata(t *testing.T) {
	cfg := DefaultConfig()
	
	// Test with no metadata
	metadata, err := cfg.LoadMetadata()
	assert.Error(t, err)
	assert.Nil(t, metadata)
	
	// Test with metadata
	cfg.Metadata = &ConfigMetadata{
		Project: ProjectMetadata{
			ID: "PVT_test",
		},
	}
	
	metadata, err = cfg.LoadMetadata()
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "PVT_test", metadata.Project.ID)
}

func TestMetadataYAMLFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Metadata = &ConfigMetadata{
		Project: ProjectMetadata{
			ID: "PVT_kwHOAAlRwM4A8arc",
		},
		Fields: []FieldInfo{
			{
				Name:     "Status",
				ID:       "PVTSSF_status",
				DataType: "SINGLE_SELECT",
				Options: []FieldOption{
					{Name: "Todo", ID: "opt_todo"},
					{Name: "Done", ID: "opt_done"},
				},
			},
		},
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	
	yamlStr := string(data)
	
	// Verify metadata section in YAML
	assert.Contains(t, yamlStr, "metadata:")
	assert.Contains(t, yamlStr, "project:")
	assert.Contains(t, yamlStr, "id: PVT_kwHOAAlRwM4A8arc")
	assert.Contains(t, yamlStr, "fields:")
	assert.Contains(t, yamlStr, "- name: Status")
	assert.Contains(t, yamlStr, "id: PVTSSF_status")
	assert.Contains(t, yamlStr, "options:")
	assert.Contains(t, yamlStr, "- name: Todo")
	assert.Contains(t, yamlStr, "id: opt_todo")
	assert.Contains(t, yamlStr, "- name: Done")
	assert.Contains(t, yamlStr, "id: opt_done")
}

func TestBackwardCompatibility(t *testing.T) {
	// Create a YAML string without metadata (old format)
	oldFormatYAML := `
project:
  name: Old Project
  org: old-org
repositories:
  - old-org/repo
defaults:
  priority: low
  status: Todo
  labels:
    - old-label
fields:
  status:
    field: Status
    values:
      todo: Todo
`
	
	// Unmarshal old format
	var cfg Config
	err := yaml.Unmarshal([]byte(oldFormatYAML), &cfg)
	require.NoError(t, err)
	
	// Verify it loads correctly without metadata
	assert.Equal(t, "Old Project", cfg.Project.Name)
	assert.Equal(t, "old-org", cfg.Project.Org)
	assert.Nil(t, cfg.Metadata)
	
	// Save and reload to ensure it remains compatible
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yml")
	err = cfg.Save(configPath)
	require.NoError(t, err)
	
	// Reload
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	
	var reloadedCfg Config
	err = yaml.Unmarshal(data, &reloadedCfg)
	require.NoError(t, err)
	
	assert.Equal(t, cfg.Project.Name, reloadedCfg.Project.Name)
	assert.Nil(t, reloadedCfg.Metadata)
}