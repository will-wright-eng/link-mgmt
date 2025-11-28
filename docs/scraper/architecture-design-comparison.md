# Scraper Architecture Design: Container vs Service Approaches

## Overview

This document evaluates two architectural approaches for integrating Playwright-based web scraping with the Go CLI, given that bundling Playwright is not feasible due to native dependencies and large browser binaries.

## Current Architecture

The scraper is a TypeScript/Bun application that:

- Uses Playwright to render web pages
- Extracts main content using Mozilla Readability
- Can run in interactive mode (selects links from API) or batch mode (processes URLs)
- Communicates with the link management API

**Current Challenge**: Playwright cannot be bundled, making distribution and deployment complex.

## Approach Comparison

### Option 1: Container-Based Deployment

**Concept**: Package the scraper in a Docker container with Playwright browsers pre-installed. The Go CLI executes scraping operations by running Docker containers.

#### Architecture

```
┌─────────────────┐
│   Go CLI        │
│                 │
│  ┌───────────┐  │
│  │ Scraper   │  │──docker run──>┌──────────────────┐
│  │ Executor  │  │               │ Scraper Container │
│  └───────────┘  │               │                   │
│                 │               │ • Bun runtime     │
│                 │               │ • Playwright      │
│                 │               │ • Browsers        │
│                 │               │ • Scraper code    │
│                 │               └──────────────────┘
└─────────────────┘
```

#### Implementation Details

**1. Scraper Container**

Create `scraper/Dockerfile`:

```dockerfile
FROM mcr.microsoft.com/playwright:v1.56.1-focal

# Install Bun
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

# Copy scraper code
WORKDIR /app
COPY package.json bun.lock ./
COPY src ./src
COPY tsconfig.json ./

# Install dependencies
RUN bun install

# Entry point
ENTRYPOINT ["bun", "run", "src/cli.ts"]
```

**2. Go CLI Integration**

Create `link-mgmt-go/pkg/cli/scraper_docker.go`:

```go
package cli

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "strings"
)

func (a *App) RunScraperDocker(args []string) error {
    // Build image if needed
    imageName := "link-mgmt-scraper:latest"

    // Check if image exists
    cmd := exec.Command("docker", "images", "-q", imageName)
    output, _ := cmd.Output()
    if len(output) == 0 {
        // Build image
        fmt.Println("Building scraper Docker image...")
        buildCmd := exec.Command("docker", "build", "-t", imageName, "../scraper")
        buildCmd.Stdout = os.Stdout
        buildCmd.Stderr = os.Stderr
        if err := buildCmd.Run(); err != nil {
            return fmt.Errorf("failed to build Docker image: %w", err)
        }
    }

    // Prepare Docker run command
    dockerArgs := []string{
        "run",
        "--rm", // Remove container after execution
        "-i",   // Interactive (for stdin)
    }

    // Pass environment variables
    if a.cfg != nil {
        if a.cfg.CLI.APIBaseURL != "" {
            dockerArgs = append(dockerArgs, "-e", "LNK_API_URL="+a.cfg.CLI.APIBaseURL)
        }
        if a.cfg.CLI.APIKey != "" {
            dockerArgs = append(dockerArgs, "-e", "LNK_API_KEY="+a.cfg.CLI.APIKey)
        }
    }

    // Add image name and scraper arguments
    dockerArgs = append(dockerArgs, imageName)
    dockerArgs = append(dockerArgs, args...)

    // Execute
    cmd = exec.Command("docker", dockerArgs...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    return cmd.Run()
}
```

**3. Alternative: Use Official Playwright Image**

Instead of building custom image, use official image and mount scraper code:

```go
dockerArgs := []string{
    "run", "--rm", "-i",
    "-v", fmt.Sprintf("%s:/app", scraperPath), // Mount scraper code
    "-w", "/app",
    "mcr.microsoft.com/playwright:v1.56.1-focal",
    "bun", "run", "src/cli.ts",
}
```

#### Pros

✅ **Complete isolation**: All dependencies (Bun, Playwright, browsers) in one container
✅ **Consistent environment**: Works identically across all platforms
✅ **No local installation**: Users don't need Bun or Playwright browsers installed
✅ **Easy distribution**: Single Docker image contains everything
✅ **Official images**: Microsoft maintains Playwright Docker images
✅ **Scalable**: Can run multiple containers in parallel
✅ **Version control**: Pin specific Playwright/browser versions

#### Cons

❌ **Docker requirement**: Users must have Docker installed and running
❌ **Performance overhead**: Container startup adds latency (~1-2 seconds)
❌ **Resource usage**: Containers consume more memory/CPU
❌ **Network complexity**: May need to configure Docker networking
❌ **Development friction**: Slower iteration cycle (rebuild image for changes)
❌ **Image size**: Docker images are large (~500MB-1GB with browsers)

#### Use Cases

- Production deployments
- CI/CD pipelines
- Multi-user environments
- When consistency is critical
- When users can't install dependencies locally

---

### Option 2: Separate Browser Service

**Concept**: Run Playwright in a dedicated service (container or process) that exposes an HTTP API. The Go CLI makes HTTP requests to this service for scraping operations.

#### Architecture

```
┌─────────────────┐         HTTP/REST         ┌──────────────────────┐
│   Go CLI        │──────────────────────────>│  Scraper Service     │
│                 │                           │                      │
│  ┌───────────┐  │                           │  • HTTP API          │
│  │ HTTP      │  │<──────────────────────────│  • Playwright        │
│  │ Client    │  │      JSON Response        │  • Browsers          │
│  └───────────┘  │                           │  • Container/Process  │
│                 │                           └──────────────────────┘
└─────────────────┘
```

#### Implementation Details

**1. Scraper Service API**

Create `scraper/src/server.ts`:

```typescript
import express from "express";
import { BrowserManager } from "./browser";
import { extractMainContent } from "./extractor";

const app = express();
app.use(express.json());

const manager = new BrowserManager();
let initialized = false;

// Initialize browser on startup
async function init() {
  await manager.initialize(true);
  initialized = true;
}
init();

// Health check
app.get("/health", (req, res) => {
  res.json({ status: "ok", initialized });
});

// Scrape endpoint
app.post("/scrape", async (req, res) => {
  const { url, timeout = 10000 } = req.body;

  if (!url) {
    return res.status(400).json({ error: "URL is required" });
  }

  try {
    const html = await manager.extractFromUrl(url, timeout);
    const extracted = await extractMainContent(html, url);

    res.json({
      success: true,
      url,
      title: extracted?.title || "",
      text: extracted?.text || "",
      extracted_at: new Date().toISOString(),
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: error instanceof Error ? error.message : "Unknown error",
    });
  }
});

// Batch scrape endpoint
app.post("/scrape/batch", async (req, res) => {
  const { urls, timeout = 10000 } = req.body;

  if (!Array.isArray(urls)) {
    return res.status(400).json({ error: "urls must be an array" });
  }

  const results = [];
  for (const url of urls) {
    try {
      const html = await manager.extractFromUrl(url, timeout);
      const extracted = await extractMainContent(html, url);

      results.push({
        url,
        title: extracted?.title || "",
        text: extracted?.text || "",
        extracted_at: new Date().toISOString(),
        error: null,
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

  res.json({ results });
});

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`Scraper service listening on port ${port}`);
});
```

**2. Service Dockerfile**

Create `scraper/Dockerfile.service`:

```dockerfile
FROM mcr.microsoft.com/playwright:v1.56.1-focal

RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:${PATH}"

WORKDIR /app
COPY package.json bun.lock ./
COPY src ./src
COPY tsconfig.json ./

RUN bun install

EXPOSE 3000
CMD ["bun", "run", "src/server.ts"]
```

**3. Go CLI HTTP Client**

Create `link-mgmt-go/pkg/cli/scraper_service.go`:

```go
package cli

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

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("scraper service error: %s", string(body))
    }

    var result ScrapeResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &result, nil
}

func (a *App) RunScraperService(url string) error {
    serviceURL := os.Getenv("SCRAPER_SERVICE_URL")
    if serviceURL == "" {
        serviceURL = "http://localhost:3000"
    }

    service := NewScraperService(serviceURL)

    // Check health
    resp, err := http.Get(serviceURL + "/health")
    if err != nil {
        return fmt.Errorf("scraper service not available at %s: %w", serviceURL, err)
    }
    resp.Body.Close()

    // Scrape
    result, err := service.Scrape(url, 10000)
    if err != nil {
        return err
    }

    if !result.Success {
        return fmt.Errorf("scraping failed: %s", result.Error)
    }

    fmt.Printf("Title: %s\n", result.Title)
    fmt.Printf("Content: %d characters\n", len(result.Text))

    return nil
}
```

**4. Service Management**

Add commands to start/stop service:

```go
// Start service (runs in background)
func (a *App) StartScraperService() error {
    cmd := exec.Command("docker", "run", "-d",
        "--name", "link-mgmt-scraper",
        "-p", "3000:3000",
        "link-mgmt-scraper-service:latest",
    )
    return cmd.Run()
}

// Stop service
func (a *App) StopScraperService() error {
    cmd := exec.Command("docker", "stop", "link-mgmt-scraper")
    return cmd.Run()
}
```

#### Pros

✅ **Separation of concerns**: Scraper runs independently
✅ **Reusability**: Multiple clients can use the same service
✅ **Scalability**: Can scale service independently (multiple instances)
✅ **Resource efficiency**: Single browser instance can handle multiple requests
✅ **Language agnostic**: Any language can call the HTTP API
✅ **Monitoring**: Easy to add metrics, logging, rate limiting
✅ **Development**: Can develop/test scraper independently
✅ **Production ready**: Can deploy service separately with proper infrastructure

#### Cons

❌ **Additional complexity**: Need to manage service lifecycle
❌ **Network dependency**: Requires network connection (even if localhost)
❌ **Latency**: HTTP overhead (though minimal)
❌ **State management**: Need to handle service startup/shutdown
❌ **Error handling**: Network errors, service unavailable, etc.
❌ **Security**: Need to secure HTTP endpoint (auth, rate limiting)
❌ **Development setup**: More moving parts during development

#### Use Cases

- Microservices architecture
- Multiple applications need scraping
- High-volume scraping (can pool resources)
- When scraper needs to run on different infrastructure
- When you want to scale scraping independently

---

## Comparison Matrix

| Aspect | Container-Based | Browser Service |
|--------|----------------|-----------------|
| **Complexity** | Medium | Higher |
| **Docker Required** | Yes | Yes (for service) |
| **Performance** | Slower (container startup) | Faster (persistent service) |
| **Resource Usage** | Higher (per execution) | Lower (shared instance) |
| **Scalability** | Good (parallel containers) | Excellent (load balancing) |
| **Development** | Slower (rebuild images) | Faster (hot reload service) |
| **Distribution** | Single image | Image + client code |
| **Network** | No (local execution) | Yes (HTTP) |
| **State** | Stateless | Stateful (browser pool) |
| **Best For** | Simple deployments | Production, scale |

## Hybrid Approach

**Option 3: Container with Service Mode**

Combine both: Container can run in two modes:

1. **CLI mode**: Direct execution (like Option 1)
2. **Service mode**: HTTP server (like Option 2)

```dockerfile
# Support both modes
ENTRYPOINT []
CMD ["bun", "run", "src/cli.ts"]
# Or: CMD ["bun", "run", "src/server.ts"]
```

Go CLI can:

- Run container directly for one-off scraping
- Start service container for repeated operations
- Detect if service is running, use it; otherwise run container

## Recommendation

### For Development: **Container-Based (Option 1)**

- Simpler to implement
- No service management needed
- Works well for CLI tool usage patterns
- Easy to test and debug

### For Production: **Browser Service (Option 2)**

- Better resource utilization
- Can handle high-volume scraping
- Easier to monitor and scale
- More production-ready architecture

### For Flexibility: **Hybrid Approach (Option 3)**

- Best of both worlds
- CLI mode for simple use cases
- Service mode for production
- Single container image

## Implementation Plan

### Phase 1: Container-Based (Quick Win)

1. Create `scraper/Dockerfile`
2. Implement `scraper_docker.go` in Go CLI
3. Add `--scrape` commands that use Docker
4. Test with local Docker

### Phase 2: Add Service Mode (If Needed)

1. Create `scraper/src/server.ts`
2. Create `scraper/Dockerfile.service`
3. Implement `scraper_service.go` in Go CLI
4. Add service management commands (`start-service`, `stop-service`)
5. Auto-detect service, fallback to container

### Phase 3: Production Hardening

1. Add health checks
2. Add retry logic
3. Add rate limiting (service mode)
4. Add monitoring/metrics
5. Add authentication (if service exposed)

## Next Steps

1. **Decide on approach** based on use case
2. **Start with Option 1** (container-based) for simplicity
3. **Evaluate need for Option 2** (service) based on usage patterns
4. **Consider Option 3** (hybrid) for maximum flexibility
