# Terminal Infinite Outliner — Project Specification v3

Oct 15, 2026 · @Thor

## 1. Overview

We are building a local-first, terminal-native infinite outliner in Go, inspired by Workflowy, for Linux and macOS. It is one continuous tree of bullets that you can capture into, indent, reorder, collapse and zoom. The outline is stored in a single SQLite file with full edit history and automatic snapshots, and can be exported to Markdown, JSON or OPML at any time.

**Goal:** a faithful Workflowy experience in a beautiful terminal UI that matches the Omarchy desktop theme. Obsidian-inspired knowledge features (links, backlinks, properties, saved queries) are the bonus.

**Pitch:** An infinite outline for your terminal: one tree, unlimited depth, stored in one open, durable file you can always export.

**Status:** Concept / pre-implementation. Working name TBD, written `<app>` in this spec. The name appears only in user-facing places: the binary, data and config directory names, the window class, and the comment or header that marks an export. It never appears in the schema, operations, the sync log format or JSON field names, so renaming the app later only means moving directories.

## 2. Product principles

1. **One tree, no mandatory pages.** Any node can have children and can be zoomed into.
2. **Thought-first editing.** Capturing and rearranging bullets needs no mode switch. Commands that need single keys use a lightweight block-selection state, not Vim modes. Block selection and page view/edit are modes in the strict sense; they are kept few, shallow, and always one Esc away from typing.
3. **Your data is never locked in.** One SQLite file with a plain, documented schema, readable by standard tools. Export to Markdown, JSON or OPML at any time; a JSON export restores everything.
4. **Nothing is lost silently.** Every change is recorded in history, deletes go to the Trash, and snapshots protect against a damaged file.
5. **Every change is attributable and reversible.** History records whether a change came from the TUI, the CLI, the editor round trip or an agent, and any change set can be reverted.
6. **Local-first and private.** No account, server, cloud dependency or network connection.
7. **Terminal-native.** Match the system theme (Omarchy first), read like a calm page rather than a dashboard, work inside tmux, degrade gracefully in legacy terminals, and keep every action available from the keyboard.
8. **Composable.** Users, scripts and agents work through a robust CLI with JSON output, never by touching the database directly.
9. **Simple core first.** The outlining experience comes before knowledge-management features.

## 3. Users, scenarios and comparison set

**Primary users** are Workflowy-style outliners who live in the terminal and value plaintext, Git history and local ownership. The data lives in SQLite; scheduled Markdown export (§7.5) is the bridge to plaintext and Git. Later, users who want scripts or coding agents to read and update their outline.

**Core scenarios:**

1. **Session.** Launch from a shell and type immediately. Enter adds bullets, Tab nests them. Collapse a project, zoom into a task, filter for a word, quit.
2. **Quick capture.** From any shell or a Hyprland keybind: `app add "Call the dentist"` appends to an Inbox node without opening the TUI.
3. **Edit in `$EDITOR`.** Press a key to open the current subtree as Markdown in `$EDITOR` (Neovim on Omarchy). Save and quit; the changes apply as one undoable step.
4. **Recover.** A week later, restore a deleted branch from the Trash, or revert one bad change made by a script without touching anything else.
5. **Leave.** `app export --format markdown` produces a readable outline that works anywhere, at any time.

**Comparison set** (evaluate hands-on before Phase 1 ends):

| Tool | Why it matters to us |
| --- | --- |
| Workflowy | The editing feel we are matching: zoom, Esc selection, Enter/Backspace rules |
| Logseq | Closest precedent: a Markdown-file outliner with block IDs. Study its ID scheme and its file-sync problems |
| Obsidian Outliner plugin | Outlining inside plain Markdown files; the interop target |
| Org-mode | Mature plaintext outlining, folding and refiling in a terminal editor |
| Bike, TaskPaper | Writing-first outliner feel; simple file formats |
| outl, hnb, h-m-m | Existing terminal outliners; baseline for what a TUI can feel like |

Our claimed difference: Workflowy-grade editing, in a terminal, with local data in an open format and complete, reversible history.

### Workflowy parity checklist

"Clone" means every *Must* row works the way Workflowy users expect. Phase numbers refer to §11.

| Workflowy feature | Verdict | Phase | Spec |
| --- | --- | --- | --- |
| One tree, zoom, breadcrumbs | Must | 1 | §4.1 |
| Enter / Tab / Backspace behavior | Must | 1 | §5.2–5.4 |
| Esc block selection, multi-select, bulk move | Must | 1 | §5.1 |
| Collapse, expand all, expand to level N | Must | 1 | §6.6 |
| Complete, show / hide completed | Must | 1 | §5.4 |
| Pages on bullets (replace Workflowy notes), optional first-line preview | Must | 1 | §5.7 |
| Inline formatting and text color | Must | 1 | §5.8 |
| Duplicate node, Move to… | Must | 1 | §5.4, §6.6 |
| Import from Workflowy (OPML) | Must | 1 | §7.5 |
| Undo across restarts, Trash | Must | 1 | §5.6, §8.3 |
| `#tags`, click to filter | Must | 2 | §5.9 |
| Links to other bullets | Must | 2 | §10.1 |
| Search operators (`is:complete`, `has:page`, `changed:7d`) | Must | 2 | §6.2 |
| Starred locations | Must | 2 | §6.6 |
| Templates | Later | 3 | §10.1 |
| Mirrors (one bullet in two places) | Later, undecided | 3 | §12.3 |
| Dates and a Today node | Later, optional | 3 | §10.1 |
| `@person` mentions | Never | — | — |
| Board (kanban) view | Never in V1 | — | — |
| Sharing and collaboration | Never | — | §12.1 |

## 4. Information model and UI

### 4.1 Global tree

- Exactly one root per database ("workspace" and "outline" both mean one database file). The root is a real row with `parent_id` NULL, its ID stored in `meta`, and is shown as **Home**. It can't be renamed, deleted, moved, completed or starred, and has no page.
- Every other bullet is a node with editable text, ordered children, and a permanent random ID (§7.2).
- No distinction between bullet, task or project. Any bullet can also carry a Markdown page (§5.7). Completion is an attribute any node can have.
- **Zoom** makes any node the temporary root. The zoomed node is shown as a heading, with only its descendants below it. The heading can be focused and edited like a bullet.
- **Breadcrumbs** show the path to the zoom target. They always start at Home and use `›` as the separator. Long paths are truncated in the middle: `Home › … › Development › Tooltips`.
- Collapse/expand is independent of zoom and persists across sessions (§7.2).
- No artificial depth limit.

### 4.2 Corrected mockup (zoomed into Projects)

```text
Home › Projects                                            outline  ●

Projects

▾ ControllerCraft
│ ▾ Development
│ │ • Inventory navigation        (completed: dimmed, struck)
│ │ • Improve controller mappings
│ │ • Fix tooltip positioning
│ ▸ Design notes
▸ Lighthouse

Enter new · Tab indent · Esc select · Ctrl+P commands · F1 help
```

### 4.3 Visual language

| State | Glyph | ASCII fallback |
| --- | --- | --- |
| Collapsed, has children | `▸` | `>` |
| Expanded, has children | `▾` | `v` |
| Leaf | `•` | `-` |
| Task, open / done | `☐` / `☑` after the bullet glyph | `[ ]` / `[x]` |
| Heading style | Bold text, extra space above | Bold, or `#` marker without bold |
| Completed | Dimmed + strikethrough text; glyph unchanged | Dimmed text + `x` marker |
| Selected (block selection) | Reverse-video row | Same |

- Completion never replaces the bullet glyph, so a completed parent still shows whether it is collapsed.
- No information is carried by color alone.
- Indent is 2 columns with faint `│` guides. Past 8 levels, indentation compresses and the status line suggests zooming.

### 4.4 Theme matching (Omarchy first)

**Must have:** on Omarchy, the app matches the desktop theme and follows every theme change while running, with no restart and no configuration.

- **Colors, layer 1 (always on):** all UI colors use the terminal's 16 ANSI colors plus its default foreground and background. Omarchy themes recolor the terminal, so the app follows instantly, on any OS, in any terminal.
- **Colors, layer 2 (Omarchy enhancement):** when an Omarchy installation is detected, the app reads the current theme's palette and uses its exact accent, selection and muted colors in truecolor. It also reads the theme's light/dark marker and adjusts dim and highlight levels.
- **Live updates:** the app watches the current-theme location (a symlink and directory that the theme switcher replaces) and reloads the palette within a second. Optionally, installing a `theme-set` hook gives instant notification instead of watching.
- **Fonts:** a terminal app always draws in the terminal's font, so the app matches Omarchy's font choice automatically. Omarchy's font switcher changes the terminal font, and the app re-renders on the resize/redraw event that follows.
- Omarchy's theme file layout has changed between releases, so the Omarchy reader is a small, isolated module with fallbacks: palette file, then terminal theme files, then plain ANSI. Verify paths against the current Omarchy release in Phase 1.
- Off Omarchy: layer 1 only, plus optional truecolor themes in config. Strikethrough, dim and italic each have fallbacks.

### 4.5 Chrome

- **Header:** breadcrumb left; database name (the file name without `.db`) and save indicator right.
- **Save indicator:** nothing when saved; `●` while a coalesced edit is not yet committed (at most about 1 second); a red label if a write fails.
- **Footer:** contextual key hints, dismissible via config. `?` opens a full help overlay.
- **Toasts:** one line, auto-dismissing, for undoable actions: `Moved "Design notes" and 214 descendants to Trash · Ctrl+Z`.
- **Banners:** persistent, for failed writes, a database opened read-only, or a restore that just ran.
- **Views on demand:** Trash and History open as temporary full-screen views, not panels. No sidebar, no daily journal.
- **First run:** create the database with a short sample outline that teaches the core keys.

### 4.6 Visual design

The outline should read like a calm page, not a terminal dashboard.

- **Centered reading column.** Text width is capped (default 90 columns, configurable) and centered in wide terminals, like Workflowy's page. Breadcrumbs and status sit inside the same column.
- **Inline formatting rendered, markers concealed.** `**bold**` shows as bold; the markers appear only on the bullet being edited, as in Obsidian's live preview (§5.8).
- **Typographic hierarchy:**
  - Optional heading bullets: bold, with extra space above, for section-like nodes.
  - A bullet with a page shows a dim page marker after its text (a Nerd Font document glyph on Omarchy, ≡ elsewhere). With the optional preview turned on, the page's first line appears dimmed beneath the focused bullet.
  - Muted breadcrumbs and footer; full-strength color only for text you wrote.
  - Tags as colored chips from the theme palette (§5.9).
- **Stability and motion:**
  - Scroll margin of 3 rows; the cursor never sits on the screen edge.
  - The cursor row stays fixed on screen when collapsing or expanding.
  - No flicker: only changed cells are redrawn.
  - Optional, very brief zoom transition, off by default.
- **Density modes:** *comfortable* (blank line between top-level siblings) and *compact*.
- **Glyph sets:** Nerd Font set by default on Omarchy, Unicode elsewhere, ASCII fallback.
- **Reference mockups** are part of the spec before Phase 1 ends: a 100-column laptop terminal, a full-screen ultrawide, a deep zoom, an active filter, and a light theme. Phase 1's exit gate includes a side-by-side "calm" review against Workflowy and against Omarchy's bundled terminal apps on the same screen.

### 4.7 At home on Omarchy

Omarchy is *omakase*: one beautiful, opinionated default, keyboard first, themed everywhere, no bloat, with plain config files for anyone who disagrees. The app follows the same rules, so it feels like it shipped with the system.

**Omakase defaults**

- One carefully tuned default look and keymap. Settings exist for real needs (column width, density, keymap), not for every pixel.
- No settings screen. "Edit settings" in the command palette opens the commented `config.toml` in `$EDITOR` and reloads it on save, the way Omarchy hands you its own config files.
- On Omarchy, defaults adapt automatically: Nerd Font glyphs on (Omarchy's fonts include them) and the Omarchy palette on (§4.4).

**Let the desktop frame the app**

- Hyprland already draws the window border, gaps and focus color, so the main outline draws no border of its own.
- The app never paints its own background. It uses the terminal's default background, so the theme's background and any terminal opacity show through.
- Rounded borders (`╭─╮`) appear only on floating overlays: palettes, help, peek and backlinks. The title sits in the top border and the border uses the theme accent, as in btop and lazygit.

**Shared keyboard conventions**

The full keymap is in §6.4. The conventions it follows:

- `?` in block selection, or F1 anywhere, shows every command with its key, as lazygit, Impala and BlueTUI do.
- Esc backs out one level and never zooms out: overlay or page → outline; text editing → block selection → clear filter. Zooming out is always explicit.
- Block selection accepts Vim motions (`h` `j` `k` `l`, `gg`, `G`), matching Omarchy's Neovim-first setup.
- There are no side panes. Backlinks and peek are floating overlays like the palettes; Esc closes them.
- Ctrl+Q quits from anywhere. Pending typing is flushed first, so there is never anything to save.

**Native app integration**

- Ships a `.desktop` entry and its own window class, so it launches from the app launcher like Omarchy's other terminal apps and can be targeted by Hyprland window rules.
- **Capture window:** `app capture` opens a small floating window (via a window rule on its class). Type one line, press Enter, and it lands in the Inbox and the window closes. The docs suggest a Super binding, checked against Omarchy's defaults.
- Desktop notifications only for things that need attention, such as a failed snapshot, through the standard notification daemon.

**Restraint**

- No splash screen, toolbars or busy status bar. Breadcrumbs, the save dot and one line of hints are all the chrome.
- Instant response is part of the look: every keypress lands in the next frame (§9.4).

**Reference sketches** (the window border is Hyprland's, not the app's):

```text
  Home › Projects › ControllerCraft

  ControllerCraft

  ▾ Development
    • Inventory navigation                    ← completed: dimmed, struck
    • Improve controller mappings  #input     ← tag chip in theme color
    • Fix tooltip positioning  ≡
      Clips at the right edge on 1080p        ← page preview (optional)
  ▸ Design notes

                                      ? help · ^P commands · ^K jump
```

```text
             ╭─ Commands ─────────────────────────────╮
             │ › col                                  │
             │   Color bullet…                 Alt+C  │
             │   Collapse all                         │
             │   Expand to level…           Alt+1–9   │
             ╰────────────────────────────────────────╯
```

## 5. Editing behavior

### 5.1 Two states, no modes

- **Text editing (default).** Printable keys type. Commands use modifiers.
- **Block selection.** Esc selects the current bullet as a whole. Up/Down move the selection; Shift+Up/Down extend it across siblings only. Single keys act on the selection (§6.4). Enter or `a` returns to text editing with the cursor at the end of the bullet, `i` at the start. Other printable keys do nothing except show a hint, so text typed after Esc never runs commands by surprise.
- Block selection accepts Vim-style motions by default (§4.7) because they are single keys. Full Vim-style modal editing is an optional keymap in Phase 4 (§11), never a requirement.

### 5.2 Enter

| Cursor position | Result |
| --- | --- |
| Start of a non-empty bullet | Insert an empty sibling above; cursor stays on the current text |
| Middle of text | Split: text after the cursor becomes a new sibling below; children stay with the first half |
| End of an expanded bullet with children | Create a new first child |
| End of a collapsed bullet, or a leaf | Create the next sibling |
| Empty bullet that is not at the top level of the current zoom | Outdent it (Workflowy behavior) |

On a split, the first half keeps the node's identity and everything attached to it: ID, children, page, star, completion, inbound links and history. The second half is a new node.

### 5.3 Backspace at the start of a bullet

| Situation | Result |
| --- | --- |
| Bullet is empty, no children | Delete it; cursor to the end of the previous visible bullet |
| Bullet is empty, has children | No-op; status line hints to use Esc then Delete |
| Bullet has text, no children | Merge text into the end of the previous visible bullet |
| Bullet has text and children, previous bullet has no children | Merge text; children are adopted by the previous bullet |
| Bullet has text and children, previous bullet has children | No-op with a hint. Merging would interleave two child lists unpredictably |
| Both bullets have a page | No-op with a hint, for the same reason |

On a merge, the previous bullet survives with its ID, completion state and history. The merged-away bullet goes to the Trash in the same change set; its page moves to the survivor (if the survivor has none), and its star and inbound links are retargeted to the survivor.

Delete at the end of a bullet mirrors these rules forward.

### 5.4 Structure

- **Tab / Shift+Tab** indent or outdent the current bullet (or selection) with its descendants. Indent is a no-op on the first sibling.
- **Move up/down** among siblings moves whole subtrees. Moving past the first or last sibling moves to the adjacent parent's sibling list, as Workflowy does.
- **Delete a subtree** (Backspace, Delete or `d` in block selection) never asks for confirmation. It shows an undo toast with the descendant count.
- **Complete / reopen** toggles a node's completion (storage rule: §7.2). Completed nodes stay in place; a toggle hides them.
- **Tasks:** toggle task makes a bullet a task, shown with a checkbox (§4.3). Typing `[ ] ` at the start of a bullet does the same and removes the marker. Completing a task checks its box.
- **Duplicate** copies the subtree as the next sibling, with new IDs, pages, completion, task and heading state. Stars are not copied, and links inside the copy still point to their original targets.
- **Heading style:** toggle heading makes a bullet a section-like heading (§4.6). It changes only the look, never the structure.
- **Move to…** opens a fuzzy parent picker over all node text, built on the command palette (Phase 1). From Phase 2 it uses the jump palette, with starred locations first. This is how subtrees cross long distances.

### 5.5 Paste and clipboard

- Bracketed paste is enabled. Single-line paste inserts text.
- Multi-line paste is parsed as Markdown into a subtree inserted as siblings after the cursor. Plain lines become bullets; indentation becomes nesting.
- Copy in block selection writes indented Markdown to the system clipboard via OSC 52, with `wl-copy` as a fallback.

### 5.6 Undo and redo

- Undo is driven by history (§8.2). Each change set is one undo step. Undo applies the inverse of the newest change set not yet undone as a new change set whose `undo_of` points at it; redo undoes that undo. The undo position is derived from history, so it needs no extra state.
- Any new change set from the TUI clears the redo steps.
- If a later change from another source (CLI, agent, sync) touched the same fields, undo follows the revert rule: those fields are skipped and reported (§8.2).
- Undoing a change outside the current zoom zooms to show it, with a toast.
- Typing coalesces into one step per node, closed by a 2-second pause, leaving the node, or any structural action.
- Undo restores cursor and selection position, recorded with each change set.
- **Undo survives restarts.** Ctrl+Z after reopening the app continues where you left off.
- Ctrl+Z undoes change sets you started from the TUI, including `$EDITOR` round trips and page saves made from it. Changes from the CLI or an agent are never undone by Ctrl+Z, so you never undo something you didn't do; they are reverted with `app revert` (Phase 1) or from the History view (Phase 2).

### 5.7 Pages

- **Every bullet stays one line.** Long-form writing lives on an optional **page** attached to the bullet: a full Markdown document with headings, lists, code blocks, quotes and tables, as long as you like.
- **Page zoom:** Alt+Down (also Shift+Enter, or `p` in block selection) opens the bullet's page in view mode; on a bullet without one, it creates an empty page and opens edit mode. Esc from view mode, or Alt+Left, returns to the outline with the cursor where it was. Normal zoom (Alt+Right) still shows only the bullet's children.
- **Title:** the bullet's text is the page title, shown as its heading. Editing either one changes both.
- **Indicator and preview:** a dim page marker follows the bullet's text. An optional setting, off by default, shows the page's first non-empty line dimmed beneath the focused bullet (§4.6).
- **Two modes, no live preview.** Editing inside rendered output is the hardest and buggiest part of a Markdown editor, so pages separate reading from writing:
  - **View mode** renders the Markdown read-only: headings, lists, quotes, tables, and code blocks syntax-highlighted with the theme palette. It scrolls, and Tab moves between links; Enter on the focused `[[link]]` zooms to its target.
  - **Edit mode** is a plain multi-line text editor showing the Markdown exactly as typed. Syntax highlighting keeps it pleasant without concealment: headings bold in the accent color, markers dimmed, code blocks tinted, links underlined. It shares the outline's wrap-layout and cursor engine.
- **Switching:** `e` in view mode, or Enter with no link focused, enters edit mode; Esc in edit mode returns to view mode. The parser's source ranges keep your place: switching scrolls to the same block and puts the cursor on it.
- **Split view** (wide terminals): `s` in view mode shows source and rendered output side by side, updating as you type. Off by default; a setting can make it the default above a chosen width.
- **Common elements of both modes:**
  - The breadcrumb continues with a page marker and the mode: `Projects › ControllerCraft › ≡ view`.
  - Same centered reading column and theme as the outline.
- **Editing, staged:**
  - **Phase 1:** view mode, with `e` opening the page in `$EDITOR` (Neovim on Omarchy). On exit, the page is saved back as one undoable change set.
  - **Phase 2:** the built-in edit mode, mode switching with position sync, and split view. `e` then opens the built-in editor and `E` opens `$EDITOR`.
  - Live preview is not planned. Concealed markers remain only on single-line bullets in the outline (§5.8), where they are simple.
- **A page belongs to its bullet:** it moves with it, goes to the Trash and comes back with it, and is copied by Duplicate.
- **Links and tags work on pages:** `[[` links to bullets, `#tags` are indexed, and pages appear in backlinks (Phase 3). Properties and saved queries live at the top of a page (§10.1).
- **Rule of thumb for users:** structure goes in bullets, prose goes in pages. Lists inside a page are text, not tree nodes, so they can't be zoomed or moved in the outline.
- **Export:** each page becomes its own `.md` file linked from the outline export (§7.5). OPML export stores the page in the `_note` attribute, which Workflowy reads as a note.

### 5.8 Inline formatting and color

Formatting uses Markdown-style markers, typed directly or applied with shortcuts. Markers are concealed except on the bullet being edited.

| Style | Syntax | Shortcut |
| --- | --- | --- |
| Bold | `**text**` | Ctrl+B (Alt+B inside tmux, §6.4) |
| Italic | `*text*` | Ctrl+I (Alt+I without the kitty protocol, §6.4) |
| Strikethrough | `~~text~~` | — |
| Inline code | `` `text` `` | — |
| Link (URL) | `[label](https://…)` | — |
| Highlight | `==text==` | — |
| Text color | `<red>text</red>` | Alt+C opens a color picker |

- **Colors:** `red`, `orange`, `yellow`, `green`, `cyan`, `blue`, `purple`, `gray`. Each maps to the theme palette (ANSI first, Omarchy truecolor when available), so colored text follows theme changes.
- `<red>` and `==highlight==` were chosen because they are short and readable when concealment is off. `<red>` is not HTML, so Markdown export converts it (below). `==highlight==` renders in Obsidian but not in plain CommonMark tools. Unknown tags are left as plain text.
- Shortcuts apply to the text selection (Shift+Left/Right in text editing) or, in block selection, to the whole bullet.
- A **color the whole bullet** command applies a color to all selected bullets at once.
- Markdown export keeps the markers; colors export as `<span style="color: …">` so Obsidian renders them; plain-text export strips all markup.

### 5.9 Tags

- `#tag` anywhere in a bullet or page. Letters, digits, `-`, `_` and `/` are allowed; `/` makes nested tags (`#project/lighthouse`), as in Obsidian.
- Tags render as colored chips. Each tag gets a stable color from the palette, overridable in config.
- Alt+Enter on a tag, or a click (Phase 2), sets the in-place filter to that tag. Enter never follows a tag; it always edits. Typing `#` offers completion from existing tags.
- A tag list in the command palette shows every tag with its count.

## 6. Navigation, search, integration and input

### 6.1 Navigation

- Collapse/expand one node, or all its descendants (keys in §6.4).
- Zoom in on the current node; zoom out one level; zoom to root.
- The cursor position is remembered per zoom level. Zooming out returns exactly where you were.
- Up/Down move through wrapped visual lines, not just logical bullets. Home/End are per visual line, then per bullet on a second press.

### 6.2 Search

- **In-place filter (Phase 1).** Ctrl+F (or `/` in block selection) filters the tree as you type: matching nodes stay visible with their ancestors, matches highlighted, everything else hidden. This includes collapsed branches and page text. Esc clears it and keeps the cursor on the selected match.
- **Jump palette (Phase 2).** Ctrl+K opens a fuzzy finder over all node text, showing each result's breadcrumb, with starred locations listed first. Enter zooms to the result's parent with the result selected. From Phase 2 the same palette backs "Move to…" (§5.4).
- **Search operators (Phase 2),** Workflowy-style, in both the filter and the palette:
  - `is:complete`, `is:open` (not completed, any bullet), `is:task` (open or done), `has:page`, `is:starred`; `is:task is:open` finds open tasks
  - `#tag` (includes nested tags), `changed:7d`, `created:2026-10`
  - `"exact phrase"`, `-word` to exclude, terms combined with AND by default, `OR` between terms
- Case-insensitive substring matching by default, ignoring diacritics. FTS5 with the `trigram` tokenizer serves queries of three or more characters; shorter queries scan the in-memory tree. Phase 0 verifies Unicode case folding, diacritics and the 50 ms target at 100,000 nodes with the chosen driver.
- Trashed nodes never match.
- A match inside a page keeps its bullet visible with the page marker highlighted, and shows the matching line dimmed beneath it.
- Saved queries arrive in Phase 3 (§10.1).

### 6.3 External editor and quick capture

- **Edit in `$EDITOR`** (Phase 1): writes the current subtree to a temp Markdown file, suspends the TUI and opens `$EDITOR`. Each line carries its node's ID as a `^id` suffix so edits map back.
- On exit, the file is parsed and diffed against the subtree. Changed lines update nodes; new lines become new nodes; removed lines move to the Trash. Everything applies as one change set with source `editor`, undoable with Ctrl+Z.
- If the file can't be parsed, nothing is applied: the temp file is kept and a banner offers to reopen it.
- **Quick capture** (Phase 1): `app add "text" [--under <path-or-id>]` appends to the Inbox node by default, in its own transaction. The Inbox is a top-level node named `Inbox`, created on first use; its ID is stored in `meta`, so it can be renamed or moved freely and capture follows it. A running TUI notices the change through SQLite's `PRAGMA data_version` and refreshes.

### 6.4 Keymap and terminal support

This section is the only place key bindings are defined. Other sections name commands and may mention their keys, but this table wins. In code the keymap is data, one table per context, and a test fails if a key is bound twice in one context or a command has no binding that works without the kitty keyboard protocol.

**Contexts:** global, text editing, block selection, filter input, page view, page edit (Phase 2) and overlays (palettes, pickers, help, backlinks, peek). The Esc chain moves between them (§4.7).

**Global** (every context)

| Command | Key | Notes |
| --- | --- | --- |
| Quit | Ctrl+Q | Flushes pending typing first |
| Help overlay | F1 | Also `?` in block selection; F1 needs Fn on Mac laptops |
| Command palette | Ctrl+P | |
| Jump palette (Phase 2) | Ctrl+K | |
| Undo / redo | Ctrl+Z / Ctrl+Shift+Z | Ctrl+Y also redoes, for terminals that can't send Ctrl+Shift+Z |

**Text editing** (the default state). The fallback column applies without the kitty keyboard protocol, inside tmux, and on macOS with Option not set as Meta.

| Command | Key | Fallback |
| --- | --- | --- |
| New bullet, split, outdent empty bullet | Enter | Same (§5.2) |
| Backspace / Delete at bullet edges | Backspace / Delete | Same (§5.3) |
| Indent / outdent | Tab / Shift+Tab | Same |
| Block selection | Esc | Same |
| Move by character / visual line | Arrow keys | Same |
| Move by word | Ctrl+Left / Ctrl+Right | Same; see macOS below |
| Start / end of visual line, then bullet | Home / End | Same |
| First / last bullet in the zoom | Ctrl+Home / Ctrl+End | Same |
| Scroll a screen | Page Up / Page Down | Same |
| Delete word backward | Ctrl+W, Ctrl+Backspace, Alt+Backspace | Ctrl+W |
| Extend text selection | Shift+arrows, Shift+Home/End, Ctrl+Shift+Left/Right | Same |
| Copy / cut selected text | Ctrl+C / Ctrl+X | Same. Ctrl+C with no selection shows "Ctrl+Q quits" |
| Paste | The terminal's paste (Ctrl+Shift+V, Cmd+V) | Bracketed paste (§5.5); the app binds no paste key |
| Bold / italic | Ctrl+B / Ctrl+I | Alt+B inside tmux (whose prefix is Ctrl+B) / Alt+I (Ctrl+I equals Tab) |
| Color picker | Alt+C | Esc `c` |
| Toggle complete | Ctrl+Enter | Esc `x` |
| Move subtree up / down | Ctrl+Shift+Up / Down | Esc `K` / `J` |
| Collapse / expand | Ctrl+Up / Ctrl+Down | Esc `h` / `l` |
| Zoom in / out | Alt+Right / Alt+Left | Esc `z` / `Z` |
| Zoom to root | Alt+Home | Esc `gh` |
| Expand to level 1–9 | Alt+1 … Alt+9 | Esc `1` … `9` |
| Open page (page zoom) | Alt+Down, Shift+Enter | Alt+Down, Esc `p`. Without the kitty protocol Shift+Enter arrives as Enter |
| Follow the tag or link under the cursor | Alt+Enter | Same (see the Alt rule below); click in Phase 2 |
| Toggle star (Phase 2) | Alt+S | Esc `s` |
| Filter | Ctrl+F | Esc `/` |
| Open subtree in `$EDITOR` | Ctrl+E | Esc `e` |
| Duplicate subtree | Ctrl+D | Same |
| Show / hide completed | Ctrl+O | Same |

**Block selection.** Single keys act on the selected bullets. Modifier bindings from text editing that act on a bullet also work here and apply to the whole selection; Ctrl+B, Ctrl+I and Alt+C format whole bullets (§5.8).

| Key | Command |
| --- | --- |
| Up / Down, `j` / `k` | Move the selection |
| Shift+Up / Shift+Down | Extend the selection across siblings |
| `gg` / `G` | First / last visible bullet |
| `h` / `l` | Collapse / expand |
| `H` / `L` | Collapse / expand all descendants |
| `1` … `9` | Expand to level N |
| `J` / `K` | Move subtree down / up |
| Tab / Shift+Tab | Indent / outdent |
| `z` / `Z` | Zoom in / out |
| `gh` | Zoom to root |
| Enter, `a` / `i` | Return to text editing, cursor at the end / start |
| `x` | Toggle complete |
| `t` | Toggle task |
| `#` | Toggle heading style |
| Backspace, Delete, `d` | Delete to the Trash, with an undo toast |
| `y`, Ctrl+C | Copy as Markdown (§5.5) |
| Ctrl+X | Cut: copy, then delete to the Trash |
| `p` | Open page |
| `e` | Open subtree in `$EDITOR` |
| `c` | Color picker |
| `s` | Toggle star (Phase 2) |
| `m` | Move to… (§5.4) |
| `/` | Filter |
| `?` | Help overlay |
| Esc | Clear an active filter; otherwise nothing, with a hint |
| Other printable keys | Nothing, with a hint |

**Filter input** (§6.2): typing edits the query. Enter or Down moves focus to the outline with the filter still active, on the selected match. Esc clears the filter and leaves the cursor on the selected match. With the filter active, Ctrl+F (or `/` in block selection) returns to the input, and Esc in block selection clears it.

**Page view** (§5.7)

| Key | Command |
| --- | --- |
| Up / Down, `j` / `k` | Scroll by line |
| Space, Page Down / Page Up | Scroll by screen |
| `gg` / `G` | Top / bottom |
| Tab / Shift+Tab | Focus next / previous link |
| Enter | Follow the focused link; with no link focused, edit |
| `e` | Edit: `$EDITOR` in Phase 1, the built-in editor from Phase 2 |
| `E` (Phase 2) | Open in `$EDITOR` |
| `s` (Phase 2) | Toggle split view |
| Esc, Alt+Left | Back to the outline, cursor where it was |

**Page edit** (Phase 2): the text-editing keys for movement, selection, clipboard and formatting. Enter inserts a newline, Tab inserts two spaces, and Esc returns to view mode.

**Overlays:** typing filters the list; Up/Down or Ctrl+N/Ctrl+P move; Enter chooses; Esc closes.

**Terminal rules**

- **Kitty keyboard protocol.** The app enables it through Bubble Tea when available. Without it, Ctrl+Enter, Shift+Enter, Ctrl+Shift+Z and Ctrl+I can't be told apart from Enter, Enter, Ctrl+Z and Tab, so the fallback column applies.
- **Alt is Esc then key.** Legacy terminals and tmux send Alt+key as Esc followed by the key, and a quick Esc then key arrives the same way. The app treats both identically: block-selection keys run as if Esc then the key were typed, which leaves you in block selection. There are two exceptions, where text editing binds Alt+key to something else: Alt+Enter follows a tag or link, and Alt+I applies italic. Those bindings win when the sequence arrives quickly; Esc then Enter or `i` typed slowly (after the escape timeout) keeps its block-selection meaning. The duplicate-key test treats any other Alt binding that differs from its Esc meaning as an error. A bare Esc is recognised after Bubble Tea's escape timeout; the kitty protocol removes that wait.
- **tmux.** Inside tmux (detected through `$TMUX`), bold moves to Alt+B. The docs recommend `set -s escape-time 10` and `set -s extended-keys on` (with the `extkeys` terminal feature) and ship the snippet.
- **Readline and Cocoa keys** are deliberately not emulated, except Ctrl+W (delete word). Ctrl+E, Ctrl+D, Ctrl+K and Ctrl+P are app commands; Home, End and Delete cover their usual editing meanings.
- **Flow control.** Raw mode disables XON/XOFF, so Ctrl+Q and Ctrl+S reach the app; the test matrix verifies this.
- **Compositors.** Bindings that a desktop compositor commonly intercepts (Super combinations under Hyprland) are never used.
- Keymap is configurable in TOML, per context. The same duplicate check runs when the config loads; on a conflict the app keeps the default binding for that command and shows a banner.
- A test matrix covers Ghostty, Kitty, Alacritty, foot and WezTerm on Linux, and Ghostty, Kitty, Alacritty, WezTerm, iTerm2 and Terminal.app on macOS, each bare and inside tmux, recording which bindings arrive intact.

**macOS keyboard:** by default, macOS terminals send special characters (å, ∫) for Option+key instead of Alt, so every Alt binding has an Esc-based fallback, and on first run on macOS the app shows a one-line hint explaining how to set Option to act as Meta (Alt). macOS also intercepts Ctrl+Up/Down (Mission Control, App Exposé) and Ctrl+Left/Right (Spaces) by default; the same hint mentions this, and `h`/`l` in block selection always work. How Option+Left/Right (word movement in most Mac apps) arrives in each terminal is recorded in the test matrix. Apple's Terminal.app is treated as ANSI-only; Ghostty, kitty, WezTerm, Alacritty and iTerm2 are fully supported.

### 6.5 Mouse

- Phase 2: scroll, click to place the cursor, click a glyph to toggle collapse.
- Mouse capture disables the terminal's native text selection. The app documents Shift+drag (which most terminals use to bypass capture) and offers a config toggle to turn capture off.
- Drag-to-move nodes is exploratory; keyboard moves remain complete without it.

### 6.6 Palettes, stars and small commands

- **Command palette (Ctrl+P, Phase 1):** every command searchable by name, showing its key binding, recently used first. It is the discoverability layer for everything in this spec, so terminal key limits never hide a feature.
- **Stars (Phase 2):** "Toggle star" stars the bullet you're on: the focused bullet, or the selected ones in block selection. To star the zoom target, toggle it on the zoom heading. Stars appear first in the jump palette and in a Stars view; `is:starred` finds them.
- **Small commands (Phase 1),** with keys in §6.4:
  - Expand all / collapse all under the current node
  - Expand to level 1–9
  - Duplicate subtree
  - Show / hide completed
  - Toggle task, toggle heading style
  - Move to… (fuzzy parent picker, §5.4)
  - Copy link to node — Phase 2, with node links (§10.1)
  - Sort children (alphabetical, completed last) — Phase 2

## 7. Storage architecture

The outline lives in one SQLite database, which is the only source of truth. Snapshots and exports are copies of it.

### 7.1 Locations

```text
Data directory
  Linux:  ~/.local/share/<app>/            (honors XDG_DATA_HOME)
  macOS:  ~/Library/Application Support/<app>/

  outline.db          # the data (WAL mode)
  snapshots/          # automatic backups (§8.5)
  exports/            # default target for scheduled export

Config (both platforms, CLI convention)
  ~/.config/<app>/config.toml              (honors XDG_CONFIG_HOME)

Sync folder (Phase 5, user-chosen, e.g. inside iCloud Drive or Syncthing)
  <sync-folder>/<app>/<device-id>.log      # §7.6
```

Separate outlines are separate database files, selected with `--db` or a config default. `app paths` prints every location in use.

### 7.2 Schema (sketch)

```sql
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT);
-- schema_version, format_id (neutral, never the app name), device_id,
-- root_id, inbox_id, a one-paragraph description of the schema

CREATE TABLE nodes (
  id           TEXT PRIMARY KEY,     -- random, stable forever
  parent_id    TEXT REFERENCES nodes(id),
  position     TEXT NOT NULL,        -- fractional index among siblings
  text         TEXT NOT NULL,        -- one line, inline markup as typed (§5.8)
  is_task      INTEGER NOT NULL DEFAULT 0,
  is_heading   INTEGER NOT NULL DEFAULT 0,  -- heading-style bullet (§4.6)
  completed_at TEXT,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL,
  version      INTEGER NOT NULL DEFAULT 1,  -- bumped on every write (§7.3)
  deleted_at   TEXT,                 -- soft delete; non-null = in Trash
  deleted_by   TEXT                  -- change set that deleted it (§8.3)
);

CREATE TABLE pages (                 -- at most one page per bullet (§5.7)
  node_id    TEXT PRIMARY KEY REFERENCES nodes(id),
  markdown   TEXT NOT NULL,          -- full Markdown document
  updated_at TEXT NOT NULL,
  version    INTEGER NOT NULL DEFAULT 1   -- bumped on every write (§7.3)
);

CREATE TABLE node_tags (             -- derived from bullet text and page on every write
  node_id TEXT NOT NULL REFERENCES nodes(id),
  tag     TEXT NOT NULL,             -- lowercased, e.g. 'project/lighthouse'
  PRIMARY KEY (node_id, tag)
);

CREATE TABLE stars (
  node_id    TEXT PRIMARY KEY REFERENCES nodes(id),
  starred_at TEXT NOT NULL
);

CREATE TABLE view_state (           -- per device; not in history or sync
  node_id   TEXT PRIMARY KEY REFERENCES nodes(id),
  collapsed INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE ui_state (              -- per device; not in history or sync
  key   TEXT PRIMARY KEY,            -- show_completed, zoom, cursor:<zoom-id>, palette_recent
  value TEXT NOT NULL                -- JSON
);

CREATE INDEX nodes_by_parent  ON nodes(parent_id, position);
CREATE INDEX nodes_in_trash   ON nodes(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX node_tags_by_tag ON node_tags(tag);

-- change_sets and ops: see §8.2
-- search_fts: FTS5 index over live (non-trashed) nodes.text and pages.markdown,
--             maintained by triggers (§6.2)
```

- **IDs** are random and permanent. They are what the CLI, links and the editor round trip use.
- **Ordering** uses fractional index strings, so moving or reordering a node rewrites one row.
- **Completion:** `is_task` marks an open task (`[ ]`); `completed_at` marks completion for any node. Reopening clears `completed_at` and keeps `is_task`, so a plain bullet never turns into a task by accident.
- **View state** (collapse) and **UI state** (show/hide completed, the current zoom, the cursor per zoom, palette recents) are per device, excluded from history and sync: folding never creates undo steps. Both are included in JSON export so a restore brings back the same view. The filter is never persisted.
- **Timestamps** are RFC 3339 in UTC with milliseconds (`2026-10-17T09:30:00.123Z`), displayed and interpreted in local time (`created:2026-10`, the Today node). `updated_at` changes when the node's own text, page, task, heading or completion changes, not on moves, stars or changes to descendants; `changed:7d` uses it.
- **Integrity:** `PRAGMA foreign_keys=ON` on every connection. History ops reference nodes by ID without a foreign key, so purging a node never cascades into history (§8.3).
- **File permissions:** the data directory is created `0700`, and the database, snapshots and exports `0600`.
- **Search** uses an FTS5 index kept in sync by triggers (§6.2).
- The schema stays plain enough that anyone opening the file in a SQLite browser understands it in a minute.
- **Formatting** is stored as the markup the user typed, not as separate style rows. The text stays readable in any SQLite browser and exports without conversion; rendering parses it on display.
- **Tags** are kept in `node_tags`, rebuilt from the bullet text and page in the same transaction as every edit, so tag filters and the tag list are indexed queries. A tag written in a page attaches to the page's bullet.
- **Stars** are user data and part of history; collapse state is not.

### 7.3 Transactions and concurrency

- Every user action is one transaction that updates `nodes` and writes its history together (§8.2).
- Typing is committed as its coalesced operation at most about every second, so a crash loses at most the last second of typing.
- Pending typing is flushed and sealed on quit, on SIGHUP (terminal closed) and on SIGTERM, before the app exits.
- WAL mode with `synchronous=FULL`. The extra cost is negligible at this write rate.
- The TUI keeps an in-memory tree for rendering; all writes go through `core` to `store`.
- The CLI opens its own connection with a busy timeout. The TUI polls `PRAGMA data_version` about every 500 ms and reloads changed nodes.
- **Concurrent edits to one node.** Every write bumps the node's `version`. If another connection changes a node while the TUI holds uncommitted typing on it, the user's typing is never dropped: it commits as a new change set after the other change, which stays in history and can be reverted, and a toast says so. A CLI or agent that must not overwrite newer work passes `--if-version <n>` and gets a conflict error instead. The same rule applies to a page open in `$EDITOR` while `app page set` runs.

### 7.4 Migrations

- Schema versions are numbered and migrations run forward only.
- A snapshot is taken before every migration.
- A database with a newer schema than the app knows opens read-only with a banner, never migrated backwards.

### 7.5 Export and import

| Format | Export | Import |
| --- | --- | --- |
| JSON | Lossless: every node including the Trash, pages, stars, collapse and UI state; IDs, positions and timestamps. History with `--history` | Into an empty database only. `--replace` snapshots the database, then replaces its contents. `--under <id>` imports as new nodes with new IDs instead |
| Markdown | The shared dialect below; one file, or one file per top-level node; pages as separate files | New nodes under a chosen parent |
| OPML | Workflowy-compatible (mapping below) | New nodes under a chosen parent (for migrating from Workflowy and other outliners) |
| Plain text | Indented lines with all markup stripped; pages omitted | — |

- JSON export never contains the `device_id`, the config file or snapshots. An import always keeps the receiving installation's own `device_id`.
- Every export records the app and the export time without breaking its format: a top-level `exported_by` field in JSON, `<head>` elements in OPML (after the `<?xml` declaration), and an HTML comment on the first line of Markdown, which importers skip.
- Exports are written atomically (temp file, then rename).
- **Scheduled export** is an optional setting: Markdown or JSON, daily or on quit, into a folder you choose (for example a Git repo), keeping the last N exports. There is no daemon: a daily export runs on quit, or at launch if it is overdue. `app export --scheduled` runs it from a systemd timer, launchd or cron.
- CI checks that database → JSON → database is identical: every exported field is equal, including IDs, positions, timestamps, pages, stars and view state, and history when exported with `--history`.

**Markdown dialect.** Export, import, multi-line paste (§5.5) and the editor round trip (§6.3) share one dialect, close to GitHub-flavored Markdown:

```markdown
- Plain bullet
- [ ] Open task
- [x] Completed task
- ## Heading-style bullet
  - Child, indented two spaces
- Bullet text with its ID ^k3j9x2
```

- Inline markup is written as stored (§5.8). Export converts color tags to `<span style="color: …">`; the editor round trip keeps them as typed.
- The ` ^id` suffix is written by the editor round trip and by `app export --ids`. Only the editor round trip maps IDs back to existing nodes; Markdown import always creates new nodes and ignores the suffix.
- A completed bullet that is not a task exports as `[x]` and re-imports as a completed task. This small loss is documented; JSON and the editor round trip (through `^id`) keep it exact.
- Import accepts any consistent indentation (spaces or tabs) and `-`, `*`, `+` or numbered list markers. Lines that aren't list items become bullets.

**Pages in exports:** Markdown export writes each page as its own `.md` file in a `pages/` folder, and the outline links to it. File names are a slug of the bullet text (lowercase letters, digits and hyphens, at most 50 characters) plus the node ID, so they never collide. The result reads much like an Obsidian vault. JSON export includes pages in full; OPML puts them in `_note`.

**Workflowy OPML mapping.** Verified against a real Workflowy export in Phase 1 (§12.4).

| Workflowy | Imported as | Exported back |
| --- | --- | --- |
| Outline text with inline HTML | Bullet text with Markdown-style markup (§5.8) | Workflowy HTML |
| `<b>`, `<i>`, `<s>`, `<code>`, `<a href>` | `**…**`, `*…*`, `~~…~~`, `` `…` ``, `[label](url)` | Same tags |
| `<span class="colored c-red">` and other text colors | `<red>…</red>`, nearest of the eight colors | `colored` span |
| Background-color spans | `==…==` | Yellow background span |
| `_note` | Page | `_note`, as Markdown text |
| `_complete="true"` | Completed bullet (not a task) | `_complete="true"` |
| `#tags` | Tags, unchanged | Unchanged |
| `@mentions`, dates | Plain text | Plain text |
| Mirrors | Independent copies; the mirror link is lost | — |
| Boards | Plain children; the board layout is lost | — |

### 7.6 Sync readiness

Multi-device sync ships in Phase 5, and an iPhone app is a separate follow-up project. Phase 0 designs the data so that sync is an added feature, not a rewrite.

**The model: per-device logs in a shared folder.** Each device appends its operations to its own file in a user-chosen sync folder and reads the others' files to merge. Devices never write the same file, so the file-sync tool never sees a conflict. No server is involved.

- Works with any folder-sync tool: iCloud Drive (Mac and iPhone), Syncthing (Linux and macOS; on iPhone through a third-party client), or similar.
- The live database never lives in the sync folder (§8.1); only the logs do.
- A direct Wi-Fi transport (local discovery and pairing, no cloud) can be added later over the same log format.

**Required from Phase 0:**

- Every node, page and star has a random, global ID (§7.2).
- Each installation has a `device_id`, stored in `meta`.
- Every change set carries a hybrid logical clock (HLC) timestamp and its device ID (§8.2).
- Operations are self-contained: `after` holds complete field values, not diffs, so they apply correctly in any order.
- Fractional positions include a device-specific tie-breaker, so concurrent inserts at the same spot never collide.
- Deletes stay soft; undo and revert create new operations and never rewrite history. Only sealed operations are emitted to the log (§8.2).
- A node's parent and position form one **location** field, so a move is never split across two fields by a merge.
- Deleted state is one field too: a delete and a concurrent restore resolve by latest HLC like any other field.

**Merge rules (Phase 5):**

- Each field (text, page, completion, star, location) takes the value with the latest HLC; ties break by device ID.
- Moves use a proven tree-move algorithm (Kleppmann et al., "A highly-available move operation for replicated trees"): concurrent moves are applied in HLC order, undoing and redoing later moves as needed, so every device skips the same cycle-creating move and all devices converge. Skipped moves are reported.
- A node whose parent no longer exists goes under its nearest surviving ancestor, else the root (the same rule as Trash restore, §8.3), never lost.
- An edit to a node that another device deleted keeps the node deleted, with the edit, so it can be restored from the Trash.
- Page text merges whole-page by latest HLC at first; a line-level merge is a later improvement.

**Housekeeping:**

- Logs are JSON Lines with a versioned, documented format, so a future iPhone app can read and write them with its own SQLite store.
- Each device writes an acknowledgment watermark; logs are compacted once every known device has merged past a point.
- Merged remote changes appear in history with source `sync:<device>`, and a Sync view lists recent ones by device.
- **Device identity.** The `device_id` is also written to a small `device` file in the data directory. If the database's ID doesn't match it (the database was copied from another machine), the app generates a new `device_id` before writing anything.
- **Retired devices** would block compaction forever; `app sync forget-device <id>` removes one from the watermark calculation.
- **Snapshot restore with sync on** applies the difference between the current state and the snapshot as one change set (source `restore`), so other devices receive it and it can be reverted. Swapping the file is reserved for corruption recovery, after which the device takes a new `device_id` and merges every log again.
- Compaction, `app history purge` and Trash purges are local and never touch the logs; Forget node is the only erasure that syncs (§8.4).
- Convergence tests run with skewed simulated clocks until the HLC drift policy is decided (§12.3).

## 8. Safety, history and recovery

Three layers protect the data, each against a different failure: history against bad changes, the Trash against deletes, and snapshots against a damaged database file. Off-machine backup of the data folder remains the user's job, and the docs say so plainly.

### 8.1 Durability

- WAL mode, `synchronous=FULL`, one transaction per action (§7.3).
- `PRAGMA quick_check` runs in the background after the first frame, so it never delays launch; its duration on large databases is measured in Phase 0. A failure switches the session to read-only at once, writes any uncommitted typing to a recovery Markdown file in the data directory, and offers to restore the latest good snapshot.
- **Failed writes** (disk full, permissions): the edit stays in memory, the save indicator turns red, a banner explains, and the write is retried every few seconds. Typing continues. Quitting with unsaved edits writes them to a recovery Markdown file and says where.
- **Never sync the live database** with Syncthing, Dropbox or similar; it can corrupt. The app warns if `outline.db` sits inside a known sync folder. Sync snapshots or exports instead.

### 8.2 History

```sql
CREATE TABLE change_sets (
  id          TEXT PRIMARY KEY,  -- random, globally unique (sync-ready)
  device_id   TEXT NOT NULL,     -- installation that made the change (§7.6)
  hlc         TEXT NOT NULL,     -- hybrid logical clock timestamp, sortable
  at          TEXT NOT NULL,     -- wall-clock ISO timestamp, for display
  source      TEXT NOT NULL,     -- 'tui', 'cli', 'editor', 'import', 'restore', 'agent:<name>', 'sync:<device>'
  description TEXT,              -- e.g. 'Moved "Design notes"'
  undo_of     TEXT,              -- change set this one undoes (§5.6)
  cursor      TEXT               -- JSON: zoom, focused node, offset, selection before the change
);
CREATE INDEX change_sets_by_hlc ON change_sets(hlc);

CREATE TABLE ops (
  id            INTEGER PRIMARY KEY,
  change_set_id TEXT NOT NULL REFERENCES change_sets(id),
  node_id       TEXT NOT NULL,
  kind          TEXT NOT NULL,   -- insert, delete, restore, edit_text, edit_page, move,
                                 -- set_task, set_heading, set_complete, set_star, forget
  before        TEXT,            -- JSON: prior values of the changed fields
  after         TEXT             -- JSON: complete new values (self-contained, §7.6)
);
CREATE INDEX ops_by_node ON ops(node_id);
CREATE INDEX ops_by_change_set ON ops(change_set_id);
-- move carries one location field: {parent_id, position} (§7.6)
-- edit_page with before = null creates a page; with after = null deletes it
```

- **Current state is authoritative;** history is a record beside it, written in the same transaction. Nothing is rebuilt by replay.
- **Before and after values** make every operation reversible by applying its inverse.
- **Change sets** group the operations of one action: a subtree move, a CLI command, an editor round trip. They are the unit of undo, revert and "what did this agent do?".
- **Coalescing:** consecutive edits to one node's text form a single operation. While open, that operation's row is updated in place at each commit (about every second, §7.3). It is **sealed** by a 2-second pause, leaving the node, a structural action, or quitting. Sealed operations are never changed, and only sealed operations are written to the sync log.
- **Append-only:** history is never rewritten, except by explicit erasure: compaction, `app history purge` and Forget node (§8.4).
- **Revert** of an older change set applies its inverses to fields not changed since. Any skipped fields are reported, never overwritten.
- **History view** (Phase 2): browse a node's past versions and restore one, or list change sets by time and source.

### 8.3 Trash

- Deleting sets `deleted_at` and `deleted_by` (the deleting change set) on every node in the subtree; nothing is removed. The Trash view lists deletions, newest first.
- Restore brings back exactly one deletion: the nodes whose `deleted_by` matches. A child deleted separately earlier stays in the Trash.
- A restored subtree returns to its original position among its siblings when that position still exists, otherwise to the end of its parent. If the parent is gone too, it goes under the nearest surviving ancestor, else the root, and the toast says where. Sync uses the same rule (§7.6).
- Items older than 30 days are purged for real (configurable). "Empty Trash" purges immediately.
- Purging deletes the node rows with their pages, stars, tags and view state. History ops that mention them stay until history retention removes them; reverting one reports the node as gone. Purging is local: each device purges its own Trash by age, and purges are not synced.
- After a delete the cursor moves to the next sibling, else the previous one, else the parent. The toast names the bullet and counts its descendants.

### 8.4 Retention and erasure

- Full history is kept for 90 days (configurable). After that, compaction squashes consecutive text and page edits made from the TUI into one operation per node per day. Structural change sets and change sets from the CLI, the editor round trip and agents are never compacted, so they can still be reviewed and reverted as a unit.
- `app history purge --before <date>` removes older history.
- **Forget node** erases a node from `nodes`, history and the search index, and writes a `forget` tombstone operation so the node cannot return from another device's sync log. Snapshots still contain it until they rotate out, and the command says so.
- Expected growth is a few hundred KB per day of heavy use before compaction. This is an estimate to verify in Phase 0.

### 8.5 Snapshots

- Taken with SQLite's online backup API: hourly while changes are happening, and before every migration or restore.
- Retention: the last 48 hourly snapshots and 30 daily ones, within a size cap (default 1 GB) that drops the oldest daily snapshots first. `app snapshot list` shows each snapshot's size.
- `app snapshot list` lists them; `app snapshot restore <name>` restores one after snapshotting the current state first, so a restore is itself reversible.
- Restore never replaces the file under a live connection: it refuses while a TUI holds the lock (§8.6). When the TUI offers a restore itself (after a failed integrity check), it closes its connection first and reopens afterwards.

### 8.6 Locking

- One TUI per database, enforced with an exclusive `flock` on a lock file next to the database (`outline.db.lock`). It is not an SQLite lock, so CLI writers are never blocked by it, and the kernel releases it if the TUI dies.
- A second TUI opens read-only with a banner. It still follows changes through `PRAGMA data_version`; typing in it changes nothing and shows a hint.
- The CLI and `app capture` don't take the lock. They are ordinary writers with a 5-second busy timeout; past it they exit with a `busy` error code (§10.2). SQLite serializes the writes.

## 9. Implementation

### 9.1 Stack

| Area | Choice | Notes |
| --- | --- | --- |
| Language | Go | Simple, fast to compile, one static binary; well suited to agent-driven development |
| UI framework | Bubble Tea (v2 if released) | Elm-style update/view loop; v2 brings kitty keyboard protocol support. Confirm release status in Phase 0 |
| Styling and layout | Lip Gloss | Theme-driven styles; rendered rows cached per bullet |
| Components | Bubbles (text area, viewport, list) | Starting point for the page edit mode and palettes; the outline editor is custom |
| CLI | Cobra | Shared by quick capture, scripting and agents |
| Database | Pure-Go SQLite driver (`modernc.org/sqlite` or `ncruces/go-sqlite3`) | No C toolchain, so cross-builds for Linux and macOS stay simple. Confirm FTS5 support when choosing |
| Text | `rivo/uniseg`, `mattn/go-runewidth` | Grapheme-aware cursor; correct display width for CJK and emoji |
| Markdown | goldmark | Page rendering, with source positions for mode switching |
| Fuzzy matching | A Go fuzzy-finder library such as `sahilm/fuzzy` | Jump and command palettes |
| Import / export | `encoding/json`, `encoding/xml`, small list parser | JSON, OPML, Markdown; the parser also serves the editor round trip |
| Config | TOML (`BurntSushi/toml` or `pelletier/go-toml`) | Platform paths per §7.1 |
| Distribution | GoReleaser | Homebrew tap for macOS, AUR package for Arch and Omarchy |

### 9.2 Package architecture

Both front ends share one core; only the database is authoritative.

```text
┌──────────────────────────────┐      ┌──────────────────────────────┐
│ tui                          │      │ cli                          │
│ Bubble Tea + Lip Gloss       │      │ Cobra commands, JSON output  │
└──────────────┬───────────────┘      └───────────────┬──────────────┘
               │ commands                             │ same operations
┌──────────────▼──────────────────────────────────────▼──────────────┐
│ core                                                               │
│ In-memory tree, domain operations, undo via history, search        │
└─────────────────────────────────┬──────────────────────────────────┘
                                  │ load, commit, revert
┌─────────────────────────────────▼──────────────────────────────────┐
│ store                                                              │
│ SQLite access, history writes, snapshots, import and export        │
└──────────┬──────────────────────┬───────────────────────┬──────────┘
           │ transactions         │ backup API            │ on demand
┌──────────▼─────────┐  ┌─────────▼──────────┐  ┌─────────▼──────────┐
│ outline.db         │  │ Snapshots          │  │ Exports            │
│ Nodes, history,    │  │ Hourly and         │  │ Markdown, JSON,    │
│ search; the only   │  │ pre-migration      │  │ OPML, on demand    │
│ source of truth    │  │ restorable copies  │  │ or scheduled       │
└────────────────────┘  └────────────────────┘  └────────────────────┘
```

Four Go packages (`tui`, `cli`, `core`, `store`) and three kinds of on-disk data.

The TUI and CLI never touch the database directly; only `store` does. That keeps storage replaceable without rewriting either front end.

### 9.3 Runtime model

- The TUI follows Bubble Tea's model–update–view loop. Key presses, change notifications and timers are all messages; the model owns the in-memory tree.
- Slow work (database commits, snapshots, exports, the `data_version` poll) runs as Bubble Tea commands off the UI loop and reports back as messages. The view never waits on disk.
- A tick about every 500 ms checks `PRAGMA data_version` and reports changes made by other connections, such as the CLI.
- The view renders only the visible rows from a flattened list of visible nodes. Each row's styled output is cached and rebuilt only when that bullet, its focus state or the theme changes, which keeps Lip Gloss styling cost flat.
- Wrapping is computed by our own layout function, shared by rendering and cursor movement so they never disagree.

### 9.4 Performance targets

| Measure | Target | Workload |
| --- | --- | --- |
| Cold launch to first frame | < 100 ms | 10,000 nodes |
| Keypress to frame | < 16 ms | Same, typing in a deep node |
| One action committed (state + history) | < 5 ms | Same |
| In-place filter update | < 50 ms per keystroke | 100,000 nodes |
| CLI change visible in a running TUI | < 1 s | Any |

Targets are checked in CI with generated databases.

### 9.5 Testing

- **Export round trip:** database → JSON → database is identical, for generated and real outlines (property tests with `pgregory.net/rapid`).
- **History tests:** random sequences of operations followed by undo-all restore the original state; reverting any single change set leaves all other changes intact.
- **Crash injection:** kill the process at random points during editing; the database passes `PRAGMA integrity_check` and loses at most about one second of typing.
- **Migration tests:** a fixture database from every past schema version migrates cleanly, with a snapshot taken first.
- **Editor round trip:** scripted edits to the temp file (rename, reorder, delete, add, malformed lines) apply exactly as expected, or not at all.
- **Import tests:** real Workflowy OPML and Markdown exports from Logseq and Obsidian import without loss.
- **Parser fuzzing** with Go's built-in fuzzing for the inline-markup parser, the list importer and OPML.
- **UI tests:** Bubble Tea's `teatest` drives scripted key sequences and compares the rendered screen with golden files.
- **Interaction matrix:** scripted keystroke tests for every row of the Enter and Backspace tables (§5).
- **CLI tests:** every command's `--json` output is checked against golden files.
- **Sync convergence tests** (from Phase 0, before sync ships): two simulated devices apply random operations offline, including concurrent moves that would create cycles, exchange logs in any order, and end with identical databases (§7.6).

### 9.6 Built by agents

The app will be built entirely by AI coding agents, working from a phased implementation plan derived from this spec. The spec and repository follow rules that make that reliable:

- **Acceptance criteria per feature,** written as observable behavior ("Backspace on an empty bullet with children does nothing and shows a hint"), each mapped to at least one automated test.
- **No manual verification steps.** Every behavior is checkable by `go test`: UI through `teatest` golden screens, data through the CLI's `--json` output, safety through crash and round-trip tests. The human checkpoints are listed explicitly and are not acceptance criteria for agents: the visual "calm" reviews (§4.6), the hands-on keymap matrix in real terminals (§6.4), the comparison-set evaluation (§3), and the Phase 1 week of daily use (§11).
- **The CLI is the agents' test harness.** Agents create outlines, run commands and inspect results through the same CLI users get (§10.2), which is why it ships in Phase 1.
- **One package per task where possible.** The plan hands agents work along package boundaries (`core`, `store`, `tui`, `cli`) with stable interfaces between them.
- **Golden files are reviewed, not regenerated blindly.** A changed golden screen must be explained in the change description.
- **Stable section numbers** in this spec are referenced from the plan, tests and commit messages.

## 10. Obsidian-inspired features and the CLI

### 10.1 Obsidian-inspired features (Phases 2–3)

Obsidian's best ideas apply here to nodes rather than files.

- **Node links (Phase 2):** type `[[` to search and link to any bullet. Links are stored by node ID, so they survive moves and renames, and they display the target's current text. Alt+Enter on a link (or a click) zooms to its target. Markdown export writes Obsidian-style links.
- **Backlinks (Phase 3):** an on-demand floating overlay listing every bullet that links to the current node, each with its breadcrumb.
- **Unlinked mentions (Phase 3):** bullets containing the current node's text that don't link to it yet, with a one-key "link it" action.
- **Peek (Phase 3):** a popup previewing a linked node's subtree without zooming away. The terminal version of Obsidian's hover preview.
- **Properties (Phase 3):** optional `key:: value` lines at the top of a node's page (`status:: waiting`, `due:: 2026-11-01`), shown as quiet chips and indexed for search (`status:waiting`).
- **Saved queries (Phase 3):** a bullet whose page starts with `query:: #bug is:open` shows live search results as its virtual children. Results are read-only references; Enter zooms to the real node.
- **Templates (Phase 3):** any subtree tagged `#template` can be instantiated under the current node with one command, from the command palette.
- **Today node (Phase 3, optional):** a command that zooms to today's dated bullet under a Journal node, creating it if needed. Off by default; there is no mandatory journal.
- Graph view and plugins remain non-goals.

### 10.2 CLI and agent interface

A robust CLI ships in **Phase 1**, for users, scripts and AI agents alike. It is a thin layer over the same `core` operations as the TUI, and it doubles as the build agents' test harness (§9.6). Phase 4 adds agent conveniences on top.

```bash
app add "Fix tooltip positioning" --under "Projects/ControllerCraft"
app show <id|path> [--depth N] --json
app tree [--root <id>] --json
app search "tooltip #bug is:open" --json
app edit <id> --text "New text" [--if-version N]
app move <id> --under <id> [--after <id>]
app complete <id>        # and: reopen, star, unstar
app delete <id>          # to Trash
app trash list|restore <id>|empty
app page get <id>        # Markdown to stdout
app page set <id> --file page.md   # or from stdin
app apply --file ops.json          # many operations as one change set
app history list [--source agent:claude] --json
app history purge --before <date>
app revert <change-set-id>
app export --format markdown|json|opml|text --out <dir> [--ids] [--history]
app import --format json|opml|markdown <file> [--under <id>] [--replace]
app snapshot list|restore <name>
app paths
```

- Core verbs on nodes (`add`, `show`, `edit`, `move`, `complete`, `delete`) are top-level; everything else is grouped by noun (`trash`, `page`, `history`, `snapshot`).
- Nodes are addressed by ID, or by breadcrumb path for convenience. Path lookups return the ID for later calls.
- Every command supports `--json`, with a stable, versioned schema, consistent error codes and exit statuses. Human-readable output is the default.
- Every write is one change set tagged with its source: `cli` by default, `--source agent:<name>` for agents, so their changes can be reviewed and reverted as a unit.
- `app apply` lets an agent submit a batch of operations atomically; either all apply or none do.
- Writes are safe while the TUI is open; the TUI picks them up within a second (§7.3).
- The database is always explicit (`--db`, environment variable, or config default).
- **Phase 4:** an MCP server built on these commands, a shipped usage guide for agents, and richer query output.

## 11. Staged delivery plan

Phase 1 delivers the core of the Workflowy clone (every Phase 1 *Must* row), usable daily on your own imported outline. Phase 2 completes Workflowy parity, Phase 3 adds the Obsidian-inspired features, and agent integration and sync come last.

Six phases, each with an exit gate:

| Phase | Scope | Exit gate |
|---|---|---|
| **0 · Storage and editing spike** | Schema, sync-ready history, JSON round trip; wrap layout, keymap matrix, Omarchy palette | Export and import are identical; a crash loses at most 1 s of typing |
| **1 · Minimum lovable Workflowy clone** | Editing rules, block selection, pages, formatting; Omarchy theme, palettes, CLI, Workflowy import | A week of daily use on your own outline; side-by-side calm review passes |
| **2 · Full Workflowy parity** | Tags, node links, search operators, stars; jump palette, history, mouse, page edit mode | Every *Must* row of the parity checklist works |
| **3 · Obsidian-inspired features** | Backlinks, unlinked mentions, peek, properties; saved queries, templates, optional Today node | Links survive moves and renames and export cleanly |
| **4 · Agent integration** | MCP server, agent usage guide, richer queries; optional Vim keymap for the outline | Every agent change is reviewable and revertible |
| **5 · Multi-device sync** | Per-device logs, HLC merge, compaction; any-folder sync (iCloud Drive, Syncthing) | Two machines edit offline and converge with nothing lost |

Each phase starts only after the previous gate passes. Sync-ready data design starts in Phase 0 even though sync itself ships in Phase 5 (§7.6).

**Separate follow-up project:** an iPhone app that syncs through the same per-device log format. It is out of scope here; this project only guarantees the format is documented and stable enough to build it.

Future and exploratory, with no phase: direct Wi-Fi sync, collaboration, graph view, node mirrors, plugins.

## 12. Non-goals, decisions and next steps

### 12.1 Non-goals

- No editing of the data outside the app. Exports are copies; the `$EDITOR` round trip is the supported way to edit with another tool.
- No live sync through Git, Syncthing or cloud folders.
- No `@person` mentions and no board (kanban) view in V1.
- No mandatory daily journal or page model.
- No accounts, subscription, cloud backend or database server.
- No rich-text or WYSIWYG editing; formatting is Markdown-style markup, rendered with concealment.
- No real-time multi-writer collaboration.
- No full Obsidian replacement, graph view or plugin system in V1.
- No Windows support.
- No iPhone or Android app in this project; a phone app is a separate follow-up (§11).

### 12.2 Resolved decisions

| # | Decision | Resolution | Section |
| --- | --- | --- | --- |
| 1 | Product Goal | Workflowy clone first; Obsidian-inspired features as the bonus | §1, §3 |
| 2 | Source of truth | One SQLite file; exports and snapshots are copies | §7 |
| 3 | Node identity | Random, permanent IDs in the database | §7.2 |
| 4 | Sibling order | Fractional index positions | §7.2 |
| 5 | History | Change sets with before/after operations, coalesced typing | §8.2 |
| 6 | Deletion | Soft delete into a Trash, purged after 30 days | §8.3 |
| 7 | Backup | Hourly and pre-migration snapshots; export on demand or on schedule | §8.5, §7.5 |
| 8 | Editor ergonomics | Modeless + Esc block selection; Enter/Backspace tables | §5 |
| 9 | Formatting | Markdown-style markers, concealed when not editing | §5.8 |
| 10 | Text color | `<red>text</red>` with theme-mapped color names; `==highlight==` | §5.8 |
| 11 | Tags | `#tag` with nested `/` tags; no `@person` | §5.9 |
| 12 | Theme | ANSI palette always; Omarchy palette and live updates when detected | §4.4 |
| 13 | Tasks | `is_task` plus `completed_at`; exported as checkboxes | §7.2 |
| 14 | Search | FTS5; filter in P1, jump palette and operators in P2 | §6.2 |
| 15 | External editing | Temp-file round trip only | §6.3 |
| 16 | Configuration | XDG paths, TOML, explicit `--db` | §7.1 |
| 17 | Long-form content | Bullets stay one line; an optional Markdown page per bullet, opened with page zoom. Replaces notes | §5.7 |
| 18 | Language and UI stack | Go with Bubble Tea, Lip Gloss and Bubbles; pure-Go SQLite | §9.1 |
| 19 | Platforms | Linux (Omarchy first) and macOS; no Windows | §6.4, §7.1 |
| 20 | Sync | Sync-ready history from Phase 0; per-device logs in a shared folder in Phase 5; no server | §7.6 |
| 21 | CLI | Full CLI with JSON output in Phase 1; MCP server in Phase 4 | §10.2 |
| 22 | Builders | Built entirely by AI agents from a phased plan; every behavior has an automated test | §9.6 |

### 12.3 Still open

- **Project name and branding.** Where the name may appear is settled (§1); the name itself is not.
- **Mirrors.** One bullet shown live in two places (for example a task in its project and on a weekly list). Decide after Phase 2 whether daily use calls for it. Mirrors turn the tree into a graph and affect export, Trash, cycle checks and sync, so they need their own design; random IDs and self-contained ops don't rule them out.
- **Color palette size.** Eight named colors is the starting set; confirm against Omarchy themes that each stays legible on light and dark backgrounds.
- **History retention defaults.** 90 days full, then daily, is a starting point; measure real growth in Phase 0.
- **Default Markdown export layout.** One file, or one file per top-level node.
- **Sync details.** The direction is settled (§7.6). Open details: HLC drift handling, compaction thresholds, and whether page text later merges line by line.

### 12.4 Immediate next steps

1. **Schema spike:** nodes, pages, tags, sync-ready history and coalescing in Go with the chosen SQLite driver. Simulate a week of heavy editing and measure database growth.
2. **Safety tests:** the JSON export round trip, crash injection during typing, and snapshot restore on a clean machine.
3. **Wrap-layout and rendering prototype:** one Bubble Tea outline view with grapheme-aware cursor movement, concealed inline formatting, and the centered reading column.
4. **Omarchy theme reader:** read the current theme's palette, follow live theme switches, and verify file locations against the current Omarchy release.
5. **Reference mockups:** the five screens listed in §4.6, reviewed side by side with Workflowy and with lazygit and btop under three Omarchy themes, one of them light.
6. **Keymap matrix:** record which bindings arrive in each terminal, bare and in tmux, before finalizing §6.4, including how Option+Left/Right arrive on macOS.
7. **Workflowy import:** run a real Workflowy OPML export through the importer and list anything lost.
8. **Stack checks:** confirm Bubble Tea v2 status and its keyboard-protocol support, and choose the pure-Go SQLite driver after verifying FTS5.
9. **Sync-readiness test:** two simulated devices exchange operation logs in random order and converge (§9.5), before any sync UI exists.
10. **Implementation plan:** turn this spec into the phased, agent-ready plan, with acceptance criteria per feature (§9.6).
