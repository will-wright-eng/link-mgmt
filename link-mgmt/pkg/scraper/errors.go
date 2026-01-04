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
	ErrorTypeBrowserError       ErrorType = "browser_error"
	ErrorTypeRateLimit          ErrorType = "rate_limit"
	ErrorTypeBlocked            ErrorType = "blocked"
	ErrorTypeUnknown            ErrorType = "unknown"
)

// ScraperError represents a structured error from the scraper service
type ScraperError struct {
	Type         ErrorType
	Message      string
	Cause        error
	retryable    bool
	retryableSet bool // tracks if retryable was explicitly set
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
// If the retryable field is explicitly set, it takes precedence
func (e *ScraperError) IsRetryable() bool {
	// If retryable is explicitly set, use that value
	if e.retryableSet {
		return e.retryable
	}
	// Otherwise, use default behavior based on error type
	switch e.Type {
	case ErrorTypeServiceUnavailable, ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeBrowserError, ErrorTypeRateLimit:
		return true
	case ErrorTypeExtraction, ErrorTypeInvalidURL, ErrorTypeInvalidResponse, ErrorTypeCancelled, ErrorTypeBlocked, ErrorTypeUnknown:
		return false
	default:
		return false
	}
}

// SetRetryable explicitly sets the retryable flag
func (e *ScraperError) SetRetryable(retryable bool) {
	e.retryable = retryable
	e.retryableSet = true
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
	case ErrorTypeBrowserError:
		return "Browser error occurred. The scraper service may need to recover."
	case ErrorTypeRateLimit:
		return "Rate limit exceeded. Please wait before trying again."
	case ErrorTypeBlocked:
		return "Access blocked by network security. The request may be blocked by security policies."
	case ErrorTypeUnknown:
		return fmt.Sprintf("Unknown error occurred: %s", e.Message)
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

func newBrowserError(message string, cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeBrowserError,
		Message: message,
		Cause:   cause,
	}
}

func newRateLimitError(message string, cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeRateLimit,
		Message: message,
		Cause:   cause,
	}
}

func newBlockedError(message string, cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeBlocked,
		Message: message,
		Cause:   cause,
	}
}

func newUnknownError(message string, cause error) *ScraperError {
	return &ScraperError{
		Type:    ErrorTypeUnknown,
		Message: message,
		Cause:   cause,
	}
}

// MapErrorTypeFromString maps a string error type from the scraper service to ErrorType
func MapErrorTypeFromString(errorType string) ErrorType {
	switch errorType {
	case "timeout":
		return ErrorTypeTimeout
	case "network":
		return ErrorTypeNetwork
	case "extraction":
		return ErrorTypeExtraction
	case "invalid_url":
		return ErrorTypeInvalidURL
	case "browser_error":
		return ErrorTypeBrowserError
	case "rate_limit":
		return ErrorTypeRateLimit
	case "blocked":
		return ErrorTypeBlocked
	case "unknown":
		return ErrorTypeUnknown
	default:
		return ErrorTypeUnknown
	}
}

// NewScraperErrorFromType creates a ScraperError from an ErrorType and message
func NewScraperErrorFromType(errorType ErrorType, message string, cause error) *ScraperError {
	switch errorType {
	case ErrorTypeTimeout:
		return newTimeoutError(cause)
	case ErrorTypeNetwork:
		return newNetworkError(cause)
	case ErrorTypeExtraction:
		return newExtractionError(message)
	case ErrorTypeInvalidURL:
		return &ScraperError{
			Type:    ErrorTypeInvalidURL,
			Message: message,
			Cause:   cause,
		}
	case ErrorTypeBrowserError:
		return newBrowserError(message, cause)
	case ErrorTypeRateLimit:
		return newRateLimitError(message, cause)
	case ErrorTypeBlocked:
		return newBlockedError(message, cause)
	case ErrorTypeServiceUnavailable:
		return newServiceUnavailableError(cause)
	case ErrorTypeInvalidResponse:
		return newInvalidResponseError(message, cause)
	case ErrorTypeCancelled:
		return newCancelledError(cause)
	default:
		return newUnknownError(message, cause)
	}
}
