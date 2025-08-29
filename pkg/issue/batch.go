package issue

// BatchResult represents the result of batch issue creation
type BatchResult struct {
	Total      int           `json:"total"`
	Succeeded  int           `json:"succeeded"`
	Failed     int           `json:"failed"`
	Issues     []*Issue      `json:"issues"`
	Errors     []BatchError  `json:"errors,omitempty"`
}

// BatchError represents an error during batch processing
type BatchError struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Error   string `json:"error"`
}