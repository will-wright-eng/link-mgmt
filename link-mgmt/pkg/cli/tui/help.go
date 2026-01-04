package tui

import (
	"fmt"
	"strings"
)

// HelpItem represents a single keyboard shortcut and its description
type HelpItem struct {
	Key         string
	Description string
}

// CommonHelpContent returns help for common commands
func CommonHelpContent() string {
	items := []HelpItem{
		{"?", "Toggle help"},
		{"m", "Return to main menu"},
		{"q / Esc", "Quit application"},
		{"Ctrl+C", "Force quit"},
	}
	return renderHelpItems(items)
}

// RootMenuHelpContent returns help for root menu
func RootMenuHelpContent() string {
	items := []HelpItem{
		{"1-2", "Select menu option (Add link / Manage links)"},
		{"q / Esc", "Quit"},
		{"?", "Show this help"},
	}
	return renderHelpItems(items)
}

// ManageLinksHelpContent returns help for manage links flow
func ManageLinksHelpContent() string {
	items := []HelpItem{
		{"↑ / ↓ / j / k", "Navigate link list"},
		{"Enter", "Select link"},
		{"Esc / b", "Go back"},
		{"1 / v", "View details"},
		{"2 / d", "Delete link"},
		{"3 / s", "Scrape & enrich"},
		{"m", "Return to menu"},
		{"q", "Quit"},
		{"?", "Show this help"},
	}
	return renderHelpItems(items)
}

// AddLinkFormHelpContent returns help for add link form
func AddLinkFormHelpContent() string {
	items := []HelpItem{
		{"Enter", "Start scraping (URL input) / Save link (review)"},
		{"s", "Skip scraping, go to review"},
		{"Tab / Shift+Tab", "Navigate fields (review step)"},
		{"Esc", "Cancel / Quit"},
		{"m", "Return to menu"},
		{"?", "Show this help"},
	}
	return renderHelpItems(items)
}

// renderHelpItems formats help items into a readable string
func renderHelpItems(items []HelpItem) string {
	var b strings.Builder
	for _, item := range items {
		keyStyle := boldStyle.Foreground(colorPrimary)
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			keyStyle.Render(item.Key),
			item.Description))
	}
	return b.String()
}
