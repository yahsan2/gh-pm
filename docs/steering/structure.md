# Project Structure

## Organization Philosophy

Standard Go CLI project layout with clear separation between command definitions (`cmd/`) and reusable business logic (`pkg/`). Commands are thin wrappers that delegate to packages for testability and reusability.

## Directory Patterns

### Commands (`/cmd/`)
**Purpose**: CLI command definitions and flag handling
**Pattern**: One file per command (e.g., `init.go`, `create.go`, `triage.go`)
**Example**:
```go
// cmd/create.go
var createCmd = &cobra.Command{
    Use:   "create",
    Short: "Create new issue",
    RunE:  runCreate,
}
```
- Commands register with root command via `init()`
- Minimal logic; delegates to `pkg/` packages

### Business Logic (`/pkg/`)
**Purpose**: Domain logic organized by capability
**Pattern**: Package per domain area
**Packages**:
- `pkg/config/` - Configuration loading and validation
- `pkg/project/` - Project API interactions and URL generation
- `pkg/issue/` - Issue CRUD operations, search, and batch processing
- `pkg/init/` - Initialization workflow (detection, metadata, prompts)
- `pkg/filter/` - Query parsing and filtering logic
- `pkg/output/` - Output formatters (table, JSON, CSV)
- `pkg/utils/` - Shared utilities (date conversion, etc.)

### Configuration (`/`)
**Location**: Root directory
**Files**: `.gh-pm.yml` (user-facing), `go.mod`, `Makefile`
**Example**: `.gh-pm.yml` contains project settings, field mappings, and cached metadata

### Tests
**Location**: Co-located with source as `*_test.go`
**Example**: `cmd/init_test.go`, `pkg/config/config_test.go`

### Build Artifacts (`/dist/`)
**Purpose**: GoReleaser output (ignored in `.gitignore`)

## Naming Conventions

- **Files**: lowercase with underscores for Go files (e.g., `date_conv.go`, `user_project.go`)
- **Packages**: lowercase, single word when possible (e.g., `config`, `project`, `issue`)
- **Commands**: lowercase, match CLI command names (e.g., `init.go`, `triage.go`)
- **Types**: PascalCase (e.g., `Config`, `ProjectMetadata`)
- **Functions**: camelCase, exported start with capital (e.g., `Load()`, `findConfigFile()`)

## Import Organization

```go
// Standard library imports first
import (
    "fmt"
    "os"

    // External dependencies
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"

    // Internal packages (absolute from module root)
    "github.com/yahsan2/gh-pm/pkg/config"
    "github.com/yahsan2/gh-pm/pkg/project"
)
```

**Module Path**: `github.com/yahsan2/gh-pm` (defined in `go.mod`)
**Import Style**: Absolute imports from module root

## Code Organization Principles

**Command-Package Separation**: Commands in `cmd/` handle CLI concerns (flags, output); packages in `pkg/` contain pure business logic.

**Package Independence**: Packages should not depend on `cmd/` package. Inter-package dependencies flow from specific to general (e.g., `issue` may use `config`, but not vice versa).

**Configuration Centralization**: Single `.gh-pm.yml` file manages all settings. Config package provides type-safe access throughout codebase.

**Test Isolation**: Tests use mocks for GitHub API interactions to avoid authentication requirements. See `pkg/issue/client_test.go` for mock examples.

**Entry Point**: `main.go` is minimal - calls `cmd.Execute()` and exits with status code.

---
_Created: 2025-12-08_
