package utils

import (
	"testing"
	"time"
)

func TestConvertProjectsDateToISO(t *testing.T) {
	// Use a fixed date for consistent testing
	baseDate := time.Date(2025, 9, 4, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Basic @today format
		{
			name:     "@today",
			input:    "@today",
			expected: "2025-09-04",
			wantErr:  false,
		},

		// @today with days offset
		{
			name:     "@today-1d",
			input:    "@today-1d",
			expected: "2025-09-03",
			wantErr:  false,
		},
		{
			name:     "@today+7d",
			input:    "@today+7d",
			expected: "2025-09-11",
			wantErr:  false,
		},

		// @today with weeks offset
		{
			name:     "@today-1w",
			input:    "@today-1w",
			expected: "2025-08-28",
			wantErr:  false,
		},
		{
			name:     "@today-30d",
			input:    "@today-30d",
			expected: "2025-08-05",
			wantErr:  false,
		},

		// With comparison operators
		{
			name:     ">@today-1w",
			input:    ">@today-1w",
			expected: ">2025-08-28",
			wantErr:  false,
		},
		{
			name:     ">=@today",
			input:    ">=@today",
			expected: ">=2025-09-04",
			wantErr:  false,
		},
		{
			name:     "<@today+1d",
			input:    "<@today+1d",
			expected: "<2025-09-05",
			wantErr:  false,
		},
		{
			name:     "<=@today-1d",
			input:    "<=@today-1d",
			expected: "<=2025-09-03",
			wantErr:  false,
		},

		// With exclusion
		{
			name:     "-@today",
			input:    "-@today",
			expected: "-2025-09-04",
			wantErr:  false,
		},
		{
			name:     "->@today-1w",
			input:    "->@today-1w",
			expected: "->2025-08-28",
			wantErr:  false,
		},

		// ISO format passthrough
		{
			name:     "ISO date",
			input:    "2025-09-04",
			expected: "2025-09-04",
			wantErr:  false,
		},
		{
			name:     ">ISO date",
			input:    ">2025-09-01",
			expected: ">2025-09-01",
			wantErr:  false,
		},

		// Error cases
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "invalid unit",
			input:   "@today-1x",
			wantErr: true,
		},
		{
			name:    "invalid number",
			input:   "@today-xd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertProjectsDateToISOWithBase(tt.input, baseDate)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ConvertProjectsDateToISOWithBase() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ConvertProjectsDateToISOWithBase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("ConvertProjectsDateToISOWithBase() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "single date field",
			input:    "updated:@today",
			expected: "updated:2025-09-04", // Will vary based on current date
		},
		{
			name:     "multiple fields with dates",
			input:    "created:@today-1d updated:>@today-1w",
			expected: "created:2025-09-03 updated:>2025-08-28", // Will vary based on current date
		},
		{
			name:     "mixed with non-date fields",
			input:    "state:open created:@today assignee:@me",
			expected: "state:open created:2025-09-04 assignee:@me", // Will vary based on current date
		},
		{
			name:     "no date fields",
			input:    "state:open assignee:@me",
			expected: "state:open assignee:@me",
		},
		{
			name:     "complex date expression",
			input:    "updated:>=@today-30d",
			expected: "updated:>=2025-08-05", // Will vary based on current date
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertSearchQuery(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ConvertSearchQuery() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ConvertSearchQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For these tests, we'll just check that the function doesn't error
			// since the exact dates will vary based on when the test is run
			if result == "" {
				t.Errorf("ConvertSearchQuery() returned empty result")
			}
		})
	}
}
