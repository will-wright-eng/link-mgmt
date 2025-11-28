# Service-Based Scraper Implementation

## Overview

This document provides a step-by-step implementation plan for adding HTTP service mode to the scraper, enabling better scalability and resource efficiency.

**Goal**: Add HTTP service mode for better scalability and resource efficiency
**Timeline**: 2-3 days
**Complexity**: Medium
**Prerequisites**: Existing scraper codebase with BrowserManager and extraction working (see `scraper/` directory)

**Status**: ✅ **Implementation Complete** - All code implemented, ready for testing

**Completed**:

- ✅ HTTP service server (`scraper/src/server.ts`)
- ✅ Package.json dependencies verified and updated (added `@types/jsdom`)
- ✅ Service Dockerfile (`scraper/Dockerfile.service`)
- ✅ Go HTTP client (`link-mgmt-go/pkg/scraper/client.go`)
- ✅ TypeScript type issues fixed (type-only imports for Playwright types)

**Remaining**:

- ⏳ Testing service mode (manual testing required)

---

## Step 1: Create Scraper HTTP Service

**File**: `scraper/src/server.ts` (new file)

**Action**: Create HTTP server using Bun's native server that exposes scraping via REST API

```typescript
import { BrowserManager } from "./browser";
import { extractMainContent } from "./extractor";
import type { ExtractionResult } from "./types";

let manager: BrowserManager | null = null;
let initialized = false;

// Initialize browser on startup
async function initBrowser() {
  try {
    manager = new BrowserManager();
    await manager.initialize(true); // headless mode
    initialized = true;
    console.log("Browser initialized");
  } catch (error) {
    console.error("Failed to initialize browser:", error);
    process.exit(1);
  }
}

// Graceful shutdown
process.on("SIGTERM", async () => {
  console.log("SIGTERM received, shutting down gracefully");
  if (manager) {
    await manager.cleanup();
  }
  process.exit(0);
});

process.on("SIGINT", async () => {
  console.log("SIGINT received, shutting down gracefully");
  if (manager) {
    await manager.cleanup();
  }
  process.exit(0);
});

// Helper to send JSON response
function sendJSON(response: Response, data: unknown, status: number = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

// Helper to parse JSON body
async function parseJSON(request: Request): Promise<unknown> {
  try {
    return await request.json();
  } catch {
    return null;
  }
}

// Health check
async function handleHealth(): Promise<Response> {
  return sendJSON(
    new Response(),
    {
      status: "ok",
      initialized,
      timestamp: new Date().toISOString(),
    },
    200
  );
}

// Scrape single URL
async function handleScrape(request: Request): Promise<Response> {
  const body = await parseJSON(request);
  if (!body || typeof body !== "object" || !("url" in body)) {
    return sendJSON(new Response(), { error: "URL is required" }, 400);
  }

  const { url, timeout = 10000 } = body as { url: string; timeout?: number };

  if (!url || typeof url !== "string") {
    return sendJSON(new Response(), { error: "URL is required" }, 400);
  }

  if (!initialized || !manager) {
    return sendJSON(new Response(), { error: "Browser not initialized" }, 503);
  }

  try {
    const html = await manager.extractFromUrl(url, timeout);
    const extracted = await extractMainContent(html, url);

    if (!extracted) {
      return sendJSON(
        new Response(),
        {
          success: false,
          url,
          error: "Failed to extract content",
        },
        500
      );
    }

    return sendJSON(
      new Response(),
      {
        success: true,
        url,
        title: extracted.title || "",
        text: extracted.text || "",
        extracted_at: new Date().toISOString(),
      },
      200
    );
  } catch (error) {
    return sendJSON(
      new Response(),
      {
        success: false,
        url,
        error: error instanceof Error ? error.message : "Unknown error",
      },
      500
    );
  }
}

// Batch scrape
async function handleBatchScrape(request: Request): Promise<Response> {
  const body = await parseJSON(request);
  if (
    !body ||
    typeof body !== "object" ||
    !("urls" in body) ||
    !Array.isArray(body.urls)
  ) {
    return sendJSON(new Response(), { error: "urls must be an array" }, 400);
  }

  const { urls, timeout = 10000 } = body as {
    urls: string[];
    timeout?: number;
  };

  if (!initialized || !manager) {
    return sendJSON(new Response(), { error: "Browser not initialized" }, 503);
  }

  const results: ExtractionResult[] = [];

  for (const url of urls) {
    try {
      const html = await manager.extractFromUrl(url, timeout);
      const extracted = await extractMainContent(html, url);

      results.push({
        url,
        title: extracted?.title || "",
        text: extracted?.text || "",
        extracted_at: new Date().toISOString(),
        error: extracted ? null : "Failed to extract content",
      });
    } catch (error) {
      results.push({
        url,
        title: "",
        text: "",
        extracted_at: new Date().toISOString(),
        error: error instanceof Error ? error.message : "Unknown error",
      });
    }
  }

  return sendJSON(new Response(), { results }, 200);
}

// Request router
async function handleRequest(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const path = url.pathname;
  const method = request.method;

  // Health check
  if (path === "/health" && method === "GET") {
    return handleHealth();
  }

  // Single scrape
  if (path === "/scrape" && method === "POST") {
    return handleScrape(request);
  }

  // Batch scrape
  if (path === "/scrape/batch" && method === "POST") {
    return handleBatchScrape(request);
  }

  // 404 for unknown routes
  return sendJSON(new Response(), { error: "Not found" }, 404);
}

// Initialize browser and start server
async function startServer() {
  await initBrowser();

  const port = parseInt(process.env.PORT || "3000", 10);
  const server = Bun.serve({
    port,
    fetch: handleRequest,
  });

  console.log(`Scraper service listening on port ${server.port}`);
}

startServer().catch((error) => {
  console.error("Failed to start server:", error);
  process.exit(1);
});
```

**Tasks**:

- [x] Create `scraper/src/server.ts`
- [x] Implement health check endpoint
- [x] Implement single scrape endpoint
- [x] Implement batch scrape endpoint
- [x] Add graceful shutdown handling
- [ ] Test server locally: `bun run src/server.ts`

**Note**: The server can be run directly with Bun for local development:

```bash
cd scraper
bun run src/server.ts
```

**Acceptance Criteria**:

- [x] Server code implemented ✅
- [ ] Server starts successfully (testing pending)
- [ ] Health check returns 200 (testing pending)
- [ ] Can scrape URLs via HTTP POST (testing pending)
- [x] Graceful shutdown code implemented ✅

---

## Step 2: Update Package.json (if needed)

**File**: `scraper/package.json`

**Action**: Verify dependencies are sufficient (no new dependencies needed since we're using Bun's native server)

The existing dependencies in `package.json` are sufficient:

- `playwright` - for browser automation (already installed)
- `jsdom` - for HTML parsing (already installed)
- `@mozilla/readability` - for content extraction (already installed)

**Additional Changes Made**:

- Added `@types/jsdom` to `devDependencies` for TypeScript support
- Fixed TypeScript import issues in `browser.ts` (type-only imports for `Browser` and `BrowserContext`)

**Tasks**:

- [x] Verify existing dependencies are up to date
- [x] Run `bun install` to ensure all packages are installed
- [x] Added `@types/jsdom` for TypeScript type definitions

**Acceptance Criteria**:

- All existing dependencies are installed
- No dependency conflicts

---

## Step 3: Create Service Dockerfile

**File**: `scraper/Dockerfile.service` (new file)

**Action**: Create Dockerfile specifically for service mode

```dockerfile
FROM mcr.microsoft.com/playwright:v1.56.1-focal

# Install Bun
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

# Set working directory
WORKDIR /app

# Copy dependency files
COPY package.json bun.lock ./

# Install dependencies
RUN bun install

# Copy source code
COPY src ./src
COPY tsconfig.json ./

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3000/health || exit 1

# Run server
CMD ["bun", "run", "src/server.ts"]
```

**Tasks**:

- [x] Create `scraper/Dockerfile.service`
- [ ] Test build: `docker build -f Dockerfile.service -t link-mgmt-scraper-service:latest .`
- [ ] Test run: `docker run -p 3000:3000 link-mgmt-scraper-service:latest`
- [ ] Test health check: `curl http://localhost:3000/health`

**Acceptance Criteria**:

- Service image builds successfully
- Service starts and listens on port 3000
- Health check works

---

## Step 4: Implement Go HTTP Client

**File**: `link-mgmt-go/pkg/scraper/client.go` (new file)

**Action**: Create HTTP client for scraper service

```go
package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ScraperService struct {
	baseURL string
	client  *http.Client
}

func NewScraperService(baseURL string) *ScraperService {
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	return &ScraperService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type ScrapeRequest struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout,omitempty"`
}

type ScrapeResponse struct {
	Success     bool   `json:"success"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Text        string `json:"text"`
	ExtractedAt string `json:"extracted_at"`
	Error       string `json:"error,omitempty"`
}

// CheckHealth verifies the service is available
func (s *ScraperService) CheckHealth() error {
	resp, err := s.client.Get(s.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("service not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// Scrape scrapes a single URL
func (s *ScraperService) Scrape(url string, timeout int) (*ScrapeResponse, error) {
	reqBody := ScrapeRequest{
		URL:     url,
		Timeout: timeout,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.client.Post(
		s.baseURL+"/scrape",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call scraper service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper service error (status %d): %s", resp.StatusCode, string(body))
	}

	var result ScrapeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

```

**Tasks**:

- [x] Create `link-mgmt-go/pkg/scraper/client.go`
- [x] Implement `ScraperService` struct
- [x] Implement `CheckHealth()` method
- [x] Implement `Scrape()` method
- [x] Add error handling
- [x] Code compiles successfully
- [ ] Test HTTP client locally (requires running service)

**Acceptance Criteria**:

- [x] Code compiles without errors ✅
- [ ] Can connect to service (testing pending)
- [ ] Can scrape URLs via HTTP (testing pending)
- [x] Error handling is robust ✅

---

## Step 5: Testing Service Mode

**Action**: Comprehensive testing of service mode

**Test Cases**:

1. **Service Health**:

   ```bash
   curl http://localhost:3000/health
   ```

   - Should return 200 OK
   - Should show initialized status

2. **Scrape Single URL**:

   ```bash
   curl -X POST http://localhost:3000/scrape \
     -H "Content-Type: application/json" \
     -d '{"url": "https://example.com"}'
   ```

   - Should return scraping results
   - Should include title and text content

3. **Batch Scrape**:

   ```bash
   curl -X POST http://localhost:3000/scrape/batch \
     -H "Content-Type: application/json" \
     -d '{"urls": ["https://example.com", "https://example.org"]}'
   ```

   - Should return results for all URLs
   - Should handle errors gracefully

4. **Service with HTTP Client**:
   - Use Go HTTP client from Step 4
   - Verify connection and scraping work
   - Test error handling

**Tasks**:

- [ ] Test all scenarios above
- [ ] Test error cases
- [ ] Test concurrent requests
- [ ] Test service restart
- [ ] Document any issues

**Acceptance Criteria**:

- All test cases pass
- Service mode works correctly
- HTTP API responds correctly
- Performance is acceptable

---

## Summary

**Implementation Status**: ✅ **Complete** (Testing Pending)

**Deliverables**:

- ✅ HTTP service for scraping (`scraper/src/server.ts`)
- ✅ Service Dockerfile (`scraper/Dockerfile.service`)
- ✅ Go HTTP client (`link-mgmt-go/pkg/scraper/client.go`)
- ✅ TypeScript fixes (type-only imports, added @types/jsdom)

**Files Created**:

- `scraper/src/server.ts` - HTTP service implementation using Bun's native server
- `scraper/Dockerfile.service` - Docker configuration for service deployment
- `link-mgmt-go/pkg/scraper/client.go` - Go HTTP client for scraper service

**Files Modified**:

- `scraper/package.json` - Added `@types/jsdom` to devDependencies
- `scraper/src/browser.ts` - Fixed TypeScript imports (type-only imports for Playwright types)

**Success Metrics**:

- [x] Code implemented and compiles successfully
- [x] TypeScript type checking passes
- [x] Go code compiles without errors
- [ ] Service starts and runs correctly (testing pending)
- [ ] Can scrape via HTTP API (testing pending)
- [x] HTTP client code implemented correctly

**Benefits of Service Mode**:

- Better resource utilization (shared browser instance)
- Can handle multiple concurrent requests
- Easier to scale independently
- Better for high-volume scraping

**When to Use Service Mode**:

- High-volume scraping scenarios
- Multiple users/applications need scraping
- Want to scale scraping independently
- Need better resource efficiency
