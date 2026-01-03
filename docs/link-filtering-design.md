# Link Filtering Design Document

## Overview

This document outlines patterns and methods for implementing a simple string matching filter for links. The filter will search across the `url`, `title`, and `description` fields.

## Current State

- **Link Model**: Contains `ID`, `UserID`, `URL`, `Title`, `Description`, `Text`, `CreatedAt`, `UpdatedAt`
- **API**: `GET /api/v1/links` returns all links for a user, ordered by `created_at DESC`
- **CLI/TUI**: Displays all links in a scrollable list view
- **Database**: PostgreSQL with simple query: `SELECT ... FROM links WHERE user_id = $1 ORDER BY created_at DESC`

## Filter Requirements

- **Simple string match**: Case-insensitive substring search
- **Search fields**: URL, Title, Description
- **Match logic**: A link matches if the search term appears in ANY of the three fields (OR logic)

## Implementation Patterns

### Pattern 1: Server-Side Filtering (Recommended for API)

**Description**: Filtering happens at the database level via SQL query.

**Pros**:

- Efficient for large datasets
- Reduces network transfer
- Leverages database indexing

**Cons**:

- Requires API changes
- More complex SQL query

**Use Cases**:

- API endpoints
- Large datasets (>500 links)

### Pattern 2: Client-Side Filtering (Recommended for CLI/TUI)

**Description**: Fetch all links, filter in memory on the client.

**Pros**:

- Simple to implement
- Fast for small datasets
- No API changes needed
- Enables instant filtering feedback

**Cons**:

- Inefficient for large datasets
- Requires loading all data

**Use Cases**:

- CLI/TUI with small to medium datasets (<1000 links)
- When filtering needs to be instant/real-time

## API Design

### Endpoint Design

Add a `search` query parameter to the existing endpoint:

```
GET /api/v1/links?search=term
```

**Request Model**:

```go
type LinkFilterRequest struct {
    Search string `form:"search"` // Optional search term
}
```

**Response**: Same as current response (array of links), just filtered.

### Database Query Implementation

```go
func (db *DB) GetLinksByUserIDWithSearch(
    ctx context.Context,
    userID uuid.UUID,
    searchTerm string,
) ([]models.Link, error) {
    query := `
        SELECT id, user_id, url, title, description, text, created_at, updated_at
        FROM links
        WHERE user_id = $1`

    args := []interface{}{userID}

    // Add search filter if provided
    if searchTerm != "" {
        query += ` AND (
            url ILIKE $2 OR
            title ILIKE $2 OR
            description ILIKE $2
        )`
        args = append(args, "%"+searchTerm+"%")
    }

    query += ` ORDER BY created_at DESC`

    rows, err := db.Pool.Query(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query links: %w", err)
    }
    defer rows.Close()

    var links []models.Link
    for rows.Next() {
        var link models.Link
        err := rows.Scan(
            &link.ID, &link.UserID, &link.URL, &link.Title,
            &link.Description, &link.Text, &link.CreatedAt, &link.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan link: %w", err)
        }
        links = append(links, link)
    }

    return links, rows.Err()
}
```

### Handler Implementation

```go
func ListLinks(db *db.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.MustGet("userID").(uuid.UUID)

        searchTerm := c.Query("search")

        links, err := db.GetLinksByUserIDWithSearch(c.Request.Context(), userID, searchTerm)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        c.JSON(http.StatusOK, links)
    }
}
```

## CLI/TUI Design

### Integrated In-Memory Filtering

**Description**: Filter input appears inline in the list view. Filtering happens in real-time as the user types. The filtered list is navigated and selected exactly like the original list - all existing functionality (view, delete, scrape) works seamlessly with the filtered results.

**Key Design Principles**:

- **Seamless Integration**: The filtered list (`m.links`) replaces the original list for all navigation and selection operations
- **Real-time Filtering**: Filter updates as user types (no need to press Enter)
- **Preserve Original**: Keep original unfiltered list (`allLinks`) to re-apply filter after data changes
- **No Mode Switching**: Filter input is always available in list view, no separate "filter mode"

**UI Flow**:

1. User presses `/` to focus filter input (appears at top of list)
2. User types search term - list filters in real-time
3. User navigates filtered list with arrow keys (same as before)
4. User selects link and performs actions (view/delete/scrape) - works normally
5. Press `Esc` or `Ctrl+U` to clear filter and restore full list

**Model Structure**:

```go
type manageLinksModel struct {
    // ... existing fields ...
    client         *client.Client
    scraperService *scraper.ScraperService

    allLinks    []models.Link  // Original unfiltered list from API
    links       []models.Link  // Filtered list (what's displayed/navigated)
    selected    int
    step        int
    err         error
    ready       bool

    // Filter state
    filterInput textinput.Model
    filterFocused bool  // Whether filter input has focus

    // ... other existing fields (confirm, scrapeState, width) ...
}
```

**Initialization**:

```go
func NewManageLinksModel(
    c *client.Client,
    scraperService *scraper.ScraperService,
    timeoutSeconds int,
) tea.Model {
    // ... existing initialization ...

    filterInput := textinput.New()
    filterInput.Placeholder = "Filter by URL, title, or description... (Press / to focus)"
    filterInput.Prompt = "Filter: "
    filterInput.CharLimit = 200

    model := &manageLinksModel{
        // ... existing fields ...
        filterInput: filterInput,
    }

    // ... rest of initialization ...
}
```

**Filtering Function**:

```go
// filterLinks performs case-insensitive substring matching on URL, title, and description
func filterLinks(links []models.Link, searchTerm string) []models.Link {
    if searchTerm == "" {
        return links
    }

    search := strings.ToLower(strings.TrimSpace(searchTerm))
    var filtered []models.Link

    for _, link := range links {
        // Check URL
        if strings.Contains(strings.ToLower(link.URL), search) {
            filtered = append(filtered, link)
            continue
        }

        // Check Title
        if link.Title != nil && strings.Contains(strings.ToLower(*link.Title), search) {
            filtered = append(filtered, link)
            continue
        }

        // Check Description
        if link.Description != nil && strings.Contains(strings.ToLower(*link.Description), search) {
            filtered = append(filtered, link)
            continue
        }
    }

    return filtered
}

// applyFilter updates m.links based on current filter input value
func (m *manageLinksModel) applyFilter() {
    searchTerm := m.filterInput.Value()
    m.links = filterLinks(m.allLinks, searchTerm)

    // Adjust selected index if current selection is out of bounds
    if m.selected >= len(m.links) {
        m.selected = 0
        if len(m.links) > 0 {
            m.selected = len(m.links) - 1
        }
    }
}
```

**Update Handler Integration**:

```go
func (m *manageLinksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ... existing message handling ...

    switch msg := msg.(type) {
    case managelinks.LinksLoadedMsg:
        if msg.Err != nil {
            m.err = msg.Err
            m.ready = true
            return m, nil
        }
        // Store original list and apply current filter
        m.allLinks = msg.Links
        m.applyFilter()  // Apply filter to new data
        m.ready = true
        return m, nil

    case tea.KeyMsg:
        // Handle filter input focus
        if m.step == managelinks.StepListLinks {
            if msg.String() == "/" && !m.filterFocused {
                // Focus filter input
                m.filterFocused = true
                m.filterInput.Focus()
                return m, textinput.Blink
            }

            if m.filterFocused {
                switch msg.String() {
                case "esc":
                    // Clear filter and blur
                    m.filterFocused = false
                    m.filterInput.Blur()
                    m.filterInput.SetValue("")
                    m.applyFilter()  // Restore full list
                    return m, nil
                case "ctrl+u":
                    // Clear filter input but keep focus
                    m.filterInput.SetValue("")
                    m.applyFilter()  // Restore full list
                    return m, nil
                case "enter":
                    // Blur filter but keep filter applied
                    m.filterFocused = false
                    m.filterInput.Blur()
                    return m, nil
                }

                // Update filter input (real-time filtering)
                var cmd tea.Cmd
                m.filterInput, cmd = m.filterInput.Update(msg)
                m.applyFilter()  // Re-apply filter as user types
                return m, cmd
            }
        }

        // Existing key handling for list navigation, etc.
        switch m.step {
        case managelinks.StepListLinks:
            return m.handleListKeys(msg)
        // ... rest of existing step handling ...
        }
    }

    // ... rest of existing Update logic ...
}
```

**View Integration**:

```go
func (m *manageLinksModel) renderList() string {
    if len(m.links) == 0 {
        // Show different message if filtering vs no links at all
        if m.filterInput.Value() != "" {
            return renderEmptyState(fmt.Sprintf("No links match '%s'", m.filterInput.Value()))
        }
        return renderEmptyState("No links found.")
    }

    maxWidth := m.getMaxWidth()

    var b strings.Builder

    // Render filter input at top
    filterDisplay := m.filterInput.View()
    if m.filterFocused {
        filterDisplay = focusedStyle.Render(filterDisplay)
    }
    b.WriteString(filterDisplay)
    b.WriteString("\n\n")

    // Show filter status if active
    if m.filterInput.Value() != "" {
        status := fmt.Sprintf("Showing %d of %d links", len(m.links), len(m.allLinks))
        b.WriteString(helpStyle.Render(status))
        b.WriteString("\n\n")
    }

    // Render link list (existing logic)
    b.WriteString(renderLinkList(m.links, m.selected, "", "Select a link:", maxWidth))

    // Help text
    helpText := "(Use ↑/↓ or j/k to navigate, Enter to select, / to filter, Esc to quit)"
    if m.filterFocused {
        helpText = "(Type to filter, Enter to apply, Esc/Ctrl+U to clear, Esc again to quit)"
    }
    b.WriteString(helpStyle.Render(helpText))
    b.WriteString("\n")

    return b.String()
}
```

**Key Integration Points**:

1. **Links Reloaded**: When links are reloaded (after delete, scrape, etc.), update `allLinks` and re-apply filter:

   ```go
   case managelinks.LinksLoadedMsg:
       m.allLinks = msg.Links
       m.applyFilter()  // Re-apply current filter to new data
   ```

2. **Selection Bounds**: Navigation already uses `len(m.links)`, so it automatically works with filtered list:

   ```go
   // Existing code already works:
   if newSelected, handled := handleListNavigation(msg.String(), m.selected, len(m.links)); handled {
       m.selected = newSelected
   }
   ```

3. **Action Operations**: All actions use `m.links[m.selected]`, so they work with filtered list:

   ```go
   // Existing code already works:
   link := m.links[m.selected]  // Gets the selected link from filtered list
   ```

4. **Viewport Scrolling**: The viewport wrapper uses `GetSelectedIndex()` which returns `m.selected`, so scrolling works correctly with filtered list.

**Why This Integration Works Seamlessly**:

The key insight is that by replacing `m.links` with the filtered list, all existing functionality automatically works because:

- **Navigation**: `handleListNavigation()` uses `len(m.links)` for bounds checking - works with filtered list
- **Selection**: `m.selected` is an index into `m.links` - works with filtered list
- **Actions**: All operations use `m.links[m.selected]` to get the selected link - works with filtered list
- **Rendering**: `renderLinkList(m.links, m.selected, ...)` renders the filtered list - works correctly
- **Viewport**: Scrolling uses `GetSelectedIndex()` which returns `m.selected` - works with filtered list

The only changes needed are:

1. Store original list separately (`allLinks`)
2. Filter `allLinks` into `m.links` when filter changes
3. Re-apply filter when data reloads

All other code continues to work unchanged because it operates on `m.links` and `m.selected`, which now point to the filtered list instead of the original list.

## Database Considerations

### Simple Indexing (Optional)

For better performance with search queries, you can add basic indexes:

```sql
-- Index on user_id (likely already exists)
CREATE INDEX IF NOT EXISTS idx_links_user_id ON links (user_id);

-- Indexes for text search (optional, helps with ILIKE queries)
CREATE INDEX IF NOT EXISTS idx_links_url_lower ON links (LOWER(url));
CREATE INDEX IF NOT EXISTS idx_links_title_lower ON links (LOWER(title)) WHERE title IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_links_description_lower ON links (LOWER(description)) WHERE description IS NOT NULL;
```

**Note**: For small datasets (<1000 links), these indexes may not be necessary. PostgreSQL's ILIKE with `%term%` pattern matching can be slow on large datasets, but for simple use cases, it's acceptable.

## Examples

### Example 1: API Search

**Request**:

```
GET /api/v1/links?search=github
```

**Response**:

```json
[
  {
    "id": "...",
    "url": "https://github.com/user/repo",
    "title": "GitHub Repository",
    "description": "A project on GitHub",
    ...
  }
]
```

### Example 2: CLI Integrated Filtering

**User Interaction**:

```
┌─────────────────────────────────────────┐
│ Filter: github                          │
│ Showing 3 of 15 links                   │
│                                         │
│ > 1. GitHub Repository                  │
│    https://github.com/user/repo         │
│    2. Another GitHub Link               │
│    https://github.com/org/project       │
│    3. GitHub Docs                       │
│    https://docs.github.com/guide        │
│                                         │
│ (Use ↑/↓ to navigate, Enter to select) │
└─────────────────────────────────────────┘
```

**Flow**:

1. User presses `/` - filter input gets focus
2. User types "github" - list filters in real-time as they type
3. User navigates filtered list with arrow keys (same as normal)
4. User presses Enter on a link - action menu appears (works normally)
5. User performs actions (view/delete/scrape) - all work with selected filtered link
6. User presses Esc to clear filter - full list restored

### Example 3: Filter Persistence

**Scenario**: User filters, deletes a link, filter remains applied:

1. User filters to show 5 links
2. User selects and deletes one link
3. After deletion, links reload from API
4. Filter is automatically re-applied to new data
5. User sees 4 filtered links (filter persisted)

## Implementation Recommendations

### Phase 1: CLI/TUI In-Memory Filtering (Recommended)

**Implementation Steps**:

1. **Add filter state to model**:
   - Add `allLinks []models.Link` to store original unfiltered list
   - Add `filterInput textinput.Model` for filter input
   - Add `filterFocused bool` to track focus state

2. **Initialize filter input** in `NewManageLinksModel()`

3. **Implement `filterLinks()` function** (pure function, easy to test)

4. **Implement `applyFilter()` method** that:
   - Filters `allLinks` into `m.links`
   - Adjusts `m.selected` if out of bounds

5. **Update `LinksLoadedMsg` handler** to:
   - Store in `allLinks`
   - Call `applyFilter()` to populate `m.links`

6. **Update `handleListKeys()`** to:
   - Handle `/` key to focus filter input
   - Handle filter input updates when focused
   - Re-apply filter on each keystroke (real-time)

7. **Update `renderList()`** to:
   - Show filter input at top
   - Show filter status (e.g., "Showing 5 of 20 links")
   - Render filtered list normally

**Key Benefits**:

- ✅ **No API changes needed** - works immediately
- ✅ **Seamless integration** - filtered list works with all existing features
- ✅ **Real-time feedback** - instant filtering as user types
- ✅ **Simple implementation** - minimal changes to existing code
- ✅ **No mode switching** - filter always available, no separate mode

**How It Works**:

- `m.links` becomes the filtered list (replaces original list for display/navigation)
- All existing code that uses `m.links` and `m.selected` works unchanged
- Filter persists across data reloads (delete, scrape, etc.)
- Navigation, selection, and actions all work normally with filtered list

### Phase 2: API Server-Side Filtering (Optional, for large datasets)

Only needed if you have >1000 links and performance becomes an issue.

1. Update `GetLinksByUserID` to accept optional search parameter
2. Update handler to read `search` query parameter
3. Add database indexes if performance becomes an issue

**Pros**: More efficient for large datasets, consistent across clients

## Testing Considerations

### Unit Tests

- `filterLinks()` function with various search terms
- Edge cases: empty search, nil title/description, special characters

### Integration Tests

- API endpoint with search parameter
- API endpoint without search parameter (should return all links)

### Manual Testing

- TUI filter mode interactions
- Filter persistence during navigation
- Clearing filter restores original list
