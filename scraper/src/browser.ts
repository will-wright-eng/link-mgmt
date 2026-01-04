import { chromium } from "playwright";
import type { Browser, BrowserContext } from "playwright";
import {
  categorizeError,
  createScrapeError,
  ScrapeErrorType,
  type ScrapeError,
} from "./errors";

export class BrowserManager {
  private browser: Browser | null = null;
  private context: BrowserContext | null = null;

  async initialize(headless: boolean = true): Promise<void> {
    try {
      this.browser = await chromium.launch({ headless });
      this.context = await this.browser.newContext();
    } catch (error) {
      const scrapeError = categorizeError(error);
      // Re-throw with categorized error information
      const enhancedError = new Error(scrapeError.message);
      (enhancedError as any).scrapeError = scrapeError;
      throw enhancedError;
    }
  }

  async extractFromUrl(url: string, timeout: number = 10000): Promise<string> {
    if (!this.context) {
      const error = createScrapeError(
        ScrapeErrorType.BROWSER_ERROR,
        "Browser not initialized"
      );
      const enhancedError = new Error(error.message);
      (enhancedError as any).scrapeError = error;
      throw enhancedError;
    }

    // Validate URL format
    try {
      new URL(url);
    } catch {
      const error = createScrapeError(
        ScrapeErrorType.INVALID_URL,
        `Invalid URL: ${url}`
      );
      const enhancedError = new Error(error.message);
      (enhancedError as any).scrapeError = error;
      throw enhancedError;
    }

    const page = await this.context.newPage();
    try {
      await page.goto(url, { waitUntil: "networkidle", timeout });
      const content = await page.content();
      return content;
    } catch (error) {
      // Categorize the error and attach it to the thrown error
      const scrapeError = categorizeError(error);
      const enhancedError = new Error(scrapeError.message);
      (enhancedError as any).scrapeError = scrapeError;
      (enhancedError as any).originalError = error;
      throw enhancedError;
    } finally {
      await page.close();
    }
  }

  async cleanup(): Promise<void> {
    try {
      await this.context?.close();
      await this.browser?.close();
    } catch (error) {
      // Log but don't throw - cleanup errors shouldn't fail the operation
      const scrapeError = categorizeError(error);
      console.warn("Error during browser cleanup:", scrapeError.message);
    }
  }
}

/**
 * Extracts ScrapeError from an error object if it exists
 */
export function extractScrapeError(error: unknown): ScrapeError | null {
  if (error && typeof error === "object" && "scrapeError" in error) {
    return (error as any).scrapeError as ScrapeError;
  }
  return null;
}
