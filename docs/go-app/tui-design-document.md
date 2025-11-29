# TUI Design Document - Interactive Link Addition with Scraping

## Overview

This document outlines the design for an interactive Terminal User Interface (TUI) that integrates URL scraping into the link addition workflow. The TUI provides a seamless flow: **Add → Scrape → Confirm/Edit → Save**.

**Goal**: Create an intuitive, interactive TUI that automatically scrapes URLs when adding links, allows users to review and edit scraped content, and saves the final result.

**Timeline**: 2-3 days
**Complexity**: Medium-High
**Status**: 🟢 **Phase 2 implemented (initial TUI integration complete); ready for UX polish & testing**

**Prerequisites**:

- ✅ Bubble Tea TUI framework (already in dependencies)
- ✅ Scraper service client implemented (`pkg/scraper/client.go`)
- ✅ API client implemented (`pkg/cli/client/`)
- ✅ Basic add link form exists (`pkg/cli/app.go` - `addLinkForm`)
- ✅ Configuration with `BaseURL` and `ScrapeTimeout`

---

## User Experience Flow

### Primary Flow: Add Link with Auto-Scraping

```
┌─────────────────────────────────────────────────────────────┐
│ Step 1: URL Input                                           │
│ ─────────────────────────────────────────────────────────── │
│ URL (required):                                             │
│ [https://example.com/article                    ]           │
│                                                             │
│ Press Enter to scrape, Esc to cancel                        │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: Scraping (Loading State)                            │
│ ─────────────────────────────────────────────────────────── │
│ ⏳ Scraping URL... (this may take a few seconds)            │
│                                                             │
│ Checking scraper service... ✓                               │
│ Scraping content...                                         │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Review & Edit                                       │
│ ─────────────────────────────────────────────────────────── │
│ ✓ URL: https://example.com/article                          │
│                                                             │
│ Title: [Example Article Title                    ]          │
│   (scraped)                                                 │
│                                                             │
│ Description (optional): [                          ]        │
│                                                             │
│ Text (optional):                                            │
│ ┌─────────────────────────────────────────────────────┐     │
│ │ Article content here...                             │     │
│ │ (scraped, truncated for display)                    │     │
│ │                                                     │     │
│ └─────────────────────────────────────────────────────┘     │
│                                                             │
│ [Tab] Navigate  [Enter] Save  [Esc] Cancel  [s] Skip scrape │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 4: Success                                             │
│ ─────────────────────────────────────────────────────────── │
│ ✓ Link created successfully!                                │
│                                                             │
│   ID:          a1b2c3d4...                                  │
│   URL:         https://example.com/article                  │
│   Title:       Example Article Title                        │
│   Created:     2024-01-15 14:30                             │
│                                                             │
│ Press any key to exit...                                    │
└─────────────────────────────────────────────────────────────┘
```

### Alternative Flows

#### Flow 2: Manual Entry (Skip Scraping)

1. User enters URL
2. User presses `s` to skip scraping
3. Form proceeds directly to manual entry (Title, Description, Text)
4. User fills fields manually
5. Save

#### Flow 3: Scraping Fails (Graceful Degradation)

1. User enters URL
2. Scraping fails (service unavailable, timeout, or extraction error)
3. Form shows error message but continues
4. User can manually enter all fields
5. Save

#### Flow 4: Edit Scraped Content

1. User enters URL
2. Scraping succeeds
3. User reviews scraped title/text
4. User edits fields as needed
5. Save

---

## Architecture

### Component Structure

```
pkg/cli/
├── app.go                    # Main CLI app: wiring, config, and Program startup
├── tui/                      # TUI models and views
│   └── add_link_form.go      # Enhanced add link form with scraping (implemented)
└── client/                   # API client (existing)
```

### State Machine

```
┌─────────────┐
│  URL_INPUT  │ ← Initial state
└──────┬──────┘
       │ User enters URL + Enter
       ↓
┌─────────────┐
│  SCRAPING   │ ← Loading state (async)
└──────┬──────┘
       │ Scraping completes
       ↓
┌─────────────┐
│  REVIEW     │ ← Edit scraped content
└──────┬──────┘
       │ User edits + Enter
       ↓
┌─────────────┐
│  SAVING     │ ← Submitting to API
└──────┬──────┘
       │ API responds
       ↓
┌─────────────┐
│  SUCCESS    │ ← Final state
└─────────────┘
```

### Key States

1. **URL_INPUT** (step 0)
   - User enters URL
   - Validates URL format
   - Triggers scraping on Enter

2. **SCRAPING** (step 1)
   - Shows loading indicator
   - Calls scraper service asynchronously
   - Handles errors gracefully

3. **REVIEW** (step 2)
   - Displays scraped content
   - Allows editing all fields
   - Multi-field navigation (Tab)

4. **SAVING** (step 3)
   - Submits to API
   - Shows loading state

5. **SUCCESS** (step 4)
   - Displays created link
   - Waits for keypress to exit

---

## Implementation Details

### Enhanced Add Link Form

**File**: `pkg/cli/tui/add_link_form.go` (new Bubble Tea model)
**Wiring**: `pkg/cli/app.go` (`App.Run` creates `NewAddLinkForm` with API client, scraper service, and timeout)

#### State Management

```go
type addLinkForm struct {
    // Existing fields
    client     *client.Client
    urlInput   textinput.Model
    titleInput textinput.Model
    descInput  textinput.Model
    textInput  textarea.Model
    step       int
    err        error
    created    *models.Link

    // New fields for scraping integration
    scraperService *scraper.ScraperService
    scraping        bool  // true when scraping is in progress
    scrapeResult    *scraper.ScrapeResponse
    scrapeError     error
    skipScraping    bool  // user chose to skip scraping
    currentField    int   // 0=URL, 1=Title, 2=Description, 3=Text
}
```

#### Step Flow

```go
const (
    stepURLInput = iota  // 0: Enter URL
    stepScraping         // 1: Scraping in progress
    stepReview           // 2: Review and edit scraped content
    stepSaving           // 3: Saving to API
    stepSuccess          // 4: Success message
)
```

#### Key Methods

**1. URL Input Handler**

The URL input handler should validate the URL format, then either:

- Start scraping (Enter key) - creates context, calls `ScrapeWithProgress()` with progress callback
- Skip scraping (s key) - moves directly to review step for manual entry

**2. Scraping Command**

The scraping command should use `ScrapeWithProgress()` with a progress callback that sends messages to the TUI. The callback will receive progress updates at each stage (health check, fetching, extracting, complete).

**3. Scraping Result Handler**

The handler should process `scrapeSuccessMsg`, `scrapeErrorMsg`, and `scrapeProgressMsg` messages. For errors, use `errors.As()` to check for `*scraper.ScraperError` and access structured error information. Progress messages should update the form's progress state for display.

**4. Review Step (Multi-Field Navigation)**

```go
func (m *addLinkForm) handleReviewStep(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            // Navigate to next field
            m.currentField = (m.currentField + 1) % 4
            m.focusCurrentField()
            return m, textinput.Blink

        case "shift+tab":
            // Navigate to previous field
            m.currentField = (m.currentField - 1 + 4) % 4
            m.focusCurrentField()
            return m, textinput.Blink

        case "enter":
            // Save the link
            m.step = stepSaving
            return m, m.submit()

        case "esc":
            return m, tea.Quit
        }
    }

    // Route input to current field
    var cmd tea.Cmd
    switch m.currentField {
    case 0:
        m.urlInput, cmd = m.urlInput.Update(msg)
    case 1:
        m.titleInput, cmd = m.titleInput.Update(msg)
    case 2:
        m.descInput, cmd = m.descInput.Update(msg)
    case 3:
        m.textInput, cmd = m.textInput.Update(msg)
    }
    return m, cmd
}
```

### View Rendering

#### URL Input View

```go
func (m *addLinkForm) renderURLInput() string {
    var s strings.Builder
    s.WriteString("\nAdd New Link\n\n")
    s.WriteString("URL (required):\n")
    s.WriteString(m.urlInput.View())

    if m.err != nil {
        s.WriteString(fmt.Sprintf("\n\n❌ %s", m.err))
    }

    s.WriteString("\n\n")
    s.WriteString("Press Enter to scrape, 's' to skip scraping, Esc to cancel")

    return s.String()
}
```

#### Scraping View

The scraping view should display the current progress stage and message. Use `m.scrapeProgress` to determine which stage indicator to show, and display `m.scrapeProgressMessage` for the current operation status.

#### Review View

```go
func (m *addLinkForm) renderReview() string {
    var s strings.Builder
    s.WriteString("\nReview & Edit Link\n\n")

    // URL (read-only, shows scraped indicator)
    s.WriteString("✓ URL: " + m.urlInput.Value() + "\n\n")

    // Title field
    s.WriteString("Title: ")
    if m.scrapeResult != nil && m.scrapeResult.Title != "" {
        s.WriteString("(scraped) ")
    }
    s.WriteString("\n")
    if m.currentField == 1 {
        s.WriteString(lipgloss.NewStyle().Bold(true).Render(m.titleInput.View()))
    } else {
        s.WriteString(m.titleInput.View())
    }
    s.WriteString("\n\n")

    // Description field
    s.WriteString("Description (optional):\n")
    if m.currentField == 2 {
        s.WriteString(lipgloss.NewStyle().Bold(true).Render(m.descInput.View()))
    } else {
        s.WriteString(m.descInput.View())
    }
    s.WriteString("\n\n")

    // Text field (textarea)
    s.WriteString("Text (optional):\n")
    if m.currentField == 3 {
        s.WriteString(lipgloss.NewStyle().Bold(true).Render(m.textInput.View()))
    } else {
        s.WriteString(m.textInput.View())
    }

    // Show scraping error if any
    if m.scrapeError != nil {
        s.WriteString(fmt.Sprintf("\n\n⚠️  Scraping failed: %v (you can still fill fields manually)", m.scrapeError))
    }

    s.WriteString("\n\n")
    s.WriteString("[Tab] Navigate  [Enter] Save  [Esc] Cancel")

    return s.String()
}
```

### Message Types

```go
// Scraping messages
type scrapeSuccessMsg struct {
    result *scraper.ScrapeResponse
}

type scrapeErrorMsg struct {
    err error  // Will be *scraper.ScraperError after refactoring
}

type scrapeProgressMsg struct {
    stage   scraper.ScrapeStage
    message string
}

// Existing messages
type submitErrorMsg struct {
    err error
}

type submitSuccessMsg struct {
    link *models.Link
}
```

### Additional Form Fields

The form needs additional fields to track scraping state:

- `scrapeProgress` - Current scraping stage (`scraper.ScrapeStage`)
- `scrapeProgressMessage` - Current progress message
- `scrapeErrorType` - Error type if scraping failed (`scraper.ErrorType`)
- `scrapeCtx` - Context for cancellation
- `scrapeCancel` - Cancel function for the context

---

## User Experience Enhancements

### Visual Indicators

1. **Scraped Content Indicator**
   - Show "(scraped)" label next to auto-filled fields
   - Use subtle color to distinguish scraped vs. manual content

2. **Field Focus**
   - Highlight current field with bold text or border
   - Clear visual indication of which field is active

3. **Loading States**
   - Spinner or progress indicator during scraping
   - Non-blocking UI (user can still see progress)

4. **Error Handling**
   - Show errors inline without blocking flow
   - Allow continuation even if scraping fails

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Submit current step / Save link |
| `Tab` | Navigate to next field |
| `Shift+Tab` | Navigate to previous field |
| `Esc` | Cancel / Exit |
| `s` | Skip scraping (from URL input) |
| `Ctrl+C` | Force quit |

### Field Navigation

- **Tab Order**: URL → Title → Description → Text
- **Visual Focus**: Bold text or border on active field
- **Auto-focus**: Automatically focus first editable field after scraping

---

## Error Handling

### Scraping Errors

1. **Service Unavailable**
   - Show warning message
   - Continue to manual entry
   - Don't block user from adding link

2. **Scraping Timeout**
   - Show timeout message
   - Continue to manual entry
   - Suggest retry option

3. **Invalid URL**
   - Show validation error
   - Stay on URL input step
   - Highlight error message

4. **Extraction Failure**
   - Show error from scraper
   - Continue to manual entry
   - Allow user to fill fields manually

### API Errors

1. **Network Error**
   - Show error message
   - Stay on review step
   - Allow retry

2. **Validation Error**
   - Show field-specific errors
   - Highlight problematic fields
   - Allow correction

3. **Authentication Error**
   - Show clear error message
   - Suggest checking API key
   - Exit gracefully

---

## Integration Points

### Scraper Service Integration

```go
// In NewApp or getScraperService
func (a *App) getScraperService() (*scraper.ScraperService, error) {
    if a.scraperService == nil {
        if a.cfg.CLI.BaseURL == "" {
            return nil, fmt.Errorf("base URL not configured")
        }
        a.scraperService = scraper.NewScraperService(a.cfg.CLI.BaseURL)
    }
    return a.scraperService, nil
}

// In newAddLinkForm
func newAddLinkForm(client *client.Client, scraperService *scraper.ScraperService) *addLinkForm {
    // ... existing initialization ...
    return &addLinkForm{
        // ... existing fields ...
        scraperService: scraperService,
        scraping:        false,
        skipScraping:   false,
        currentField:   0,
    }
}
```

### Configuration Integration

- Use `cfg.CLI.ScrapeTimeout` for scraping timeout
- Use `cfg.CLI.BaseURL` for scraper service URL
- Respect user preferences (future: allow disabling auto-scrape)

---

## Scraper Client Refactoring Status

The scraper client has been refactored to support better TUI integration. The following improvements have been completed:

### 1. Context Support for Cancellation ✅ COMPLETE

**Status**: ✅ Implemented in `pkg/scraper/client.go`

- ✅ `ScrapeWithContext()` method added with context support
- ✅ `CheckHealthWithContext()` method added
- ✅ Backward compatible wrappers maintain existing API
- ✅ Supports cancellation during HTTP requests
- ✅ Detects context cancellation and timeout errors

**Benefits for TUI**:

- Users can cancel scraping operations
- Better resource management
- Cleaner error handling when cancelled

### 2. Structured Error Types ✅ COMPLETE

**Status**: ✅ Implemented in `pkg/scraper/errors.go`

- ✅ `ScraperError` type with error categories
- ✅ `ErrorType` enum with 7 error types (ServiceUnavailable, Timeout, Network, Extraction, InvalidURL, InvalidResponse, Cancelled)
- ✅ Helper methods: `Error()`, `Unwrap()`, `IsRetryable()`, `UserMessage()`
- ✅ All error returns use structured types
- ✅ Helper functions for creating specific error types

**Benefits for TUI**:

- Show specific, actionable error messages
- Decide whether to retry automatically
- Provide better user guidance (e.g., "Service unavailable - check if services are running")

### 3. Progress Callbacks ✅ COMPLETE

**Status**: ✅ Implemented in `pkg/scraper/types.go` and `pkg/scraper/client.go`

- ✅ `ProgressCallback` type and `ScrapeStage` enum
- ✅ `ScrapeWithProgress()` method with progress updates
- ✅ `CheckHealthWithProgress()` method
- ✅ Progress updates at key stages (health check, fetching, extracting, complete)
- ✅ Optional callbacks (pass `nil` to disable)

**Benefits for TUI**:

- Show real-time progress to users
- Better UX during long operations
- Clear indication of what's happening

### 4. Enhanced Response Metadata ⏳ TODO (Optional)

**Current Issue**: `ScrapeResponse` only includes basic fields. Additional metadata would help the TUI provide better feedback.

**Proposed Solution**: Add metadata fields to response (Duration, Partial, Warnings).

**Benefits for TUI**:

- Show scraping duration to user
- Indicate if content is partial
- Display warnings (e.g., "Title extracted but text extraction failed")

**Note**: This is optional and can be implemented later if needed.

### 5. Automatic Retry Logic ⏳ TODO (Future Feature)

**Status**: Not yet implemented

**Proposed**: Add configurable retry logic for transient errors.

**Benefits for TUI**:

- Better reliability for transient errors
- Reduced user friction
- Configurable retry behavior

### 6. Health Check with Details ⏳ TODO (Optional)

**Status**: Not yet implemented - Optional enhancement

**Proposed**: Return detailed health information (`HealthStatus` type) for better diagnostics.

**Benefits for TUI**:

- Show more informative health status
- Better error messages
- Help users diagnose issues

### 7. Request/Response Logging Abstraction ⏳ TODO (Future Feature)

**Status**: Not yet implemented

**Proposed**: Abstract logging behind an interface for better TUI control.

**Benefits for TUI**:

- Better control over logging in TUI context
- Can suppress or redirect logs
- Easier testing

### Implementation Status Summary

✅ **Complete** (High Priority - Required for TUI):

- Context support (#1)
- Structured error types (#2)
- Progress callbacks (#3)

⏳ **Optional** (Can be added later):

- Enhanced response metadata (#4)
- Health check details (#6)

⏳ **Future Features**:

- Automatic retry (#5)
- Logging abstraction (#7)

---

## Testing Strategy

### Unit Tests

- Test state transitions
- Test field navigation
- Test error handling
- Test scraping integration

### Integration Tests

1. **Happy Path**
   - Enter URL → Scrape succeeds → Edit → Save
   - Verify all fields populated correctly

2. **Error Paths**
   - Scraping fails → Manual entry works
   - Service unavailable → Graceful degradation
   - Invalid URL → Validation works

3. **User Interactions**
   - Tab navigation works
   - Skip scraping works
   - Cancel works at any step

### Manual Testing

- Test with various URL types
- Test with slow-loading URLs
- Test with invalid URLs
- Test keyboard navigation
- Test error scenarios

---

## Implementation Phases

### Phase 0: Scraper Client Refactoring ✅ COMPLETE

**Goal**: Refactor scraper client to support better TUI integration

- [x] **Context Support** (#1) ✅
    - ✅ `ScrapeWithContext()` method implemented
    - ✅ `CheckHealthWithContext()` method implemented
    - ✅ Supports cancellation during operations
    - ✅ Existing `Scrape()` uses context.Background() for backward compatibility

- [x] **Structured Error Types** (#2) ✅
    - ✅ `ScraperError` type with error categories implemented
    - ✅ `ErrorType` enum with 7 error types
    - ✅ Helper methods (`IsRetryable()`, `UserMessage()`, `Error()`, `Unwrap()`)
    - ✅ All error returns use structured types

- [x] **Progress Callbacks** (#3) ✅
    - ✅ `ProgressCallback` type and `ScrapeStage` enum implemented
    - ✅ `ScrapeWithProgress()` method implemented
    - ✅ `CheckHealthWithProgress()` method implemented
    - ✅ Progress updates at key stages (health check, fetching, extracting, complete)

- [ ] **Enhanced Response Metadata** (#4) ⏳ TODO (Optional)
    - Add `Duration`, `Partial`, `Warnings` fields to `ScrapeResponse`
    - Update scraper service to return metadata (if available)

- [ ] **Health Check Details** (#7) ⏳ TODO (Optional)
    - Create `HealthStatus` type with detailed information
    - Implement `CheckHealthDetailed()` method
    - Keep existing `CheckHealth()` for backward compatibility

**Status**: ✅ **Minimum required refactorings complete** - Ready for TUI integration

The high-priority refactorings (Context Support, Structured Error Types, Progress Callbacks) are complete and the scraper client is ready for TUI integration.

### Phase 1: Basic Integration ✅ (Foundation Exists)

- [x] Existing `addLinkForm` structure
- [x] Basic field inputs (URL, Title, Description, Text)
- [x] Form submission to API

### Phase 2: Scraping Integration ✅ IMPLEMENTED (Initial Version)

**Prerequisites**: Phase 0 refactorings (#1, #2, #3 minimum)

- [x] Add scraper service to form (`App.getScraperService`, passed into `NewAddLinkForm`)
- [x] Implement scraping command with context support (`startScraping` + `runScrapeCommand` with `context.WithTimeout`)
- [x] Handle scraping results with structured errors (`errors.As` with `*scraper.ScraperError`, mapped to `UserMessage()` in the TUI)
- [x] Auto-fill fields with scraped content (title and text pre-filled when available)
- [ ] Integrate progress callbacks for loading states (callbacks are wired, but progress messages are not yet surfaced into the Bubble Tea event loop)
- [x] Handle cancellation (user presses Esc during scraping; context is cancelled)

### Phase 3: Enhanced UX ⏳ PARTIALLY COMPLETE

- [x] Multi-field navigation (Tab / Shift+Tab in review step)
- [x] Visual focus indicators (active field rendered in bold using `lipgloss`)
- [x] Scraped content indicators (`(scraped)` label next to auto-filled title)
- [x] Loading states with basic progress text (`renderScraping`, stage label, message)
- [x] Error messages using structured error types (map `ScraperError.UserMessage()` into UI via `userFacingError`)
- [ ] Show scraping duration in success message

### Phase 4: Polish ⏳ PARTIALLY COMPLETE

- [x] Keyboard shortcuts (Enter, Tab, Shift+Tab, Esc, `s`, Ctrl+C)
- [x] Skip scraping option (`s` from URL input to go directly to review/manual entry)
- [ ] Better error handling with retry suggestions
- [ ] Success animations
- [ ] Help text
- [ ] Show warnings from scraping (if any)

---

## Acceptance Criteria

### Functional Requirements

- [x] User can enter URL and trigger scraping
- [x] Scraped content auto-fills title and text fields (when available)
- [x] User can edit scraped content before saving
- [x] User can skip scraping and enter manually
- [x] Form handles scraping errors gracefully (falls back to manual entry; shows inline error)
- [x] User can navigate between fields with Tab / Shift+Tab
- [x] Link saves successfully to API

### Non-Functional Requirements

- [x] Scraping doesn't block UI indefinitely (context timeout + cancellable; scraper timeout passed in milliseconds)
- [x] Error messages are clear and actionable (structured `ScraperError` mapped to user-facing messages)
- [x] Visual feedback is clear and consistent (distinct views for URL, scraping, review, success)
- [x] Keyboard navigation is intuitive
- [x] Form works even if scraping fails

### User Experience

- [x] Flow feels natural and intuitive (Add → Scrape → Review/Edit → Save)
- [x] Loading states are clear (dedicated scraping view)
- [x] Errors don't block user progress
- [x] Scraped content is clearly indicated
- [x] Field navigation is smooth

---

## Future Enhancements

### Phase 2 Features

1. **Scraping Preview**
   - Show preview of scraped content before auto-filling
   - Allow user to accept/reject scraped content

2. **Advanced Editing**
   - Rich text editing for text field
   - Markdown preview
   - Text formatting shortcuts

3. **Configuration**
   - Toggle auto-scraping on/off
   - Set default scraping behavior
   - Configure field auto-fill preferences

---

## Related Documents

- [`cli-scraper-integration-design.md`](./cli-scraper-integration-design.md) - Separate scrape command design
- [`go-api-cli-design-document.md`](./go-api-cli-design-document.md) - Overall CLI architecture
- [`service-implementation.md`](../scraper/service-implementation.md) - Scraper service details

---

**Last Updated**: 2025-11-29 – Phase 2 scraping integration implemented; Phase 3/4 UX polish in progress
**Next Steps**:

1. ✅ Phase 0: Scraper client refactorings complete (context support, error types, progress callbacks)
2. ✅ Phase 2: Implement scraping integration in TUI using refactored client (initial version complete)
3. Phase 3: Wire structured `ScraperError` messages and progress callbacks into the UI; add duration to success view
4. Phase 4: Polish UX (help text, retry suggestions, animations) and complete testing per the Testing Strategy
