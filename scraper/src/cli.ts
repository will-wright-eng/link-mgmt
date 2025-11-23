import { BrowserManager } from "./browser";
import { extractMainContent } from "./extractor";
import { formatResult, formatResults } from "./output";
import type { ExtractionResult, CliOptions } from "./types";
import { readFileSync } from "fs";
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

function parseArgs(): CliOptions {
  const args = process.argv.slice(2);
  const options: CliOptions = {
    timeout: 10000,
    headless: true,
    format: "jsonl",
  };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === "--input" && args[i + 1]) {
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
      options.timeout = parseInt(args[i + 1], 10);
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

async function main(): Promise<void> {
  const options = parseArgs();
  const urls = await getUrls(options);

  if (urls.length === 0) {
    console.error("No valid URLs provided");
    process.exit(1);
  }

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
