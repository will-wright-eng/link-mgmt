# Viewport and Common Commands Design Document

## Overview

This document describes the design for a reusable viewport wrapper and common command system that can be easily applied to the root menu and all TUI flows. The system provides:

1. **Automatic terminal size handling** - Responsive layouts that adapt to terminal dimensions
2. **Viewport management** - Scrolling for content that exceeds terminal height
3. **Common commands** - Global keyboard shortcuts (help, menu, quit) available in all flows
4. **Easy integration** - Minimal changes required to existing models

---

## Architecture

### Design Pattern: Composition with Wrapper Model

We'll use a **wrapper model pattern** that wraps existing models and adds viewport + common command functionality. This allows us to:

- Keep existing models mostly unchanged
- Add functionality transparently
- Support both viewport-based and simple responsive layouts
- Handle common commands at the wrapper level

### Component Structure

```
┌────────────────────────────────────┐
│   WrapperModel (viewport + cmds)   │
│  ┌───────────────────────────────┐ │
│  │   Header (title, breadcrumb)  │ │
│  └───────────────────────────────┘ │
│  ┌───────────────────────────────┐ │
│  │   Viewport (scrollable area)  │ │
│  │  ┌──────────────────────────┐ │ │
│  │  │   Wrapped Model Content  │ │ │
│  │  │   (existing flows)       │ │ │
│  │  └──────────────────────────┘ │ │
│  └───────────────────────────────┘ │
│  ┌───────────────────────────────┐ │
│  │   Footer (help, status)       │ │
│  └───────────────────────────────┘ │
└────────────────────────────────────┘
```

---

## Core Components

### 1. ViewportWrapper

A wrapper model that handles:

- Terminal size detection and updates
- Viewport management for scrollable content
- Common command interception
- Layout calculation (header/footer space)

**Location**: `pkg/cli/tui/viewport_wrapper.go`

**Interface**:

```go
type ViewportWrapper struct {
    // Wrapped model (the actual flow)
    model tea.Model

    // Viewport for scrollable content
    viewport viewport.Model

    // Layout dimensions
    width  int
    height int

    // Layout configuration
    config ViewportConfig

    // Common command state
    showHelp bool
    helpModel tea.Model
}

type ViewportConfig struct {
    // Layout options
    ShowHeader    bool
    ShowFooter    bool
    HeaderHeight  int  // Fixed header height (0 = auto)
    FooterHeight  int  // Fixed footer height (0 = auto)

    // Viewport options
    UseViewport   bool  // Enable scrolling (false = simple responsive)
    MinWidth      int   // Minimum terminal width
    MinHeight     int   // Minimum terminal height

    // Common commands
    EnableHelp    bool  // Enable '?' for help
    EnableMenu    bool  // Enable 'm' to return to menu
    HelpContent   func() string  // Function to generate help text

    // Styling
    HeaderStyle   lipgloss.Style
    FooterStyle   lipgloss.Style
}
```

### 2. Common Commands

Global keyboard shortcuts available in all flows:

| Key | Command | Description |
|-----|---------|-------------|
| `?` | Help | Toggle help overlay showing keyboard shortcuts |
| `m` | Menu | Return to root menu (if not already there) |
| `q` / `Esc` | Quit | Exit the application |
| `Ctrl+C` | Force Quit | Force exit (handled by Bubble Tea) |

**Help Overlay**:

- Shows context-sensitive help based on current flow
- Can be toggled on/off
- Overlays on top of current view (semi-transparent background)

### 3. Layout System

**Responsive Layout Calculation**:

```go
func (w *ViewportWrapper) calculateLayout() {
    // Calculate available space
    headerH := w.config.HeaderHeight
    if headerH == 0 {
        headerH = w.measureHeader() // Auto-calculate
    }

    footerH := w.config.FooterHeight
    if footerH == 0 {
        footerH = w.measureFooter() // Auto-calculate
    }

    // Content area
    contentH := w.height - headerH - footerH
    if contentH < 1 {
        contentH = 1 // Minimum 1 line
    }

    // Update viewport dimensions
    if w.config.UseViewport {
        w.viewport.Width = w.width
        w.viewport.Height = contentH
    }
}
```

---

## Implementation Details

### ViewportWrapper Implementation

**File**: `pkg/cli/tui/viewport_wrapper.go`

```go
package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// ViewportWrapper wraps a model with viewport and common command support
type ViewportWrapper struct {
    model   tea.Model
    viewport viewport.Model
    width   int
    height  int
    config  ViewportConfig

    // Common commands
    showHelp bool
    helpContent string
}

// ViewportConfig configures the wrapper behavior
type ViewportConfig struct {
    Title         string
    ShowHeader    bool
    ShowFooter    bool
    HeaderHeight  int
    FooterHeight  int
    UseViewport   bool
    MinWidth      int
    MinHeight     int
    EnableHelp    bool
    EnableMenu    bool
    HelpContent   func() string
    OnMenu        func() tea.Cmd  // Callback for menu command
}

// NewViewportWrapper creates a new wrapper around a model
func NewViewportWrapper(model tea.Model, config ViewportConfig) *ViewportWrapper {
    vp := viewport.New(0, 0)
    vp.KeyMap = viewport.KeyMap{
        Up:     tea.Key{Type: tea.KeyUp},
        Down:   tea.Key{Type: tea.KeyDown},
        PageUp: tea.Key{Type: tea.KeyPgUp},
        PageDown: tea.Key{Type: tea.KeyPgDown},
    }

    return &ViewportWrapper{
        model:    model,
        viewport: vp,
        config:   config,
        width:    80,  // Default
        height:   24,  // Default
    }
}

func (w *ViewportWrapper) Init() tea.Cmd {
    // Initialize wrapped model
    if initer, ok := w.model.(interface{ Init() tea.Cmd }); ok {
        return initer.Init()
    }
    return nil
}

func (w *ViewportWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle window size first
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        w.width = msg.Width
        w.height = msg.Height

        // Validate minimum size
        if w.width < w.config.MinWidth {
            w.width = w.config.MinWidth
        }
        if w.height < w.config.MinHeight {
            w.height = w.config.MinHeight
        }

        // Calculate layout
        w.calculateLayout()

        // Sync viewport
        if w.config.UseViewport {
            w.viewport, _ = w.viewport.Update(msg)
        }

        // Forward to wrapped model (it may need size info)
        var cmd tea.Cmd
        w.model, cmd = w.model.Update(msg)
        return w, cmd
    }

    // Handle common commands
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "?":
            if w.config.EnableHelp {
                w.showHelp = !w.showHelp
                if w.showHelp && w.config.HelpContent != nil {
                    w.helpContent = w.config.HelpContent()
                }
                return w, nil
            }
        case "m":
            if w.config.EnableMenu && w.config.OnMenu != nil {
                return w, w.config.OnMenu()
            }
        case "ctrl+c", "q", "esc":
            // Only quit if help is not showing
            if !w.showHelp {
                return w, tea.Quit
            }
            // If help is showing, close it
            w.showHelp = false
            return w, nil
        }
    }

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
    w.model, cmd = w.model.Update(msg)

    // If using viewport, also update it
    if w.config.UseViewport {
        var vpCmd tea.Cmd
        w.viewport, vpCmd = w.viewport.Update(msg)
        cmd = tea.Batch(cmd, vpCmd)
    }

    return w, cmd
}

func (w *ViewportWrapper) View() string {
    // If help is showing, render help overlay
    if w.showHelp {
        return w.renderHelpOverlay()
    }

    // Get content from wrapped model
    content := w.model.View()

    // Apply viewport if enabled
    if w.config.UseViewport {
        w.viewport.SetContent(content)
        content = w.viewport.View()
    }

    // Build layout
    var parts []string

    // Header
    if w.config.ShowHeader {
        parts = append(parts, w.renderHeader())
    }

    // Content
    parts = append(parts, content)

    // Footer
    if w.config.ShowFooter {
        parts = append(parts, w.renderFooter())
    }

    // Join with proper spacing
    return lipgloss.JoinVertical(lipgloss.Left, parts...)
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
        w.viewport.Width = w.width
        w.viewport.Height = contentH
    }
}

func (w *ViewportWrapper) renderHeader() string {
    var b strings.Builder

    if w.config.Title != "" {
        b.WriteString(renderTitle(w.config.Title))
    }

    // Breadcrumb or navigation hint
    if w.config.EnableMenu {
        b.WriteString(helpStyle.Render("Press 'm' for menu, '?' for help"))
    } else {
        b.WriteString(helpStyle.Render("Press '?' for help"))
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

    return overlayStyle.Render(
        lipgloss.JoinVertical(lipgloss.Left, title, "", content, "",
            helpStyle.Render("Press '?' or Esc to close")),
    )
}
```

### Help Content Generator

**File**: `pkg/cli/tui/help.go`

```go
package tui

import "strings"

// HelpContent provides context-sensitive help
type HelpContent struct {
    Global    []HelpItem
    Flow      []HelpItem
    FlowName  string
}

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
        {"1-6", "Select menu option"},
        {"q / Esc", "Quit"},
        {"?", "Show this help"},
    }
    return renderHelpItems(items)
}

// ListLinksHelpContent returns help for list links flow
func ListLinksHelpContent() string {
    items := []HelpItem{
        {"↑ / ↓ / j / k", "Navigate list"},
        {"Enter", "View details / Exit"},
        {"m", "Return to menu"},
        {"q / Esc", "Quit"},
        {"?", "Show this help"},
    }
    return renderHelpItems(items)
}

// FormHelpContent returns help for form flows
func FormHelpContent() string {
    items := []HelpItem{
        {"Tab / Shift+Tab", "Navigate fields"},
        {"Enter", "Submit / Continue"},
        {"Esc", "Cancel"},
        {"m", "Return to menu"},
        {"?", "Show this help"},
    }
    return renderHelpItems(items)
}

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
```

---

## Integration Guide

### Step 1: Wrap Root Model

**File**: `pkg/cli/tui/root.go`

```go
func NewRootModel(
    apiClient *client.Client,
    scraperService *scraper.ScraperService,
    scrapeTimeoutSeconds int,
) tea.Model {
    if scrapeTimeoutSeconds <= 0 {
        scrapeTimeoutSeconds = 30
    }

    root := &rootModel{
        client:         apiClient,
        scraperService: scraperService,
        scrapeTimeout:  scrapeTimeoutSeconds,
    }

    // Wrap with viewport (simple responsive, no scrolling needed for menu)
    return NewViewportWrapper(root, ViewportConfig{
        Title:       "Link Management",
        ShowHeader:  true,
        ShowFooter:  true,
        UseViewport: false,  // Menu is short, no scrolling needed
        EnableHelp:  true,
        EnableMenu:  false,  // Already at menu
        HelpContent: RootMenuHelpContent,
        MinWidth:    60,
        MinHeight:   10,
    })
}
```

### Step 2: Wrap List Links Model

**File**: `pkg/cli/tui/list_links.go`

```go
func NewListLinksModel(c *client.Client) tea.Model {
    model := &listLinksModel{
        client: c,
    }

    // Wrap with viewport (enable scrolling for long lists)
    return NewViewportWrapper(model, ViewportConfig{
        Title:       "Your Links",
        ShowHeader:  true,
        ShowFooter:  true,
        UseViewport: true,  // Enable scrolling
        EnableHelp:  true,
        EnableMenu:  true,
        HelpContent: ListLinksHelpContent,
        OnMenu: func() tea.Cmd {
            // Return to root menu
            return tea.Quit  // Root will handle showing menu
        },
        MinWidth:    60,
        MinHeight:   10,
    })
}
```

### Step 3: Update Root Model to Handle Menu Navigation

**File**: `pkg/cli/tui/root.go`

```go
func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle menu navigation message
    switch msg := msg.(type) {
    case MenuNavigationMsg:
        // Return to menu (clear current flow)
        m.current = nil
        return m, nil
    }

    // ... rest of existing code
}
```

### Step 4: Update Forms to Use Wrapper

**File**: `pkg/cli/tui/form_add_link.go`

```go
func NewAddLinkForm(
    apiClient *client.Client,
    scraperService *scraper.ScraperService,
    scrapeTimeoutSeconds int,
) tea.Model {
    form := &addLinkForm{
        // ... existing initialization
    }

    // Wrap with viewport
    return NewViewportWrapper(form, ViewportConfig{
        Title:       "Add New Link",
        ShowHeader:  true,
        ShowFooter:  true,
        UseViewport: false,  // Forms are typically short
        EnableHelp:  true,
        EnableMenu:  true,
        HelpContent: FormHelpContent,
        MinWidth:    60,
        MinHeight:   15,
    })
}
```

---

## Migration Strategy

### Phase 1: Infrastructure (Low Risk)

1. Create `viewport_wrapper.go` with wrapper implementation
2. Create `help.go` with help content generators
3. Add tests for wrapper functionality

**Files to create**:

- `pkg/cli/tui/viewport_wrapper.go`
- `pkg/cli/tui/help.go`

### Phase 2: Root Menu (Low Risk)

1. Wrap root model with viewport wrapper
2. Test menu navigation and help
3. Verify responsive layout

**Files to modify**:

- `pkg/cli/tui/root.go`

### Phase 3: List Views (Medium Risk)

1. Wrap `listLinksModel` with viewport
2. Enable scrolling for long lists
3. Test with various list sizes

**Files to modify**:

- `pkg/cli/tui/list_links.go`
- `pkg/cli/tui/view_link_details.go`

### Phase 4: Forms (Low Risk)

1. Wrap all form models
2. Test form navigation with help
3. Verify menu navigation from forms

**Files to modify**:

- `pkg/cli/tui/form_add_link.go`
- `pkg/cli/tui/form_basic_add_link.go`
- `pkg/cli/tui/form_delete_link.go`
- `pkg/cli/tui/form_scrape_existing_link.go`

### Phase 5: Polish (Low Risk)

1. Add breadcrumbs to header
2. Enhance help content with flow-specific info
3. Add keyboard shortcut hints in footer
4. Test terminal resize handling

---

## Benefits

### 1. **Consistent UX**

- All flows have the same keyboard shortcuts
- Help is always available
- Menu navigation works from anywhere

### 2. **Responsive Design**

- Automatic terminal size handling
- Layouts adapt to available space
- Minimum size validation

### 3. **Easy Maintenance**

- Common functionality in one place
- Models remain focused on their specific logic
- Easy to add new common commands

### 4. **Production Ready**

- Handles edge cases (small terminals, resizing)
- Provides user guidance (help system)
- Consistent error handling

---

## Testing Strategy

### Unit Tests

1. **ViewportWrapper Tests**:
   - Window size handling
   - Viewport scrolling
   - Common command interception
   - Help toggle

2. **Help Content Tests**:
   - Help text generation
   - Context-sensitive content

### Integration Tests

1. **Root Menu**:
   - Menu navigation
   - Help display
   - Terminal resize

2. **List Views**:
   - Scrolling with viewport
   - Navigation with help overlay
   - Menu return

3. **Forms**:
   - Form interaction with wrapper
   - Help during form filling
   - Menu navigation

### Manual Testing

1. **Terminal Sizes**:
   - Small (40x10)
   - Medium (80x24)
   - Large (120x40)
   - Resize during use

2. **Keyboard Shortcuts**:
   - Help toggle in all flows
   - Menu navigation
   - Quit from various states

3. **Edge Cases**:
   - Very small terminal
   - Rapid resizing
   - Help during async operations

---

## Future Enhancements

### 1. Breadcrumb Navigation

Add breadcrumb trail in header:

```
Link Management > Add Link > Review
```

### 2. Search Integration

Add search command (`/`) that works across flows:

- Search links in list view
- Search help content
- Search form field values

### 3. Command Palette

Add command palette (`Ctrl+P`) for:

- Quick navigation
- Command search
- Recent actions

### 4. Customizable Shortcuts

Allow users to customize keyboard shortcuts via config file.

### 5. Mouse Support

Add mouse support for:

- Scrolling viewport
- Clicking menu items
- Selecting list items

---

## Summary

This design provides:

✅ **Reusable viewport wrapper** that handles terminal sizing automatically
✅ **Common commands** (help, menu, quit) available in all flows
✅ **Easy integration** with minimal changes to existing models
✅ **Responsive layouts** that adapt to terminal size
✅ **Scrollable content** for long lists
✅ **Consistent UX** across all flows

The wrapper pattern allows us to add this functionality without major refactoring, making it a low-risk, high-value improvement to the TUI.
