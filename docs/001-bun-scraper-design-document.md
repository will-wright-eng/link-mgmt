# Web Scraper CLI Tool - Design Document

## Overview

A command-line tool that extracts main body text from URLs using headless browser rendering and readability algorithms. Built with Bun for fast execution and minimal overhead.

**Goal**: Take a list of URLs and return structured text content from each, handling diverse website architectures.

**Target Use Cases**:

- Content extraction for knowledge bases
- Article harvesting from multiple sources
- Integration into data pipeline workflows
- Development of downstream text analysis tools

## Architecture

### High-Level Flow

```
Input (URLs)
    ↓
Validate & normalize URLs
    ↓
Render with Playwright
    ↓
Extract main content (readability)
    ↓
Normalize & clean text
    ↓
Output (structured format)
```

### Core Components

**URL Input Handler**: Accepts URLs from stdin, file, or CLI arguments. Validates format and handles duplicates.

**Browser Manager**: Handles Playwright browser lifecycle. Reuses browser context across URL batches rather than spawning new instances per URL (resource efficiency).

**Content Extractor**: Combines readability algorithm with DOM parsing to identify and extract main article/body content.

**Output Formatter**: Converts extracted content to structured format (JSONL, JSON, or plain text).

**Error Handler**: Graceful failure handling for timeouts, invalid content, network errors with structured error reporting.

## Technology Stack

### Primary Dependencies

**Playwright**: Headless browser automation. Renders JavaScript-heavy sites and provides DOM access.

- Alternatives: Puppeteer (older, less polished API), Cheerio (no rendering, static HTML only), Selenium (overkill for this use case)

**@mozilla/readability**: Mozilla's readability algorithm. Identifies article/main content using heuristics on DOM structure.

- Alternatives: node-readability (older port), custom DOM analysis (unreliable across architectures)

**jsdom**: DOM implementation for Node/Bun environments. Used by readability to parse and traverse DOM.

**zod** (optional but recommended): Input validation and schema definition for CLI arguments and output structure.

### Build & Runtime

- **mise**: Runtime version manager for managing Bun and Node versions
- **Bun 1.0+**: Runtime and package manager
- **TypeScript**: For type safety and developer experience
- **Node 20+**: Fallback if Playwright issues arise with Bun (can still use Node with same code)

## Implementation Status

### ✅ Phase 1: MVP (Core Functionality) - COMPLETE

**Status**: Implemented and ready for testing

**Implementation Details**:

- All core files created in `src/` directory
- BrowserManager class implemented with Playwright integration
- Content extraction using Mozilla Readability
- CLI supports multiple input methods (args, stdin, files)
- Output formatting for JSONL, JSON, and plain text
- Error handling with structured error reporting
- Makefile integration for common tasks

**Makefile Commands Available**:

```bash
make scraper URL="https://example.com"  # Run scraper
make scraper-install                     # Install dependencies
make scraper-check                       # Type check
make scraper-fmt                         # Format code
make scraper-fmt-check                   # Check formatting
```

**Usage Examples**:

```bash
# Direct bun usage
cd scraper && bun run src/cli.ts "https://example.com"
cd scraper && bun run src/cli.ts --input urls.txt --output results.jsonl --format jsonl

# Via Makefile
make scraper URL="https://example.com"
```

## Implementation Plan

### Phase 1: MVP (Core Functionality) ✅

**Files**:

```
src/
  ├── cli.ts           # CLI argument parsing and orchestration
  ├── browser.ts       # Playwright browser management
  ├── extractor.ts     # Content extraction logic
  ├── output.ts        # Output formatting
  └── types.ts         # TypeScript interfaces
```

**CLI Interface**:

```bash
# Single URL
bun run src/cli.ts "https://example.com"

# Multiple URLs from stdin
cat urls.txt | bun run src/cli.ts

# Multiple URLs as arguments
bun run src/cli.ts "https://example.com" "https://another.com"

# Output options
bun run src/cli.ts --input urls.txt --output results.jsonl --format jsonl
bun run src/cli.ts --timeout 10000 --headless true
```

**Output Format (JSONL)**:

```json
{"url":"https://example.com","title":"Page Title","text":"Extracted body text...","extracted_at":"2024-01-15T10:30:00Z","error":null}
{"url":"https://another.com","title":"","text":"","extracted_at":"2024-01-15T10:30:01Z","error":"Timeout after 10000ms"}
```

### Phase 2: Robustness (Next Steps)

- Concurrency control (limit parallel page renders)
- Retry logic with exponential backoff
- Better error classification (timeout vs invalid content vs network error)
- Metadata extraction (author, publish date, etc.)
- Caching layer for repeated URLs

### Phase 3: Integration (Future)

- HTTP API wrapper (expose as lightweight service)
- Message queue integration (process from job queue)
- Integration hooks for Temporal/Prefect pipelines

## Quick Start

After setup (see below), you can immediately start using the scraper:

```bash
# Test with a single URL
cd scraper && bun run src/cli.ts "https://example.com"

# Or use the Makefile
make scraper URL="https://example.com"

# Process multiple URLs from a file
echo "https://example.com" > urls.txt
echo "https://another.com" >> urls.txt
cd scraper && bun run src/cli.ts --input urls.txt --output results.jsonl

# Process URLs from stdin
cat urls.txt | cd scraper && bun run src/cli.ts
```

## Setup Instructions

### Prerequisites

```bash
# Or using Homebrew (macOS)
brew install mise

# Verify installation
mise --version
```

### Runtime Setup

```bash
# init mise
touch .mise.toml
mise use bun@1.3.3
mise install

# Verify Bun is available
bun --version
```

**Note**: `mise install` automatically downloads and installs the versions specified in `.mise.toml`. You do not need to install Bun or Node separately when using mise.

### Project initialization

```bash
# Create project directory
bun init scraper
cd scraper

# Add dependencies
bun add playwright @mozilla/readability jsdom zod
```

**Note**: Dependencies have been installed. The project structure is complete with all Phase 1 files in `src/`.

### Playwright browser installation

```bash
# Playwright downloads browsers on first use, but you can pre-install
bunx playwright install chromium
```

**Troubleshooting Playwright + Bun**: Bun's dependency resolution occasionally conflicts with Playwright's binary downloads. If you hit issues:

- Clear bun cache: `rm -rf ~/.bun/install/cache`
- Use npm for playwright installation: `npm install playwright` then `npx playwright install`
- Fall back to Node for this project if necessary (code works with both)

## Implementation Reference

### File Structure

The implementation follows the planned structure:

```
scraper/
├── src/
│   ├── cli.ts           # CLI argument parsing and orchestration ✅
│   ├── browser.ts       # Playwright browser management ✅
│   ├── extractor.ts     # Content extraction logic ✅
│   ├── output.ts        # Output formatting ✅
│   └── types.ts         # TypeScript interfaces ✅
├── package.json         # Dependencies configured ✅
├── tsconfig.json        # TypeScript configuration ✅
└── bun.lock             # Dependency lock file ✅
```

### Key Implementation Details

**BrowserManager** (`src/browser.ts`):

- Manages Playwright browser lifecycle
- Reuses browser context for efficiency
- Configurable headless mode
- Configurable timeout per URL

**Content Extractor** (`src/extractor.ts`):

- Uses Mozilla Readability algorithm
- Handles extraction failures gracefully
- Returns structured content (title + text)

**CLI** (`src/cli.ts`):

- Supports multiple input methods:
    - Command-line arguments (single or multiple URLs)
    - File input via `--input` flag
    - Stdin (piped input)
- Output options:
    - JSONL (default, one result per line)
    - JSON (pretty-printed array)
    - Plain text (human-readable)
- Configurable options:
    - `--timeout`: Page load timeout (default: 10000ms)
    - `--headless`: Browser headless mode (default: true)
    - `--output`: Write results to file
    - `--format`: Output format (jsonl/json/text)

**Output Formatter** (`src/output.ts`):

- Formats individual results and batches
- Supports all three output formats
- Handles error cases in output

### Code Sketches (Reference)

The actual implementation closely follows the original sketches with enhancements:

- Added proper TypeScript types
- Enhanced CLI argument parsing
- Added file input/output support
- Improved error handling structure

## Considerations & Tradeoffs

### Resource Usage

Playwright browsers consume significant memory (~100-150MB per instance). For processing many URLs, you have options:

**Single browser context** (current approach): Reuse one browser for all URLs. Lower memory, but slower (sequential). Good for small batches.

**Connection pool**: Maintain 3-5 browser instances, queue URLs across them. Requires more infrastructure but handles concurrency.

**Lambda/serverless**: Spawn fresh browser per URL, tear down after. Good for infrequent processing, wasteful for frequent use.

**Recommendation for MVP**: Single browser context. Add pooling in Phase 2 if benchmarks show it's necessary.

### Text Quality

Readability works well for articles and blog posts (70-80% accuracy), but struggles with:

- News sites with heavy sidebar content
- Documentation (often has important content outside main article)
- E-commerce pages (product pages ≠ articles)
- Paywalled content (often hidden behind JavaScript)

For these, you might need site-specific extraction logic. Build hooks for that in Phase 2.

### Performance

Expected performance on modern hardware:

- Cold start (browser launch): ~2-3 seconds
- Per-URL (render + extract): ~1-3 seconds depending on site complexity
- Batch of 100 URLs: ~2-5 minutes

Bun's advantage is lower overhead than Node - ~15-20% faster execution and startup.

### Alternatives to Consider

**Firecrawl**: If you want a managed service instead of self-hosted. Better extraction, handles more edge cases. Trade-off: costs money, external dependency, API latency.

**Cheerio-only** (no browser): Much faster (~100ms per URL) but only works on static HTML. Won't render JavaScript-heavy sites. Good for sites you know are static.

**Combination approach**: Try Cheerio first (fast), fallback to Playwright if extraction fails. Hybrid approach balances speed and compatibility.

**OCR-based**: For images/PDFs, add Tesseract. Out of scope for MVP but relevant for some content.

## Error Handling Strategy

**Timeout** (page doesn't load): Return empty result with error message. Set reasonable timeout (10s default, configurable).

**Invalid content** (readability returns null): Could be non-article page. Return page title + first 500 chars as fallback.

**Network error** (unreachable URL): Catch at Playwright level, return error.

**Invalid URL**: Validate before attempting to render.

Structure errors consistently in output so downstream tooling can categorize and retry intelligently.

## Testing Approach

### Current Status (Phase 1)

The MVP implementation is complete and ready for testing. Recommended next steps:

1. **Manual Testing**:
   - Test against 5-10 real URLs (mix of article sites, docs, product pages)
   - Verify extraction quality manually
   - Time execution across different site types
   - Test all input methods (args, stdin, files)
   - Test all output formats (jsonl, json, text)

2. **Error Handling Verification**:
   - Test with invalid URLs
   - Test with unreachable URLs
   - Test with timeout scenarios
   - Verify error messages are structured correctly

### Future Testing (Phase 2+)

- Snapshot testing against known page structures
- Error rate tracking for different site categories
- Benchmark suite comparing to alternatives
- Automated test suite with sample URLs

## Success Criteria

- Successfully extract text from 80%+ of tested URLs
- Execution time under 3 seconds per URL on average
- Graceful error handling for invalid/unreachable URLs
- Clear CLI interface with useful output
- Code structured for Phase 2 enhancements (pooling, caching, etc.)

## Future Enhancements

- **Phase 2**: Concurrency pooling, caching, retry logic, better metadata extraction
- **Phase 3**: HTTP service, queue integration, site-specific extractors for popular domains
- **Phase 4**: Compare against Firecrawl for quality, consider hybrid approach
- **Long-term**: ML-based content detection for higher accuracy across diverse architectures
