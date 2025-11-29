# CLI Scraper Integration Design Document

## Overview

This document details the integration of the scraper service into the Go CLI application, enabling automatic content extraction when adding links. The flow will: (1) send URLs to the scraper service, (2) process the scraper response in the CLI, and (3) post the enriched link data to the API.

**Goal**: Automatically extract title and text content from URLs using the scraper service before saving links to the API.

**Timeline**: 1-2 days
**Complexity**: Medium
**Status**: 🟡 **In Progress** - Configuration complete, form integration pending

**Prerequisites**:

- ✅ Scraper HTTP service implemented (`scraper/src/server.ts`)
- ✅ Go scraper client implemented (`link-mgmt-go/pkg/scraper/client.go`)
- ✅ CLI add link form implemented (`link-mgmt-go/pkg/cli/app.go`)
- ✅ API client implemented (`link-mgmt-go/pkg/cli/client/`)
- ✅ Nginx reverse proxy configured (`nginx/nginx.conf`, `docker-compose.yml`)
- ✅ Configuration updated with `BaseURL` and `ScrapeTimeout` (`pkg/config/config.go`)

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
User Input → CLI Form → Nginx → Scraper Service → CLI Processing → Nginx → API Client → API Server → Database
   ↓           ↓         ↓            ↓                  ↓            ↓          ↓
URL only   Scrape    /scrape      ScrapeResponse   Enrich Link   /api/v1   LinkCreate
           Request   endpoint                                    /links     JSON
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
       │ [HTTP POST http://localhost/scrape]
       ▼
┌─────────────────────────────────────┐
│  Nginx Reverse Proxy                │
│  (nginx/nginx.conf)                 │
│  - Routes /scrape → scraper service │
│  - Routes /scraper/* → scraper      │
└──────┬──────────────────────────────┘
       │
       │ [HTTP POST http://scraper-dev:3000/scrape]
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
       │ [HTTP POST http://localhost/api/v1/links]
       ▼
┌─────────────────────────────────────┐
│  Nginx Reverse Proxy                │
│  (nginx/nginx.conf)                 │
│  - Routes /api/* → API service      │
└──────┬──────────────────────────────┘
       │
       │ [HTTP POST http://api-dev:8080/api/v1/links]
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

## Configuration

### Current Config Structure

✅ **Already Implemented**: The configuration has been updated to support nginx reverse proxy:

```go
// pkg/config/config.go
type Config struct {
    CLI struct {
        BaseURL       string `toml:"base_url"`       // Single base URL for all services (via nginx)
        APIKey        string `toml:"api_key"`
        ScrapeTimeout int    `toml:"scrape_timeout"` // Timeout for scraping operations in seconds
    } `toml:"cli"`
}
```

**Note**: The CLI constructs URLs as:

- API: `{base_url}/api/v1/*` → nginx routes to `api-dev:8080`
- Scraper: `{base_url}/scrape` → nginx routes to `scraper-dev:3000`
- Scraper Health: `{base_url}/scraper/health` → nginx routes to `scraper-dev:3000/health`

### Default Values

✅ **Already Implemented**: Default configuration:

```go
func DefaultConfig() *Config {
    cfg := &Config{}
    cfg.CLI.BaseURL = "http://localhost"  // nginx reverse proxy on port 80
    cfg.CLI.ScrapeTimeout = 30            // 30 seconds default
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

### Nginx Routing Structure

The nginx reverse proxy (`nginx/nginx.conf`) routes requests as follows:

- **API Routes**:
    - `/api/*` → `http://api-dev:8080/api/*` (or `http://api:8080` in production)
    - `/health` → `http://api-dev:8080/health`

- **Scraper Routes**:
    - `/scrape` → `http://scraper-dev:3000/scrape` (direct mapping)
    - `/scraper/*` → `http://scraper-dev:3000/*` (prefix stripped via rewrite)
    - `/scraper/health` → `http://scraper-dev:3000/health`

All services are accessed through nginx on port 80, which simplifies CLI configuration to a single `base_url`.

### Docker Compose Setup

The `docker-compose.yml` orchestrates all services with the following structure:

**Services (dev profile)**:

- **nginx**: Reverse proxy container (port 80 exposed to host)
- **api-dev**: Go API service (port 8080, internal only)
- **scraper-dev**: Bun scraper service (port 3000, internal only)
- **postgres**: PostgreSQL database (port 5432 exposed to host)

**Key Features**:

- All services communicate via Docker networking (service names as hostnames)
- Nginx depends on `api-dev` and `scraper-dev` services
- Health checks configured for all services
- Volume mounts for hot-reloading in development
- Services use `profiles: [dev]` for development environment

**CLI Access**:

- CLI runs on host machine (not in Docker)
- Accesses services via `http://localhost` (nginx on port 80)
- No need to know internal service ports or hostnames

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

### 1. Configuration ✅ COMPLETE

**File**: `link-mgmt-go/pkg/config/config.go`

**Status**: ✅ Already implemented

- ✅ `BaseURL` field exists in `CLI` struct (replaces `APIBaseURL`)
- ✅ `ScrapeTimeout` field exists in `CLI` struct
- ✅ `DefaultConfig()` sets defaults (`base_url = "http://localhost"`, `scrape_timeout = 30`)
- ✅ Config loading and saving implemented

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

**Note**: ✅ The scraper client (`pkg/scraper/client.go`) already uses the `/scrape` endpoint, which nginx routes correctly to the scraper service. Health checks use `/scraper/health` with fallback to `/health`.

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
        // Use timeout from config (already available via cfg.CLI.ScrapeTimeout)
        timeout := m.cfg.CLI.ScrapeTimeout
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

**Note**: The scraper client (`pkg/scraper/client.go`) is already implemented and:

- ✅ Uses `/scrape` endpoint (routed via nginx)
- ✅ Has `CheckHealth()` method using `/scraper/health` endpoint
- ✅ Returns `ScrapeResponse` with `Success`, `Title`, `Text`, `URL`, `Error` fields

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

1. **`pkg/config/config.go`** ✅ COMPLETE
   - ✅ `BaseURL` and `ScrapeTimeout` already in config struct
   - ✅ Defaults and config parsing already implemented

2. **`pkg/cli/app.go`** ⏳ TODO
   - ⏳ Add scraper service initialization
   - ⏳ Pass scraper service to add link form
   - ⏳ Update `addLinkForm` struct and methods
   - ⏳ Add scraping state management

3. **Already implemented**:
   - ✅ `pkg/scraper/client.go` - Scraper HTTP client with nginx routing support
   - ✅ `pkg/cli/client/` - API HTTP client
   - ✅ `nginx/nginx.conf` - Reverse proxy routing configuration
   - ✅ `docker-compose.yml` - Service orchestration with nginx

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

### Phase 1: Configuration ✅ COMPLETE

- [x] Add `BaseURL` to config struct (replaces `APIBaseURL`)
- [x] Add `ScrapeTimeout` to config struct
- [x] Update `DefaultConfig()` with defaults
- [x] Config loading and saving implemented
- [x] Nginx reverse proxy configured and routing working

### Phase 2: Scraper Service Integration ⏳ TODO

- [ ] Add scraper service client to `App` struct
- [ ] Implement `getScraperService()` method
- [ ] Add health check on initialization (optional)
- [ ] Test scraper service connection via nginx

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
- [x] CLI can configure base URL (via `base_url` in config)
- [x] Nginx reverse proxy routes requests correctly
- [ ] CLI automatically scrapes URLs when adding links
- [ ] Form auto-fills title and text from scraped data
- [ ] User can edit scraped data before submission
- [ ] CLI handles scraper service errors gracefully
- [ ] Link creation works even if scraping fails
- [ ] CLI can skip scraping (via config or flag)

### Non-Functional Requirements

- [x] Scraping timeout is configurable (default: 30 seconds) - `ScrapeTimeout` in config
- [ ] Scraping doesn't block CLI indefinitely
- [ ] Error messages are user-friendly
- [ ] Loading indicators show scraping progress
- [x] Configuration can be set via config file (`~/.config/link-mgmt/config.toml`)

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

- ✅ `link-mgmt-go/pkg/scraper/client.go` - Scraper HTTP client (uses nginx routing)
- ✅ `link-mgmt-go/pkg/cli/client/` - API HTTP client (uses nginx routing)
- ✅ `nginx/nginx.conf` - Reverse proxy configuration
- ✅ `docker-compose.yml` - Service orchestration with nginx
- ✅ Bubble Tea - TUI framework (already in use)

### Infrastructure

- ✅ **Nginx Reverse Proxy**: Routes all requests through port 80
    - `/api/*` → API service (api-dev:8080)
    - `/scrape` → Scraper service (scraper-dev:3000)
    - `/scraper/*` → Scraper service (scraper-dev:3000)
- ✅ **Docker Compose**: Orchestrates all services with proper networking
- ✅ **Health Checks**: Configured for all services

### No New Dependencies Required

All necessary packages and infrastructure are already available in the codebase.

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

**Last Updated**: Updated to reflect nginx reverse proxy architecture
**Next Steps**: Implementation Phase 2 (Scraper Service Integration) and Phase 3 (Form Enhancement)

## Architecture Notes

### Nginx Reverse Proxy Setup

The system uses nginx as a reverse proxy to simplify service access:

- **Single Entry Point**: All services accessed via `http://localhost` (port 80)
- **Service Routing**:
    - API: `http://localhost/api/v1/*` → `api-dev:8080`
    - Scraper: `http://localhost/scrape` → `scraper-dev:3000`
    - Scraper (with prefix): `http://localhost/scraper/*` → `scraper-dev:3000/*`
- **Health Checks**:
    - `/health` → API health
    - `/scraper/health` → Scraper health
- **Timeouts**: Extended timeouts for scraping operations (120s) to handle long-running requests

### Docker Compose Services

- **nginx**: Reverse proxy (port 80)
- **api-dev**: Go API service (port 8080, exposed internally)
- **scraper-dev**: Bun scraper service (port 3000, exposed internally)
- **postgres**: Database (port 5432)

All services run in the `dev` profile and communicate via Docker networking.
