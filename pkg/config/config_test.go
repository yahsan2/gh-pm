package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg string
	}{
		{
			name: "valid config with project name",
			config: &Config{
				Project: ProjectConfig{
					Name: "My Project",
				},
				Repositories: []string{"owner/repo"},
			},
			wantErr: false,
		},
		{
			name: "valid config with project number",
			config: &Config{
				Project: ProjectConfig{
					Number: 1,
				},
				Repositories: []string{"owner/repo"},
			},
			wantErr: false,
		},
		{
			name: "missing project name and number",
			config: &Config{
				Repositories: []string{"owner/repo"},
			},
			wantErr: true,
			errMsg:  "project name or number is required",
		},
		{
			name: "missing repositories",
			config: &Config{
				Project: ProjectConfig{
					Name: "My Project",
				},
			},
			wantErr: true,
			errMsg:  "at least one repository must be configured",
		},
		{
			name: "invalid repository format",
			config: &Config{
				Project: ProjectConfig{
					Name: "My Project",
				},
				Repositories: []string{"invalid-repo"},
			},
			wantErr: true,
			errMsg:  "invalid repository format",
		},
		{
			name: "field without field name",
			config: &Config{
				Project: ProjectConfig{
					Name: "My Project",
				},
				Repositories: []string{"owner/repo"},
				Fields: map[string]Field{
					"priority": {
						Field:  "",
						Values: map[string]string{"high": "HIGH"},
					},
				},
			},
			wantErr: true,
			errMsg:  "field name is required",
		},
		{
			name: "field without values",
			config: &Config{
				Project: ProjectConfig{
					Name: "My Project",
				},
				Repositories: []string{"owner/repo"},
				Fields: map[string]Field{
					"priority": {
						Field:  "Priority",
						Values: map[string]string{},
					},
				},
			},
			wantErr: true,
			errMsg:  "at least one value mapping is required",
		},
		{
			name: "invalid default priority",
			config: &Config{
				Project: ProjectConfig{
					Name: "My Project",
				},
				Repositories: []string{"owner/repo"},
				Fields: map[string]Field{
					"priority": {
						Field:  "Priority",
						Values: map[string]string{"high": "HIGH"},
					},
				},
				Defaults: DefaultsConfig{
					Priority: "medium",
				},
			},
			wantErr: true,
			errMsg:  "default priority 'medium' is not defined",
		},
		{
			name: "invalid default status",
			config: &Config{
				Project: ProjectConfig{
					Name: "My Project",
				},
				Repositories: []string{"owner/repo"},
				Fields: map[string]Field{
					"status": {
						Field:  "Status",
						Values: map[string]string{"todo": "Todo"},
					},
				},
				Defaults: DefaultsConfig{
					Status: "done",
				},
			},
			wantErr: true,
			errMsg:  "default status 'done' is not defined",
		},
		{
			name: "valid config with all fields",
			config: &Config{
				Project: ProjectConfig{
					Name:   "My Project",
					Number: 1,
				},
				Repositories: []string{"owner/repo", "owner2/repo2"},
				Fields: map[string]Field{
					"priority": {
						Field:  "Priority",
						Values: map[string]string{"high": "HIGH", "medium": "MEDIUM"},
					},
					"status": {
						Field:  "Status",
						Values: map[string]string{"todo": "Todo", "done": "Done"},
					},
				},
				Defaults: DefaultsConfig{
					Priority: "medium",
					Status:   "todo",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func() func()
		wantErr    bool
		wantConfig *Config
	}{
		{
			name: "load valid config",
			setupFunc: func() func() {
				tmpDir, _ := os.MkdirTemp("", "test")
				configFile := filepath.Join(tmpDir, ConfigFileName)
				
				config := &Config{
					Project: ProjectConfig{
						Name:   "Test Project",
						Number: 123,
					},
					Repositories: []string{"test/repo"},
					Defaults: DefaultsConfig{
						Labels: []string{"bug", "enhancement"},
					},
				}
				
				data, _ := yaml.Marshal(config)
				os.WriteFile(configFile, data, 0644)
				
				origDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				
				return func() {
					os.Chdir(origDir)
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: false,
			wantConfig: &Config{
				Project: ProjectConfig{
					Name:   "Test Project",
					Number: 123,
				},
				Repositories: []string{"test/repo"},
				Defaults: DefaultsConfig{
					Labels: []string{"bug", "enhancement"},
				},
			},
		},
		{
			name: "config file not found",
			setupFunc: func() func() {
				tmpDir, _ := os.MkdirTemp("", "test")
				origDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				
				return func() {
					os.Chdir(origDir)
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: true,
		},
		{
			name: "invalid yaml format",
			setupFunc: func() func() {
				tmpDir, _ := os.MkdirTemp("", "test")
				configFile := filepath.Join(tmpDir, ConfigFileName)
				
				os.WriteFile(configFile, []byte("invalid: yaml: content:"), 0644)
				
				origDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				
				return func() {
					os.Chdir(origDir)
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc()
			defer cleanup()

			got, err := LoadConfig()
			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadConfig() error = nil, wantErr %v", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
				if tt.wantConfig != nil {
					if got.Project.Name != tt.wantConfig.Project.Name ||
						got.Project.Number != tt.wantConfig.Project.Number ||
						len(got.Repositories) != len(tt.wantConfig.Repositories) {
						t.Errorf("LoadConfig() = %v, want %v", got, tt.wantConfig)
					}
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}