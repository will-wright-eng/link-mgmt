# Manage Links Model Evaluation & Code Quality Analysis

## Overview

This document provides a comprehensive evaluation of the `NewManageLinksModel` implementation and identifies code smells and antipatterns that should be addressed before modifying or adding features.

## Architecture Overview

### Function Signature

- **Location**: `manage_links.go:92-132`
- **Parameters**:
    - `*client.Client` - API client for link operations
    - `*scraper.ScraperService` - Scraping/enrichment service
    - `timeoutSeconds int` - Scraping timeout duration
- **Returns**: `tea.Model` (wrapped in `ViewportWrapper`)

### Core Structure

The model uses a state machine pattern with 8 distinct steps:

1. `manageStepListLinks` (0) - Display list of links
2. `manageStepActionMenu` (1) - Show action menu for selected link
3. `manageStepViewDetails` (2) - Display full link details
4. `manageStepDeleteConfirm` (3) - Delete confirmation prompt
5. `manageStepScraping` (4) - Scraping in progress
6. `manageStepScrapeSaving` (5) - Saving enriched data
7. `manageStepScrapeDone` (6) - Scraping complete
8. `manageStepDone` (7) - Deletion complete

### Key Dependencies

1. **ViewportWrapper** (`viewport_wrapper.go`)
   - Handles viewport/scrolling, headers/footers, help overlay
   - Provides `SelectableModel` interface for auto-scrolling

2. **client.Client** (`pkg/cli/client/`)
   - API client for CRUD operations on links

3. **scraper.ScraperService** (`pkg/scraper/`)
   - Scraping and enrichment functionality

4. **Rendering Helpers** (`helpers.go`)
   - `renderLinkList()` - Formats link list (2 lines per item: title + URL)
   - `formatLinkTitle()` - Returns title or "(no title)"
   - `renderLinkDetailsFull()` - Full link details with text wrapping
   - `truncateURL()` - URL truncation utility

5. **Styles** (`styles.go`)
   - Centralized lipgloss style definitions
   - Consistent color palette and styling

6. **Help Content** (`help.go`)
   - `ManageLinksHelpContent()` - Keyboard shortcut documentation

### Current List Format

The list rendering (`renderLinkList` in `helpers.go`) produces:

```text
→ Selected Title
  https://url.com/...
  Other Title
  https://other.com/...
```

Each item is 2 lines:

- **Line 1**: Selection marker (→ or space) + styled title
- **Line 2**: Indented styled URL (truncated to fit width)

### Interface Implementation

Implements `SelectableModel` interface for automatic viewport scrolling:

- `GetSelectedIndex()` - Returns selected index when in list step (-1 otherwise)
- `GetItemHeight()` - Returns 2 (title + URL)
- `GetListHeaderHeight()` - Returns 2 (subtitle + blank line)

## Code Smells & Antipatterns

### 🔴 Critical Issues

#### 1. Magic Number Duplication - Width Fallback

**Location**: Multiple locations (lines 156, 439, 458, 497, 523)
**Issue**: The width fallback value `80` is hardcoded in 5+ places

```go
if m.width == 0 {
    m.width = 80 // Default fallback
}
```

**Impact**: Difficult to maintain, inconsistent behavior if values differ
**Recommendation**: Extract to a constant:

```go
const defaultWidth = 80
```

#### 2. Repeated Width Validation Logic

**Location**: `renderList()`, `renderActionMenu()`, `renderViewDetails()`, `renderDeleteConfirm()`
**Issue**: Same width fallback pattern repeated in every render method
**Impact**: Code duplication, maintenance burden
**Recommendation**: Extract to helper method:

```go
func (m *manageLinksModel) getMaxWidth() int {
    if m.width > 0 {
        return m.width
    }
    return defaultWidth
}
```

#### 3. Multiple Error Fields

**Location**: `manageLinksModel` struct (lines 27, 36)
**Issue**: Both `err error` and `scrapeError error` fields exist
**Impact**: Confusion about which error to check, error handling complexity
**Recommendation**: Unify error handling or clearly document when each is used

#### 4. Context Cancellation in Defer

**Location**: `runScrapeCommand()` line 593-597
**Issue**: Cancelling context in defer that's already part of the model's lifecycle

```go
defer func() {
    if m.scrapeCancel != nil {
        m.scrapeCancel()
    }
}()
```

**Impact**: Potential double cancellation, unclear ownership
**Recommendation**: Remove defer cancellation; context is managed by the model's lifecycle

### 🟡 Design Issues

#### 5. Complex State Machine

**Location**: `step int` field (line 26)
**Issue**: Single integer represents 8 different states with comment mapping
**Impact**: Easy to introduce invalid states, difficult to extend
**Recommendation**: Consider using a type with methods:

```go
type manageStep int

const (
    stepListLinks manageStep = iota
    stepActionMenu
    // ...
)

func (s manageStep) String() string { /* ... */ }
```

#### 6. Large Struct with Multiple Concerns

**Location**: `manageLinksModel` struct (lines 20-46)
**Issue**: Struct contains fields for list navigation, delete confirmation, scraping, and viewport
**Impact**: Violates Single Responsibility Principle, difficult to test
**Recommendation**: Consider extracting scraping state into a separate struct:

```go
type scrapeState struct {
    scraping      bool
    result        *scraper.ScrapeResponse
    error         error
    stage         scraper.ScrapeStage
    message       string
    ctx           context.Context
    cancel        context.CancelFunc
    progressChan  chan manageScrapeProgressMsg
}
```

#### 7. Progress Watching Complexity

**Location**: `watchProgress()` and `manageProgressTickMsg` (lines 627-646, 87-89)
**Issue**: Uses channel + tick pattern for progress updates, seems overly complex
**Impact**: Difficult to understand, potential race conditions
**Recommendation**: Review if this pattern is necessary or if callback pattern would be simpler

#### 8. Inconsistent Error Handling

**Location**: Multiple message handlers in `Update()` method
**Issue**:

- `manageDeleteErrorMsg` causes `tea.Quit` (line 174)
- Other errors set `m.err` and continue (lines 164, 198)
**Impact**: Inconsistent UX, some errors quit app, others display error
**Recommendation**: Standardize error handling strategy - either all errors show error view or document why deletion errors quit

#### 9. Missing Bounds Checking

**Location**: Various render methods and key handlers
**Issue**: Some places check bounds (line 451, 490, 516), others assume validity
**Impact**: Potential panic if state is invalid
**Recommendation**: Add defensive checks or validate state transitions

#### 10. Hardcoded Default Timeout

**Location**: `NewManageLinksModel()` line 98
**Issue**: Magic number `30` for default timeout
**Impact**: Difficult to change default, not self-documenting
**Recommendation**: Extract to constant:

```go
const defaultScrapeTimeoutSeconds = 30
```

### 🟢 Minor Issues

#### 11. Magic Strings for Key Bindings

**Location**: Key handlers throughout (e.g., lines 299, 302, 305)
**Issue**: Key bindings are string literals scattered throughout code
**Impact**: Difficult to change key bindings, no central configuration
**Recommendation**: Extract to constants or configuration:

```go
const (
    keyViewDetails = "1"
    keyDelete      = "2"
    keyScrape      = "3"
)
```

#### 12. Inline String Rendering

**Location**: `View()` method line 384
**Issue**: Inline string concatenation and rendering for scrape saving step

```go
result = "\n" + infoStyle.Render("Saving enriched content...") + "\n"
```

**Impact**: Inconsistent with other render methods that have dedicated functions
**Recommendation**: Extract to `renderScrapeSaving()` method for consistency

#### 13. Default Case Returns Empty String

**Location**: `View()` method line 392
**Issue**: Default case in switch returns empty string

```go
default:
    logger.Log("View: unknown step=%d, returning empty string", m.step)
    return ""
```

**Impact**: Silent failure, user sees blank screen
**Recommendation**: Return error view or fallback rendering

#### 14. Commented Code Documentation

**Location**: Line 26, line 418
**Issue**: Comment explains what numeric step values mean
**Impact**: Documentation in code suggests the design could be clearer
**Recommendation**: If using enum/type for steps (see #5), this becomes unnecessary

#### 15. URL Truncation Width Calculation

**Location**: Multiple render methods (lines 464, 529)
**Issue**: Repeated calculation: `urlTruncateWidth := maxWidth - 10`
**Impact**: Magic number `10`, duplicated logic
**Recommendation**: Extract to constant or helper:

```go
const urlTruncateMargin = 10
```

## Positive Patterns

### ✅ Good Practices Observed

1. **Separation of Concerns**: Rendering logic separated into helper functions
2. **Consistent Styling**: Centralized styles in `styles.go`
3. **Width-Aware Rendering**: Respects terminal width for truncation/wrapping
4. **Auto-Scrolling**: Implements `SelectableModel` for viewport scrolling
5. **Logging**: Comprehensive logging for debugging
6. **Error Messages**: User-facing error conversion via `userFacingError()`
7. **State Machine Pattern**: Clear flow between states (even if implementation could improve)
8. **Viewport Wrapper**: Good use of wrapper pattern for common functionality

## Recommendations for Refactoring Priority

### High Priority (Before Adding Features)

1. Extract width fallback to constant/helper (#1, #2)
2. Unify error handling strategy (#8)
3. Extract hardcoded constants (timeout, default width) (#1, #10)
4. Add bounds checking (#9)

### Medium Priority (Improve Maintainability)

5. Extract scraping state to separate struct (#6)
6. Replace magic strings with constants (#11)
7. Improve state machine implementation (#5)
8. Standardize render method patterns (#12)

### Low Priority (Nice to Have)

9. Simplify progress watching (#7)
10. Review context cancellation pattern (#4)
11. Improve default case handling (#13)

## Testing Considerations

Before modifying or adding features, consider:

- **State transitions**: How are invalid state transitions handled?
- **Error scenarios**: Are all error paths tested?
- **Width handling**: Test with various terminal widths (0, small, large)
- **Bounds checking**: Test with empty lists, invalid selections
- **Concurrent operations**: Test scraping cancellation, multiple rapid actions

## Summary

The `NewManageLinksModel` implementation is functional but has several code smells that should be addressed before adding new features. The most critical issues are:

1. **Duplication**: Width fallback logic repeated 5+ times
2. **Inconsistency**: Error handling patterns vary
3. **Magic numbers**: Hardcoded values scattered throughout
4. **Complexity**: Large struct with multiple concerns

Addressing these issues will make the codebase more maintainable and reduce the risk of introducing bugs when modifying the format or adding features.
