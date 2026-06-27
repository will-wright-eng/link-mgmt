# Scrape Status Implementation Plan

> **Status: Planned** — Not implemented. There is no `003_add_scrape_status` migration, and no `Scraped`/`ScrapeSuccessful` fields on the `Link` model or in `pkg/db/queries.go`.

## Overview

Add fields to track whether a link has been scraped and whether the scrape was successful. This will help users understand the scraping state of their links and enable features like filtering, retry logic, and status reporting.

## Design Decisions

### Field Structure

Two separate boolean fields:

- `scraped` (boolean) - indicates if scraping was attempted
- `scrape_successful` (boolean, nullable) - indicates if scrape succeeded (null = not scraped, true = success, false = failed)

This approach is flexible, allows NULL to represent "not attempted", and is easier to query/filter.

### Database Schema

```sql
ALTER TABLE links
ADD COLUMN scraped BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN scrape_successful BOOLEAN;
```

- `scraped`: `false` = never attempted, `true` = attempted at least once
- `scrape_successful`: `NULL` = not scraped, `true` = last scrape succeeded, `false` = last scrape failed

## Implementation Steps

### 1. Database Migration

**File**: `link-mgmt/migrations/003_add_scrape_status.sql`

- Add `scraped` and `scrape_successful` columns to `links` table
- Set default values for existing rows (`scraped = false`, `scrape_successful = NULL`)
- Consider adding index on `scraped` if filtering by scrape status will be common

### 2. Model Updates

**File**: `link-mgmt/pkg/models/link.go`

- Add `Scraped bool` and `ScrapeSuccessful *bool` fields to `Link` struct
- Add JSON tags: `json:"scraped"` and `json:"scrape_successful,omitempty"`
- Update `LinkUpdate` struct to include optional `Scraped *bool` and `ScrapeSuccessful *bool` fields (for manual updates if needed)

### 3. Database Layer Updates

**File**: `link-mgmt/pkg/db/queries.go`

Update all queries that interact with links:

- **`CreateLink`**: Include new fields in INSERT and RETURNING clauses
- **`GetLinksByUserID`**: Add fields to SELECT and Scan operations
- **`GetLinkByID`**: Add fields to SELECT and Scan operations
- **`UpdateLink`**: Add support for updating `scraped` and `scrape_successful` fields in dynamic UPDATE query

### 4. Service Layer Updates

**File**: `link-mgmt/pkg/services/link_service.go`

Update scraping methods to track status:

- **`CreateLinkWithScraping`**:
    - Set `scraped = true` when scraping is attempted (regardless of success/failure)
    - Set `scrape_successful = true` if scraping succeeds and content is updated
    - Set `scrape_successful = false` if scraping fails (error returned)
    - Update link with scrape status via `UpdateLink`

- **`EnrichLink`**:
    - Set `scraped = true` when scraping is attempted
    - Set `scrape_successful = true` if scraping succeeds
    - Set `scrape_successful = false` if scraping fails (error returned)
    - Update link with scrape status via `UpdateLink`

**Note**: Consider whether to track status even when scraping succeeds but no content changes (e.g., empty scrape result). Recommendation: still mark as `scrape_successful = true` if the scrape completed without errors.

### 5. API Layer

**File**: `link-mgmt/pkg/api/handlers/links.go`

- No changes required - JSON serialization will automatically include new fields from the model
- Existing endpoints will return scrape status in responses

### 6. CLI/Client Layer

**File**: `link-mgmt/pkg/cli/tui/helpers.go`

Add scrape status display to TUI:

- **Create helper function `formatScrapeStatus(link models.Link) string`**:
    - Returns formatted status indicator with appropriate styling
    - Status indicators:
        - `scraped = false`: "○ Not scraped" (muted/gray style)
        - `scraped = true, scrape_successful = true`: "✓ Scraped" (success/green style)
        - `scraped = true, scrape_successful = false`: "✗ Scrape failed" (error/red style)
    - Use existing styles from `styles.go` or create new status-specific styles

- **Update `renderLinkList` function**:
    - Add scrape status indicator next to each link title or URL
    - Format: `[Title] [Status]` or `[Title]` on one line, `[URL] [Status]` on next line
    - Keep status compact to avoid cluttering the list view

- **Update `renderLinkDetails` function**:
    - Add "Scrape Status:" field to the details view
    - Display full status text with appropriate styling
    - Place after "Created:" field or in a logical position

- **Update `renderLinkDetailsFull` function**:
    - Inherits scrape status from `renderLinkDetails`, no additional changes needed

**File**: `link-mgmt/pkg/cli/tui/styles.go` (if needed)

- Add new styles for scrape status indicators:
    - `scrapeStatusNotScrapedStyle` - muted/gray for not scraped
    - `scrapeStatusSuccessStyle` - green for successful scrape
    - `scrapeStatusFailedStyle` - red for failed scrape

**Future considerations** (out of scope for initial implementation):

- Add filtering options (e.g., show only unscraped links, show only failed scrapes)
- Add keyboard shortcut to retry failed scrapes from list view

## Testing Considerations

1. **Migration Testing**: Verify migration works on existing database with data
2. **Backward Compatibility**: Ensure existing links default to `scraped = false`, `scrape_successful = NULL`
3. **Scraping Flow Testing**:
   - Test successful scrape updates status correctly
   - Test failed scrape sets `scraped = true`, `scrape_successful = false`
   - Test scraping disabled leaves status as default
4. **API Response Testing**: Verify new fields appear in JSON responses
5. **Database Query Testing**: Ensure all SELECT queries handle new fields correctly
6. **TUI Display Testing**: Verify scrape status displays correctly in:
   - Link list view (status indicator visible)
   - Link details view (status field present)
   - All three status states (not scraped, success, failed) render with appropriate styling

## Future Enhancements (Out of Scope)

- Add `scraped_at` timestamp to track when scraping occurred
- Add `scrape_error_message` to store error details
- Add retry mechanism for failed scrapes
- Add filtering/querying by scrape status
- Add metrics/dashboard for scrape success rates

## Migration Strategy

1. Create migration file
2. Test migration on development database
3. Update models
4. Update database queries
5. Update service layer
6. Test end-to-end scraping flows
7. Deploy migration to production

## Estimated Effort

- Database migration: 30 minutes
- Model updates: 15 minutes
- Database query updates: 30 minutes
- Service layer updates: 45 minutes
- TUI display updates: 45 minutes
- Testing: 1 hour
- **Total: ~4 hours**
