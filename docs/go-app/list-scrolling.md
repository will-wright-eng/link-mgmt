# List Scrolling in Manage Links View

This document explains how scrolling works when viewing a list of stored links in the CLI application.

## Overview

The "Manage Links" view displays a scrollable list of all stored links. When the list content exceeds the available viewport height, users can scroll to view all links while maintaining the ability to navigate and select items.

## Scrolling Architecture

### Viewport Wrapper

The manage links view is wrapped with a `ViewportWrapper` that enables scrolling:

```go
// From manage_links.go
return NewViewportWrapper(model, ViewportConfig{
    Title:       "Manage Links",
    ShowHeader:  true,
    ShowFooter:  true,
    UseViewport: true, // Enable scrolling for long link lists
    EnableHelp:  true,
    EnableMenu:  true,
    MinWidth:  60,
    MinHeight: 10,
})
```

The `UseViewport: true` setting enables the viewport scrolling functionality, which uses the `bubbles/viewport` package from Charm.

## Key Bindings

### Navigation Keys (List Selection)

These keys are handled by the `manageLinksModel` and are **not** intercepted by the viewport:

- **↑ / ↓** - Move selection up/down in the list
- **j / k** - Vim-style navigation (j = down, k = up)
- **Enter** - Select the highlighted link and open action menu

**Important:** These keys work for navigating the list selection, not for scrolling the viewport. The viewport automatically scrolls to keep the selected item visible.

### Scrolling Keys (Viewport Scrolling)

These keys are handled by the viewport and allow scrolling when content exceeds the viewport height:

- **↑ / ↓** - Scroll viewport up/down (only when content is scrollable)
- **Page Up / Page Down** - Scroll one page up/down
- **Home / End** - Jump to top/bottom of content
- **Ctrl+U / Ctrl+D** - Half-page scrolling
- **Ctrl+B / Ctrl+F** - Page scrolling (alternative)
- **Space / Shift+Space** - Page down / Page up

**Note:** The viewport only receives scrolling keys when they are explicitly scrolling keys. Navigation keys (j/k, Enter) are **never** forwarded to the viewport, ensuring they always work for list navigation.

### Other Keys

- **Esc / q** - Quit or go back
- **?** - Toggle help overlay
- **m** - Return to main menu

## How Scrolling Works

### Automatic Scrolling

**✅ IMPLEMENTED:** Automatic scrolling is now fully implemented.

When you navigate with ↑/↓ or j/k:

1. The `manageLinksModel` updates the `selected` index ✅
2. The viewport renders the updated list ✅
3. If the selected item is outside the visible area, the viewport automatically scrolls to show it ✅

**How It Works:**

The `ViewportWrapper` implements automatic scrolling through the `SelectableModel` interface:

1. **Selection Detection**: Models implementing `SelectableModel` expose their selected index via `GetSelectedIndex()`
2. **Position Calculation**: The wrapper calculates the Y position of the selected item based on:
   - List header height (subtitle + blank lines)
   - Item height (2 lines per link: title + URL)
   - Selected index
3. **Visibility Check**: Checks if the selected item is within the current viewport's visible area
4. **Automatic Scrolling**: If the item is not visible, programmatically scrolls the viewport using `viewport.SetYOffset()` to bring it into view

**Implementation Details:**

- The `manageLinksModel` implements `SelectableModel` with:
    - `GetSelectedIndex()`: Returns the current selection (only in list view step)
    - `GetItemHeight()`: Returns 2 (each link = title + URL)
    - `GetListHeaderHeight()`: Returns 2 (subtitle + blank line)
- Scrolling happens automatically in `ViewportWrapper.Update()` after the wrapped model processes navigation keys
- The viewport scrolls with padding (2 lines) to show context around the selected item

### Manual Scrolling

When the list content is longer than the viewport height, you can manually scroll using the scrolling keys listed above. This allows you to:

- Preview links above or below the current selection
- Scroll through the entire list without changing selection
- Jump to specific positions in the list

### Content Height Calculation

The viewport calculates the available height by:

1. Getting the terminal height
2. Subtracting header height (measured dynamically)
3. Subtracting footer height (measured dynamically)
4. Using the remaining space for the scrollable content area

```go
// From viewport_wrapper.go
contentH := w.height - headerH - footerH
w.viewport.Height = contentH
```

## Implementation Details

### Key Handling Separation

The viewport wrapper uses an optimized key handling system that prevents conflicts:

```go
// From viewport_wrapper.go
func isScrollingKey(msg tea.Msg) bool {
    keyMsg, ok := msg.(tea.KeyMsg)
    if !ok {
        return false
    }

    key := keyMsg.String()
    switch key {
    case "up", "down", "pgup", "pgdown", "home", "end":
        return true
    case "ctrl+u", "ctrl+d", "ctrl+b", "ctrl+f":
        return true
    case " ", "shift+space":
        return true
    default:
        // Navigation keys (j/k, enter, etc.) return false
        return false
    }
}
```

This ensures:

- Navigation keys (j/k, Enter) are handled by the model
- Scrolling keys are handled by the viewport
- No key conflicts or interception issues

### Content Rendering

The list is rendered with width awareness:

```go
// From manage_links.go
func (m *manageLinksModel) renderList() string {
    maxWidth := m.width
    if maxWidth == 0 {
        maxWidth = 80
    }

    s := renderLinkList(m.links, m.selected, "", "Select a link:", maxWidth)
    s += helpStyle.Render("(Use ↑/↓ or j/k to navigate, Enter to select, Esc to quit)") + "\n"
    return s
}
```

The content width is constrained to match the viewport width, ensuring proper height measurement and scrolling behavior.

### Viewport Content Setting

Every render cycle, the viewport receives the full list content:

```go
// From viewport_wrapper.go
w.viewport.SetContent(content)
content = w.viewport.View()
```

The viewport then:

1. Measures the content height
2. Determines if scrolling is needed
3. Renders only the visible portion
4. Handles scroll position

## User Experience

### When Scrolling is Active

Scrolling is active when:

- The list has more links than can fit in the viewport
- Content height exceeds viewport height

### When Scrolling is Not Needed

Scrolling is automatically disabled when:

- All links fit within the viewport
- Content height is less than or equal to viewport height

In this case, scrolling keys are still forwarded to the viewport, but the viewport won't scroll (there's nothing to scroll).

### Visual Feedback

- **Selected item** is marked with a `→` arrow
- **Selected item styling** uses highlighted colors
- **Help text** at the bottom shows available navigation keys
- **Footer** shows common shortcuts (help, menu, quit)

## Best Practices

### For Users

1. **Use ↑/↓ or j/k for navigation** - These keys move the selection and automatically scroll to keep the selected item visible
2. **Use Page Up/Down for quick browsing** - When you want to see other parts of the list without changing selection
3. **Use Home/End to jump** - Quickly navigate to the first or last link
4. **Press ? for help** - See all available keyboard shortcuts

### For Developers

1. **Navigation keys are handled by the model** - Don't forward j/k, Enter to viewport
2. **Scrolling keys are handled by viewport** - Only forward explicit scrolling keys
3. **Content width must match viewport width** - Ensures proper height measurement
4. **Use dynamic height measurement** - Header/footer heights are measured, not hardcoded

## Technical Notes

### Viewport Dimensions

The viewport dimensions are calculated in `Update()` when handling `WindowSizeMsg`:

```go
case tea.WindowSizeMsg:
    w.width = msg.Width
    w.height = msg.Height
    w.calculateLayout()  // Sets viewport.Width and viewport.Height
    w.viewport, vpCmd = w.viewport.Update(msg)
```

### Content Width Constraint

Content is constrained to viewport width in `View()`:

```go
contentWidth := lipgloss.Width(content)
if contentWidth > 0 && w.viewport.Width > 0 && contentWidth != w.viewport.Width {
    content = lipgloss.NewStyle().
        Width(w.viewport.Width).
        Render(content)
}
```

This ensures the viewport can accurately measure content height for scrolling.

### Performance Considerations

- Content is re-rendered every cycle (necessary for selection updates)
- Viewport efficiently handles scrolling without re-rendering entire content
- Header/footer heights are cached to avoid unnecessary re-measurement
- Width is passed to models to avoid post-render constraints when possible

## Implementation Details: Automatic Scrolling

### How It Works

The automatic scrolling functionality is implemented in `ViewportWrapper.Update()` in `viewport_wrapper.go`. The implementation uses the `SelectableModel` interface to allow models to expose their selection state.

### The SelectableModel Interface

Models that want automatic scrolling implement this interface:

```go
type SelectableModel interface {
    GetSelectedIndex() int      // Returns selected index, or -1 if no selection
    GetItemHeight() int          // Height in lines of a single item (default: 1)
    GetListHeaderHeight() int    // Lines before list items start (default: 0)
}
```

### The Scrolling Algorithm

1. **Selection Change Detection**: After the wrapped model processes a message, the wrapper checks if it implements `SelectableModel` and if the selected index changed
2. **Position Calculation**: Calculates the Y position: `listHeaderHeight + (selectedIndex * itemHeight)`
3. **Visibility Check**: Determines if the item is visible: `currentYOffset <= selectedY < currentYOffset + viewportHeight`
4. **Automatic Scrolling**:
   - If item is above visible area: Scrolls up to show it (with 2 lines of padding)
   - If item is below visible area: Scrolls down to position it near the bottom (with 2 lines of padding)

### Example: manageLinksModel Implementation

```go
func (m *manageLinksModel) GetSelectedIndex() int {
    if m.step == manageStepListLinks {
        return m.selected
    }
    return -1
}

func (m *manageLinksModel) GetItemHeight() int {
    return 2  // Each link = title + URL
}

func (m *manageLinksModel) GetListHeaderHeight() int {
    return 2  // Subtitle + blank line
}
```

## Troubleshooting

### Scrolling Not Working

If scrolling doesn't work:

1. **Check viewport is enabled** - `UseViewport: true` in config
2. **Verify content exceeds viewport** - Content must be taller than available height
3. **Check key handling** - Ensure scrolling keys are being forwarded
4. **Verify dimensions** - Viewport must have valid width and height
5. **Automatic scrolling** - Automatic scrolling to keep selected item visible is implemented via the `SelectableModel` interface (see Implementation Details above)

### Navigation Keys Not Working

If j/k or Enter don't work:

1. **Check key forwarding** - Navigation keys should NOT go to viewport
2. **Verify model handling** - `handleListKeys()` should process these keys
3. **Check step state** - Must be in `manageStepListLinks` state

### Content Not Visible

If content is cut off:

1. **Check width constraint** - Content should match viewport width
2. **Verify height calculation** - Header/footer heights should be correct
3. **Check minimum dimensions** - `MinWidth` and `MinHeight` may be restricting

## Related Documentation

- [Viewport Best Practices](./viewport-best-practices.md) - General viewport implementation guidelines
- [CLI Implementation Plan](./cli-implementation-plan.md) - Overall CLI architecture
- [Viewport and Common Commands Design](./viewport-and-common-commands-design.md) - Design decisions
