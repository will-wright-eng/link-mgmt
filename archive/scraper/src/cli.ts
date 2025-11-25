import { BrowserManager } from "./browser";
import { extractMainContent } from "./extractor";
import { formatResult, formatResults } from "./output";
import type { ExtractionResult, CliOptions } from "./types";
import { loadConfig } from "./config";
import { ApiClient } from "./api-client";
import { askToForceUpdateLink, askToUpdateLink, selectLink } from "./selector";
import { readFileSync } from "fs";
import { join } from "path";
import { homedir } from "os";
import { stdin } from "process";

async function readUrlsFromStdin(): Promise<string[]> {
  return new Promise((resolve) => {
    const chunks: string[] = [];
    stdin.setEncoding("utf8");

    stdin.on("data", (chunk: string) => {
      chunks.push(chunk);
    });

    stdin.on("end", () => {
      const input = chunks.join("");
      const urls = input
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line.length > 0 && isValidUrl(line));
      resolve(urls);
    });
  });
}

function isValidUrl(url: string): boolean {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

function parseArgs(): CliOptions & { checkConfig?: boolean } {
  const args = process.argv.slice(2);
  const options: CliOptions & { checkConfig?: boolean } = {
    timeout: 10000,
    headless: true,
    format: "jsonl",
  };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (!arg) continue;

    if (arg === "--config" || arg === "--check-config") {
      options.checkConfig = true;
    } else if (arg === "--input" && args[i + 1]) {
      options.input = args[i + 1];
      i++;
    } else if (arg === "--output" && args[i + 1]) {
      options.output = args[i + 1];
      i++;
    } else if (arg === "--format" && args[i + 1]) {
      const format = args[i + 1] as CliOptions["format"];
      if (format === "jsonl" || format === "json" || format === "text") {
        options.format = format;
      }
      i++;
    } else if (arg === "--timeout" && args[i + 1]) {
      const timeoutValue = args[i + 1];
      if (timeoutValue) {
        options.timeout = parseInt(timeoutValue, 10);
      }
      i++;
    } else if (arg === "--headless" && args[i + 1]) {
      options.headless = args[i + 1] === "true";
      i++;
    } else if (arg.startsWith("http://") || arg.startsWith("https://")) {
      // URL argument
      if (!options.input) {
        options.input = arg;
      }
    }
  }

  return options;
}

function checkConfig(): void {
  try {
    const config = loadConfig();
    console.log("✓ Config loaded successfully!");
    console.log("");
    console.log("Configuration:");
    console.log(`  API URL: ${config.apiUrl}`);
    console.log(
      `  API Key: ${config.apiKey.substring(0, 8)}...${config.apiKey.substring(config.apiKey.length - 4)}`
    );
    console.log("");
    console.log("Config source:");
    if (process.env.LNK_API_URL && process.env.LNK_API_KEY) {
      console.log("  Environment variables (LNK_API_URL, LNK_API_KEY)");
    } else {
      const configPath = join(homedir(), ".config", "lnk", "config.toml");
      console.log(`  Config file: ${configPath}`);
    }
  } catch (error) {
    console.error("✗ Failed to load config:");
    if (error instanceof Error) {
      console.error(`  ${error.message}`);
    } else {
      console.error(`  ${error}`);
    }
    process.exit(1);
  }
}

async function getUrls(options: CliOptions): Promise<string[]> {
  // If input is a file path, read from file
  if (options.input) {
    try {
      const content = readFileSync(options.input, "utf-8");
      const urls = content
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line.length > 0 && isValidUrl(line));
      return urls;
    } catch (error) {
      // If file read fails, treat as URL
      if (isValidUrl(options.input)) {
        return [options.input];
      }
      throw error;
    }
  }

  // Check if URLs are provided as arguments
  const args = process.argv.slice(2);
  const urlArgs = args.filter(
    (arg) =>
      (arg.startsWith("http://") || arg.startsWith("https://")) &&
      isValidUrl(arg),
  );

  if (urlArgs.length > 0) {
    return urlArgs;
  }

  // Try reading from stdin
  if (!process.stdin.isTTY) {
    return await readUrlsFromStdin();
  }

  return [];
}

async function interactiveMode(options: CliOptions): Promise<void> {
  // Load config
  let config;
  try {
    config = loadConfig();
  } catch (error) {
    console.error("Failed to load configuration:");
    if (error instanceof Error) {
      console.error(`  ${error.message}`);
    }
    process.exit(1);
  }

  // Create API client
  const apiClient = new ApiClient(config);

  // List links
  console.log("Fetching saved links...");
  let links;
  try {
    links = await apiClient.listLinks();
  } catch (error) {
    console.error("Failed to fetch links:");
    if (error instanceof Error) {
      console.error(`  ${error.message}`);
    }
    process.exit(1);
  }

  if (links.length === 0) {
    console.log("No saved links found.");
    process.exit(0);
  }

  // Select a link
  let selectedLink;
  try {
    selectedLink = await selectLink(links);
  } catch (error) {
    if (error instanceof Error && error.message === "Selection cancelled") {
      console.log("\nCancelled.");
      process.exit(0);
    }
    throw error;
  }

  // Scrape the selected URL
  console.log(`\nScraping ${selectedLink.url}...`);
  const manager = new BrowserManager();
  await manager.initialize(options.headless);

  let extracted;
  try {
    const html = await manager.extractFromUrl(
      selectedLink.url,
      options.timeout
    );
    extracted = await extractMainContent(html, selectedLink.url);
  } catch (error) {
    await manager.cleanup();
    console.error("Failed to scrape URL:");
    if (error instanceof Error) {
      console.error(`  ${error.message}`);
    }
    process.exit(1);
  }

  await manager.cleanup();

  if (!extracted) {
    console.error("Failed to extract content from the page.");
    process.exit(1);
  }

  console.log("✓ Successfully extracted content");
  console.log(`  Title: ${extracted.title || "(no title)"}`);
  console.log(`  Content: ${extracted.text.length} characters`);

  // Update the link in the API
  console.log("\nUpdating link in API...");
  const updateData: { title?: string; text?: string } = {};

  if (extracted.title) {
    updateData.title = extracted.title;
  }

  // Set text field with extracted content (no truncation needed)
  if (extracted.text) {
    updateData.text = extracted.text;
  }

  // display results and get link by ID from API
  console.log("\nGetting link from API...");
  const link = await apiClient.getLink(selectedLink.id);
  console.log(`  ID: ${link.id}`);
  console.log(`  Title: ${link.title}`);
  console.log(`  Description: ${link.description}`);
  console.log(`  Created: ${link.created_at}`);
  console.log(`  Updated: ${link.updated_at}`);
  console.log("");

  console.log("\nData extracted from the page:");
  console.log(`  Title: ${extracted.title}`);
  console.log(`  Content: ${extracted.text.length} characters`);
  console.log("");
  const updateLink = await askToUpdateLink();
  const forceUpdateLink = await askToForceUpdateLink();

  if (updateLink) {
    try {
      const updatedLink = await apiClient.updateLink(selectedLink.id, updateData, forceUpdateLink);
      console.log("✓ Link updated successfully!");
      console.log(`  ID: ${updatedLink.id}`);
      console.log(`  Title: ${updatedLink.title}`);
    } catch (error) {
      console.error("Failed to update link:");
      if (error instanceof Error) {
        console.error(`  ${error.message}`);
      }
      process.exit(1);
    }
  } else {
    console.log("Link not updated.");
    process.exit(0);
  }
  process.exit(0);
}

async function main(): Promise<void> {
  const options = parseArgs();

  // Handle config check command
  if (options.checkConfig) {
    checkConfig();
    return;
  }

  const urls = await getUrls(options);

  // If no URLs provided, enter interactive mode
  if (urls.length === 0) {
    await interactiveMode(options);
    return;
  }

  // Existing URL-based mode (backward compatible)
  const manager = new BrowserManager();
  await manager.initialize(options.headless);

  const results: ExtractionResult[] = [];

  for (const url of urls) {
    try {
      const html = await manager.extractFromUrl(url, options.timeout);
      const extracted = await extractMainContent(html, url);

      const result: ExtractionResult = {
        url,
        title: extracted?.title || "",
        text: extracted?.text || "",
        extracted_at: new Date().toISOString(),
        error: extracted ? null : "Failed to extract content",
      };

      results.push(result);
      console.log(formatResult(result, options.format));
    } catch (error) {
      const result: ExtractionResult = {
        url,
        title: "",
        text: "",
        extracted_at: new Date().toISOString(),
        error: error instanceof Error ? error.message : "Unknown error",
      };

      results.push(result);
      console.log(formatResult(result, options.format));
    }
  }

  await manager.cleanup();

  // Write to output file if specified
  if (options.output) {
    const { writeFileSync } = await import("fs");
    const output = formatResults(results, options.format);
    writeFileSync(options.output, output, "utf-8");
  }
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
