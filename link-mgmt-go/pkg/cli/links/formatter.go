package links

import (
	"time"

	"link-mgmt-go/pkg/models"

	"github.com/google/uuid"
)

// GetTitle returns the title of a link, or a default value if missing
func GetTitle(link models.Link) string {
	if link.Title != nil && *link.Title != "" {
		return *link.Title
	}
	return "(no title)"
}

// TruncateURL truncates a URL to the specified max length
func TruncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

// ShortenID returns a shortened version of a UUID (first 8 characters + "...")
func ShortenID(id uuid.UUID) string {
	return id.String()[:8] + "..."
}

// FormatDate formats a time as a readable date string
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}
