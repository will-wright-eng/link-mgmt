import { readFileSync } from "fs";
import { join } from "path";
import { homedir } from "os";

export interface Config {
  apiUrl: string;
  apiKey: string;
}

/**
 * Simple TOML parser for config file.
 * Handles sections and key-value pairs with quoted/unquoted strings.
 */
function parseToml(content: string): Record<string, Record<string, string>> {
  const result: Record<string, Record<string, string>> = {};
  let currentSection = "";

  for (const line of content.split("\n")) {
    const trimmed = line.trim();

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    // Parse section header: [section]
    const sectionMatch = trimmed.match(/^\[([^\]]+)\]$/);
    if (sectionMatch && sectionMatch[1]) {
      currentSection = sectionMatch[1];
      if (!result[currentSection]) {
        result[currentSection] = {};
      }
      continue;
    }

    // Parse key-value pair: key = "value" or key = value
    const kvMatch = trimmed.match(/^([^=]+)=(.*)$/);
    if (kvMatch && kvMatch[1] && kvMatch[2] && currentSection) {
      const section = result[currentSection];
      if (section) {
        const key = kvMatch[1].trim();
        let value = kvMatch[2].trim();

        // Remove quotes if present
        if (
          (value.startsWith('"') && value.endsWith('"')) ||
          (value.startsWith("'") && value.endsWith("'"))
        ) {
          value = value.slice(1, -1);
        }

        section[key] = value;
      }
    }
  }

  return result;
}

/**
 * Load configuration from environment variables or config file.
 * Precedence: env vars > config file > defaults
 */
export function loadConfig(): Config {
  // Try environment variables first
  const envApiUrl = process.env.LNK_API_URL;
  const envApiKey = process.env.LNK_API_KEY;

  if (envApiUrl && envApiKey) {
    return {
      apiUrl: envApiUrl,
      apiKey: envApiKey,
    };
  }

  // Try config file
  const configPath = join(homedir(), ".config", "lnk", "config.toml");

  try {
    const content = readFileSync(configPath, "utf-8");
    const config = parseToml(content);

    const apiSection = config.api || {};
    const apiUrl = apiSection.url || envApiUrl || "http://localhost:8000";
    const apiKey = apiSection.key || envApiKey || "";

    if (!apiKey) {
      throw new Error(
        "API key not found. Please set LNK_API_KEY environment variable " +
          "or configure ~/.config/lnk/config.toml with [api.key]"
      );
    }

    return {
      apiUrl,
      apiKey,
    };
  } catch (error) {
    if (error instanceof Error && (error as NodeJS.ErrnoException).code === "ENOENT") {
      throw new Error(
        `Config file not found at ${configPath}. ` +
          "Please set LNK_API_URL and LNK_API_KEY environment variables, " +
          "or configure ~/.config/lnk/config.toml"
      );
    }
    throw error;
  }
}
