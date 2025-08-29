package project

import (
	"testing"
	
	"github.com/yahsan2/gh-pm/pkg/config"
)

func TestURLBuilder_GetProjectURL(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		want   string
	}{
		{
			name: "user project",
			config: &config.Config{
				Project: config.ProjectConfig{
					Owner:  "yahsan2",
					Number: 8,
				},
			},
			want: "https://github.com/users/yahsan2/projects/8",
		},
		{
			name: "organization project",
			config: &config.Config{
				Project: config.ProjectConfig{
					Org:    "myorg",
					Owner:  "yahsan2", // should be ignored when Org is set
					Number: 5,
				},
			},
			want: "https://github.com/orgs/myorg/projects/5",
		},
		{
			name: "no project number",
			config: &config.Config{
				Project: config.ProjectConfig{
					Owner: "yahsan2",
				},
			},
			want: "",
		},
		{
			name: "no owner",
			config: &config.Config{
				Project: config.ProjectConfig{
					Number: 8,
				},
			},
			want: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewURLBuilder(tt.config, nil)
			got := b.GetProjectURL()
			if got != tt.want {
				t.Errorf("GetProjectURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURLBuilder_GetProjectItemURL(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		itemDatabaseID int
		want           string
	}{
		{
			name: "user project with item",
			config: &config.Config{
				Project: config.ProjectConfig{
					Owner:  "yahsan2",
					Number: 8,
				},
			},
			itemDatabaseID: 126562770,
			want:           "https://github.com/users/yahsan2/projects/8?pane=issue&itemId=126562770",
		},
		{
			name: "organization project with item",
			config: &config.Config{
				Project: config.ProjectConfig{
					Org:    "myorg",
					Number: 5,
				},
			},
			itemDatabaseID: 123456,
			want:           "https://github.com/orgs/myorg/projects/5?pane=issue&itemId=123456",
		},
		{
			name: "no project configured",
			config: &config.Config{
				Project: config.ProjectConfig{},
			},
			itemDatabaseID: 123456,
			want:           "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewURLBuilder(tt.config, nil)
			got := b.GetProjectItemURL(tt.itemDatabaseID)
			if got != tt.want {
				t.Errorf("GetProjectItemURL() = %v, want %v", got, tt.want)
			}
		})
	}
}