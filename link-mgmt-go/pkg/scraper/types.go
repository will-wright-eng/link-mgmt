package scraper

// ScrapeRequest represents a request to scrape a URL
type ScrapeRequest struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout,omitempty"`
}

// ScrapeResponse represents the response from a scrape operation
type ScrapeResponse struct {
	Success     bool   `json:"success"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Text        string `json:"text"`
	ExtractedAt string `json:"extracted_at"`
	Error       string `json:"error,omitempty"`
}

// ScrapeStage represents the current stage of a scraping operation
type ScrapeStage string

const (
	StageHealthCheck ScrapeStage = "health_check"
	StageFetching    ScrapeStage = "fetching"
	StageExtracting  ScrapeStage = "extracting"
	StageComplete    ScrapeStage = "complete"
)

// ProgressCallback is called to report progress during scraping operations
// stage: The current stage of the operation
// message: A human-readable message describing the current progress
type ProgressCallback func(stage ScrapeStage, message string)
