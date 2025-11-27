# Steps to Integrate Bun Scraper into Go CLI

## Overview

This document outlines the steps necessary to embed the Bun TypeScript scraper as an executable within the Go CLI, following the approach described in `embed-typescript-in-go-app.md` (Option #1: Embed compiled standalone executable).

## Quick Summary

1. **Build** standalone Bun executable from TypeScript scraper
2. **Embed** the executable binary in Go using `//go:embed`
3. **Extract** binary to temp file at runtime
4. **Execute** extracted binary with appropriate arguments
5. **Add** CLI commands to invoke scraper from Go CLI
6. **Update** build process to compile scraper before Go build

## Key Files to Modify

- `scraper/Makefile` - Add build target for standalone executable
- `link-mgmt-go/pkg/cli/app.go` - Add `RunScraper()` method
- `link-mgmt-go/pkg/cli/scraper.go` - New file with embedding logic
- `link-mgmt-go/cmd/cli/main.go` - Add `--scrape` flags
- `link-mgmt-go/Makefile` - Integrate scraper build into Go build

## Approach

We'll use **Bun's standalone executable feature** to create a single binary that includes the runtime, then embed that binary in the Go CLI using Go's `//go:embed` directive.

## Prerequisites & Verification

**Before starting**: Verify that Bun supports creating standalone executables.

1. Check Bun version: `bun --version` (should be recent version)
2. Check if `bun build --compile` is available: `bun build --help | grep compile`
3. If `--compile` is not available, see "Alternative Approaches" section below

**Note**: As of late 2024, Bun's compilation features may vary. If `--compile` doesn't exist, we'll need to use alternative approaches (see bottom of document).

## Steps

### 1. Build Standalone Bun Executable

**Location**: `scraper/` directory

**Action**: Create a build script/Makefile target that compiles the TypeScript scraper into a standalone executable.

**If Bun supports `--compile`**:

```bash
bun build --compile --outdir=dist --target=bun src/cli.ts
```

**If Bun doesn't support `--compile`**, use alternative approach (see "Alternative Approaches" section).

**Requirements**:

- The executable should be platform-specific (we'll need separate builds for different OS/architectures)
- The entry point should be `src/cli.ts`
- Output should be a single binary (e.g., `scraper-darwin-amd64`, `scraper-darwin-arm64`, `scraper-linux-amd64`, `scraper-linux-arm64`, `scraper-windows-amd64.exe`)

**Example command**:

```bash
bun build --compile --outdir=dist --target=bun src/cli.ts
```

**Note**: Bun's `--compile` flag creates a standalone executable that includes the Bun runtime, making it perfect for embedding.

### 2. Create Build Script for Multi-Platform Support

**Location**: `scraper/Makefile` or `scraper/build.sh`

**Action**: Create a build script that:

- Builds executables for multiple platforms (darwin, linux, windows)
- Builds for multiple architectures (amd64, arm64)
- Names outputs with platform/arch suffixes
- Places outputs in a `dist/` directory

**Considerations**:

- May need to use Docker or cross-compilation tools if building on a single platform
- Bun's compile feature may have platform limitations - may need to build on each target platform

### 3. Embed Executable in Go CLI

**Location**: `link-mgmt-go/pkg/cli/` (new file: `scraper.go` or add to `app.go`)

**Action**:

- Use `//go:embed` to embed the scraper binary
- Handle platform-specific embedding (use build tags: `//go:build darwin && amd64`, etc.)
- Create a function to extract and execute the embedded binary

**Implementation pattern**:

```go
//go:build darwin && amd64
// +build darwin,amd64

package cli

import (
    _ "embed"
    "os"
    "os/exec"
    "path/filepath"
)

//go:embed ../../../scraper/dist/scraper-darwin-amd64
var scraperBinary []byte

func (a *App) runScraper(args []string) error {
    // Write embedded binary to temp location
    tmpFile := filepath.Join(os.TempDir(), "scraper-binary")
    if err := os.WriteFile(tmpFile, scraperBinary, 0755); err != nil {
        return err
    }
    defer os.Remove(tmpFile)

    // Execute it with provided args
    cmd := exec.Command(tmpFile, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin
    return cmd.Run()
}
```

**Note**: Need separate files or build tags for each platform/architecture combination.

### 4. Add Scrape Command to Go CLI

**Location**: `link-mgmt-go/cmd/cli/main.go`

**Action**: Add a new flag and handler for scraping:

- Add `--scrape` flag (or `--scrape-url` with URL argument)
- Add `--scrape-interactive` flag for interactive mode
- Pass appropriate arguments to the embedded scraper binary

**Implementation**:

```go
scrapeMode := flag.Bool("scrape", false, "Scrape a URL and extract content")
scrapeURL := flag.String("scrape-url", "", "URL to scrape")
scrapeInteractive := flag.Bool("scrape-interactive", false, "Interactive scraping mode")

// In main():
if *scrapeMode || *scrapeURL != "" || *scrapeInteractive {
    var args []string
    if *scrapeInteractive {
        args = []string{} // Interactive mode - no args needed
    } else if *scrapeURL != "" {
        args = []string{*scrapeURL}
    }
    // Add other scraper options as needed

    if err := app.RunScraper(args); err != nil {
        log.Fatalf("scraping failed: %v", err)
    }
    return
}
```

### 5. Add Scraper Method to App

**Location**: `link-mgmt-go/pkg/cli/app.go`

**Action**: Add a `RunScraper` method that:

- Handles the embedded binary execution
- Passes through CLI arguments
- Handles environment variables if needed (e.g., API URL/key for interactive mode)
- Manages error handling and output

**Considerations**:

- The scraper may need access to config (API URL/key) for interactive mode
- May need to set environment variables before executing
- Should handle both interactive and non-interactive modes

### 6. Update Build Process

**Location**: `link-mgmt-go/Makefile` and root `Makefile`

**Action**:

- Add a build step that compiles the scraper executable before building the Go CLI
- Ensure the scraper binary exists before embedding
- Add a `make build-scraper` target
- Integrate scraper build into the main Go CLI build process

**Build order**:

1. Build scraper executables for all platforms
2. Build Go CLI (which embeds the appropriate scraper binary)

### 7. Handle Platform Detection

**Location**: `link-mgmt-go/pkg/cli/scraper.go` (or similar)

**Action**: Since we need platform-specific binaries, we have two options:

**Option A**: Build separate Go binaries per platform (recommended)

- Use Go build tags to embed the correct scraper binary
- Build process creates platform-specific Go binaries
- Each Go binary contains only its platform's scraper

**Option B**: Embed all platform binaries and select at runtime

- Embed all platform binaries
- Detect platform at runtime
- Extract and use the correct one
- More complex but single build process

**Recommendation**: Option A is cleaner and results in smaller binaries.

### 8. Testing

**Action**: Test the integration:

- Test scraping a URL from Go CLI
- Test interactive scraping mode
- Test error handling (invalid URL, network errors, etc.)
- Test on different platforms if possible
- Verify embedded binary is properly extracted and executed

### 9. Documentation

**Action**: Update documentation:

- Update CLI README with scraping commands
- Document the build process
- Note any platform-specific requirements

## Implementation Order

1. **First**: Set up Bun standalone executable build (Step 1-2)
2. **Second**: Test the standalone executable works independently
3. **Third**: Implement embedding in Go (Step 3-5)
4. **Fourth**: Add CLI commands (Step 4)
5. **Fifth**: Update build process (Step 6)
6. **Sixth**: Test and document (Step 8-9)

## Challenges & Considerations

1. **Playwright Dependencies**: The scraper uses Playwright, which requires browser binaries. The standalone Bun executable should include these, but verify they're properly bundled.

2. **File Size**: Standalone executables can be large (50-100MB+). This will increase the Go CLI binary size significantly.

3. **Cross-Compilation**: Building Bun executables for multiple platforms may require building on each platform or using Docker.

4. **Temporary Files**: The embedded binary is written to a temp file before execution. Ensure proper cleanup and handle cases where temp directory isn't writable.

5. **Permissions**: The extracted binary needs execute permissions (handled with `0755` mode in `os.WriteFile`).

6. **Config Sharing**: The scraper may need access to the same config as the Go CLI (API URL/key). Consider:
   - Passing config via environment variables
   - Passing config via command-line arguments
   - Having scraper read from same config file location

## Alternative Approaches (If Bun Standalone Doesn't Work)

If Bun's standalone executable feature doesn't work as expected, consider these alternatives:

### Alternative 1: Bundle JS and Execute with Bun Runtime

**Approach**: Bundle TypeScript to a single JS file, then execute it with an embedded or system Bun runtime.

**Steps**:

1. Bundle TypeScript: `bun build src/cli.ts --outdir=dist --target=bun --minify`
2. Embed the bundled JS file in Go
3. Execute with `bun run <embedded-js>` (requires Bun to be installed on system)
   - OR embed Bun runtime binary and execute: `embedded-bun run <embedded-js>`

**Pros**: Simpler build process
**Cons**: Requires Bun runtime (either system-installed or embedded)

### Alternative 2: Use Node.js with pkg/nexe

**Approach**: Convert scraper to use Node.js instead of Bun, then use `pkg` or `nexe` to create standalone executable.

**Steps**:

1. Adapt scraper code to work with Node.js (replace Bun-specific APIs)
2. Install dependencies: `npm install`
3. Create standalone with pkg: `pkg src/cli.ts --targets node18-linux-x64,node18-darwin-x64,node18-darwin-arm64`
4. Embed the resulting executables in Go

**Pros**: Well-established tooling, proven approach
**Cons**: Requires rewriting Bun-specific code, larger binaries

### Alternative 3: V8 Embedding (Most Complex)

**Approach**: Use `rogchap/v8go` to execute JavaScript directly within Go process.

**Steps**:

1. Bundle TypeScript to JavaScript
2. Use v8go to execute the bundled JS
3. Handle Playwright/Node.js APIs (may require polyfills or alternative implementations)

**Pros**: No external process, single binary
**Cons**: Most complex, may not support all Node.js/Bun APIs (Playwright especially)

### Alternative 4: Keep Separate, Call as Subprocess

**Approach**: Don't embed - keep scraper as separate binary and call it as subprocess.

**Steps**:

1. Build scraper as standalone (using any method above)
2. Include scraper binary alongside Go CLI binary
3. Call scraper binary from Go CLI using `exec.Command`

**Pros**: Simplest, no embedding complexity
**Cons**: Requires distributing two binaries, not truly "embedded"

**Recommendation**: Start with verifying Bun's compile feature. If unavailable, try Alternative 1 (bundle + Bun runtime) as it requires minimal changes.

## Next Steps

1. Verify Bun's `--compile` flag works for creating standalone executables
2. Test building a standalone executable for your platform
3. Verify the standalone executable works independently
4. Proceed with embedding implementation
