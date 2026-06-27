# State Machine Refactoring Plan

> **Status: Planned — design finalized, ready to implement (decisions resolved 2026-06-26).** Not yet implemented: `manageLinksModel` still uses the integer step machine (`StepListLinks` in `pkg/cli/tui/managelinks/constants.go`); there is no `ViewMode`/`OperationState` or `state.go`. The open forks below have been resolved — see **Resolved Design**.

## Resolved Design (authoritative)

This section supersedes the exploratory "Proposed Solution", "Alternative: Option B", and the inconsistent struct definitions further down, which are kept only as background.

**Decisions**

- **Approach A** — a single `State` value with transition methods, defined in `pkg/cli/tui/managelinks/state.go`. The model holds **one** `state managelinks.State` (no separate `StateMachine` wrapper — that duplication is what made the original Phase-3 snippet desync `m.state` and `m.stateMachine`).
- **Scope**: `manageLinksModel` only. `form_add_link.go` keeps its own step machine for now.
- **Tests**: add `managelinks/state_test.go` (first tests in the repo).

**Canonical state** (resolves the conflicting definitions):

```go
// pkg/cli/tui/managelinks/state.go
type ViewMode int
const (
    ViewList ViewMode = iota
    ViewActionMenu
    ViewDetails
    ViewDeleteConfirm
)

type OperationState int
const (
    OpIdle OperationState = iota
    OpEnriching
    OpDeleting
)

type OpResult struct {
    Kind    OperationState // OpEnriching | OpDeleting (which op produced this)
    Success bool
    Link    *models.Link   // set on successful enrich; nil otherwise
    Err     error
}

type State struct {
    View   ViewMode
    Op     OperationState
    Result *OpResult
}

func (s *State) ToView(v ViewMode)       { s.View = v; s.Op = OpIdle; s.Result = nil }
func (s *State) Start(op OperationState) { s.Op = op; s.Result = nil }
func (s *State) Complete(r *OpResult)    { s.Op = OpIdle; s.Result = r }
```

**Transition rules**

| From | Trigger | Effect |
|------|---------|--------|
| ViewList | `enter` | `ToView(ViewActionMenu)` |
| ViewActionMenu | `1`/`v` | `ToView(ViewDetails)` |
| ViewActionMenu | `2`/`d` | `ToView(ViewDeleteConfirm)` + focus confirm |
| ViewActionMenu | `3`/`s` | `Start(OpEnriching)` + `enrichLink()` |
| ViewActionMenu | `esc`/`b` | `ToView(ViewList)` |
| ViewDetails | `esc`/`b`/`enter` | `ToView(ViewActionMenu)` |
| ViewDeleteConfirm | `enter`+`y` | `Start(OpDeleting)` + `deleteLink()` |
| ViewDeleteConfirm | `esc` | `ToView(ViewActionMenu)` |
| (op in flight) | `EnrichSuccessMsg` | `Complete{OpEnriching, Success, Link}`; reload links; View stays `ViewActionMenu` |
| (op in flight) | `EnrichErrorMsg` | `Complete{OpEnriching, err}`; View stays `ViewActionMenu` |
| (op in flight) | `DeleteSuccessMsg` | reload links; `ToView(ViewList)`; **clamp `selected` to new len** |
| (op in flight) | `DeleteErrorMsg` | `Complete{OpDeleting, err}`; View → `ViewList` |
| Result shown | any key | clear `Result` (stay in current View) |

**Behavior changes from current code** (intentional, per decisions):

- **Delete success no longer quits the flow** — it returns to the (reloaded) list. Removes `StepDone`.
- **Delete error no longer quits** — it surfaces inline as an `OpResult` error, then returns to the list. (Today `DeleteErrorMsg` calls `tea.Quit`.)
- **The fake enrich "cancel" is removed.** Today `esc` during `StepEnriching` jumps to the action menu while the API call is still running, and the late `EnrichSuccessMsg` overrides it. New rule: while `Op != OpIdle`, ignore all keys except quit; the operation completes and shows its result.

**Rendering** — keep today's **full-screen replacement** (no true overlays). `View()` order: not-ready → loading; non-op `err` → error view; `Op != OpIdle` → operation loading screen; `Result != nil` → result screen (reuse `renderEnrichDone`/`renderSuccessView`); else `switch state.View`. The `renderWithOperationOverlay`/`overlayLoading` helpers from the sketch are **not** needed.

**Don't forget the `SelectableModel` coupling** (omitted from the original plan): `GetSelectedIndex` and `GetListHeaderHeight` must gate on `state.View == ViewList && state.Op == OpIdle && state.Result == nil` instead of `step == StepListLinks`.

**Tests** (`managelinks/state_test.go`): table-driven over `State` methods — `list→action→details→back`, enrich start→success/error result, delete start→success (post-state = `ViewList`)→error, and that `Start`/`Complete` clear `Result` correctly.

---

## Current Problems

The `manageLinksModel` uses a 7-step integer-based state machine that mixes concerns:

1. **View Modes** (what screen to show): List, Action Menu, View Details, Delete Confirm
2. **Operation States** (what async operation is happening): Idle, Enriching, Enrich Done, Done
3. **Mixed Concerns**: Both view and operation state are encoded in a single `step` integer

### Current States (7 steps)

- `StepListLinks` (0) - View mode
- `StepActionMenu` (1) - View mode
- `StepViewDetails` (2) - View mode
- `StepDeleteConfirm` (3) - View mode
- `StepEnriching` (4) - Operation state
- `StepEnrichDone` (5) - Operation state (transient)
- `StepDone` (6) - Operation state (transient)

### Issues

- Transient states (4, 5, 6) are just loading/result states that could be handled differently
- State transitions scattered across multiple handler functions
- No clear separation between "what to show" vs "what's happening"
- Hard to add new operations without adding more steps

---

## Proposed Solution: Separated State Model

### Core Idea

Separate **View Mode** from **Operation State** using a struct-based approach instead of integer steps.

### New State Structure

```go
type ViewMode int
const (
    ViewList ViewMode = iota
    ViewActionMenu
    ViewDetails
    ViewDeleteConfirm
)

type OperationState int
const (
    OpIdle OperationState = iota
    OpEnriching
    OpDeleting
)

type OperationResult struct {
    Type    string // "enrich", "delete"
    Success bool
    Data    interface{} // Link for enrich, nil for delete
    Error   error
}

type manageLinksState struct {
    viewMode       ViewMode
    operationState OperationState
    operationResult *OperationResult // Set when operation completes
}
```

### Benefits

1. **Clear separation**: View mode vs operation state
2. **Fewer states**: 4 view modes + 2 operation states = 6 total (vs 7, but clearer)
3. **Extensible**: Easy to add new operations without touching view modes
4. **Declarative**: State structure is self-documenting

---

## Refactoring Approach

### Phase 1: Extract State Machine Logic (Recommended)

Create a separate package `pkg/cli/tui/managelinks/state.go` that handles state transitions:

```go
package managelinks

type State struct {
    ViewMode       ViewMode
    OperationState OperationState
    Result         *OperationResult
}

type StateMachine struct {
    state State
}

func (sm *StateMachine) TransitionToView(mode ViewMode) {
    sm.state.ViewMode = mode
    sm.state.OperationState = OpIdle
    sm.state.Result = nil
}

func (sm *StateMachine) StartOperation(op OperationState) {
    sm.state.OperationState = op
    sm.state.Result = nil
}

func (sm *StateMachine) CompleteOperation(result *OperationResult) {
    sm.state.OperationState = OpIdle
    sm.state.Result = result
    // Auto-transition: if we were in a view that triggered an operation,
    // go back to action menu after completion
    if sm.state.ViewMode == ViewDeleteConfirm ||
       sm.state.ViewMode == ViewActionMenu {
        sm.state.ViewMode = ViewActionMenu
    }
}
```

### Phase 2: Simplify View Logic

Instead of 7 different cases in `View()`, use composition:

```go
func (m *manageLinksModel) View() string {
    // Handle loading/error states first
    if !m.ready {
        return renderLoadingState("Loading links...")
    }

    if m.err != nil && m.state.OperationState == OpIdle {
        return renderErrorView(m.err)
    }

    // Handle operation states (loading/result overlays)
    if m.state.OperationState != OpIdle {
        return m.renderWithOperationOverlay()
    }

    // Render based on view mode
    switch m.state.ViewMode {
    case ViewList:
        return m.renderList()
    case ViewActionMenu:
        return m.renderActionMenu()
    case ViewDetails:
        return m.renderViewDetails()
    case ViewDeleteConfirm:
        return m.renderDeleteConfirm()
    }
}
```

### Phase 3: Simplify Update Logic

Use the state machine for transitions:

```go
func (m *manageLinksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle global messages
    switch msg := msg.(type) {
    case MenuNavigationMsg:
        return m, nil
    case tea.WindowSizeMsg:
        m.width = msg.Width
        return m, nil
    case LinksLoadedMsg:
        return m.handleLinksLoaded(msg)
    case EnrichSuccessMsg:
        m.stateMachine.CompleteOperation(&OperationResult{
            Type: "enrich", Success: true, Data: msg.Link,
        })
        return m, m.reloadLinks()
    case EnrichErrorMsg:
        m.stateMachine.CompleteOperation(&OperationResult{
            Type: "enrich", Success: false, Error: msg.Err,
        })
        return m, nil
    // ... similar for delete
    }

    // Route to view-specific handlers
    return m.handleViewUpdate(msg)
}

func (m *manageLinksModel) handleViewUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.state.ViewMode {
    case ViewList:
        return m.handleListKeys(msg)
    case ViewActionMenu:
        return m.handleActionMenuKeys(msg)
    case ViewDetails:
        return m.handleViewDetailsKeys(msg)
    case ViewDeleteConfirm:
        return m.handleDeleteConfirmKeys(msg)
    }
}
```

---

## Alternative: Even Simpler Approach (Option B)

If we want to be more aggressive, we can eliminate transient states entirely:

### Remove Transient States

- Instead of `StepEnrichDone`, show result as an overlay/modal on the action menu
- Instead of `StepDone`, show success message and auto-return to list
- Keep only 4 view modes + operation flags

### Implementation

```go
type manageLinksModel struct {
    // ... existing fields

    // Replace step with:
    viewMode ViewMode // 4 modes only

    // Operation tracking (not separate states)
    isEnriching bool
    enrichResult *EnrichResult // nil when not applicable

    isDeleting bool
    deleteResult *DeleteResult
}

// In View(), show overlays when operations are active:
func (m *manageLinksModel) View() string {
    baseView := m.renderViewMode() // Based on viewMode

    // Overlay operation status/results
    if m.isEnriching {
        return overlayLoading(baseView, "Enriching...")
    }
    if m.enrichResult != nil {
        return overlayResult(baseView, m.enrichResult)
    }
    // ... similar for delete

    return baseView
}
```

This reduces from 7 states to 4 view modes + operation flags.

---

## Recommended Implementation Plan

### Step 1: Create State Package (1-2 hours)

- Create `pkg/cli/tui/managelinks/state.go`
- Define `ViewMode`, `OperationState`, `StateMachine`
- Add transition methods

### Step 2: Refactor Model (2-3 hours)

- Replace `step int` with `state State`
- Update all state transitions to use state machine
- Simplify `Update()` method

### Step 3: Simplify View Logic (1-2 hours)

- Refactor `View()` to use view mode + operation overlay pattern
- Remove transient state rendering

### Step 4: Test & Cleanup (1 hour)

- Test all flows (list, view, delete, enrich)
- Remove old step constants
- Update documentation

**Total Effort**: ~5-8 hours (1 day)

---

## Benefits After Refactoring

1. **Clearer Code**: View modes and operations are separate concerns
2. **Easier to Extend**: Add new operations without touching view logic
3. **Better Testability**: State machine can be tested independently
4. **Self-Documenting**: State structure shows what's possible
5. **Fewer Bugs**: Impossible states are harder to create (e.g., can't be in "enriching" and "delete confirm" at once)

---

## Migration Strategy

1. **Keep old code working**: Add new state alongside old `step` field
2. **Gradually migrate**: Update one view mode at a time
3. **Remove old code**: Once all modes migrated, remove `step` field
4. **Update tests**: Ensure all flows still work

This allows incremental refactoring without breaking functionality.
