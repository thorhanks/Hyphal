# SPEC.md Review — Developer & UX Analysis

Review of `SPEC.md` (Project Specification v3, Oct 15, 2026). Reviewed 2026-10-17.

Line references are to `SPEC.md` as of commit `ed8bf43` + working tree.

## Summary

The spec is strong: clear principles, honest non-goals, and testing considered early. Most problems fall into four areas:

1. **Block-selection keys conflict with each other.** Several keys mean two things, and the legacy/tmux fallbacks rely on Esc-then-key sequences that are ambiguous in exactly those terminals.
2. **Statements contradict each other.** Several look like leftovers from an earlier Rust draft.
3. **The data model has gaps** around identity, undo persistence, Trash grouping and sync.
4. **Several screens are named but never specified** (Trash, History, Stars, Sync, help, color picker, tag list).

### Top 5 to fix first

1. Write one unified block-selection keymap that resolves the Alt+Down, Enter, Backspace and Esc-timing conflicts (§2).
2. Decide the undo scope for editor-round-trip and CLI changes (contradiction 1.3).
3. Define split/merge identity rules and Trash deletion grouping (§3).
4. Restore the missing §9.2 package diagram and §11 delivery plan (§7).
5. Define the conflict rule for concurrent TUI typing and CLI edits (§3).

---

## 1. Direct contradictions

| # | Where | Issue |
|---|---|---|
| 1.1 | SPEC.md:83 vs :439 | Nodes have an "in-memory identity" in §4.1 but "random, stable forever" IDs in §7.2. |
| 1.2 | SPEC.md:355 | "Enables it through **Crossterm**". Crossterm is a Rust library; the stack is Go and Bubble Tea. Search for other leftovers from a Rust draft. |
| 1.3 | SPEC.md:275 vs :35, :292, :349 | §5.6 says Ctrl+Z never undoes editor-round-trip changes; they are reverted from the History view instead. Scenario 3, §5.7 and §6.3 all say the round trip is "undoable with Ctrl+Z". The History view is Phase 2 (:586), so in Phase 1 those changes can only be reverted with `app revert`. |
| 1.4 | SPEC.md:231 vs :183 | §5.1: Vim navigation is "a later setting, never a requirement". §4.7: `h j k l gg G` are the default in block selection. |
| 1.5 | SPEC.md:141 vs :196 | §4.5: first run creates "a short sample outline". §4.7: an "empty first-run screen" with an ASCII wordmark (which also depends on the still-undecided name). |
| 1.6 | SPEC.md:261, :391 vs :339 | Move to… is a Phase 1 command (parity table, §6.6) but runs through the jump palette, which is Phase 2. Copy link to node is Phase 1 (:392) while node links are Phase 2. The Phase 1 mockup footer (:105) advertises Ctrl+K, a Phase 2 feature. |
| 1.7 | SPEC.md:509 | "Every exported file starts with a one-line header." This makes JSON invalid and breaks OPML (the `<?xml` declaration must come first). On Markdown re-import the header becomes a bullet. Use a field/element for JSON and OPML, and a recognizable comment for Markdown. |
| 1.8 | SPEC.md:13 vs :415–427, :435, :509, :189 | The name "must not appear in architecture or file formats", yet `<app>` appears in data/config paths, the sync folder, `meta` ("app name"), the export header and the `.desktop` window class. Decide what is acceptable and what happens if the name changes later. |
| 1.9 | SPEC.md:140 vs :689, :184, :177 | §4.5: views are "not panels… no sidebar". §10.1: backlinks is an "on-demand pane"; §4.7 says Ctrl+W moves between "side panes". But §4.7 (:177) lists backlinks and peek as *floating overlays*. Pick one model. |
| 1.10 | SPEC.md:584 | Coalescing: "all edits to one node's text form a single operation", without "consecutive". As written it reads as one op forever. |
| 1.11 | SPEC.md:490 vs :272 vs :533 | Typing commits every ~1 s but an undo step closes after a 2 s pause, so the op row must be updated in place. §7.6 says history is "never rewritten", and a sync log can't amend an op it has already appended. Define when an op is sealed and when it is emitted to the log. |
| 1.12 | SPEC.md:596 vs :533, :583 | Compacting to "one change set per node per day" breaks multi-node change sets, so "what did this agent do?" and revert-by-change-set stop working for old history. `history purge` and **Forget node** hard-delete history, contradicting "never rewrite history", and have no sync tombstone op. |
| 1.13 | SPEC.md:591 vs :539 | Trash restore falls back to the root when the parent is gone; sync falls back to a "Recovered" node. Use one rule. |
| 1.14 | SPEC.md:792 vs :230 | "Multi-select range" is an open question, but §5.1 already says Shift+Up/Down extends "across siblings". |
| 1.15 | SPEC.md:793 vs :351 | Inbox location is still open, yet Phase 1 quick capture (`app add`) depends on it. Must be decided before Phase 1. |
| 1.16 | SPEC.md:318 vs :503 | "Plain-text export" is mentioned but isn't in the export formats table. |
| 1.17 | SPEC.md:732 | "Automation comes last", but sync (Phase 5) comes after automation (Phase 4). By the definition at :54 ("Clone" = every *Must* row), Phase 1 isn't a Workflowy clone either: tags, links, operators and stars are Musts in Phase 2. |
| 1.18 | SPEC.md:374 vs :371 | iTerm2 is "fully supported" but missing from the terminal test matrix. Alacritty and foot are in the matrix but not mentioned for macOS (Alacritty runs there). |
| 1.19 | SPEC.md:315 vs :318 | `<red>` is justified because it "renders in Obsidian after export", but `<red>` is not valid HTML and Obsidian won't color it; the real reason is that export converts it to `<span style>`. Fix the rationale. `==highlight==` is Obsidian-specific and won't render in CommonMark tools. |
| 1.20 | SPEC.md:229 vs §5.7 | Principle 2 says "no mode switch", but block selection and page view/edit are modes in practice. Fine as a design, but acknowledge it rather than claiming there are none. |

---

## 2. Keyboard and interaction conflicts (highest UX risk)

### 2.1 Keys with two meanings

- **Alt+Down** is the legacy fallback for "move subtree down" (:362) *and* "open page" (:280, :399).
- **Enter in block selection** "returns to text editing" (:230) *and* means "zoom in" in the fallback table (Esc then Enter, :364).
- **Backspace vs Delete in block selection.** Esc then Backspace = zoom out (:364); Esc then "delete key" = delete subtree (:259). On a Mac keyboard the key labelled "delete" sends Backspace, so a Mac user trying to delete a subtree zooms out instead.
- **Enter on a tag** sets the filter (:324), but in text editing Enter splits the bullet. Same for Enter on a `[[link]]` in the outline (:688).
- **Enter in page view mode** means both "enter edit mode" (:286) and "follow the focused link" (:284). Define focus: if a link is focused, Enter follows it; otherwise it edits.
- **Esc then `s`** stars the "current zoom target" (:385), but in block selection users will expect it to star the *selected* bullet.
- **Esc then `e`** opens the subtree in `$EDITOR`; `e` in page view enters edit mode. Context-dependent, which is fine, but document it.

### 2.2 Esc-then-key fallbacks are ambiguous where they're needed

The fallbacks exist for legacy terminals and tmux, but in exactly those terminals **Esc followed quickly by K arrives as Alt+K**. tmux's default `escape-time` is 500 ms, which also makes a bare Esc feel sluggish. The spec needs:

- An Esc-timeout policy (how long to wait before treating Esc as standalone).
- tmux guidance (recommended `escape-time`, `extended-keys`).
- A statement that kitty keyboard protocol removes the ambiguity when available.

### 2.3 Leaving block selection by typing is dangerous

"Any printable key that isn't a command returns to text editing" (:230). With `h l j k J K x e p s c g ?` all being commands, a user who presses Esc and starts typing "these" will run several commands. Choose one:

- Printable keys never exit block selection (only Enter / `i` / `a` do), or
- Specify exactly what happens to the triggering key: is it inserted, where does the cursor go?

### 2.4 The Esc chain

- :182 says text editing → block selection → clear filter → zoom out. §6.2 (:338) says Esc clears the filter directly. The filter input field needs its own focus state ("typing in filter" vs "outline with active filter").
- Repeated Esc presses can zoom out by accident. Say what Esc does at the root (nothing? a hint?).
- Esc from page edit mode → view mode → outline is consistent; say so in the chain.

### 2.5 Missing fallbacks and system collisions

- **Alt+1–9** (expand to level) has no Esc fallback; on macOS Option+1 produces `¡`. Alt+B and Alt+I have no fallback either. This breaks the claim at :374 that every Alt binding has one.
- **macOS intercepts Ctrl+Up/Down** (Mission Control / App Exposé) and Ctrl+Left/Right (Spaces) by default. The spec only considers Hyprland intercepts (:372).
- **F1 on Mac laptops** requires Fn.
- **Ctrl+Q** is XON in some terminal setups; raw mode usually disables it, but verify.

### 2.6 Readline / Cocoa expectations

Ctrl+E, K, P, D and W are end-of-line, kill-line, previous-line, delete-char and delete-word in most text fields; Alt+Left/Right is word-jump; Alt+B is word-back. Overriding them may be the right call, but make it a deliberate, documented decision. Ctrl+W as a pane prefix (:184) clashes with delete-word whenever a pane is open.

### 2.7 Undefined keys

- Text editing: word movement, word delete, select-all, copy/cut of a text selection, Page Up/Down, Ctrl+Home/End.
- Navigation: jump to root (:332), the "modifier" for expand-all (:331).
- Block selection: copy, cut, paste of nodes (:267 says copy writes Markdown but not which key), formatting keys applied to the whole bullet (:316).
- **Ctrl+C**: copy, quit, or ignored?
- **Shift+Enter**: core Workflowy muscle memory for notes. Notes become pages here; does Shift+Enter open the page, or is it a no-op?
- Making a bullet a task, making it a heading.

### 2.8 Recommendation

Write **one authoritative block-selection keymap table**. Today those keys are spread across §4.7, §5.1, §5.4, §6.4, §6.6 and §5.7, which is how the collisions above slipped in. Add a CI check that no key is bound twice in the same context.

---

## 3. Data model and storage gaps

### 3.1 Identity on split and merge

- **Enter mid-text (split)**: which half keeps the ID, page, star, completion, inbound `[[links]]`, and history? The spec only says children stay with the first half.
- **Backspace merge**: the merged-away node disappears. What happens to its page, star, completion state and inbound links? If it has a page and the target also has one, which wins?
- **Duplicate** (:295): new IDs for the whole subtree; copies pages. Does it copy stars? Links inside the copy still point to the originals? Where is the copy placed?

### 3.2 Tasks

- `is_task` exists (:442) but there's no glyph, no key, no `[ ]` typing rule, and no visual-language row in §4.3.
- Markdown round trip is lossy: a completed non-task exports as `[x]` and re-imports as a task (`is_task = 1`).
- `is:open` (:341): does it mean "not completed" (any node) or "open task"?

### 3.3 Headings

`is_heading` (:443) has no command, no key, no export mapping, and no op kind.

### 3.4 Op kinds

The list at :575 lacks: page create/delete, `set_heading`, purge (hard delete), restore from Trash (or is it `delete` inverted?), Forget node.

### 3.5 Undo persistence

"Undo survives restarts" and "undo restores cursor and selection" (:273–274) need state that isn't in the schema:

- An undo/redo pointer (which change set is next).
- Cursor/selection position recorded per change set.
- Redo invalidation rules.
- Undoing a change set when a later CLI/agent change touched the same field: apply revert's skip-and-report rule?
- Undo of a change outside the current zoom: does it navigate there and show a toast?

### 3.6 Concurrent TUI typing vs CLI writes

The TUI holds up to ~1 s of uncommitted typing on node X. If the CLI edits X in that window, either the TUI commit overwrites the CLI edit, or the `data_version` reload overwrites the user's typing. A rule is needed (e.g. a per-node version column with optimistic concurrency; agents can pass `--if-version`). The same race applies to a page open in `$EDITOR` while `app page set` runs.

### 3.7 Trash

- If every descendant gets `deleted_at`, nothing records which deleted node was the *root* of a deletion. A child deleted separately earlier would come back when its parent is restored. Add a `deleted_root` / delete-change-set reference.
- Does restore return the node to its original *position* among siblings, or just under its original parent? Fractional positions may need regenerating.
- Hard purge after 30 days conflicts with foreign keys, ops that reference the node, revert of older change sets, undo across restarts, and sync.
- What does the toast say and where does the cursor go after delete?

### 3.8 Root and workspace

- Is the root a row with `parent_id NULL`? Can it be renamed, have a page, be starred?
- "Workspace" (:83) is never defined. It appears to mean "database".
- Breadcrumbs show "Root" in one mockup but not the other.

### 3.9 Persisted UI state

Unspecified where these live: show/hide completed (global? per zoom?), cursor position per zoom, current zoom on restart, command palette recents, filter state. Is `view_state` per device and excluded from sync? Say so.

### 3.10 Timestamps

Format and timezone are unspecified (RFC 3339 UTC with ms?). This matters for `created:2026-10`, `changed:7d` and the Today node. What counts as "changed": text, page, completion, or any descendant?

### 3.11 Schema hygiene

- No indexes declared (`nodes(parent_id, position)`, `deleted_at`, `ops(node_id)`, `change_sets(hlc)`).
- `PRAGMA foreign_keys` isn't mentioned; purge cascade behavior undefined.
- `search_fts` should exclude trashed nodes.
- `node_tags` from page text: say explicitly that a page's tags attach to its bullet for filtering.
- Database file permissions (0600).

### 3.12 Locking

"Enforced with a lock held by the TUI's connection" (:609): SQLite in WAL mode can't express "one TUI writer, many CLI writers" (`locking_mode=EXCLUSIVE` would block the CLI). Specify an `flock`-based lock file. Also:

- Is `app capture` a TUI (it would hit the lock and open read-only) or a writer like the CLI?
- Does a read-only second TUI still refresh via `data_version`?
- What happens when the user types in read-only mode?

### 3.13 Snapshots and restore

- Replacing the database file under a live WAL connection (TUI open) is unsafe. Specify the mechanism: backup API *into* the live DB, or require the TUI to close and reopen.
- **CLI name collision**: `app restore` means Trash restore (:710) and snapshot restore (:605). §10.2 also calls it `restore-snapshot`.
- Disk budget: 48 hourly + 30 daily full copies of a database that includes history.
- `quick_check` "in the background at launch" (:556) can flip a live session to read-only mid-edit. Define that UX. On large databases it may take a while.

### 3.14 Export / import fidelity

- "A JSON export restores everything" (:19), yet history is optional (:505). List exactly what "everything" is: stars, view state, device_id, history, pages, settings?
- Import JSON is "full restore into an empty database". What if it isn't empty?
- Define "identical" for the CI round trip (IDs, positions, timestamps, history?).
- Markdown export of tags, links, colors, headings and non-task completion.
- Page filenames "from bullet text plus ID": sanitization, length limit, collisions.
- **Workflowy OPML mapping** needs a table: `_complete`, `_note`, inline HTML (`<b>`, `<i>`, `<span class="colored c-red">`), mirrors, dates, boards. What is lost?
- "Scheduled export daily" needs a running process; with no daemon, is it caught up on next launch?

### 3.15 Search

- "Case-insensitive substring matching" (:344) isn't how FTS5 works by default; you'd need the `trigram` tokenizer (min 3 chars) or an in-memory scan. Check against the < 50 ms / 100k-node target.
- Diacritics and Unicode case folding.
- A match in page text: how is it shown on the bullet? Snippet?

---

## 4. Sync design risks (cheap to fix now, expensive later)

- **Cycle-skip on moves isn't order-independent.** With per-field LWW and "nothing is rebuilt by replay" (:581), two devices can apply concurrent moves in different orders and skip different ones, so they diverge. A correct move CRDT (e.g. Kleppmann et al.) needs undo/redo of ops in HLC order.
- **Parent and position must be one LWW field** ("location"), not two independent fields.
- **"Edit to a deleted node keeps it deleted"** needs a rule for concurrent restore-from-Trash vs delete.
- **Duplicate `device_id`**: copying the DB to a new machine, restoring a snapshot, or importing JSON clones the device ID. It must be regenerated, and copies detected.
- **Retired devices block log compaction forever** (:546). Add a "forget device" command.
- **Snapshot restore and history purge** interact with logs other devices have already consumed. Unspecified.
- **Forget node / purge** need tombstone ops or the node returns from another device's log.
- **Mirrors**: "can add a reference row type without migration pain" (:787) is optimistic. Mirrors turn the tree into a graph and affect export, Trash, cycles and sync. Either decide now whether IDs and ops would support them, or drop the claim.
- **Coalesced ops** (see 1.11): when is an op emitted to the log?
- HLC drift handling is listed as open; fine, but the convergence tests in Phase 0 need a decision.

---

## 5. Underspecified behavior

### 5.1 Editing tables (§5.2–5.4)

- "Delete at the end of a bullet mirrors these rules forward" (:253) should be its own table.
- Cases not covered:
  - Enter on an empty bullet at the top level of the zoom.
  - Enter or Backspace with an active text selection.
  - Editing the zoom heading itself (can you? Enter on it creates a first child?).
  - Backspace when the "previous visible" bullet is deeper, is the zoom heading, doesn't exist, or is hidden by a filter.
- Structure:
  - Indenting under a *collapsed* previous sibling: does it expand, or does the bullet vanish?
  - Outdent: does it take later siblings as children (Workflowy doesn't)?
  - "Moves to the adjacent parent's sibling list" (:258) needs worked examples.
  - Indenting a mixed-depth multi-selection.
  - Toggling completion on a mixed selection.
  - Where the cursor goes when a newly completed item hides.
- **Editing while filtered**: a new bullet that doesn't match the filter — visible or hidden? Indent when the previous sibling is hidden?

### 5.2 Paste and clipboard (§5.5)

- Multi-line paste with the cursor mid-text: split first?
- Pasting Markdown that contains `^id` suffixes: always new IDs?
- Internal node copy/paste that preserves pages, completion and structure (Workflowy does this).
- Copy/cut keys in both states.
- Clipboard fallback only names `wl-copy`; add `pbcopy` (macOS) and `xclip`/`xsel` (X11).

### 5.3 Pages (§5.7)

- Is the title heading stored inside `pages.markdown`, or rendered from the bullet? If the user deletes or edits the H1 in `$EDITOR`, what happens? Inline markup in the title?
- An empty page saved (or not saved) from `$EDITOR`: created or discarded?
- There's no "delete page" command.
- Workflowy notes → pages on import: short notes become full pages; is the first-line preview the substitute for visible notes?

### 5.4 Editor round trip (§6.3)

Unspecified:

- Whether pages are included.
- How completion, tasks and headings are represented.
- What a duplicated `^id` (copy-pasted line) means.
- What a stripped `^id` does (silently loses history, page, links, star).
- Removing a parent line but keeping its children.
- A literal `^` in text; ID length/format in the file.
- Concurrent CLI edits while the editor is open.
- Temp files must be 0600 in a private directory, not world-readable `/tmp` (Neovim swap/undo files too).

### 5.5 Formatting (§5.8)

- Needs a grammar: nesting, overlap, escaping a literal `*`, `_italic_` / `__bold__` support, markup inside code spans.
- **Orange has no ANSI equivalent**; purple → magenta, gray → bright black (often unreadable). Specify the layer-1 mapping.
- Is "color the whole bullet" implemented by wrapping the text in `<color>` tags? Is it the same command as Alt+C?
- How does the app detect tmux (for the Ctrl+B → Alt+B rule) and legacy terminals (for Ctrl+I)?

### 5.6 Tags (§5.9)

- `#` collides with Markdown headings in pages (`# Title`), issue numbers (`#123`), `C#`, URLs with fragments, and code.
- Unicode letters allowed? Numeric-only tags? Trailing `/`?
- Display case vs stored lowercase.
- Tag rename command.
- With 8 palette colors, tag colors will repeat; fine as long as `#` text carries meaning.

### 5.7 Links and knowledge features (§10.1)

- What is the stored form of a `[[link]]`? If it's `[[id]]`, the raw text is no longer "readable in a SQLite browser" (:481).
- Obsidian links target files, not nodes; define export (`[[page#^id]]`? plain text?).
- Links to trashed or purged nodes.
- Copy link to node: what format?
- **Unlinked mentions** "containing the current node's text": long node texts rarely match; define tokenization/minimum length.
- **Properties** `status:waiting` share the namespace with `is:`, `has:`, `changed:`. Define precedence or reserve keys.
- **Saved queries**: virtual children in the outline — how do Enter/Tab/Backspace/move behave on them? Are they exported? How are they visually distinguished?
- **Templates**: strip `#template` from instantiated copies; variables (date)?
- **Today node**: date format, locale, timezone.

### 5.8 Screens that exist only as names

Need layouts, keys and mockups:

- Trash view (restore, purge, preview)
- History view (per-node versions, change-set list, revert)
- Stars view
- Sync view
- Help overlay
- Color picker
- Tag list
- Jump palette result rows
- Read-only mode (what happens when you type?)
- Narrow terminals (minimum width/height behavior)
- How the *focused* bullet is shown in text editing (terminal cursor only? subtle highlight?) — the page preview depends on "focused".

### 5.9 Visual system

- No **semantic color-token table** mapping UI roles (accent, muted, selection, error, chip, highlight, page marker, guides) to ANSI slots and to Omarchy palette keys. Agents will invent one otherwise.
- Detection of Unicode vs ASCII glyphs (locale?) and truecolor (`COLORTERM`, tmux `Tc`).
- Concealed markers appearing on the focused bullet reflow its line whenever focus moves, which works against "the cursor row stays fixed".
- Which density mode is the default; "top-level siblings" means top level of the zoom?
- Dim text on light themes often has poor contrast; specify a minimum.
- "Optional truecolor themes in config" vs omakase "settings for real needs, not every pixel".

### 5.10 Mockups (§4.2 vs §4.7)

They disagree on:

- Breadcrumb separator: `/` vs `›` (§5.7 also uses `›`).
- Whether "Root" is shown.
- Indent guides (`│`) present vs absent.
- Footer content.
- Header: `~/knowledge` vs the database name (which is `outline.db` by default).

### 5.11 Desktop integration

- `app capture` can't open a window by itself; it must spawn a terminal with a window class (`xdg-terminal-exec`? which terminals accept `--class`?).
- How the `theme-set` hook notifies running instances (signal? socket? file touch?), including multiple instances.
- Notifications and clipboard on macOS.
- `.desktop` entry is Linux-only; macOS launch story?

### 5.12 CLI (§10.2)

- Path addressing breaks with `/` in text (`#project/lighthouse`, dates like `10/17`), duplicate sibling names, and inline markup (match raw or rendered?). Needs escaping and ambiguity rules (error with candidates).
- `<id>` vs `<id|path>` is inconsistent across commands.
- No `--before` or index option for `move`; no `--under` for `import`.
- Missing commands: Trash list/empty, `forget`, `history purge` (only in §8.4), `capture`, heading/task setters, tags, an integrity `check`.
- `ops.json` format for `app apply` is undocumented.
- The `--json` envelope, error codes and exit statuses are unspecified.
- The name of the `--db` environment variable.
- Export `--out <dir>` vs stdout.
- `--source agent:<name>` is self-declared; fine for attribution, but say it is not a security boundary.
- Behavior when the database is read-only (newer schema) or locked past the busy timeout.

### 5.13 Configuration

Config keys are referenced throughout but never listed: column width, density, keymap, glyph set, footer, page preview, split-view width, trash days, history retention, scheduled export, mouse capture, tag colors, truecolor themes, zoom transition, default `--db`. Also: what happens when a saved `config.toml` is invalid (keep old config + banner?).

### 5.14 Error states

- Failed write (disk full, permission): are edits kept in memory and retried? Can the user keep typing?
- Banner wording and dismissal rules.

---

## 6. Feasibility and engineering notes

- **Commit target**: "< 5 ms per commit" with `synchronous=FULL` is optimistic. On macOS, real durability needs `PRAGMA fullfsync`, often 5–20 ms per commit. Some Linux SSDs also exceed 5 ms on fsync.
- **Timing in CI**: absolute timing targets are flaky on shared runners. Use fixed hardware or relative baselines.
- **Shared engine**: Bubbles `textarea` can't share your custom wrap and cursor engine (:285, :621). Plan a custom page editor.
- **Width library**: pick either `uniseg` or `runewidth` for width so they can't disagree.
- **Stale wording**: "Bubble Tea v2 if released" — confirm and update.
- **Pure-Go SQLite**: also verify the online backup API (needed for snapshots), not just FTS5.
- **"No manual verification"** (:676) conflicts with the hands-on keymap matrix in real terminals, the comparison-set evaluation and the calm review. List all human checkpoints explicitly.
- **Shutdown**: Ctrl+Q's "there is never anything to save" (:185) needs an explicit flush of coalesced typing on quit, SIGHUP (terminal closed) and SIGTERM.
- **Cold launch < 100 ms at 10k nodes** while loading the in-memory tree and running `quick_check`: make sure the check doesn't block the first frame.
- **Jump palette fuzzy matching over 100k nodes**: confirm `sahilm/fuzzy` speed at that size.

---

## 7. Editorial

- **Missing content**: §9.2 (:633) and §11 (:734) contain broken `&#91;embedded content…\]` placeholders. The package diagram and the whole phase plan with exit gates are missing, and the agent plan depends on them.
- **Self-reference**: :374 cites its own section ("§6.4, §6.6").
- **List markers**: the list switches from `-` to `*` at :483, so it renders as a separate list.
- **Toast wording** (:138): "and 214 children" should be "descendants".
- **Audience framing**: §3 targets users who value "plaintext, Git history", but storage is SQLite. Acknowledge that scheduled export is the bridge.
- **Scenario 3** says "in Neovim"; the mechanism is `$EDITOR`.

---

## Decisions (2026-10-17)

Answers to the open calls raised by this review. Applied to `SPEC.md` in later commits.

| # | Topic | Review items | Decision |
|---|---|---|---|
| D1 | Undo scope | 1.3 | Ctrl+Z undoes anything started from the TUI, including `$EDITOR` round trips and page saves. CLI and agent changes are reverted with `app revert` (Phase 1) or the History view (Phase 2). |
| D2 | First run | 1.5 | Create the sample outline that teaches the core keys. Drop the ASCII wordmark until there is a name. |
| D3 | Panes vs overlays | 1.9, 2.6 | Floating overlays only (backlinks, peek). Drop the Ctrl+W pane prefix. |
| D4 | Move to… in Phase 1 | 1.6 | Phase 1 ships a simple fuzzy parent picker built on the command palette. The full jump palette stays Phase 2. Copy link to node moves to Phase 2. |
| D5 | Inbox | 1.15 | A top-level `Inbox` node created on first use; its ID is stored in `meta`, so it can be renamed or moved. |
| D6 | Multi-select | 1.14 | Siblings only. Removed from open questions. |
| D7 | Leaving block selection | 2.3 | Non-command printable keys do nothing except show a hint. Enter and `a` return to text editing with the cursor at the end, `i` at the start. |
| D8 | Legacy / tmux fallbacks | 2.2 | Fallbacks are block-selection commands. Alt+key is treated as Esc then key. Document tmux `escape-time 10` and `extended-keys on`. |
| D9a | Alt+Down | 2.1 | Opens the page. Move subtree drops Alt+Up/Down; `J`/`K` in block selection is the fallback. |
| D9b | Zoom in block selection | 2.1 | `z` zooms in, `Z` zooms out. Enter returns to text editing. |
| D9c | Delete in block selection | 2.1 | Backspace, Delete and `d` all delete the selection, with an undo toast. |
| D9d | Following tags and links | 2.1 | Enter always edits in text editing. Alt+Enter (or click, Phase 2) follows the tag or link under the cursor. In page view, Enter follows the focused link, else enters edit mode. |
| D9e | Alt+Enter exception | 2.1, D8 | D8 applies to block-selection command keys. Alt+Enter received as one sequence follows the link; a slow Esc then Enter still returns to editing. |
| D10 | TUI vs CLI concurrency | 3.6 | Per-node version number. TUI typing is never dropped: it commits after the CLI change, which stays in history and can be reverted; a toast reports it. Agents can pass `--if-version`. |
| D11 | Split / merge identity | 3.1 | Split: the first half keeps the ID and everything attached. Merge: the page moves to the survivor if it has none, else no-op with a hint; star and inbound links are retargeted to the survivor. |
| D12 | Trash restore | 3.7, 1.13 | Record the deleting change set per node; restore brings back only that deletion, at its original position when possible. Missing parent (Trash and sync): nearest surviving ancestor, else root. Drop the "Recovered" node. |
| D13 | History policy | 1.10, 1.11, 1.12 | Typing is one op updated in place until sealed (2 s pause, leaving the node, structural action); only sealed ops reach the sync log. History is append-only except explicit erasure. Compaction squashes only TUI text edits per node per day. Forget node emits a sync tombstone. |
| D14 | Sync moves | §4 | Parent and position form one "location" field in ops. Phase 5 uses a proven tree-move algorithm (Kleppmann et al.); Phase 0 convergence tests include concurrent moves. |
| D15 | Mirrors | §4 | Drop the "without migration pain" claim; mirrors need their own design. Stays open after Phase 2. |
| D16 | App name | 1.8 | Allowed in user-facing places only: directories, binary, window class, export header comment. Never in schema, ops, sync log or JSON field names; `meta` stores a neutral format ID. |
| D17 | CLI naming | 3.13, 5.12 | Noun groups: `app trash list\|restore\|empty`, `app snapshot list\|restore`, `app history list\|purge`, alongside `app page get\|set`. Core verbs stay top-level. Snapshot restore refuses while a TUI holds the lock. |
| K1 | Filter key | §6.2 | Ctrl+F opens the filter from text editing, `/` from block selection. Typing `/` always inserts a slash. |
| K2 | Esc chain | 2.4 | Overlay or page → outline; text editing → block selection → clear filter → nothing, with a hint. Esc never zooms out. |
| K3 | Shift+Enter | 2.7 | Opens the page where distinguishable (kitty protocol); acts as Enter elsewhere. |
| K4 | Star target | 2.1 | Toggle star acts on the focused or selected bullet; star the zoom target from the zoom heading, which is focusable. |
