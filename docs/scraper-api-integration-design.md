# Scraper API Integration Design Document

## Overview

This document describes the design for integrating the scraper with the link management API. The scraper will provide an interactive workflow where users can select a saved link, scrape its content, and automatically update it with the extracted metadata.

## Goals

- Provide a simple, interactive workflow for scraping saved links
- Automatically post scraped content back to the API
- Reuse existing API credentials from the Rust CLI tool
- Keep the implementation simple and focused

## Workflow

1. **List saved links** from the API
2. **User selects one link** interactively
3. **Scrape the selected URL** to extract title and content
4. **Automatically update the link** in the API with scraped metadata

## Current State

### Scraper

The scraper currently:

- Takes URLs as input (file, stdin, or arguments)
- Extracts content using Playwright and Readability
- Outputs extraction results in JSON/JSONL/text format

**Output format**:

```typescript
interface ExtractionResult {
  url: string;
  title: string;
  text: string;
  extracted_at: string;
  error: string | null;
}
```

### API Endpoints

The link API provides:

- **GET `/api/links`** - List all links for the current user
    - Returns: `LinkRead[]` with `id`, `url`, `title`, `description`, etc.
    - Authentication: `X-API-Key` header

- **PATCH `/api/links/{link_id}`** (to be implemented) - Update a link
    - Request Body: `{ title?: string, description?: string }`
    - Authentication: `X-API-Key` header

- **POST `/api/links`** - Create a new link (fallback if update not available)
    - Request Body: `{ url: string, title?: string, description?: string }`
    - Authentication: `X-API-Key` header

## Requirements

### Functional Requirements

1. **List Links**
   - Fetch all saved links from the API
   - Display them in an interactive list
   - Show URL and existing title (if any)

2. **Interactive Selection**
   - Allow user to select one link from the list
   - Simple prompt-based selection (no complex UI dependencies)

3. **Scrape Selected URL**
   - Use existing scraper functionality
   - Extract title and text content
   - Handle extraction errors gracefully

4. **Update Link in API**
   - Automatically post scraped content back to API
   - Update `title` with extracted title
   - Update `description` with extracted text (truncated if needed)
   - Handle API errors with clear messages

5. **Configuration**
   - Read API credentials from shared config file (`~/.config/lnk/config.toml`)
   - Support environment variable overrides
   - No separate config file needed

### Non-Functional Requirements

1. **Simplicity**
   - Keep the codebase small and focused
   - Minimal dependencies
   - Clear, straightforward implementation

2. **Error Handling**
   - Clear error messages for API failures
   - Graceful handling of extraction failures
   - Informative feedback to user

3. **User Experience**
   - Simple, intuitive workflow
   - Clear progress indicators
   - Helpful error messages

## Design Decisions

### 1. Interactive Selection

**Option A**: Simple numbered list with prompt

- User sees numbered list, types number to select
- No external dependencies
- Works in any terminal

**Option B**: Use a library like `inquirer` or `prompts`

- Better UX with arrow key navigation
- Additional dependency

**Decision**: **Option A** - Simple numbered list for now. Can enhance later if needed.

### 2. Update vs Create

**Option A**: Always update existing link

- Requires PATCH endpoint on API
- More semantically correct

**Option B**: Create new link if update fails

- Works with current API
- May create duplicates

**Decision**: **Option A** - Update existing link. If PATCH endpoint doesn't exist yet, we'll need to add it to the API, or use POST and handle duplicates.

**Note**: For MVP, we can use POST and let the API handle duplicates (if it has unique constraint on user_id + url).

### 3. Data Mapping

**Mapping Strategy**:

- `ExtractionResult.title` → `LinkUpdate.title` (if not empty)
- `ExtractionResult.text` → `LinkUpdate.description` (truncated to 5000 chars)
- Only update if extraction was successful (`error` is null)

### 4. Configuration

**Decision**: Reuse `~/.config/lnk/config.toml` from Rust CLI tool

- Read `api.url` for base URL
- Read `api.key` for API key
- Environment variables can override: `LNK_API_URL`, `LNK_API_KEY`
- Precedence: env vars > config file > defaults

## Implementation Plan

### Phase 1: Core Components

1. **API Client Module** (`src/api-client.ts`)

   ```typescript
   interface ApiClientOptions {
     baseUrl: string;
     apiKey: string;
   }

   interface Link {
     id: string;
     url: string;
     title?: string;
     description?: string;
   }

   class ApiClient {
     async listLinks(): Promise<Link[]>
     async updateLink(linkId: string, data: { title?: string; description?: string }): Promise<Link>
   }
   ```

2. **Configuration Module** (`src/config.ts`)

   ```typescript
   interface Config {
     apiUrl: string;
     apiKey: string;
   }

   function loadConfig(): Config
   ```

   - Read from `~/.config/lnk/config.toml`
   - Support environment variable overrides
   - Default to `http://localhost:8000` if not configured

3. **Interactive Selection** (`src/selector.ts`)

   ```typescript
   async function selectLink(links: Link[]): Promise<Link>
   ```

   - Display numbered list of links
   - Prompt user for selection
   - Return selected link

### Phase 2: Main Workflow

1. **Update CLI Entry Point** (`src/cli.ts`)
   - Add new interactive mode (default when no URLs provided)
   - Keep existing URL-based mode for backward compatibility
   - Workflow:

     ```
     Load config → List links → Select link → Scrape → Update API
     ```

2. **Integration**
   - After scraping, automatically call API to update link
   - Show success/error feedback
   - Display updated link information

## Configuration

### Config File (`~/.config/lnk/config.toml`)

The scraper reuses the same config file as the Rust CLI tool:

```toml
[api]
url = "http://localhost:8000"
key = "your-api-key-here"

[user]
email = "user@example.com"
```

The scraper reads:

- `api.url` - API base URL
- `api.key` - API key for authentication

### Environment Variables

```bash
LNK_API_URL=http://localhost:8000  # Overrides config file
LNK_API_KEY=your-api-key-here      # Overrides config file
```

### CLI Usage

```bash
# Interactive mode (default - no arguments)
bun run src/cli.ts

# Existing URL-based mode (backward compatible)
bun run src/cli.ts https://example.com

# With explicit config override
LNK_API_URL=http://api.example.com bun run src/cli.ts
```

## Error Handling

### Configuration Errors

- **Missing config file**: Show helpful error with setup instructions
- **Missing API key**: Prompt user to run `lnk auth login` or set `LNK_API_KEY`
- **Invalid API URL**: Show error and suggest checking config

### API Errors

- **401 Unauthorized**: Invalid API key - show error and exit
- **404 Not Found**: Link not found (shouldn't happen, but handle gracefully)
- **400 Bad Request**: Invalid data - show error message
- **Network errors**: Show connection error, suggest checking API URL

### Extraction Errors

- If scraping fails, show error message
- Don't attempt to update API
- Allow user to try again

## Example Usage

### Interactive Workflow

```bash
$ bun run src/cli.ts

Fetching saved links...
Found 5 links:

1. https://example.com/article-1
   Title: Example Article 1

2. https://example.com/article-2
   Title: (no title)

3. https://example.com/article-3
   Title: Another Article

Select a link to scrape (1-5): 2

Scraping https://example.com/article-2...
✓ Successfully extracted content
  Title: "The Complete Guide to..."
  Content: 1234 characters

Updating link in API...
✓ Link updated successfully!
  ID: 123e4567-e89b-12d3-a456-426614174000
  Title: The Complete Guide to...
```

### Backward Compatible Mode

```bash
# Still works as before (no API interaction)
$ bun run src/cli.ts https://example.com
{"url":"https://example.com","title":"...","text":"...","extracted_at":"..."}
```

## API Endpoint Requirements

### Current State

The API currently has:

- `GET /api/links` - List links ✅
- `POST /api/links` - Create link ✅
- `PATCH /api/links/{link_id}` - Update link ❌ (needs to be added)

### Option 1: Add PATCH Endpoint (Recommended)

Add to `link-api/app/api/v1/links.py`:

```python
@router.patch("/{link_id}", response_model=LinkRead)
async def update_link(
    link_id: UUID,
    payload: LinkUpdate,  # New schema with optional title/description
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_session)],
) -> LinkRead:
    """Update a link's title and/or description."""
    # Implementation
```

## Dependencies

### New Dependencies

- **None** - No new dependencies required!

### Existing Dependencies

- All existing scraper dependencies remain unchanged
- Use Bun's built-in `fetch` for HTTP requests
- Custom TOML parser (simple implementation, no dependency)

### TOML Parsing

Instead of adding a TOML library, we'll implement a simple custom parser that handles our specific use case:

- Parse section headers: `[api]`, `[user]`
- Parse key-value pairs: `url = "value"` or `key = value`
- Handle quoted and unquoted strings
- Basic string escaping

This is sufficient for parsing `~/.config/lnk/config.toml` which has a simple structure:

```toml
[api]
url = "http://localhost:8000"
key = "your-api-key-here"

[user]
email = "user@example.com"
```

A full TOML parser is not needed for this use case.

## Implementation Details

### File Structure

```
scraper/src/
  ├── api-client.ts    # API client for listing/updating links
  ├── config.ts        # Configuration loading from shared config file
  ├── selector.ts      # Interactive link selection
  ├── cli.ts           # Updated CLI with interactive mode
  ├── browser.ts       # Existing browser manager
  ├── extractor.ts     # Existing content extractor
  ├── output.ts        # Existing output formatting
  └── types.ts         # Type definitions
```

### Configuration Loading

```typescript
// src/config.ts
import { readFileSync } from "fs";
import { join } from "path";
import { homedir } from "os";

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
    if (sectionMatch) {
      currentSection = sectionMatch[1];
      if (!result[currentSection]) {
        result[currentSection] = {};
      }
      continue;
    }

    // Parse key-value pair: key = "value" or key = value
    const kvMatch = trimmed.match(/^([^=]+)=(.*)$/);
    if (kvMatch && currentSection) {
      const key = kvMatch[1].trim();
      let value = kvMatch[2].trim();

      // Remove quotes if present
      if ((value.startsWith('"') && value.endsWith('"')) ||
          (value.startsWith("'") && value.endsWith("'"))) {
        value = value.slice(1, -1);
      }

      result[currentSection][key] = value;
    }
  }

  return result;
}

function loadConfig(): { apiUrl: string; apiKey: string } {
  const configPath = join(homedir(), ".config", "lnk", "config.toml");

  // Try environment variables first
  const apiUrl = process.env.LNK_API_URL;
  const apiKey = process.env.LNK_API_KEY;

  if (apiUrl && apiKey) {
    return { apiUrl, apiKey };
  }

  // Try config file
  try {
    const content = readFileSync(configPath, "utf-8");
    const config = parseToml(content);
    return {
      apiUrl: config.api?.url || "http://localhost:8000",
      apiKey: config.api?.key || "",
    };
  } catch (error) {
    throw new Error(
      `Failed to load config. Please set LNK_API_URL and LNK_API_KEY, ` +
      `or configure ~/.config/lnk/config.toml`
    );
  }
}
```

**Note**: This simple parser handles the basic TOML structure needed for the config file. It doesn't support all TOML features (arrays, nested tables, etc.), but that's not needed for this use case.

## Success Criteria

- ✅ User can list saved links from API
- ✅ User can interactively select a link
- ✅ Selected link is scraped successfully
- ✅ Scraped content is automatically posted to API
- ✅ Clear error messages for all failure cases
- ✅ Backward compatible with existing URL-based usage
- ✅ Reuses credentials from Rust CLI tool
- ✅ Simple, focused implementation
