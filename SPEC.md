# Terminal Infinite Outliner — Project Specification v3

Oct 15, 2026 · @Thor

## 1. Overview

We are building a local-first, terminal-native infinite outliner in Go, inspired by Workflowy, for Linux and macOS. It is one continuous tree of bullets that you can capture into, indent, reorder, collapse and zoom. The outline is stored in a single SQLite file with full edit history and automatic snapshots, and can be exported to Markdown, JSON or OPML at any time.

**Goal:** a faithful Workflowy experience in a beautiful terminal UI that matches the Omarchy desktop theme. Obsidian-inspired knowledge features (links, backlinks, properties, saved queries) are the bonus.

**Pitch:** An infinite outline for your terminal: one tree, unlimited depth, stored in one open, durable file you can always export.

**Status:** Concept / pre-implementation. Working name TBD; the name must not appear in architecture or file formats.

## 2. Product principles

1. **One tree, no mandatory pages.** Any node can have children and can be zoomed into.
2. **Thought-first editing.** Capturing and rearranging bullets needs no mode switch. Commands that need single keys use a lightweight block-selection state, not Vim modes.
3. **Your data is never locked in.** One SQLite file with a plain, documented schema, readable by standard tools. Export to Markdown, JSON or OPML at any time; a JSON export restores everything.
4. **Nothing is lost silently.** Every change is recorded in history, deletes go to the Trash, and snapshots protect against a damaged file.
5. **Every change is attributable and reversible.** History records whether a change came from the TUI, the CLI, the editor round trip or an agent, and any change set can be reverted.
6. **Local-first and private.** No account, server, cloud dependency or network connection.
7. **Terminal-native.** Match the system theme (Omarchy first), read like a calm page rather than a dashboard, work inside tmux, degrade gracefully in legacy terminals, and keep every action available from the keyboard.
8. **Composable.** Users, scripts and agents work through a robust CLI with JSON output, never by touching the database directly.
9. **Simple core first.** The outlining experience comes before knowledge-management features.

## 3. Users, scenarios and comparison set

**Primary users** are Workflowy-style outliners who live in the terminal and value plaintext, Git history and local ownership. Later, users who want scripts or coding agents to read and update their outline.

**Core scenarios:**

1. **Session.** Launch from a shell and type immediately. Enter adds bullets, Tab nests them. Collapse a project, zoom into a task, filter for a word, quit.
2. **Quick capture.** From any shell or a Hyprland keybind: `app add "Call the dentist"` appends to an Inbox node without opening the TUI.
3. **Edit in `$EDITOR`.** Press a key to open the current subtree as Markdown in Neovim. Save and quit; the changes apply as one undoable step.
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

- Exactly one logical root per workspace. Every bullet is a node with editable text, ordered children, and an in-memory identity.
- No distinction between bullet, task or project. Any bullet can also carry a Markdown page (§5.7). Completion is an attribute any node can have.
- **Zoom** makes any node the temporary root. The zoomed node is shown as a heading, with only its descendants below it.
- **Breadcrumbs** show the path to the zoom target. Long paths are truncated in the middle: `Root / … / Development / Tooltips`.
- Collapse/expand is independent of zoom and persists across sessions (§7.2).
- No artificial depth limit.

### 4.2 Corrected mockup (zoomed into Projects)

```text
Root / Projects                                         ~/knowledge  ●

Projects

▾ ControllerCraft
│ ▾ Development
│ │ • Inventory navigation        (completed: dimmed, struck)
│ │ • Improve controller mappings
│ │ • Fix tooltip positioning
│ ▸ Design notes
▸ Lighthouse

Enter new · Tab indent · Esc select · Ctrl+K jump · ? help
```

### 4.3 Visual language

| State | Glyph | ASCII fallback |
| --- | --- | --- |
| Collapsed, has children | `▸` | `>` |
| Expanded, has children | `▾` | `v` |
| Leaf | `•` | `-` |
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

- **Header:** breadcrumb left; database name and save indicator right.
- **Save indicator:** nothing when saved; `●` while a coalesced edit is not yet committed (at most about 1 second); a red label if a write fails.
- **Footer:** contextual key hints, dismissible via config. `?` opens a full help overlay.
- **Toasts:** one line, auto-dismissing, for undoable actions: `Moved "Design notes" and 214 children to Trash · Ctrl+Z`.
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

- `?` in block selection, or F1 anywhere, shows every command with its key, as lazygit, Impala and BlueTUI do.
- Esc always backs out one level: text editing → block selection → clear filter → zoom out.
- Block selection accepts Vim motions (`h` `j` `k` `l`, `gg`, `G`), matching Omarchy's Neovim-first setup.
- With a side pane open (backlinks, peek), Ctrl+W then a direction moves focus, Vim-style. Tab keeps its outline meaning.
- Ctrl+Q quits from anywhere; there is never anything to save.

**Native app integration**

- Ships a `.desktop` entry and its own window class, so it launches from the app launcher like Omarchy's other terminal apps and can be targeted by Hyprland window rules.
- **Capture window:** `app capture` opens a small floating window (via a window rule on its class). Type one line, press Enter, and it lands in the Inbox and the window closes. The docs suggest a Super binding, checked against Omarchy's defaults.
- Desktop notifications only for things that need attention, such as a failed snapshot, through the standard notification daemon.

**Restraint**

- No splash screen, toolbars or busy status bar. Breadcrumbs, the save dot and one line of hints are all the chrome.
- The empty first-run screen may show a small ASCII wordmark, in the spirit of Omarchy's ASCII branding, and then gets out of the way.
- Instant response is part of the look: every keypress lands in the next frame (§9.4).

**Reference sketches** (the window border is Hyprland's, not the app's):

```text
  Projects › ControllerCraft

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
- **Block selection.** Esc selects the current bullet as a whole. Up/Down move the selection; Shift+Up/Down extend it across siblings. Single keys act on the selection. Any printable key that isn't a command, or Enter, returns to text editing.
- Optional Vim-style navigation is a later setting, never a requirement.

### 5.2 Enter

| Cursor position | Result |
| --- | --- |
| Start of a non-empty bullet | Insert an empty sibling above; cursor stays on the current text |
| Middle of text | Split: text after the cursor becomes a new sibling below; children stay with the first half |
| End of an expanded bullet with children | Create a new first child |
| End of a collapsed bullet, or a leaf | Create the next sibling |
| Empty bullet that is not at the top level of the current zoom | Outdent it (Workflowy behavior) |

### 5.3 Backspace at the start of a bullet

| Situation | Result |
| --- | --- |
| Bullet is empty, no children | Delete it; cursor to the end of the previous visible bullet |
| Bullet is empty, has children | No-op; status line hints to use Esc then delete |
| Bullet has text, no children | Merge text into the end of the previous visible bullet |
| Bullet has text and children, previous bullet has no children | Merge text; children are adopted by the previous bullet |
| Bullet has text and children, previous bullet has children | No-op with a hint. Merging would interleave two child lists unpredictably |

Delete at the end of a bullet mirrors these rules forward.

### 5.4 Structure

- **Tab / Shift+Tab** indent or outdent the current bullet (or selection) with its descendants. Indent is a no-op on the first sibling.
- **Move up/down** among siblings moves whole subtrees. Moving past the first or last sibling moves to the adjacent parent's sibling list, as Workflowy does.
- **Delete a subtree** (Esc then delete key) never asks for confirmation. It shows an undo toast with the descendant count.
- **Complete / reopen** toggles a node's completion (storage rule: §7.2). Completed nodes stay in place; a toggle hides them.
- **Move to…** opens the jump palette to pick a new parent. This is how subtrees cross long distances.

### 5.5 Paste and clipboard

- Bracketed paste is enabled. Single-line paste inserts text.
- Multi-line paste is parsed as Markdown into a subtree inserted as siblings after the cursor. Plain lines become bullets; indentation becomes nesting.
- Copy in block selection writes indented Markdown to the system clipboard via OSC 52, with `wl-copy` as a fallback.

### 5.6 Undo and redo

- Undo is driven by history (§8.2). Each change set is one undo step.
- Typing coalesces into one step per node, closed by a 2-second pause, leaving the node, or any structural action.
- Undo restores cursor and selection position.
- **Undo survives restarts.** Ctrl+Z after reopening the app continues where you left off.
- Ctrl+Z only undoes your own TUI change sets. Changes from the CLI, the editor round trip or an agent are reverted from the History view instead, so you never undo something you didn't do.

### 5.7 Pages

- **Every bullet stays one line.** Long-form writing lives on an optional **page** attached to the bullet: a full Markdown document with headings, lists, code blocks, quotes and tables, as long as you like.
- **Page zoom:** Alt+Down (or Esc then `p`) opens the bullet's page in view mode; on a bullet without one, it creates an empty page and opens edit mode. Esc from view mode, or Alt+Left, returns to the outline with the cursor where it was. Normal zoom (Alt+Right) still shows only the bullet's children.
- **Title:** the bullet's text is the page title, shown as its heading. Editing either one changes both.
- **Indicator and preview:** a dim page marker follows the bullet's text. An optional setting, off by default, shows the page's first non-empty line dimmed beneath the focused bullet (§4.6).
- **Two modes, no live preview.** Editing inside rendered output is the hardest and buggiest part of a Markdown editor, so pages separate reading from writing:
  - **View mode** renders the Markdown read-only: headings, lists, quotes, tables, and code blocks syntax-highlighted with the theme palette. It scrolls, and Tab moves between links; Enter on a `[[link]]` zooms to its target.
  - **Edit mode** is a plain multi-line text editor showing the Markdown exactly as typed. Syntax highlighting keeps it pleasant without concealment: headings bold in the accent color, markers dimmed, code blocks tinted, links underlined. It shares the outline's wrap-layout and cursor engine.
- **Switching:** `e` or Enter in view mode enters edit mode; Esc in edit mode returns to view mode. The parser's source ranges keep your place: switching scrolls to the same block and puts the cursor on it.
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
| Bold | `**text**` | Ctrl+B (Alt+B inside tmux, whose default prefix is Ctrl+B) |
| Italic | `*text*` | Ctrl+I (Alt+I in legacy terminals, where Ctrl+I equals Tab) |
| Strikethrough | `~~text~~` | — |
| Inline code | `` `text` `` | — |
| Link (URL) | `[label](https://…)` | — |
| Highlight | `==text==` | — |
| Text color | `<red>text</red>` | Alt+C opens a color picker |

- **Colors:** `red`, `orange`, `yellow`, `green`, `cyan`, `blue`, `purple`, `gray`. Each maps to the theme palette (ANSI first, Omarchy truecolor when available), so colored text follows theme changes.
- `<red>` and `==highlight==` were chosen because they are readable when concealment is off and render in Obsidian after export. Unknown tags are left as plain text.
- Shortcuts apply to the text selection (Shift+Left/Right in text editing) or, in block selection, to the whole bullet.
- A **color the whole bullet** command applies a color to all selected bullets at once.
- Markdown export keeps the markers; colors export as `<span style="color: …">` so Obsidian renders them; plain-text export strips all markup.

### 5.9 Tags

- `#tag` anywhere in a bullet or page. Letters, digits, `-`, `_` and `/` are allowed; `/` makes nested tags (`#project/lighthouse`), as in Obsidian.
- Tags render as colored chips. Each tag gets a stable color from the palette, overridable in config.
- Enter or click on a tag sets the in-place filter to that tag. Typing `#` offers completion from existing tags.
- A tag list in the command palette shows every tag with its count.

## 6. Navigation, search, integration and input

### 6.1 Navigation

- Collapse/expand one node, or all descendants (expand-all with a modifier).
- Zoom in on the current node; zoom out one level; jump to root.
- The cursor position is remembered per zoom level. Zooming out returns exactly where you were.
- Up/Down move through wrapped visual lines, not just logical bullets. Home/End are per visual line, then per bullet on a second press.

### 6.2 Search

- **In-place filter (Phase 1).** `/` filters the tree as you type: matching nodes stay visible with their ancestors, matches highlighted, everything else hidden. This includes collapsed branches and page text. Esc clears it and keeps the cursor on the selected match.
- **Jump palette (Phase 2).** Ctrl+K opens a fuzzy finder over all node text, showing each result's breadcrumb, with starred locations listed first. Enter zooms to the result's parent with the result selected. The same palette backs "Move to…".
- **Search operators (Phase 2),** Workflowy-style, in both the filter and the palette:
  - `is:complete`, `is:open`, `is:task`, `has:page`, `is:starred`
  - `#tag` (includes nested tags), `changed:7d`, `created:2026-10`
  - `"exact phrase"`, `-word` to exclude, terms combined with AND by default, `OR` between terms
- Case-insensitive substring matching by default. Saved queries arrive in Phase 3 (§10.1).

### 6.3 External editor and quick capture

- **Edit in `$EDITOR`** (Phase 1): writes the current subtree to a temp Markdown file, suspends the TUI and opens `$EDITOR`. Each line carries its node's ID as a `^id` suffix so edits map back.
- On exit, the file is parsed and diffed against the subtree. Changed lines update nodes; new lines become new nodes; removed lines move to the Trash. Everything applies as one change set with source `editor`, undoable with Ctrl+Z.
- If the file can't be parsed, nothing is applied: the temp file is kept and a banner offers to reopen it.
- **Quick capture** (Phase 1): `app add "text" [--under <path-or-id>]` appends to the Inbox node by default, in its own transaction. A running TUI notices the change through SQLite's `PRAGMA data_version` and refreshes.

### 6.4 Keymap and terminal support

Some key combinations are only distinguishable when the terminal supports the kitty keyboard protocol. The app enables it through Crossterm when available. Every binding has a fallback that works in legacy terminals and inside tmux.

| Action | Primary | Legacy / tmux fallback |
| --- | --- | --- |
| New bullet | Enter | Enter |
| Indent / outdent | Tab / Shift+Tab | Tab / Shift+Tab |
| Block selection | Esc | Esc |
| Move subtree up/down | Ctrl+Shift+Up/Down | Alt+Up/Down, or Esc then K/J |
| Collapse / expand | Ctrl+Up / Ctrl+Down | Esc then h / l |
| Zoom in / out | Alt+Right / Alt+Left | Esc then Enter / Backspace |
| Toggle complete | Ctrl+Enter | Esc then x |
| Filter / jump | / and Ctrl+K | Same |
| Open in editor | Ctrl+E | Esc then e |
| Undo / redo | Ctrl+Z / Ctrl+Shift+Z | Ctrl+Z / Ctrl+Y |

- Keymap is configurable in TOML.
- A test matrix covers Ghostty, Kitty, Alacritty, foot and WezTerm, each bare and inside tmux, recording which bindings arrive intact.
- Bindings that a desktop compositor commonly intercepts (Super combinations under Hyprland) are never used.

**macOS keyboard:** by default, macOS terminals send special characters (å, ∫) for Option+key instead of Alt. Every Alt binding therefore has an Esc-based fallback (§6.4, §6.6), and on first run on macOS the app shows a one-line hint explaining how to set Option to act as Meta (Alt) in the user's terminal. Apple's Terminal.app is treated as ANSI-only; Ghostty, kitty, WezTerm and iTerm2 are fully supported.

### 6.5 Mouse

- Phase 2: scroll, click to place the cursor, click a glyph to toggle collapse.
- Mouse capture disables the terminal's native text selection. The app documents Shift+drag (which most terminals use to bypass capture) and offers a config toggle to turn capture off.
- Drag-to-move nodes is exploratory; keyboard moves remain complete without it.

### 6.6 Palettes, stars and small commands

- **Command palette (Ctrl+P, Phase 1):** every command searchable by name, showing its key binding, recently used first. It is the discoverability layer for everything in this spec, so terminal key limits never hide a feature.
- **Stars (Phase 2):** "Toggle star" (Alt+S, or Esc then `s`) stars the current zoom target. Stars appear first in the jump palette and in a Stars view; `is:starred` finds them.
- **Small commands (Phase 1):**
  - Expand all / collapse all under the current node
  - Expand to level 1–9 (Alt+1 … Alt+9)
  - Duplicate subtree (Ctrl+D)
  - Show / hide completed (Ctrl+O)
  - Move to… (via the jump palette)
  - Copy link to node
  - Sort children (alphabetical, completed last) — Phase 2

Additional bindings, extending the §6.4 table:

| Action | Primary | Legacy / tmux fallback |
| --- | --- | --- |
| Open page (page zoom) | Alt+Down | Esc then `p` |
| Command palette | Ctrl+P | Same |
| Color picker | Alt+C | Esc then `c` |
| Toggle star | Alt+S | Esc then `s` |
| Page: edit mode / back to view | e or Enter / Esc | Same |
| Page: open in $EDITOR | E (e in Phase 1) | Same |
| Page: split view | s | Same |

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
-- schema_version, app name, a one-paragraph description of the schema

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
  deleted_at   TEXT                  -- soft delete; non-null = in Trash
);

CREATE TABLE pages (                 -- at most one page per bullet (§5.7)
  node_id    TEXT PRIMARY KEY REFERENCES nodes(id),
  markdown   TEXT NOT NULL,          -- full Markdown document
  updated_at TEXT NOT NULL
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

CREATE TABLE view_state (
  node_id   TEXT PRIMARY KEY REFERENCES nodes(id),
  collapsed INTEGER NOT NULL DEFAULT 0
);

-- change_sets and ops: see §8.2
-- search_fts: FTS5 index over nodes.text and pages.markdown, maintained by triggers
```

- **IDs** are random and permanent. They are what the CLI, links and the editor round trip use.
- **Ordering** uses fractional index strings, so moving or reordering a node rewrites one row.
- **Completion:** `is_task` marks an open task (`[ ]`); `completed_at` marks completion for any node. Reopening clears `completed_at` and keeps `is_task`, so a plain bullet never turns into a task by accident.
- **View state** (collapse) lives in its own table and is excluded from history: folding never creates undo steps.
- **Search** uses an FTS5 index kept in sync by triggers.
- The schema stays plain enough that anyone opening the file in a SQLite browser understands it in a minute.

* **Formatting** is stored as the markup the user typed, not as separate style rows. The text stays readable in any SQLite browser and exports without conversion; rendering parses it on display.
* **Tags** are kept in `node_tags`, rebuilt from the bullet text and page in the same transaction as every edit, so tag filters and the tag list are indexed queries.
* **Stars** are user data and part of history; collapse state is not.

### 7.3 Transactions and concurrency

- Every user action is one transaction that updates `nodes` and writes its history together (§8.2).
- Typing is committed as its coalesced operation at most about every second, so a crash loses at most the last second of typing.
- WAL mode with `synchronous=FULL`. The extra cost is negligible at this write rate.
- The TUI keeps an in-memory tree for rendering; all writes go through `core` to `store`.
- The CLI opens its own connection with a busy timeout. The TUI polls `PRAGMA data_version` about every 500 ms and reloads changed nodes.

### 7.4 Migrations

- Schema versions are numbered and migrations run forward only.
- A snapshot is taken before every migration.
- A database with a newer schema than the app knows opens read-only with a banner, never migrated backwards.

### 7.5 Export and import

| Format | Export | Import |
| --- | --- | --- |
| JSON | Full fidelity: IDs, order, task and completion state, timestamps; history optional | Full restore into an empty database |
| Markdown | Nested lists with `[ ]` and `[x]`; one file, or one file per top-level node | New nodes under a chosen parent |
| OPML | Standard outliner exchange format | New nodes under a chosen parent (for migrating from Workflowy and other outliners) |

- Every exported file starts with a one-line header naming the app and the export time.
- Exports are written atomically (temp file, then rename).
- **Scheduled export** is an optional setting: Markdown or JSON, daily or on quit, into a folder you choose (for example a Git repo), keeping the last N exports.
- CI checks that database → JSON → database produces an identical outline.

**Pages in exports:** Markdown export writes each page as its own `.md` file in a `pages/` folder, named from the bullet text plus its ID, and the outline links to it. The result reads much like an Obsidian vault. JSON export includes pages in full; OPML puts them in `_note`.

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
- Deletes stay soft; undo and revert create new operations and never rewrite history.

**Merge rules (Phase 5):**

- Each field (text, page, completion, star, position) takes the value with the latest HLC; ties break by device ID.
- A move that would make a node its own ancestor is skipped and reported.
- A node whose parent no longer exists is placed under a "Recovered" node at the root, never lost.
- An edit to a node that another device deleted keeps the node deleted, with the edit, so it can be restored from the Trash.
- Page text merges whole-page by latest HLC at first; a line-level merge is a later improvement.

**Housekeeping:**

- Logs are JSON Lines with a versioned, documented format, so a future iPhone app can read and write them with its own SQLite store.
- Each device writes an acknowledgment watermark; logs are compacted once every known device has merged past a point.
- Merged remote changes appear in history with source `sync:<device>`, and a Sync view lists recent ones by device.

## 8. Safety, history and recovery

Three layers protect the data, each against a different failure: history against bad changes, the Trash against deletes, and snapshots against a damaged database file. Off-machine backup of the data folder remains the user's job, and the docs say so plainly.

### 8.1 Durability

- WAL mode, `synchronous=FULL`, one transaction per action (§7.3).
- `PRAGMA quick_check` runs in the background at launch. A failure opens the database read-only and offers to restore the latest good snapshot.
- **Never sync the live database** with Syncthing, Dropbox or similar; it can corrupt. The app warns if `outline.db` sits inside a known sync folder. Sync snapshots or exports instead.

### 8.2 History

```sql
CREATE TABLE change_sets (
  id          TEXT PRIMARY KEY,  -- random, globally unique (sync-ready)
  device_id   TEXT NOT NULL,     -- installation that made the change (§7.6)
  hlc         TEXT NOT NULL,     -- hybrid logical clock timestamp, sortable
  at          TEXT NOT NULL,     -- wall-clock ISO timestamp, for display
  source      TEXT NOT NULL,     -- 'tui', 'cli', 'editor', 'import', 'agent:<name>', 'sync:<device>'
  description TEXT               -- e.g. 'Moved "Design notes"'
);

CREATE TABLE ops (
  id            INTEGER PRIMARY KEY,
  change_set_id TEXT NOT NULL REFERENCES change_sets(id),
  node_id       TEXT NOT NULL,
  kind          TEXT NOT NULL,   -- insert, delete, edit_text, edit_page, move, set_task, set_complete, set_star
  before        TEXT,            -- JSON: prior values of the changed fields
  after         TEXT             -- JSON: complete new values (self-contained, §7.6)
);
```

- **Current state is authoritative;** history is a record beside it, written in the same transaction. Nothing is rebuilt by replay.
- **Before and after values** make every operation reversible by applying its inverse.
- **Change sets** group the operations of one action: a subtree move, a CLI command, an editor round trip. They are the unit of undo, revert and "what did this agent do?".
- **Coalescing:** all edits to one node's text form a single operation, closed by a 2-second pause, leaving the node, or a structural action.
- **Revert** of an older change set applies its inverses to fields not changed since. Any skipped fields are reported, never overwritten.
- **History view** (Phase 2): browse a node's past versions and restore one, or list change sets by time and source.

### 8.3 Trash

- Deleting sets `deleted_at` on the subtree; nothing is removed. The Trash view lists deleted subtrees, newest first.
- Restore puts a subtree back under its original parent, or under the root if that parent is gone too.
- Items older than 30 days are purged for real (configurable). "Empty Trash" purges immediately.

### 8.4 Retention and erasure

- Full history is kept for 90 days, then compacted to one change set per node per day (configurable).
- `app history purge --before <date>` removes older history.
- **Forget node** erases a node from `nodes`, history and the search index. Snapshots still contain it until they rotate out, and the command says so.
- Expected growth is a few hundred KB per day of heavy use before compaction. This is an estimate to verify in Phase 0.

### 8.5 Snapshots

- Taken with SQLite's online backup API: hourly while changes are happening, and before every migration or restore.
- Retention: the last 48 hourly snapshots and 30 daily ones.
- `app snapshots` lists them; `app restore <snapshot>` restores one after snapshotting the current state first, so a restore is itself reversible.

### 8.6 Locking

- One TUI per database, enforced with a lock held by the TUI's connection. A second TUI opens read-only with a banner.
- CLI commands run alongside the TUI safely; SQLite serializes the writes.

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

&#91;embedded content: package architecture · 4 Go packages, 3 kinds of on-disk data\]

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
- **Sync convergence tests** (from Phase 0, before sync ships): two simulated devices apply random operations offline, exchange logs in any order, and end with identical databases (§7.6).

### 9.6 Built by agents

The app will be built entirely by AI coding agents, working from a phased implementation plan derived from this spec. The spec and repository follow rules that make that reliable:

- **Acceptance criteria per feature,** written as observable behavior ("Backspace on an empty bullet with children does nothing and shows a hint"), each mapped to at least one automated test.
- **No manual verification steps.** Every behavior is checkable by `go test`: UI through `teatest` golden screens, data through the CLI's `--json` output, safety through crash and round-trip tests. Visual taste reviews (§4.6) are the one human checkpoint.
- **The CLI is the agents' test harness.** Agents create outlines, run commands and inspect results through the same CLI users get (§10.2), which is why it ships in Phase 1.
- **One package per task where possible.** The plan hands agents work along package boundaries (`core`, `store`, `tui`, `cli`) with stable interfaces between them.
- **Golden files are reviewed, not regenerated blindly.** A changed golden screen must be explained in the change description.
- **Stable section numbers** in this spec are referenced from the plan, tests and commit messages.

## 10. Obsidian-inspired features and the CLI

### 10.1 Obsidian-inspired features (Phases 2–3)

Obsidian's best ideas apply here to nodes rather than files.

- **Node links (Phase 2):** type `[[` to search and link to any bullet. Links are stored by node ID, so they survive moves and renames, and they display the target's current text. Enter on a link zooms to its target. Markdown export writes Obsidian-style links.
- **Backlinks (Phase 3):** an on-demand pane listing every bullet that links to the current node, each with its breadcrumb.
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
app edit <id> --text "New text"
app move <id> --under <id> [--after <id>]
app complete <id>        # and: reopen, star, unstar
app delete <id>          # to Trash; restore <id> brings it back
app page get <id>        # Markdown to stdout
app page set <id> --file page.md   # or from stdin
app apply --file ops.json          # many operations as one change set
app history [--source agent:claude] --json
app revert <change-set-id>
app export --format markdown|json|opml --out <dir>
app import --format json|opml|markdown <file>
app snapshots            # and: restore-snapshot <name>
app paths
```

- Nodes are addressed by ID, or by breadcrumb path for convenience. Path lookups return the ID for later calls.
- Every command supports `--json`, with a stable, versioned schema, consistent error codes and exit statuses. Human-readable output is the default.
- Every write is one change set tagged with its source: `cli` by default, `--source agent:<name>` for agents, so their changes can be reviewed and reverted as a unit.
- `app apply` lets an agent submit a batch of operations atomically; either all apply or none do.
- Writes are safe while the TUI is open; the TUI picks them up within a second (§7.3).
- The database is always explicit (`--db`, environment variable, or config default).
- **Phase 4:** an MCP server built on these commands, a shipped usage guide for agents, and richer query output.

## 11. Staged delivery plan

Phase 1 delivers a Workflowy clone you can use daily on your own imported outline. Phase 2 completes Workflowy parity, Phase 3 adds the Obsidian-inspired features, and automation comes last.

&#91;embedded content: delivery plan · 6 phases, each with an exit gate\]

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

- **Project name and branding.** Must not appear in the schema or export formats.
- **Mirrors.** One bullet shown live in two places (for example a task in its project and on a weekly list). Decide after Phase 2 whether daily use calls for it; the schema can add a reference row type without migration pain.
- **Color palette size.** Eight named colors is the starting set; confirm against Omarchy themes that each stays legible on light and dark backgrounds.
- **History retention defaults.** 90 days full, then daily, is a starting point; measure real growth in Phase 0.
- **Default Markdown export layout.** One file, or one file per top-level node.
- **Inbox location.** A fixed top-level `Inbox` node versus a configurable default.
- **Multi-select range.** Siblings only (simpler, Workflowy-like) or any contiguous visible rows.
- **Sync details.** The direction is settled (§7.6). Open details: HLC drift handling, compaction thresholds, and whether page text later merges line by line.

### 12.4 Immediate next steps

1. **Schema spike:** nodes, pages, tags, sync-ready history and coalescing in Go with the chosen SQLite driver. Simulate a week of heavy editing and measure database growth.
2. **Safety tests:** the JSON export round trip, crash injection during typing, and snapshot restore on a clean machine.
3. **Wrap-layout and rendering prototype:** one Bubble Tea outline view with grapheme-aware cursor movement, concealed inline formatting, and the centered reading column.
4. **Omarchy theme reader:** read the current theme's palette, follow live theme switches, and verify file locations against the current Omarchy release.
5. **Reference mockups:** the five screens listed in §4.6, reviewed side by side with Workflowy and with lazygit and btop under three Omarchy themes, one of them light.
6. **Keymap matrix:** record which bindings arrive in each terminal, bare and in tmux, before finalizing §6.4 and §6.6.
7. **Workflowy import:** run a real Workflowy OPML export through the importer and list anything lost.
8. **Stack checks:** confirm Bubble Tea v2 status and its keyboard-protocol support, and choose the pure-Go SQLite driver after verifying FTS5.
9. **Sync-readiness test:** two simulated devices exchange operation logs in random order and converge (§9.5), before any sync UI exists.
10. **Implementation plan:** turn this spec into the phased, agent-ready plan, with acceptance criteria per feature (§9.6).
