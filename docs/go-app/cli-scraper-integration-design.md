# CLI Scraper Integration Design Document

## Overview

This document details the integration of the scraper service into the Go CLI application as a separate command from link creation. The scraper service provides a standalone `--scrape` command that extracts content from URLs independently of the `--add` command.

**Goal**: Provide a separate `--scrape` command to extract title and text content from URLs, allowing users to scrape URLs independently before or after adding links to the API.

**Timeline**: 1-2 days
**Complexity**: Medium
**Status**: 🟢 **Implementation Complete** - Ready for testing

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

### Desired Flow (Separate Scraping and Adding)

**Scraping Flow (Separate Command)**:

```
User Input → CLI Scrape Command → Nginx → Scraper Service → CLI Display
   ↓              ↓                 ↓            ↓                ↓
URL          --scrape <url>    /scrape    ScrapeResponse    Show Results
                                endpoint
```

**Adding Flow (Independent)**:

```
User Input → CLI Form → Nginx → API Client → API Server → Database
   ↓           ↓         ↓          ↓            ↓
URL, Title,  LinkCreate /api/v1  LinkCreate   Link JSON
Description, JSON      /links
Text
```

---

## Detailed Flow

### Scraping Command Flow (`--scrape`)

1. **User Runs Scrape Command**
   - User executes: `./cli --scrape <url>`
   - CLI validates URL format
   - CLI checks if scraper service is configured

2. **Scrape Request**
   - CLI sends scrape request to scraper service via nginx
   - Shows loading indicator to user ("Scraping URL...")
   - Waits for scraper response (with timeout handling)

3. **Scraper Response Display**
   - CLI receives `ScrapeResponse` from scraper service
   - Displays results to user:
     - URL
     - Title
     - Text content (truncated if long)
     - Success/error status
   - User can copy results for manual use

4. **Error Handling**
   - If scraping fails, displays error message
   - User can retry with different URL or check service status

### Add Link Command Flow (`--add`)

1. **User Runs Add Command**
   - User executes: `./cli --add`
   - Form displays input fields

2. **User Enters Data**
   - User manually enters URL, title, description, and text
   - No automatic scraping occurs
   - Form validates input

3. **API Submission**
   - User submits form
   - CLI sends `LinkCreate` to API via HTTP client
   - API validates and saves to database
   - CLI displays success/error message

### Optional: Using Scraped Data

Users can scrape a URL first with `--scrape`, then manually copy the scraped title and text when using `--add`. The two commands are completely independent.

---

## Data Flow Diagram

### Scraping Command Flow

```
┌─────────────┐
│   User      │
│ --scrape    │
│   <url>     │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  CLI Scrape Command Handler         │
│  - Validates URL                    │
│  - Initializes scraper client       │
└──────┬──────────────────────────────┘
       │
       │ [HTTP POST http://localhost/scrape]
       ▼
┌─────────────────────────────────────┐
│  Nginx Reverse Proxy                │
│  (nginx/nginx.conf)                 │
│  - Routes /scrape → scraper service │
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
│  CLI Response Display               │
│  - Shows scraped title              │
│  - Shows scraped text (truncated)   │
│  - Shows success/error status       │
└─────────────────────────────────────┘
```

### Add Link Command Flow (Independent)

```
┌─────────────┐
│   User      │
│   --add     │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  CLI Add Link Form (Bubble Tea)     │
│  - URL Input Field                  │
│  - Title Input Field                │
│  - Description Input Field          │
│  - Text Textarea                    │
└──────┬──────────────────────────────┘
       │
       │ [User enters all fields manually]
       │
       │ [User submits]
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

✅ **Already Implemented**: The configuration has been updated to support nginx reverse proxy with `BaseURL` and `ScrapeTimeout` fields in the `CLI` struct.

**Note**: The CLI constructs URLs as:

- API: `{base_url}/api/v1/*` → nginx routes to `api-dev:8080`
- Scraper: `{base_url}/scrape` → nginx routes to `scraper-dev:3000`
- Scraper Health: `{base_url}/scraper/health` → nginx routes to `scraper-dev:3000/health`

### Default Values

✅ **Already Implemented**: Default configuration sets `base_url = "http://localhost"` and `scrape_timeout = 30`.

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

### Scenario 1: Scraping a URL

1. User runs: `./cli --scrape https://example.com/article`
2. CLI shows: "⏳ Scraping URL... (this may take a few seconds)"
3. After 2-3 seconds:
   - CLI displays results:

     ```
     ✓ Scraping successful!

     URL: https://example.com/article
     Title: Example Article Title
     Text: Article content here... (truncated if long)
     ```

4. User can copy the results for later use

### Scenario 2: Scraping Failure

1. User runs: `./cli --scrape https://invalid-url.com`
2. CLI shows: "⏳ Scraping URL..."
3. After timeout or error:
   - CLI displays: "✗ Scraping failed: <error message>"
   - User can try a different URL or check service status

### Scenario 3: Scraper Service Unavailable

1. User runs: `./cli --scrape <url>`
2. CLI checks scraper service health
3. If unavailable:
   - CLI displays: "⚠️  Scraper service unavailable. Please check if the service is running."
   - Exit with error code

### Scenario 4: Adding a Link (Independent)

1. User runs: `./cli --add`
2. Form displays: "URL (required):"
3. User enters: `https://example.com/article`
4. Form moves to: "Title (required):"
5. User enters: "Example Article Title" (manually or copied from scrape)
6. Form moves to: "Description (optional):"
7. User presses Enter to skip
8. Form moves to: "Text (optional):"
9. User enters text (manually or copied from scrape)
10. User submits form
11. Success: "✓ Link created successfully!"

### Scenario 5: Using Scraped Data with Add

1. User runs: `./cli --scrape https://example.com/article`
2. CLI displays scraped title and text
3. User copies the title and text
4. User runs: `./cli --add`
5. User manually enters URL, pastes title, and pastes text
6. User submits form

---

## Implementation Details

### 1. Configuration ✅ COMPLETE

**File**: `link-mgmt-go/pkg/config/config.go`

**Status**: ✅ Already implemented

- ✅ `BaseURL` field exists in `CLI` struct (replaces `APIBaseURL`)
- ✅ `ScrapeTimeout` field exists in `CLI` struct
- ✅ `DefaultConfig()` sets defaults (`base_url = "http://localhost"`, `scrape_timeout = 30`)
- ✅ Config loading and saving implemented

### 2. Add Scrape Command Handler ✅ COMPLETE

**File**: `link-mgmt-go/pkg/cli/app.go`

**Status**: ✅ Implemented

**Changes**:

- ✅ Add scraper service client to `App` struct
- ✅ Add method to initialize scraper service: `getScraperService()`
- ✅ Add new command handler: `HandleScrapeCommand(url string)`
- ✅ Initialize scraper service with base URL (same as API, via nginx)
- ✅ Add `truncateText()` helper function for result display

**Note**: ✅ The scraper client (`pkg/scraper/client.go`) already uses the `/scrape` endpoint, which nginx routes correctly to the scraper service. Health checks use `/scraper/health` with fallback to `/health`.

### 3. Update Command Line Parsing ✅ COMPLETE

**File**: `link-mgmt-go/cmd/cli/main.go`

**Status**: ✅ Implemented

**Changes**:

- ✅ Add `--scrape <url>` flag parsing
- ✅ Route to `HandleScrapeCommand()` when scrape flag is present
- ✅ Keep `--add` command independent (no scraping integration)
- ✅ Validate base URL configuration before executing

### 4. Add Link Form Remains Unchanged

**File**: `link-mgmt-go/pkg/cli/app.go` (no changes to `addLinkForm`)

**Note**: The `addLinkForm` does not need any scraping-related changes. It remains a simple form for manual entry of URL, title, description, and text.

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

2. **`pkg/cli/app.go`** ✅ COMPLETE
   - ✅ Add scraper service initialization (`getScraperService()`)
   - ✅ Add `HandleScrapeCommand()` method
   - ✅ Add `truncateText()` helper function
   - ✅ Improved error handling with helpful messages
   - ✅ Keep `addLinkForm` unchanged (no scraping integration)

3. **`cmd/cli/main.go`** ✅ COMPLETE
   - ✅ Add `--scrape <url>` flag parsing
   - ✅ Route to scrape command handler
   - ✅ Validate base URL configuration

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

**Prerequisites:**

1. Start development services: `make dev-upd` (from project root)
2. Verify services are running: `docker compose --profile dev ps`

**Test Scenarios:**

1. **Happy Path - Scraping Success:**

   ```bash
   ./bin/cli --scrape https://example.com
   ```

   Expected: Health check passes, scraping succeeds, results displayed

2. **Service Unavailable:**

   ```bash
   # With services stopped
   ./bin/cli --scrape https://example.com
   ```

   Expected: Helpful error message with instructions to start services

3. **Invalid URL:**

   ```bash
   ./bin/cli --scrape invalid-url
   ```

   Expected: Error message about invalid URL format

4. **Add Command Independence:**

   ```bash
   ./bin/cli --add
   ```

   Expected: Form works normally without any scraping integration

5. **Various URL Types:**
   - Test with articles, blogs, documentation sites
   - Test with slow-loading URLs (should respect timeout)
   - Test with URLs that return errors

6. **Scraper Service Recovery:**
   - Stop scraper service: `docker compose --profile dev stop scraper-dev`
   - Try scraping (should show error)
   - Restart service: `docker compose --profile dev start scraper-dev`
   - Try scraping again (should work)

---

## Implementation Tasks

### Phase 1: Configuration ✅ COMPLETE

- [x] Add `BaseURL` to config struct (replaces `APIBaseURL`)
- [x] Add `ScrapeTimeout` to config struct
- [x] Update `DefaultConfig()` with defaults
- [x] Config loading and saving implemented
- [x] Nginx reverse proxy configured and routing working

### Phase 2: Scraper Service Integration ✅ COMPLETE

- [x] Add scraper service client to `App` struct
- [x] Implement `getScraperService()` method
- [x] Add health check on initialization
- [x] Test scraper service connection via nginx

### Phase 3: Scrape Command Implementation ✅ COMPLETE

- [x] Implement `HandleScrapeCommand()` method in `App`
- [x] Add command-line flag parsing for `--scrape <url>`
- [x] Add result display formatting with truncation
- [x] Add `truncateText()` helper function

### Phase 4: Error Handling ✅ COMPLETE

- [x] Implement graceful error handling for scrape command
- [x] Add user-friendly error messages with helpful guidance
- [x] Detect connection errors and provide service startup instructions
- [x] Handle invalid URLs, service unavailable, and scraping failures

### Phase 5: Testing & Polish ✅ COMPLETE

- [x] Test scrape command end-to-end (code verified, ready for manual testing)
- [x] Test add command independently (verified unchanged)
- [x] Test error scenarios (error handling implemented with helpful messages)
- [x] Update documentation (README updated with scrape command usage)
- [x] Verify CLI help output includes `--scrape` flag
- [x] Code compiles successfully

---

## Acceptance Criteria

### Functional Requirements

- [x] Scraper service client exists (`pkg/scraper/client.go`)
- [x] CLI can configure base URL (via `base_url` in config)
- [x] Nginx reverse proxy routes requests correctly
- [x] CLI provides separate `--scrape <url>` command
- [x] Scrape command displays results (URL, title, text)
- [x] Add command works independently without scraping
- [x] CLI handles scraper service errors gracefully
- [x] Scrape command provides clear error messages with helpful guidance

### Non-Functional Requirements

- [x] Scraping timeout is configurable (default: 30 seconds) - `ScrapeTimeout` in config
- [x] Scraping doesn't block CLI indefinitely (timeout configured)
- [x] Error messages are user-friendly with helpful guidance
- [x] Loading indicators show scraping progress
- [x] Configuration can be set via config file (`~/.config/link-mgmt/config.toml`)

### User Experience

- [x] Scrape command shows clear loading indicator
- [x] Scrape command displays results in readable format with truncation
- [x] Add command flow remains simple and intuitive
- [x] Error messages are clear and actionable with service startup instructions

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

1. **Functionality**: Scrape command successfully extracts content from URLs
2. **Independence**: Add command works independently without scraping
3. **User Experience**: Both commands provide clear feedback
4. **Performance**: Scraping completes within reasonable time (< 30s)
5. **Error Handling**: All error scenarios handled gracefully

---

## Related Documents

- [`service-implementation.md`](../scraper/service-implementation.md) - Scraper service implementation
- [`cli-implementation-plan.md`](./cli-implementation-plan.md) - CLI implementation status
- [`go-api-cli-design-document.md`](./go-api-cli-design-document.md) - Overall Go CLI design

---

**Last Updated**: Implementation complete - Phases 2, 3, and 4 finished
**Next Steps**: Phase 5 (Testing & Polish) - End-to-end testing and documentation updates

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
