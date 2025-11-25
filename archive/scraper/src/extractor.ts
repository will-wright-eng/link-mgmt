import { JSDOM } from "jsdom";
import { Readability } from "@mozilla/readability";
import type { ExtractedContent } from "./types";

export function cleanupText(text: string): string {
  return text.replace(/^[\n\r]+|[\n\r]+$/g, '').trim();
}

export async function extractMainContent(
  html: string,
  url: string,
): Promise<ExtractedContent | null> {
  try {
    const dom = new JSDOM(html, { url });
    const reader = new Readability(dom.window.document);
    const article = reader.parse();

    if (!article) {
      return null;
    }

    return {
      title: article.title || "",
      text: cleanupText(article.textContent || ""),
    };
  } catch (error) {
    console.error("Extraction failed:", error);
    return null;
  }
}
