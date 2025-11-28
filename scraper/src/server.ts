import { BrowserManager } from "./browser";
import { extractMainContent } from "./extractor";
import { logger } from "./logger";
import type { ExtractionResult } from "./types";

let manager: BrowserManager | null = null;
let initialized = false;

// Initialize browser on startup
async function initBrowser() {
  try {
    logger.info("Initializing browser");
    manager = new BrowserManager();
    await manager.initialize(true); // headless mode
    initialized = true;
    logger.info("Browser initialized successfully");
  } catch (error) {
    logger.error("Failed to initialize browser", error);
    process.exit(1);
  }
}

// Graceful shutdown
process.on("SIGTERM", async () => {
  logger.info("SIGTERM received, shutting down gracefully");
  if (manager) {
    await manager.cleanup();
  }
  process.exit(0);
});

process.on("SIGINT", async () => {
  logger.info("SIGINT received, shutting down gracefully");
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
    logger.warn("Invalid scrape request: missing URL in body");
    return sendJSON({ error: "URL is required" }, 400);
  }

  const { url, timeout = 10000 } = body as { url: string; timeout?: number };

  if (!url || typeof url !== "string") {
    logger.warn("Invalid scrape request: URL is not a string", { url });
    return sendJSON({ error: "URL is required" }, 400);
  }

  if (!initialized || !manager) {
    logger.error("Scrape request received but browser not initialized", undefined, { url });
    return sendJSON({ error: "Browser not initialized" }, 503);
  }

  logger.info("Starting scrape", { url, timeout });

  try {
    const scrapeStartTime = Date.now();
    const html = await manager.extractFromUrl(url, timeout);
    const extractionStartTime = Date.now();
    const extracted = await extractMainContent(html, url);
    const extractionDuration = Date.now() - extractionStartTime;

    if (!extracted) {
      logger.warn("Failed to extract content", { url, extractionDuration: `${extractionDuration}ms` });
      return sendJSON(
        {
          success: false,
          url,
          error: "Failed to extract content",
        },
        500
      );
    }

    const totalDuration = Date.now() - scrapeStartTime;
    logger.info("Scrape completed successfully", {
      url,
      title: extracted.title || undefined,
      contentLength: extracted.text?.length || 0,
      extractionDuration: `${extractionDuration}ms`,
      totalDuration: `${totalDuration}ms`,
    });

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
    logger.error("Scrape failed", error, { url });
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
    logger.warn("Invalid batch scrape request: urls must be an array");
    return sendJSON({ error: "urls must be an array" }, 400);
  }

  const { urls, timeout = 10000 } = body as {
    urls: string[];
    timeout?: number;
  };

  if (!initialized || !manager) {
    logger.error("Batch scrape request received but browser not initialized", undefined, {
      urlCount: urls.length,
    });
    return sendJSON({ error: "Browser not initialized" }, 503);
  }

  logger.info("Starting batch scrape", { urlCount: urls.length, timeout });
  const batchStartTime = Date.now();

  const results: ExtractionResult[] = [];

  for (const url of urls) {
    try {
      const urlStartTime = Date.now();
      const html = await manager.extractFromUrl(url, timeout);
      const extracted = await extractMainContent(html, url);
      const urlDuration = Date.now() - urlStartTime;

      results.push({
        url,
        title: extracted?.title || "",
        text: extracted?.text || "",
        extracted_at: new Date().toISOString(),
        error: extracted ? null : "Failed to extract content",
      });

      logger.debug("Batch scrape URL completed", {
        url,
        success: !!extracted,
        duration: `${urlDuration}ms`,
      });
    } catch (error) {
      logger.warn("Batch scrape URL failed", { url, error: error instanceof Error ? error.message : String(error) });
      results.push({
        url,
        title: "",
        text: "",
        extracted_at: new Date().toISOString(),
        error: error instanceof Error ? error.message : "Unknown error",
      });
    }
  }

  const batchDuration = Date.now() - batchStartTime;
  const successCount = results.filter((r) => !r.error).length;
  const errorCount = results.filter((r) => r.error).length;

  logger.info("Batch scrape completed", {
    urlCount: urls.length,
    successCount,
    errorCount,
    totalDuration: `${batchDuration}ms`,
  });

  return sendJSON({ results }, 200);
}

// Request logging middleware
async function logRequest(
  request: Request,
  handler: (request: Request) => Promise<Response>
): Promise<Response> {
  const startTime = Date.now();
  const url = new URL(request.url);
  const method = request.method;
  const path = url.pathname;

  logger.info("Request received", {
    method,
    path,
    userAgent: request.headers.get("user-agent") || undefined,
  });

  try {
    const response = await handler(request);
    const duration = Date.now() - startTime;

    logger.info("Request completed", {
      method,
      path,
      status: response.status,
      duration: `${duration}ms`,
    });

    return response;
  } catch (error) {
    const duration = Date.now() - startTime;

    logger.error("Request failed", error, {
      method,
      path,
      duration: `${duration}ms`,
    });

    throw error;
  }
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
    fetch: (request: Request) => logRequest(request, handleRequest),
  });

  logger.info("Scraper service started", { port: server.port });
}

startServer().catch((error) => {
  logger.error("Failed to start server", error);
  process.exit(1);
});
