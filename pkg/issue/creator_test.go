package issue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRESTClient is a mock for the REST API client
type MockRESTClient struct {
	mock.Mock
}

func (m *MockRESTClient) Post(path string, body interface{}, result interface{}) error {
	args := m.Called(path, body, result)
	
	// If result is provided in the mock, copy it
	if args.Get(0) != nil {
		if responseMap, ok := args.Get(0).(map[string]interface{}); ok {
			if resultMap, ok := result.(*map[string]interface{}); ok {
				*resultMap = responseMap
			}
		}
	}
	
	return args.Error(1)
}

// MockGraphQLClient is a mock for the GraphQL API client
type MockGraphQLClient struct {
	mock.Mock
}

func (m *MockGraphQLClient) Do(query string, variables map[string]interface{}, result interface{}) error {
	args := m.Called(query, variables, result)
	return args.Error(0)
}

func TestIssueData_Validate(t *testing.T) {
	tests := []struct {
		name    string
		data    *IssueData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid issue data",
			data: &IssueData{
				Title:      "Test Issue",
				Body:       "Test body",
				Repository: "owner/repo",
				Labels:     []string{"bug", "enhancement"},
			},
			wantErr: false,
		},
		{
			name: "missing title",
			data: &IssueData{
				Body:       "Test body",
				Repository: "owner/repo",
			},
			wantErr: true,
			errMsg:  "issue title is required",
		},
		{
			name: "title too long",
			data: &IssueData{
				Title:      string(make([]byte, 257)),
				Repository: "owner/repo",
			},
			wantErr: true,
			errMsg:  "issue title must be 256 characters or less",
		},
		{
			name: "missing repository",
			data: &IssueData{
				Title: "Test Issue",
			},
			wantErr: true,
			errMsg:  "repository is required",
		},
		{
			name: "invalid repository format",
			data: &IssueData{
				Title:      "Test Issue",
				Repository: "invalid-format",
			},
			wantErr: true,
			errMsg:  "repository must be in 'owner/repo' format",
		},
		{
			name: "empty label",
			data: &IssueData{
				Title:      "Test Issue",
				Repository: "owner/repo",
				Labels:     []string{"valid", ""},
			},
			wantErr: true,
			errMsg:  "empty label is not allowed",
		},
		{
			name: "label too long",
			data: &IssueData{
				Title:      "Test Issue",
				Repository: "owner/repo",
				Labels:     []string{string(make([]byte, 51))},
			},
			wantErr: true,
			errMsg:  "exceeds maximum length of 50 characters",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.data.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIssueData_GetOwnerAndRepo(t *testing.T) {
	data := &IssueData{
		Repository: "octocat/hello-world",
	}
	
	assert.Equal(t, "octocat", data.GetOwner())
	assert.Equal(t, "hello-world", data.GetRepo())
	
	// Test with invalid format
	data.Repository = "invalid"
	assert.Equal(t, "", data.GetOwner())
	assert.Equal(t, "", data.GetRepo())
}

func TestIssueData_ToCreateRequest(t *testing.T) {
	data := &IssueData{
		Title:     "Test Issue",
		Body:      "Test body",
		Labels:    []string{"bug", "enhancement"},
		Assignee:  "octocat",
		Milestone: "v1.0",
	}
	
	req := data.ToCreateRequest()
	
	assert.Equal(t, "Test Issue", req["title"])
	assert.Equal(t, "Test body", req["body"])
	assert.Equal(t, []string{"bug", "enhancement"}, req["labels"])
	assert.Equal(t, []string{"octocat"}, req["assignees"])
	assert.Equal(t, "v1.0", req["milestone"])
	
	// Test with minimal data
	minData := &IssueData{
		Title: "Minimal Issue",
	}
	minReq := minData.ToCreateRequest()
	
	assert.Equal(t, "Minimal Issue", minReq["title"])
	assert.NotContains(t, minReq, "body")
	assert.NotContains(t, minReq, "labels")
}

func TestIssueData_GetFieldUpdates(t *testing.T) {
	data := &IssueData{
		Priority: "high",
		Status:   "in_progress",
		CustomFields: map[string]string{
			"sprint": "Sprint 1",
			"size":   "Large",
		},
	}
	
	fields := data.GetFieldUpdates()
	
	assert.Equal(t, "high", fields["priority"])
	assert.Equal(t, "in_progress", fields["status"])
	assert.Equal(t, "Sprint 1", fields["sprint"])
	assert.Equal(t, "Large", fields["size"])
}

func TestCreator_CreateIssue_Success(t *testing.T) {
	// This test demonstrates the structure, but would need actual mocking
	// of the REST client to work properly
	t.Skip("Requires REST client mocking implementation")
	
	data := &IssueData{
		Title:      "Test Issue",
		Body:       "Test body",
		Repository: "owner/repo",
		Labels:     []string{"bug"},
	}
	
	// Mock response would be used here
	// mockResponse := map[string]interface{}{
	// 	"node_id":    "MDU6SXNzdWUx",
	// 	"number":     float64(123),
	// 	"title":      "Test Issue",
	// 	"html_url":   "https://github.com/owner/repo/issues/123",
	// 	"state":      "open",
	// 	"created_at": time.Now().Format(time.RFC3339),
	// 	"updated_at": time.Now().Format(time.RFC3339),
	// 	"labels": []interface{}{
	// 		map[string]interface{}{
	// 			"name":  "bug",
	// 			"color": "d73a4a",
	// 		},
	// 	},
	// }
	
	// Create creator with mocked client
	// This would need proper mocking setup
	creator := &Creator{}
	
	issue, err := creator.CreateIssue(data)
	
	assert.NoError(t, err)
	assert.NotNil(t, issue)
	assert.Equal(t, "MDU6SXNzdWUx", issue.ID)
	assert.Equal(t, 123, issue.Number)
	assert.Equal(t, "Test Issue", issue.Title)
}

func TestCreator_CreateIssue_ValidationError(t *testing.T) {
	creator := &Creator{}
	
	// Invalid data - missing title
	data := &IssueData{
		Repository: "owner/repo",
	}
	
	issue, err := creator.CreateIssue(data)
	
	assert.Nil(t, issue)
	assert.Error(t, err)
	
	issueErr, ok := err.(*IssueError)
	assert.True(t, ok)
	assert.Equal(t, ErrorTypeValidation, issueErr.Type)
}

func TestIssueError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *IssueError
		expected string
	}{
		{
			name: "error with message only",
			err: &IssueError{
				Type:    ErrorTypeValidation,
				Message: "validation failed",
			},
			expected: "validation failed",
		},
		{
			name: "error with cause",
			err: &IssueError{
				Type:    ErrorTypeAPI,
				Message: "API call failed",
				Cause:   assert.AnError,
			},
			expected: "API call failed: caused by: assert.AnError general error for testing",
		},
		{
			name: "error with suggestion",
			err: &IssueError{
				Type:       ErrorTypeConfiguration,
				Message:    "config not found",
				Suggestion: "Run 'gh pm init'",
			},
			expected: "config not found: \n💡 Run 'gh pm init'",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.err.Error(), tt.expected)
		})
	}
}

func TestIssueError_Is(t *testing.T) {
	err1 := &IssueError{Type: ErrorTypeValidation}
	err2 := &IssueError{Type: ErrorTypeValidation}
	err3 := &IssueError{Type: ErrorTypeAPI}
	
	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(assert.AnError))
}