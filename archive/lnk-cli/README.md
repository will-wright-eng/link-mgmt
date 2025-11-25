# lnk - Link Management CLI

A command-line interface for the link management API.

## Installation

```bash
cd lnk-cli
cargo build --release
```

The binary will be available at `target/release/lnk`.

## Usage

### Configuration

Set the API URL (defaults to `http://localhost:8000`):

```bash
lnk config set api.url http://localhost:8000
```

Or use the `--api-url` flag or `LNK_API_URL` environment variable:

```bash
lnk --api-url http://localhost:8000 save https://example.com
```

### Authentication

Currently, the API doesn't require authentication, but the CLI is ready for when it's added:

```bash
# Login with API key (when authentication is enabled)
lnk auth login --api-key <your-api-key>

# Check authentication status
lnk auth status

# Logout
lnk auth logout
```

### Commands

#### Save a link

```bash
lnk save https://example.com
lnk save https://example.com --title "Example Site"
lnk save https://example.com --title "Example" --description "An example website"
```

#### List links

```bash
lnk list
lnk list --limit 10
```

#### Get a specific link

```bash
lnk get <link-id>
```

## Development

```bash
# Run in development mode
cargo run -- save https://example.com

# Check for errors
cargo check

# Build release
cargo build --release
```
