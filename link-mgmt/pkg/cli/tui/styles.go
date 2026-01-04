package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Define a consistent color palette
var (
	// Colors
	colorPrimary   = lipgloss.Color("62")  // Purple/blue
	colorSecondary = lipgloss.Color("244") // Gray
	colorSuccess   = lipgloss.Color("42")  // Green
	colorError     = lipgloss.Color("196") // Red
	colorWarning   = lipgloss.Color("214") // Orange/Yellow
	colorInfo      = lipgloss.Color("39")  // Cyan
	colorMuted     = lipgloss.Color("240") // Dark gray
	colorBorder    = lipgloss.Color("238") // Border gray
)

// Reusable style definitions
var (
	// Title/Header styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	// Text styles
	boldStyle = lipgloss.NewStyle().Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Status styles
	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorInfo)

	// Link/item styles
	linkIDStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	linkTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

	linkURLStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	// Field label styles
	fieldLabelStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginRight(2)

	// List/item styles
	selectedStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	selectedMarkerStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(colorBorder)

	// Help text
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
)

// Helper functions for common formatting patterns
func renderTitle(title string) string {
	return "\n" + titleStyle.Render(title) + "\n"
}

func renderSuccess(msg string) string {
	return successStyle.Render("✓ " + msg)
}

func renderError(msg string) string {
	return errorStyle.Render("❌ " + msg)
}

func renderDivider(length int) string {
	return dividerStyle.Render(strings.Repeat("─", length))
}
