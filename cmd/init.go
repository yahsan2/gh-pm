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
	initpkg "github.com/yahsan2/gh-pm/pkg/init"
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
	initListProjects bool
	skipMetadata     bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	
	initCmd.Flags().StringVar(&initProject, "project", "", "Project name or ID")
	initCmd.Flags().StringVar(&initOrg, "org", "", "Organization name")
	initCmd.Flags().StringSliceVar(&initRepos, "repo", []string{}, "Repositories (owner/repo format)")
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", true, "Interactive mode")
	initCmd.Flags().BoolVarP(&initListProjects, "list", "l", false, "List all available projects to choose from")
	initCmd.Flags().BoolVar(&skipMetadata, "skip-metadata", false, "Skip fetching project metadata (IDs for fields and options)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if config already exists
	var cfg *config.Config
	
	if config.Exists() {
		fmt.Println("Configuration file .gh-pm.yml already exists in this directory or a parent directory.")
		fmt.Print("Do you want to update it? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Initialization cancelled.")
			return nil
		}
		
		// Load existing config
		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			fmt.Printf("Warning: Could not load existing config, creating new one: %v\n", err)
			cfg = config.DefaultConfig()
		} else {
			fmt.Println("Updating existing configuration...")
		}
	} else {
		// Create default config
		cfg = config.DefaultConfig()
	}

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

		// Try to auto-detect projects from current repository or show all projects
		if initProject == "" && (initInteractive || initListProjects) {
			client, err := project.NewClient()
			if err == nil {
				var projectsToShow []project.Project
				var sourceDescription string

				if initListProjects {
					// User explicitly wants to see all projects
					fmt.Printf("Fetching all projects from organization %s...\n", cfg.Project.Org)
					orgProjects, err := client.ListProjects(cfg.Project.Org)
					if err == nil {
						projectsToShow = orgProjects
						sourceDescription = fmt.Sprintf("organization %s", cfg.Project.Org)
					}
				} else {
					// Try repository projects first
					fmt.Printf("Detecting projects for repository %s/%s...\n", org, repo)
					repoProjects, err := client.GetRepoProjects(org, repo)
					if err == nil && len(repoProjects) > 0 {
						projectsToShow = repoProjects
						sourceDescription = fmt.Sprintf("repository %s/%s", org, repo)
					} else if err == nil && len(repoProjects) == 0 {
						// No projects in repo, try org projects
						fmt.Printf("No projects found in repository. Checking organization projects...\n")
						orgProjects, err := client.ListProjects(cfg.Project.Org)
						if err == nil {
							projectsToShow = orgProjects
							sourceDescription = fmt.Sprintf("organization %s", cfg.Project.Org)
						}
					}
				}

				// Show projects and let user select
				if len(projectsToShow) > 0 {
					selectedProject := selectProjectWithDetails(projectsToShow, sourceDescription)
					if selectedProject != nil {
						cfg.Project.Name = selectedProject.Title
						cfg.Project.Number = selectedProject.Number
						initProject = selectedProject.Title
						fmt.Printf("✓ Selected project: %s (#%d)\n", selectedProject.Title, selectedProject.Number)
						
						// After selecting project, configure field mappings if interactive
						if initInteractive {
							configureFieldMappings(cfg, selectedProject, client)
						}
					}
				} else {
					fmt.Println("No projects found.")
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
		// Use repo name as default if available
		defaultName := ""
		if repoErr == nil {
			defaultName = repo
		}
		
		if defaultName != "" {
			fmt.Printf("Enter project name (default: %s, leave empty to skip): ", defaultName)
		} else {
			fmt.Print("Enter project name (leave empty to skip): ")
		}
		
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input == "" && defaultName != "" {
				// Use default if no input provided
				initProject = defaultName
			} else if input != "" {
				initProject = input
			}
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
	// Always fetch to ensure we have the latest owner information
	if cfg.Project.Name != "" || cfg.Project.Number > 0 {
		fmt.Printf("Fetching project details from GitHub...\n")
		
		client, err := project.NewClient()
		if err != nil {
			fmt.Printf("Warning: Could not connect to GitHub: %v\n", err)
		} else {
			var proj *project.Project
			
			// Try as organization project first if org is specified
			if cfg.Project.Org != "" {
				// Use project number if available, otherwise use name
				if cfg.Project.Number > 0 {
					proj, err = client.GetProject(cfg.Project.Org, "", cfg.Project.Number)
				} else {
					proj, err = client.GetProject(cfg.Project.Org, cfg.Project.Name, 0)
				}
			}
			
			// If failed or no org specified, try as user project
			if proj == nil || err != nil {
				// Extract owner from repository if available
				var owner string
				if len(cfg.Repositories) > 0 {
					parts := strings.Split(cfg.Repositories[0], "/")
					if len(parts) == 2 {
						owner = parts[0]
					}
				}
				
				// Try with owner as user
				if owner != "" {
					var userProj *project.Project
					var userErr error
					if cfg.Project.Number > 0 {
						userProj, userErr = client.GetUserProject(owner, "", cfg.Project.Number)
					} else {
						userProj, userErr = client.GetUserProject(owner, cfg.Project.Name, 0)
					}
					if userErr == nil {
						proj = userProj
						err = nil
						// Clear org since it's a user project
						cfg.Project.Org = ""
					}
				}
				
				// If still no project, try current user
				if proj == nil {
					var userProj *project.Project
					var userErr error
					if cfg.Project.Number > 0 {
						userProj, userErr = client.GetCurrentUserProject("", cfg.Project.Number)
					} else {
						userProj, userErr = client.GetCurrentUserProject(cfg.Project.Name, 0)
					}
					if userErr == nil {
						proj = userProj
						err = nil
						cfg.Project.Org = ""
					}
				}
			}
			
			if err != nil || proj == nil {
				fmt.Printf("Warning: Could not fetch project details: %v\n", err)
			} else {
				cfg.Project.Number = proj.Number
				// Store the project owner for URL generation
				if proj.Owner.Login != "" {
					cfg.Project.Owner = proj.Owner.Login
					fmt.Printf("✓ Updated project owner: %s\n", proj.Owner.Login)
				}
				fmt.Printf("✓ Found project: %s (#%d)\n", proj.Title, proj.Number)
				
				// Fetch and display available fields
				fields, err := client.GetProjectFields(proj.ID)
				if err == nil && len(fields) > 0 {
					fmt.Println("\nAvailable project fields:")
					for _, field := range fields {
						fmt.Printf("  - %s (%s)\n", field.Name, field.DataType)
					}
					
					// Auto-configure field mappings in non-interactive mode
					if !initInteractive {
						autoConfigureFieldMappings(cfg, fields)
					}
					
					// Build metadata if not skipping
					if !skipMetadata {
						metadataManager := initpkg.NewMetadataManager(client)
						metadata, err := metadataManager.BuildMetadata(proj, fields)
						if err != nil {
							fmt.Printf("Warning: Could not build metadata: %v\n", err)
						} else {
							cfg.Metadata = metadata
						}
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

// selectProjectWithDetails presents a detailed list of projects and allows the user to select one
func selectProjectWithDetails(projects []project.Project, source string) *project.Project {
	if len(projects) == 0 {
		return nil
	}

	// Show detailed project list
	fmt.Printf("\nAvailable projects from %s:\n", source)
	fmt.Println(strings.Repeat("-", 70))
	for i, p := range projects {
		fmt.Printf("%2d. %-40s #%-6d\n", i+1, truncateString(p.Title, 40), p.Number)
		if p.URL != "" {
			fmt.Printf("    URL: %s\n", p.URL)
		}
		if i < len(projects)-1 {
			fmt.Println()
		}
	}
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("  0. Skip project selection\n")
	fmt.Printf("  N. Enter project number directly\n")
	fmt.Printf("\nSelect a project (0-%d) or enter 'N' for project number: ", len(projects))

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		
		// Check if user wants to enter project number directly
		if strings.ToLower(input) == "n" {
			fmt.Print("Enter project number: ")
			if scanner.Scan() {
				projectNum := strings.TrimSpace(scanner.Text())
				if num, err := strconv.Atoi(projectNum); err == nil {
					// Find project by number
					for _, p := range projects {
						if p.Number == num {
							return &p
						}
					}
					fmt.Printf("Project #%d not found in the list.\n", num)
				}
			}
			return nil
		}
		
		// Try to parse as selection number
		choice, err := strconv.Atoi(input)
		if err == nil && choice >= 0 && choice <= len(projects) {
			if choice == 0 {
				return nil
			}
			return &projects[choice-1]
		}
		
		// Try to match by project name (partial match)
		lowerInput := strings.ToLower(input)
		for _, p := range projects {
			if strings.Contains(strings.ToLower(p.Title), lowerInput) {
				fmt.Printf("Found matching project: %s (#%d)\n", p.Title, p.Number)
				fmt.Print("Use this project? (Y/n): ")
				if scanner.Scan() {
					response := strings.ToLower(strings.TrimSpace(scanner.Text()))
					if response == "" || response == "y" || response == "yes" {
						return &p
					}
				}
			}
		}
	}

	fmt.Println("Invalid selection, skipping project selection.")
	return nil
}

// configureFieldMappings allows user to configure field mappings for the selected project
// autoConfigureFieldMappings automatically configures field mappings without user interaction
func autoConfigureFieldMappings(cfg *config.Config, fields []project.Field) {
	// Look for Status field
	for _, field := range fields {
		if strings.EqualFold(field.Name, "status") && field.DataType == "SINGLE_SELECT" && len(field.Options) > 0 {
			statusMapping := config.Field{
				Field:  "Status",
				Values: make(map[string]string),
			}
			
			// Set the first status option as the default
			firstOption := field.Options[0].Name
			normalizedKey := strings.ToLower(strings.ReplaceAll(firstOption, " ", "_"))
			cfg.Defaults.Status = normalizedKey
			
			// Map all status options using their normalized names as keys
			for _, opt := range field.Options {
				normalizedKey := strings.ToLower(strings.ReplaceAll(opt.Name, " ", "_"))
				statusMapping.Values[normalizedKey] = opt.Name
				
				// Also add common aliases
				lowerOpt := strings.ToLower(opt.Name)
				if strings.Contains(lowerOpt, "todo") && normalizedKey != "todo" {
					statusMapping.Values["todo"] = opt.Name
				}
				if strings.Contains(lowerOpt, "backlog") && normalizedKey != "backlog" {
					statusMapping.Values["backlog"] = opt.Name
				}
				if strings.Contains(lowerOpt, "progress") && normalizedKey != "in_progress" {
					statusMapping.Values["in_progress"] = opt.Name
				}
				if strings.Contains(lowerOpt, "review") && normalizedKey != "in_review" {
					statusMapping.Values["in_review"] = opt.Name
				}
				if strings.Contains(lowerOpt, "done") && normalizedKey != "done" {
					statusMapping.Values["done"] = opt.Name
				}
			}
			
			cfg.Fields["status"] = statusMapping
			fmt.Printf("✓ Auto-configured Status field mappings\n")
			break
		}
	}
	
	// Look for Priority field
	for _, field := range fields {
		if strings.EqualFold(field.Name, "priority") && field.DataType == "SINGLE_SELECT" && len(field.Options) > 0 {
			priorityMapping := config.Field{
				Field:  "Priority",
				Values: make(map[string]string),
			}
			
			// Set default priority to middle option
			middleIndex := len(field.Options) / 2
			if middleIndex < len(field.Options) {
				defaultPriority := field.Options[middleIndex].Name
				normalizedKey := strings.ToLower(strings.ReplaceAll(defaultPriority, " ", "_"))
				cfg.Defaults.Priority = normalizedKey
			}
			
			// Map all priority options using their normalized names as keys
			for _, opt := range field.Options {
				normalizedKey := strings.ToLower(strings.ReplaceAll(opt.Name, " ", "_"))
				priorityMapping.Values[normalizedKey] = opt.Name
				
				// Also add common aliases
				lowerOpt := strings.ToLower(opt.Name)
				if (strings.Contains(lowerOpt, "p0") || strings.Contains(lowerOpt, "critical")) && normalizedKey != "critical" {
					priorityMapping.Values["critical"] = opt.Name
				}
				if (strings.Contains(lowerOpt, "p1") || strings.Contains(lowerOpt, "high")) && normalizedKey != "high" {
					priorityMapping.Values["high"] = opt.Name
				}
				if (strings.Contains(lowerOpt, "p2") || strings.Contains(lowerOpt, "medium")) && normalizedKey != "medium" {
					priorityMapping.Values["medium"] = opt.Name
				}
				if (strings.Contains(lowerOpt, "p3") || strings.Contains(lowerOpt, "low")) && normalizedKey != "low" {
					priorityMapping.Values["low"] = opt.Name
				}
			}
			
			cfg.Fields["priority"] = priorityMapping
			fmt.Printf("✓ Auto-configured Priority field mappings\n")
			break
		}
	}
}

func configureFieldMappings(cfg *config.Config, proj *project.Project, client *project.Client) {
	fmt.Println("\nFetching project fields...")
	
	fields, err := client.GetProjectFields(proj.ID)
	if err != nil {
		fmt.Printf("Warning: Could not fetch project fields: %v\n", err)
		return
	}
	
	if len(fields) == 0 {
		fmt.Println("No custom fields found in the project.")
		return
	}
	
	// Build metadata if not skipping
	if !skipMetadata {
		metadataManager := initpkg.NewMetadataManager(client)
		metadata, err := metadataManager.BuildMetadata(proj, fields)
		if err != nil {
			fmt.Printf("Warning: Could not build metadata: %v\n", err)
		} else {
			cfg.Metadata = metadata
			fmt.Println("✓ Project metadata captured for faster operations")
		}
	}
	
	// Look for Status field
	var statusField *project.Field
	for _, field := range fields {
		if strings.EqualFold(field.Name, "status") && field.DataType == "SINGLE_SELECT" {
			statusField = &field
			break
		}
	}
	
	if statusField != nil && len(statusField.Options) > 0 {
		fmt.Println("\nFound Status field with the following options:")
		for i, opt := range statusField.Options {
			fmt.Printf("  %d. %s\n", i+1, opt.Name)
		}
		
		// Always update status field mappings based on actual project options
		statusMapping := config.Field{
			Field:  "Status",
			Values: make(map[string]string),
		}
		
		// Set the first status option as the default
		if len(statusField.Options) > 0 {
			firstOption := statusField.Options[0].Name
			// Create a normalized key from the actual value
			normalizedKey := strings.ToLower(strings.ReplaceAll(firstOption, " ", "_"))
			cfg.Defaults.Status = normalizedKey
			statusMapping.Values[normalizedKey] = firstOption
		}
		
		// Map all status options using their normalized names as keys
		for _, opt := range statusField.Options {
			// Create normalized key from the actual option name
			normalizedKey := strings.ToLower(strings.ReplaceAll(opt.Name, " ", "_"))
			statusMapping.Values[normalizedKey] = opt.Name
			
			// Also add common aliases for convenience
			lowerOpt := strings.ToLower(opt.Name)
			switch {
			case strings.Contains(lowerOpt, "todo") && normalizedKey != "todo":
				statusMapping.Values["todo"] = opt.Name
			case strings.Contains(lowerOpt, "backlog") && normalizedKey != "backlog":
				statusMapping.Values["backlog"] = opt.Name
			case strings.Contains(lowerOpt, "ready") && normalizedKey != "ready":
				statusMapping.Values["ready"] = opt.Name
			case strings.Contains(lowerOpt, "progress") && normalizedKey != "in_progress":
				statusMapping.Values["in_progress"] = opt.Name
			case strings.Contains(lowerOpt, "doing") && normalizedKey != "doing":
				statusMapping.Values["doing"] = opt.Name
			case strings.Contains(lowerOpt, "review") && normalizedKey != "in_review":
				statusMapping.Values["in_review"] = opt.Name
			case strings.Contains(lowerOpt, "done") && normalizedKey != "done":
				statusMapping.Values["done"] = opt.Name
			case strings.Contains(lowerOpt, "complete") && normalizedKey != "complete":
				statusMapping.Values["complete"] = opt.Name
			}
		}
		
		fmt.Print("\nWould you like to configure status field mappings? (y/N): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response == "y" || response == "yes" {
				// Allow manual override
				fmt.Println("\nCurrent status mappings:")
				for key, val := range statusMapping.Values {
					fmt.Printf("  %s -> %s\n", key, val)
				}
				
				fmt.Print("\nWould you like to customize these mappings? (y/N): ")
				if scanner.Scan() {
					response := strings.ToLower(strings.TrimSpace(scanner.Text()))
					if response == "y" || response == "yes" {
						for _, opt := range statusField.Options {
							fmt.Printf("\nMap '%s' to (todo/ready/in_progress/in_review/done/skip): ", opt.Name)
							if scanner.Scan() {
								mapping := strings.TrimSpace(scanner.Text())
								if mapping != "" && mapping != "skip" {
									statusMapping.Values[mapping] = opt.Name
								}
							}
						}
					}
				}
				
				fmt.Println("✓ Status field mappings configured")
			}
		}
		
		// Always update the config with the actual field mappings
		cfg.Fields["status"] = statusMapping
	}
	
	// Look for Priority field
	var priorityField *project.Field
	for _, field := range fields {
		if strings.EqualFold(field.Name, "priority") && field.DataType == "SINGLE_SELECT" {
			priorityField = &field
			break
		}
	}
	
	if priorityField != nil && len(priorityField.Options) > 0 {
		fmt.Println("\nFound Priority field with the following options:")
		for i, opt := range priorityField.Options {
			fmt.Printf("  %d. %s\n", i+1, opt.Name)
		}
		
		// Always update priority field mappings based on actual project options
		priorityMapping := config.Field{
			Field:  "Priority",
			Values: make(map[string]string),
		}
		
		// Set default priority to middle option if available, otherwise first
		middleIndex := len(priorityField.Options) / 2
		if middleIndex < len(priorityField.Options) {
			defaultPriority := priorityField.Options[middleIndex].Name
			normalizedKey := strings.ToLower(strings.ReplaceAll(defaultPriority, " ", "_"))
			cfg.Defaults.Priority = normalizedKey
			priorityMapping.Values[normalizedKey] = defaultPriority
		}
		
		// Map all priority options using their normalized names as keys
		for _, opt := range priorityField.Options {
			// Create normalized key from the actual option name
			normalizedKey := strings.ToLower(strings.ReplaceAll(opt.Name, " ", "_"))
			priorityMapping.Values[normalizedKey] = opt.Name
			
			// Also add common aliases for convenience
			lowerOpt := strings.ToLower(opt.Name)
			switch {
			case strings.Contains(lowerOpt, "p0") && normalizedKey != "p0":
				priorityMapping.Values["p0"] = opt.Name
			case strings.Contains(lowerOpt, "critical") && normalizedKey != "critical":
				priorityMapping.Values["critical"] = opt.Name
			case strings.Contains(lowerOpt, "urgent") && normalizedKey != "urgent":
				priorityMapping.Values["urgent"] = opt.Name
			case strings.Contains(lowerOpt, "p1") && normalizedKey != "p1":
				priorityMapping.Values["p1"] = opt.Name
			case strings.Contains(lowerOpt, "high") && normalizedKey != "high":
				priorityMapping.Values["high"] = opt.Name
			case strings.Contains(lowerOpt, "p2") && normalizedKey != "p2":
				priorityMapping.Values["p2"] = opt.Name
			case strings.Contains(lowerOpt, "medium") && normalizedKey != "medium":
				priorityMapping.Values["medium"] = opt.Name
			case strings.Contains(lowerOpt, "normal") && normalizedKey != "normal":
				priorityMapping.Values["normal"] = opt.Name
			case strings.Contains(lowerOpt, "p3") && normalizedKey != "p3":
				priorityMapping.Values["p3"] = opt.Name
			case strings.Contains(lowerOpt, "low") && normalizedKey != "low":
				priorityMapping.Values["low"] = opt.Name
			}
		}
		
		fmt.Print("\nWould you like to configure priority field mappings? (y/N): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response == "y" || response == "yes" {
				// Allow manual override
				fmt.Println("\nCurrent priority mappings:")
				for key, val := range priorityMapping.Values {
					fmt.Printf("  %s -> %s\n", key, val)
				}
				
				fmt.Print("\nWould you like to customize these mappings? (y/N): ")
				if scanner.Scan() {
					response := strings.ToLower(strings.TrimSpace(scanner.Text()))
					if response == "y" || response == "yes" {
						for _, opt := range priorityField.Options {
							fmt.Printf("\nMap '%s' to (low/medium/high/critical/skip): ", opt.Name)
							if scanner.Scan() {
								mapping := strings.TrimSpace(scanner.Text())
								if mapping != "" && mapping != "skip" {
									priorityMapping.Values[mapping] = opt.Name
								}
							}
						}
					}
				}
				
				fmt.Println("✓ Priority field mappings configured")
			}
		}
		
		// Always update the config with the actual field mappings
		cfg.Fields["priority"] = priorityMapping
	}
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}