# Phase 1 · Minimum lovable Workflowy clone

Spec: §11 row 1. Scope: editing rules, block selection, pages (view mode + `$EDITOR`), formatting, theme matching, command palette, CLI, quick capture, Workflowy import. Every Phase 1 *Must* row in §3's parity checklist.

Exit gate: **a week of daily use on your own outline; side-by-side calm review passes** (human). The automated part is P1-54 (parity suite) and P1-55 (interaction matrix).

## Waves and dependencies

Cards in the same wave can run in parallel (separate worktrees) once their listed dependencies are merged.

```text
Wave A (leaves + store):   P1-01 config · P1-02 dialect · P1-03 opml · P1-04 clipboard · P1-05 platform
                           P1-06 trash store · P1-07 history store · P1-08 integrity · P1-09 snapshots
Wave B (core):             P1-10 visibility · P1-11 paths · P1-12 Enter · P1-13 Backspace/Delete
                           P1-14 indent/move · P1-15 small ops · P1-16 filter · P1-22 inbox+sample
                           P1-23 committer · P1-24 TUI outline mode
Wave C (core interchange): P1-17 paste/cut · P1-18 editor round trip · P1-19 pages via editor
                           P1-20 Markdown · P1-21 OPML
Wave D (CLI):              P1-25 skeleton → P1-26 node cmds · P1-27 trash/page/history · P1-28 export/import/…
                           P1-34 capture prompt
Wave E (TUI base):         P1-29 theme → P1-30 shell → P1-31 rendering → P1-32 text editing → P1-33 structure keys
Wave F (TUI features):     P1-35 block selection · P1-36 collapse/zoom · P1-37 undo · P1-38 overlays+palette+help
                           P1-39 pickers + o · P1-40 filter UI · P1-41 clipboard/paste · P1-42 pageview renderer
                           P1-43 page screen · P1-44 $EDITOR subtree · P1-45 Trash view
Wave G (TUI robustness):   P1-46 read-only/reload · P1-47 durability · P1-48 config in TUI · P1-49 first run
Wave H (ship + gate):      P1-50 packaging/docs · P1-51 TUI benchmarks · P1-52 mockups · P1-53 Workflowy import (human)
                           P1-54 parity suite · P1-55 interaction matrix · P1-56 gate
```

Standard messages and hints referenced below are constants in `core/messages.go` (core-generated) or `tui/strings.go` (UI-only). Exact texts given in quotes here are binding; others are the agent's choice (log nothing; they are not contracts).

---

## Wave A — leaves and store

### P1-01 · `config`
Type: impl · Packages: `internal/config` · Depends on: P0-01 · Spec: §7.1, §13, §4.7, §6.4 (keymap config)

Deliverables:
- `Config` struct with every key in §13 and its default; `Paths` (data dir, config file, db, snapshots, exports, runtime dir) per CONTRACTS §9 with `XDG_*` and macOS rules (detect with `runtime.GOOS`).
- `Load(path) (Config, []Problem, error)`: missing file → defaults, no problem. Invalid values → that key falls back to default, `Problem{Line, Key, Message}`. Unknown keys → `Problem{Severity: Warning}`. TOML syntax error → all defaults + one problem with the line.
- `Reload(path, previous) (Config, []Problem)`: if any error-severity problem exists, return `previous` unchanged (§13 "an invalid file is ignored entirely").
- `CommentedTemplate() string`: a full `config.toml` listing every key commented out with its default and a one-line explanation (§4.7, §13). Written when "Edit settings" opens a missing file (P1-48).
- `[keys.<context>]` parsing: `command_id = "key"` or `["key1","key2"]`; returned raw as `Keys map[string]map[string][]string` (config does not import `keymap`; validation happens in `keymap.WithOverrides`, wired in P1-48).
- `capture.terminal` default: `xdg-terminal-exec --app-id=hyphal-capture -e hyphal capture` (verify Omarchy's default terminal flag naming in `docs/` and note it in P2-14).

Acceptance criteria:
- AC1 Empty/missing config gives the §13 defaults → `TestLoad_Defaults` (table of every key).
- AC2 `density = "tight"` at line 3 → default used, problem with line 3 → `TestLoad_InvalidValueLine`.
- AC3 Unknown key → warning, other keys applied → `TestLoad_UnknownKey`.
- AC4 Reload with an invalid file keeps the previous config and returns the problem → `TestReload_InvalidKeepsPrevious`.
- AC5 Linux paths honor `XDG_DATA_HOME`/`XDG_CONFIG_HOME`; macOS data dir is `~/Library/Application Support/hyphal` while config stays `~/.config/hyphal` → `TestPaths_Linux`, `TestPaths_Darwin` (inject GOOS).
- AC6 `CommentedTemplate()` parses as valid TOML and, uncommented, equals the defaults → `TestTemplate_MatchesDefaults`.

### P1-02 · `dialect`: shared Markdown list dialect
Type: impl · Packages: `internal/dialect` · Depends on: P0-16, P0-02 · Spec: §7.5 (dialect), §5.4 (leading markers), §5.5, §6.3

Deliverables: `Parse(r, ParseOptions{IDs bool, SkipHeader bool})`, `Write(w, items, WriteOptions{IDs bool, Header string, ExportColors bool, PageLinks func(*Item) string})`.

Parse rules:
- List markers `-`, `*`, `+`, `1.`/`1)`; indentation spaces or tabs; infer the indent unit from the first indented line (tab = one level; spaces: the smallest positive indent delta). Inconsistent deeper indents attach to the previous line (D-022).
- Lines that are not list items become bullets at the current level (§7.5); blank lines ignored.
- Leading markers converted with `markup.ConvertLeadingMarker` after the list marker (`[ ] `, `[x] `, `## `, combined per D-035).
- `^id` suffix: only when `IDs` is true, and only a trailing ` ^<valid id>`; otherwise text (§6.3). On import (`IDs=false`) a valid trailing ` ^id` is **stripped and ignored** (§7.5: "ignores the suffix"); on paste too (§5.5).
- First line `<!-- … -->` skipped when `SkipHeader`.
- Strip a trailing page link per D-011 and expose it as `Item.PageLink`.
- `_italic_` and `__bold__` converted to `*…*`/`**…**` on import (§5.8 "Importers convert"); not on the editor round trip or paste (decision: paste converts too, since pasted text comes from other apps — log as A-entry if you disagree).
- Text with control chars handled per D-020.

Write rules: two-space indent, `- ` marker, `[ ] `/`[x] ` (completed non-task also `[x]`, §7.5), `## `, text, optional ` ^id`, optional page link.

Acceptance criteria:
- AC1 The §7.5 example parses into the expected items and writes back identically (with IDs) → `TestDialect_SpecExample`.
- AC2 Indentation table: 2-space, 4-space, tabs, mixed numbered markers, non-list lines, jumps of two levels → `TestParse_Indentation`.
- AC3 `^id` handling: valid trailing ID kept only with `IDs`; `a ^b` (invalid) is text; `x ^k3j9x2m4q7tb y` is text → `TestParse_IDSuffix`.
- AC4 Write→Parse round-trip is identity for random items (text from the markup generator, flags, IDs) → `FuzzDialectRoundTrip`, `TestDialect_RoundTripProperty`.
- AC5 Parser never panics → `FuzzParse`.

### P1-03 · `opml`
Type: impl · Packages: `internal/opml` · Depends on: P0-16 · Spec: §7.5 (Workflowy mapping, OPML export)

Deliverables: `Parse(r) ([]*Outline, Stats, error)` with `Outline{Text, Note, Complete, Children}`, and conversion `Text` from Workflowy inline HTML to stored markup per the mapping table: `<b>` → `**`, `<i>` → `*`, `<s>`/`<strike>` → `~~`, `<code>` → `` ` ``, `<a href>` → `[label](url)`, `<span class="colored c-red">` etc. → nearest of 8 colors, background-color classes (`bc-*`) → `==…==`, unknown tags dropped keeping text, entities decoded; escape characters that would become markup (`*`, `~`, `` ` ``, `=`, `<`, `[`, `\`) with `\` when literal. `Stats` counts mirrors when detectable (attributes found in P1-53) and boards. `Write(w, items, head)` for export: plain text (`markup.Plain`), `_note` = page Markdown, `_complete="true"`. The export marker goes in `<head>` (§7.5) as `<title>Exported by hyphal <version></title>` and `<dateCreated>` (RFC 822 date, as OPML 2.0 specifies).

Acceptance criteria:
- AC1 A synthetic Workflowy export (`testdata/opml/workflowy-synthetic.opml`, built by the agent from public descriptions of the format) imports into the expected markup → `TestParse_WorkflowyMapping` (table per mapping row).
- AC2 Leading markers inside OPML text are converted (§5.4) → `TestParse_LeadingMarkers`.
- AC3 Export → parse keeps text (plain), notes and completion → `TestOPMLRoundTrip`.
- AC4 Never panics on malformed XML; returns a parse error → `FuzzParse`.

### P1-04 · `clipboard`
Type: impl · Packages: `internal/clipboard` · Depends on: P0-01 · Spec: §5.5 (transport)

Deliverables: `type Clipboard interface { Write(text string) error; Read() (string, error) }`. `Detect(env, lookPath) Clipboard`: if `SSH_TTY` set or no tool → OSC 52 writer (writes `ESC ] 52 ; c ; <base64> BEL` through a provided `io.Writer` — the TUI routes it through Bubble Tea's raw output; inside tmux wrap in the DCS passthrough only if `TMUX` is set and the agent verifies it is needed — default: plain OSC 52, relying on `set-clipboard on`). Local tools in order: Wayland (`WAYLAND_DISPLAY`) `wl-copy`/`wl-paste`; X11 (`DISPLAY`) `xclip -selection clipboard`, then `xsel --clipboard`; macOS `pbcopy`/`pbpaste`. Tool errors are returned. OSC 52 `Read` returns `ErrUnsupported`. Cut-move detection never reads the clipboard: it compares the bracketed-paste text with the remembered cut (P1-17).

Acceptance criteria:
- AC1 Detection table (env × available tools × GOOS) → `TestDetect`.
- AC2 OSC 52 output bytes exact for a sample string → `TestOSC52_Encoding`.
- AC3 A failing tool (fake binary exiting 1) returns an error → `TestTool_Failure`.

### P1-05 · `platform`
Type: impl · Packages: `internal/platform` · Depends on: P0-01 · Spec: §8.6, §8.1 (sync folder warning), §6.4 (`o` opens URLs)

Deliverables: `Flock(path) (*FileLock, error)` exclusive non-blocking (`unix.Flock LOCK_EX|LOCK_NB`), `IsLocked(path)`; `OpenURL(url)` (`xdg-open` / `open`, detached, no shell); `InSyncFolder(path) (name string, ok bool)` for known roots (`~/Dropbox`, `~/Library/Mobile Documents` (iCloud Drive), `~/Library/CloudStorage/*`, `~/OneDrive`, `~/Google Drive`, any ancestor containing `.stfolder` or `.stignore` (Syncthing)); `WriteFileAtomic` (moved here from P0-12 if not already).

Acceptance criteria:
- AC1 A second process can't take the lock; it is released when the holder exits (even by SIGKILL) → `TestFlock_Exclusive`, `TestFlock_ReleasedOnKill`.
- AC2 Sync folder detection table → `TestInSyncFolder`.
- AC3 `OpenURL` invokes the right command with the URL as a single argument (fake exec) → `TestOpenURL`.

### P1-06 · Store: Trash queries and purge
Type: impl · Packages: `internal/store` · Depends on: P0-08 · Spec: §8.3, §8.4 · D-015, D-016, D-027

Deliverables: `Deletions(includeHidden)` (one row per `deleted_by` among currently trashed nodes; top node = the trashed node of that deletion whose parent is not trashed by the same change set; count; hidden per D-027 using the op rows), `Purge(deletions, source)`, `PurgeOlderThan(cutoff, source)`, `EmptyTrash(source)`. Purge deletes `nodes` rows of the deletion plus any descendants still trashed from earlier deletions (§8.3), with their `pages`, `stars`, `view_state` rows, in one transaction, and records an empty change set (D-015). Never touches `ops`.

Acceptance criteria:
- AC1 Two deletions → two rows, newest first, with counts → `TestDeletions_List`.
- AC2 Merge leftovers and single empty leaves are hidden unless `includeHidden` → `TestDeletions_Hidden`.
- AC3 Purging a deletion also purges earlier-trashed descendants; no trashed row references a purged parent → `TestPurge_Cascade`.
- AC4 History ops for purged nodes remain → `TestPurge_KeepsHistory`.
- AC5 `PurgeOlderThan` purges by `deleted_at` → `TestPurgeOlderThan`.

### P1-07 · Store: history queries and purge
Type: impl · Packages: `internal/store` · Depends on: P0-08 · Spec: §8.2, §8.4, §10.2

Deliverables: `ChangeSets(HistoryFilter{Source, Since, Until, NodeID, Limit, WithOps})` newest first; `ChangeSetByID`; `PurgeHistory(before)` deleting change sets with `at < before` and their ops (one transaction).

Acceptance criteria:
- AC1 Filter by source `agent:claude` and by node → `TestChangeSets_Filter`.
- AC2 `PurgeHistory` removes only older change sets and their ops → `TestPurgeHistory`.

### P1-08 · Store: integrity checks
Type: impl · Packages: `internal/store` · Depends on: P0-09 · Spec: §8.1, §10.2 (`check`) · CONTRACTS §8

Deliverables: `QuickCheck`, `IntegrityCheck`, `CheckInvariants(ctx) []Problem` (SQL implementation of CONTRACTS §8 items 1–10, independent of `core` so it works on any file).

Acceptance criteria:
- AC1 Each invariant violation is reported on a crafted DB (write rows with raw SQL) → `TestCheckInvariants` (table).
- AC2 A byte-corrupted DB copy fails `QuickCheck` → `TestQuickCheck_Corrupt`.

### P1-09 · Store: snapshots
Type: impl · Packages: `internal/store` · Depends on: P0-06, P1-05 · Spec: §8.5, §7.4 · CONTRACTS §9 · D-014

Deliverables: `Snapshot(reason)` (backup API to `*.tmp`, `quick_check` on the copy, rename, 0600; on failure delete and return error), `SnapshotIfDue` (due when the newest good snapshot is > 24 h old **and** the outline changed since it — compare max `change_sets.at` with `meta.last_snapshot_at`), retention (`keep` per DB name; daily snapshots count toward keep; pre-* snapshots too — keep the newest N overall, D-014), `ListSnapshots`, `RestoreSnapshot(name)` (refuse with `ErrTUIRunning` if another process holds the TUI lock; take `pre-restore` snapshot; copy the snapshot **into** the live DB with the backup API, never rename; record a change set with source `restore` and description `Restored snapshot <name>` and no ops; set a `meta` flag `restore_banner` = name so the TUI can show the informational banner, §4.5).

Acceptance criteria:
- AC1 Snapshot passes quick_check and has mode 0600; failure leaves no file → `TestSnapshot_Verified`.
- AC2 Due logic: not due within 24 h; not due if unchanged; due otherwise → `TestSnapshotIfDue` (fake clock).
- AC3 Only the newest `keep` are kept → `TestSnapshot_Retention`.
- AC4 Restore takes a pre-restore snapshot first, so restoring that one returns the previous state → `TestRestore_Reversible`.
- AC5 Restore refuses while a lock holder process exists → `TestRestore_RefusesWithTUILock`.
- AC6 Restore while `-wal` has uncheckpointed data yields a consistent DB (integrity_check ok) → `TestRestore_WithWAL`.

---

## Wave B — core

### P1-10 · Visibility, view state and cursor memory
Type: impl · Packages: `internal/core` · Depends on: P0-10 · Spec: §4.1, §5.4 (filter stickiness), §6.1, §7.2 (view/ui state) · D-026, D-036

Deliverables:
- `Tree.VisibleRows(ViewOpts)`: rows under the zoom (zoom heading first unless zoom is Home, D-036), skipping collapsed descendants, completed nodes when `ShowCompleted` is false (a completed node hides with its subtree), and nodes outside the filter result (matches + ancestors), but always including `Sticky` nodes (created/edited under the filter, §5.4) and their ancestors.
- Helpers: `PrevVisible(rows, id)`, `NextVisible(rows, id)`, `IsTopLevelOfZoom`.
- View state API on `Outline`: `SetCollapsed(ids, bool)` (persists through `store.SetCollapsed`, no history), `ExpandToLevel(ids, n)` (D-026), `CollapseAllInZoom()`, `SetShowCompleted`, `SetZoom`, `RememberCursor(zoom, node, offset)`, `CursorFor(zoom)`. Persist ui_state keys (CONTRACTS §4).
- `Breadcrumb(zoom, maxWidth) []Crumb` with middle truncation `Home › … › Development › Tooltips` (§4.1).

Acceptance criteria:
- AC1 Collapsed nodes hide descendants; zoom shows only the target's descendants with the heading first → `TestVisibleRows_CollapseZoom` (fixtures).
- AC2 `ShowCompleted=false` hides completed nodes and their subtrees → `TestVisibleRows_HideCompleted`.
- AC3 Sticky nodes stay visible under a filter even when they no longer match → `TestVisibleRows_Sticky`.
- AC4 Expand-to-level table for N = 1, 2, 9 and `0` → `TestExpandToLevel`.
- AC5 Breadcrumb truncation keeps Home, the last two crumbs and `…` within width → `TestBreadcrumb_Truncation`.
- AC6 Collapse/zoom changes create no change sets → `TestViewState_NoHistory`.

### P1-11 · Path resolution
Type: impl · Packages: `internal/core` · Depends on: P0-10 · Spec: §10.2 (paths) · CONTRACTS §7.3 · D-010

Deliverables: `Resolve(tree, ref string, opts) (NodeID, error)` with `ErrNotFound`, `ErrAmbiguous{Candidates}`; `PathOf(tree, id) string` (escaping `/` and `\`).

Acceptance criteria:
- AC1 Exact match beats case-insensitive; two exact matches → ambiguous with both candidates → `TestResolve_Matching`.
- AC2 Markup stripped for matching (`**Projects**` matches `Projects`) → `TestResolve_Plain`.
- AC3 `\/` and `\\` escapes; `/` and empty = Home; `id:` / `path:` prefixes → `TestResolve_Syntax`.
- AC4 `PathOf` → `Resolve` round-trips for random trees with slashes in text → `TestPathRoundTrip`.
- AC5 Trashed nodes never match paths; IDs of trashed nodes resolve only with `opts.AllowTrashed` → `TestResolve_Trashed`.

### P1-12 · Enter rules
Type: impl · Packages: `internal/core` · Depends on: P1-10 · Spec: §5.2, §5.4 (filter), §5.4 (leading markers) · D-029, D-036

Deliverables: `PlanEnter(ctx, pos TextPos, sel *TextRange) Plan`. `TextPos{Node, Offset}`; the zoom heading is the zoom node itself with `IsZoomHeading`. Selection: delete the selected text first (part of the same change set as the resulting action, folded into the `edit_text`), then apply the row for the resulting position. Split: first half keeps ID and everything attached (§5.2 last paragraph); second half = new node inserted directly after the first half (after its subtree, as next sibling). New nodes created under a filter are added to `Sticky`.

Notes: Enter seals pending typing first (the caller does `Outline.Seal()` before planning). The new empty node's `edit_text` typing begins after it.

Acceptance criteria → `TestEnter` table (fixture notation, CONTRACTS §11), one case per row of §5.2 plus:
- AC1 Start of non-empty: `- ‸Alpha` → `- \n- ‸Alpha`.
- AC2 Middle: `- Al‸pha\n  - Child` → `- Al\n  - Child\n- ‸pha`.
- AC3 End, expanded with children: `- Alpha‸\n  - Child` → `- Alpha\n  - ‸\n  - Child`.
- AC4 End, collapsed: `- Alpha‸ {collapsed}\n  - Child\n- Beta` → `- Alpha {collapsed}\n  - Child\n- ‸\n- Beta`.
- AC5 End of leaf: `- Alpha‸\n- Beta` → `- Alpha\n- ‸\n- Beta`.
- AC6 Empty, nested: `- Alpha\n  - ‸\n  - Gamma` → `- Alpha\n  - Gamma\n- ‸` (outdent; later siblings stay).
- AC7 Empty, top level of zoom: `- Alpha\n- ‸` → `- Alpha\n- \n- ‸`.
- AC8 Zoom heading: `- Proj‸ects {zoom}\n  - A` → `- Projects {zoom}\n  - ‸\n  - A` (never split).
- AC9 Selection: `- Al⟦ph⟧a` → `- Al\n- ‸a`; `- Al⟦pha⟧` → `- Al\n- ‸`.
- AC10 Split keeps page, star, completion and ID on the first half: `- Al‸pha {page star id=aaaaaaaaaaaa}` → `- Al {page star id=aaaaaaaaaaaa}\n- ‸pha`.
- AC11 Under a filter, the created bullet is sticky → `TestEnter_UnderFilterSticky`.
- AC12 One change set per Enter, containing the edit and insert ops → `TestEnter_SingleChangeSet`.

### P1-13 · Backspace and Delete rules (merges)
Type: impl · Packages: `internal/core` · Depends on: P1-10 · Spec: §5.3, §5.3a, §8.3 · D-027

Deliverables: `PlanBackspaceAtStart`, `PlanDeleteAtEnd`. Merge change set: `edit_text` on the survivor (concatenate, no separator), `move` ops adopting children (appended in order), page transfer (`edit_page` create on survivor + `edit_page` delete on merged-away) when only the merged-away has a page, star retarget (`set_star` on survivor if it isn't starred, unstar merged-away) — Phase 2 adds link retargeting via a hook `retargetLinks(from, to)` that is a no-op now — then `delete` with `merged_into`. Cursor at the join point. Rows checked top to bottom, first match wins. Hidden bullets under a filter are skipped when finding previous/next visible.

Hints (binding text):
- Empty with children: `"Use Esc then Delete to delete a bullet with children"`.
- Both pages: `"Both bullets have pages; move one page first"`.
- Both have children: `"Both bullets have children; move them first"`.
- Delete into own first child: `"Can't merge a bullet with its own child"`.

Acceptance criteria → `TestBackspace` and `TestDelete` tables, one case per row:
- AC1 Selection deletes only the text: `- Al⟦ph⟧a` → `- Al‸a`.
- AC2 No previous visible bullet: `- ‸Alpha` (zoom Home) → no-op, no hint.
- AC3 Empty leaf: `- Alpha\n- ‸` → `- Alpha‸`; at zoom heading: `- P {zoom}\n  - ‸` → `- P‸ {zoom}`.
- AC4 Empty with children → no-op with hint.
- AC5 Previous is the zoom heading: `- P {zoom}\n  - ‸A` → no-op.
- AC6 Both pages → no-op with hint.
- AC7 Text, no children: `- A\n  - A1\n- ‸B` → `- A\n  - A1‸B`.
- AC8 Text + children, previous without children: `- Alpha\n- ‸Beta\n  - B1` → `- Alpha‸Beta\n  - B1`.
- AC9 Text + children, previous has children: `- A {collapsed}\n  - A1\n- ‸B\n  - B1` → no-op with hint.
- AC10 Page and star move to the survivor; merged-away goes to the Trash with `merged_into` and is hidden from `Deletions(false)` → `TestMerge_Identity`.
- AC11 Delete rows mirror: no next; next is own first child; next empty leaf deleted; both pages; text merge; adopt children; both children → `TestDelete` (7 cases).
- AC12 Under a filter, hidden bullets are skipped → `TestBackspace_SkipsHidden`.

### P1-14 · Indent, outdent, move
Type: impl · Packages: `internal/core` · Depends on: P1-10 · Spec: §5.4

Deliverables: `PlanIndent(nodes)`, `PlanOutdent(nodes)`, `PlanMove(nodes, Up|Down)`. `nodes` is a contiguous run of siblings (block selection) or one node. All use the **real** tree including hidden bullets (§5.4); moved nodes become sticky under a filter.

Algorithms:
- Indent: previous real sibling P of the first node; none → no-op (silent). Nodes become P's last children, in order; if P is collapsed, `Plan.Expand = [P]`.
- Outdent: parent Q; if Q is the zoom target → no-op with hint `"Already at the top level of this zoom"`. Nodes become Q's next siblings (inserted right after Q, in order); later siblings of the nodes stay under Q.
- Move up (first node N, parent Q): if N has a previous sibling S → place the run before S. Else if Q is the zoom target → no-op. Else if Q has a previous sibling A → run becomes A's last children (`Expand=[A]` if collapsed). Else → run moves up a level, placed right before Q under Q's parent.
- Move down mirrors: next sibling after the run → place after it; else Q is zoom → no-op; else Q's next sibling B → run becomes B's first children (expand B); else → placed right after Q.

Acceptance criteria → tables:
- AC1 Indent: first sibling no-op; under collapsed sibling expands it; multi-selection keeps order → `TestIndent`.
- AC2 Outdent: becomes next sibling of old parent; later siblings not adopted; top level of zoom → hint → `TestOutdent`.
- AC3 Move up/down within siblings → `TestMove_Siblings`.
- AC4 Cross-parent: B's first child up → last child of A (A expands); first child of the first sibling up → becomes its parent's previous sibling; A's last child down → first child of B; mirror → `TestMove_CrossParent` (the four §5.4 examples).
- AC5 Zoom edges are no-ops → `TestMove_ZoomEdges`.
- AC6 Hidden siblings participate (move up past a hidden completed sibling swaps with it) → `TestMove_HiddenSiblings`.

### P1-15 · Small node operations
Type: impl · Packages: `internal/core` · Depends on: P1-10 · Spec: §5.4, §5.8 (color), §6.6, §8.3 (cursor after delete)

Deliverables:
- `PlanToggleComplete(nodes)`: any open → complete all (`completed_at = now`), else reopen all; reopening keeps `is_task` (§7.2). Cursor hint: if a newly completed bullet will hide (`ShowCompleted=false`), move to the next visible bullet; message `"Completed \"<text>\" · Ctrl+Z undoes"` (truncate text to 40 cells).
- `PlanToggleTask`, `PlanToggleHeading` (mixed selection: any off → all on).
- `PlanDuplicate(nodes)`: deep copy as next siblings (after the last selected) with new IDs, pages, completion, task, heading; no stars; links unchanged.
- `PlanSetColor(nodes, color|none)` using `markup.SetColor` (§5.8), `PlanWrapStyle(nodes, Bold|Italic)` for whole bullets from the palette (wrap whole text; if already wrapped entirely, unwrap — decision: toggle).
- `PlanMoveTo(nodes, parent)`: append as last children (D-028); refuse moving into own subtree (`ErrInvalid` → hint).
- `PlanDelete` cursor rule (§8.3): next sibling, else previous sibling, else parent; message `"Moved \"<text>\" and N descendants to Trash · Ctrl+Z undoes"` (`"Moved \"<text>\" to Trash · Ctrl+Z undoes"` when N = 0).
- `PlanEditText(node, text)` applies `ConvertLeadingMarker` (§5.4) and D-020 cleanup; used by typing seal, CLI `edit --text`, editor round trip.

Acceptance criteria:
- AC1 Mixed-state complete rule → `TestToggleComplete_Mixed`.
- AC2 Completing with hide-completed moves the cursor to the next visible bullet and returns the message → `TestToggleComplete_HidesAndMovesCursor`.
- AC3 Reopen keeps `is_task` → `TestReopen_KeepsTask`.
- AC4 Duplicate copies subtree with new IDs, copies pages, drops stars → `TestDuplicate`.
- AC5 Color replaces existing color tag; "No color" removes it → `TestSetColor_Bullets`.
- AC6 Delete cursor rule and message counts → `TestDelete_CursorAndMessage`.
- AC7 Typing `[ ] ` at the start converts to a task and strips the marker in the sealed edit → `TestEditText_LeadingMarker`.
- AC8 Move to own descendant is refused with a hint → `TestMoveTo_IntoSelf`.

### P1-16 · Filter search engine (Phase 1)
Type: impl · Packages: `internal/core` · Depends on: P1-10, P0-20 · Spec: §6.2 · D-021

Deliverables: `Filter(tree, query) FilterResult{Matches map[NodeID][]Range (plain-text ranges), PageLines map[NodeID]string, Count int}`; per-node cached folded plain text (invalidated on text change); page text included once `Tree.SetPages` ran (§6.2: loaded in the background after the first frame); trashed nodes never match; incremental refinement when the new query extends the previous (search only previous matches). `PageLines` holds the first matching page line, trimmed to 80 cells. Range mapping to rendered (concealed) text uses `markup.Plain` offsets.

Acceptance criteria:
- AC1 Terms AND, case/diacritic-insensitive, match on plain text (`**bold** text` matches "bold text") → `TestFilter_Matching`.
- AC2 Matches include collapsed descendants and page text (page match keeps the bullet with `PageLines`) → `TestFilter_CollapsedAndPages`.
- AC3 Trashed nodes never match → `TestFilter_Trash`.
- AC4 Highlight ranges map to the concealed render positions → `TestFilter_RangesMapToConcealed`.
- AC5 `BenchmarkFilter100k` within target or documented (P0-20).

### P1-17 · Paste and cut-move
Type: impl · Packages: `internal/core` · Depends on: P1-12, P1-02, P1-15 · Spec: §5.5 · D-029, D-030

Deliverables:
- `PlanPaste(ctx, target PasteTarget, text string) Plan`: single line (after D-020 cleanup) → text insert replacing the selection (an `edit_text`, merged into pending typing by the caller; leading markers convert only if the result starts with one). Multi-line → `dialect.Parse` (IDs ignored) → subtree inserted per §5.5: end of bullet → siblings after it; mid-text → split (§5.2) and insert between halves; empty bullet → replaces it (the empty node is deleted — use the empty node's ID for the first top-level item instead of deleting, to avoid a Trash entry: decision, log A-entry if changed); block selection → siblings after the selection; zoom heading → first children (D-029).
- Copy: `CopyMarkdown(tree, nodes) string` (dialect, no IDs, no pages, colors kept as typed).
- Cut-move: `CutRecord{Text string, ChangeSetID, Tops []NodeID}` remembered by the TUI. When the pasted text (from bracketed paste) equals `rec.Text` after normalizing line endings and one trailing newline, the clipboard still holds what cut wrote (§5.5). `PlanPasteCut(ctx, target, rec)`: if those nodes are still trashed by that change set → one change set with `restore` ops + `move` ops to the paste position (same IDs, pages, stars, history). Otherwise fall back to `PlanPaste`.

Acceptance criteria:
- AC1 Each §5.5 placement case (end, mid-text, empty, block selection, zoom heading) → `TestPaste_Placement` (fixtures).
- AC2 Indented Markdown with `[ ]`, `[x]`, `##`, `^id` → tasks, completed, headings, new IDs → `TestPaste_Dialect`.
- AC3 Copy output for a subtree with markup and colors → `TestCopyMarkdown` (golden).
- AC4 Cut then paste of the same text restores the same IDs and page at the new position, as one change set; the cut stays undoable separately → `TestCutPaste_IsMove`.
- AC5 Cut, then the clipboard changes (different text) → paste creates new nodes → `TestCutPaste_ClipboardChanged`.

### P1-18 · `$EDITOR` round trip (core)
Type: impl · Packages: `internal/core` · Depends on: P1-02, P1-15 · Spec: §6.3 · D-022

Deliverables:
- `EditorExport(tree, scope EditScope) (text string, manifest Manifest)`: scope = focused node | selection | zoom; includes **every live descendant** regardless of collapse, completion visibility or filter (§6.3); each line ends with ` ^id`.
- `EditorApply(ctx, manifest, edited string) (Plan, Summary, error)`: parse with IDs; dedupe IDs (first wins, later lines new); IDs outside the manifest or trashed → new nodes (D-022); compute ops: `edit_text`/`set_task`/`set_complete`/`set_heading` for changed lines; `insert` for new; `delete` for manifest nodes missing from the file (with their whole remaining subtrees? No — only nodes missing; children that remain get re-parented per their file position, §6.3); `move` for changed parent or order (minimize moves: keep the longest increasing subsequence of existing keys per parent, generate keys for the rest). Source `editor`. Summary `"3 changed, 2 new, 1 to Trash"` (omit zero parts; `"No changes"`).
- Parse failure → `ErrParse{Line}`; nothing applied.
- Concurrency: the plan is computed against the tree at save time; the caller commits it after any intervening changes (§7.3); `Stale` from the commit produces the D-033 message.

Acceptance criteria (scripted edits, §9.5):
- AC1 Rename, reorder, re-indent, add line, remove line → exact op sets → `TestEditorApply_Basic` (table).
- AC2 Collapsed, completed-hidden and filtered-out descendants are in the file → `TestEditorExport_IncludesHidden`.
- AC3 Deleting a parent line while keeping its children re-parents them by indentation → `TestEditorApply_DeleteParentKeepChildren`.
- AC4 Duplicated ID → first keeps the node, second new → `TestEditorApply_DuplicateID`.
- AC5 Stripped ID → new node + old to Trash, visible in the summary → `TestEditorApply_StrippedID`.
- AC6 Foreign or trashed ID → new node; the foreign node is untouched → `TestEditorApply_ForeignID`.
- AC7 Malformed input that the dialect rejects → error, no plan → `TestEditorApply_ParseError`.
- AC8 Export → Apply with no edits → "No changes", no change set → `TestEditorApply_Identity` (property over random trees).

### P1-19 · Page editing through the external editor (core)
Type: impl · Packages: `internal/core` · Depends on: P0-10 · Spec: §5.7

Deliverables: `PageEditorText(tree, node) string` = `# <bullet text>\n\n<markdown>` (no blank body line when empty: `# Title\n`); `PageEditorApply(ctx, node, edited) Plan`: if the first line is an H1 (`# …`), its text is the title → `edit_text` if changed (leading-marker conversion applies); the rest (with one leading blank line stripped) is the page; otherwise the whole file is the page and the title is unchanged. Empty page text → no page created / existing page deleted (§5.7). One change set (`source=tui`, as it's started from the TUI). `PlanDeletePage(node)`.

Acceptance criteria:
- AC1 Title change renames the bullet; body saved → `TestPageEditor_Rename`.
- AC2 First line not H1 → whole file is the page, title unchanged → `TestPageEditor_NoTitleLine`.
- AC3 The §5.7 example: deleting the title line of a page whose body starts with `# Intro` renames the bullet to "Intro" → `TestPageEditor_IntroExample`.
- AC4 Saving empty on a new page creates nothing; clearing an existing page deletes it (one change set) → `TestPageEditor_Empty`.

### P1-20 · Markdown import and export (core)
Type: impl · Packages: `internal/core` · Depends on: P1-02 · Spec: §7.5 · CONTRACTS §10 · D-011

Deliverables: `ExportMarkdown(tree, pages, root, ExportOptions{IDs, Split, OutDir, Version})` writing `outline.md` (or split files) + `pages/` atomically, headers, color export via `markup.ExportColors`, page links (D-011), trashed nodes excluded; to an `io.Writer` when no `OutDir` and not split. `PlanImportMarkdown(ctx, reader, filename, under)` → new top-level bullet named after the file (basename with extension, §7.5) unless `under` is set; pages attached from `pages/` when present (D-011); source `import`.

Acceptance criteria:
- AC1 Export of a fixture tree with pages equals golden files (outline.md + pages/*) → `TestExportMarkdown_Golden`.
- AC2 `--split` layout → `TestExportMarkdown_Split`.
- AC3 Slug rules and collision-free names → `TestPageSlug`.
- AC4 Import lands under a new top-level bullet named after the file, or directly under `--under` → `TestImportMarkdown_Placement`.
- AC5 Export → import keeps text, tasks, completion (non-task completed becomes completed task, documented loss), headings, pages, structure → `TestMarkdownRoundTrip`.
- AC6 Logseq and Obsidian sample exports (`testdata/markdown/logseq-sample.md`, `obsidian-outliner-sample.md`, built by the agent) import without loss of text or structure → `TestImportMarkdown_Samples`.

### P1-21 · OPML import and export (core)
Type: impl · Packages: `internal/core` · Depends on: P1-03 · Spec: §7.5

Deliverables: `PlanImportOPML(ctx, reader, filename, under) (Plan, ImportSummary)` (pages from `_note`, completion from `_complete`, summary counts nodes, pages, mirrors if detectable); `ExportOPML(tree, pages, root, w)`.

Acceptance criteria:
- AC1 Workflowy synthetic fixture imports with correct text/markup, pages and completion → `TestImportOPML_Workflowy`.
- AC2 Export contains plain text, `_note`, `_complete`, head elements → `TestExportOPML_Golden`.

### P1-22 · Inbox, quick add and first-run sample
Type: impl · Packages: `internal/core` · Depends on: P0-10 · Spec: §6.3 (quick capture), §4.5 (first run) · D-024

Deliverables:
- `PlanAdd(ctx, text, under, anchor)`: default target = Inbox. `EnsureInbox`: `meta.inbox_id` missing, trashed or purged → create a new top-level `Inbox` (appended at the end of Home's children), set `meta.inbox_id` (after commit), and return message `"Created a new Inbox"` (§6.3). A trashed old Inbox stays untouched.
- `SampleOutline` embedded from `core/sample.md` (dialect) teaching the core keys. Content (binding):
  ```markdown
  - Welcome to Hyphal
    - Type to edit a bullet. Enter adds a new one, Tab indents it, Shift+Tab outdents
    - Press Esc to select a bullet, then use single keys
      - z zooms in, Z zooms out, - and = collapse and expand
      - x completes, d deletes (it goes to the Trash), [ and ] move it
      - p opens the bullet's page, for long notes
    - [ ] Try completing this task: Esc, then x
    - Ctrl+P lists every command, and F1 shows every key
    - When you're done, delete this whole branch: select "Welcome to Hyphal" with Esc, then press d
  ```
- `PlanFirstRun(tree)`: if `meta.tui_opened` unset and Home has no children other than the Inbox → insert the sample (source `tui`); set `tui_opened` either way.

Acceptance criteria:
- AC1 Add without Inbox creates one and records its ID; second add reuses it → `TestAdd_CreatesInbox`.
- AC2 Renamed/moved Inbox is still the target → `TestAdd_FollowsInboxID`.
- AC3 Trashed Inbox → new Inbox, message, old stays trashed → `TestAdd_TrashedInbox`.
- AC4 First run inserts the sample once; not when the outline has other content (e.g. after import) → `TestFirstRun`.

### P1-23 · Committer, retries, reload, recovery files
Type: impl · Packages: `internal/core` · Depends on: P0-11, P0-13 · Spec: §7.3, §8.1, §9.3 · D-007, D-033

Deliverables: `Committer` (ARCHITECTURE §4.3) with jobs `Commit{cs, expect, seen}` and `Reload`; results `Committed{result}`, `CommitFailed{err, attempt}` (retry every 3 s; `ErrInvalid` is not retried: it triggers a Reload and reports `Rejected`), `Reloaded{dump}`; `Drain(timeout)`. `SyncCommitter` for tests/CLI. `WriteRecoveryFile(dataDir, dbName, tree, pending []model.ChangeSet, typing *Typing) (path, error)`: dialect with `^id`s of the nodes affected by uncommitted change sets and unsealed typing, each with a comment header describing what didn't save (CONTRACTS §9).

Acceptance criteria:
- AC1 Jobs execute in submission order; a reload after commits reflects them → `TestCommitter_FIFO`.
- AC2 A failing store (injected) retries every 3 s, later jobs wait, success resumes → `TestCommitter_Retry` (fake clock).
- AC3 `ErrInvalid` → Rejected + automatic reload → `TestCommitter_Rejected`.
- AC4 `Drain` returns after the queue empties or times out → `TestCommitter_Drain`.
- AC5 Recovery file contains the unsaved text and IDs, mode 0600 → `TestRecoveryFile`.

### P1-24 · `Outline` in TUI mode: undo with cursor, messages
Type: impl · Packages: `internal/core` · Depends on: P1-23, P0-13 · Spec: §5.6

Deliverables: `Outline` async mode wiring `Do` → `Tree.Apply` + `Committer.Submit` + undo push (optimistic; replaced by recorded ops on `Committed`; removed on `Rejected`); `Undo()`/`Redo()` return a `Plan` whose `Cursor` is the stored cursor-before and a `ZoomTo` when the affected nodes are outside the current zoom (§5.6), message `"Undid: <description>"` / `"Redid: <description>"` and, when fields were skipped, `"Undid: <description> · N fields changed elsewhere were kept"`; Undo seals typing first (§5.6); reload merging (`ApplyReload(dump)` keeps unsealed typing on top, §7.3).

Acceptance criteria:
- AC1 Ctrl+Z right after typing undoes the typing just done (seal first) → `TestUndo_SealsTypingFirst`.
- AC2 Undo restores the stored cursor and selection → `TestUndo_RestoresCursor`.
- AC3 Undo of a change outside the zoom returns `ZoomTo` → `TestUndo_ZoomsToShow`.
- AC4 CLI change sets are never on the stack → `TestUndo_IgnoresCLIChanges`.
- AC5 Reload with unsealed typing keeps the typed text and later commits it after the other change → `TestReload_KeepsTyping`.

---

## Wave D — CLI

### P1-25 · CLI skeleton
Type: impl · Packages: `internal/cli`, `cmd/hyphal` · Depends on: P1-01, P1-11, P0-12 · Spec: §10.2 · CONTRACTS §7

Deliverables: Cobra root `hyphal` (no subcommand → start the TUI via a function variable set by `tui` to avoid an import cycle: `cli.RunTUI = tui.Run` in `cmd/hyphal/main.go`); persistent flags `--db`, `--json`, `--source` (validate `cli` | `agent:<name>`; `tui`, `editor`, `import`, `restore` are reserved), `--version`; DB resolution order (§10.2); envelope printer; error→code→exit mapping (CONTRACTS §7.2) in one place; human printers; version string injected by `-ldflags` (default `dev`).

Acceptance criteria:
- AC1 `--db` beats `HYPHAL_DB` beats config → `TestResolveDB`.
- AC2 Each error code maps to its exit status and envelope → `TestErrorMapping` (table).
- AC3 Usage errors exit 2 with `usage` code under `--json` → `TestUsageError`.
- AC4 `--source agent:claude` is recorded; `--source tui` is rejected → `TestSourceFlag`.

### P1-26 · CLI node commands
Type: impl · Packages: `internal/cli` · Depends on: P1-25, P1-15, P1-16, P1-22 · Spec: §10.2

Commands: `add "<text>" [--under N] [--after N | --before N]`, `show N [--depth K]`, `tree [--root N] [--depth K]`, `search "<query>"` (Phase 1 semantics, D-021), `edit N --text T [--if-version V]`, `set N [--task|--no-task] [--heading|--no-heading]`, `move N --under P [--after S | --before S | --index I]`, `complete N`, `reopen N`, `delete N`. Every write: one change set, then `SnapshotIfDue` and purge-due check (ARCHITECTURE §6). Data shapes per CONTRACTS §7.4.

Acceptance criteria (CLI goldens, TESTING §3):
- AC1 Each command's `--json` output matches its golden → `TestCLI_<Command>_JSON` (one per command).
- AC2 `edit --if-version` mismatch → exit 4, `conflict` → `TestCLI_Edit_Conflict`.
- AC3 Ambiguous path → exit 3 with candidates → `TestCLI_Ambiguous`.
- AC4 `add` lands in the Inbox by default; `--under`/`--after` placement → `TestCLI_Add_Placement`.
- AC5 `move --index` places at that index among live siblings; moving under own descendant → `invalid` → `TestCLI_Move`.
- AC6 Editing the root → `invalid` → `TestCLI_RootImmutable`.
- AC7 Text with a leading `[ ] ` becomes a task (§5.4) → `TestCLI_Edit_LeadingMarker`.
- AC8 A write while the DB is held by an exclusive SQLite write lock for > 5 s → exit 5 `busy` → `TestCLI_Busy`.
- AC9 A DB with a newer schema → writes exit 6 `read_only`; reads work → `TestCLI_ReadOnly`.

### P1-27 · CLI trash, page, history, revert
Type: impl · Packages: `internal/cli` · Depends on: P1-25, P1-06, P1-07, P0-13, P1-19 · Spec: §10.2, §8.2, §8.3

Commands: `trash list [--all]`, `trash restore <id>` (top node ID of the deletion; message names where it landed, D-017), `trash empty --yes` (purge is final; without `--yes` in a TTY ask `Empty the Trash? This can't be undone [y/N]`, without TTY → usage error), `page get N`, `page set N [--file F | stdin] [--if-version V]` (raw Markdown, no title line — the CLI sets the body only; decision, log A-entry if changed), `page delete N`, `history list [--source S] [--node N] [--limit K] [--ops]`, `history purge --before DATE --yes`, `revert <change-set-id>`.

Acceptance criteria:
- AC1 Goldens for every command's JSON → `TestCLI_Trash_*`, `TestCLI_Page_*`, `TestCLI_History_*`, `TestCLI_Revert_JSON`.
- AC2 `revert` reports skipped fields → `TestCLI_Revert_Skipped`.
- AC3 `trash list` hides merge leftovers; `--all` shows them → `TestCLI_TrashList_All`.
- AC4 `page set --if-version` conflict → exit 4 → `TestCLI_PageSet_Conflict`.

### P1-28 · CLI export, import, snapshot, check, paths
Type: impl · Packages: `internal/cli` · Depends on: P1-25, P1-20, P1-21, P1-09, P1-08 · Spec: §10.2, §7.5, §8.5

Commands: `export --format markdown|json|opml [--out DIR] [--ids] [--history] [--split]` (no `--out`: single-file formats to stdout; `--split` requires `--out`), `import --format json|opml|markdown FILE [--under N] [--replace]` (JSON without `--replace` into non-empty → `not_empty` naming `--replace`, §7.5; `--under` with JSON imports as new nodes with new IDs: build via planners), `snapshot list`, `snapshot restore NAME` (`tui_running` if locked), `check` (`QuickCheck` + `CheckInvariants`; exit 1 `integrity` with problems), `paths`. The `--scheduled` flag is not added until P2-13.

Acceptance criteria:
- AC1 JSON export → import into an empty DB → export is identical (via CLI) → `TestCLI_JSONRoundTrip`.
- AC2 Import JSON into non-empty without `--replace` → `not_empty` message mentions `--replace` → `TestCLI_Import_NotEmpty`.
- AC3 `--replace` takes a snapshot first (listed by `snapshot list`) → `TestCLI_Import_Replace`.
- AC4 `snapshot restore` refuses while a TUI lock is held → `TestCLI_SnapshotRestore_Locked`.
- AC5 `check` on a corrupted DB exits 1 with problems → `TestCLI_Check`.
- AC6 `paths` JSON golden → `TestCLI_Paths`.
- AC7 Every CLI write takes a due snapshot (fake clock via env `HYPHAL_TEST_NOW`, test-only build tag) → `TestCLI_SnapshotOnWrite`.

### P1-34 · `hyphal capture` prompt
Type: impl · Packages: `internal/cli` · Depends on: P1-26 · Spec: §4.7, §6.3 · D-023

Deliverables: `hyphal capture`: a minimal inline Bubble Tea program (not alt-screen) showing a `Capture › ` prompt in ANSI blue (the accent token's color; use Lip Gloss directly, since `cli` must not import `tui` packages), single-line input with the text-editing basics (Left/Right, Backspace, Ctrl+W/U/A/E, paste), Enter → `PlanAdd` to the Inbox and exit 0 with message `Added to Inbox` on stderr; Esc / Ctrl+C / Ctrl+Q → exit 0 without adding; empty Enter → exit without adding.

Acceptance criteria:
- AC1 Typing text + Enter adds one node to the Inbox with source `cli` → `TestCapture_Adds`.
- AC2 Esc adds nothing → `TestCapture_Cancel`.

---

## Wave E — TUI base

All TUI cards: test with `tuitest.Harness`; screen goldens with teatest where the card says. Use `plan/reports/p0-18-stack.md` for Bubble Tea v2 API facts.

### P1-29 · Theme, tokens, background detection, glyphs
Type: impl · Packages: `internal/tui/theme`, `internal/tui` · Depends on: P0-18, P1-01 · Spec: §4.3, §4.4, §4.6 (tokens, contrast, glyph sets)

Deliverables:
- `theme.Tokens` → Lip Gloss styles for every role in §4.6's table, ANSI 16 colors only (indices 0–15) plus default fg/bg; never set a background color except `highlight` (yellow bg) and `selection` (reverse).
- `muted` = bright black (8) on light backgrounds, `Faint` on dark (§4.6 Contrast). Text colors map per §5.8 color mapping.
- Background: query OSC 11 at startup (Bubble Tea v2 background color request), re-query on focus-in and on the color-scheme notification if the pinned Bubble Tea exposes it; no reply within 200 ms → dark; `background` config overrides.
- Fallbacks: if the terminal lacks strikethrough/dim/italic (`TERM` heuristics: `linux`, `vt100`, Terminal.app `TERM_PROGRAM=Apple_Terminal` treated as ANSI-only), use the ASCII markers (§4.3: completed `x` marker, heading `#`).
- Glyph sets: Unicode/ASCII per §4.3 table; `auto` = ASCII when `LC_ALL`, then `LC_CTYPE`, then `LANG` (first non-empty) doesn't contain `UTF-8`/`utf8` (case-insensitive).
- Theme changes invalidate the row cache.

Acceptance criteria:
- AC1 Every token resolves to an ANSI style; no truecolor or 256-color codes appear in any rendered golden → `TestTheme_ANSIOnly` (scans `.ansi` goldens for `38;2`/`38;5`/`48;2`/`48;5`).
- AC2 `muted` is bright black on light, faint on dark → `TestTheme_MutedContrast`.
- AC3 A simulated OSC 11 light reply switches to light; no reply → dark; config overrides → `TestBackground_Detection` (teatest).
- AC4 Locale table selects glyphs → `TestGlyphs_Locale`.
- AC5 The §4.3 table renders correctly in both glyph sets → `TestRender_GlyphTable` (goldens: unicode, ascii).

### P1-30 · App shell: layout, header, footer, banners, messages, state machine
Type: impl · Packages: `internal/tui` · Depends on: P1-29, P1-24, P1-10 · Spec: §4.5, §4.6 (column, too small, restraint), §4.7 (Esc chain), §5.1, §6.4 (global keys)

Deliverables:
- Screen layout (§4.5): header row (breadcrumb left, muted; DB name + save indicator right), optional banner row, blank row, body, footer row (last terminal row). All inside the centered reading column (`column_width`, fills width below the cap).
- Save indicator: nothing / `●` while typing unsealed or commits queued / red `Save failed` label on commit failure.
- Footer priority (§4.5): message → filter query (`/ tooltip · 12 matches`) → hints (if `footer=true`). Message persists until the next keypress (any key, including one that does nothing) then the previous footer returns; newer message replaces older.
- Banners (§4.5): a stack with priority (failed write > read-only > invalid config > restore info); informational ones clear on the next outline change.
- State machine: `TextEditing`, `BlockSelection`, `FilterInput`, `PageView`, `Overlay` (stacked over a base state), `Trash`. Esc chain per §4.7. Key routing: overlay → state context table → global table (`keymap.Lookup`).
- Global commands: Ctrl+Q quit (seal + drain, ARCHITECTURE §5), Ctrl+S message `"Saved automatically"`, F1 help (stub until P1-38), Ctrl+P palette (stub until P1-38), Ctrl+Z/Ctrl+Y (wired in P1-37).
- Footer hint strings (binding): text editing `Enter new · Tab indent · Esc select · Ctrl+P commands · F1 help`; block selection `Enter edit · x complete · d delete · z zoom · ? help · Ctrl+P commands`; page view `e edit · Tab next link · Esc back`; trash `Enter preview · r restore · d purge · Esc back`.

Acceptance criteria:
- AC1 Header/banner/blank/body/footer layout at 100×30 and 200×50 (centered column) → `TestScreen_Layout` (goldens).
- AC2 Message shows until the next key, then hints return → `TestFooter_MessageUntilNextKey`.
- AC3 Footer priority order → `TestFooter_Priority`.
- AC4 `footer=false` hides hints but messages still show → `TestFooter_Disabled`.
- AC5 Save indicator states → `TestSaveIndicator`.
- AC6 Ctrl+S shows "Saved automatically"; Ctrl+Q seals typing and the text is in the DB after exit → `TestGlobal_SaveAndQuit`.
- AC7 Banner priority and informational clearing → `TestBanners`.

### P1-31 · Outline rendering
Type: impl · Packages: `internal/tui` · Depends on: P1-30 · Spec: §4.1–§4.3, §4.6, §5.7 (page marker), §9.3

Deliverables:
- Zoom heading row (bold text; not at Home, D-036) followed by a blank row; rows from `VisibleRows`; glyphs; task checkboxes after the glyph (`☐`/`☑`); completed: dim + strikethrough text, glyph unchanged (§4.3); heading-style bullets bold with one blank row above (not when first under the heading; never two blank rows in a row, §4.6 density rule); comfortable density = blank line between top-level siblings of the zoom; compact = none.
- Indentation 2 columns per level up to 8 levels below the zoom, then 1 column per level; when deeper than 8, footer hint (as a message on entering such a row, once per zoom): `"Deep outline · Esc z zooms in"`.
- Indent guides `│` (guide token) when `indent_guides=true`.
- Page marker ` ≡` (muted) after the text (§4.6).
- Focus: focused row shows revealed markup, accent-colored glyph, terminal cursor; concealment reflow keeps the bullet's first row fixed on screen (§4.6).
- Row cache keyed by (node version, page flag, focused, filter-match ranges hash, width, theme, glyph set, density context).
- Scrolling: viewport over visual lines, scroll margin 3 (§4.6); cursor row stays fixed on screen when collapsing/expanding (§4.6); Page Up/Down scroll a screen.
- Filter highlights (match token) and page-match lines beneath the bullet (dim, §6.2) — rendering only; filter state arrives in P1-40.
- Virtual empty bullet in empty zooms (D-036).

Acceptance criteria:
- AC1 The §4.2 mockup content renders as specified (comfortable, Unicode) → `TestScreen_Mockup42` (golden).
- AC2 Density rules: comfortable vs compact; heading spacing never doubles blank rows → `TestRender_Density` (goldens).
- AC3 Deep indentation beyond 8 levels uses 1 column per level → `TestRender_DeepIndent` (golden).
- AC4 Completed parent keeps its collapse glyph; text dim + struck → `TestRender_Completed` (ANSI golden).
- AC5 Scroll margin: moving down never leaves the cursor within 3 rows of the bottom unless at the end → `TestScroll_Margin`.
- AC6 Collapse keeps the cursor's screen row fixed → `TestScroll_CollapseStable`.
- AC7 Unchanged rows are served from the cache (counter in test build) → `TestRowCache_Hits`.
- AC8 Empty zoom shows the virtual bullet; typing creates a real node → `TestVirtualBullet`.

### P1-32 · Text editing
Type: impl · Packages: `internal/tui` · Depends on: P1-31, P0-17, P1-15 · Spec: §5.1, §5.8 (wrap keys), §6.1, §6.4 (text-editing table), §7.3

Deliverables:
- Cursor model `{Node, Offset, Anchor (selection), PreferredCol}`; movement keys from §6.4: arrows (visual lines, across bullets), word moves (Ctrl/Alt+Left/Right, Alt+b/f), Home/End and Ctrl+A/E (visual line, then bullet on second press), Ctrl+Home/End (first/last bullet in the zoom), Page Up/Down.
- Selection: Shift+arrows, Shift+Home/End, Ctrl+Shift+Left/Right; selection is within one bullet (Shift+Up/Down at the bullet's first/last visual line clamps to the start/end — decision; cross-bullet text selection isn't in the spec).
- Deletion: Backspace/Delete within text (edge cases go to P1-33), Ctrl+W/Ctrl+Backspace/Alt+Backspace (word back), Ctrl+U (to start of visual line).
- Typing printable runes: `Outline.TypeInto`; replaces selection; leading markers convert live when the bullet starts with a marker followed by space (§5.4) — the conversion is part of the typing operation (the sealed `edit_text` plus `set_task`/`set_heading` ops in one change set).
- Wrap keys `*`, `~`, `=`, `` ` `` with a selection → `markup.WrapSelection`, selection kept (§5.8). Without a selection they type.
- Ctrl+C / Ctrl+X with a text selection: copy / cut text (D-030); Ctrl+C without selection → message `"Ctrl+Q quits"`.
- Seal typing on leaving the node, on structural commands, and via the 250 ms tick (2 s idle / 5 s open).
- Alt+other keys do nothing (§6.4 terminal rules).

Acceptance criteria:
- AC1 Each movement binding in the §6.4 text-editing table moves as specified, including Home twice → `TestTextNav_*` (one test per binding, tagged `// covers: <command>@text`).
- AC2 Up/Down move through wrapped lines and across bullets keeping the column → `TestTextNav_UpDownWrapped`.
- AC3 Typing coalesces into one change set per node sealed after 2 s idle (fake clock) → `TestTyping_OneChangeSet`.
- AC4 Wrap keys: select "word", press `*` twice → `**word**`, selection still on "word" → `TestWrapKeys`.
- AC5 Typing `## ` at the start makes a heading and removes the marker → `TestLeadingMarker_Typed`.
- AC6 Ctrl+C without selection shows "Ctrl+Q quits" → `TestCtrlC_Hint`.
- AC7 Alt+x does nothing; Alt+b moves a word back → `TestAltKeys`.

### P1-33 · Structural keys in text editing
Type: impl · Packages: `internal/tui` · Depends on: P1-32, P1-12, P1-13, P1-14 · Spec: §5.2–§5.4, §6.4

Deliverables: wire Enter, Backspace at start, Delete at end, Tab/Shift+Tab, Ctrl+Shift+Up/Down, Ctrl+Enter (toggle complete, kitty), Ctrl+F (filter, wired in P1-40) to the planners; seal typing before each; apply `Plan.Cursor`, `Plan.Expand`, `Plan.Hint`/`Message`.

Acceptance criteria:
- AC1 Every §5.2 row via keys → `TestKeys_Enter_*` (same fixtures as P1-12, driven through the harness; tag `// covers: new_bullet@text`).
- AC2 Every §5.3 and §5.3a row via keys → `TestKeys_Backspace_*`, `TestKeys_Delete_*`.
- AC3 Tab/Shift+Tab and Ctrl+Shift+Up/Down via keys, including hints → `TestKeys_IndentMove`.
- AC4 Ctrl+Enter toggles complete when it arrives (kitty on); without kitty it arrives as Enter and acts as Enter → `TestKeys_CtrlEnter`.

---

## Wave F — TUI features

### P1-35 · Block selection
Type: impl · Packages: `internal/tui` · Depends on: P1-33 · Spec: §5.1, §6.4 (block table), §4.3 (selected row) · D-036

Deliverables: Esc from text editing selects the current bullet (terminal cursor hidden, reverse-video rows); Up/Down move; Shift+Up/Down extend across siblings only (stop at the first/last sibling); every single key in the §6.4 block table dispatched to its command (commands from later cards show `"Not available yet"` until wired — remove when wired); Enter → text editing at end of the (first) selected bullet; Backspace → hint `"Press Delete or d to delete"`; other printable keys → hint `"Press Enter to edit, or ? for help"`; Esc → clear filter if active else hint `"Press Enter to edit"`. Chords from text editing that act on bullets apply to the whole selection (Tab, Shift+Tab, Ctrl+Shift+Up/Down, Ctrl+C/X, Ctrl+Enter). Slow Esc then key runs the key in block selection (§6.4 terminal rules).

Acceptance criteria:
- AC1 Esc selects; Enter returns with the cursor at the end → `TestBlock_EnterExit`.
- AC2 Shift+Down extends across siblings only; it stops at the last sibling → `TestBlock_ExtendSiblingsOnly`.
- AC3 Printable keys without bindings never change the outline and show the hint → `TestBlock_TypingDoesNothing` (property: random printable unbound keys).
- AC4 `d`/Delete delete the selection with the undo message; Backspace shows the hint → `TestBlock_Delete`.
- AC5 `x`, `t`, `#`, `D`, `[`, `]`, Tab, Shift+Tab act on the whole selection → `TestBlock_Commands` (tagged per command).

### P1-36 · Collapse, expand, zoom, breadcrumbs
Type: impl · Packages: `internal/tui` · Depends on: P1-35, P1-10 · Spec: §4.1, §6.1, §6.6, §4.6 (zoom transition) · D-026

Deliverables: `-`/`=` collapse/expand; `1`–`9`, `0` levels (D-026); palette "Collapse all"; `z` zoom in (selected node becomes zoom; cursor to its first child or the heading), `Z` zoom out one level (cursor returns to where it was in that zoom, §6.1, else on the node we came from), `~` zoom to root; `h` show/hide completed (persisted, message `"Completed bullets hidden"` / `"…shown"`); breadcrumbs in the header with truncation; cursor memory per zoom (persisted in `ui_state`); zoom persisted across sessions; optional zoom transition (`zoom_transition=true`: a ≤ 120 ms 3-frame expand animation of the heading row; skipped in tests unless enabled).

Acceptance criteria:
- AC1 Collapse/expand and levels change visibility and persist across restart (new harness on the same DB) → `TestCollapse_Persist`.
- AC2 Zoom in/out/root and per-zoom cursor memory → `TestZoom_CursorMemory`.
- AC3 Esc never zooms out (Esc chain test) → `TestEsc_NeverZoomsOut`.
- AC4 Breadcrumb at depth 6 in a 60-column terminal truncates in the middle → `TestBreadcrumb_Render` (golden).
- AC5 `h` hides completed and persists → `TestShowCompleted_Toggle`.

### P1-37 · Undo and redo in the TUI
Type: impl · Packages: `internal/tui` · Depends on: P1-24, P1-35 · Spec: §5.6

Acceptance criteria:
- AC1 Ctrl+Z after typing undoes that typing; Ctrl+Y redoes → `TestTUI_UndoTyping`.
- AC2 Undo after a subtree delete restores it and the cursor/selection from before → `TestTUI_UndoDelete`.
- AC3 Undo of a change outside the zoom zooms there with a message → `TestTUI_UndoZooms`.
- AC4 Undo with nothing to undo → message `"Nothing to undo"` → `TestTUI_UndoEmpty`.

### P1-38 · Overlay framework, command palette, help overlay
Type: impl · Packages: `internal/tui` · Depends on: P1-30, P0-19 · Spec: §4.6 (overlay look), §4.8 (help), §6.4 (overlay keys), §6.6 (palette)

Deliverables:
- Overlay component: rounded border in accent, title in the top border (§4.6 sketch), centered, width min(60, column), filter input line `› query`, list with highlight (selection token), keys: typing filters (fuzzy), Up/Down and Ctrl+N/Ctrl+P move (Ctrl+P moves up inside overlays), Enter chooses, Esc closes.
- Command palette (Ctrl+P): every command with a display name available in the current context, showing its key binding right-aligned (from the active keymap, e.g. `Esc c` for block-only commands invoked from text editing), recently used first (`palette_recent`). Running a command from the palette executes it in the context it was opened from (block-only commands from text editing operate on the focused bullet).
- Palette-only commands in Phase 1: Collapse all, Delete page, Bold (bullets), Italic (bullets), Color…, Edit settings, Edit zoom in $EDITOR (D-036), Empty Trash, Open Trash. (Open Trash also has no key; it's palette-only by design.)
- Help overlay (F1, `?` in block selection): all commands in the current context grouped by context with their keys, generated from the keymap, filterable by typing (§4.8).

Acceptance criteria:
- AC1 Palette lists commands with keys; typing `col` matches Color…, Collapse all, Expand to level… (sketch in §4.6) → `TestPalette_Filter` (golden).
- AC2 Recently used commands come first and persist → `TestPalette_Recent`.
- AC3 Ctrl+P inside the overlay moves up; Esc closes and returns to the previous state → `TestOverlay_Keys`.
- AC4 Help lists every binding of the current context and filters → `TestHelp_GeneratedFromKeymap` (compares against `keymap.Defaults()`).

### P1-39 · Color picker, Move to…, link picker and `o`
Type: impl · Packages: `internal/tui` · Depends on: P1-38, P1-15 · Spec: §4.8 (color picker), §5.4 (Move to), §5.8, §6.4 (`o`, `c`, `m`) · D-025, D-028

Deliverables: color picker overlay (8 swatches with names in their colors + "No color"; `1`–`8`, `0`, arrows+Enter, Esc); Move to… overlay (D-028); `o` follows the bullet's URL link via `platform.OpenURL`; several links → picker overlay listing them; no link → hint (D-025).

Acceptance criteria:
- AC1 `c` then `3` colors the selected bullets yellow; `0` removes color → `TestColorPicker`.
- AC2 `m`, type a parent name, Enter → moved to the end of that parent with an undo message → `TestMoveTo`.
- AC3 `o` opens the URL (fake opener); with two links a picker appears → `TestFollowLink`.

### P1-40 · Filter UI
Type: impl · Packages: `internal/tui` · Depends on: P1-16, P1-35 · Spec: §6.2, §6.4 (filter input), §5.4 (editing under a filter)

Deliverables: Ctrl+F / `/` opens filter input in the footer row (`/ query · N matches`); outline filters as you type (debounce none; target < 50 ms); Enter or Down moves focus to the outline on the selected match (first match by default) with the filter active; Esc in input clears the filter and keeps the cursor on the selected match; with the filter active, Ctrl+F or `/` returns to the input; Esc in block selection clears it; sticky nodes per §5.4; page matches show the line beneath (dim) and highlight `≡`.

Acceptance criteria:
- AC1 Typing filters; matches highlighted; ancestors shown → `TestFilter_Typing` (golden).
- AC2 Enter moves to the outline on the first match; Esc clears and keeps the cursor there → `TestFilter_EnterEsc`.
- AC3 Editing a bullet so it no longer matches keeps it visible until the query changes → `TestFilter_Sticky`.
- AC4 Page match shows the matching line dimmed → `TestFilter_PageMatch` (golden).

### P1-41 · Clipboard and paste in the TUI
Type: impl · Packages: `internal/tui` · Depends on: P1-17, P1-04, P1-35 · Spec: §5.5

Deliverables: enable bracketed paste; route `PasteMsg` to `PlanPaste`/`PlanPasteCut`; block-selection Ctrl+C copies Markdown (message `"Copied N bullets"`), Ctrl+X cuts (copy then delete; remember `CutRecord`); clipboard failures → message `"Couldn't copy: <error>"`; OSC 52 output routed through the program's output.

Acceptance criteria:
- AC1 Multi-line paste in each position creates the expected subtree → `TestTUI_Paste_*`.
- AC2 Copy writes Markdown to the (fake) clipboard → `TestTUI_Copy`.
- AC3 Cut + paste elsewhere moves the nodes (same IDs) → `TestTUI_CutPasteMoves`.
- AC4 Clipboard tool failure shows the message → `TestTUI_CopyFailure`.

### P1-42 · Page view renderer
Type: impl · Packages: `internal/tui/pageview` · Depends on: P1-29 · Spec: §5.7 (view mode), §4.6

Deliverables: goldmark (GFM tables, strikethrough, task lists) → terminal lines within the column width: headings (bold in `text`; H1 and H2 followed by a muted rule line `─`; the accent color is reserved for edit mode, §4.6), paragraphs wrapped with `layout`, lists with bullets/numbers, block quotes with a muted `│`, code blocks highlighted by chroma mapped to ANSI 16 colors (custom chroma formatter or style mapping token types → 8 colors), tables with box-drawing (ASCII fallback), inline markup incl. `==highlight==` and `<color>` tags (reuse `markup` styles for inline spans inside paragraphs), links underlined with a focusable index. Source positions: each rendered line records the source byte range of its block (for P2 mode sync).

Acceptance criteria:
- AC1 A fixture page with every element renders as the golden (dark, light, ASCII) → `TestPageView_Elements` (goldens).
- AC2 Code blocks use only ANSI 16 colors → `TestPageView_CodeColors`.
- AC3 Link list in document order with source ranges → `TestPageView_Links`.
- AC4 Wrapping respects width with CJK/emoji → `TestPageView_Wrap`.

### P1-43 · Page screen, `p`, `e` via `$EDITOR`, Delete page
Type: impl · Packages: `internal/tui` · Depends on: P1-42, P1-19 · Spec: §5.7, §6.4 (page view table) · D-034

Deliverables: `p` in block selection opens the page in view mode; on a bullet without a page opens `$EDITOR` directly with `# Title\n` (Phase 1 edit mode = `$EDITOR`); breadcrumb `… › ≡ view`; title rendered as the page heading with inline markup; scrolling keys; Tab/Shift+Tab focus URL links, Enter opens the focused link or (no focus) edits; `e` edits in `$EDITOR` (suspend via `tea.ExecProcess`, temp file in the private dir, CONTRACTS §9), applies `PageEditorApply` as one undoable change set, returns to view mode (or to the outline if the page was deleted); Esc returns to the outline in block selection on the bullet. Palette "Delete page" (undo message). `$EDITOR` unset → `$VISUAL` → `vi`; message on editor failure.

Acceptance criteria:
- AC1 `p` on a bullet with a page shows view mode with breadcrumb `≡ view` → `TestPage_Open` (golden).
- AC2 `p` on a bullet without a page → fake editor writes body → page created; saving empty creates nothing → `TestPage_CreateViaEditor`.
- AC3 Editing the title line renames the bullet; Ctrl+Z after returning undoes it → `TestPage_RenameUndo`.
- AC4 Esc returns to block selection on the page's bullet → `TestPage_EscReturns`.
- AC5 Tab focuses links, Enter opens the URL → `TestPage_Links`.

### P1-44 · `$EDITOR` subtree round trip in the TUI
Type: impl · Packages: `internal/tui` · Depends on: P1-18, P1-35 · Spec: §6.3

Deliverables: `e` in block selection (selection), palette "Edit zoom in $EDITOR" (zoom scope), and `e` on a focused zoom heading in block selection (zoom scope); writes the temp file (0600 in the private 0700 dir), suspends, runs the editor, parses, applies one change set (source `editor`, on the undo stack), shows the summary message; parse failure → nothing applied, file kept, banner `"Couldn't read the edited file (line N) · Enter reopens it · Esc discards"` with Enter reopening; temp file removed after success.

Acceptance criteria:
- AC1 Fake editor reorders and renames lines → applied; summary message; Ctrl+Z undoes the whole round trip → `TestEditorRoundTrip_Apply`.
- AC2 Malformed file → banner, nothing applied; Enter reopens the same file → `TestEditorRoundTrip_ParseError`.
- AC3 Temp file permissions 0600 in a 0700 dir → `TestEditorRoundTrip_Permissions`.
- AC4 A CLI change to a bullet while the editor is open → the edit lands after it; message per D-033 → `TestEditorRoundTrip_Concurrent`.

### P1-45 · Trash view
Type: impl · Packages: `internal/tui` · Depends on: P1-06, P1-38 · Spec: §4.8, §8.3, §6.4 (Trash table)

Deliverables: full-screen view opened from the palette (Open Trash): rows `text · former breadcrumb · N bullets · 3d ago`, newest first, hidden entries excluded; Enter → read-only preview of the subtree (rendered like the outline, Esc back to the list); `r` restores (message names where it landed, D-017; the view stays in the Trash so several entries can be restored in a row); `d` purges after a one-line confirmation in the footer `"Purge \"<text>\" for good? y/N"`; palette Empty Trash with confirmation `"Empty the Trash for good? y/N"`; Esc returns to the outline.

Acceptance criteria:
- AC1 List rows for two deletions with breadcrumbs and counts → `TestTrash_List` (golden).
- AC2 `r` restores to the original position and shows the message → `TestTrash_Restore`.
- AC3 `d` + `y` purges; `d` + any other key cancels → `TestTrash_PurgeConfirm`.
- AC4 Preview is read-only (typing keys show a hint) → `TestTrash_PreviewReadOnly`.

---

## Wave G — TUI robustness

### P1-46 · Read-only mode, locking, live reload
Type: impl · Packages: `internal/tui` · Depends on: P1-30, P1-05, P1-23 · Spec: §4.8 (read-only), §7.3, §8.6

Deliverables: take the TUI lock at startup (`store.AcquireTUILock`); lock held → open read-only with banner `"Another Hyphal window is editing this outline · read-only"`; newer schema → banner `"This outline was created by a newer version · read-only"`; integrity failure → P1-47. In read-only: every editing key/command shows hint `"Read-only: <reason>"`; navigation, filter, pages (view), palette, help work. `data_version` poll every 500 ms (tea.Cmd) → Reload job → swap tree, keep zoom/cursor/selection by ID (fallback to nearest surviving ancestor), keep unsealed typing. Stale/rejected messages per D-033.

Acceptance criteria:
- AC1 Second TUI on the same DB is read-only with the banner; editing keys show the hint → `TestReadOnly_SecondTUI`.
- AC2 A CLI `add` appears in a running TUI within 1 s (fake ticker advanced) → `TestReload_CLIChangeVisible`.
- AC3 Reload keeps zoom and cursor; when the cursor's node was deleted elsewhere, the cursor goes to its nearest surviving ancestor → `TestReload_KeepsPosition`.
- AC4 Unsealed typing on a node changed by the CLI commits after the CLI change; message shown; both change sets in history → `TestReload_ConcurrentTyping`.

### P1-47 · Durability wiring
Type: impl · Packages: `internal/tui` · Depends on: P1-46, P1-09, P1-08, P1-23 · Spec: §7.3, §8.1, §8.5, §8.3 · D-016

Deliverables: async committer in the TUI; commit failure → red save label + banner `"Couldn't save: <error> · retrying"`, retry every 3 s, typing continues; quitting with unsaved edits → recovery file + message on stderr after exit `"Unsaved edits written to <path>"`; SIGHUP/SIGTERM handling (ARCHITECTURE §5); `quick_check` after the first frame → failure switches to read-only immediately, writes a recovery file for unsealed typing, banner `"This outline failed its integrity check · read-only · Enter restores the latest good snapshot"` with Enter → close connections, `RestoreSnapshot(latest good)`, reopen; snapshot due check at launch, quit and every 60 s; trash purge due (D-016); sync-folder warning banner if `platform.InSyncFolder(db)`: `"outline.db is inside <name>; syncing a live database can corrupt it"`. It persists while the cause exists (§4.5).

Acceptance criteria:
- AC1 Injected commit failure: red label, banner, typing continues, success after recovery clears both → `TestDurability_FailedWrite`.
- AC2 Quit while the store fails → recovery file with the unsaved text → `TestDurability_RecoveryOnQuit`.
- AC3 SIGTERM during typing → typing sealed and committed before exit → `TestDurability_SIGTERM` (subprocess).
- AC4 Corrupt DB → read-only banner; Enter restores the latest good snapshot and reopens editable → `TestDurability_IntegrityRestore`.
- AC5 Snapshot taken when due at launch and on the 60 s tick (fake clock) → `TestDurability_SnapshotTimer`.
- AC6 First frame is rendered before `quick_check` completes (check blocked by a test hook) → `TestStartup_QuickCheckAfterFirstFrame`.

### P1-48 · Config in the TUI
Type: impl · Packages: `internal/tui` · Depends on: P1-01, P1-38 · Spec: §4.7, §13, §6.4 (keymap config)

Deliverables: apply config at startup (column width, density, guides, glyphs, background, footer, zoom transition, trash days, snapshot keep, keys); invalid config banner `"config.toml line N: <message> · using defaults for those keys"`; palette "Edit settings": write `CommentedTemplate()` if the file is missing, open `$EDITOR`, `Reload` on return (invalid → keep previous + banner); keymap overrides validated (`keymap.WithOverrides`): conflicts keep the default for that command + banner (§6.4).

Acceptance criteria:
- AC1 `column_width = 60` narrows the column; `density = "compact"` removes blank lines → `TestConfig_Applied` (goldens).
- AC2 Invalid value → banner naming the line; valid keys still applied → `TestConfig_InvalidBanner`.
- AC3 Edit settings with an invalid save keeps the previous config and shows the banner → `TestConfig_EditSettingsInvalid`.
- AC4 Key override `toggle_complete = "y"` in `[keys.block]` works; a duplicate binding keeps the default and shows the banner → `TestConfig_KeyOverrides`.

### P1-49 · First run and Inbox refresh in the TUI
Type: impl · Packages: `internal/tui` · Depends on: P1-22, P1-46 · Spec: §4.5 (first run), §6.3

Acceptance criteria:
- AC1 First TUI open of a CLI-created empty DB shows the sample outline; the second open does not re-add it → `TestFirstRun_TUI`.
- AC2 After `hyphal import` of OPML into a fresh DB, the first TUI open adds no sample → `TestFirstRun_AfterImport`.
- AC3 `hyphal add` while the TUI shows the Inbox → the new bullet appears → `TestInbox_LiveRefresh`.

---

## Wave H — ship and gate

### P1-50 · Packaging and docs
Type: impl · Packages: `packaging/`, `docs/`, `.goreleaser.yaml`, `.github/workflows/release.yml` · Depends on: P1-28, P1-34 · Spec: §4.7, §9.1 (distribution), §5.5 (tmux), §6.4 (tmux, macOS) · Human review: yes

Deliverables: `packaging/hyphal.desktop` (`Exec=xdg-terminal-exec --app-id=hyphal hyphal` or equivalent; `StartupWMClass=hyphal`; categories Office;Utility), Hyprland window rule examples for `hyphal` and `hyphal-capture` in `docs/omarchy.md` (suggest a Super binding checked against Omarchy defaults — list the chosen key and how it was checked), `.goreleaser.yaml` (linux amd64/arm64, darwin amd64/arm64, `CGO_ENABLED=0`, version ldflags, Homebrew tap formula, AUR `PKGBUILD` via goreleaser's `aurs`), `docs/` pages: install, keymap (generated from `keymap.Defaults()` by `go generate`), tmux snippet (§6.4), clipboard (OSC 52, `set-clipboard on`), editor temp files/swap files note (§6.3), backups (off-machine backup is the user's job, §8), never sync the live DB (§8.1), scheduled export via systemd timer/launchd/cron (stub for P2-13), CLI reference (generated by Cobra), `README.md` update.

Acceptance criteria:
- AC1 `goreleaser check` and `goreleaser build --snapshot --clean` succeed in CI → workflow job.
- AC2 `docs/keymap.md` is up to date with `keymap.Defaults()` (CI fails on drift) → `TestDocs_KeymapUpToDate`.
- AC3 `desktop-file-validate packaging/hyphal.desktop` passes (CI on Linux) → workflow step.

### P1-51 · TUI performance benchmarks
Type: impl · Packages: `internal/tui`, `internal/store` · Depends on: P1-47 · Spec: §9.4 · TESTING §9

Acceptance criteria:
- AC1 Benchmarks in TESTING §9 exist and run in nightly CI against the baseline → workflow + `testdata/bench/baseline.txt` updated.
- AC2 `plan/reports/p1-51-perf.md` lists the numbers on the CI runner and instructions for the human's fixed-hardware run.

### P1-52 · Reference mockups
Type: impl · Packages: `internal/tui` (tests only), `plan/reports/` · Depends on: P1-48, P1-40 · Spec: §4.6 (reference mockups), §12.4.4 · Human review: yes

Deliverables: five teatest golden screens (ANSI and plain): 100-column laptop (100×30), full-screen ultrawide (240×60), deep zoom (breadcrumb truncation, deep indent), active filter, light theme. A script `make mockups` renders the `.ansi` goldens to the terminal one by one for the human, and `plan/reports/p1-52-mockups.md` explains how to run the calm review (side by side with Workflowy and with lazygit/btop under three Omarchy themes, one light).

Acceptance criteria:
- AC1 Five goldens exist and are stable → `TestMockups_*`.

### P1-53 · Real Workflowy import check
Type: human · Depends on: P1-21, P1-28 · Spec: §7.5, §12.4.6

Agent part: `hyphal import --format opml FILE --json` prints an `ImportSummary` (nodes, pages, completed, colors mapped, unknown HTML tags seen, mirrors detected). Write `plan/reports/p1-53-workflowy.md` with instructions: export from Workflowy as OPML, place at `testdata/private/workflowy.opml` (gitignored), run the import into a temp DB, review the summary and the outline. Then stop. A follow-up agent turns the human's findings into fixes and a redacted fixture in `testdata/opml/`.

### P1-54 · Parity checklist suite
Type: impl · Packages: `internal/tui` (tests) · Depends on: all Phase 1 TUI cards · Spec: §3 parity checklist (Phase 1 rows)

Deliverables: `parity_test.go` with one end-to-end scripted test per Phase 1 *Must* row, named `TestParity_<Row>` (OneTreeZoomBreadcrumbs, EnterTabBackspace, BlockSelectionMultiBulk, CollapseExpandLevels, CompleteShowHide, Pages, InlineFormattingColor, DuplicateMoveTo, ImportWorkflowyOPML, UndoTrash).

Acceptance criteria:
- AC1 All ten parity tests pass → `TestParity_*`.

### P1-55 · Interaction matrix meta-test
Type: impl · Packages: `internal/tui` (tests) · Depends on: P1-54 · Spec: §9.5 (interaction matrix) · TESTING §8

Acceptance criteria:
- AC1 Every binding in `keymap.Defaults()` for Phase 1 commands has at least one test tagged `// covers: <command>@<context>` → `TestMatrix_EveryBindingCovered`.
- AC2 Every row of §5.2, §5.3, §5.3a has a key-driven test (list of row IDs in the meta-test) → `TestMatrix_EditingTablesCovered`.

### P1-56 · Phase 1 gate
Type: gate · Depends on: P1-01…P1-55 · Spec: §11 row 1, §9.6

Deliverables: `plan/reports/phase-1-gate.md`: automated evidence (parity suite, matrix, CLI goldens, crash suite, benchmarks), the release build tag, the dogfooding template `plan/reports/phase-1-dogfood.md` (daily log: what felt wrong, bugs, missing), the calm review checklist, and the comparison-set evaluation checklist (§3). Then stop for the human's week of daily use and sign-off. Bugs found during dogfooding become new cards `P1-57+` appended to this file.
