# gh-pm

A GitHub CLI extension for project management with GitHub Projects (v2) and Issues. Streamline requirements definition, prioritization, task decomposition, and progress tracking from the command line.

## Features

- 📊 **Project Management** - Manage GitHub Projects v2 directly from CLI
- 🔄 **Issue Workflow** - Create, update, view, and track issues with rich metadata
- 📥 **Issue Intake** - Find and add issues not in project with gh issue list compatible interface
- 🔍 **Issue Triage** - Bulk process issues with configurable rules and interactive mode
- 🎯 **Priority Management** - Set and track priorities across issues
- 🔗 **Project Board Integration** - Direct links to GitHub Projects board views
- 📈 **Progress Tracking** - Monitor task completion and project status
- ➗ **Issue Splitting** - Decompose issues into sub-issues using GitHub's native hierarchy
- 🚀 **Dry-run Mode** - Preview changes before applying them
- 🎨 **Multiple output formats** - Table, JSON, CSV, and quiet modes

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

### Issue Intake

#### `gh pm intake`

プロジェクトに含まれていない Issue を検索し、選択的にプロジェクトに追加するコマンドです。`gh issue list` と互換性のあるフィルタリングオプションを提供し、既存のプロジェクトIssueを自動的に除外します。

**主な機能:**
- 🔍 **柔軟なフィルタリング** - ラベル、アサイン、著者、状態などで Issue を絞り込み
- 🚫 **重複回避** - プロジェクトに既に存在する Issue を自動除外
- 📊 **一括追加** - 複数の Issue を一度にプロジェクトに追加
- 🏷️ **フィールド設定** - 追加時に Status や Priority を自動設定
- 👀 **プレビューモード** - `--dry-run` で実際の変更なしに確認

**使用例:**
```bash
# プロジェクトに含まれていない全ての open issue を表示
gh pm intake

# ラベルでフィルタリング（複数指定可能）
gh pm intake --label bug --label enhancement

# アサイン済みの自分の issue を追加
gh pm intake --assignee @me

# 検索クエリで絞り込み
gh pm intake --search "authentication error"

# 追加時にプロジェクトフィールドを設定
gh pm intake --apply "status:backlog,priority:p2"

# ドライランモード（実際には追加せずプレビュー）
gh pm intake --dry-run

# 特定の著者の issue を検索
gh pm intake --author octocat

# マイルストーンでフィルタリング
gh pm intake --milestone "v1.0"

# 状態を指定（open, closed, all）
gh pm intake --state all

# 取得数を制限
gh pm intake --limit 50
```

**フィルタオプション:**
- `--label, -l` - ラベルでフィルタ（複数指定可能）
- `--assignee, -a` - アサイン先でフィルタ（`@me` で自分）
- `--author, -A` - 作成者でフィルタ
- `--state, -s` - Issue の状態（open/closed/all、デフォルト: open）
- `--milestone, -m` - マイルストーン番号またはタイトル
- `--search, -S` - GitHub 検索クエリ
- `--mention` - メンションされたユーザーでフィルタ
- `--app` - GitHub App 作成者でフィルタ
- `--limit, -L` - 取得する Issue の最大数（デフォルト: 100）

**追加オプション:**
- `--dry-run` - 実際に追加せずに何が追加されるかを表示
- `--apply` - 追加時に設定するフィールド値（例: `status:backlog`, `priority:p2`）

**処理フロー:**
1. 指定されたフィルタで Issue を検索（`gh issue list` 互換）
2. プロジェクトに既に存在する Issue を取得
3. 重複を除外して追加対象の Issue リストを作成
4. 対象 Issue を表示し、ユーザーに確認
5. 確認後、Issue をプロジェクトに追加
6. `--apply` で指定されたフィールド値を設定

**例: バグラベルの Issue を優先度 P2 で追加**
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

### Project Management

#### List Issues
```bash
# List all issues in current project
gh pm list

# Filter by status
gh pm list --status "In Progress"

# Filter by priority
gh pm list --priority high,critical

# JSON output
gh pm list --json number,title,priority,status
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

### Issue Splitting (Task Decomposition)

#### Split Command

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

### View Issue Details

#### View Command
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

### Triage Operations

#### Triage Command
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

### Priority Management

#### Set Priority
```bash
# Set single issue priority using move command
gh pm move 123 --priority p0  # Critical priority

# Set priority with status update
gh pm move 123 --status in_progress --priority p1

# Note: Bulk priority updates coming in future releases
# gh pm set-priority 123,124,125 --level critical  # Coming soon
```

#### Priority Matrix
```bash
# View priority matrix
gh pm priority-matrix

# Export as CSV
gh pm priority-matrix --output csv > priorities.csv
```

### Progress Tracking

#### Project Status
```bash
# Overall project status
gh pm status

# Detailed progress report
gh pm status --detailed

# Specific milestone
gh pm status --milestone "v1.0"
```

#### Burndown
```bash
# Sprint burndown (when sprint support is added)
gh pm burndown

# Custom date range
gh pm burndown --from 2024-01-01 --to 2024-01-31
```

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

### Global Configuration

```bash
# Set default project
gh pm config set default-project "My Project"

# Set default output format
gh pm config set output-format json

# View all settings
gh pm config list
```

## Advanced Usage

### Cross-Repository Operations

```bash
# Create issue in specific repo
gh pm create --repo owner/other-repo --title "Cross-repo task"

# List issues from multiple repos
gh pm list --repo owner/repo1,owner/repo2

# Move issue between repos
gh pm move 123 --to owner/other-repo
```

### Bulk Operations

```bash
# Bulk update from CSV
gh pm bulk-update --file updates.csv

# Export issues to CSV
gh pm export --format csv --output issues.csv

# Import issues from JSON
gh pm import --file issues.json
```

### Templates

```bash
# Create issue from template
gh pm create --template bug-report

# List available templates
gh pm templates list

# Create custom template
gh pm templates create --name "feature-request"
```

### Automation

```bash
# Watch for status changes
gh pm watch --interval 30s

# Run webhook on changes
gh pm watch --webhook https://example.com/hook

# Generate daily report
gh pm report daily --email team@example.com
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
      - name: Update project
        run: |
          gh pm update ${{ github.event.issue.number }} \
            --status "Todo" \
            --priority medium
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
- [ ] Issue listing and filtering (`gh pm list`)
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
