package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yahsan2/gh-pm/pkg/config"
)

func TestInitCommand(t *testing.T) {
	t.Run("contains function", func(t *testing.T) {
		slice := []string{"apple", "banana", "orange"}
		
		assert.True(t, contains(slice, "banana"))
		assert.False(t, contains(slice, "grape"))
		assert.False(t, contains([]string{}, "anything"))
	})

	t.Run("config file creation", func(t *testing.T) {
		// Create temp directory for test
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		
		// Change to temp directory
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		
		// Create default config
		cfg := config.DefaultConfig()
		cfg.Project.Name = "Test Project"
		cfg.Project.Org = "test-org"
		cfg.Repositories = []string{"test-org/test-repo"}
		
		// Save config
		configPath := filepath.Join(tmpDir, config.ConfigFileName)
		err = cfg.Save(configPath)
		require.NoError(t, err)
		
		// Verify file exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)
		
		// Load config and verify
		loadedCfg, err := config.Load()
		require.NoError(t, err)
		
		assert.Equal(t, "Test Project", loadedCfg.Project.Name)
		assert.Equal(t, "test-org", loadedCfg.Project.Org)
		assert.Equal(t, []string{"test-org/test-repo"}, loadedCfg.Repositories)
		assert.Equal(t, "medium", loadedCfg.Defaults.Priority)
		assert.Equal(t, "Todo", loadedCfg.Defaults.Status)
	})
}