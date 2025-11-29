# CLI Scraper Integration Design Document

## Overview

This document details the integration of the scraper service into the Go CLI application, enabling automatic content extraction when adding links. The flow will: (1) send URLs to the scraper service, (2) process the scraper response in the CLI, and (3) post the enriched link data to the API.

**Goal**: Automatically extract title and text content from URLs using the scraper service before saving links to the API.

**Timeline**: 1-2 days
**Complexity**: Medium
**Status**: ❌ **Not Implemented** - Design phase

**Prerequisites**:

- ✅ Scraper HTTP service implemented (`scraper/src/server.ts`)
- ✅ Go scraper client implemented (`link-mgmt-go/pkg/scraper/client.go`)
- ✅ CLI add link form implemented (`link-mgmt-go/pkg/cli/app.go`)
- ✅ API client implemented (`link-mgmt-go/pkg/cli/client/`)

---

## Architecture Overview

### Current Flow (Manual Entry)

```
User Input → CLI Form → API Client → API Server → Database
   ↓           ↓
URL, Title,   LinkCreate
Description,  JSON
Text
```

### Desired Flow (With Scraping)

```
User Input → CLI Form → Scraper Service → CLI Processing → API Client → API Server → Database
   ↓           ↓              ↓                  ↓               ↓
URL only   Scrape Request   ScrapeResponse   Enrich Link     LinkCreate
                                                              JSON
```

---

## Detailed Flow

### Step-by-Step Process

1. **User Enters URL**
   - User provides URL in the add link form
   - CLI validates URL format
   - URL is stored temporarily in form state

2. **Scrape Request** (Automatic)
   - CLI checks if scraper service is configured
   - If configured, automatically sends scrape request to scraper service
   - Shows loading indicator to user ("Scraping URL...")
   - Waits for scraper response (with timeout handling)

3. **Scraper Response Handling**
   - CLI receives `ScrapeResponse` from scraper service
   - Processes response:
     - Maps `title` → `LinkCreate.Title`
     - Maps `text` → `LinkCreate.Text`
     - Preserves original `url`
     - Leaves `description` empty (can be manually added later)

4. **User Review/Edit** (Optional)
   - Pre-fills form fields with scraped data
   - User can review and edit title/text before submission
   - User can skip scraping if desired (flag or prompt)

5. **API Submission**
   - User submits form (or confirms pre-filled data)
   - CLI sends `LinkCreate` to API via HTTP client
   - API validates and saves to database
   - CLI displays success/error message

---

## Data Flow Diagram

```
┌─────────────┐
│   User      │
│  Enters URL │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  CLI Add Link Form (Bubble Tea)     │
│  - URL Input Field                  │
│  - Title Input Field (auto-filled)  │
│  - Description Input Field          │
│  - Text Textarea (auto-filled)      │
└──────┬──────────────────────────────┘
       │
       │ [User enters URL]
       ▼
┌─────────────────────────────────────┐
│  Scraper Service Client             │
│  (pkg/scraper/client.go)            │
│  - Scrape(url, timeout)             │
└──────┬──────────────────────────────┘
       │
       │ [HTTP POST /scrape]
       ▼
┌─────────────────────────────────────┐
│  Scraper Service                    │
│  (scraper/src/server.ts)            │
│  - BrowserManager.extractFromUrl()  │
│  - extractMainContent()             │
└──────┬──────────────────────────────┘
       │
       │ [ScrapeResponse JSON]
       │ { success, url, title, text }
       ▼
┌─────────────────────────────────────┐
│  CLI Response Processing            │
│  - Map title → LinkCreate.Title     │
│  - Map text → LinkCreate.Text       │
│  - Handle errors gracefully         │
└──────┬──────────────────────────────┘
       │
       │ [Auto-fill form fields]
       ▼
┌─────────────────────────────────────┐
│  User Reviews/Edits (Optional)      │
│  - Can modify title                 │
│  - Can modify text                  │
│  - Can add description              │
└──────┬──────────────────────────────┘
       │
       │ [User confirms]
       ▼
┌─────────────────────────────────────┐
│  API Client                         │
│  (pkg/cli/client/links.go)          │
│  - CreateLink(LinkCreate)           │
└──────┬──────────────────────────────┘
       │
       │ [HTTP POST /api/v1/links]
       ▼
┌─────────────────────────────────────┐
│  API Server                         │
│  - Validates request                │
│  - Saves to database                │
└──────┬──────────────────────────────┘
       │
       │ [Link JSON response]
       ▼
┌─────────────────────────────────────┐
│  CLI Success Display                │
│  - Shows created link details       │
└─────────────────────────────────────┘
```

---

## Configuration Changes

### Current Config Structure

```go
// pkg/config/config.go
type Config struct {
    CLI struct {
        APIBaseURL string `toml:"api_base_url"`
        APIKey     string `toml:"api_key"`
    } `toml:"cli"`
}
```

### Required Changes

With nginx reverse proxy, we can simplify configuration to use a single base URL:

```go
// pkg/config/config.go
type Config struct {
    CLI struct {
        BaseURL     string `toml:"base_url"`      // Single base URL for all services
        APIKey      string `toml:"api_key"`
        ScrapeTimeout int  `toml:"scrape_timeout"` // NEW (seconds, default: 30)
    } `toml:"cli"`
}
```

**Note**: The CLI will construct URLs as:

- API: `{base_url}/api/v1/*`
- Scraper: `{base_url}/scraper/*` or `{base_url}/scrape`

### Default Values

```go
func DefaultConfig() *Config {
    cfg := &Config{}
    // ... existing defaults ...
    cfg.CLI.BaseURL = "http://localhost"  // nginx reverse proxy on port 80
    cfg.CLI.ScrapeTimeout = 30  // 30 seconds default
    return cfg
}
```

### Config File Example

```toml
[cli]
base_url = "http://localhost"
api_key = "your-api-key-here"
scrape_timeout = 30
```

### Migration from Old Config

If you have an existing config with `api_base_url` and `scraper_service_url`, the CLI can:

1. Read the old config format
2. Automatically migrate to the new `base_url` format
3. Or maintain backward compatibility by deriving `base_url` from `api_base_url` if `scraper_service_url` matches the same host

---

## User Experience Flow

### Scenario 1: Successful Scraping

1. User runs: `./cli --add`
2. Form displays: "URL (required):"
3. User enters: `https://example.com/article`
4. User presses Enter
5. Form shows: "⏳ Scraping URL... (this may take a few seconds)"
6. After 2-3 seconds:
   - Title field auto-fills: "Example Article Title"
   - Text field auto-fills: "Article content here..."
   - Cursor moves to Title field
   - Form shows: "✓ Scraped successfully"
7. User can:
   - Press Enter to skip Title (keep scraped value)
   - Edit Title if needed
   - Press Enter to continue
8. Form moves to Description (optional)
9. User presses Enter to skip
10. Form shows Text field (pre-filled)
11. User presses Enter to submit
12. Success: "✓ Link created successfully!"

### Scenario 2: Scraping Failure

1. User enters URL and presses Enter
2. Form shows: "⏳ Scraping URL..."
3. After timeout or error:
   - Form shows: "⚠️  Could not scrape URL. Continue manually? (y/n)"
   - Title field remains empty
   - Text field remains empty
   - User can proceed to fill manually or retry

### Scenario 3: Scraper Service Unavailable

1. CLI checks scraper service health on startup (or first use)
2. If unavailable:
   - Logs warning: "Scraper service not available. Scraping disabled."
   - Form works normally but without auto-scraping
   - User manually enters all fields

### Scenario 4: User Skips Scraping

1. User enters URL
2. Before scraping starts, user presses a key (e.g., `s` for "skip")
3. Or user explicitly disables scraping via flag: `--add --no-scrape`
4. Form proceeds with manual entry only

---

## Implementation Details

### 1. Update Configuration

**File**: `link-mgmt-go/pkg/config/config.go`

**Changes**:

- Replace `APIBaseURL` with `BaseURL` (single URL for all services via nginx)
- Add `ScrapeTimeout` field to `CLI` struct
- Update `DefaultConfig()` with defaults (base_url = "<http://localhost>")
- Update `SetConfig()` to handle new keys (e.g., `cli.base_url`)
- Maintain backward compatibility: if `api_base_url` exists, derive `base_url` from it

### 2. Update CLI App Structure

**File**: `link-mgmt-go/pkg/cli/app.go`

**Changes**:

- Add scraper service client to `App` struct
- Add method to initialize scraper service: `getScraperService()`
- Pass scraper service to add link form
- Initialize scraper service with base URL (same as API, via nginx)

```go
type App struct {
    cfg            *config.Config
    client         *client.Client
    scraperService *scraper.ScraperService  // NEW
}

func (a *App) getScraperService() *scraper.ScraperService {
    if a.scraperService == nil {
        // Use same base URL as API (nginx routes /scraper/* to scraper service)
        baseURL := a.cfg.CLI.BaseURL
        a.scraperService = scraper.NewScraperService(baseURL)
    }
    return a.scraperService
}
```

**Note**: The scraper client will need to be updated to use `/scraper/` prefix for paths when using nginx, or nginx can be configured to strip the prefix. For simplicity, we'll use `/scrape` endpoint directly (nginx routes this to scraper service).

### 3. Enhance Add Link Form

**File**: `link-mgmt-go/pkg/cli/app.go` (update `addLinkForm`)

**Changes**:

- Add scraper service to form struct
- Add scraping state: `scraping`, `scraped`, `scrapeError`
- Add scraping step after URL entry
- Auto-fill title and text from scrape response
- Add loading indicator UI

```go
type addLinkForm struct {
    client         *client.Client
    scraperService *scraper.ScraperService  // NEW
    urlInput       textinput.Model
    titleInput     textinput.Model
    descInput      textinput.Model
    textInput      textarea.Model
    step           int  // 0=URL, 1=Scraping, 2=Title, 3=Description, 4=Text, 5=Done
    scraping       bool // NEW: indicates scraping in progress
    scraped        bool // NEW: indicates scraping completed
    scrapeError    error // NEW: stores scraping error if any
    scrapedData    *scraper.ScrapeResponse // NEW: stores scrape result
    err            error
    created        *models.Link
}
```

### 4. Scraping Logic

**New Method in `addLinkForm`**:

```go
func (m *addLinkForm) scrapeURL(url string) tea.Cmd {
    return func() tea.Msg {
        if m.scraperService == nil {
            return scrapeErrorMsg{err: fmt.Errorf("scraper service not configured")}
        }

        // Check health first (optional, but good for UX)
        if err := m.scraperService.CheckHealth(); err != nil {
            return scrapeErrorMsg{err: fmt.Errorf("scraper service unavailable: %w", err)}
        }

        // Scrape the URL
        timeout := 30 // Could come from config
        result, err := m.scraperService.Scrape(url, timeout*1000) // timeout in ms
        if err != nil {
            return scrapeErrorMsg{err: err}
        }

        if !result.Success {
            return scrapeErrorMsg{err: fmt.Errorf("scraping failed: %s", result.Error)}
        }

        return scrapeSuccessMsg{result: result}
    }
}
```

### 5. Message Types

**New message types for scraping state**:

```go
type scrapeStartMsg struct{}
type scrapeSuccessMsg struct {
    result *scraper.ScrapeResponse
}
type scrapeErrorMsg struct {
    err error
}
```

### 6. Form Update Logic

**Update `Update()` method**:

- Handle `scrapeStartMsg` → set `scraping = true`, trigger scrape command
- Handle `scrapeSuccessMsg` → set `scraping = false`, `scraped = true`, auto-fill fields, move to next step
- Handle `scrapeErrorMsg` → set `scraping = false`, show error, allow manual entry

### 7. Form View Updates

**Update `View()` method**:

- Show "⏳ Scraping URL..." when `scraping = true`
- Show "✓ Scraped successfully" when `scraped = true`
- Show error message when `scrapeError != nil`
- Pre-fill title and text fields with scraped data

---

## Error Handling Strategy

### Scraper Service Errors

1. **Service Unavailable** (connection refused, timeout):
   - Log warning
   - Allow manual entry
   - Don't block link creation

2. **Scraping Failed** (invalid URL, timeout, extraction error):
   - Show error message to user
   - Allow retry or manual entry
   - Don't block link creation

3. **Invalid Response** (malformed JSON, unexpected structure):
   - Log error
   - Fall back to manual entry
   - Don't block link creation

### Error Messages

- Service unavailable: `"⚠️  Scraper service unavailable. Please fill fields manually."`
- Scraping failed: `"⚠️  Could not scrape URL: <error>. Continue manually? (press Enter)"`
- Timeout: `"⏱️  Scraping timed out. Continue manually? (press Enter)"`

### Graceful Degradation

- If scraper service is not configured, skip scraping entirely
- If scraping fails, allow manual entry
- Never block link creation due to scraping issues

---

## Code Structure

### File Changes Summary

1. **`pkg/config/config.go`**
   - Add `ScraperServiceURL` and `ScrapeTimeout` to config struct
   - Update defaults and config parsing

2. **`pkg/cli/app.go`**
   - Add scraper service initialization
   - Pass scraper service to add link form
   - Update `addLinkForm` struct and methods
   - Add scraping state management

3. **No changes needed to**:
   - `pkg/scraper/client.go` (already implemented)
   - `pkg/cli/client/` (API client already implemented)

---

## Testing Strategy

### Unit Tests

- Test scraper service initialization
- Test scraping error handling
- Test form state transitions

### Integration Tests

1. **Happy Path**:
   - Configure scraper service URL
   - Run CLI add command
   - Verify scraping happens
   - Verify form auto-fills
   - Verify link created in API

2. **Error Paths**:
   - Scraper service unavailable
   - Scraping fails
   - Invalid URL
   - Timeout

3. **Configuration**:
   - Missing scraper service URL (should skip scraping)
   - Invalid scraper service URL
   - Config override via command line

### Manual Testing

- Test with various URL types (articles, blogs, documentation)
- Test with slow-loading URLs
- Test with invalid URLs
- Test scraper service restart/recovery

---

## Implementation Tasks

### Phase 1: Configuration

- [ ] Add `ScraperServiceURL` to config struct
- [ ] Add `ScrapeTimeout` to config struct
- [ ] Update `DefaultConfig()` with defaults
- [ ] Update `SetConfig()` to handle new config keys
- [ ] Test config loading and saving

### Phase 2: Scraper Service Integration

- [ ] Add scraper service client to `App` struct
- [ ] Implement `getScraperService()` method
- [ ] Add health check on initialization (optional)
- [ ] Test scraper service connection

### Phase 3: Form Enhancement

- [ ] Update `addLinkForm` struct with scraping fields
- [ ] Add scraping message types (`scrapeStartMsg`, `scrapeSuccessMsg`, `scrapeErrorMsg`)
- [ ] Implement `scrapeURL()` method
- [ ] Update `Update()` method to handle scraping messages
- [ ] Update `View()` method to show scraping state
- [ ] Auto-fill form fields from scrape response

### Phase 4: Error Handling

- [ ] Implement graceful error handling
- [ ] Add user-friendly error messages
- [ ] Implement fallback to manual entry
- [ ] Test error scenarios

### Phase 5: Testing & Polish

- [ ] Test full flow end-to-end
- [ ] Test error scenarios
- [ ] Update documentation
- [ ] Add CLI flag to disable scraping (`--no-scrape`)

---

## Acceptance Criteria

### Functional Requirements

- [x] Scraper service client exists (`pkg/scraper/client.go`)
- [ ] CLI can configure scraper service URL
- [ ] CLI automatically scrapes URLs when adding links
- [ ] Form auto-fills title and text from scraped data
- [ ] User can edit scraped data before submission
- [ ] CLI handles scraper service errors gracefully
- [ ] Link creation works even if scraping fails
- [ ] CLI can skip scraping (via config or flag)

### Non-Functional Requirements

- [ ] Scraping timeout is configurable (default: 30 seconds)
- [ ] Scraping doesn't block CLI indefinitely
- [ ] Error messages are user-friendly
- [ ] Loading indicators show scraping progress
- [ ] Configuration can be set via `--config-set`

### User Experience

- [ ] User sees clear indication when scraping is happening
- [ ] User can proceed manually if scraping fails
- [ ] Scraped data is clearly distinguished from user input
- [ ] Form flow is smooth and intuitive

---

## Future Enhancements

### Phase 2 Features (Not in Initial Implementation)

1. **Batch Scraping**: Scrape multiple URLs at once
2. **Scrape Cache**: Cache scraped content to avoid re-scraping
3. **Scrape Preview**: Show preview of scraped content before auto-filling
4. **Custom Scrape Timeouts**: Per-URL timeout configuration
5. **Scrape Retry**: Automatic retry on failure
6. **Background Scraping**: Scrape in background while user edits

### Configuration Enhancements

- Environment variable support: `LINK_MGMT_SCRAPER_SERVICE_URL`
- Scrape timeout per URL type
- Disable scraping globally via config

---

## Dependencies

### Existing Dependencies

- ✅ `link-mgmt-go/pkg/scraper/client.go` - Scraper HTTP client
- ✅ `link-mgmt-go/pkg/cli/client/` - API HTTP client
- ✅ Bubble Tea - TUI framework (already in use)

### No New Dependencies Required

All necessary packages are already available in the codebase.

---

## Success Metrics

1. **Functionality**: Links can be created with auto-scraped content
2. **Reliability**: Scraping failures don't block link creation
3. **User Experience**: Smooth flow with clear feedback
4. **Performance**: Scraping completes within reasonable time (< 30s)
5. **Error Handling**: All error scenarios handled gracefully

---

## Related Documents

- [`service-implementation.md`](../scraper/service-implementation.md) - Scraper service implementation
- [`cli-implementation-plan.md`](./cli-implementation-plan.md) - CLI implementation status
- [`go-api-cli-design-document.md`](./go-api-cli-design-document.md) - Overall Go CLI design

---

**Last Updated**: Design phase
**Next Steps**: Implementation Phase 1 (Configuration)
