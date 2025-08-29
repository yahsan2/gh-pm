package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
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

	// Try to detect from current repository first
	org, repo, repoErr := getCurrentRepo()
	if repoErr == nil {
		// Set org if not specified
		if initOrg == "" {
			cfg.Project.Org = org
		} else {
			cfg.Project.Org = initOrg
		}
		
		// Add current repo to repositories if not already included
		currentRepo := fmt.Sprintf("%s/%s", org, repo)
		if !contains(initRepos, currentRepo) {
			cfg.Repositories = append([]string{currentRepo}, initRepos...)
		} else {
			cfg.Repositories = initRepos
		}

		// Try to auto-detect projects from current repository
		if initProject == "" && initInteractive {
			fmt.Printf("Detecting projects for repository %s/%s...\n", org, repo)
			
			client, err := project.NewClient()
			if err == nil {
				repoProjects, err := client.GetRepoProjects(org, repo)
				if err == nil && len(repoProjects) > 0 {
					selectedProject := selectProject(repoProjects, org)
					if selectedProject != nil {
						cfg.Project.Name = selectedProject.Title
						cfg.Project.Number = selectedProject.Number
						initProject = selectedProject.Title
						fmt.Printf("✓ Selected project: %s (#%d)\n", selectedProject.Title, selectedProject.Number)
					}
				} else if err == nil && len(repoProjects) == 0 {
					// No projects in repo, try to list org projects
					fmt.Printf("No projects found in repository. Checking organization projects...\n")
					orgProjects, err := client.ListProjects(cfg.Project.Org)
					if err == nil && len(orgProjects) > 0 {
						selectedProject := selectProject(orgProjects, cfg.Project.Org)
						if selectedProject != nil {
							cfg.Project.Name = selectedProject.Title
							cfg.Project.Number = selectedProject.Number
							initProject = selectedProject.Title
							fmt.Printf("✓ Selected project: %s (#%d)\n", selectedProject.Title, selectedProject.Number)
						}
					}
				}
			}
		}
	} else {
		// No repo detected, use provided values
		cfg.Project.Org = initOrg
		cfg.Repositories = initRepos
	}

	// If interactive mode and project still not set, prompt for it
	if initInteractive && initProject == "" {
		fmt.Print("Enter project name (leave empty to skip): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			initProject = strings.TrimSpace(scanner.Text())
		}
	}

	// If interactive mode and org still not set, prompt for it
	if initInteractive && cfg.Project.Org == "" {
		fmt.Print("Enter organization name: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			cfg.Project.Org = strings.TrimSpace(scanner.Text())
		}
	}

	// If interactive mode and no repos, prompt for them
	if initInteractive && len(cfg.Repositories) == 0 {
		fmt.Print("Enter repositories (comma-separated, e.g., owner/repo1,owner/repo2): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			repoInput := strings.TrimSpace(scanner.Text())
			if repoInput != "" {
				repos := strings.Split(repoInput, ",")
				for _, r := range repos {
					cfg.Repositories = append(cfg.Repositories, strings.TrimSpace(r))
				}
			}
		}
	}

	// Set the project name if it was provided or selected
	if initProject != "" {
		cfg.Project.Name = initProject
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

// selectProject presents a list of projects and allows the user to select one
func selectProject(projects []project.Project, orgName string) *project.Project {
	if len(projects) == 0 {
		return nil
	}

	// If only one project, auto-select it
	if len(projects) == 1 {
		fmt.Printf("Found 1 project: %s (#%d)\n", projects[0].Title, projects[0].Number)
		fmt.Print("Use this project? (Y/n): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response == "" || response == "y" || response == "yes" {
				return &projects[0]
			}
		}
		return nil
	}

	// Multiple projects, show selection menu
	fmt.Printf("\nFound %d projects:\n", len(projects))
	for i, p := range projects {
		fmt.Printf("  %d. %s (#%d)\n", i+1, p.Title, p.Number)
	}
	fmt.Printf("  0. Skip project selection\n")
	fmt.Printf("\nSelect a project (0-%d): ", len(projects))

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		choice, err := strconv.Atoi(input)
		if err == nil && choice >= 0 && choice <= len(projects) {
			if choice == 0 {
				return nil
			}
			return &projects[choice-1]
		}
	}

	fmt.Println("Invalid selection, skipping project selection.")
	return nil
}