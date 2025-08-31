package init

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/yahsan2/gh-pm/pkg/project"
)

// InteractivePrompt handles interactive user input
type InteractivePrompt struct {
	scanner *bufio.Scanner
}

// NewInteractivePrompt creates a new InteractivePrompt instance
func NewInteractivePrompt() *InteractivePrompt {
	return &InteractivePrompt{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// ConfirmOverwrite prompts the user to confirm overwriting an existing file
func (p *InteractivePrompt) ConfirmOverwrite() bool {
	fmt.Println("Configuration file .gh-pm.yml already exists in this directory or a parent directory.")
	fmt.Print("Do you want to overwrite it? (y/N): ")

	if p.scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(p.scanner.Text()))
		return response == "y" || response == "yes"
	}

	return false
}

// SelectProject presents a list of projects and allows the user to select one
func (p *InteractivePrompt) SelectProject(projects []project.Project, source string) *project.Project {
	if len(projects) == 0 {
		return nil
	}

	// If only one project, auto-select with confirmation
	if len(projects) == 1 {
		fmt.Printf("Found 1 project: %s (#%d)\n", projects[0].Title, projects[0].Number)
		fmt.Print("Use this project? (Y/n): ")
		if p.scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(p.scanner.Text()))
			if response == "" || response == "y" || response == "yes" {
				return &projects[0]
			}
		}
		return nil
	}

	// Multiple projects, show selection menu
	fmt.Printf("\nAvailable projects from %s:\n", source)
	fmt.Println(strings.Repeat("-", 70))
	for i, proj := range projects {
		fmt.Printf("%2d. %-40s #%-6d\n", i+1, truncateString(proj.Title, 40), proj.Number)
		if proj.URL != "" {
			fmt.Printf("    URL: %s\n", proj.URL)
		}
		if i < len(projects)-1 {
			fmt.Println()
		}
	}
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("  0. Skip project selection\n")
	fmt.Printf("  N. Enter project number directly\n")
	fmt.Printf("\nSelect a project (0-%d) or enter 'N' for project number: ", len(projects))

	if p.scanner.Scan() {
		input := strings.TrimSpace(p.scanner.Text())

		// Check if user wants to enter project number directly
		if strings.ToLower(input) == "n" {
			fmt.Print("Enter project number: ")
			if p.scanner.Scan() {
				projectNum := strings.TrimSpace(p.scanner.Text())
				if num, err := strconv.Atoi(projectNum); err == nil {
					// Find project by number
					for _, proj := range projects {
						if proj.Number == num {
							return &proj
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
		for _, proj := range projects {
			if strings.Contains(strings.ToLower(proj.Title), lowerInput) {
				fmt.Printf("Found matching project: %s (#%d)\n", proj.Title, proj.Number)
				fmt.Print("Use this project? (Y/n): ")
				if p.scanner.Scan() {
					response := strings.ToLower(strings.TrimSpace(p.scanner.Text()))
					if response == "" || response == "y" || response == "yes" {
						return &proj
					}
				}
			}
		}
	}

	fmt.Println("Invalid selection, skipping project selection.")
	return nil
}

// ConfigureFieldMapping allows user to configure field mappings
func (p *InteractivePrompt) ConfigureFieldMapping(field project.Field) map[string]string {
	if field.DataType != "SINGLE_SELECT" || len(field.Options) == 0 {
		return nil
	}

	fmt.Printf("\nFound %s field with the following options:\n", field.Name)
	for i, opt := range field.Options {
		fmt.Printf("  %d. %s\n", i+1, opt.Name)
	}

	fmt.Printf("\nWould you like to configure %s field mappings? (y/N): ", strings.ToLower(field.Name))
	if p.scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(p.scanner.Text()))
		if response == "y" || response == "yes" {
			mappings := make(map[string]string)

			// Auto-generate mappings first
			for _, opt := range field.Options {
				lowerOpt := strings.ToLower(opt.Name)
				var key string

				if strings.EqualFold(field.Name, "Status") {
					switch {
					case strings.Contains(lowerOpt, "todo") || strings.Contains(lowerOpt, "backlog"):
						key = "todo"
					case strings.Contains(lowerOpt, "progress") || strings.Contains(lowerOpt, "doing"):
						key = "in_progress"
					case strings.Contains(lowerOpt, "review"):
						key = "in_review"
					case strings.Contains(lowerOpt, "done") || strings.Contains(lowerOpt, "complete"):
						key = "done"
					}
				} else if strings.EqualFold(field.Name, "Priority") {
					switch {
					case strings.Contains(lowerOpt, "low"):
						key = "low"
					case strings.Contains(lowerOpt, "medium") || strings.Contains(lowerOpt, "normal"):
						key = "medium"
					case strings.Contains(lowerOpt, "high"):
						key = "high"
					case strings.Contains(lowerOpt, "critical") || strings.Contains(lowerOpt, "urgent"):
						key = "critical"
					}
				}

				if key != "" {
					mappings[key] = opt.Name
				}
			}

			// Show auto-generated mappings
			if len(mappings) > 0 {
				fmt.Println("\nConfigured mappings:")
				for key, val := range mappings {
					fmt.Printf("  %s -> %s\n", key, val)
				}

				fmt.Print("\nWould you like to customize these mappings? (y/N): ")
				if p.scanner.Scan() {
					response := strings.ToLower(strings.TrimSpace(p.scanner.Text()))
					if response == "y" || response == "yes" {
						// Allow manual customization
						mappings = make(map[string]string)

						for _, opt := range field.Options {
							validKeys := p.getValidKeys(field.Name)
							fmt.Printf("\nMap '%s' to (%s/skip): ", opt.Name, strings.Join(validKeys, "/"))
							if p.scanner.Scan() {
								mapping := strings.TrimSpace(p.scanner.Text())
								if mapping != "" && mapping != "skip" {
									mappings[mapping] = opt.Name
								}
							}
						}
					}
				}
			}

			return mappings
		}
	}

	return nil
}

// GetStringInput prompts for a string input with an optional default value
func (p *InteractivePrompt) GetStringInput(prompt string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s (default: %s): ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	if p.scanner.Scan() {
		input := strings.TrimSpace(p.scanner.Text())
		if input == "" && defaultValue != "" {
			return defaultValue
		}
		return input
	}

	return defaultValue
}

// getValidKeys returns valid mapping keys for a field type
func (p *InteractivePrompt) getValidKeys(fieldName string) []string {
	if strings.EqualFold(fieldName, "Status") {
		return []string{"todo", "in_progress", "in_review", "done"}
	} else if strings.EqualFold(fieldName, "Priority") {
		return []string{"low", "medium", "high", "critical"}
	}
	return []string{}
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
