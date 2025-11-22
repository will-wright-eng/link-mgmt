# Interactive Save Command Implementation Plan

## Overview

This document outlines the implementation plan for making the `lnk save` command interactive. The goal is to allow users to run `lnk save` without arguments and be prompted for the URL, title, and description, while maintaining full backward compatibility with the existing command-line argument interface.

## Current State

The `save` command currently:

- Requires a `url` as a positional argument
- Accepts optional `--title` and `--description` flags
- Immediately creates the link without any prompts

**Current usage:**

```bash
lnk save https://example.com
lnk save https://example.com --title "Example Site"
lnk save https://example.com --title "Example" --description "An example website"
```

## Desired Behavior

After implementation, the command should support:

1. **Interactive mode** (when no URL is provided):

   ```bash
   lnk save
   # Prompts for URL, title, and description
   ```

2. **Partially interactive mode** (when URL is provided but other fields are missing):

   ```bash
   lnk save https://example.com
   # Prompts for title and description (optional)
   ```

3. **Non-interactive mode** (current behavior, fully preserved):

   ```bash
   lnk save https://example.com --title "Title" --description "Desc"
   # No prompts, works exactly as before
   ```

## Implementation Steps

### 1. Add Interactive Prompt Library Dependency

Add `inquire` to `Cargo.toml`:

```toml
[dependencies]
# ... existing dependencies ...
inquire = "0.7"
```

**Why `inquire`?**

- Modern, well-maintained Rust library
- Rich feature set (text input, validation, autocomplete)
- Good error handling
- Cross-platform support
- Active development and community

### 2. Modify CLI Structure

#### 2.1 Make `url` Optional

In `src/cli.rs`, change the `Save` command struct:

```rust
#[derive(Subcommand)]
pub enum Commands {
    /// Save a link to the API
    Save {
        /// URL to save (optional - if not provided, interactive mode will prompt)
        url: Option<String>,

        /// Optional title for the link
        #[arg(short, long)]
        title: Option<String>,

        /// Optional description for the link
        #[arg(short, long)]
        description: Option<String>,
    },
    // ... rest of commands
}
```

**Note:** Making `url` optional means users can run `lnk save` without arguments. We'll need to handle the `None` case by prompting.

### 3. Create Interactive Prompt Module

Create a new module `src/prompts.rs` to handle interactive prompts:

```rust
use anyhow::{Context, Result};
use inquire::{Text, TextCustomType, validator::Validation};

/// Prompt for a URL with validation
pub fn prompt_url() -> Result<String> {
    let url_validator = |input: &str| -> Result<Validation, String> {
        if input.trim().is_empty() {
            return Ok(Validation::Invalid("URL cannot be empty".into()));
        }

        // Basic URL validation
        if url::Url::parse(input).is_ok() {
            Ok(Validation::Valid)
        } else {
            Ok(Validation::Invalid("Please enter a valid URL".into()))
        }
    };

    Text::new("URL:")
        .with_validator(url_validator)
        .prompt()
        .context("Failed to read URL input")
}

/// Prompt for an optional title
pub fn prompt_title() -> Result<Option<String>> {
    let title = Text::new("Title (optional, press Enter to skip):")
        .prompt()
        .ok()?;

    let trimmed = title.trim();
    if trimmed.is_empty() {
        Ok(None)
    } else {
        Ok(Some(trimmed.to_string()))
    }
}

/// Prompt for an optional description
pub fn prompt_description() -> Result<Option<String>> {
    let description = Text::new("Description (optional, press Enter to skip):")
        .prompt()
        .ok()?;

    let trimmed = description.trim();
    if trimmed.is_empty() {
        Ok(None)
    } else {
        Ok(Some(trimmed.to_string()))
    }
}
```

**Alternative approach:** Use `inquire::Editor` for multi-line description input.

### 4. Update Save Command Handler

Modify the `handle_save()` method in `src/cli.rs`. After refactoring, the save command logic is isolated in this method, making it easy to add interactive prompts:

```rust
use crate::prompts::{prompt_url, prompt_title, prompt_description};
use std::io::IsTerminal;

async fn handle_save(
    api_url: String,
    config: Config,
    url: Option<String>,
    title: Option<String>,
    description: Option<String>,
) -> Result<()> {
    // Check if we're in a non-interactive environment
    let is_interactive = std::io::stdin().is_terminal();

    // Collect values, prompting for missing ones
    let final_url = match url {
        Some(u) => u,
        None => {
            if !is_interactive {
                anyhow::bail!("URL is required when running in non-interactive mode");
            }
            prompt_url().context("Failed to get URL")?
        },
    };

    let final_title = match title {
        Some(t) => Some(t),
        None => {
            if is_interactive {
                prompt_title().context("Failed to get title")?
            } else {
                None
            }
        },
    };

    let final_description = match description {
        Some(d) => Some(d),
        None => {
            if is_interactive {
                prompt_description().context("Failed to get description")?
            } else {
                None
            }
        },
    };

    let client = Self::create_client(&api_url, &config)?;
    let link = client
        .create_link(
            &final_url,
            final_title.as_deref(),
            final_description.as_deref(),
        )
        .await
        .context("Failed to create link")?;

    Self::display_link_saved(&link);
    Ok(())
}
```

**Note:** The refactored structure uses:

- `Self::create_client()` - Helper method for client creation
- `Self::display_link_saved()` - Helper method for displaying saved link
- The method signature already matches the pattern used by other handlers

### 5. Handle Non-Interactive Mode

Non-interactive mode detection is now integrated into the `handle_save()` method (see Step 4 above). The check ensures scripts and CI/CD pipelines don't hang waiting for input.

**Key points:**

- Check `std::io::stdin().is_terminal()` to detect interactive mode
- Only prompt for optional fields (title, description) in interactive mode
- Require URL in non-interactive mode to prevent hanging

### 6. Add Module Declaration

Update `src/main.rs` to include the new module:

```rust
mod cli;
mod client;
mod config;
mod prompts;  // Add this line
```

## Implementation Considerations

### Refactoring Status

✅ **CLI refactoring completed** (see `docs/cli-refactoring-plan.md`):

- Command handlers extracted into separate methods
- Helper methods for client creation and display formatting
- `handle_save()` method is isolated and ready for interactive prompts
- Code is well-organized and maintainable

### Backward Compatibility

✅ **Full backward compatibility maintained:**

- All existing command invocations will work exactly as before
- No breaking changes to the CLI interface
- Optional prompts only appear when fields are missing
- Non-interactive mode fully supported (scripts, CI/CD)

### User Experience Enhancements

1. **URL Validation:** Validate URLs in real-time during input
2. **Default Values:** Could pre-fill title/description from clipboard or browser
3. **Multi-line Description:** Use `inquire::Editor` for longer descriptions
4. **Confirmation:** Optionally show a summary before saving

### Error Handling

- Handle Ctrl+C gracefully (inquire does this by default)
- Provide clear error messages if validation fails
- Ensure prompts don't block in non-interactive environments

## Example Usage After Implementation

### Fully Interactive

```bash
$ lnk save
URL: https://example.com
Title (optional, press Enter to skip): Example Site
Description (optional, press Enter to skip): An example website
✓ Link saved successfully!
```

### Partially Interactive

```bash
$ lnk save https://example.com
Title (optional, press Enter to skip): Example Site
Description (optional, press Enter to skip):
✓ Link saved successfully!
```

### Non-Interactive (Current Behavior)

```bash
$ lnk save https://example.com --title "Example" --description "Desc"
✓ Link saved successfully!
```

## File Structure Changes

```
lnk-cli/
├── src/
│   ├── cli.rs          # Modified:
│   │                    #   - Make url optional in Save command
│   │                    #   - Update handle_save() to add prompt logic
│   │                    #   - Use existing helper methods (create_client, display_link_saved)
│   ├── main.rs         # Modified: Add prompts module
│   ├── prompts.rs      # New: Interactive prompt functions
│   ├── client/
│   │   ├── mod.rs      # Already exports Link type
│   │   └── links.rs
│   ├── config/
│   └── ...
├── Cargo.toml          # Modified: Add inquire dependency
└── ...
```

## Dependencies

### New Dependency

- `inquire = "0.7"` - Interactive prompts library

### Existing Dependencies Used

- `url = "2"` - Already in Cargo.toml, used for URL validation
- `anyhow` - Already used for error handling

## Future Enhancements

1. **URL Autocomplete:** Suggest recently saved URLs
2. **Title/Description Autofill:** Fetch from URL metadata (Open Graph, etc.)
3. **Multi-line Editor:** Use `inquire::Editor` for descriptions
4. **Confirmation Dialog:** Show summary before saving
5. **Batch Mode:** Save multiple links in one session
6. **Template Support:** Pre-fill common fields from templates

## Testing Checklist

- [ ] Test fully interactive mode (`lnk save`)
- [ ] Test partially interactive mode (`lnk save <url>`)
- [ ] Test non-interactive mode (all args provided)
- [ ] Test URL validation (invalid URLs rejected)
- [ ] Test empty input handling (Enter to skip optional fields)
- [ ] Test Ctrl+C handling (graceful exit)
- [ ] Test non-TTY environment (script/CI mode)
- [ ] Verify backward compatibility (all existing commands work)
- [ ] Test error handling (network errors, API errors)
