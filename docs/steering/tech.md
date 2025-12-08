# Technology Stack

## Architecture

CLI extension architecture built on GitHub CLI's official SDK, using a command-pattern design with Cobra for routing and YAML-based configuration management.

## Core Technologies

- **Language**: Go 1.23+
- **CLI Framework**: spf13/cobra for command structure and flag parsing
- **GitHub Integration**: cli/go-gh/v2 (official GitHub CLI SDK) for API access and authentication
- **Configuration**: gopkg.in/yaml.v3 for `.gh-pm.yml` parsing

## Key Libraries

- `cli/go-gh/v2` - GitHub CLI SDK for authenticated API calls and GraphQL queries
- `spf13/cobra` - Command-line interface framework
- `yaml.v3` - Configuration file management
- `stretchr/testify` - Testing assertions and mocking

## Development Standards

### Code Organization
- Package-based structure: `cmd/` for CLI commands, `pkg/` for reusable logic
- Separation of concerns: config, project, issue, filter, output as distinct packages
- Command files map 1:1 with CLI commands (e.g., `cmd/init.go`, `cmd/triage.go`)

### Testing
- Unit tests co-located with source files (`*_test.go`)
- Mock GitHub API clients to avoid authentication in tests
- Coverage tracking via `make test-coverage`

### Quality Tools
- `golangci-lint` for comprehensive linting
- `go fmt` for consistent formatting
- `go vet` for static analysis
- Pre-commit hooks via `.pre-commit-config.yaml`

## Development Environment

### Required Tools
- Go 1.23 or later
- GitHub CLI 2.0+ (`gh`)
- golangci-lint (for linting)

### Common Commands
```bash
# Dev: make dev          # format, vet, build, install
# Build: make build      # compile binary
# Test: make test        # run tests
# Lint: make lint        # run golangci-lint
```

## Key Technical Decisions

**CLI Extension Model**: Built as `gh` extension rather than standalone CLI to leverage GitHub's authentication, configuration, and ecosystem integration.

**GraphQL for Projects API**: GitHub Projects v2 API is GraphQL-only; uses `cli/shurcooL-graphql` for type-safe queries.

**Configuration Caching**: Stores project and field IDs in `.gh-pm.yml` metadata section to minimize API calls and improve performance.

**Cobra Command Structure**: Each subcommand is a separate file in `cmd/`, sharing common flags via root command's persistent flags.

**Package-Based Architecture**: Clear separation between CLI layer (`cmd/`), business logic (`pkg/issue`, `pkg/project`), and infrastructure (`pkg/config`, `pkg/output`).

---
_Created: 2025-12-08_
