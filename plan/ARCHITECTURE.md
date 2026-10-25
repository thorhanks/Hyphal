# Architecture

Implements §9.1–9.3 of the spec. The four architectural layers are `tui`, `cli`, `core` and `store` (§9.2). Shared, dependency-free code lives in small **leaf packages** that import nothing from the four layers (D-001).

## 1. Repository layout

```text
cmd/
  hyphal/main.go              # calls cli.Execute(); no logic
  hyphal-keyprobe/main.go     # Phase 0 tool for the human keymap matrix
internal/
  model/       # shared data types: Node, Page, ChangeSet, Op, payloads. No logic beyond (un)marshalling
  ids/         # node and change-set ID generation and validation (CONTRACTS §2)
  fracindex/   # fractional index keys (CONTRACTS §3)
  textfold/    # case folding + diacritic removal with offset maps (§6.2)
  markup/      # inline markup grammar (§5.8): parse, strip, styled runs, wrap/color edits, leading markers
  dialect/     # shared Markdown list dialect (§7.5): parse and write, ^id suffixes
  opml/        # OPML read/write, Workflowy HTML → markup (§7.5)
  layout/      # wrap layout and cursor engine over grapheme clusters (§9.3)
  keymap/      # command IDs, per-context binding tables, validation, overrides (§6.4)
  config/      # TOML config, defaults, validation with line numbers, platform paths (§7.1, §13)
  clipboard/   # platform tools + OSC 52 (§5.5)
  platform/    # flock, open-URL, sync-folder detection, signals helpers
  store/       # SQLite: schema, migrations, commit+history, trash, snapshots, JSON export/import, locking
  core/        # in-memory tree, domain operations, rules engines, undo, search, editor round trip,
               # Markdown/OPML import/export, committer
  cli/         # Cobra commands, JSON envelope, exit codes, `capture` prompt
  tui/         # Bubble Tea app
    theme/     # color tokens → Lip Gloss styles, glyph sets
    pageview/  # Markdown page renderer (goldmark + chroma)
    tuitest/   # test harness (TESTING.md §4)
  testgen/     # test-only: outline fixture notation (CONTRACTS §11), random outline/history generators,
               # temp-DB helpers. May import core and store; imported only by _test packages and benchmarks
docs/          # user docs: tmux snippet, clipboard, Omarchy keybind, agent guide (Phase 4)
packaging/     # hyphal.desktop, AUR PKGBUILD template, capture window rule examples
testdata/      # shared fixtures (OPML, Markdown, legacy DBs); per-package testdata/ alongside packages
plan/          # this plan
```

Sub-packages `tui/theme`, `tui/pageview` and `tui/tuitest` belong to the `tui` layer. `core` may be split into files freely but stays one package unless a card says otherwise.

## 2. Import rules (enforced by a test)

```text
leaf packages (model, ids, fracindex, textfold, markup, dialect, opml, layout,
               keymap, config, clipboard, platform)
    → may import: stdlib, approved third-party deps, `model` (any leaf), and these leaf edges only:
      markup→textfold; dialect→markup,ids; opml→markup; layout→markup.
      keymap and config do not import each other: config keeps key overrides as raw strings
      (map[context]map[command][]string) and keymap.WithOverrides validates them.
store → model, ids, fracindex, platform
core  → store, all leaf packages
cli   → core, store (open/lock/snapshot/export plumbing only), leaf packages
tui   → core, store (open/lock only), leaf packages
cmd/hyphal → cli, tui (sets cli.RunTUI = tui.Run);  cmd/hyphal-keyprobe → bubbletea, keymap
```

- `tui` and `cli` never execute SQL and never import `database/sql` (§9.2).
- `store` never imports `core`.
- A test `internal/archtest` (P0-01) parses imports with `go/packages` (or `go list -json`) and fails on violations.

## 3. Approved dependencies

Pin exact versions in `go.mod` at P0-01 (latest stable at that time). Adding any other module requires asking (README §4 DoD).

| Module | Use |
| --- | --- |
| `charm.land/bubbletea/v2` | TUI loop (§9.1) |
| `charm.land/lipgloss/v2` | Styling |
| `charm.land/bubbles/v2` | viewport/list helpers where useful (not for the outline or page editor) |
| `github.com/charmbracelet/x/exp/teatest/v2` (or the v2-compatible successor found in P0-18) | UI golden tests |
| `github.com/spf13/cobra` | CLI |
| `modernc.org/sqlite` **or** `github.com/ncruces/go-sqlite3` | Chosen in P0-05; default `modernc.org/sqlite` |
| `github.com/rivo/uniseg` | Graphemes, widths, word boundaries |
| `golang.org/x/text` | Case folding, normalization |
| `golang.org/x/sys/unix` | flock |
| `github.com/yuin/goldmark` | Page Markdown parsing with source positions |
| `github.com/alecthomas/chroma/v2` | Code block highlighting |
| `github.com/sahilm/fuzzy` | Palettes (re-evaluated in P0-20) |
| `github.com/pelletier/go-toml/v2` | Config (needs positions for error lines; D-012) |
| `pgregory.net/rapid` | Property tests |

Go version: latest stable at bootstrap (≥ 1.26), set in `go.mod` `go` and `toolchain` lines.

## 4. Package contracts

Signatures below are binding in shape. Names may gain parameters (for example `context.Context`) but not change meaning. Changing them requires a PR with `Human review: yes`.

### 4.1 `model`

```go
package model

type NodeID string        // CONTRACTS §2
type ChangeSetID string

type Node struct {
    ID          NodeID
    ParentID    NodeID     // "" for the root
    Position    string     // fractional key (CONTRACTS §3)
    Text        string     // one line, stored markup (§5.8), never starts with a leading marker (§5.4)
    IsTask      bool
    IsHeading   bool
    CompletedAt *Time
    CreatedAt   Time
    UpdatedAt   Time
    Version     int64
    DeletedAt   *Time
    DeletedBy   ChangeSetID // "" when live
}

type Page struct { NodeID NodeID; Markdown string; UpdatedAt Time; Version int64 }

type Time struct{ time.Time } // marshals as RFC 3339 UTC with milliseconds (§7.2)

type OpKind string // insert, delete, restore, edit_text, edit_page, move, set_task, set_heading, set_complete, set_star

type Op struct {
    NodeID NodeID
    Kind   OpKind
    Before json.RawMessage // CONTRACTS §5; null allowed where specified
    After  json.RawMessage
}

type ChangeSet struct {
    ID          ChangeSetID
    At          Time
    Source      string       // "tui" | "cli" | "editor" | "import" | "restore" | "agent:<name>"
    Description string
    UndoOf      ChangeSetID  // "" if none
    Ops         []Op
}
```

Typed payload structs for each op kind live in `model/payload.go` with `Encode`/`Decode` helpers (CONTRACTS §5).

### 4.2 `store`

```go
package store

type OpenOptions struct {
    Create        bool          // create file and schema if missing (CLI and TUI: true)
    BusyTimeout   time.Duration // CLI: 5s (§8.6); TUI: 5s
    ReadOnly      bool          // force read-only (second TUI)
    Clock         func() time.Time
    SnapshotDir   string        // default: <dir of db>/snapshots (D-014)
}

type ReadOnlyReason int // None, NewerSchema, IntegrityFailed, LockHeld, Forced

func Open(path string, o OpenOptions) (*DB, error)   // pragmas, schema create/migrate (snapshot first), permissions
func (db *DB) Close() error
func (db *DB) ReadOnly() ReadOnlyReason
func (db *DB) Path() string

// Reads
func (db *DB) LoadAll(ctx) (*Dump, error)            // all nodes (live + trashed), page node IDs + versions,
                                                     // stars, view_state, ui_state, meta. No page markdown.
func (db *DB) LoadPages(ctx) (map[model.NodeID]model.Page, error)
func (db *DB) Page(ctx, id) (model.Page, bool, error)
func (db *DB) DataVersion(ctx) (int64, error)        // PRAGMA data_version on a dedicated read connection
func (db *DB) Meta(ctx, key) (string, bool, error)

// Writes with history (§7.3, §8.2). One SQLite transaction.
type Expect struct { NodeID model.NodeID; Version int64; Page bool } // --if-version (§7.3)
type CommitResult struct {
    ChangeSet model.ChangeSet  // as recorded: Before values recomputed from the database
    Stale     []model.NodeID   // nodes whose version differed from what the caller last saw (for messages)
}
func (db *DB) Commit(ctx, cs model.ChangeSet, expect []Expect, seen map[model.NodeID]int64) (CommitResult, error)
// errors: ErrConflict (an Expect failed), ErrInvalid (tree invariant would break: missing/trashed parent,
// cycle, root modified), ErrReadOnly, ErrBusy.

// Writes without history
func (db *DB) SetMeta(ctx, key, value string) error
func (db *DB) SetCollapsed(ctx, map[model.NodeID]bool) error   // view_state, batched
func (db *DB) SetUIState(ctx, key string, value json.RawMessage) error

// Trash (§8.3)
type Deletion struct { ChangeSetID; Top model.NodeID; DeletedAt model.Time; Count int; Hidden bool; HiddenReason string }
func (db *DB) Deletions(ctx, includeHidden bool) ([]Deletion, error)
func (db *DB) Purge(ctx, deletions []model.ChangeSetID, source string) error   // hard delete + cascade (§8.3)
func (db *DB) PurgeOlderThan(ctx, cutoff time.Time, source string) (int, error)

// History (§8.2, §8.4)
func (db *DB) ChangeSets(ctx, filter HistoryFilter) ([]model.ChangeSet, error)
func (db *DB) ChangeSetByID(ctx, id) (model.ChangeSet, error)
func (db *DB) PurgeHistory(ctx, before time.Time) (int, error)

// Safety (§8.1, §8.5)
func (db *DB) QuickCheck(ctx) error
func (db *DB) IntegrityCheck(ctx) error
func (db *DB) SnapshotIfDue(ctx) (name string, taken bool, err error)
func (db *DB) Snapshot(ctx, reason string) (string, error)
func ListSnapshots(dir string) ([]SnapshotInfo, error)
func (db *DB) RestoreSnapshot(ctx, name string) error   // refuses if TUI lock held by another process

// Interchange (§7.5) — lossless JSON only; Markdown/OPML live in core (D-002)
func (db *DB) ExportJSON(ctx, w io.Writer, withHistory bool, exportedBy string) error
func (db *DB) ImportJSON(ctx, r io.Reader, mode ImportMode) error  // Empty | Replace

// Locking (§8.6)
func AcquireTUILock(dbPath string) (*Lock, error)  // ErrLockHeld if another process holds it
func (l *Lock) Release() error
func TUILockHeld(dbPath string) (bool, error)
```

The store applies ops **generically** from their `After` payloads (CONTRACTS §5) and recomputes each op's `Before` from the current rows inside the transaction, so history is always truthful even when the caller's tree was stale.

### 4.3 `core`

```go
package core

// Tree is the in-memory outline. Not safe for concurrent use; owned by one goroutine.
type Tree struct { /* nodes map, sorted child slices, trashed set, pages, stars, collapsed, ui state, tags (P2) */ }

func NewTree(d *store.Dump) *Tree
func (t *Tree) Root() model.NodeID
func (t *Tree) Node(id) (*model.Node, bool)
func (t *Tree) Children(id) []model.NodeID          // live only, ordered by (position, id)
func (t *Tree) Parent(id) model.NodeID
func (t *Tree) Ancestors(id) []model.NodeID          // root first
func (t *Tree) Apply(cs model.ChangeSet) error       // optimistic in-memory application of ops
func (t *Tree) SetPages(map[model.NodeID]model.Page) // background page load
func (t *Tree) Check() []Problem                     // invariants (CONTRACTS §8)

// Visibility (used by rules engines and the TUI).
type ViewOpts struct { Zoom model.NodeID; ShowCompleted bool; Filter *FilterResult; Sticky map[model.NodeID]bool }
type Row struct { ID model.NodeID; Depth int; IsZoomHeading bool; PageMatchLine string }
func (t *Tree) VisibleRows(o ViewOpts) []Row

// Domain operations: pure planners. They never touch the store.
// Each returns a Plan: either a ChangeSet to commit, or a NoOp with a hint message key.
type Plan struct {
    ChangeSet *model.ChangeSet
    NoOp      bool
    Hint      string   // message text for no-ops (§4.5)
    Message   string   // confirmation message for undoable actions
    Cursor    CursorHint
    Expand    []model.NodeID // view-state side effects (indent under collapsed, move into collapsed)
}
type Ctx struct { Tree *Tree; View ViewOpts; Clock func() time.Time; Source string; NewID func() model.NodeID }

func PlanEnter(c Ctx, at TextPos, sel *TextRange) Plan               // §5.2
func PlanBackspaceAtStart(c Ctx, node model.NodeID, sel *TextRange) Plan // §5.3
func PlanDeleteAtEnd(c Ctx, node model.NodeID, sel *TextRange) Plan  // §5.3a
func PlanIndent(c Ctx, nodes []model.NodeID) Plan                    // §5.4
func PlanOutdent(c Ctx, nodes []model.NodeID) Plan
func PlanMove(c Ctx, nodes []model.NodeID, dir Dir) Plan
func PlanDelete(c Ctx, nodes []model.NodeID) Plan
func PlanToggleComplete(c Ctx, nodes []model.NodeID) Plan
// … one Plan* per command; see phase cards.

// Outline couples a Tree with a store and the committer (TUI) or synchronous commits (CLI).
type Outline struct { /* tree, db, committer, undo, source */ }
func OpenOutline(db *store.DB, o OutlineOptions) (*Outline, error)
func (o *Outline) Tree() *Tree
func (o *Outline) Do(p Plan) (model.ChangeSetID, error) // apply to tree, enqueue/perform commit, push undo if TUI
func (o *Outline) Undo() (Plan, error)
func (o *Outline) Redo() (Plan, error)

// Typing (§7.3)
type Typing struct { Node model.NodeID; Before string; Opened, Last time.Time }
func (o *Outline) TypeInto(node model.NodeID, newText string, now time.Time)
func (o *Outline) SealIfDue(now time.Time) bool  // 2 s idle or 5 s since open
func (o *Outline) Seal() (model.ChangeSetID, bool)
func (o *Outline) PendingTyping() *Typing

// Committer (TUI): single FIFO worker for commits and reloads (D-007)
type Committer interface {
    Submit(job Job)              // Commit or Reload
    Results() <-chan JobResult   // delivered to the TUI as tea messages
    Drain(timeout time.Duration) error
}
```

Other `core` areas (each its own file set): `paths.go` (path resolution, §10.2), `search.go` (§6.2), `editor.go` (round trip, §6.3), `paste.go` (§5.5), `trash.go` (restore placement, §8.3), `revert.go` (§8.2), `interchange_md.go`, `interchange_opml.go` (§7.5), `inbox.go` (§6.3), `sample.go` (§4.5).

### 4.4 Leaf packages (headline APIs)

```go
package ids
func NewNode() model.NodeID; func NewChangeSet() model.ChangeSetID; func Valid(s string) bool

package fracindex
func Between(a, b string) (string, error)       // "" means open end
func NBetween(a, b string, n int) ([]string, error)

package textfold
type Folded struct { S string; Map []int } // Map[i] = byte offset in the original for folded byte i
func Fold(s string) Folded
func Index(haystack Folded, needle string) []Range // all matches in original offsets

package markup
type Style uint16  // bitset: Bold, Italic, Strike, Code, Highlight, Link, Marker(dim), Color(3 bits)
type Run struct { Text string; Style Style; Color Color; Src Range; Link *LinkTarget }
func Parse(src string) *Doc
func (d *Doc) Concealed() []Run      // markers hidden (unfocused bullet)
func (d *Doc) Revealed() []Run       // markers shown dim (focused bullet); Text == src, 1:1 offsets
func Plain(src string) (string, []int) // markup stripped, with plain→src offset map (§6.2)
func ConvertLeadingMarker(src string) (rest string, m Leading) // Leading{Task, Completed, Heading}
func WrapSelection(src string, sel Range, marker string) (string, Range)
func SetColor(src string, c Color) string      // whole-bullet color (§5.8)
func ExportColors(src string) string           // <red>…</red> → <span style="color: …">
func Links(src string) []LinkRef               // URLs (P1), node links (P2)

package dialect
type Item struct { Text string; Task, Completed, Heading bool; ID model.NodeID; Children []*Item; Line int }
func Parse(r io.Reader, o ParseOptions) ([]*Item, error) // o.IDs: honor ^id suffixes (editor round trip only)
func Write(w io.Writer, items []*Item, o WriteOptions) error // o.IDs, o.Header (export comment)

package layout
type Line struct { Start, End int; Width int; Indent int } // byte offsets into the source text
func Wrap(runs []markup.Run, width, firstIndent, contIndent int) []Line
type Cursor struct { Offset int; PreferredCol int }
func (l Layout) Up/Down/Left/Right/WordLeft/WordRight/LineHome/LineEnd(c Cursor) (Cursor, bool) // bool=false at edge

package keymap
type Context string // global, text, block, filter, page_view, page_edit, overlay, trash
type CommandID string
type Binding struct { Context Context; Key string; Command CommandID }
func Defaults() Table
func (t Table) Lookup(ctx Context, key string) (CommandID, bool)  // context first, then global
func (t Table) Validate() []Problem       // §6.4 rules
func (t Table) WithOverrides(o map[string]map[string][]string) (Table, []Problem) // raw from config
```

## 5. Runtime model (TUI)

Implements §9.3 and §7.3.

```text
                    ┌───────────── Bubble Tea goroutine ──────────────┐
 key/paste/mouse ──▶│ Model.Update                                    │
 timer tick 250ms ─▶│  • resolve key → command (keymap)               │
 job results ──────▶│  • core.Plan* → Plan                            │
 data_version msg ─▶│  • Outline.Do(plan): Tree.Apply (optimistic)    │──▶ View (rows cache)
                    │    and Committer.Submit(commit job)             │
                    └───────────────┬─────────────────────────────────┘
                                    │ jobs (FIFO)
                    ┌───────────────▼──────────────┐   ┌───────────────────────────┐
                    │ Committer worker goroutine   │   │ tea.Cmd goroutines         │
                    │ commit, reload (LoadAll)     │   │ data_version poll (500ms), │
                    │ retry failed commit every 3s │   │ snapshots, quick_check,    │
                    └───────────────┬──────────────┘   │ page load, export, purge   │
                                    ▼                  └───────────────────────────┘
                                 store.DB
```

Rules:

- **Only the Bubble Tea goroutine mutates the `Tree` and the model.** Other goroutines return values via messages.
- **Commits and reloads go through one FIFO worker** so a reload always reflects every earlier commit (D-007). A failed commit blocks the queue and is retried every 3 s; the model shows the red save label and a banner (§8.1). Typing continues in memory.
- **Optimistic apply**: the planner's change set is applied to the in-memory tree immediately so the next frame shows it (< 16 ms, §9.4). If the store rejects it with `ErrInvalid` (the tree was stale), the worker reloads and the model shows "The outline changed elsewhere; nothing was changed".
- **Typing** is held in `Outline` as an open `Typing`; the tree shows the typed text immediately; the edit is committed when it seals (§7.3). The save dot `●` shows while a `Typing` is open or the queue is non-empty.
- **Reload** (other connection wrote, §7.3): the worker loads a fresh `Dump`; the model swaps the tree, then re-applies any unsealed typing text on top, and keeps zoom, cursor and selection by node ID (falling back to the nearest surviving ancestor).
- **Timers**: one `tea.Tick` every 250 ms drives typing seal checks; the `data_version` poll runs every 500 ms; a 60 s tick checks snapshot due, trash purge due, scheduled export due (P2), and lock takeover (P2, on the data_version tick).
- **Signals**: SIGHUP and SIGTERM are caught with `signal.Notify` and forwarded as a message; Update seals typing and returns `tea.Sequence(drainCmd, tea.Quit)`; `drainCmd` waits up to 3 s for the worker, else writes the recovery file (§8.1).
- **Tests** inject a synchronous committer and a fake clock (TESTING.md §4).

## 6. CLI model

Each invocation: resolve DB path (`--db` → `HYPHAL_DB` → config `db` → default, §10.2) → `store.Open` → `core.OpenOutline` (synchronous commits, source `cli` or `--source`) → run → print → exit code (CONTRACTS §7). Every write command calls `SnapshotIfDue` after committing (§8.5) and `PurgeOlderThan` if the trash purge is due (D-016).

## 7. Coding conventions

- `gofmt`, `go vet`, `golangci-lint` with the config from P0-01 (errcheck, staticcheck, revive basic, gocritic diagnostic tags, misspell, unparam).
- Errors: wrap with `fmt.Errorf("…: %w", err)`; sentinel errors in `store` and `core` (`ErrNotFound`, `ErrAmbiguous`, `ErrConflict`, `ErrBusy`, `ErrReadOnly`, `ErrInvalid`). The CLI maps them to codes (CONTRACTS §7).
- No global mutable state except the default keymap table (immutable after init). Clocks and ID generators are injected.
- No `panic` in production paths except programmer errors on impossible states.
- Logging: none to the terminal. A debug log is written only when `HYPHAL_DEBUG_LOG=<path>` is set (D-018).
- User-visible strings (messages, hints, banners) live in `core/messages.go` and `tui/strings.go` as constants or format functions, so goldens and tests reference one source.
- Comments: explain why, cite § numbers. Do not restate code.
