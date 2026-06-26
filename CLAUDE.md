# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository layout

Monorepo with three deployables plus infra, orchestrated by Docker Compose:

- `link-mgmt/` — Go module (`module link-mgmt`, Go 1.25). Contains **both** the REST API server (`cmd/api`) and the CLI/TUI (`cmd/cli`), sharing packages under `pkg/`.
- `scraper/` — TypeScript content-extraction service running on Bun.
- `nginx/` — reverse proxy; the single entry point for everything.
- `docs/` — design documents (see end of file).

## Commands

Prefer the Makefiles over the READMEs — the READMEs have drifted (see "Doc drift" below). Each Makefile has a `help` target.

### Root (Docker orchestration + delegation)

- `make upd` / `make up` — start all containers (detached / foreground, with build). Compose has **no profiles**; `docker compose up` starts everything.
- `make down`, `make logs`
- `make migrate` — run SQL migrations into the running Postgres container. **Migrations are not auto-applied on API startup**; run this after first `up`.
- `make postgres-up` / `make postgres-down` — Postgres only.
- `make prod-up` / `make prod-upd` / `make prod-down` / `make prod-logs` — use `docker-compose.prod.yml`.
- Delegation targets: `make go-build-all`, `make go-run-api`, `make go-run-cli`, `make go-test`, `make scraper`, `make scraper-check`, `make scraper-fmt`.

### Go (`cd link-mgmt`)

- `make build-all` (→ `bin/api`, `bin/cli`), `make build-api`, `make build-cli`
- `make run-api`, `make run-cli`
- `make deps` — `go mod download && go mod tidy`
- `make seed` — `bash add-links.sh links.txt`
- Tests: `go test ./...`; single test: `go test ./pkg/<pkg> -run TestName`. (No test files exist yet.)
- Lint (matches pre-commit): `golangci-lint run --timeout=5m ./...` and `go fmt ./...`

### Scraper (`cd scraper`)

- `make dev-server` (`bun --watch`) / `make server` (`bun start`)
- `make check` — `tsc --noEmit`; `make lint` / `make lint-fix` — eslint; `make fmt` / `make fmt-check` — prettier
- `make dev` — runs check + lint + fmt-check together
- `make test-health`, `URL=https://example.com make test-scrape` — curl the running service
- Bun version is pinned to 1.3.3 via `.mise.toml`. No unit-test framework is configured.

`pre-commit` runs golangci-lint, go fmt, eslint, prettier, and markdownlint across the monorepo.

## Architecture

### Request flow

Everything is reached through nginx on port 80. Services are **not** exposed to the host directly (except Postgres on 5432 for migrations).

```
CLI (host) ──HTTP :80──> nginx ──/api/*──────────> Go API (gin)  :8080 ──pgx──> Postgres :5432
                              └──/scrape,/scraper/*─> Bun scraper :3000 ──Playwright──> external URLs
```

The Go API itself is also a client of the scraper service (it holds its own `pkg/scraper` HTTP client), so scraping happens **server-side**, not from the CLI.

### Go API (`link-mgmt/pkg/api`, `pkg/services`, `pkg/db`)

Layered: **handlers → services → db**.

- Routing is **gin** (`pkg/api/router.go`), not net/http stdlib despite what the README says. Entry point `cmd/api/main.go` wires db + config and runs graceful shutdown.
- `RequireAuth` middleware (`pkg/api/middleware/auth.go`) reads the `Authorization` header (`Bearer <key>` or raw key), looks the key up in `users`, and sets `userID`/`user` on the context. **Every link query is scoped by `user_id`** — the system is multi-tenant by API key.
- `LinkService` (`pkg/services/link_service.go`) holds the orchestration. Key methods: `CreateLink` (plain insert, no scraping), `CreateLinkWithScraping` and `EnrichLink` (insert/fetch then call the scraper and merge results). Merge honors `ScrapeOptions.OnlyFillEmpty`; scrape failures are swallowed so the link is still saved.
- DB layer (`pkg/db/queries.go`) uses **pgx v5** (`db.Pool`). `UpdateLink` builds a dynamic partial-update query from non-nil pointer fields. Migrations live in `link-mgmt/migrations/`.

### CLI / TUI (`link-mgmt/cmd/cli`, `pkg/cli`)

- Actual flags (from `cmd/cli/main.go`): `--register <email>`, `--scrape <url>`, `--save <url>`, `--config-show`, `--config-set section.key=value`. With no flags it launches the **Bubble Tea** TUI (`pkg/cli/tui`). The READMEs' `--add`/`--list`/`--delete` flags do not exist.
- `--scrape` calls the scraper directly (via nginx) for ad-hoc extraction; `--save` and the TUI go through the API (`pkg/cli/client`), which performs any scraping itself.

### Scraper service (`scraper/src`)

- `server.ts` — manual router on `Bun.serve` exposing `GET /health`, `POST /scrape`, `POST /scrape/batch`. A single headless Chromium browser+context is initialized once at startup (`browser.ts`) and reused; SIGINT/SIGTERM trigger graceful cleanup.
- Pipeline: Playwright `page.goto(..., waitUntil: "networkidle")` → `page.content()` → `extractor.ts` runs **JSDOM + Mozilla Readability** → cleaned text. `isBlockedContent` flags bot-wall/security-page responses as a `BLOCKED` error.

### Cross-cutting contracts (read both sides before changing)

- **Error taxonomy** is shared across the language boundary. The scraper returns `{ success, error, error_type, retryable }` (types in `scraper/src/errors.ts`); the Go client maps `error_type` strings to typed errors via `MapErrorTypeFromString` (`pkg/scraper/errors.go`, response shape in `pkg/scraper/types.go`). Adding/renaming an error type means updating both.
- **Timeout units**: config and CLI use **seconds**; the scraper HTTP API expects **milliseconds**. The Go scraper client converts seconds→ms internally (`ScrapeWithProgress` in `pkg/scraper/client.go`, and `ScrapeRequest.Timeout` is documented as ms). Pass seconds to the client; do not pre-multiply.

### Configuration

- Single `config.Config` (`pkg/config/config.go`) shared by API and CLI, loaded from `~/.config/link-mgmt/config.toml`, **auto-created with defaults** (which match `docker-compose.yml` Postgres creds and `http://localhost` nginx) on first run.
- Env overrides applied on load: `DATABASE_URL`, `SCRAPER_BASE_URL` (and `BASE_URL` on first creation). Docker sets `DATABASE_URL` and `SCRAPER_BASE_URL` for the API container.
- `CLI.BaseURL` is the nginx URL used for all services; the API resolves the scraper base from `Scraper.BaseURL`, falling back to `CLI.BaseURL`.

## Doc drift (don't trust these blindly)

The `readme.md` and `link-mgmt/README.md` predate refactors. Known-stale claims: Docker `--profile dev` and `api-dev`/`scraper-dev` service names (compose has no profiles; containers are `link-mgmt-api`/`-scraper`/`-nginx`), `make dev-upd` (it's `make upd`), the API being "standard library HTTP" (it's gin), and CLI flags `--add`/`--list`/`--delete`. When in doubt, trust the Makefiles and source.

## Design docs

`docs/` holds the design history: `architecture-design-review.md`, `scraper-integration-design.md` (how scraping moved into the API), `bun-scraper-design-document.md`, `core-lib-design-document.md`, `link-filtering-design.md`, `state-machine-refactoring-plan.md`, `scrape-status-implementation-plan.md`, `cli-api-language-comparison.md`, `embed-typescript-in-go-app.md`.
