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
function sendJSON(data: unknown, status: number = 200): Response {
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
    return sendJSON({ error: "URL is required" }, 400);
  }

  const { url, timeout = 10000 } = body as { url: string; timeout?: number };

  if (!url || typeof url !== "string") {
    return sendJSON({ error: "URL is required" }, 400);
  }

  if (!initialized || !manager) {
    return sendJSON({ error: "Browser not initialized" }, 503);
  }

  try {
    const html = await manager.extractFromUrl(url, timeout);
    const extracted = await extractMainContent(html, url);

    if (!extracted) {
      return sendJSON(
        {
          success: false,
          url,
          error: "Failed to extract content",
        },
        500
      );
    }

    return sendJSON(
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
    return sendJSON({ error: "urls must be an array" }, 400);
  }

  const { urls, timeout = 10000 } = body as {
    urls: string[];
    timeout?: number;
  };

  if (!initialized || !manager) {
    return sendJSON({ error: "Browser not initialized" }, 503);
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

  return sendJSON({ results }, 200);
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
  return sendJSON({ error: "Not found" }, 404);
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
