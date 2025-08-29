# Technology Stack: GitHub Project Manager (gh-pm)

## Architecture

### System Design
- **Type**: Hybrid CLI application (Go primary, Node.js secondary)
- **Pattern**: Command-based architecture using Cobra framework
- **Integration**: GitHub CLI extension utilizing `gh` API
- **Distribution**: Compiled binary (Go) + optional Node.js components

### Component Architecture
```
┌─────────────────┐
│  GitHub CLI     │
│    (gh)         │
└────────┬────────┘
         │
┌────────▼────────┐
│   gh-pm CLI     │
│  (Go Binary)    │
└────────┬────────┘
         │
    ┌────┼────┐
    │         │
┌───▼──┐  ┌──▼───────┐
│ Core │  │ Node.js  │
│ (Go) │  │ Features │
└──────┘  └──────────┘
```

## Primary Stack (Go)

### Core Language & Runtime
- **Language**: Go 1.23.0
- **Toolchain**: Go 1.24.4
- **Module System**: Go modules with semantic versioning

### Key Dependencies
```go
// CLI Framework
github.com/spf13/cobra v1.9.1       // Command-line interface
github.com/spf13/pflag v1.0.6       // POSIX/GNU-style flags

// GitHub Integration
github.com/cli/go-gh/v2 v2.12.1     // GitHub CLI Go library
github.com/cli/shurcooL-graphql     // GraphQL client for GitHub API

// Configuration
gopkg.in/yaml.v3 v3.0.1             // YAML parsing for config files

// Testing
github.com/stretchr/testify v1.7.0  // Testing assertions and mocks

// Terminal UI
github.com/muesli/termenv v0.16.0   // Terminal environment detection
github.com/mattn/go-isatty v0.0.20  // TTY detection
```

### Build System
```makefile
# Build commands
make build    # Build binary
make test     # Run tests
make install  # Install as gh extension
make release  # Create release artifacts
```

## Secondary Stack (Node.js)

### Runtime & Package Management
- **Runtime**: Node.js 18+ (ESM modules)
- **Package Manager**: npm/yarn
- **Module System**: ES Modules (`"type": "module"`)

### Dependencies (package.json)
```json
{
  "dependencies": {
    "commander": "^12.0.0",    // CLI framework
    "inquirer": "^9.2.15",      // Interactive prompts
    "js-yaml": "^4.1.0",        // YAML processing
    "chalk": "^5.3.0",          // Terminal colors
    "ora": "^8.0.1"             // Loading spinners
  },
  "devDependencies": {
    "jest": "^29.7.0",          // Testing framework
    "eslint": "^8.57.0",        // Code linting
    "prettier": "^3.2.5"        // Code formatting
  }
}
```

## Development Environment

### Required Tools
```bash
# Core Requirements
go version        # Go 1.23.0 or higher
gh version        # GitHub CLI 2.0.0+
git version       # Git for version control

# Optional (for Node.js features)
node --version    # Node.js 18+
npm --version     # npm package manager
```

### Development Setup
```bash
# Clone and setup
git clone https://github.com/yahsan2/gh-pm.git
cd gh-pm

# Go development
go mod download       # Download dependencies
go build             # Build binary
go test ./...        # Run all tests

# Node.js development (optional)
npm install          # Install Node dependencies
npm run dev          # Development mode
npm test            # Run Jest tests
```

## Common Commands

### Development Commands
```bash
# Go Commands
go run main.go              # Run without building
go build -o gh-pm          # Build binary
go test -v ./...           # Verbose test output
go mod tidy                # Clean up dependencies
go fmt ./...               # Format code
go vet ./...               # Static analysis

# Make Commands
make build                 # Production build
make test                  # Run test suite
make coverage             # Generate coverage report
make lint                 # Run linters
make clean                # Clean build artifacts

# Node.js Commands (for new features)
npm start                  # Run gh-pm CLI
npm run build             # Build JavaScript
npm run lint              # ESLint check
npm run format            # Prettier format
npm test                  # Jest tests
```

### GitHub CLI Integration
```bash
# Extension Management
gh extension install yahsan2/gh-pm     # Install
gh extension upgrade pm                # Update
gh extension remove pm                 # Uninstall

# API Testing
gh api graphql -f query='...'         # Test GraphQL queries
gh api repos/{owner}/{repo}/projects  # Test REST endpoints
```

## Environment Variables

### GitHub Configuration
```bash
# Authentication
GH_TOKEN            # GitHub personal access token
GITHUB_TOKEN        # Alternative token variable

# API Configuration
GH_HOST             # GitHub Enterprise Server host
GH_ENTERPRISE_TOKEN # Enterprise-specific token
```

### Application Configuration
```bash
# Development
DEBUG=true          # Enable debug logging
VERBOSE=true        # Verbose output
NO_COLOR=true       # Disable colored output

# Testing
TEST_MODE=true      # Run in test mode
MOCK_API=true       # Use mock API responses
```

### Build Configuration
```bash
# Version Info
VERSION             # Application version
BUILD_DATE          # Build timestamp
COMMIT_SHA          # Git commit hash

# Feature Flags
ENABLE_BETA=true    # Enable beta features
ENABLE_CACHE=true   # Enable response caching
```

## Port Configuration

### Development Ports
- **3000**: Node.js development server (future web UI)
- **8080**: Local API mock server for testing
- **9229**: Node.js debugger port

### Service Integration
- **443**: HTTPS for GitHub API calls
- **22**: SSH for git operations

## API Integration

### GitHub GraphQL API
```graphql
# Primary API for Projects v2
https://api.github.com/graphql

# Queries for:
- Project metadata
- Issue management
- Field updates
- Status tracking
```

### GitHub REST API
```bash
# Fallback for specific operations
https://api.github.com/repos/{owner}/{repo}

# Endpoints:
- /issues
- /projects
- /labels
- /milestones
```

## Testing Strategy

### Go Testing
```bash
# Unit Tests
pkg/config/config_test.go
pkg/init/detector_test.go
pkg/init/metadata_test.go
cmd/init_test.go

# Test Execution
go test -cover ./...
go test -race ./...
```

### Node.js Testing
```javascript
// Jest Configuration
{
  "testEnvironment": "node",
  "coverageThreshold": {
    "global": {
      "branches": 80,
      "functions": 80,
      "lines": 80
    }
  }
}
```

## Performance Considerations

### Optimization Strategies
- **Metadata Caching**: Store project IDs to reduce API calls
- **Parallel Processing**: Concurrent API requests where possible
- **Response Caching**: 15-minute cache for repeated queries
- **Lazy Loading**: Load features on-demand

### Resource Limits
- **API Rate Limits**: 5000 requests/hour (authenticated)
- **Memory Usage**: < 50MB for typical operations
- **Response Time**: < 2s for most commands
- **Binary Size**: ~10MB (Go binary)