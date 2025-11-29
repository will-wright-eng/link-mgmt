# TUI Design Document - Interactive Link Addition with Scraping

## Overview

This document outlines the design for an interactive Terminal User Interface (TUI) that integrates URL scraping into the link addition workflow. The TUI provides a seamless flow: **Add → Scrape → Confirm/Edit → Save**.

**Goal**: Create an intuitive, interactive TUI that automatically scrapes URLs when adding links, allows users to review and edit scraped content, and saves the final result.

**Timeline**: 2-3 days
**Complexity**: Medium-High
**Status**: 🟡 **Design Phase**

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
│ Step 1: URL Input                                            │
│ ─────────────────────────────────────────────────────────── │
│ URL (required):                                              │
│ [https://example.com/article                    ]            │
│                                                               │
│ Press Enter to scrape, Esc to cancel                         │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: Scraping (Loading State)                            │
│ ─────────────────────────────────────────────────────────── │
│ ⏳ Scraping URL... (this may take a few seconds)           │
│                                                               │
│ Checking scraper service... ✓                               │
│ Scraping content...                                          │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Review & Edit                                        │
│ ─────────────────────────────────────────────────────────── │
│ ✓ URL: https://example.com/article                          │
│                                                               │
│ Title: [Example Article Title                    ]          │
│   (scraped)                                                  │
│                                                               │
│ Description (optional): [                          ]        │
│                                                               │
│ Text (optional):                                             │
│ ┌─────────────────────────────────────────────────────┐     │
│ │ Article content here...                             │     │
│ │ (scraped, truncated for display)                     │     │
│ │                                                       │     │
│ └─────────────────────────────────────────────────────┘     │
│                                                               │
│ [Tab] Navigate  [Enter] Save  [Esc] Cancel  [s] Skip scrape │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 4: Success                                              │
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
├── app.go                    # Main App struct (existing)
├── models/                   # TUI Models (new)
│   ├── add_link_form.go     # Enhanced add link form with scraping
│   └── common.go            # Shared TUI utilities
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

**File**: `pkg/cli/app.go` (modify existing `addLinkForm`)

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

```go
func (m *addLinkForm) handleURLInput(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            urlStr := strings.TrimSpace(m.urlInput.Value())
            if urlStr == "" {
                m.err = fmt.Errorf("URL is required")
                return m, nil
            }
            if _, err := url.Parse(urlStr); err != nil {
                m.err = fmt.Errorf("invalid URL: %v", err)
                return m, nil
            }
            // Start scraping
            m.step = stepScraping
            m.scraping = true
            m.err = nil
            return m, m.scrapeURL(urlStr)
        case "s":
            // Skip scraping, go directly to manual entry
            m.skipScraping = true
            m.step = stepReview
            m.titleInput.Focus()
            return m, textinput.Blink
        }
    }
    // Handle text input
    var cmd tea.Cmd
    m.urlInput, cmd = m.urlInput.Update(msg)
    return m, cmd
}
```

**2. Scraping Command (After Refactoring)**

```go
func (m *addLinkForm) scrapeURL(ctx context.Context, urlStr string) tea.Cmd {
    return func() tea.Msg {
        // Check health first (with detailed status)
        healthStatus, err := m.scraperService.CheckHealthDetailed()
        if err != nil {
            return scrapeErrorMsg{
                err: &scraper.ScraperError{
                    Type:    scraper.ErrorTypeServiceUnavailable,
                    Message: "Scraper service unavailable",
                    Cause:   err,
                },
            }
        }

        // Progress callback for UI updates
        onProgress := func(stage scraper.ScrapeStage, message string) {
            // Send progress update to TUI (would need progress message type)
            // This allows showing "Checking service...", "Fetching page...", etc.
        }

        // Scrape the URL with context and progress
        timeout := 30 // from config
        result, err := m.scraperService.ScrapeWithProgress(
            ctx,
            urlStr,
            timeout*1000,
            onProgress,
        )
        if err != nil {
            // Handle structured errors
            var scraperErr *scraper.ScraperError
            if errors.As(err, &scraperErr) {
                return scrapeErrorMsg{err: scraperErr}
            }
            // Fallback for non-structured errors
            return scrapeErrorMsg{
                err: &scraper.ScraperError{
                    Type:    scraper.ErrorTypeNetwork,
                    Message: "Scraping failed",
                    Cause:   err,
                },
            }
        }

        if !result.Success {
            return scrapeErrorMsg{
                err: &scraper.ScraperError{
                    Type:    scraper.ErrorTypeExtraction,
                    Message: result.Error,
                },
            }
        }

        return scrapeSuccessMsg{result: result}
    }
}
```

**3. Scraping Result Handler (After Refactoring)**

```go
func (m *addLinkForm) handleScrapeResult(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case scrapeSuccessMsg:
        m.scraping = false
        m.scrapeResult = msg.result

        // Auto-fill fields with scraped content
        if msg.result.Title != "" {
            m.titleInput.SetValue(msg.result.Title)
        }
        if msg.result.Text != "" {
            m.textInput.SetValue(msg.result.Text)
        }

        // Store metadata for display
        if msg.result.Duration > 0 {
            m.scrapeDuration = msg.result.Duration
        }
        if len(msg.result.Warnings) > 0 {
            m.scrapeWarnings = msg.result.Warnings
        }

        // Move to review step
        m.step = stepReview
        m.titleInput.Focus()
        return m, textinput.Blink

    case scrapeErrorMsg:
        m.scraping = false
        m.scrapeError = msg.err

        // Use structured error for better user messages
        var scraperErr *scraper.ScraperError
        if errors.As(msg.err, &scraperErr) {
            // Store error type for conditional handling
            m.scrapeErrorType = scraperErr.Type
            // Can show specific messages based on error type
            // e.g., "Service unavailable - check if services are running"
        }

        // Continue to manual entry even if scraping failed
        m.step = stepReview
        m.titleInput.Focus()
        return m, textinput.Blink

    case scrapeProgressMsg:
        // Handle progress updates
        m.scrapeProgress = msg.stage
        m.scrapeProgressMessage = msg.message
        return m, nil
    }
    return m, nil
}
```

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

#### Scraping View (After Refactoring)

```go
func (m *addLinkForm) renderScraping() string {
    var s strings.Builder
    s.WriteString("\nAdd New Link\n\n")
    s.WriteString("✓ URL: " + m.urlInput.Value() + "\n\n")

    // Show current progress stage
    switch m.scrapeProgress {
    case scraper.StageHealthCheck:
        s.WriteString("⏳ Checking scraper service...\n")
    case scraper.StageFetching:
        s.WriteString("✓ Service available\n")
        s.WriteString("⏳ Fetching page content...\n")
    case scraper.StageExtracting:
        s.WriteString("✓ Page fetched\n")
        s.WriteString("⏳ Extracting content...\n")
    default:
        s.WriteString("⏳ Scraping URL... (this may take a few seconds)\n")
    }

    // Show progress message if available
    if m.scrapeProgressMessage != "" {
        s.WriteString("\n")
        s.WriteString(m.scrapeProgressMessage)
        s.WriteString("\n")
    }

    s.WriteString("\n(Press Esc to cancel)\n")

    return s.String()
}
```

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

### Additional Form Fields (After Refactoring)

```go
type addLinkForm struct {
    // ... existing fields ...

    // New fields for enhanced features
    scrapeDuration    time.Duration
    scrapeWarnings    []string
    scrapeProgress    scraper.ScrapeStage
    scrapeProgressMessage string
    scrapeErrorType   scraper.ErrorType
    scrapeCtx         context.Context
    scrapeCancel      context.CancelFunc
}
```

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

## Planned Refactoring for Cleaner TUI Integration

To make the TUI implementation cleaner and more maintainable, the following refactorings are planned for the scraper client (`pkg/scraper/client.go`):

### 1. Context Support for Cancellation ⏳ TODO

**Current Issue**: The scraper client doesn't support context cancellation, making it difficult to cancel long-running scrape operations from the TUI.

**Proposed Solution**:

- Add `context.Context` parameter to `Scrape()` method
- Support cancellation during HTTP requests
- Allow TUI to cancel scraping when user presses Esc

```go
// Proposed API
func (s *ScraperService) ScrapeWithContext(ctx context.Context, url string, timeout int) (*ScrapeResponse, error) {
    req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/scrape", ...)
    // ... rest of implementation
}

// Backward compatibility wrapper
func (s *ScraperService) Scrape(url string, timeout int) (*ScrapeResponse, error) {
    return s.ScrapeWithContext(context.Background(), url, timeout)
}
```

**Benefits for TUI**:

- Users can cancel scraping operations
- Better resource management
- Cleaner error handling when cancelled

### 2. Structured Error Types ⏳ TODO

**Current Issue**: All errors are generic `error` types, making it difficult for the TUI to provide specific error messages and handle different error scenarios appropriately.

**Proposed Solution**: Create specific error types that categorize different failure modes:

```go
// Error types
type ScraperError struct {
    Type    ErrorType
    Message string
    Cause   error
}

type ErrorType string

const (
    ErrorTypeServiceUnavailable ErrorType = "service_unavailable"
    ErrorTypeTimeout            ErrorType = "timeout"
    ErrorTypeNetwork            ErrorType = "network"
    ErrorTypeExtraction         ErrorType = "extraction"
    ErrorTypeInvalidURL         ErrorType = "invalid_url"
    ErrorTypeInvalidResponse    ErrorType = "invalid_response"
)

// Helper methods
func (e *ScraperError) Error() string { ... }
func (e *ScraperError) IsRetryable() bool { ... }
func (e *ScraperError) UserMessage() string { ... }
```

**Benefits for TUI**:

- Show specific, actionable error messages
- Decide whether to retry automatically
- Provide better user guidance (e.g., "Service unavailable - check if services are running")

### 3. Progress Callbacks ⏳ TODO

**Current Issue**: The TUI can't show intermediate progress during scraping (e.g., "Checking service...", "Fetching page...", "Extracting content...").

**Proposed Solution**: Add optional progress callback support:

```go
type ProgressCallback func(stage ScrapeStage, message string)

type ScrapeStage string

const (
    StageHealthCheck    ScrapeStage = "health_check"
    StageFetching       ScrapeStage = "fetching"
    StageExtracting     ScrapeStage = "extracting"
    StageComplete       ScrapeStage = "complete"
)

func (s *ScraperService) ScrapeWithProgress(
    ctx context.Context,
    url string,
    timeout int,
    onProgress ProgressCallback,
) (*ScrapeResponse, error) {
    if onProgress != nil {
        onProgress(StageHealthCheck, "Checking scraper service...")
    }
    // ... rest of implementation with progress updates
}
```

**Benefits for TUI**:

- Show real-time progress to users
- Better UX during long operations
- Clear indication of what's happening

### 4. Enhanced Response Metadata ⏳ TODO

**Current Issue**: `ScrapeResponse` only includes basic fields. Additional metadata would help the TUI provide better feedback.

**Proposed Solution**: Add metadata fields to response:

```go
type ScrapeResponse struct {
    Success     bool   `json:"success"`
    URL         string `json:"url"`
    Title       string `json:"title"`
    Text        string `json:"text"`
    ExtractedAt string `json:"extracted_at"`
    Error       string `json:"error,omitempty"`

    // New metadata fields
    Duration    time.Duration `json:"duration,omitempty"`    // Total scraping duration
    Partial     bool          `json:"partial,omitempty"`     // True if only partial content extracted
    Warnings    []string      `json:"warnings,omitempty"`    // Non-fatal warnings
}
```

**Benefits for TUI**:

- Show scraping duration to user
- Indicate if content is partial
- Display warnings (e.g., "Title extracted but text extraction failed")

### 5. Batch Scraping Support ⏳ TODO

**Current Issue**: The scraper service supports batch scraping (`/scrape/batch`), but the Go client doesn't expose this functionality.

**Proposed Solution**: Add batch scraping method:

```go
type BatchScrapeRequest struct {
    URLs    []string `json:"urls"`
    Timeout int      `json:"timeout,omitempty"`
}

type BatchScrapeResult struct {
    URL         string
    Title       string
    Text        string
    ExtractedAt string
    Error       error
}

func (s *ScraperService) ScrapeBatch(
    ctx context.Context,
    urls []string,
    timeout int,
) ([]BatchScrapeResult, error) {
    // Implementation
}
```

**Benefits for TUI**:

- Future feature: batch URL import
- Better performance for multiple URLs
- Consistent API surface

### 6. Automatic Retry Logic ⏳ TODO

**Current Issue**: Transient errors (network hiccups, temporary service unavailability) require manual retry from the user.

**Proposed Solution**: Add configurable retry logic:

```go
type RetryConfig struct {
    MaxRetries      int
    RetryDelay      time.Duration
    RetryableErrors []ErrorType  // Which error types to retry
}

func (s *ScraperService) ScrapeWithRetry(
    ctx context.Context,
    url string,
    timeout int,
    config RetryConfig,
) (*ScrapeResponse, error) {
    // Implementation with retry logic
}
```

**Benefits for TUI**:

- Better reliability for transient errors
- Reduced user friction
- Configurable retry behavior

### 7. Health Check with Details ⏳ TODO

**Current Issue**: Health check only returns success/failure, no additional context.

**Proposed Solution**: Return detailed health information:

```go
type HealthStatus struct {
    Status      string    `json:"status"`       // "ok", "degraded", "down"
    Initialized bool      `json:"initialized"`
    Timestamp   time.Time `json:"timestamp"`
    Version     string    `json:"version,omitempty"`
    Message     string    `json:"message,omitempty"`
}

func (s *ScraperService) CheckHealthDetailed() (*HealthStatus, error) {
    // Implementation
}
```

**Benefits for TUI**:

- Show more informative health status
- Better error messages
- Help users diagnose issues

### 8. Request/Response Logging Abstraction ⏳ TODO

**Current Issue**: HTTP-level logging is mixed with business logic.

**Proposed Solution**: Abstract logging behind an interface:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, err error, fields ...Field)
}

type ScraperService struct {
    baseURL string
    client  *http.Client
    logger  Logger  // Optional logger
}
```

**Benefits for TUI**:

- Better control over logging in TUI context
- Can suppress or redirect logs
- Easier testing

### Implementation Priority

1. **High Priority** (Needed for TUI):
   - Context support (#1)
   - Structured error types (#2)
   - Progress callbacks (#3)

2. **Medium Priority** (Nice to have):
   - Enhanced response metadata (#4)
   - Health check details (#7)

3. **Low Priority** (Future features):
   - Batch scraping (#5)
   - Automatic retry (#6)
   - Logging abstraction (#8)

### Migration Strategy

1. **Phase 1**: Add new methods alongside existing ones (backward compatible)
2. **Phase 2**: Update TUI to use new methods
3. **Phase 3**: Deprecate old methods (with warnings)
4. **Phase 4**: Remove old methods in next major version

This approach ensures:

- No breaking changes to existing code
- TUI can use improved API immediately
- Gradual migration path

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

### Phase 0: Scraper Client Refactoring ⏳ TODO (Prerequisites)

**Goal**: Refactor scraper client to support better TUI integration

- [ ] **Context Support** (#1)
    - Add `ScrapeWithContext()` method
    - Support cancellation during operations
    - Update existing `Scrape()` to use context.Background()

- [ ] **Structured Error Types** (#2)
    - Create `ScraperError` type with error categories
    - Implement `ErrorType` enum (ServiceUnavailable, Timeout, Network, etc.)
    - Add helper methods (`IsRetryable()`, `UserMessage()`)
    - Update all error returns to use structured types

- [ ] **Progress Callbacks** (#3)
    - Add `ProgressCallback` type and `ScrapeStage` enum
    - Implement `ScrapeWithProgress()` method
    - Add progress updates at key stages (health check, fetching, extracting)

- [ ] **Enhanced Response Metadata** (#4)
    - Add `Duration`, `Partial`, `Warnings` fields to `ScrapeResponse`
    - Update scraper service to return metadata (if available)

- [ ] **Health Check Details** (#7)
    - Create `HealthStatus` type with detailed information
    - Implement `CheckHealthDetailed()` method
    - Keep existing `CheckHealth()` for backward compatibility

**Note**: These refactorings can be done incrementally. Minimum required for TUI:

- Context support (#1) - Required for cancellation
- Structured error types (#2) - Required for better error handling
- Progress callbacks (#3) - Highly recommended for UX

### Phase 1: Basic Integration ✅ (Foundation Exists)

- [x] Existing `addLinkForm` structure
- [x] Basic field inputs (URL, Title, Description, Text)
- [x] Form submission to API

### Phase 2: Scraping Integration ⏳ TODO

**Prerequisites**: Phase 0 refactorings (#1, #2, #3 minimum)

- [ ] Add scraper service to form
- [ ] Implement scraping command with context support
- [ ] Handle scraping results with structured errors
- [ ] Auto-fill fields with scraped content
- [ ] Integrate progress callbacks for loading states
- [ ] Handle cancellation (user presses Esc during scraping)

### Phase 3: Enhanced UX ⏳ TODO

- [ ] Multi-field navigation (Tab)
- [ ] Visual focus indicators
- [ ] Scraped content indicators
- [ ] Loading states with progress updates
- [ ] Error messages using structured error types
- [ ] Show scraping duration in success message

### Phase 4: Polish ⏳ TODO

- [ ] Keyboard shortcuts
- [ ] Skip scraping option
- [ ] Better error handling with retry suggestions
- [ ] Success animations
- [ ] Help text
- [ ] Show warnings from scraping (if any)

---

## Acceptance Criteria

### Functional Requirements

- [ ] User can enter URL and trigger scraping
- [ ] Scraped content auto-fills title and text fields
- [ ] User can edit scraped content before saving
- [ ] User can skip scraping and enter manually
- [ ] Form handles scraping errors gracefully
- [ ] User can navigate between fields with Tab
- [ ] Link saves successfully to API

### Non-Functional Requirements

- [ ] Scraping doesn't block UI indefinitely
- [ ] Error messages are clear and actionable
- [ ] Visual feedback is clear and consistent
- [ ] Keyboard navigation is intuitive
- [ ] Form works even if scraping fails

### User Experience

- [ ] Flow feels natural and intuitive
- [ ] Loading states are clear
- [ ] Errors don't block user progress
- [ ] Scraped content is clearly indicated
- [ ] Field navigation is smooth

---

## Future Enhancements

### Phase 2 Features

1. **Scraping Preview**
   - Show preview of scraped content before auto-filling
   - Allow user to accept/reject scraped content

2. **Batch Scraping**
   - Support multiple URLs at once
   - Show progress for each URL

3. **Scraping History**
   - Cache scraped content
   - Reuse scraped content for same URLs

4. **Advanced Editing**
   - Rich text editing for text field
   - Markdown preview
   - Text formatting shortcuts

5. **Configuration**
   - Toggle auto-scraping on/off
   - Set default scraping behavior
   - Configure field auto-fill preferences

---

## Related Documents

- [`cli-scraper-integration-design.md`](./cli-scraper-integration-design.md) - Separate scrape command design
- [`go-api-cli-design-document.md`](./go-api-cli-design-document.md) - Overall CLI architecture
- [`service-implementation.md`](../scraper/service-implementation.md) - Scraper service details

---

**Last Updated**: Added planned refactorings for cleaner TUI integration
**Next Steps**:

1. Phase 0: Implement scraper client refactorings (context support, error types, progress callbacks)
2. Phase 2: Implement scraping integration in TUI using refactored client
