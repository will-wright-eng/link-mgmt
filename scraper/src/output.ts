import type { ExtractionResult, CliOptions } from "./types";

export function formatResult(
  result: ExtractionResult,
  format: CliOptions["format"] = "jsonl",
): string {
  switch (format) {
    case "json":
      return JSON.stringify(result, null, 2);
    case "text":
      if (result.error) {
        return `Error for ${result.url}: ${result.error}\n`;
      }
      return `${result.title}\n\n${result.text}\n\n---\n`;
    case "jsonl":
    default:
      return JSON.stringify(result);
  }
}

export function formatResults(
  results: ExtractionResult[],
  format: CliOptions["format"] = "jsonl",
): string {
  if (format === "json") {
    return JSON.stringify(results, null, 2);
  }

  return results.map((result) => formatResult(result, format)).join("\n");
}
