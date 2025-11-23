import { chromium, Browser, BrowserContext } from "playwright";

export class BrowserManager {
  private browser: Browser | null = null;
  private context: BrowserContext | null = null;

  async initialize(headless: boolean = true): Promise<void> {
    this.browser = await chromium.launch({ headless });
    this.context = await this.browser.newContext();
  }

  async extractFromUrl(url: string, timeout: number = 10000): Promise<string> {
    if (!this.context) {
      throw new Error("Browser not initialized");
    }

    const page = await this.context.newPage();
    try {
      await page.goto(url, { waitUntil: "networkidle", timeout });
      const content = await page.content();
      return content;
    } finally {
      await page.close();
    }
  }

  async cleanup(): Promise<void> {
    await this.context?.close();
    await this.browser?.close();
  }
}
