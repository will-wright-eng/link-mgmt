package scraper

import "fmt"

// ErrorType categorizes different types of scraper errors
type ErrorType string

const (
	ErrorTypeServiceUnavailable ErrorType = "service_unavailable"
	ErrorTypeTimeout            ErrorType = "timeout"
	ErrorTypeNetwork            ErrorType = "network"
	ErrorTypeExtraction         ErrorType = "extraction"
	ErrorTypeInvalidURL         ErrorType = "invalid_url"
	ErrorTypeInvalidResponse    ErrorType = "invalid_response"
	ErrorTypeCancelled          ErrorType = "cancelled"
)

// ScraperError represents a structured error from the scraper service
type ScraperError struct {
	Type    ErrorType
	Message string
	Cause   error
}

// Error implements the error interface
func (e *ScraperError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error for error unwrapping
func (e *ScraperError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true if the error is likely to succeed on retry
func (e *ScraperError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeServiceUnavailable, ErrorTypeNetwork, ErrorTypeTimeout:
		return true
	case ErrorTypeExtraction, ErrorTypeInvalidURL, ErrorTypeInvalidResponse, ErrorTypeCancelled:
		return false
	default:
		return false
	}
}

// UserMessage returns a user-friendly error message
func (e *ScraperError) UserMessage() string {
	switch e.Type {
	case ErrorTypeServiceUnavailable:
		return "Scraper service unavailable. Please check if the service is running."
	case ErrorTypeTimeout:
		return "Scraping timed out. The URL may be slow to load or the service may be busy."
	case ErrorTypeNetwork:
		return "Network error occurred while scraping. Please check your connection and try again."
	case ErrorTypeExtraction:
		return fmt.Sprintf("Failed to extract content from URL: %s", e.Message)
	case ErrorTypeInvalidURL:
		return fmt.Sprintf("Invalid URL: %s", e.Message)
	case ErrorTypeInvalidResponse:
		return "Received invalid response from scraper service. Please try again."
	case ErrorTypeCancelled:
		return "Scraping was cancelled."
	default:
		return e.Message
	}
}

// Helper functions to create specific error types
func newServiceUnavailableError(cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeServiceUnavailable,
		Message: "Service not available",
		Cause:   cause,
	}
}

func newTimeoutError(cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeTimeout,
		Message: "Request timed out",
		Cause:   cause,
	}
}

func newNetworkError(cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeNetwork,
		Message: "Network error",
		Cause:   cause,
	}
}

func newExtractionError(message string) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeExtraction,
		Message: message,
	}
}

func newInvalidResponseError(message string, cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeInvalidResponse,
		Message: message,
		Cause:   cause,
	}
}

func newCancelledError(cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeCancelled,
		Message: "Operation cancelled",
		Cause:   cause,
	}
}
