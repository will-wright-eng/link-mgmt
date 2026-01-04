package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"link-mgmt-go/pkg/cli/logger"
)

// SelectableModel is an interface that models can implement to expose their
// selection state for automatic viewport scrolling.
type SelectableModel interface {
	// GetSelectedIndex returns the currently selected index, or -1 if no selection.
	// This is used to automatically scroll the viewport to keep the selected item visible.
	GetSelectedIndex() int

	// GetItemHeight returns the height in lines of a single item in the list.
	// For links, this is 2 (title + URL). Default is 1 if not implemented.
	GetItemHeight() int

	// GetListHeaderHeight returns the number of lines before the list items start.
	// This includes titles, subtitles, etc. Default is 0 if not implemented.
	GetListHeaderHeight() int
}

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

	// Cached header/footer heights to avoid re-rendering on every layout calculation
	cachedHeaderHeight int
	cachedFooterHeight int
	headerFooterDirty  bool // Flag to indicate if heights need recalculation

	// Track previous selection for automatic scrolling
	prevSelectedIndex int
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
		model:             model,
		viewport:          vp,
		config:            config,
		width:             80, // Default
		height:            24, // Default
		prevSelectedIndex: -1, // Initialize to -1 to detect first selection change
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
		// Mark header/footer heights as dirty if dimensions changed significantly
		// (header/footer might wrap differently with different widths)
		if w.width != msg.Width {
			w.headerFooterDirty = true
		}
		w.width = msg.Width
		w.height = msg.Height

		// Validate minimum size
		if w.config.MinWidth > 0 && w.width < w.config.MinWidth {
			w.width = w.config.MinWidth
		}
		if w.config.MinHeight > 0 && w.height < w.config.MinHeight {
			w.height = w.config.MinHeight
		}

		// Calculate layout and set viewport dimensions
		w.calculateLayout()
		logger.Log("ViewportWrapper.Update: calculated layout, viewport=%dx%d", w.viewport.Width, w.viewport.Height)

		// Sync viewport with new dimensions
		if w.config.UseViewport {
			// Ensure viewport has valid dimensions before updating
			if w.viewport.Width == 0 || w.viewport.Height == 0 {
				logger.Log("ViewportWrapper.Update: viewport has invalid dimensions after calculateLayout, fixing...")
				// Recalculate with fallback defaults if needed
				if w.viewport.Width == 0 {
					w.viewport.Width = w.width
					if w.viewport.Width == 0 {
						w.viewport.Width = 80 // Fallback to default
					}
				}
				if w.viewport.Height == 0 {
					headerH := w.config.HeaderHeight
					if headerH == 0 && w.config.ShowHeader {
						headerH = 2
					}
					footerH := w.config.FooterHeight
					if footerH == 0 && w.config.ShowFooter {
						footerH = 1
					}
					w.viewport.Height = w.height - headerH - footerH
					if w.viewport.Height < 1 {
						w.viewport.Height = 1
					}
				}
				logger.Log("ViewportWrapper.Update: fixed viewport dimensions to %dx%d", w.viewport.Width, w.viewport.Height)
			}

			// Update viewport with new dimensions
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
				// Help toggle doesn't affect header/footer height, so no need to mark dirty
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

	// Automatic scrolling: if model implements SelectableModel and selection changed,
	// scroll viewport to keep selected item visible
	if w.config.UseViewport && w.model != nil {
		if selectable, ok := w.model.(SelectableModel); ok {
			currentSelected := selectable.GetSelectedIndex()
			if currentSelected != w.prevSelectedIndex {
				// If transitioning from list view (prevSelectedIndex >= 0) to non-list view (currentSelected == -1),
				// reset viewport scroll position to top
				if w.prevSelectedIndex >= 0 && currentSelected == -1 {
					logger.Log("ViewportWrapper.Update: transitioning from list to non-list view, resetting scroll position")
					w.viewport.SetYOffset(0)
				} else if currentSelected >= 0 {
					// Only scroll to selected item if we're in a list view
					logger.Log("ViewportWrapper.Update: selection changed from %d to %d, scrolling to keep visible",
						w.prevSelectedIndex, currentSelected)
					w.scrollToSelected(selectable, currentSelected)
				}
				w.prevSelectedIndex = currentSelected
			}
		}
	}

	// If using viewport, only forward scrolling keys to prevent intercepting navigation keys
	// This ensures viewport only handles scrolling when appropriate, allowing wrapped models
	// to handle navigation keys (j/k, enter, etc.) without interference
	if w.config.UseViewport {
		// Only forward scrolling keys to viewport
		// This prevents viewport from intercepting navigation keys when scrolling isn't needed
		if isScrollingKey(msg) {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				logger.Log("ViewportWrapper.Update: forwarding scrolling key %q to viewport", keyMsg.String())
			}
			var vpCmd tea.Cmd
			w.viewport, vpCmd = w.viewport.Update(msg)
			if vpCmd != nil {
				logger.Log("ViewportWrapper.Update: viewport returned a command")
			}
			cmd = tea.Batch(cmd, vpCmd)
		} else if keyMsg, ok := msg.(tea.KeyMsg); ok {
			logger.Log("ViewportWrapper.Update: key %q is not a scrolling key, skipping viewport", keyMsg.String())
		}
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
		// Ensure viewport has valid dimensions (should be set in Update, but check as safety)
		if w.viewport.Width == 0 || w.viewport.Height == 0 {
			logger.Log("ViewportWrapper.View: viewport has invalid dimensions, recalculating...")
			w.calculateLayout()
		}

		logger.Log("ViewportWrapper.View: applying viewport, viewport size=%dx%d, wrapper size=%dx%d, content length=%d",
			w.viewport.Width, w.viewport.Height, w.width, w.height, len(content))

		// Constrain content width to viewport width for proper measurement
		// The viewport needs content that matches its width to properly measure height
		contentWidth := lipgloss.Width(content)
		if contentWidth > 0 && w.viewport.Width > 0 && contentWidth != w.viewport.Width {
			// Content width doesn't match viewport width - wrap it
			// This ensures viewport can properly measure content height
			content = lipgloss.NewStyle().
				Width(w.viewport.Width).
				Render(content)
			logger.Log("ViewportWrapper.View: constrained content width from %d to %d", contentWidth, w.viewport.Width)
		}

		// Set content - viewport will handle scrolling
		// Note: SetContent should be called every render cycle, which is correct here
		w.viewport.SetContent(content)
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
	// Calculate header height
	headerH := w.config.HeaderHeight
	if headerH == 0 && w.config.ShowHeader {
		// Use cached height if available and not dirty
		if !w.headerFooterDirty && w.cachedHeaderHeight > 0 {
			headerH = w.cachedHeaderHeight
			logger.Log("ViewportWrapper.calculateLayout: using cached header height: %d", headerH)
		} else {
			// Measure actual header height dynamically
			header := w.renderHeader()
			headerH = lipgloss.Height(header)
			w.cachedHeaderHeight = headerH
			logger.Log("ViewportWrapper.calculateLayout: measured header height: %d", headerH)
		}
	}

	// Calculate footer height
	footerH := w.config.FooterHeight
	if footerH == 0 && w.config.ShowFooter {
		// Use cached height if available and not dirty
		if !w.headerFooterDirty && w.cachedFooterHeight > 0 {
			footerH = w.cachedFooterHeight
			logger.Log("ViewportWrapper.calculateLayout: using cached footer height: %d", footerH)
		} else {
			// Measure actual footer height dynamically
			footer := w.renderFooter()
			footerH = lipgloss.Height(footer)
			w.cachedFooterHeight = footerH
			logger.Log("ViewportWrapper.calculateLayout: measured footer height: %d", footerH)
		}
	}

	// Mark as clean after measurement
	w.headerFooterDirty = false

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

// scrollToSelected calculates the position of the selected item and scrolls the viewport
// to keep it visible. This implements automatic scrolling when navigation keys change selection.
func (w *ViewportWrapper) scrollToSelected(selectable SelectableModel, selectedIndex int) {
	if selectedIndex < 0 {
		// No selection, nothing to scroll to
		return
	}

	// Get item dimensions from the model
	itemHeight := selectable.GetItemHeight()
	if itemHeight <= 0 {
		itemHeight = 1 // Default to 1 line per item
	}
	listHeaderHeight := selectable.GetListHeaderHeight()
	if listHeaderHeight < 0 {
		listHeaderHeight = 0
	}

	// Calculate the Y position of the selected item in the content
	// Position = header lines + (selected index * item height)
	selectedY := listHeaderHeight + (selectedIndex * itemHeight)

	// Get current viewport scroll position and height
	currentYOffset := w.viewport.YOffset
	viewportHeight := w.viewport.Height

	if viewportHeight <= 0 {
		// Viewport not initialized yet, can't scroll
		logger.Log("ViewportWrapper.scrollToSelected: viewport height is 0, skipping scroll")
		return
	}

	// Check if selected item is already visible
	// Item is visible if: currentYOffset <= selectedY < currentYOffset + viewportHeight
	// But we want the item to be fully visible, so we check if the item's bottom is visible
	itemBottomY := selectedY + itemHeight
	viewportBottom := currentYOffset + viewportHeight

	isVisible := selectedY >= currentYOffset && itemBottomY <= viewportBottom

	if !isVisible {
		// Item is not visible, need to scroll
		if selectedY < currentYOffset {
			// Item is above visible area, scroll up to show it
			newOffset := selectedY
			// Add some padding: show a few items above if possible
			if newOffset > 2 {
				newOffset -= 2
			}
			w.viewport.SetYOffset(newOffset)
			logger.Log("ViewportWrapper.scrollToSelected: scrolled up, new offset=%d (selected at %d)",
				newOffset, selectedY)
		} else {
			// Item is below visible area, scroll down to show it
			// Position item near the bottom of viewport with some padding
			newOffset := selectedY - viewportHeight + itemHeight + 2
			if newOffset < 0 {
				newOffset = 0
			}
			w.viewport.SetYOffset(newOffset)
			logger.Log("ViewportWrapper.scrollToSelected: scrolled down, new offset=%d (selected at %d)",
				newOffset, selectedY)
		}
	} else {
		logger.Log("ViewportWrapper.scrollToSelected: item already visible at Y=%d (offset=%d, height=%d)",
			selectedY, currentYOffset, viewportHeight)
	}
}

// isScrollingKey checks if a message is a key that should be handled by the viewport for scrolling.
// This prevents the viewport from intercepting navigation keys (j/k, enter, etc.) used by wrapped models.
func isScrollingKey(msg tea.Msg) bool {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}

	key := keyMsg.String()
	switch key {
	case "up", "down", "pgup", "pgdown", "home", "end":
		// Standard scrolling keys
		return true
	case "ctrl+u", "ctrl+d":
		// Half-page scrolling
		return true
	case "ctrl+b", "ctrl+f":
		// Page scrolling (alternative)
		return true
	case " ", "shift+space":
		// Space for page down, shift+space for page up
		return true
	default:
		// All other keys (including j/k, enter, etc.) are not scrolling keys
		// This allows wrapped models to handle navigation without interference
		return false
	}
}
