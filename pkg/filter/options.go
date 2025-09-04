package filter

// IssueFilters contains common filtering options compatible with gh issue list
type IssueFilters struct {
	// GitHub issue list compatible filters
	Labels    []string `json:"labels,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Author    string   `json:"author,omitempty"`
	State     string   `json:"state,omitempty"`
	Milestone string   `json:"milestone,omitempty"`
	Search    string   `json:"search,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Mention   string   `json:"mention,omitempty"`
	App       string   `json:"app,omitempty"`

	// Project-specific filters
	Status   string `json:"status,omitempty"`
	Priority string `json:"priority,omitempty"`
}

// NewIssueFilters creates a new IssueFilters with default values
func NewIssueFilters() *IssueFilters {
	return &IssueFilters{
		State: "open",
		Limit: 100,
	}
}

// ProjectIssue represents an issue with project-specific fields
type ProjectIssue struct {
	Number     int                    `json:"number"`
	Title      string                 `json:"title"`
	State      string                 `json:"state"`
	URL        string                 `json:"url"`
	ID         string                 `json:"id"`
	Body       string                 `json:"body,omitempty"`
	Author     string                 `json:"author,omitempty"`
	Assignees  []string               `json:"assignees,omitempty"`
	Labels     []string               `json:"labels,omitempty"`
	Milestone  string                 `json:"milestone,omitempty"`
	CreatedAt  string                 `json:"createdAt,omitempty"`
	UpdatedAt  string                 `json:"updatedAt,omitempty"`
	ClosedAt   string                 `json:"closedAt,omitempty"`
	Comments   int                    `json:"comments,omitempty"`
	ProjectURL string                 `json:"projectUrl,omitempty"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

// GitHubIssue represents a basic GitHub issue
type GitHubIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	ID     string `json:"node_id"`
	URL    string `json:"html_url"`
}
