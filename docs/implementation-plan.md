# Implementation Plan for Link Management CLI

## Overview

This document outlines the implementation plan for completing the Go CLI application, following the completion of Phase 1 (API CRUD operations).

## Current Status

✅ **Phase 1: COMPLETE**

- Full CRUD API (GET/POST/PUT/DELETE for links)
- Database layer with all queries
- Authentication middleware
- Configuration management
- CLI entry point structure

⚠️ **Phase 2: IN PROGRESS (~60% Complete)**

- ✅ HTTP client package created
- ✅ Removed direct DB connection from CLI
- ✅ User registration command (`--register`) with auto API key saving
- ✅ `--list` command implemented and working
- ✅ Migration commands added to Makefile
- ⚠️ `--add` command - pending
- ⚠️ `--delete` command - pending

❌ **Phase 3: NOT STARTED**

- Interactive TUI with Bubble Tea
- List view, form view, navigation

## Architecture Decision

**CLI should use HTTP client to call the API, not direct database access.**

**Rationale:**

- Config already has `CLI.APIBaseURL` and `CLI.APIKey` (designed for HTTP)
- Separation of concerns: CLI as API client
- Allows CLI to work remotely without DB credentials
- Consistent with other CLI implementations (Rust CLI, scraper)

**Status:**

- ✅ Removed `ConnectDB()` method - CLI now uses HTTP client only
- ✅ HTTP client implementation complete

## Phase 2: CLI Basic Commands

### 2.1 Create HTTP API Client Package

**Location**: `pkg/cli/client/`

**Structure**:

```
pkg/cli/client/
├── client.go      # HTTP client setup and base methods
├── links.go       # Link-related API calls
└── users.go       # User-related API calls (optional, for user management)
```

**Implementation Details**:

1. **Client Setup** (`client.go`):
   - Create `Client` struct with base URL and API key
   - Constructor: `NewClient(baseURL, apiKey string) *Client`
   - Helper method: `buildRequest(method, path string) *http.Request`
   - Handle Authorization header: `Authorization: Bearer <api_key>`
   - Error handling for HTTP errors

2. **Link Methods** (`links.go`):
   - `ListLinks() ([]models.Link, error)`
   - `GetLink(id uuid.UUID) (*models.Link, error)`
   - `CreateLink(link models.LinkCreate) (*models.Link, error)`
   - `UpdateLink(id uuid.UUID, update models.LinkUpdate) (*models.Link, error)`
   - `DeleteLink(id uuid.UUID) error`

**Dependencies**:

- `net/http` - HTTP client
- `encoding/json` - JSON encoding/decoding
- `link-mgmt-go/pkg/models` - Shared models

### 2.2 Implement `--list` Command

**Location**: `pkg/cli/app.go`

**Functionality**:

- Create HTTP client using config (`CLI.APIBaseURL`, `CLI.APIKey`)
- Call `client.ListLinks()`
- Display links in readable format (table or list)
- Handle errors gracefully

**Output Format**:

```
ID                                    URL                              Title                Created
───────────────────────────────────── ──────────────────────────────── ──────────────────── ──────────────
550e8400-e29b-41d4-a716-446655440000 https://example.com               Example Site         2024-01-15
...
```

**Error Handling**:

- API not reachable
- Invalid API key
- Empty list
- JSON parsing errors

### 2.3 Implement `--add` Command

**Location**: `pkg/cli/app.go`

**Functionality**:

- Interactive prompts for:
    - URL (required)
    - Title (optional)
    - Description (optional)
    - Text (optional)
- Create `LinkCreate` struct
- Call `client.CreateLink()`
- Display success message with created link details

**Input Methods** (choose one):

1. **Command-line flags**: `--add --url <url> --title <title> --desc <desc>`
2. **Interactive prompts**: Prompt for each field
3. **Both**: Support flags, prompt for missing required fields

**Recommendation**: Start with interactive prompts, add flags later.

**Dependencies**:

- `bufio` or `github.com/AlecAivazis/survey/v2` for prompts

### 2.4 Implement `--delete` Command

**Location**: `pkg/cli/app.go`

**Functionality**:

- Fetch list of links
- Display numbered list for user selection
- Prompt for confirmation
- Call `client.DeleteLink(id)`
- Display success message

**User Experience**:

```
$ link-mgmt-cli --delete
Select a link to delete:
1. Example Site (https://example.com)
2. Another Link (https://example2.com)
3. Cancel

Enter number: 1

Are you sure you want to delete "Example Site"? (y/N): y
✓ Link deleted successfully
```

**Error Handling**:

- No links available
- Invalid selection
- Delete confirmation cancelled
- API errors

## Phase 3: Interactive TUI with Bubble Tea

### 3.1 List View Model

**Location**: `pkg/cli/models/list.go`

**Features**:

- Display list of links in scrollable view
- Filter/search functionality
- Keyboard navigation (arrow keys, vim keys)
- Status bar showing count

**Key Bindings**:

- `q` / `Ctrl+C` - Quit
- `a` - Add link (transition to form)
- `d` - Delete selected link
- `e` - Edit selected link (future)
- `r` - Refresh list
- `/` - Search/filter

**State Management**:

- Load links on init
- Refresh on demand
- Handle loading/error states

### 3.2 Form View Model

**Location**: `pkg/cli/models/form.go`

**Features**:

- Multi-field form for creating/editing links
- Field validation
- Save/Cancel actions
- Navigate between fields with Tab

**Fields**:

- URL (required, validate format)
- Title (optional)
- Description (optional)
- Text (optional, multi-line)

**Key Bindings**:

- `Esc` / `Ctrl+C` - Cancel
- `Ctrl+S` / `Enter` (on last field) - Save
- `Tab` / `Shift+Tab` - Navigate fields

**Dependencies**:

- `github.com/charmbracelet/bubbles/textinput`
- `github.com/charmbracelet/bubbles/textarea`

### 3.3 TUI Integration

**Location**: `pkg/cli/app.go`

**Updates to `Run()` method**:

- Initialize ListModel
- Create Bubble Tea program
- Handle navigation between views
- Pass HTTP client to models

**Model Navigation**:

- Start with ListModel
- Transition to FormModel on "Add"
- Return to ListModel after save/cancel
- Refresh list after operations

## Phase 4: Advanced Features (Future)

- Search/filter functionality
- Tags/categories
- Export to JSON/CSV
- Bulk operations
- Link previews

## Implementation Order

### Immediate (Phase 2)

1. **Step 1**: Create HTTP client package ✅ **COMPLETE**
   - [x] `pkg/cli/client/client.go` - Base client
   - [x] `pkg/cli/client/links.go` - Link methods
   - [x] `pkg/cli/client/users.go` - User methods (for registration)
   - [ ] Tests for HTTP client

2. **Step 2**: Update CLI app to use HTTP client ✅ **COMPLETE**
   - [x] Removed `ConnectDB()` method
   - [x] Added `getClient()` method using config
   - [x] Added `getClientForRegistration()` for unauthenticated requests
   - [x] Updated `ListLinks()` to use HTTP client

3. **Step 3**: Implement `--list` command ✅ **COMPLETE**
   - [x] Format output nicely (table format with truncated IDs/URLs)
   - [x] Handle errors gracefully
   - [x] Tested with real API

4. **Step 3.5**: User Registration ✅ **COMPLETE** (Bonus feature)
   - [x] Added `--register <email>` command
   - [x] Automatically saves API key to config
   - [x] Improved error messages for missing database tables
   - [x] Added migration commands to Makefile

5. **Step 4**: Implement `--add` command ⚠️ **PENDING**

   - [ ] Add prompt library dependency
   - [ ] Interactive prompts
   - [ ] Validate input
   - [ ] Call API

6. **Step 5**: Implement `--delete` command ⚠️ **PENDING**
   - [ ] Fetch and display links
   - [ ] Selection prompt
   - [ ] Confirmation
   - [ ] Delete via API

### Future (Phase 3)

6. **Step 6**: TUI ListModel
7. **Step 7**: TUI FormModel
8. **Step 8**: TUI integration

## Dependencies to Add

```go
// For HTTP client
- net/http (stdlib)
- encoding/json (stdlib)

// For interactive prompts (Phase 2.3, 2.4)
- github.com/AlecAivazis/survey/v2

// Already present for TUI
- github.com/charmbracelet/bubbletea
- github.com/charmbracelet/bubbles/list
- github.com/charmbracelet/bubbles/textinput
- github.com/charmbracelet/bubbles/textarea
```

## Testing Strategy

### Unit Tests

- HTTP client error handling
- JSON parsing
- Input validation

### Integration Tests

- Test CLI commands against running API
- Test error scenarios (API down, invalid key, etc.)

### Manual Testing

- Test interactive prompts
- Test TUI navigation (Phase 3)
- Test on different terminals

## File Changes Summary

### New Files ✅

- ✅ `pkg/cli/client/client.go` - HTTP client base
- ✅ `pkg/cli/client/links.go` - Link API methods
- ✅ `pkg/cli/client/users.go` - User API methods
- `pkg/cli/models/list.go` (Phase 3)
- `pkg/cli/models/form.go` (Phase 3)

### Modified Files ✅

- ✅ `pkg/cli/app.go` - Replaced DB calls with HTTP client calls, added registration
- ✅ `cmd/cli/main.go` - Added `--register` flag, improved error messages
- ✅ `Makefile` (root) - Added `migrate` command
- ✅ `link-mgmt-go/Makefile` - Added migration commands
- `go.mod` - Will need survey dependency for Phase 2.3

### Removed ✅

- ✅ `ConnectDB()` method in `pkg/cli/app.go` - Removed completely
- ✅ Database dependency from CLI - Now uses HTTP API only

## Success Criteria

### Phase 2 Complete When

- [x] `--list` command works and displays links nicely
- [x] All commands use HTTP client (not direct DB)
- [x] Error messages are user-friendly (with helpful migration instructions)
- [x] User registration workflow working
- [ ] `--add` command allows creating links via prompts
- [ ] `--delete` command allows deleting links with confirmation
- [ ] Basic tests pass

### Phase 3 Complete When

- [ ] Interactive TUI launches with `link-mgmt-cli` (no flags)
- [ ] Can navigate list of links
- [ ] Can add links via form
- [ ] Can delete links from list
- [ ] Smooth transitions between views

## Notes

- Start with Phase 2 (basic commands) before TUI
- HTTP client should be reusable for TUI
- Consider making TUI optional (use `--tui` flag) vs default
- Keep database connection code as fallback for admin/debug tools
