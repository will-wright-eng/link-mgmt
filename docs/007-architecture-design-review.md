# Architecture Design Review

> **Status: Partial** — The completed improvements are real: the service layer (`pkg/services/link_service.go`) and orchestration endpoints (`/links/with-scraping`, `/links/:id/enrich`) exist. The remaining opportunities (async job queue, caching layer, further state-machine simplification) are not implemented.

## Executive Summary

This document reviews the design patterns, complexity distribution, and architectural opportunities in the link management system. The system consists of five components: PostgreSQL, Go API, Nginx reverse proxy, TypeScript/Bun scraper service, and Go CLI.

**Key Finding**: Significant architectural improvements have been made since the initial review. A service layer has been added to the API, orchestration has been moved from CLI to API endpoints, and business logic is now centralized. The CLI has been simplified, but still contains most of the system's UI complexity. Remaining opportunities include async job queues, caching, and further CLI state management simplification.

---

## Current Architecture Overview

```text
┌─────────────┐
│     CLI     │  (Go, Bubble Tea TUI - Thin Client)
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────┐
│    Nginx    │  (Reverse Proxy)
└──────┬──────┘
       │
   ┌───┴───┐
   │       │
   ▼       ▼
┌─────┐ ┌────────┐
│ API │ │Scraper │  (Go API with Service Layer, TypeScript/Bun Scraper)
└──┬──┘ └────────┘
   │
   ▼
┌──────────┐
│ Postgres │  (Database)
└──────────┘
```

### Component Responsibilities

1. **PostgreSQL**: Data persistence
2. **API (Go)**: REST endpoints, authentication, business logic orchestration, service layer
3. **Nginx**: Request routing, reverse proxy
4. **Scraper (TypeScript/Bun)**: Content extraction from URLs
5. **CLI (Go)**: Interactive TUI, thin client for API operations

---

## Design Patterns Analysis

### Current Patterns

#### 1. **Layered Architecture** ✅

- **API Layer**: Handlers → Service Layer → DB Layer → Models
- **CLI Layer**: TUI Models → Client → API
- Clear separation between presentation, business logic, and data access

#### 2. **State Machine Pattern** ⚠️

- **Location**: CLI TUI models (`manage_links.go`, `form_add_link.go`)
- **Implementation**: Step-based state machines (0=list, 1=action menu, 2=view, etc.)
- **Status**: Simplified from 8 steps to 7 steps
- **Improvement**: Business logic moved to API, but UI state management still complex
- **Example**: `manageLinksModel` has 7 different steps (list, action menu, view, delete confirm, enriching, enrich done, done)

#### 3. **Command Pattern** ✅

- **Location**: Bubble Tea commands for async operations
- **Implementation**: Commands return messages that update models
- **Example**: `runScrapeCommand`, `watchProgress`, `saveEnrichedLink`

#### 4. **Repository Pattern** ✅

- **Location**: `pkg/db/queries.go`
- **Implementation**: DB methods abstract SQL queries
- **Strength**: Clean separation of data access

#### 5. **Dependency Injection** ✅

- **Location**: Services passed to TUI models
- **Implementation**: `client.Client`, `scraper.ScraperService` injected
- **Strength**: Testable, loosely coupled

#### 6. **Client-Server Pattern** ✅

- **Location**: CLI → API/Scraper via HTTP
- **Implementation**: RESTful API, HTTP client abstractions
- **Strength**: Clear service boundaries

### Implemented Patterns (Recent Improvements)

#### 1. **Service Layer Pattern** ✅ **IMPLEMENTED**

- **Location**: `pkg/services/link_service.go` (187 lines)
- **Implementation**: `LinkService` encapsulates business logic for link operations
- **Benefits**: Business logic centralized, handlers are thin wrappers, improved testability
- **Status**: All link handlers use `LinkService` instead of calling DB directly

#### 2. **Facade Pattern** ✅ **IMPLEMENTED**

- **Location**: API endpoints `/api/v1/links/with-scraping` and `/api/v1/links/:id/enrich`
- **Implementation**: API orchestrates scraping + saving, CLI just calls endpoints
- **Benefits**: Orchestration logic moved from CLI to API, reusable by other clients
- **Status**: CLI no longer coordinates scraping locally

### Missing Patterns

#### 1. **Observer/Event Pattern** ❌

- **Current**: Synchronous scraping blocks API/CLI requests
- **Issue**: Long-running operations block connections (30s+ timeouts)
- **Opportunity**: Async job queue with event notifications

#### 2. **Strategy Pattern** ⚠️

- **Current**: Single scraping strategy (readability-based extraction)
- **Opportunity**: Pluggable extraction strategies (readability, custom, etc.)

---

## Complexity Distribution

### CLI Complexity (Medium-High) 🟡

**Lines of Code**: ~2,138 lines across all TUI components

**Complexity Sources**:

1. **State Management** (505 lines in `manage_links.go`)
   - 7-step state machine (simplified from 8)
   - Multiple view modes (list, action menu, view, delete, enriching, enrich done, done)
   - State transitions simplified (no local scraping coordination)

2. **UI Orchestration** (400 lines in `form_add_link.go`, 597 lines in `viewport_wrapper.go`)
   - Form field management
   - Viewport scrolling and rendering
   - Multi-field navigation
   - Help system integration

3. **Viewport & Rendering** (597 lines in `viewport_wrapper.go`, 291 lines in `helpers.go`)
   - Viewport management for scrolling
   - Rendering helpers and styles
   - Layout management

**Improvements Made**:

- ✅ **Orchestration removed**: No longer coordinates scraping locally - calls API endpoints
- ✅ **Business logic removed**: Link enrichment logic moved to API service layer
- ✅ **Simplified state machine**: Reduced from 8 to 7 steps
- ✅ **Thin client pattern**: CLI now primarily calls API endpoints

**Remaining Complexity**:

- UI state management still complex (7-step state machine)
- Form field coordination and viewport management
- Rendering and styling logic

**Example Current State**:

```go
// manage_links.go - Simplified state machine
type manageLinksModel struct {
    step int // 0=list, 1=action menu, 2=view, 3=delete confirm,
             // 4=enriching, 5=enrich done, 6=done
    // No scrapeState - API handles enrichment
}

// Simple API call - no local orchestration
func (m *manageLinksModel) enrichLink() tea.Cmd {
    return func() tea.Msg {
        updated, err := m.client.EnrichLink(...) // API handles everything
        if err != nil {
            return managelinks.EnrichErrorMsg{Err: err}
        }
        return managelinks.EnrichSuccessMsg{Link: updated}
    }
}
```

### API Complexity (Medium) 🟡

**Lines of Code**: ~696 lines (handlers: 296, services: 187, router: 58, middleware: ~155)

**Complexity Sources**:

1. **Service Layer** (187 lines in `link_service.go`)
   - Business logic for link operations
   - Orchestration of scraping + saving
   - `CreateLinkWithScraping()` and `EnrichLink()` methods
   - Link enrichment logic (only fill empty fields)

2. **Handlers** (296 lines - 224 links + 59 users + 13 health)
   - Thin wrappers around service layer
   - Request/response formatting
   - Error handling

3. **Router Setup** (58 lines)
   - Route definitions with service injection
   - Middleware configuration
   - New endpoints: `/with-scraping`, `/:id/enrich`

4. **Middleware** (~155 lines)
   - Auth, logging, error recovery
   - Standard patterns

**Strengths**:

- ✅ Service layer centralizes business logic
- ✅ Handlers are thin and focused
- ✅ Orchestration capabilities added
- ✅ Testable business logic

**Current Architecture**:

```
Handlers → LinkService → DB/Scraper
         (thin)    (business logic)
```

### Scraper Complexity (Low) 🟢

**Lines of Code**: ~300 lines

**Complexity Sources**:

1. **HTTP Server** (~150 lines)
   - Simple request routing
   - Health checks
   - Error handling

2. **Browser Management** (~50 lines)
   - Playwright lifecycle
   - Resource cleanup

3. **Content Extraction** (~50 lines)
   - Readability algorithm
   - DOM parsing

**Strengths**: Focused, single responsibility

---

## Architectural Improvements (Completed)

### 1. **Service Layer Added to API** ✅ **COMPLETED**

**Status**: Implemented in `pkg/services/link_service.go` (187 lines)

**Implementation**:

```go
// pkg/services/link_service.go
type LinkService struct {
    db      *db.DB
    scraper *scraper.ScraperService
}

func (s *LinkService) CreateLinkWithScraping(ctx context.Context,
    userID uuid.UUID, linkCreate models.LinkCreate, scrapeOptions ScrapeOptions) (*models.Link, error) {
    // Business logic: create link, scrape, merge, update
}

func (s *LinkService) EnrichLink(ctx context.Context, linkID, userID uuid.UUID,
    scrapeOptions ScrapeOptions) (*models.Link, error) {
    // Business logic: scrape existing link, merge, update
}
```

**Benefits Achieved**:

- ✅ Business logic centralized in service layer
- ✅ Handlers are thin wrappers (296 lines total)
- ✅ Testable without HTTP layer
- ✅ Clear separation of concerns

---

### 2. **Orchestration Moved to API** ✅ **COMPLETED**

**Status**: Implemented with `/api/v1/links/with-scraping` and `/api/v1/links/:id/enrich` endpoints

**Implementation**:

```go
// API endpoints
POST /api/v1/links/with-scraping
POST /api/v1/links/:id/enrich

// CLI now just calls API
func (m *manageLinksModel) enrichLink() tea.Cmd {
    return func() tea.Msg {
        updated, err := m.client.EnrichLink(link.ID, timeout, onlyFillEmpty)
        // API handles all orchestration
    }
}
```

**Benefits Achieved**:

- ✅ CLI is now a thin client (calls API endpoints)
- ✅ Business logic centralized in API service layer
- ✅ Reusable by other clients (web UI, mobile, etc.)
- ✅ CLI complexity reduced (orchestration removed)
- ✅ Easier to test business logic

---

## Architectural Opportunities (Remaining)

### 1. **Async Job Queue for Scraping** 🎯 Medium Impact

---

### 1. **Async Job Queue for Scraping** 🎯 Medium Impact

**Current Problem**:

- Synchronous scraping blocks CLI/API
- Long timeouts (30s+) tie up connections
- No retry mechanism
- Hard to scale

**Proposed Solution**:

```
User Request → API → Job Queue → Worker → Scraper → Update DB
                ↓
            Return 202 Accepted
                ↓
         Webhook/WebSocket → Notify completion
```

**Technology Options**:

- **Simple**: In-memory queue with goroutines
- **Production**: Redis + BullMQ or RabbitMQ
- **Cloud**: AWS SQS, Google Cloud Tasks

**Benefits**:

- ✅ Non-blocking requests
- ✅ Better scalability
- ✅ Retry logic
- ✅ Background processing
- ✅ Progress tracking via job status

**Implementation Effort**: High (5-7 days)

---

### 2. **Further Simplify CLI State Management** 🎯 Medium Impact

**Current Status**: Already improved (7 steps, orchestration removed)

**Remaining Problem**:

- 7-step state machine still complex
- State transitions could be further simplified
- Some UI logic could be extracted

**Proposed Solutions**:

**Option A: Reduce to 4-5 core states**

```go
type manageLinksModel struct {
    mode Mode // list | action | editing | viewing
    selected int
    links []models.Link
    // Fewer states, clearer transitions
}
```

**Option B: Extract state machine to separate package**

```go
// pkg/cli/tui/state/flow.go
type FlowState struct {
    // State machine logic here
}

// TUI just renders current state
```

**Benefits**:

- ✅ Easier to maintain
- ✅ Clearer state transitions
- ✅ Better testability

**Implementation Effort**: Medium (2-3 days)

**Priority**: Lower than async jobs (current state is manageable)

---

### 3. **Add Caching Layer** 🎯 Low-Medium Impact

**Current Problem**:

- No caching of scraped content
- Re-scraping same URL wastes resources
- No deduplication

**Proposed Solution**:

```go
// Cache key: URL + timestamp (invalidate after 24h)
type ScrapeCache interface {
    Get(url string) (*ScrapeResult, bool)
    Set(url string, result *ScrapeResult, ttl time.Duration)
}

// In scraper service or API
if cached, ok := cache.Get(url); ok {
    return cached, nil
}
```

**Benefits**:

- ✅ Faster responses
- ✅ Reduced load on scraper
- ✅ Cost savings

**Implementation Effort**: Low (1 day)

---

### 4. **Event-Driven Architecture** 🎯 Low Priority

**Current Problem**:

- Tight coupling between components
- Hard to add new features (e.g., notifications, analytics)

**Proposed Solution**:

```
Link Created → Event Bus → [Analytics, Notifications, Webhooks, ...]
```

**Benefits**:

- ✅ Loose coupling
- ✅ Extensible
- ✅ Better observability

**Implementation Effort**: High (5+ days)

**Recommendation**: Defer until needed

---

## Recommended Refactoring Priority

### Phase 1: ✅ **COMPLETED** (Quick Wins)

1. ✅ **Service Layer Added to API** - COMPLETED
   - Business logic extracted to `LinkService`
   - Handlers are now thin wrappers
   - Improved testability

2. ✅ **"Create with Scraping" API Endpoints Added** - COMPLETED
   - `/api/v1/links/with-scraping` endpoint implemented
   - `/api/v1/links/:id/enrich` endpoint implemented
   - Orchestration moved from CLI to API
   - CLI complexity reduced

**Result**: Significant architectural improvement, complexity better distributed

---

### Phase 2: Scalability & Performance (2-3 weeks) - **NEXT STEPS**

1. **Add Caching Layer** (1 day) - **Quick Win**
   - Cache scraped content
   - Reduce duplicate scraping
   - Faster responses

2. **Async Job Queue** (5-7 days) - **High Impact**
   - Non-blocking scraping operations
   - Better scalability
   - Retry logic
   - Progress tracking

3. **Further Simplify CLI State Management** (2-3 days) - **Optional**
   - Reduce state machine complexity
   - Extract to separate package
   - Lower priority if current state is manageable

**Total Effort**: ~2 weeks
**Benefits**: 10x scalability improvement, better performance

### Phase 3: Advanced Features (Future)

1. **Event-Driven Architecture** (5+ days)
   - Only if needed for extensibility
   - Webhooks, analytics, etc.

---

## Design Principles Assessment

### ✅ Strengths

1. **Clear Component Boundaries**: Each service has distinct responsibility
2. **Layered Architecture**: Excellent separation with service layer
3. **Service Layer Pattern**: Business logic centralized in `LinkService`
4. **Dependency Injection**: Testable, loosely coupled
5. **Repository Pattern**: Clean data access abstraction
6. **Orchestration in API**: Scraping coordination handled by API, not CLI
7. **Thin CLI Client**: CLI focuses on UI, delegates to API

### ⚠️ Areas for Improvement

1. **CLI State Management**: Still complex (7-step state machine)
2. **Synchronous Operations**: Scraping blocks requests (no async queue)
3. **No Caching**: Re-scraping same URLs wastes resources
4. **State Machine Complexity**: Could be further simplified or extracted

---

## Metrics & Complexity Scores

| Component | LOC | Complexity Score | Maintainability |
|-----------|-----|------------------|-----------------|
| CLI TUI   | ~2,138 | 🟡 Medium-High (6/10) | ✅ Good |
| API       | ~696  | 🟡 Medium (5/10)   | ✅ High |
| Service Layer | 187 | 🟢 Low-Medium (4/10) | ✅ High |
| Scraper   | ~300  | 🟢 Low (2/10)   | ✅ High |
| DB Layer  | ~249  | 🟢 Low (2/10)   | ✅ High |

**Complexity Distribution**: ~60% in CLI, ~25% in API, ~15% in Scraper/DB

**Previous Distribution**: 70% CLI, 20% API, 10% Scraper

**Improvement**: Complexity better distributed, API now handles business logic

**Breakdown**:

- CLI: 2,138 lines (UI state management, rendering, viewport)
- API Handlers: 296 lines (thin wrappers)
- Service Layer: 187 lines (business logic)
- Router/Middleware: ~213 lines
- Total API: ~696 lines

---

## Conclusion

The architecture has been **significantly improved** since the initial review. Major refactoring work has been completed:

### ✅ **Completed Improvements**

1. **Service Layer Added** - Business logic centralized in `LinkService`
2. **Orchestration Moved to API** - `/with-scraping` and `/enrich` endpoints handle coordination
3. **CLI Simplified** - Removed business logic and orchestration, now thin client
4. **Better Separation of Concerns** - Clear layers: Handlers → Services → DB/Scraper

### 📊 **Current State**

- **Complexity Distribution**: Much improved - ~60% CLI (UI), ~25% API (business logic), ~15% Scraper/DB
- **Architecture Quality**: High - clear layers, testable, maintainable
- **CLI Complexity**: Reduced from High (8/10) to Medium-High (6/10)
- **API Complexity**: Increased from Low (3/10) to Medium (5/10) - appropriate for business logic

### 🎯 **Remaining Opportunities**

1. **Async Job Queue** - Improve scalability for long-running scraping operations
2. **Caching Layer** - Reduce duplicate scraping, improve performance
3. **Further CLI Simplification** - Optional, lower priority if current state is manageable

### 📈 **Recommended Next Steps**

1. ✅ Phase 1 completed - Service layer and orchestration endpoints added
2. **Proceed to Phase 2** - Add caching and async job queue for scalability
3. Consider Phase 3 only if needed for advanced features

The architecture is now **well-structured** with good separation of concerns. The improvements have successfully redistributed complexity and centralized business logic in the API layer, making the system more maintainable and testable.
