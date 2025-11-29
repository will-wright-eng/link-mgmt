package links

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"link-mgmt-go/pkg/models"
)

// FormatTableOutput formats links as a polished table for CLI output
func FormatTableOutput(links []models.Link) string {
	if len(links) == 0 {
		return "No links found."
	}

	var b strings.Builder

	// Header
	b.WriteString("\n")
	b.WriteString(renderHeader())
	b.WriteString("\n")

	// Table
	w := tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tURL\tTitle\tCreated")
	fmt.Fprintln(w, strings.Repeat("─", 8)+"\t"+strings.Repeat("─", 50)+"\t"+strings.Repeat("─", 40)+"\t"+strings.Repeat("─", 16))

	for _, link := range links {
		title := GetTitle(link)
		url := TruncateURL(link.URL, 50)
		idShort := ShortenID(link.ID)
		created := FormatDate(link.CreatedAt)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			idShort,
			url,
			title,
			created,
		)
	}

	w.Flush()
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Total: %d link(s)\n", len(links)))

	return b.String()
}

// FormatSuccessMessage formats a success message for link creation
func FormatSuccessMessage(link *models.Link) string {
	var b strings.Builder

	title := GetTitle(*link)
	idShort := ShortenID(link.ID)
	created := FormatDate(link.CreatedAt)

	b.WriteString("\n")
	b.WriteString("✓ Link created successfully!\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  ID:      %s\n", idShort))
	b.WriteString(fmt.Sprintf("  URL:     %s\n", link.URL))
	b.WriteString(fmt.Sprintf("  Title:   %s\n", title))
	b.WriteString(fmt.Sprintf("  Created: %s\n", created))
	b.WriteString("\n")

	return b.String()
}

// FormatErrorMessage formats an error message consistently
func FormatErrorMessage(err error) string {
	return fmt.Sprintf("❌ Error: %v\n", err)
}

// renderHeader renders a styled header
func renderHeader() string {
	return "Your Links"
}

// FormatEmptyState formats an empty state message
func FormatEmptyState(message string) string {
	return fmt.Sprintf("\n%s\n", message)
}

// WriteToStdout writes formatted output to stdout with proper handling
func WriteToStdout(content string) {
	fmt.Fprint(os.Stdout, content)
}

// WriteToStderr writes formatted output to stderr
func WriteToStderr(content string) {
	fmt.Fprint(os.Stderr, content)
}
