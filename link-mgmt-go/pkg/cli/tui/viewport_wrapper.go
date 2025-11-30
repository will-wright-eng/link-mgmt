package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"link-mgmt-go/pkg/cli/logger"
)

// ViewportWrapper wraps a model with viewport and common command support
type ViewportWrapper struct {
	model    tea.Model
	viewport viewport.Model
	width    int
	height   int
	config   ViewportConfig

	// Common commands
	showHelp    bool
	helpContent string
}

// ViewportConfig configures the wrapper behavior
type ViewportConfig struct {
	Title        string
	ShowHeader   bool
	ShowFooter   bool
	HeaderHeight int            // Fixed header height (0 = auto)
	FooterHeight int            // Fixed footer height (0 = auto)
	UseViewport  bool           // Enable scrolling (false = simple responsive)
	MinWidth     int            // Minimum terminal width
	MinHeight    int            // Minimum terminal height
	EnableHelp   bool           // Enable '?' for help (proposed)
	EnableMenu   bool           // Enable 'm' to return to menu (proposed)
	HelpContent  func() string  // Function to generate help text
	OnMenu       func() tea.Cmd // Callback for menu command
}

// NewViewportWrapper creates a new wrapper around a model
func NewViewportWrapper(model tea.Model, config ViewportConfig) *ViewportWrapper {
	vp := viewport.New(0, 0)
	// Viewport uses default key bindings (arrow keys, page up/down)

	return &ViewportWrapper{
		model:    model,
		viewport: vp,
		config:   config,
		width:    80, // Default
		height:   24, // Default
	}
}

func (w *ViewportWrapper) Init() tea.Cmd {
	// Initialize wrapped model
	var cmd tea.Cmd
	if w.model != nil {
		cmd = w.model.Init()
	}
	return cmd
}

func (w *ViewportWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	logger.Log("ViewportWrapper.Update() called: msg_type=%T", msg)

	// Handle window size first
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		logger.Log("ViewportWrapper.Update: WindowSizeMsg, width=%d, height=%d", msg.Width, msg.Height)
		w.width = msg.Width
		w.height = msg.Height

		// Validate minimum size
		if w.config.MinWidth > 0 && w.width < w.config.MinWidth {
			w.width = w.config.MinWidth
		}
		if w.config.MinHeight > 0 && w.height < w.config.MinHeight {
			w.height = w.config.MinHeight
		}

		// Calculate layout
		w.calculateLayout()
		logger.Log("ViewportWrapper.Update: calculated layout, viewport=%dx%d", w.viewport.Width, w.viewport.Height)

		// Sync viewport
		if w.config.UseViewport {
			var vpCmd tea.Cmd
			w.viewport, vpCmd = w.viewport.Update(msg)
			// Forward window size to wrapped model (it may need size info)
			var cmd tea.Cmd
			if w.model != nil {
				w.model, cmd = w.model.Update(msg)
			}
			return w, tea.Batch(vpCmd, cmd)
		}

		// Forward to wrapped model
		var cmd tea.Cmd
		if w.model != nil {
			w.model, cmd = w.model.Update(msg)
		}
		return w, cmd
	}

	// Handle common commands
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		logger.Log("ViewportWrapper.Update: KeyMsg, key=%q, showHelp=%v", key, w.showHelp)
		switch key {
		case "?":
			if w.config.EnableHelp {
				w.showHelp = !w.showHelp
				logger.Log("ViewportWrapper.Update: toggled help, showHelp=%v", w.showHelp)
				if w.showHelp && w.config.HelpContent != nil {
					w.helpContent = w.config.HelpContent()
				}
				return w, nil
			}
		case "m":
			if w.config.EnableMenu {
				logger.Log("ViewportWrapper.Update: menu key pressed")
				if w.config.OnMenu != nil {
					// Use the callback if provided
					return w, w.config.OnMenu()
				}
				// Otherwise, send MenuNavigationMsg directly
				return w, func() tea.Msg {
					return MenuNavigationMsg{}
				}
			}
		case "ctrl+c", "q", "esc":
			// Only quit if help is not showing
			if !w.showHelp {
				logger.Log("ViewportWrapper.Update: quit key pressed")
				return w, tea.Quit
			}
			// If help is showing, close it
			logger.Log("ViewportWrapper.Update: closing help overlay")
			w.showHelp = false
			return w, nil
		}
	}

	// Forward MenuNavigationMsg unchanged (let it bubble up to root)
	// We need to forward it to the wrapped model, and it will bubble up from there
	// But we also need to return a command so it reaches the root
	// Actually, MenuNavigationMsg should just pass through as-is to wrapped model
	// and then bubble up. The root will catch it.

	// If help is showing, only handle help-related keys
	if w.showHelp {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "?", "esc", "q":
				w.showHelp = false
				return w, nil
			}
		}
		return w, nil
	}

	// Forward all other messages to wrapped model
	var cmd tea.Cmd
	if w.model != nil {
		logger.Log("ViewportWrapper.Update: forwarding msg type=%T to wrapped model", msg)
		w.model, cmd = w.model.Update(msg)
		logger.Log("ViewportWrapper.Update: wrapped model updated, cmd=%v", cmd != nil)
	} else {
		logger.Log("ViewportWrapper.Update: WARNING - wrapped model is nil")
	}

	// If using viewport, also pass messages for scrolling
	// Note: This may intercept navigation keys. The viewport should only scroll when content exceeds height.
	// If there are conflicts, we may need to conditionally enable viewport or use different keys.
	if w.config.UseViewport {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			logger.Log("ViewportWrapper.Update: passing key %q to viewport", keyMsg.String())
		}
		var vpCmd tea.Cmd
		w.viewport, vpCmd = w.viewport.Update(msg)
		if vpCmd != nil {
			logger.Log("ViewportWrapper.Update: viewport returned a command")
		}
		cmd = tea.Batch(cmd, vpCmd)
	}

	return w, cmd
}

func (w *ViewportWrapper) View() string {
	logger.Log("ViewportWrapper.View() called: showHelp=%v, UseViewport=%v, width=%d, height=%d, model=%v",
		w.showHelp, w.config.UseViewport, w.width, w.height, w.model != nil)

	// If help is showing, render help overlay
	if w.showHelp {
		logger.Log("ViewportWrapper.View: rendering help overlay")
		return w.renderHelpOverlay()
	}

	// Get content from wrapped model
	content := ""
	if w.model != nil {
		logger.Log("ViewportWrapper.View: getting content from wrapped model")
		content = w.model.View()
		logger.Log("ViewportWrapper.View: wrapped model returned content, length=%d bytes", len(content))

		// Check if wrapped model is delegating to another wrapped model (like rootModel -> manageLinksModel)
		// If the wrapped model is a rootModel with an active flow, pass through without adding headers/footers
		if w.isDelegatingToWrappedModel() {
			logger.Log("ViewportWrapper.View: wrapped model is delegating, passing through without headers/footers")
			return content
		}
	} else {
		logger.Log("ViewportWrapper.View: WARNING - wrapped model is nil")
	}

	// Apply viewport if enabled
	if w.config.UseViewport {
		// Ensure layout is calculated before using viewport
		// Always recalculate to ensure dimensions are up-to-date
		w.calculateLayout()
		logger.Log("ViewportWrapper.View: applying viewport, viewport size=%dx%d, wrapper size=%dx%d, content length=%d",
			w.viewport.Width, w.viewport.Height, w.width, w.height, len(content))

		// If viewport still has invalid dimensions, use wrapper dimensions directly
		if w.viewport.Width == 0 || w.viewport.Height == 0 {
			logger.Log("ViewportWrapper.View: viewport has invalid dimensions, fixing...")
			if w.viewport.Width == 0 {
				w.viewport.Width = w.width
				if w.viewport.Width == 0 {
					w.viewport.Width = 80 // Fallback to default
				}
			}
			if w.viewport.Height == 0 {
				headerH := 2
				footerH := 1
				w.viewport.Height = w.height - headerH - footerH
				if w.viewport.Height < 1 {
					w.viewport.Height = 1
				}
			}
			logger.Log("ViewportWrapper.View: fixed viewport dimensions to %dx%d", w.viewport.Width, w.viewport.Height)
		}

		// Set content and ensure viewport is synced
		w.viewport.SetContent(content)
		// Sync the viewport to ensure it's properly initialized with the content
		// Create a WindowSizeMsg to sync the viewport
		syncMsg := tea.WindowSizeMsg{
			Width:  w.viewport.Width,
			Height: w.viewport.Height,
		}
		w.viewport, _ = w.viewport.Update(syncMsg)
		content = w.viewport.View()
		logger.Log("ViewportWrapper.View: viewport returned content, length=%d bytes", len(content))
	}

	// Build layout
	var parts []string

	// Header
	if w.config.ShowHeader {
		header := w.renderHeader()
		parts = append(parts, header)
		logger.Log("ViewportWrapper.View: added header, length=%d bytes", len(header))
	}

	// Content
	parts = append(parts, content)
	logger.Log("ViewportWrapper.View: added content, length=%d bytes", len(content))

	// Footer
	if w.config.ShowFooter {
		footer := w.renderFooter()
		parts = append(parts, footer)
		logger.Log("ViewportWrapper.View: added footer, length=%d bytes", len(footer))
	}

	// Join with proper spacing
	result := lipgloss.JoinVertical(lipgloss.Left, parts...)
	logger.Log("ViewportWrapper.View: final result length=%d bytes", len(result))
	return result
}

func (w *ViewportWrapper) calculateLayout() {
	headerH := w.config.HeaderHeight
	if headerH == 0 && w.config.ShowHeader {
		headerH = 2 // Default header height
	}

	footerH := w.config.FooterHeight
	if footerH == 0 && w.config.ShowFooter {
		footerH = 1 // Default footer height
	}

	contentH := w.height - headerH - footerH
	if contentH < 1 {
		contentH = 1
	}

	if w.config.UseViewport {
		// Ensure we have valid dimensions
		if w.width <= 0 {
			w.width = 80 // Default width
		}
		if w.height <= 0 {
			w.height = 24 // Default height
			contentH = w.height - headerH - footerH
			if contentH < 1 {
				contentH = 1
			}
		}
		w.viewport.Width = w.width
		w.viewport.Height = contentH
		logger.Log("ViewportWrapper.calculateLayout: set viewport to %dx%d (wrapper: %dx%d, headerH: %d, footerH: %d)",
			w.viewport.Width, w.viewport.Height, w.width, w.height, headerH, footerH)
	}
}

func (w *ViewportWrapper) renderHeader() string {
	var b strings.Builder

	if w.config.Title != "" {
		b.WriteString(renderTitle(w.config.Title))
	}

	// Breadcrumb or navigation hint
	if w.config.EnableMenu && w.config.EnableHelp {
		b.WriteString(helpStyle.Render("Press 'm' for menu, '?' for help") + "\n")
	} else if w.config.EnableHelp {
		b.WriteString(helpStyle.Render("Press '?' for help") + "\n")
	} else if w.config.EnableMenu {
		b.WriteString(helpStyle.Render("Press 'm' for menu") + "\n")
	}

	return b.String()
}

func (w *ViewportWrapper) renderFooter() string {
	// Footer shows current status or common shortcuts
	shortcuts := []string{}

	if w.config.EnableHelp {
		shortcuts = append(shortcuts, "? help")
	}
	if w.config.EnableMenu {
		shortcuts = append(shortcuts, "m menu")
	}
	shortcuts = append(shortcuts, "q quit")

	return helpStyle.Render(strings.Join(shortcuts, " • "))
}

// isDelegatingToWrappedModel checks if the wrapped model is delegating to another wrapped model.
// This happens when rootModel has an active flow - we should pass through without adding headers/footers.
func (w *ViewportWrapper) isDelegatingToWrappedModel() bool {
	if w.model == nil {
		return false
	}

	// Check if wrapped model is a rootModel with an active flow
	// Since rootModel is in the same package, we can check it directly
	if root, ok := w.model.(*rootModel); ok {
		return root.IsDelegating()
	}

	return false
}

func (w *ViewportWrapper) renderHelpOverlay() string {
	// Render help as overlay with semi-transparent background
	helpText := w.helpContent
	if helpText == "" {
		helpText = "No help available"
	}

	// Create overlay style
	overlayStyle := lipgloss.NewStyle().
		Width(w.width).
		Height(w.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Background(lipgloss.Color("236")). // Dark background
		Foreground(lipgloss.Color("252"))

	title := titleStyle.Render("Keyboard Shortcuts")
	content := helpText
	closeHint := helpStyle.Render("Press '?' or Esc to close")

	return overlayStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", content, "", closeHint),
	)
}
