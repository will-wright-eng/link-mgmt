## Charmbracelet TUI Architecture – Multi-Flow Design

### Overview

This document defines a Charmbracelet-based TUI architecture for the CLI that:

- **Reuses existing implementations** in `pkg/cli/forms` and `pkg/cli/tui`
- **Separates concerns** between application shell, flows, and widgets
- **Scales to multiple flows** (add link with/without scraping, delete link, list links, future flows)

Goals:

- Provide a consistent Bubble Tea architecture for all TUI flows
- Keep domain logic (API calls, scraping) out of view code
- Make it easy to plug in new flows (e.g., “list links”, “edit link”, “scrape-only”)

---

### High-Level Architecture

#### Layers

- **App Shell (`pkg/cli/app.go`)**
    - Owns process-level concerns: config, API client, scraper service.
    - Chooses which **flow** to run (based on CLI command/flags).
    - Starts a Bubble Tea `Program` with a root `tea.Model` for that flow.

- **Flows (`pkg/cli/tui` and `pkg/cli/forms`)**
    - Each flow is a **self-contained `tea.Model`** implementing a single user journey:
        - Add link (basic, no scraping) – `forms.addLinkForm`
        - Add link with scraping – `tui.addLinkForm`
        - Delete link – `forms.deleteLinkSelector`
    - Flows receive fully-constructed dependencies (clients, services, config) from the app shell.
    - Each flow manages its own internal state machine (steps), view rendering, and keyboard handling.

- **Widgets / Bubbles**
    - Reuse Charmbracelet **bubbles** (`textinput`, `textarea`) for form fields and confirmation prompts.
    - Optional: Extract common styling or helpers into a future `pkg/cli/tui/components` package if needed.

#### Separation of Concerns

- **App-level**:
    - Reads configuration.
    - Constructs `*client.Client` and `*scraper.ScraperService`.
    - Decides which flow model to instantiate.

- **Flow-level**:
    - Implements `tea.Model` (`Init`, `Update`, `View`) for a specific user scenario.
    - Owns local step/state machine (e.g., URL → Scraping → Review → Saving → Success).
    - Sends **domain commands** (HTTP calls) via injected clients/services.

- **Domain-level**:
    - `pkg/cli/client`: API client for link/user operations.
    - `pkg/scraper`: HTTP client for scraper service, with context, progress, and structured errors.
    - `pkg/utils/validation`: URL and input validation.

---

### Root App Shell and Flow Selection

**File**: `pkg/cli/app.go`

Responsibilities:

- Initialize:
    - `*client.Client` (with base URL + API key)
    - `*scraper.ScraperService` (with base URL)
- Expose a `Run()` method that:
    - Chooses a **flow model** based on CLI command (e.g., `add`, `add-scrape`, `delete`).
    - Starts Bubble Tea:
        - `tea.NewProgram(model).Run()`

Example mapping (conceptual):

- `link-mgmt add` → `forms.NewAddLinkForm(client)`
- `link-mgmt add --scrape` → `tui.NewAddLinkForm(client, scraperService, cfg.CLI.ScrapeTimeout)`
- `link-mgmt delete` → `forms.NewDeleteLinkSelector(client)`

This keeps **flow instantiation in one place**, making it straightforward to add new flows later.

---

### Flow Design Patterns

Each flow (`tea.Model`) follows a common pattern:

- **Fields**:
    - Injected dependencies (`*client.Client`, `*scraper.ScraperService`).
    - Input models (`textinput.Model`, `textarea.Model`).
    - Step/state markers (`step`, `currentField`).
    - Error/success state (`err`, `created`, etc.).

- **Init**:
    - Returns initial command (e.g., `textinput.Blink` or a load-links command).

- **Update**:
    - Handles:
        - `tea.KeyMsg` for user input and navigation.
        - Custom messages for async operations (`submitSuccessMsg`, `scrapeSuccessMsg`, `linksLoadedMsg`, etc.).
    - Updates only the relevant bubble(s) based on current step/field.

- **View**:
    - Renders the current step as text UI.
    - Uses `lipgloss` for emphasis (e.g., current field bold in `add_link_form`).

#### Existing Flows

- **Basic Add Link (`pkg/cli/forms/add_link.go`)**
    - Single linear step sequence:
        - 0: URL
        - 1: Title
        - 2: Description
        - 3: Text
        - 4: Success
    - Submission:
        - Uses `client.CreateLink` via a command returned from `submit()`.
    - Responsibilities:
        - Input validation (`utils.ValidateURL`).
        - Building `models.LinkCreate` and handling the success result.

- **Delete Link (`pkg/cli/forms/delete_link.go`)**
    - Steps:
        - 0: Load & select a link from a list.
        - 1: Confirm deletion (yes/no).
        - 2: Success.
    - Async loading and deleting:
        - `linksLoadedMsg`, `deleteErrorMsg`, `deleteSuccessMsg`.
    - Responsibilities:
        - Fetch and display links.
        - Handle selection and confirmation.

- **Add Link with Scraping (`pkg/cli/tui/add_link_form.go`)**
    - Steps:
        - `stepURLInput` – URL input, optional “skip scraping”.
        - `stepScraping` – loading state while scraper runs.
        - `stepReview` – edit title/description/text (pre-filled from scraper).
        - `stepSaving` – saving to API.
        - `stepSuccess` – success summary.
    - Async scraping:
        - Uses `ScraperService.ScrapeWithProgress` with a callback.
        - Structured errors via `scraper.ScraperError` → `UserMessage()` mapped into UI.

These flows demonstrate **how independent `tea.Model`s can coexist**, each with its own state machine and messages, while sharing clients and utilities.

---

### Separation of Concerns for Multiple Flows

To support multiple flows cleanly:

- **Flow Independence**
    - Each flow is a self-contained `tea.Model`, with:
        - Private message types (e.g., `submitSuccessMsg`, `deleteSuccessMsg`).
        - Private step constants and view logic.
    - No direct coupling between flows; the app shell chooses exactly one to run per command.

- **Shared Dependencies**
    - `client.Client` and `scraper.ScraperService` are **constructed once** and passed in.
    - Avoid global state; rely on explicit dependency injection from `App`.

- **Shared Patterns**
    - Common patterns for:
        - Submit commands (`submit() tea.Cmd`).
        - Load commands (`loadLinks() tea.Msg`).
        - Error handling (`userFacingError` for scraper errors).
    - These patterns can later be extracted into helper functions if they repeat.

---

### Extending to New Flows

Examples of future flows that fit this architecture:

- **List Links Flow**
    - `listLinksModel`:
        - Loads links via `client.ListLinks`.
        - Displays them with pagination and filtering.
        - Allows selection to transition into another command (e.g., view details or edit).

- **Edit Link Flow**
    - `editLinkForm`:
        - Takes an existing `models.Link` (or ID) and pre-fills fields.
        - Uses similar multi-field navigation as `addLinkForm`.
        - Submits updates via an appropriate API method.

- **Scrape-Only Flow**
    - `scrapePreviewModel`:
        - Accepts a URL.
        - Runs scraping and shows raw title/text.
        - Allows the user to:
            - Save as link (reusing submission logic), or
            - Just view and exit.

For each new flow:

- Add a new `tea.Model` under `pkg/cli/tui` or `pkg/cli/forms`, depending on complexity.
- Wire it into `App.Run` based on a new CLI command or flag.

---

### Charmbracelet Toolkit Usage Guidelines

- **Bubble Tea (`tea.Model`)**
    - One model per flow.
    - Keep `Update` focused on:
        - Routing messages to the correct child bubble.
        - Orchestrating state transitions (steps).
    - Avoid doing heavy logic directly in `Update`; delegate to commands.

- **Bubbles (`textinput`, `textarea`)**
    - Prefer bubbles for any user-editable input.
    - Keep bubble configuration (widths, placeholders, char limits) localized to the flow’s constructor (`NewAddLinkForm`, etc.).

- **Lipgloss**
    - Use for:
        - Emphasizing the active field (current focus).
        - Simple styling in lists (selected vs non-selected items).
    - Keep styling minimal and consistent across flows.

---

### Summary of Design Principles

- **Single Responsibility per Flow**: Each flow owns one user journey and its state machine.
- **Explicit Dependencies**: Flows receive clients/services via constructors; no hidden globals.
- **Composable Models**: Existing flows (add basic, add with scraping, delete) are already composable `tea.Model`s; new flows should follow the same pattern.
- **Charmbracelet-First**: Rely on Bubble Tea and Bubbles for input, navigation, and rendering, with Lipgloss for lightweight styling.

This architecture lets you keep `@tui` focused on richer, multi-step, possibly async experiences (like scraping), while `@forms` can continue to host simpler, linear flows—both leveraging the same Charmbracelet toolkit and design principles.
