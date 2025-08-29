package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/yahsan2/gh-pm/pkg/issue"
)

// FormatType represents the output format type
type FormatType int

const (
	// FormatTable outputs as a formatted table
	FormatTable FormatType = iota
	// FormatJSON outputs as JSON
	FormatJSON
	// FormatCSV outputs as CSV
	FormatCSV
	// FormatQuiet outputs minimal information
	FormatQuiet
)

// Formatter handles output formatting
type Formatter struct {
	format FormatType
	writer io.Writer
}

// NewFormatter creates a new formatter
func NewFormatter(format FormatType) *Formatter {
	return &Formatter{
		format: format,
		writer: os.Stdout,
	}
}

// NewFormatterWithWriter creates a new formatter with custom writer
func NewFormatterWithWriter(format FormatType, writer io.Writer) *Formatter {
	return &Formatter{
		format: format,
		writer: writer,
	}
}

// FormatIssue formats a single issue for output
func (f *Formatter) FormatIssue(issue *issue.Issue) error {
	switch f.format {
	case FormatQuiet:
		_, err := fmt.Fprintln(f.writer, issue.URL)
		return err
	case FormatJSON:
		return f.formatIssueJSON(issue)
	case FormatCSV:
		return f.formatIssueCSV(issue)
	default:
		return f.formatIssueTable(issue)
	}
}

// formatIssueTable formats an issue as a table
func (f *Formatter) formatIssueTable(issue *issue.Issue) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()
	
	fmt.Fprintf(w, "Issue created successfully!\n\n")
	fmt.Fprintf(w, "Number:\t#%d\n", issue.Number)
	fmt.Fprintf(w, "Title:\t%s\n", issue.Title)
	fmt.Fprintf(w, "URL:\t%s\n", issue.URL)
	fmt.Fprintf(w, "Repository:\t%s\n", issue.Repository)
	fmt.Fprintf(w, "State:\t%s\n", issue.State)
	
	if len(issue.Labels) > 0 {
		labels := make([]string, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i] = label.Name
		}
		fmt.Fprintf(w, "Labels:\t%s\n", strings.Join(labels, ", "))
	}
	
	if issue.ProjectItem != nil && len(issue.ProjectItem.Fields) > 0 {
		fmt.Fprintf(w, "\nProject Fields:\n")
		for key, value := range issue.ProjectItem.Fields {
			fmt.Fprintf(w, "  %s:\t%v\n", key, value)
		}
	}
	
	return nil
}

// formatIssueJSON formats an issue as JSON
func (f *Formatter) formatIssueJSON(issue *issue.Issue) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(issue)
}

// formatIssueCSV formats an issue as CSV
func (f *Formatter) formatIssueCSV(issue *issue.Issue) error {
	w := csv.NewWriter(f.writer)
	defer w.Flush()
	
	// Write header
	headers := []string{"Number", "Title", "URL", "Repository", "State", "Labels"}
	if err := w.Write(headers); err != nil {
		return err
	}
	
	// Write data
	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.Name
	}
	
	record := []string{
		fmt.Sprintf("%d", issue.Number),
		issue.Title,
		issue.URL,
		issue.Repository,
		issue.State,
		strings.Join(labels, ";"),
	}
	
	return w.Write(record)
}

// FormatBatchResult formats batch processing results
func (f *Formatter) FormatBatchResult(result *issue.BatchResult) error {
	switch f.format {
	case FormatJSON:
		return f.formatBatchResultJSON(result)
	case FormatCSV:
		return f.formatBatchResultCSV(result)
	default:
		return f.formatBatchResultTable(result)
	}
}

// formatBatchResultTable formats batch results as a table
func (f *Formatter) formatBatchResultTable(result *issue.BatchResult) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()
	
	fmt.Fprintf(w, "Batch Processing Complete\n\n")
	fmt.Fprintf(w, "Total:\t%d\n", result.Total)
	fmt.Fprintf(w, "Succeeded:\t%d\n", result.Succeeded)
	fmt.Fprintf(w, "Failed:\t%d\n", result.Failed)
	
	if len(result.Issues) > 0 {
		fmt.Fprintf(w, "\nCreated Issues:\n")
		fmt.Fprintf(w, "Number\tTitle\tURL\n")
		for _, issue := range result.Issues {
			fmt.Fprintf(w, "#%d\t%s\t%s\n", issue.Number, issue.Title, issue.URL)
		}
	}
	
	if len(result.Errors) > 0 {
		fmt.Fprintf(w, "\nErrors:\n")
		for _, err := range result.Errors {
			fmt.Fprintf(w, "  [%d] %s: %s\n", err.Index, err.Title, err.Error)
		}
	}
	
	return nil
}

// formatBatchResultJSON formats batch results as JSON
func (f *Formatter) formatBatchResultJSON(result *issue.BatchResult) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// formatBatchResultCSV formats batch results as CSV
func (f *Formatter) formatBatchResultCSV(result *issue.BatchResult) error {
	w := csv.NewWriter(f.writer)
	defer w.Flush()
	
	// Write summary
	if err := w.Write([]string{"Type", "Count"}); err != nil {
		return err
	}
	if err := w.Write([]string{"Total", fmt.Sprintf("%d", result.Total)}); err != nil {
		return err
	}
	if err := w.Write([]string{"Succeeded", fmt.Sprintf("%d", result.Succeeded)}); err != nil {
		return err
	}
	if err := w.Write([]string{"Failed", fmt.Sprintf("%d", result.Failed)}); err != nil {
		return err
	}
	
	// Empty line
	if err := w.Write([]string{}); err != nil {
		return err
	}
	
	// Write issues if any
	if len(result.Issues) > 0 {
		if err := w.Write([]string{"Number", "Title", "URL"}); err != nil {
			return err
		}
		for _, issue := range result.Issues {
			record := []string{
				fmt.Sprintf("%d", issue.Number),
				issue.Title,
				issue.URL,
			}
			if err := w.Write(record); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// FormatError formats an error for output
func (f *Formatter) FormatError(err error) error {
	if f.format == FormatJSON {
		errorData := map[string]string{
			"error": err.Error(),
		}
		
		// If it's an IssueError, include more details
		if issueErr, ok := err.(*issue.IssueError); ok {
			errorData["type"] = fmt.Sprintf("%d", issueErr.Type)
			if issueErr.Suggestion != "" {
				errorData["suggestion"] = issueErr.Suggestion
			}
		}
		
		encoder := json.NewEncoder(f.writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(errorData)
	}
	
	// For table and quiet formats, just print the error
	_, printErr := fmt.Fprintln(f.writer, err.Error())
	return printErr
}