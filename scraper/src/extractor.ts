import { JSDOM } from "jsdom";
import { Readability } from "@mozilla/readability";
import type { ExtractedContent } from "./types";
import { logger } from "./logger";
import { categorizeError, ScrapeErrorType, type ScrapeError } from "./errors";

export function cleanupText(text: string): string {
  return text.replace(/^[\n\r]+|[\n\r]+$/g, "").trim();
}

export async function extractMainContent(
  html: string,
  url: string
): Promise<ExtractedContent | null> {
  try {
    const dom = new JSDOM(html, { url });
    const reader = new Readability(dom.window.document);
    const article = reader.parse();

    if (!article) {
      // Extraction failed - readability couldn't parse the content
      // This is not necessarily an error, just means the page structure
      // doesn't match readability's expectations
      logger.debug("Readability extraction returned null", { url });
      return null;
    }

    return {
      title: article.title || "",
      text: cleanupText(article.textContent || ""),
    };
  } catch (error) {
    // Categorize extraction errors
    const scrapeError = categorizeError(error);
    logger.warn("Extraction failed", {
      url,
      error_type: scrapeError.type,
      error_message: scrapeError.message,
    });
    return null;
  }
}

/**
 * Creates an extraction error for cases where extraction explicitly fails
 */
export function createExtractionError(
  message: string,
  cause?: Error | unknown
): ScrapeError {
  return {
    type: ScrapeErrorType.EXTRACTION,
    message,
    retryable: false,
    statusCode: 500,
    cause,
  };
}
