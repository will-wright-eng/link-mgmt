package tui

import (
	"errors"
	"fmt"
	"strings"

	"link-mgmt-go/pkg/models"
	"link-mgmt-go/pkg/scraper"

	"github.com/charmbracelet/lipgloss"
)

// renderErrorView renders a standard error view with exit message
func renderErrorView(err error) string {
	return "\n" + renderError(fmt.Sprintf("Error: %v", err)) + "\n\n" +
		helpStyle.Render("Press any key to exit...") + "\n"
}

// renderEmptyState renders a standard empty state message
func renderEmptyState(message string) string {
	return "\n" + mutedStyle.Render(message) + "\n\n" +
		helpStyle.Render("Press any key to exit...") + "\n"
}

// renderLoadingState renders a standard loading message
func renderLoadingState(message string) string {
	return "\n" + infoStyle.Render(message) + "\n"
}

// renderSuccessView renders a standard success view with exit message
func renderSuccessView(message string) string {
	return "\n" + renderSuccess(message) + "\n\n" +
		helpStyle.Render("Press any key to exit...") + "\n"
}

// LinkListItem represents a link item in a selectable list
type LinkListItem struct {
	Link     models.Link
	Selected bool
	Index    int
}

// renderLinkList renders a selectable list of links with navigation markers
func renderLinkList(links []models.Link, selected int, title string, subtitle string) string {
	if len(links) == 0 {
		return renderEmptyState("No links found.")
	}

	var b strings.Builder
	b.WriteString(renderTitle(title))
	if subtitle != "" {
		b.WriteString(boldStyle.Render(subtitle) + "\n\n")
	}

	for i, link := range links {
		marker := " "
		if i == selected {
			marker = selectedMarkerStyle.Render("→")
		}

		title := formatLinkTitle(link)
		url := truncateURL(link.URL, 60)

		var titleStyle lipgloss.Style
		if i == selected {
			titleStyle = selectedStyle
		} else {
			titleStyle = linkTitleStyle
		}

		b.WriteString(fmt.Sprintf("%s %s\n", marker, titleStyle.Render(title)))
		b.WriteString(fmt.Sprintf("  %s\n", linkURLStyle.Render(url)))
	}

	b.WriteString("\n")
	return b.String()
}

// formatLinkTitle returns the title of a link or a default value
func formatLinkTitle(link models.Link) string {
	if link.Title != nil && *link.Title != "" {
		return *link.Title
	}
	return "(no title)"
}

// truncateURL truncates a URL to the specified max length
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}

// renderLinkDetails renders common link details (ID, URL, Title, Created date)
func renderLinkDetails(link *models.Link, includeUserID bool) string {
	if link == nil {
		return ""
	}

	var b strings.Builder

	title := ""
	if link.Title != nil && *link.Title != "" {
		title = *link.Title
	} else {
		title = "(no title)"
	}

	b.WriteString(fieldLabelStyle.Render("ID:"))
	b.WriteString(fmt.Sprintf(" %s\n", linkIDStyle.Render(link.ID.String()[:8]+"...")))

	if includeUserID {
		b.WriteString(fieldLabelStyle.Render("User ID:"))
		b.WriteString(fmt.Sprintf(" %s\n", linkIDStyle.Render(link.UserID.String())))
	}

	b.WriteString(fieldLabelStyle.Render("URL:"))
	b.WriteString(fmt.Sprintf(" %s\n", link.URL))

	b.WriteString(fieldLabelStyle.Render("Title:"))
	b.WriteString(fmt.Sprintf(" %s\n", title))

	b.WriteString(fieldLabelStyle.Render("Created:"))
	b.WriteString(fmt.Sprintf(" %s\n", link.CreatedAt.Format("2006-01-02 15:04")))

	return b.String()
}

// renderLinkDetailsFull renders all link details including description and text
func renderLinkDetailsFull(link *models.Link) string {
	if link == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(renderLinkDetails(link, true))

	// Description
	b.WriteString(fieldLabelStyle.Render("Description:"))
	if link.Description != nil && *link.Description != "" {
		desc := *link.Description
		b.WriteString(wrapText(desc, 80, " "))
	} else {
		b.WriteString(" " + mutedStyle.Render("(not set)") + "\n")
	}

	// Text
	b.WriteString(fieldLabelStyle.Render("Text:"))
	if link.Text != nil && *link.Text != "" {
		text := *link.Text
		if len(text) > 500 {
			preview := text[:500]
			if lastSpace := strings.LastIndex(preview, " "); lastSpace > 400 {
				preview = preview[:lastSpace]
			}
			b.WriteString(fmt.Sprintf(" %s...\n", preview))
			b.WriteString(fmt.Sprintf("  %s\n", mutedStyle.Render(fmt.Sprintf("(truncated, full length: %d characters)", len(text)))))
		} else {
			b.WriteString(wrapText(text, 80, " "))
		}
	} else {
		b.WriteString(" " + mutedStyle.Render("(not set)") + "\n")
	}

	// Updated At
	b.WriteString(fieldLabelStyle.Render("Updated At:"))
	b.WriteString(fmt.Sprintf(" %s\n", link.UpdatedAt.Format("2006-01-02 15:04:05")))

	return b.String()
}

// wrapText wraps text to a specified width, breaking at word boundaries
func wrapText(text string, width int, indent string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return indent + "\n"
	}

	var b strings.Builder
	line := ""
	for _, word := range words {
		if len(line)+len(word)+1 > width {
			b.WriteString(fmt.Sprintf("%s%s\n", indent, line))
			line = word
		} else {
			if line != "" {
				line += " "
			}
			line += word
		}
	}
	if line != "" {
		b.WriteString(fmt.Sprintf("%s%s\n", indent, line))
	}
	return b.String()
}

// handleListNavigation handles common navigation keys for list views (up/down/j/k)
// Returns the new selected index and whether navigation occurred
func handleListNavigation(key string, selected int, total int) (newSelected int, handled bool) {
	switch key {
	case "up", "k":
		if selected > 0 {
			return selected - 1, true
		}
		return selected, true
	case "down", "j":
		if selected < total-1 {
			return selected + 1, true
		}
		return selected, true
	}
	return selected, false
}

// handleQuitKeys checks if a key should quit the current view
func handleQuitKeys(key string) bool {
	switch key {
	case "ctrl+c", "q", "esc":
		return true
	}
	return false
}

// renderWarningView renders a standard warning view with exit message
func renderWarningView(message string) string {
	return "\n" + renderWarning(message) + "\n\n" +
		helpStyle.Render("Press any key to exit...") + "\n"
}

// renderSuccessWithDetails renders a success message with link details
func renderSuccessWithDetails(message string, link *models.Link, includeUserID bool) string {
	if link == nil {
		return renderSuccessView(message)
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderSuccess(message))
	b.WriteString("\n\n")
	b.WriteString(renderLinkDetails(link, includeUserID))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press any key to exit...") + "\n")
	return b.String()
}

// renderScrapingProgress renders a scraping progress view
func renderScrapingProgress(title string, stage string, message string) string {
	var b strings.Builder
	b.WriteString(renderTitle(title))

	stageLabel := stage
	if stageLabel == "" {
		stageLabel = "starting"
	}
	b.WriteString(fieldLabelStyle.Render("Stage:"))
	b.WriteString(fmt.Sprintf(" %s\n", stageLabel))

	if message != "" {
		b.WriteString(infoStyle.Render(message))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("This may take a few seconds."))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press Esc to cancel."))

	return b.String()
}

// renderInlineError renders an error message inline (without full error view formatting)
func renderInlineError(err error) string {
	if err == nil {
		return ""
	}
	return renderError(err.Error())
}

// renderInlineWarning renders a warning message inline (without full warning view formatting)
func renderInlineWarning(message string) string {
	return renderWarning(message)
}

// userFacingError converts structured scraper errors into friendly messages,
// while leaving other error types unchanged.
func userFacingError(err error) error {
	if err == nil {
		return nil
	}

	var scraperErr *scraper.ScraperError
	if errors.As(err, &scraperErr) {
		return errors.New(scraperErr.UserMessage())
	}

	return err
}
