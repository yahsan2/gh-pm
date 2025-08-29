package cmd

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
	"github.com/yahsan2/gh-pm/pkg/config"
	"github.com/yahsan2/gh-pm/pkg/project"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gh-pm configuration",
	Long: `Initialize a new gh-pm configuration file (.gh-pm.yml) in the current directory.
	
This command will:
- Create a .gh-pm.yml configuration file
- Set up project and repository settings
- Configure default values and field mappings`,
	Example: `  # Interactive initialization
  gh pm init
  
  # Specify project and repositories
  gh pm init --project "My Project" --repo owner/repo1,owner/repo2
  
  # Specify organization project
  gh pm init --project "Team Project" --org my-organization`,
	RunE: runInit,
}

var (
	initProject      string
	initOrg          string
	initRepos        []string
	initInteractive  bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	
	initCmd.Flags().StringVar(&initProject, "project", "", "Project name or ID")
	initCmd.Flags().StringVar(&initOrg, "org", "", "Organization name")
	initCmd.Flags().StringSliceVar(&initRepos, "repo", []string{}, "Repositories (owner/repo format)")
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", true, "Interactive mode")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if config already exists
	if config.Exists() {
		fmt.Println("Configuration file .gh-pm.yml already exists in this directory or a parent directory.")
		fmt.Print("Do you want to overwrite it? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Initialization cancelled.")
			return nil
		}
	}

	// Create default config
	cfg := config.DefaultConfig()

	// If interactive mode, prompt for values
	if initInteractive && initProject == "" {
		fmt.Print("Enter project name (leave empty to skip): ")
		fmt.Scanln(&initProject)
	}

	if initInteractive && initOrg == "" {
		fmt.Print("Enter organization name (leave empty for current repo's org): ")
		fmt.Scanln(&initOrg)
	}

	if initInteractive && len(initRepos) == 0 {
		fmt.Print("Enter repositories (comma-separated, e.g., owner/repo1,owner/repo2): ")
		var repoInput string
		fmt.Scanln(&repoInput)
		if repoInput != "" {
			initRepos = strings.Split(repoInput, ",")
		}
	}

	// Set values from flags or interactive input
	cfg.Project.Name = initProject
	cfg.Project.Org = initOrg
	
	// Clean up repository names
	for i, repo := range initRepos {
		initRepos[i] = strings.TrimSpace(repo)
	}
	cfg.Repositories = initRepos

	// If org is not specified, try to detect from current repository
	if cfg.Project.Org == "" {
		org, repo, err := getCurrentRepo()
		if err == nil {
			if cfg.Project.Org == "" {
				cfg.Project.Org = org
			}
			// Add current repo to repositories if not already included
			currentRepo := fmt.Sprintf("%s/%s", org, repo)
			if !contains(cfg.Repositories, currentRepo) {
				cfg.Repositories = append([]string{currentRepo}, cfg.Repositories...)
			}
		}
	}

	// If project is specified, try to fetch project details
	if cfg.Project.Name != "" && cfg.Project.Org != "" {
		fmt.Printf("Fetching project details from GitHub...\n")
		
		client, err := project.NewClient()
		if err != nil {
			fmt.Printf("Warning: Could not connect to GitHub: %v\n", err)
		} else {
			proj, err := client.GetProject(cfg.Project.Org, cfg.Project.Name, 0)
			if err != nil {
				fmt.Printf("Warning: Could not fetch project details: %v\n", err)
			} else {
				cfg.Project.Number = proj.Number
				fmt.Printf("✓ Found project: %s (#%d)\n", proj.Title, proj.Number)
				
				// Fetch and display available fields
				fields, err := client.GetProjectFields(proj.ID)
				if err == nil && len(fields) > 0 {
					fmt.Println("\nAvailable project fields:")
					for _, field := range fields {
						fmt.Printf("  - %s (%s)\n", field.Name, field.DataType)
					}
				}
			}
		}
	}

	// Save configuration
	configPath := config.ConfigFileName
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and edit .gh-pm.yml to customize settings")
	fmt.Println("  2. Run 'gh pm list' to view issues in your project")
	fmt.Println("  3. Run 'gh pm create --title \"Your task\"' to create a new issue")
	
	return nil
}

// getCurrentRepo returns the current repository's owner and name
func getCurrentRepo() (string, string, error) {
	// Try to get repo info using gh CLI
	repo, err := repository.Current()
	if err != nil {
		return "", "", err
	}
	
	return repo.Owner, repo.Name, nil
}

// contains checks if a string slice contains a specific value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}