# gh-pm

A GitHub CLI extension for project management with GitHub Projects (v2) and Issues. Streamline requirements definition, prioritization, task decomposition, and progress tracking from the command line.

## Features

- 📊 **Project Management** - List, create, update, and view issues in GitHub Projects v2
- 📥 **Issue Intake** - Find and add issues not in project with `gh issue list` compatible interface
- 🔍 **Issue Triage** - Bulk process issues with configurable rules and interactive mode
- ➗ **Issue Splitting** - Decompose issues into sub-issues using GitHub's native hierarchy
- 🎯 **Field Management** - Update Status and Priority fields directly from CLI
- 🔗 **Project Board Integration** - Direct links to GitHub Projects board views
- 🚀 **Dry-run Mode** - Preview changes before applying them
- 🎨 **Multiple output formats** - Table, JSON, CSV, and web browser views

## Installation

```bash
gh extension install yahsan2/gh-pm
```

### Update
```bash
gh extension upgrade pm
```

### Requirements
- GitHub CLI 2.0.0 or later
- GitHub account with repository and project permissions
- Access to GitHub Projects (v2)
- `gh-sub-issue` extension for split command: `gh extension install yahsan2/gh-sub-issue`

## Quick Start

### Initialize Configuration

The `init` command creates a `.gh-pm.yml` configuration file with automatic project detection and metadata caching for faster operations.

```bash
# Interactive initialization (auto-detects current repository and lists available projects)
gh pm init
```

**Features:**
- 🔍 **Auto-detection** - Automatically detects current repository and associated projects
- 📊 **Project selection** - Interactive selection from available projects
- 🚀 **Metadata caching** - Caches project/field/option IDs for faster API operations
- 🔧 **Smart field mapping** - Automatically maps common status/priority values (e.g., "Backlog" → "todo", "P0" → "critical")
- ✅ **Validation** - Verifies project access and field availability

### Basic Workflow
```bash
# Create a new issue with priority
gh pm create --title "Implement authentication" --priority p1 --label "backend"

# View issue details with project metadata
gh pm view 123

# Move issue to In Progress status
gh pm move 123 --status in_progress

# Find and add issues not in project
gh pm intake --label bug --apply "status:backlog,priority:p2"

# Split issue into sub-issues
gh pm split 123 --from=body

# Run triage to bulk update issues
gh pm triage tracked

# Preview triage changes without applying
gh pm triage estimate --list
```

## Command Reference

### Setup & Configuration
- [`gh pm init`](#gh-pm-init) - Initialize configuration with project detection

### Project Management
- [`gh pm list`](#list-issues) - List issues in project with filtering
- [`gh pm intake`](#issue-intake) - Find and add issues not in project
- [`gh pm create`](#create-issue) - Create new issue with project metadata
- [`gh pm view`](#view-issue) - View issue details with project info
- [`gh pm move`](#move-issue-update-project-fields) - Update issue status/priority

### Issue Organization
- [`gh pm split`](#split-issues-task-decomposition) - Split issue into sub-issues
- [`gh pm triage`](#triage-issues) - Bulk process issues with rules

## Core Commands

### Initialization

#### `gh pm init`

Initialize a new configuration file with automatic project detection and field mapping.

**Options:**
- `--project <name>` - Specify project name or number
- `--org <name>` - Organization name (for org projects)
- `--repo <owner/repo>` - Repository (can be specified multiple times)
- `--skip-metadata` - Skip fetching project metadata (creates simpler config)
- `--interactive` - Interactive mode (default: true)

**What it does:**
1. **Detects current repository** - Automatically identifies the current Git repository
2. **Lists available projects** - Shows all projects associated with the repository or organization
3. **Fetches project fields** - Retrieves Status and Priority fields from the selected project
4. **Maps field values** - Automatically maps values like "Backlog" → "todo", "P0" → "critical"
5. **Caches metadata** - Stores project and field IDs for faster API operations
6. **Creates config file** - Generates `.gh-pm.yml` with all settings

**Example workflow:**
```bash
$ gh pm init
Detecting projects for repository yahsan2/gh-pm...

Available projects from repository yahsan2/gh-pm:
----------------------------------------------------------------------
 1. gh-pm project                            #8
    URL: https://github.com/users/yahsan2/projects/8
----------------------------------------------------------------------

Select a project (0-1): 1
✓ Selected project: gh-pm project (#8)

Fetching project fields...
✓ Project metadata captured for faster operations

Found Status field with the following options:
  1. Backlog
  2. Ready
  3. In progress
  4. In review
  5. Done

Found Priority field with the following options:
  1. P0
  2. P1
  3. P2

✓ Configuration saved to .gh-pm.yml
```

### Project Management

#### List Issues

`gh pm list` command displays issues in the project with a `gh issue list` compatible interface. It also supports filtering by project-specific fields (Status, Priority, etc.).

**Basic Usage:**
```bash
# List all open issues in project
gh pm list

# Use alias 'ls'
gh pm ls

# Filter by state
gh pm list --state closed
gh pm list --state all

# Filter by project fields
gh pm list --status "in_progress"
gh pm list --priority "p0,p1"

# Filter by labels
gh pm list --label bug --label enhancement

# Filter by assignee
gh pm list --assignee @me
gh pm list --assignee octocat

# Filter by author
gh pm list --author johndoe

# Filter by milestone
gh pm list --milestone "v1.0"

# Search with query
gh pm list --search "authentication error"

# Limit results (default: 30)
gh pm list --limit 100

# Combined filters
gh pm list --state open --label bug --priority p0 --assignee @me
```

**Output Formats:**
```bash
# Output as JSON
gh pm list --json number,title,status,priority,assignees,labels

# Output all fields as JSON
gh pm list --json

# Process JSON with jq (coming soon)
gh pm list --json number,title --jq '.[] | select(.number > 100)'

# Format with Go template (coming soon)
gh pm list --json number,title --template '{{range .}}#{{.number}}: {{.title}}{{"\n"}}{{end}}'

# Open project board in web browser
gh pm list --web
```

**Filter Options:**
- `--state, -s` - Issue state (open/closed/all, default: open)
- `--label, -l` - Filter by label (multiple allowed)
- `--assignee, -a` - Filter by assignee (`@me` for self)
- `--author, -A` - Filter by author
- `--milestone, -m` - Milestone number or title
- `--search, -S` - Text search (title and body)
- `--mention` - Filter by mentioned user
- `--app` - Filter by GitHub App author
- `--limit, -L` - Maximum number to fetch (default: 30)

**Project-specific Filters:**
- `--status` - Filter by project Status field
- `--priority` - Filter by project Priority field (comma-separated for multiple)

**Output Example:**
```
#    TITLE                                    STATUS        PRIORITY  ASSIGNEES  LABELS
1    Implement authentication system          In Progress   P0        user1      bug, enhancement
2    Add user dashboard                       Backlog       P1        user2      feature
3    Fix database connection timeout          Done          P2        -          bug
```

#### Issue Intake

`gh pm intake` finds and adds issues not in the project with a `gh issue list` compatible interface. Automatically excludes issues already in the project.

**Key Features:**
- 🔍 **Flexible filtering** - Filter issues by labels, assignee, author, state, and more
- 🚫 **Duplicate prevention** - Automatically excludes issues already in the project
- 📊 **Bulk addition** - Add multiple issues to the project at once
- 🏷️ **Field configuration** - Set Status and Priority fields when adding issues
- 👀 **Preview mode** - Use `--dry-run` to preview changes without applying them

**Usage Examples:**
```bash
# List all open issues not in project
gh pm intake

# Filter by labels (multiple allowed)
gh pm intake --label bug --label enhancement

# Add your assigned issues
gh pm intake --assignee @me

# Search with query
gh pm intake --search "authentication error"

# Set project fields when adding
gh pm intake --apply "status:backlog,priority:p2"

# Dry-run mode (preview without adding)
gh pm intake --dry-run

# Filter by author
gh pm intake --author octocat

# Filter by milestone
gh pm intake --milestone "v1.0"

# Specify state (open, closed, all)
gh pm intake --state all

# Limit number of issues
gh pm intake --limit 50
```

**Filter Options:**
- `--label, -l` - Filter by label (multiple allowed)
- `--assignee, -a` - Filter by assignee (`@me` for self)
- `--author, -A` - Filter by author
- `--state, -s` - Issue state (open/closed/all, default: open)
- `--milestone, -m` - Milestone number or title
- `--search, -S` - GitHub search query
- `--mention` - Filter by mentioned user
- `--app` - Filter by GitHub App author
- `--limit, -L` - Maximum number of issues to fetch (default: 100)

**Additional Options:**
- `--dry-run` - Show what would be added without making changes
- `--apply` - Field values to set when adding (e.g., `status:backlog`, `priority:p2`)

**Process Flow:**
1. Search for issues using specified filters (`gh issue list` compatible)
2. Fetch issues already in the project
3. Exclude duplicates to create list of issues to add
4. Display target issues and request confirmation
5. After confirmation, add issues to project
6. Set field values specified by `--apply`

**Example: Add bug-labeled issues with priority P2**
```bash
$ gh pm intake --label bug --apply "status:backlog,priority:p2"
Fetching issues with filters...
Found 5 issues from search

Found 3 issues not in project:
  #45: Authentication fails on mobile
  #67: Database connection timeout
  #89: UI rendering issue in dark mode

Add 3 issues to project? (y/N): y
Adding issue #45 to project... ✓
Adding issue #67 to project... ✓
Adding issue #89 to project... ✓

Successfully added 3/3 issues to project
```

#### Create Issue

```bash
# Basic creation (will use defaults from .gh-pm.yml)
gh pm create --title "Add user dashboard"

# With full details
gh pm create \
  --title "Implement REST API" \
  --body "Create RESTful endpoints for user management" \
  --priority p0 \
  --status ready \
  --assignee "@me" \
  --label "api,backend" \
  --milestone "v1.0"

# Interactive mode (opens gh issue create interactively)
gh pm create --interactive

# Create from template
gh pm create --template bug

# Batch creation from file
gh pm create --from-file issues.yml
```

#### View Issue

```bash
# View issue with project metadata
gh pm view 123

# View with project URL (shows GitHub Projects board URL)
gh pm view 123 --quiet  # Only shows URLs

# Open in web browser (opens project board view)
gh pm view 123 --web

# View with comments
gh pm view 123 --comments

# View in different output formats
gh pm view 123 --output json
gh pm view 123 --output csv

# View issue in specific repository
gh pm view 456 --repo owner/repo
```

#### Move Issue (Update Project Fields)
```bash
# Update single field
gh pm move 123 --status in_review

# Update multiple fields
gh pm move 123 --status in_progress --priority p0

# Available status values (based on your project configuration)
gh pm move 15 --status ready
gh pm move 42 --status done

# Available priority values (based on your project configuration)
gh pm move 123 --priority p1  # High priority
gh pm move 456 --priority p2  # Medium priority

# Quiet mode (minimal output)
gh pm move 123 --status done --quiet

# Specify repository explicitly
gh pm move 123 --status ready --repo owner/repo
```

**Available Field Values:**
The exact field values depend on your project configuration (`.gh-pm.yml`):

- **Status**: `backlog`, `ready`, `in_progress`, `in_review`, `done`
- **Priority**: `p0` (Critical), `p1` (High), `p2` (Medium), etc.

**Important Notes:**
- The issue must already be added to the configured project
- Field values are case-sensitive and must match your project configuration
- Use `gh pm init` to see available values for your project

### Issue Organization

#### Split Issues (Task Decomposition)

Decompose parent issues into sub-issues using GitHub's native issue hierarchy feature. This command automatically creates linked sub-issues from task lists, maintaining parent-child relationships for better project organization.

**Requirements:**
- Requires `gh-sub-issue` extension: `gh extension install yahsan2/gh-sub-issue`
- The command will prompt for installation if the extension is not found

```bash
# Split from issue body checklist
gh pm split 123 --from=body

# Split from a file containing tasks
gh pm split 123 --from=./tasks.md

# Split from stdin
cat tasks.md | gh pm split 123

# Split from JSON array
gh pm split 123 '["Task 1", "Task 2", "Task 3"]'

# Split from command arguments
gh pm split 123 "Design API" "Implement backend" "Write tests"

# Specify repository explicitly
gh pm split 123 --from=body --repo owner/repo
```

**Features:**
- 🔍 **Duplicate Detection** - Automatically checks existing sub-issues to avoid duplicates
- 🏷️ **Label Inheritance** - Sub-issues inherit labels from the parent (except meta labels)
- 👥 **Assignee Inheritance** - Sub-issues inherit assignees from the parent
- 🎯 **Milestone Inheritance** - Sub-issues inherit milestone from the parent
- 🔗 **Native GitHub Integration** - Uses GitHub's built-in sub-issue hierarchy
- ✅ **Checklist Support** - Recognizes GitHub-style checkboxes (`- [ ]` format)

**Input Formats:**

1. **Issue Body** (`--from=body`): Extracts checklist items from the parent issue's description
   ```markdown
   - [ ] Design database schema
   - [ ] Implement API endpoints
   - [ ] Write unit tests
   ```

2. **File Input** (`--from=./file.md`): Reads tasks from a markdown file with checklist format

3. **JSON Array**: Pass tasks as a JSON array
   ```bash
   gh pm split 123 '["Task 1", "Task 2", "Task 3"]'
   ```

4. **Command Arguments**: Pass tasks directly as arguments
   ```bash
   gh pm split 123 "First task" "Second task" "Third task"
   ```

5. **Stdin**: Pipe tasks from another command
   ```bash
   echo "- [ ] Task 1\n- [ ] Task 2" | gh pm split 123
   ```

**Example Output:**
```
Checking for existing sub-issues and creating new ones for issue #123...
Found 2 existing sub-issues for issue #123
⏭️  Skipping (already exists): Design database schema
✓ Created sub-issue #124: Implement API endpoints
✓ Created sub-issue #125: Write unit tests

Skipped 1 task that already has sub-issues
```

**Notes:**
- Sub-issues are automatically linked to the parent using GitHub's native hierarchy
- The parent issue can be viewed with all sub-issues in GitHub's web interface
- Duplicate detection prevents creating the same sub-issue multiple times
- Sub-issues can be managed independently while maintaining their relationship to the parent

#### Triage Issues
```bash
# Run a triage configuration
gh pm triage tracked  # Applies labels and project fields to untracked issues

# Preview what would be changed (dry-run)
gh pm triage estimate --list
gh pm triage estimate --dry-run  # Same as --list

# Interactive triage (prompts for each issue)
gh pm triage estimate  # If configured with interactive fields

# Ad-hoc triage with query and apply (without configuration file)
gh pm triage --query="status:backlog -has:estimate" --apply="status:in_progress"

# Preview what would be changed without applying
gh pm triage --query="status:backlog" --apply="priority:p1" --list

# Ad-hoc triage with interactive mode for specific fields
gh pm triage --query="status:backlog" --interactive="status,estimate"
gh pm triage --query="-has:priority" --interactive="priority"

# List issues matching query without any changes
gh pm triage --query="status:backlog -has:estimate" --list
```

**Triage Configuration Example (.gh-pm.yml):**
```yaml
triage:
  # Auto-add tracking label and set defaults
  tracked:
    query: "is:issue is:open -label:pm-tracked"
    instruction: "Starting triage for untracked issues. This will add the pm-tracked label and set default project fields."
    apply:
      labels:
        - pm-tracked
      fields:
        priority: p1
        status: backlog
    interactive:
      status: true  # Prompt for status selection

  # Interactive estimation
  estimate:
    query: "is:issue is:open status:backlog -has:estimate"
    instruction: "Review issues that need estimation. Please provide time estimates for planning purposes."
    apply: {}
    interactive:
      estimate: true  # Prompt for estimate entry
```

**Configuration Fields:**
- `query`: GitHub search query to find issues to triage
- `instruction`: Optional message displayed at the start of triage operation (useful for providing context or instructions to users)
- `apply.labels`: Labels to automatically add to matching issues
- `apply.fields`: Project field values to automatically set
- `interactive.status`: Prompt for status selection for each issue
- `interactive.estimate`: Prompt for estimate entry for each issue

**Query Syntax Extensions:**

gh-pm extends GitHub's standard search syntax with project-specific filters:

- **Field filters**: `status:backlog`, `priority:critical` - Filter by project field values (case-insensitive)
- **Field exclusion** (gh-pm exclusive): `-has:estimate` - Find issues missing a field value
- **Label exclusion**: `-label:bug` - Exclude issues with specific labels
- **Combined queries**: `status:backlog -has:estimate -label:blocked`
- **Config field names**: Use either actual field names (`Status:backlog`) or config names (`status:backlog`)

Note: The `-has:` operator is a gh-pm extension not available in standard GitHub search. It uses GraphQL to filter issues missing specific project field values.

**Interactive Field Support:**

When using `--interactive`, the following field types are currently supported:
- `SINGLE_SELECT` - Select from predefined options (e.g., Status, Priority)
- `TEXT` - Free-form text input (e.g., custom text fields)
- `NUMBER` - Numeric values (e.g., story points, hours)

**Planned Support (In Development):**
The following field types are planned for future releases:
- `ITERATION` - Select project iterations ([#29](https://github.com/yahsan2/gh-pm/issues/29))
- `DATE` - Date picker with format validation ([#30](https://github.com/yahsan2/gh-pm/issues/30))
- `MILESTONE` - Select repository milestones ([#31](https://github.com/yahsan2/gh-pm/issues/31))
- `ASSIGNEES` - Assign users to issues ([#32](https://github.com/yahsan2/gh-pm/issues/32))
- `LABELS` - Add/remove issue labels ([#33](https://github.com/yahsan2/gh-pm/issues/33))

Not planned for interactive mode:
- `REPOSITORY`, `LINKED_PULL_REQUESTS` - These are read-only fields

## Configuration

### Project Configuration (.gh-pm.yml)

```yaml
# Project settings
project:
  name: "My Project"
  number: 1  # or project ID
  org: "my-organization"  # optional

# Repository settings
repositories:
  - owner/repo1
  - owner/repo2

# Default values
defaults:
  priority: medium
  status: "Todo"
  labels:
    - "pm-tracked"

# Custom fields mapping (automatically populated from project)
fields:
  priority:
    field: "Priority"
    values:
      p0: "P0"          # Critical
      p1: "P1"          # High
      p2: "P2"          # Medium

  status:
    field: "Status"
    values:
      backlog: "Backlog"
      ready: "Ready"
      in_progress: "In progress"
      in_review: "In review"
      done: "Done"

# Metadata cache (auto-generated by init command)
metadata:
  project:
    id: "PVT_kwHOAAlRwM4BBvYB"  # Project node ID for API calls
  fields:
    status:
      id: "PVTSSF_lAHOAAlRwM4BBvYBzg0KEU0"
      options:
        todo: "f75ad846"
        ready: "61e4505c"
        in_progress: "47fc9ee4"
        in_review: "df73e18b"
        done: "98236657"
    priority:
      id: "PVTSSF_lAHOAAlRwM4BBvYBzg0KEX4"
      options:
        critical: "79628723"
        high: "0a877460"
        medium: "da944a9c"
```

## Project URLs

gh-pm automatically generates correct GitHub Projects board URLs for all issues. These URLs open directly in the project board view, making it easy to navigate between CLI and web interfaces.

### URL Format
- **User projects**: `https://github.com/users/{owner}/projects/{number}?pane=issue&itemId={id}`
- **Organization projects**: `https://github.com/orgs/{org}/projects/{number}?pane=issue&itemId={id}`

### Commands with Project URL Support
- `gh pm view [issue]` - Shows project URL in output
- `gh pm view [issue] --quiet` - Shows only the project URL
- `gh pm view [issue] --web` - Opens project board in browser
- `gh pm triage [name] --list` - Shows project URLs for all affected issues
- `gh pm create` - Returns project URL after creation

## Output Formats

### Table (Default)
```
┌─────┬──────────────────────┬──────────┬────────────┬──────────┐
│ #   │ Title                │ Priority │ Status     │ Assignee │
├─────┼──────────────────────┼──────────┼────────────┼──────────┤
│ 123 │ Implement auth       │ High     │ In Progress│ @johndoe │
│ 124 │ Add user dashboard   │ Medium   │ Todo       │ @janedoe │
└─────┴──────────────────────┴──────────┴────────────┴──────────┘
```

### JSON
```json
{
  "issues": [
    {
      "number": 123,
      "title": "Implement auth",
      "priority": "high",
      "status": "in_progress",
      "assignee": "johndoe"
    }
  ]
}
```

### CSV
```csv
number,title,priority,status,assignee
123,"Implement auth",high,in_progress,johndoe
124,"Add user dashboard",medium,todo,janedoe
```

## Integration

### GitHub Actions

```yaml
name: Project Management
on:
  issues:
    types: [opened, edited]

jobs:
  update-project:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Install gh-pm extension
        run: gh extension install yahsan2/gh-pm
      - name: Update project fields
        run: |
          gh pm move ${{ github.event.issue.number }} \
            --status "backlog" \
            --priority p2
```

### Git Hooks

```bash
# .git/hooks/post-commit
#!/bin/bash
# Auto-update issue status on commit
if [[ $(git log -1 --pretty=%B) =~ "#([0-9]+)" ]]; then
  gh pm update "${BASH_REMATCH[1]}" --status "In Review"
fi
```

## Troubleshooting

### Authentication Issues
```bash
# Check authentication
gh auth status

# Re-authenticate
gh auth login

# Use specific token
export GH_TOKEN=your_token_here
```

### Project Access
```bash
# List accessible projects
gh pm projects list

# Check permissions
gh pm debug permissions
```

### Performance
```bash
# Enable caching
gh pm config set cache true

# Clear cache
gh pm cache clear

# Verbose output for debugging
gh pm list --verbose
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Clone repository
git clone https://github.com/yahsan2/gh-pm.git
cd gh-pm

# Install dependencies
npm install  # or appropriate package manager

# Run tests
npm test

# Build
npm run build
```

## Roadmap

### ✅ Completed Features
- [x] **Issue creation** with project metadata (`gh pm create`)
- [x] **Project initialization** with auto-detection (`gh pm init`)
- [x] **Issue status & priority updates** (`gh pm move`)
- [x] **Issue viewing** with project metadata and URLs (`gh pm view`)
- [x] **Issue intake** with gh issue list compatible interface (`gh pm intake`)
- [x] **Triage operations** for bulk issue processing (`gh pm triage`)
- [x] **Issue splitting** into sub-issues with duplicate detection (`gh pm split`)
- [x] **Configuration management** with field mapping
- [x] **Multiple output formats** (table, JSON, CSV)
- [x] **Project URL generation** for direct GitHub Projects board access
- [x] **Dry-run mode** for previewing changes before applying

### 🚧 In Development / Planned
- [x] Issue listing and filtering (`gh pm list`)
- [ ] Bulk operations and CSV import/export
- [ ] Progress tracking and reporting (`gh pm status`)
- [ ] Sprint management features

### 🔮 Future Features
- [ ] Sprint management (`gh pm sprint ...`)
- [ ] Gantt chart visualization
- [ ] Time tracking integration
- [ ] Custom workflow automation
- [ ] AI-powered task suggestions
- [ ] Mobile companion app
- [ ] Slack/Discord integration

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

- Built on [GitHub CLI](https://cli.github.com/)
- Inspired by modern project management best practices
- Thanks to all contributors and users

## Support

- 🐛 [Report bugs](https://github.com/yahsan2/gh-pm/issues)
- 💡 [Request features](https://github.com/yahsan2/gh-pm/discussions)
- 📖 [Read documentation](https://github.com/yahsan2/gh-pm/wiki)
- 💬 [Join discussions](https://github.com/yahsan2/gh-pm/discussions)

---

Made with ❤️ for GitHub project managers
