# Architecture Design Review

## Executive Summary

This document reviews the design patterns, complexity distribution, and architectural opportunities in the link management system. The system consists of five components: PostgreSQL, Go API, Nginx reverse proxy, TypeScript/Bun scraper service, and Go CLI.

**Key Finding**: Most complexity is concentrated in the CLI's TUI layer, which handles UI state management, business logic orchestration, and async operation coordination. This creates opportunities to redistribute complexity and improve separation of concerns.

---

## Current Architecture Overview

```
┌─────────────┐
│     CLI     │  (Go, Bubble Tea TUI)
│  (Complex)  │
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
│ API │ │Scraper │  (Go API, TypeScript/Bun Scraper)
└──┬──┘ └────────┘
   │
   ▼
┌──────────┐
│ Postgres │  (Database)
└──────────┘
```

### Component Responsibilities

1. **PostgreSQL**: Data persistence
2. **API (Go)**: REST endpoints, authentication, CRUD operations
3. **Nginx**: Request routing, reverse proxy
4. **Scraper (TypeScript/Bun)**: Content extraction from URLs
5. **CLI (Go)**: Interactive TUI, orchestration, business logic

---

## Design Patterns Analysis

### Current Patterns

#### 1. **Layered Architecture** ✅

- **API Layer**: Handlers → DB Layer → Models
- **CLI Layer**: TUI Models → Client → API
- Clear separation between presentation and data access

#### 2. **State Machine Pattern** ⚠️

- **Location**: CLI TUI models (`manage_links.go`, `form_add_link.go`)
- **Implementation**: Step-based state machines (0=list, 1=action menu, 2=view, etc.)
- **Issue**: Business logic embedded in UI state transitions
- **Example**: `manageLinksModel` has 8 different steps managing complex flows

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

### Missing Patterns

#### 1. **Service Layer Pattern** ❌

- **Current**: Handlers directly call DB layer
- **Issue**: Business logic scattered across handlers and CLI
- **Opportunity**: Extract business logic to service layer

#### 2. **Facade Pattern** ❌

- **Current**: CLI orchestrates scraping + saving manually
- **Issue**: Complex coordination logic in UI layer
- **Opportunity**: API endpoint that handles "create link with scraping"

#### 3. **Observer/Event Pattern** ❌

- **Current**: Synchronous scraping with progress callbacks
- **Issue**: Long-running operations block UI
- **Opportunity**: Async job queue with event notifications

#### 4. **Strategy Pattern** ⚠️

- **Current**: Single scraping strategy
- **Opportunity**: Pluggable extraction strategies (readability, custom, etc.)

---

## Complexity Distribution

### CLI Complexity (High) 🔴

**Lines of Code**: ~1,500+ lines across TUI components

**Complexity Sources**:

1. **State Management** (650 lines in `manage_links.go`)
   - 8-step state machine
   - Multiple view modes (list, action menu, view, delete, scraping, etc.)
   - Complex state transitions

2. **Async Operation Coordination** (200+ lines)
   - Scraping with progress callbacks
   - Channel-based progress updates
   - Context cancellation handling
   - Multiple concurrent operations

3. **UI Orchestration** (300+ lines)
   - Form field management
   - Viewport scrolling
   - Multi-field navigation
   - Help system integration

4. **Business Logic** (150+ lines)
   - Link enrichment logic (only update empty fields)
   - Validation and error handling
   - Data transformation

**Example Complexity**:

```go
// manage_links.go - Complex state machine
type manageLinksModel struct {
    step int // 0=list, 1=action menu, 2=view, 3=delete, 4=scraping,
             // 5=scrape saving, 6=scrape done, 7=done
    scrapeState managelinks.ScrapeState
    // ... 10+ fields managing state
}

// Complex async coordination
func (m *manageLinksModel) startScraping() (tea.Model, tea.Cmd) {
    // Creates context, channels, progress callbacks
    // Coordinates scraping → saving → reloading
}
```

### API Complexity (Low) 🟢

**Lines of Code**: ~400 lines (handlers + router)

**Complexity Sources**:

1. **Simple CRUD Handlers** (~200 lines)
   - Direct DB calls
   - Basic validation
   - Error handling

2. **Middleware** (~100 lines)
   - Auth, logging, error recovery
   - Standard patterns

3. **Router Setup** (~50 lines)
   - Route definitions
   - Group organization

**Strengths**: Clean, simple, focused

**Weaknesses**:

- No business logic layer
- No orchestration capabilities
- Handlers are thin wrappers around DB calls

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

## Architectural Opportunities

### 1. **Move Orchestration to API** 🎯 High Impact

**Current Problem**:

- CLI coordinates: scrape → wait → save → update
- Business logic (e.g., "only update empty fields") in CLI
- Complex async coordination in UI layer

**Proposed Solution**:

```go
// New API endpoint
POST /api/v1/links/with-scraping
{
  "url": "https://example.com",
  "scrape": true,
  "options": {
    "only_fill_empty": true,
    "timeout": 30
  }
}

// API handler orchestrates:
// 1. Call scraper service
// 2. Merge results with request
// 3. Save to database
// 4. Return enriched link
```

**Benefits**:

- ✅ CLI becomes thin client (just calls API)
- ✅ Business logic centralized in API
- ✅ Reusable by other clients (web UI, mobile, etc.)
- ✅ Easier to test business logic
- ✅ CLI complexity reduced by ~40%

**Implementation Effort**: Medium (2-3 days)

---

### 2. **Add Service Layer to API** 🎯 Medium Impact

**Current Problem**:

- Handlers directly call DB layer
- Business logic scattered (some in handlers, some in CLI)
- Hard to test business rules independently

**Proposed Solution**:

```go
// pkg/services/link_service.go
type LinkService struct {
    db      *db.DB
    scraper *scraper.Client
}

func (s *LinkService) CreateLinkWithScraping(ctx context.Context,
    userID uuid.UUID, req LinkCreateRequest) (*models.Link, error) {
    // Business logic: scrape, merge, validate, save
}

// Handlers become thin:
func CreateLink(service *LinkService) gin.HandlerFunc {
    return func(c *gin.Context) {
        link, err := service.CreateLink(...)
        // Just format response
    }
}
```

**Benefits**:

- ✅ Business logic centralized
- ✅ Testable without HTTP layer
- ✅ Reusable across handlers
- ✅ Clear separation of concerns

**Implementation Effort**: Low-Medium (1-2 days)

---

### 3. **Async Job Queue for Scraping** 🎯 Medium Impact

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

### 4. **Simplify CLI State Management** 🎯 High Impact

**Current Problem**:

- 8-step state machines
- Complex state transitions
- Business logic mixed with UI logic

**Proposed Solution**:

**Option A: Reduce to 3-4 core states**

```go
type manageLinksModel struct {
    mode Mode // list | action | editing
    selected int
    links []models.Link
    // Much simpler
}
```

**Option B: Extract state machine to separate package**

```go
// pkg/cli/state/flow.go
type FlowState struct {
    // State machine logic here
}

// TUI just renders current state
```

**Option C: Use API orchestration (combines with #1)**

- CLI becomes stateless
- API handles all coordination
- CLI just displays results

**Benefits**:

- ✅ Easier to maintain
- ✅ Less code
- ✅ Clearer responsibilities

**Implementation Effort**: Medium (2-3 days)

---

### 5. **Add Caching Layer** 🎯 Low-Medium Impact

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

### 6. **Event-Driven Architecture** 🎯 Low Priority

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

### Phase 1: Quick Wins (1-2 weeks)

1. **Add Service Layer to API** (1-2 days)
   - Extract business logic from handlers
   - Improve testability
   - Foundation for future changes

2. **Add "Create with Scraping" API Endpoint** (2-3 days)
   - Move orchestration from CLI to API
   - Reduce CLI complexity significantly
   - Enable other clients

3. **Simplify CLI State Management** (2-3 days)
   - Reduce state machine complexity
   - Extract business logic to API calls
   - Cleaner separation

**Total Effort**: ~1 week
**Complexity Reduction**: ~40% in CLI

### Phase 2: Scalability (2-3 weeks)

4. **Add Caching Layer** (1 day)
   - Quick performance win
   - Low risk

5. **Async Job Queue** (5-7 days)
   - Better scalability
   - Non-blocking operations
   - Retry logic

**Total Effort**: ~2 weeks
**Scalability**: 10x improvement

### Phase 3: Advanced Features (Future)

6. **Event-Driven Architecture** (5+ days)
   - Only if needed for extensibility
   - Webhooks, analytics, etc.

---

## Design Principles Assessment

### ✅ Strengths

1. **Clear Component Boundaries**: Each service has distinct responsibility
2. **Layered Architecture**: Good separation of concerns in API
3. **Dependency Injection**: Testable, loosely coupled
4. **Repository Pattern**: Clean data access abstraction

### ⚠️ Areas for Improvement

1. **Separation of Concerns**: Business logic in UI layer
2. **Single Responsibility**: CLI doing too much (UI + orchestration + logic)
3. **DRY Principle**: Scraping orchestration duplicated in CLI
4. **Testability**: Hard to test business logic in TUI models

---

## Metrics & Complexity Scores

| Component | LOC | Complexity Score | Maintainability |
|-----------|-----|------------------|-----------------|
| CLI TUI   | ~1,500 | 🔴 High (8/10) | ⚠️ Medium |
| API       | ~400  | 🟢 Low (3/10)   | ✅ High |
| Scraper   | ~300  | 🟢 Low (2/10)   | ✅ High |
| DB Layer  | ~220  | 🟢 Low (2/10)   | ✅ High |

**Complexity Distribution**: 70% in CLI, 20% in API, 10% in Scraper

**Target Distribution**: 40% CLI, 40% API, 20% Scraper (after refactoring)

---

## Conclusion

The current architecture is **well-structured** with clear component boundaries, but **complexity is unevenly distributed**. The CLI contains most of the system's complexity due to:

1. Complex state management
2. Business logic orchestration
3. Async operation coordination

**Key Opportunities**:

1. **Move orchestration to API** - Biggest impact, reduces CLI complexity by ~40%
2. **Add service layer** - Improves testability and maintainability
3. **Simplify CLI state** - Makes UI code more maintainable
4. **Add async jobs** - Improves scalability (future)

**Recommended Next Steps**:

1. Start with Phase 1 refactoring (1 week)
2. Measure complexity reduction
3. Proceed to Phase 2 if scalability becomes a concern
4. Defer Phase 3 until needed

The architecture is **solid** and **refactorable** - these improvements will make it more maintainable and scalable without requiring a complete rewrite.
