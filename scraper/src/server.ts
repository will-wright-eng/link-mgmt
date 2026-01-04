import { BrowserManager, extractScrapeError } from "./browser";
import { extractMainContent, createExtractionError } from "./extractor";
import { logger } from "./logger";
import type { ExtractionResult, ScrapeResponse } from "./types";
import {
  categorizeError,
  ScrapeErrorType,
  type ScrapeError,
  isBlockedContent,
  createScrapeError,
} from "./errors";

let manager: BrowserManager | null = null;
let initialized = false;
let server: ReturnType<typeof Bun.serve> | null = null;
let isShuttingDown = false;

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

// Graceful shutdown handler
async function shutdown(signal: string) {
  if (isShuttingDown) {
    logger.warn(`${signal} received but shutdown already in progress`);
    return;
  }
  isShuttingDown = true;

  logger.info(`${signal} received, shutting down gracefully`);

  try {
    // Stop accepting new connections
    if (server) {
      logger.info("Stopping HTTP server");
      server.stop();
    }

    // Cleanup browser resources
    if (manager) {
      logger.info("Cleaning up browser");
      await manager.cleanup();
    }

    logger.info("Shutdown complete");
    process.exit(0);
  } catch (error) {
    logger.error("Error during shutdown", error);
    process.exit(1);
  }
}

// Register signal handlers
process.on("SIGTERM", () => {
  shutdown("SIGTERM").catch((error) => {
    logger.error("Error in SIGTERM handler", error);
    process.exit(1);
  });
});
process.on("SIGINT", () => {
  shutdown("SIGINT").catch((error) => {
    logger.error("Error in SIGINT handler", error);
    process.exit(1);
  });
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
    return sendJSON(
      {
        success: false,
        error: "URL is required",
        error_type: ScrapeErrorType.INVALID_URL,
        retryable: false,
      },
      400
    );
  }

  const { url, timeout = 10000 } = body as { url: string; timeout?: number };

  if (!url || typeof url !== "string") {
    logger.warn("Invalid scrape request: URL is not a string", { url });
    return sendJSON(
      {
        success: false,
        url,
        error: "URL is required",
        error_type: ScrapeErrorType.INVALID_URL,
        retryable: false,
      },
      400
    );
  }

  if (!initialized || !manager) {
    const scrapeError: ScrapeError = {
      type: ScrapeErrorType.BROWSER_ERROR,
      message: "Browser not initialized",
      retryable: true,
      statusCode: 503,
    };
    logger.error(
      "Scrape request received but browser not initialized",
      undefined,
      {
        url,
        error_type: scrapeError.type,
      }
    );
    return sendJSON(
      {
        success: false,
        url,
        error: scrapeError.message,
        error_type: scrapeError.type,
        retryable: scrapeError.retryable,
      },
      scrapeError.statusCode
    );
  }

  logger.info("Starting scrape", { url, timeout });

  try {
    const scrapeStartTime = Date.now();
    const html = await manager.extractFromUrl(url, timeout);
    const extractionStartTime = Date.now();
    const extracted = await extractMainContent(html, url);
    const extractionDuration = Date.now() - extractionStartTime;

    if (!extracted) {
      const extractionError = createExtractionError(
        "Failed to extract content from page"
      );
      logger.warn("Failed to extract content", {
        url,
        extractionDuration: `${extractionDuration}ms`,
        error_type: extractionError.type,
      });
      return sendJSON(
        {
          success: false,
          url,
          error: extractionError.message,
          error_type: extractionError.type,
          retryable: extractionError.retryable,
        },
        extractionError.statusCode
      );
    }

    // Check if extracted content indicates a blocking/security message
    if (isBlockedContent(extracted.text || "", extracted.title || "")) {
      const blockedError = createScrapeError(
        ScrapeErrorType.BLOCKED,
        "Access blocked by network security"
      );
      logger.warn("Content indicates blocking", {
        url,
        error_type: blockedError.type,
        title: extracted.title || undefined,
      });
      return sendJSON(
        {
          success: false,
          url,
          error: blockedError.message,
          error_type: blockedError.type,
          retryable: blockedError.retryable,
        },
        blockedError.statusCode
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
    // Try to extract categorized error if it exists
    let scrapeError = extractScrapeError(error);
    if (!scrapeError) {
      // Categorize the error
      scrapeError = categorizeError(error);
    }

    logger.error("Scrape failed", error, {
      url,
      error_type: scrapeError.type,
      retryable: scrapeError.retryable,
    });

    return sendJSON(
      {
        success: false,
        url,
        error: scrapeError.message,
        error_type: scrapeError.type,
        retryable: scrapeError.retryable,
      },
      scrapeError.statusCode
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
    logger.error(
      "Batch scrape request received but browser not initialized",
      undefined,
      {
        urlCount: urls.length,
      }
    );
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

      if (!extracted) {
        const extractionError = createExtractionError(
          "Failed to extract content from page"
        );
        results.push({
          url,
          title: "",
          text: "",
          extracted_at: new Date().toISOString(),
          error: extractionError.message,
          error_type: extractionError.type,
          retryable: extractionError.retryable,
        });
        logger.debug("Batch scrape URL failed extraction", {
          url,
          duration: `${urlDuration}ms`,
          error_type: extractionError.type,
        });
      } else if (
        isBlockedContent(extracted.text || "", extracted.title || "")
      ) {
        // Check if extracted content indicates blocking
        const blockedError = createScrapeError(
          ScrapeErrorType.BLOCKED,
          "Access blocked by network security"
        );
        results.push({
          url,
          title: "",
          text: "",
          extracted_at: new Date().toISOString(),
          error: blockedError.message,
          error_type: blockedError.type,
          retryable: blockedError.retryable,
        });
        logger.debug("Batch scrape URL blocked", {
          url,
          duration: `${urlDuration}ms`,
          error_type: blockedError.type,
        });
      } else {
        results.push({
          url,
          title: extracted.title || "",
          text: extracted.text || "",
          extracted_at: new Date().toISOString(),
          error: null,
        });
        logger.debug("Batch scrape URL completed", {
          url,
          success: true,
          duration: `${urlDuration}ms`,
        });
      }
    } catch (error) {
      // Try to extract categorized error if it exists
      let scrapeError = extractScrapeError(error);
      if (!scrapeError) {
        // Categorize the error
        scrapeError = categorizeError(error);
      }

      logger.warn("Batch scrape URL failed", {
        url,
        error: scrapeError.message,
        error_type: scrapeError.type,
        retryable: scrapeError.retryable,
      });
      results.push({
        url,
        title: "",
        text: "",
        extracted_at: new Date().toISOString(),
        error: scrapeError.message,
        error_type: scrapeError.type,
        retryable: scrapeError.retryable,
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
  server = Bun.serve({
    port,
    fetch: (request: Request) => logRequest(request, handleRequest),
  });

  logger.info("Scraper service started", { port: server.port });
}

startServer().catch((error) => {
  logger.error("Failed to start server", error);
  process.exit(1);
});
