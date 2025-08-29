# Project Structure: GitHub Project Manager (gh-pm)

## Root Directory Organization

```
gh-pm/
├── main.go                 # Go application entry point
├── go.mod                  # Go module definition and dependencies
├── go.sum                  # Go dependency checksums
├── package.json            # Node.js package configuration (new features)
├── Makefile               # Build automation and common tasks
├── LICENSE                # MIT license file
├── README.md              # Project documentation
├── gh-pm                  # Compiled binary (git-ignored)
├── .gh-pm.yml             # Example configuration file
├── cmd/                   # Command implementations (Go)
├── pkg/                   # Core packages (Go)
├── src/                   # Node.js source code (future features)
├── bin/                   # Executable scripts (future)
├── test/                  # Integration tests
├── node_modules/          # Node.js dependencies (git-ignored)
├── .kiro/                 # Kiro spec-driven development
└── .claude/               # Claude Code configuration
```

## Subdirectory Structures

### `/cmd` - Command Layer (Go)
```
cmd/
├── root.go                # Root command and global flags
├── init.go                # Initialize configuration command
├── init_test.go           # Init command tests
├── create.go              # Create issue command (planned)
├── list.go                # List issues command (planned)
├── update.go              # Update issue command (planned)
└── status.go              # Project status command (planned)
```

**Purpose**: Contains all CLI command implementations using Cobra framework

### `/pkg` - Core Packages (Go)
```
pkg/
├── config/
│   ├── config.go          # Configuration management
│   └── config_test.go     # Configuration tests
├── init/
│   ├── detector.go        # Project auto-detection
│   ├── detector_test.go   # Detector tests
│   ├── metadata.go        # Project metadata fetching
│   ├── metadata_test.go   # Metadata tests
│   ├── prompt.go          # Interactive prompts
│   └── errors.go          # Error definitions
├── project/
│   └── project.go         # GitHub Projects v2 API client
├── issue/                 # Issue management (planned)
├── api/                   # GitHub API wrappers (planned)
└── utils/                 # Shared utilities (planned)
```

**Purpose**: Reusable packages following Go best practices

### `/src` - Node.js Features
```
src/
├── commands/
│   ├── create.js          # gh pm create command (Node.js)
│   └── index.js           # Command registry
├── services/
│   ├── config.js          # Configuration service
│   ├── github-api.js      # GitHub API client
│   ├── template.js        # Template management
│   └── issue.js           # Issue operations
├── templates/
│   ├── basic.yml          # Basic project template
│   ├── scrum.yml          # Scrum board template
│   └── feature.yml        # Feature development template
├── utils/
│   ├── logger.js          # Logging utilities
│   ├── spinner.js         # Progress indicators
│   └── validator.js       # Input validation
└── index.js               # Module exports
```

**Purpose**: Node.js implementation for advanced features with rich UI

### `/bin` - Executable Scripts
```
bin/
└── gh-pm.js               # Node.js CLI entry point
```

**Purpose**: Executable scripts for Node.js features

### `/.kiro` - Spec-Driven Development
```
.kiro/
├── steering/              # Project knowledge base
│   ├── product.md         # Product overview
│   ├── tech.md           # Technology stack
│   └── structure.md      # This file
└── specs/                # Feature specifications
    ├── gh-pm-create/     # Create command spec
    │   ├── config.json   # Spec configuration
    │   ├── requirements.md
    │   ├── design.md
    │   └── tasks.md
    └── init-command/     # Init command spec
```

**Purpose**: Kiro methodology for AI-assisted development

### `/.claude` - Claude Code Configuration
```
.claude/
├── commands/             # Custom slash commands
└── CLAUDE.md            # Project-specific AI instructions
```

**Purpose**: Claude Code IDE configuration and customization

### `/test` - Integration Tests
```
test/
├── fixtures/            # Test data and mocks
├── integration/         # End-to-end tests
└── scripts/            # Test automation scripts
```

**Purpose**: Comprehensive testing beyond unit tests

## Code Organization Patterns

### Go Code Patterns
```go
// Package structure
package cmd         // Commands
package config      // Configuration
package init        // Initialization logic
package project     // Domain logic

// Interface pattern
type ProjectClient interface {
    GetProject(id string) (*Project, error)
    UpdateField(field, value string) error
}

// Error handling
if err != nil {
    return fmt.Errorf("failed to %s: %w", action, err)
}
```

### Node.js Code Patterns
```javascript
// ES Module imports
import { Command } from 'commander';
import { GitHubAPIClient } from '../services/github-api.js';

// Class-based services
export class CreateCommand {
    constructor(options) {
        this.options = options;
    }
    
    async execute() {
        // Implementation
    }
}

// Async/await pattern
try {
    const result = await apiClient.createProject(data);
    return result;
} catch (error) {
    throw new Error(`Failed to create project: ${error.message}`);
}
```

## File Naming Conventions

### Go Files
- **Commands**: `{command}.go` (e.g., `init.go`, `create.go`)
- **Tests**: `{name}_test.go` (e.g., `init_test.go`)
- **Packages**: Lowercase, single word preferred (`config`, `project`)
- **Interfaces**: `{domain}Client` or `{domain}Service`

### JavaScript/Node.js Files
- **Commands**: `{command}.js` (e.g., `create.js`)
- **Services**: `{service-name}.js` (kebab-case)
- **Tests**: `{name}.test.js` or `{name}.spec.js`
- **Configuration**: `{name}.config.js`

### Configuration Files
- **YAML**: `.gh-pm.yml` (application config)
- **JSON**: `config.json`, `package.json`
- **Markdown**: `README.md`, `{UPPERCASE}.md` for docs

## Import Organization

### Go Imports
```go
import (
    // Standard library
    "fmt"
    "os"
    
    // Third-party packages
    "github.com/spf13/cobra"
    "github.com/cli/go-gh/v2"
    
    // Internal packages
    "github.com/yahsan2/gh-pm/pkg/config"
    "github.com/yahsan2/gh-pm/pkg/project"
)
```

### JavaScript Imports
```javascript
// Node.js built-ins
import fs from 'fs';
import path from 'path';

// Third-party packages
import { Command } from 'commander';
import inquirer from 'inquirer';

// Internal modules
import { ConfigService } from '../services/config.js';
import { templates } from '../templates/index.js';
```

## Key Architectural Principles

### Separation of Concerns
- **Commands**: User interaction and orchestration
- **Services/Packages**: Business logic and API interaction
- **Utils**: Shared, stateless helper functions
- **Config**: Centralized configuration management

### Dependency Injection
```go
// Go example
func NewInitCommand(detector Detector, client ProjectClient) *cobra.Command {
    // Command uses injected dependencies
}
```

```javascript
// JavaScript example
class CreateCommand {
    constructor(apiClient, templateService, configService) {
        // Services injected for testability
    }
}
```

### Error Handling Strategy
1. **Wrap errors** with context at each layer
2. **Return early** on error conditions
3. **User-friendly messages** at command layer
4. **Detailed logging** for debugging

### Testing Philosophy
- **Unit tests** next to implementation files
- **Integration tests** in `/test` directory
- **Mock external dependencies**
- **Test public APIs**, not implementation details

### Configuration Management
1. **Defaults** in code
2. **Project config** in `.gh-pm.yml`
3. **Environment variables** override
4. **CLI flags** highest priority

## Build Artifacts

### Ignored Files (gitignore)
```
# Binaries
gh-pm
*.exe
dist/

# Dependencies
node_modules/
vendor/

# Test artifacts
coverage/
*.test
*.out

# IDE
.vscode/
.idea/
*.swp

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local
```

### Distribution Files
- `gh-pm` - Linux/Mac binary
- `gh-pm.exe` - Windows binary
- `dist/` - Release artifacts
- `npm package` - Node.js distribution