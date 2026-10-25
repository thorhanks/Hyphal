# Hyphal — Project Specification v3

Oct 15, 2026 · @Thor

## 1. Overview

We are building a local-first, terminal-native infinite outliner in Go, inspired by Workflowy, for Linux and macOS. It is one continuous tree of bullets that you can capture into, indent, reorder, collapse and zoom. The outline is stored in a single SQLite file with full edit history and automatic snapshots, and can be exported to Markdown, JSON or OPML at any time.

**Goal:** a faithful Workflowy experience in a beautiful terminal UI that matches the Omarchy desktop theme. Obsidian-inspired knowledge features (links, backlinks, saved searches) are the bonus.

**Pitch:** An infinite outline for your terminal: one tree, unlimited depth, stored in one open, durable file you can always export.

**Status:** Concept / pre-implementation. The app is named **Hyphal**; the binary, directories and window class use `hyphal`. The name appears only in user-facing places: the binary, data and config directory names, the window class, and the comment or header that marks an export. It never appears in the schema, operations, the sync log format or JSON field names, so renaming the app later only means moving directories.

## 2. Product principles

1. **One tree, no mandatory pages.** Any node can have children and can be zoomed into.
2. **Thought-first editing.** Capturing and rearranging bullets needs no mode switch. Commands that need single keys use a lightweight block-selection state, not Vim modes. Block selection and page view/edit are modes in the strict sense; they are kept few and shallow: Esc always backs out one level, and Enter always returns to typing.
3. **Your data is never locked in.** One SQLite file with a plain, documented schema, readable by standard tools. Export to Markdown, JSON or OPML at any time; a JSON export restores the whole outline, and its history too when exported with it.
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
2. **Quick capture.** From any shell (`hyphal add "Call the dentist"`) or a Hyprland keybind (`hyphal capture --window`), text lands in the Inbox node without opening the full TUI.
3. **Edit in `$EDITOR`.** Press a key to open the current subtree as Markdown in `$EDITOR` (Neovim on Omarchy). Save and quit; the changes apply as one undoable step.
4. **Recover.** A week later, restore a deleted branch from the Trash, or revert one bad change made by a script without touching anything else.
5. **Leave.** `hyphal export --format markdown` produces a readable outline that works anywhere, at any time.

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
| Pages on bullets (replace Workflowy notes) | Must | 1 | §5.7 |
| Inline formatting and text color | Must | 1 | §5.8 |
| Duplicate node, Move to… | Must | 1 | §5.4, §6.6 |
| Import from Workflowy (OPML) | Must | 1 | §7.5 |
| Undo, Trash | Must | 1 | §5.6, §8.3 |
| `#tags`, click to filter | Must | 2 | §5.9 |
| Links to other bullets | Must | 2 | §10.1 |
| Search operators (`is:complete`, `has:page`, `changed:7d`) | Must | 2 | §6.2 |
| Starred locations | Must | 2 | §6.6 |
| Templates | Later | 3 | §10.1 |
| Dates and a Today node | Never | — | §12.1 |
| Mirrors (one bullet in two places) | Never | — | §12.1 |
| `@person` mentions | Never | — | §12.1 |
| Board (kanban) view | Never in V1 | — | §12.1 |
| Sharing and collaboration | Never | — | §12.1 |

## 4. Information model and UI

### 4.1 Global tree

- Exactly one root per database ("workspace" and "outline" both mean one database file). The root is a real row with `parent_id` NULL, its ID stored in `meta`, and is shown as **Home**. It can't be renamed, deleted, moved, completed or starred, and has no page.
- Every other bullet is a node with editable text, ordered children, exactly one parent, and a permanent random ID (§7.2).
- No distinction between bullet, task or project. Any bullet can also carry a Markdown page (§5.7). Completion is an attribute any node can have.
- **Zoom** makes any node the temporary root. The zoomed node is shown as a heading, with only its descendants below it. The heading can be focused and edited like a bullet.
- **Breadcrumbs** show the path to the zoom target. They always start at Home and use `›` as the separator. Long paths are truncated in the middle: `Home › … › Development › Tooltips`.
- Collapse/expand is independent of zoom and persists across sessions (§7.2).
- No artificial depth limit.

### 4.2 Phase 1 mockup (zoomed into Projects)

Default settings: comfortable density, no indent guides, footer hints for text editing.

```text
Home › Projects                                            outline  ●

Projects

▾ ControllerCraft
  ▾ Development
    • Inventory navigation        (completed: dimmed, struck)
    • Improve controller mappings
    • Fix tooltip positioning
  ▸ Design notes

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
- Indent is 2 columns. Faint `│` indent guides are a setting, off by default. Past 8 levels of the current zoom, each further level indents by 1 column instead of 2, and the footer suggests zooming.
- **Wrapped lines** continue under the start of the text, not under the glyph: continuation rows are indented past the bullet glyph and any checkbox, so each bullet reads as one block.
- **Focus:** in text editing, the focused bullet shows the terminal cursor and its glyph in the accent color; nothing else changes. Concealed markers follow this focus. In block selection the terminal cursor is hidden.

### 4.4 Theme matching (Omarchy first)

**Must have:** on Omarchy, the app matches the desktop theme and follows every theme change while running, with no restart and no configuration.

- **Colors:** all UI colors use the terminal's 16 ANSI colors plus its default foreground and background. Omarchy themes recolor the terminal, so the app follows every theme change instantly, on any OS and in any terminal, with no theme files to read or watch. There are no truecolor themes in V1.
- **Light or dark:** the app asks the terminal for its background color (OSC 11) at launch, and again on focus-in and on the terminal's color-scheme notification where supported, so a switch between light and dark themes is picked up. A terminal that doesn't answer is treated as dark; the `background` setting (`auto`, `light`, `dark`) overrides it.
- **Fonts:** a terminal app always draws in the terminal's font, so the app matches Omarchy's font choice automatically. Omarchy's font switcher changes the terminal font, and the app re-renders on the resize/redraw event that follows.
- Strikethrough, dim and italic each have fallbacks.

### 4.5 Chrome

- **Save indicator:** nothing when saved; `●` while typing is not yet committed (until it seals, §7.3); a red label if a write fails.
- **Screen layout,** top to bottom, all inside the reading column:
  1. **Header row:** breadcrumb left; database name (the file name without `.db`) and save indicator right.
  2. **Banner row,** only while a banner is active (below).
  3. A blank row, then the outline (or page, or full-screen view).
  4. **Footer row,** the last row of the terminal. It shows one of three things, in priority order: a message, the filter query while a filter is active (`/ tooltip · 12 matches`), or contextual key hints.
- **Footer hints:** contextual keys for the current state; the `footer` setting turns them off (messages and the filter still use the row). F1 (or `?` in block selection) opens the full help overlay (§4.8).
- **Messages** are the app's only transient feedback; nothing floats and nothing times out. A message replaces the footer's content and stays until the next keypress, then the footer returns to what it showed before. Messages confirm undoable actions (`Moved "Design notes" and 214 descendants to Trash · Ctrl+Z undoes`), summarize an editor round trip, say where a restore landed, report a clipboard failure, and explain a no-op key ("hints" elsewhere in this spec are messages). A newer message replaces an older one.
- **Banners:** persistent, for failed writes, a database opened read-only, a restore that just ran, or an invalid config. A banner stays until its cause is gone; an informational one (a restore that just ran) clears on the next change to the outline.
- **Views on demand:** Trash (§4.8) and History open as temporary full-screen views, not panels. No sidebar, no daily journal.
- **First run:** the first time the TUI opens a database (a flag in `meta`), it adds a short sample outline that teaches the core keys, unless the outline already holds anything besides an Inbox (for example after an import). Databases created by the CLI start with only Home, so scripts and agents get a clean outline.

### 4.6 Visual design

The outline should read like a calm page, not a terminal dashboard.

- **Centered reading column.** Text width is capped (default 90 columns, configurable) and centered in wide terminals, like Workflowy's page. Breadcrumbs and status sit inside the same column.
- **Inline formatting rendered, markers concealed.** `**bold**` shows as bold; the markers appear only on the bullet being edited, as in Obsidian's live preview (§5.8).
- **Typographic hierarchy:**
  - Optional heading bullets: bold, with extra space above, for section-like nodes.
  - A bullet with a page shows a dim `≡` page marker after its text. There is no page preview in the outline; zoom into the page to read it.
  - Muted breadcrumbs and footer; full-strength color only for text you wrote.
  - Tags as colored text ("chips") from the theme palette, never filled backgrounds, so they stay readable on light themes (§5.9).
- **Stability and motion:**
  - Scroll margin of 3 rows; the cursor never sits on the screen edge.
  - The cursor row stays fixed on screen when collapsing or expanding.
  - No flicker: only changed cells are redrawn.
  - Optional, very brief zoom transition, off by default.
- **Density modes:** *comfortable* (the default: a blank line between the top-level siblings of the current zoom) and *compact*. A heading-style bullet always gets one blank line above it (none when it is the first row under the zoom heading); this blank line is not added to the density blank line, so there is never more than one blank row in a row.
- **Glyph sets:** Unicode by default, ASCII when the locale isn't UTF-8 (`LC_ALL`, `LC_CTYPE`, `LANG`) or the config says so.
- **Concealed markers** appear only on the focused bullet. When they reflow its line, the bullet's first row stays fixed on screen.
- **Contrast:** muted text must stay readable on light themes, so on a light background (§4.4) the app uses bright black rather than dim.
- **Narrow terminals:** below 40 columns or 10 rows the app shows only "Terminal too small" with the current size. Below the column cap, the reading column fills the width.
- **Color tokens.** UI code uses semantic roles, never raw colors:

  | Role | Used for | Color |
  | --- | --- | --- |
  | `text` | Bullet and page text | Default foreground |
  | `muted` | Breadcrumbs, footer, page marker, completed text | Bright black, or dim on dark backgrounds (see Contrast) |
  | `accent` | Focused glyph, overlay borders, page headings in edit mode | Blue |
  | `selection` | Block selection, palette highlight | Reverse video |
  | `match` | Filter and search matches | Yellow, underlined |
  | `highlight` | `==highlight==` text | Yellow background |
  | `chip` | Tag chips | A stable ANSI color per tag (§5.9) |
  | `error` | Failed-write label, error banners | Red |
  | `warning` | Read-only and restore banners | Yellow |
  | `guide` | Optional indent guides | Bright black |

- **Let the desktop frame the app.** Hyprland already draws the window border, gaps and focus color, so the main outline draws no border of its own. The app never paints its own background: it uses the terminal's default background, so the theme's background and any terminal opacity show through. Rounded borders (`╭─╮`) appear only on floating overlays (palettes, help, backlinks), with the title in the top border and the border in the accent color, as in btop and lazygit.
- **Restraint.** No splash screen, toolbars or busy status bar. Breadcrumbs, the save dot and one footer row are all the chrome. Instant response is part of the look: every keypress lands in the next frame (§9.4).
- **Reference mockups** are part of the spec before Phase 1 ends: a 100-column laptop terminal, a full-screen ultrawide, a deep zoom, an active filter, and a light theme. Phase 1's exit gate includes a side-by-side "calm" review against Workflowy and against Omarchy's bundled terminal apps on the same screen.

**Reference sketches** (the window border is Hyprland's, not the app's). They show Phase 2 features (tag chips, the jump palette) and the block-selection footer; footer hints are contextual. The header's right side (database name and save dot) is omitted here.

```text
  Home › Projects › ControllerCraft

  ControllerCraft

  ▾ Development
    • Inventory navigation                    ← completed: dimmed, struck
    • Improve controller mappings  #input     ← tag chip in theme color
    • Fix tooltip positioning  ≡              ← has a page

  ▸ Design notes

                                      ? help · Ctrl+P commands · Ctrl+K jump
```

```text
             ╭─ Commands ─────────────────────────────╮
             │ › col                                  │
             │   Color…                        Esc c  │
             │   Collapse all                         │
             │   Expand to level…           Esc 1–9   │
             ╰────────────────────────────────────────╯
```

### 4.7 At home on Omarchy

Omarchy is *omakase*: one beautiful, opinionated default, keyboard first, themed everywhere, no bloat, with plain config files for anyone who disagrees. The app follows the same rules, so it feels like it shipped with the system.

**Omakase defaults**

- One carefully tuned default look and keymap. Settings exist for real needs (column width, density, keymap), not for every pixel.
- No settings screen. "Edit settings" in the command palette opens the commented `config.toml` in `$EDITOR` and reloads it on save, the way Omarchy hands you its own config files.
- On Omarchy there is nothing to configure: the terminal already carries the theme and the font (§4.4).

**Shared keyboard conventions**

The full keymap is in §6.4. The conventions it follows:

- `?` in block selection, or F1 anywhere, shows every command with its key, as lazygit, Impala and BlueTUI do.
- Esc backs out one level and never zooms out: overlay → where you were; page edit → page view → outline; text editing → block selection → clear filter → nothing (a hint). Zooming out is always explicit.
- There are no side panes. Backlinks is a floating overlay like the palettes; Esc closes it.
- Ctrl+Q quits from anywhere. Pending typing is flushed first, so there is never anything to save.

**Native app integration**

- On Linux, ships a `.desktop` entry and its own window class, so it launches from the app launcher like Omarchy's other terminal apps and can be targeted by Hyprland window rules. On macOS it is launched from a terminal; there is no app bundle in V1.
- **Capture window:** `hyphal capture --window` starts a new terminal window running `hyphal capture` with its own class, which a window rule makes small and floating. The terminal command is a config template; the default uses `xdg-terminal-exec` with the class option of Omarchy's default terminal. Inside, type one line and press Enter: it lands in the Inbox, and the window closes. Plain `hyphal capture` does the same in the current terminal. The docs suggest a Super binding, checked against Omarchy's defaults.

### 4.8 Phase 1 screens

Phase 1 needs these screens besides the outline and the page view. Later-phase screens are listed in §12.3.

- **Trash view** (full screen, §8.3): one row per deletion, newest first: the deleted bullet's text, its former breadcrumb, the descendant count and the age. Enter previews the subtree read-only; `r` restores; `d` purges that deletion after a one-line confirmation (purging is final, so purge and Empty Trash are the only actions in the app that ask for confirmation); Esc returns to the outline.
- **Help overlay** (F1, `?`): every command in the current context with its key, generated from the keymap (§6.4), filterable by typing.
- **Color picker** (`c` in block selection, or the command palette): an overlay listing the eight colors as swatches with their names, plus "No color". `1`–`8` (and `0` for No color) or arrows and Enter choose; Esc cancels.
- **Read-only mode** (newer schema, failed integrity check, or a second TUI): a persistent banner says why; everything works except changes, and any editing key shows a hint instead.

## 5. Editing behavior

### 5.1 Two states, no modes

- **Text editing (default).** Printable keys type. Commands use modifiers.
- **Block selection.** Esc selects the current bullet as a whole. Up/Down move the selection; Shift+Up/Down extend it across siblings only. Single keys act on the selection (§6.4). Enter returns to text editing with the cursor at the end of the bullet. Other printable keys do nothing except show a hint, so text typed after Esc never runs commands by surprise.

### 5.2 Enter

| Cursor position | Result |
| --- | --- |
| Start of a non-empty bullet | Insert an empty sibling above; cursor stays on the current text |
| Middle of text | Split: text after the cursor becomes a new sibling below; children stay with the first half |
| End of an expanded bullet with children | Create a new first child |
| End of a collapsed bullet, or a leaf | Create the next sibling |
| Empty bullet that is not at the top level of the current zoom | Outdent it (Workflowy behavior) |
| Empty bullet at the top level of the current zoom | Create the next sibling |
| Anywhere in the zoom heading | Create a new first child of the zoom target; the heading is never split |
| With a text selection | Delete the selected text, then apply the rule for the resulting cursor position |

On a split, the first half keeps the node's identity and everything attached to it: ID, children, page, star, completion, inbound links and history. The second half is a new node.

### 5.3 Backspace at the start of a bullet

Rows are checked top to bottom; the first match applies.

| Situation | Result |
| --- | --- |
| With a text selection | Delete the selected text only |
| No previous visible bullet, or the cursor is at the start of the zoom heading | No-op |
| Bullet is empty, no children | Delete it; cursor to the end of the previous visible bullet (or the zoom heading) |
| Bullet is empty, has children | No-op; a message hints to use Esc then Delete |
| The previous visible bullet is the zoom heading | No-op. Nothing is ever merged into the zoom heading |
| Both bullets have a page | No-op with a hint. Merging would have to combine two pages |
| Bullet has text, no children | Merge text into the end of the previous visible bullet |
| Bullet has text and children, previous bullet has no children | Merge text; children are adopted by the previous bullet |
| Bullet has text and children, previous bullet has children | No-op with a hint. Merging would interleave two child lists unpredictably |

"Previous visible bullet" means the row above on screen, which may be deeper than the current bullet (the last visible descendant of the previous sibling). Under an active filter, hidden bullets are skipped.

On a merge, the previous bullet survives with its ID, completion state and history. The merged-away bullet goes to the Trash in the same change set; its page moves to the survivor (if the survivor has none), and its star and inbound links are retargeted to the survivor.

### 5.3a Delete at the end of a bullet

Delete pulls the next visible bullet into the current one, mirroring Backspace: the current bullet survives and the next one is merged away.

Rows are checked top to bottom; the first match applies.

| Situation | Result |
| --- | --- |
| With a text selection | Delete the selected text only |
| No next visible bullet | No-op |
| Next visible bullet is the current bullet's own first child | No-op with a hint |
| Next visible bullet is empty, no children | Delete it |
| Both bullets have a page | No-op with a hint |
| Next visible bullet has text, no children | Merge its text onto the end of the current bullet |
| Next has children, current has no children | Merge its text; the current bullet adopts its children |
| Next has children, current has children | No-op with a hint |

Identity follows the merge rule above: the current bullet keeps its ID, and the merged-away bullet goes to the Trash.

### 5.4 Structure

- **Tab / Shift+Tab** indent or outdent the current bullet (or selection) with its descendants. Indent is a no-op on the first sibling.
  - Indenting under a collapsed sibling expands it, so the bullet stays in view.
  - Outdenting makes the bullet the next sibling of its old parent. Later siblings stay where they are; they are never adopted (Workflowy behavior).
  - Outdenting a top-level bullet of the current zoom is a no-op with a hint.
- **Move up/down** among siblings moves whole subtrees. Moving past the first or last sibling crosses into the adjacent parent's child list, as Workflowy does. For example, with siblings A and B under one parent:
  - Moving B's first child up makes it the last child of A (A expands if collapsed). If B is the first sibling (there is no A), the child moves up one level and becomes B's previous sibling.
  - Moving A's last child down makes it the first child of B, mirroring the rule above.
  - Nothing moves outside the current zoom; at its edges, moving is a no-op.
- **Delete a subtree** (Delete or `d` in block selection) never asks for confirmation. It shows an undo message with the descendant count.
- **Complete / reopen** toggles a node's completion (storage rule: §7.2). Completed nodes stay in place; a toggle hides them. On a selection with mixed state, any open bullet means "complete all", otherwise "reopen all". When a newly completed bullet hides, the cursor moves to the next visible bullet and a message offers undo.
- **Editing under a filter:** a bullet you create or edit stays visible until the filter changes, even if it doesn't match. Indent and move use the real tree, including hidden bullets; the moved bullet stays visible with its ancestors.
- **Tasks:** toggle task makes a bullet a task, shown with a checkbox (§4.3). Completing a task checks its box.
- **Leading markers convert.** Typing `[ ] ` at the start of a bullet makes it a task, `[x] ` a completed task, and `## ` a heading-style bullet; the marker is removed. Text set by paste, import, the editor round trip or `hyphal edit --text` follows the same rule, so stored bullet text never starts with a marker and the Markdown dialect needs no escape for them.
- **Duplicate** copies the subtree as the next sibling, with new IDs, pages, completion, task and heading state. Stars are not copied, and links inside the copy still point to their original targets.
- **Heading style:** toggle heading makes a bullet a section-like heading (§4.6). It changes only the look, never the structure.
- **Move to…** opens a fuzzy parent picker over all node text, built on the command palette (Phase 1). From Phase 2 it uses the jump palette, with starred locations first. This is how subtrees cross long distances.

### 5.5 Paste and clipboard

- Bracketed paste is enabled; the terminal's own paste key delivers it (§6.4). Single-line paste inserts text, replacing any text selection.
- Multi-line paste is parsed with the shared Markdown dialect (§7.5) into a subtree. Plain lines become bullets; indentation becomes nesting; `^id` suffixes are ignored, so pasted nodes always get new IDs.
  - Cursor at the end of a bullet: the subtree is inserted as siblings after it.
  - Cursor mid-text: the bullet is split first (§5.2), and the subtree goes between the two halves.
  - Empty bullet: the subtree replaces it.
  - In block selection: inserted as siblings after the selection.
- Copy (block selection) writes the selected bullets and their descendants to the system clipboard as indented Markdown in the shared dialect: text with inline markup, task, completion and heading markers. Pages and IDs are not copied. Cut copies, then deletes to the Trash.
- **Cut and paste is a move.** The app remembers its last cut. If the clipboard still holds exactly the text that cut wrote, paste restores the cut bullets at the paste position instead of creating new ones: same IDs, pages, stars and history, as one change set (the cut and the move both stay undoable). Text pasted from anywhere else, or after the clipboard changed, creates new bullets as above. To copy a subtree with its pages, use Duplicate (§5.4).
- **Clipboard transport:** in a local session, the platform tool first: `wl-copy` (Wayland), `xclip` or `xsel` (X11), `pbcopy` (macOS). These report failure, so a failed copy shows a message. Over SSH (`SSH_TTY` set), or when no local tool exists, the app uses OSC 52, which works through SSH and tmux but gives no reply, so its failures can't be detected; the docs explain tmux's `set-clipboard on`.

### 5.6 Undo and redo

- **Undo is per session.** The TUI keeps an undo stack in memory. Each step is one change set you started from the TUI, including `$EDITOR` round trips and page saves made from it. Undo applies the step's inverse as a new change set whose `undo_of` points at it; redo applies it again. Any new change from the TUI clears the redo steps. Quitting clears the stack; older changes are recovered from the Trash, with `hyphal revert`, or from the History view (Phase 2).
- Changes from the CLI or an agent are never on the stack, so Ctrl+Z never undoes something you didn't do; they are reverted with `hyphal revert`.
- If a later change from another source touched the same fields, undo follows the revert rule: those fields are skipped and reported (§8.2).
- Typing coalesces into one step per node; the step closes when the typing operation seals (§7.3). Ctrl+Z first seals any open typing, so the typing just done is the step undone.
- Each step remembers the cursor and selection from before the change, and undo restores them. Undoing a change outside the current zoom zooms to show it, with a message.

### 5.7 Pages

- **Every bullet stays one line.** Long-form writing lives on an optional **page** attached to the bullet: a full Markdown document with headings, lists, code blocks, quotes and tables, as long as you like.
- **Page zoom:** `p` in block selection opens the bullet's page in view mode; on a bullet without one, it opens an empty page in edit mode (`$EDITOR` in Phase 1). A page saved empty is not created; clearing an existing page's text deletes the page, as one undoable change. Esc from view mode returns to the outline in block selection on the page's bullet. Normal zoom (`z`) still shows only the bullet's children.
- **Title:** the bullet's text is the page title. It is not stored in `pages.markdown`; view mode renders it as the page heading, with inline markup.
  - In `$EDITOR` and in edit mode, the title appears as a first line `# Title`. On save, the first line is the title if it is an H1 (`# …`): changing it renames the bullet. If the first line is not an H1, the bullet's text is unchanged and the whole file is the page. (So deleting the title line of a page whose body starts with `# Intro` renames the bullet to "Intro"; undo reverses it.)
  - **Delete page** (command palette) removes the page from a bullet, as one undoable change.
- **Indicator:** a dim page marker follows the bullet's text (§4.6). The outline never shows page content; zoom into the page to read it.
- **Two modes, no live preview.** Editing inside rendered output is the hardest and buggiest part of a Markdown editor, so pages separate reading from writing:
  - **View mode** renders the Markdown read-only: headings, lists, quotes, tables, and code blocks syntax-highlighted with the theme palette. It scrolls, and Tab moves between links; Enter on the focused `[[link]]` zooms to its target.
  - **Edit mode** is a plain multi-line text editor showing the Markdown exactly as typed. Syntax highlighting keeps it pleasant without concealment: headings bold in the accent color, markers dimmed, code blocks tinted, links underlined. It shares the outline's wrap-layout and cursor engine.
- **Switching:** `e` in view mode, or Enter with no link focused, enters edit mode; Esc in edit mode returns to view mode. The parser's source ranges keep your place: switching scrolls to the same block and puts the cursor on it.
- **Split view** (wide terminals): `s` in view mode shows source and rendered output side by side, updating as you type. Off by default; a setting can make it the default above a chosen width.
- **Common elements of both modes:**
  - The breadcrumb continues with a page marker and the mode: `Home › Projects › ControllerCraft › ≡ view`.
  - Same centered reading column and theme as the outline.
- **Editing, staged:**
  - **Phase 1:** view mode, with `e` opening the page in `$EDITOR` (Neovim on Omarchy). On exit, the page is saved back as one undoable change set.
  - **Phase 2:** the built-in edit mode, mode switching with position sync, and split view. `e` then opens the built-in editor and `E` opens `$EDITOR`.
  - Live preview is not planned. Concealed markers remain only on single-line bullets in the outline (§5.8), where they are simple.
- **A page belongs to its bullet:** it moves with it, goes to the Trash and comes back with it, and is copied by Duplicate.
- **Links and tags work on pages:** `[[` links to bullets, `#tags` count for the page's bullet, and pages appear in backlinks (Phase 3).
- **Rule of thumb for users:** structure goes in bullets, prose goes in pages. Lists inside a page are text, not tree nodes, so they can't be zoomed or moved in the outline.
- **Export:** each page becomes its own `.md` file linked from the outline export (§7.5). OPML export stores the page in the `_note` attribute, which Workflowy reads as a note.

### 5.8 Inline formatting and color

Formatting uses Markdown-style markers, typed directly or wrapped around selected text. Markers are concealed except on the bullet being edited.

| Style | Syntax | With text selected, type |
| --- | --- | --- |
| Bold | `**text**` | `*` twice |
| Italic | `*text*` | `*` |
| Strikethrough | `~~text~~` | `~` twice |
| Inline code | `` `text` `` | `` ` `` |
| Link (URL) | `[label](https://…)` | — (rendered as the label, an OSC 8 hyperlink where the terminal supports it; `o` in block selection opens it) |
| Highlight | `==text==` | `=` twice |
| Text color | `<red>text</red>` | — (Color… in the command palette) |

- **Colors:** `red`, `orange`, `yellow`, `green`, `cyan`, `blue`, `purple`, `gray`. Each maps to an ANSI color, so colored text follows theme changes.
- `<red>` and `==highlight==` were chosen because they are short and readable when concealment is off. `<red>` is not HTML, so Markdown export converts it (below). `==highlight==` renders in Obsidian but not in plain CommonMark tools. Unknown tags are left as plain text.
- **Wrapping:** with text selected (Shift+arrows in text editing), typing `*`, `~`, `=` or `` ` `` wraps the selection in that character instead of replacing it, and keeps it selected, so a second press nests the wrap (`*` twice makes bold). The command palette's Bold, Italic and Color… commands apply to the text selection, or to whole bullets in block selection.
- Applying a color to whole bullets (`c` in block selection) wraps each selected bullet's text in one color tag, replacing any color tag that already wraps it.
- **Grammar:**
  - Opening markers can't be followed by a space and closing markers can't follow one, as in CommonMark, so `2 * 3 * 4` stays literal.
  - Styles nest (`**bold *and italic***`) but never overlap; an unmatched marker is literal text.
  - Inside an inline code span everything is literal.
  - A backslash escapes a marker: `\*`.
  - Color tags nest with other styles but not with each other.
  - Only the markers in the table are recognized while typing, so `_` and `__` stay literal (safe for `snake_case`). Importers convert `_italic_` and `__bold__`.
- **Color mapping:** red, yellow, green, cyan and blue use the matching ANSI color. Orange uses bright red, purple uses magenta, and gray uses bright black.
- Markdown export keeps the markers; colors export as `<span style="color: …">` so Obsidian renders them.

### 5.9 Tags (Phase 2)

- `#tag` anywhere in a bullet or page. Letters, digits, `-`, `_` and `/` are allowed; `/` makes nested tags (`#project/lighthouse`), as in Obsidian.
- Tags render as colored text (chips, §4.6). Each tag gets a stable color from the palette, overridable in config.
- `o` in block selection, or a click (Phase 2), sets the in-place filter to the bullet's tag. Enter never follows a tag; it always edits. Typing `#` offers completion from existing tags.
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
- **Search sees what you see.** Matching and highlighting run on the markup-stripped text (concealed markers removed, links shown as their display text), so `**bold** text` matches "bold text" and highlights map directly onto the rendered row.
- **Search runs in memory.** Every query is a case-insensitive substring scan of the in-memory tree, ignoring diacritics; there is no search index. The tree holds page text too, loaded in the background after the first frame so launch stays fast. Phase 0 verifies Unicode case folding, diacritics and the 50 ms target at 100,000 nodes.
- Trashed nodes never match.
- A match inside a page keeps its bullet visible with the page marker highlighted, and shows the matching line dimmed beneath it.
- Saved searches arrive in Phase 3 (§10.1).

### 6.3 External editor and quick capture

- **Edit in `$EDITOR`** (Phase 1): writes the focused bullet's subtree (or the selection, or the whole zoom when the zoom heading is focused) to a temp Markdown file in the shared dialect (§7.5), suspends the TUI and opens `$EDITOR`. Each line carries its node's ID as a ` ^id` suffix so edits map back.
- The file always contains every live descendant, whatever is collapsed, hidden by Show / hide completed, or hidden by an active filter. A line missing from the file therefore always means "delete this node", never "it wasn't shown".
- On exit, the file is parsed and diffed against the subtree. Everything applies as one change set with source `editor`, undoable with Ctrl+Z, and a message summarizes it ("3 changed, 2 new, 1 to Trash").
  - Changed text, task, completion or heading markers update their nodes. Order and indentation in the file become the new structure, so moving lines moves nodes.
  - Lines without an ID become new nodes. Removed lines move to the Trash, with their pages, stars and history intact.
  - Deleting a parent line but keeping its children: the children take the place the indentation in the file gives them.
  - A duplicated ID: the first line keeps the node; later copies become new nodes.
  - A stripped ID makes the line a new node and sends the old one to the Trash. The summary message makes this visible, and undo reverses it.
  - Only a trailing ` ^` followed by a valid ID (§7.2) is an ID; any other `^` is text.
- Pages are not part of the file. Each bullet keeps its page through its ID; pages are edited on their own (§5.7).
- The temp file lives in a private directory (`$XDG_RUNTIME_DIR/hyphal/`, or `hyphal/tmp/` in the data directory, mode `0700`) and is created `0600`. Editors may keep their own swap and undo files elsewhere; the docs say so.
- If a bullet in the subtree changes from another connection while the editor is open, the concurrency rule of §7.3 applies on save: your edit lands after the other change, which stays revertible, and the message says so.
- If the file can't be parsed, nothing is applied: the temp file is kept and a banner offers to reopen it.
- **Quick capture** (Phase 1): `hyphal add "text" [--under <node>]` (§10.2) appends to the Inbox node by default, in its own transaction. The Inbox is a top-level node named `Inbox`, created on first use; its ID is stored in `meta`, so it can be renamed or moved freely and capture follows it. If the Inbox is in the Trash or has been purged, capture creates a new top-level `Inbox`, points `meta` at it, and says so; a trashed old Inbox stays in the Trash untouched. A running TUI notices the change through SQLite's `PRAGMA data_version` and refreshes.

### 6.4 Keymap and terminal support

This section is the only place key bindings are defined. Other sections name commands and may mention their keys, but this table wins. In code the keymap is data, one table per context, and a test fails if a key is bound twice in one context, a command has no binding that works without the kitty keyboard protocol, a bullet command has no single key in block selection, or a default binding outside word editing uses Alt.

**Contexts:** global, text editing, block selection, filter input, page view, page edit (Phase 2), overlays (palettes, pickers, help, backlinks) and the Trash view. The Esc chain moves between them (§4.7).

**Two layers.** Block selection is the complete layer: every bullet command has a single key there, and single keys arrive the same in every terminal, on Linux and macOS. Chords are shortcuts for the most frequent commands, limited to keys that also arrive everywhere: Ctrl+letter, Tab, Shift+arrows and Ctrl+Shift+Up/Down. No command is bound to Alt, Option, Cmd or a bare Ctrl+arrow; Alt appears only in word-editing keys, as terminals send them. One key, Ctrl+Enter, needs the kitty keyboard protocol, and it is only an alternative.

**Global** (every context, unless that context's table rebinds the key)

| Command | Key | Notes |
| --- | --- | --- |
| Quit | Ctrl+Q | Flushes pending typing first |
| Help overlay | `?` in block selection, F1 | F1 needs Fn on Mac laptops |
| Command palette | Ctrl+P | |
| Jump palette (Phase 2) | Ctrl+K | |
| Undo / redo | Ctrl+Z / Ctrl+Y | |
| (Save) | Ctrl+S | Nothing to save; shows "Saved automatically" |

**Text editing** (the default state). Commands without a chord here are in block selection: Esc, then the key.

| Command | Key | Notes |
| --- | --- | --- |
| New bullet, split, outdent empty bullet | Enter | §5.2 |
| Backspace / Delete at bullet edges | Backspace / Delete | §5.3 |
| Indent / outdent | Tab / Shift+Tab | |
| Block selection | Esc | |
| Move by character / visual line | Arrow keys | |
| Move by word | Ctrl+Left / Ctrl+Right, Alt+Left / Alt+Right, Alt+`b` / Alt+`f` | Whichever the terminal sends; Option+Left/Right on macOS |
| Start / end of visual line, then bullet | Home / End, Ctrl+A / Ctrl+E | Cmd+Left/Right on macOS, where the terminal maps them |
| First / last bullet in the zoom | Ctrl+Home / Ctrl+End | |
| Scroll a screen | Page Up / Page Down | |
| Delete word backward | Ctrl+W, Ctrl+Backspace, Alt+Backspace | Option+Backspace on macOS |
| Delete to start of visual line | Ctrl+U | Cmd+Backspace on macOS, where the terminal maps it |
| Extend text selection | Shift+arrows, Shift+Home/End, Ctrl+Shift+Left/Right | |
| Wrap selected text in a marker | `*`, `~`, `=`, `` ` `` | Only with text selected (§5.8) |
| Copy / cut selected text | Ctrl+C / Ctrl+X | Ctrl+C with no selection shows "Ctrl+Q quits" |
| Paste | The terminal's paste (Ctrl+Shift+V, Cmd+V) | Bracketed paste (§5.5); the app binds no paste key |
| Move subtree up / down | Ctrl+Shift+Up / Down | |
| Toggle complete | Ctrl+Enter | Kitty protocol only; Esc `x` works everywhere |
| Filter | Ctrl+F | |

**Block selection.** Single keys act on the selected bullets. Chords from text editing that act on a bullet also work here and apply to the whole selection.

| Key | Command |
| --- | --- |
| Up / Down | Move the selection |
| Shift+Up / Shift+Down | Extend the selection across siblings |
| `-` / `=` | Collapse / expand |
| `1` … `9` | Expand to level N (`1` collapses everything below the children) |
| `0` | Expand all descendants |
| Tab / Shift+Tab | Indent / outdent |
| `[` / `]` | Move subtree up / down (also Ctrl+Shift+Up / Down) |
| `z` / `Z` | Zoom in / out |
| `~` | Zoom to root |
| Enter | Return to text editing, cursor at the end |
| `x` | Toggle complete |
| `t` | Toggle task |
| `#` | Toggle heading style |
| Delete, `d` | Delete to the Trash, with an undo message |
| Ctrl+C | Copy as Markdown (§5.5) |
| Ctrl+X | Cut: copy, then delete to the Trash |
| `p` | Open page |
| `o` | Follow the bullet's link or tag; with several, a picker lists them. A URL opens in the default browser (`xdg-open` on Linux, `open` on macOS) |
| `e` | Open subtree in `$EDITOR` |
| `D` | Duplicate subtree |
| `h` | Show / hide completed |
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
| Up / Down | Scroll by line |
| Space, Page Down / Page Up | Scroll by screen |
| Home / End | Top / bottom |
| Tab / Shift+Tab | Focus next / previous link |
| Enter | Follow the focused link; with no link focused, edit |
| `e` | Edit: `$EDITOR` in Phase 1, the built-in editor from Phase 2 |
| `E` (Phase 2) | Open in `$EDITOR` |
| `s` (Phase 2) | Toggle split view |
| Esc | Back to the outline, in block selection on the page's bullet |

**Page edit** (Phase 2): the text-editing keys for movement, selection, clipboard and wrapping. Enter inserts a newline, Tab inserts two spaces, and Esc returns to view mode.

**Overlays:** typing filters the list; Up/Down or Ctrl+N/Ctrl+P move (inside an overlay Ctrl+P moves up instead of reopening the palette); Enter chooses; Esc closes. In the color picker, `1`–`8` choose a color directly and `0` removes it.

**Trash view** (§4.8): Up/Down move; Enter previews the deletion; `r` restores it; `d` purges it after confirmation; Esc returns to the outline.

**Terminal rules**

- **Kitty keyboard protocol.** The app enables it through Bubble Tea when available. Only Ctrl+Enter needs it (without it, it arrives as Enter), and Esc `x` does the same everywhere.
- **Alt is for word editing only.** Legacy terminals and tmux send Alt+key as Esc then the key, and Bubble Tea reads a quick Esc+key as Alt+key; macOS terminals send the same for Option+key when Option is set as Meta. Alt+`b`, Alt+`f` and Alt+Backspace move or delete by word; every other Alt key does nothing. A slow Esc then a key enters block selection and runs that key there. `b`, `f` and Backspace are unbound in block selection, so neither reading of those three ever does harm.
- **Bare Esc** is recognised after Bubble Tea's escape timeout; the kitty protocol removes that wait.
- **tmux.** The docs recommend `set -s escape-time 10` and `set -s extended-keys on` (with the `extkeys` terminal feature) and ship the snippet.
- **Readline keys.** The editing keys that macOS terminals send for Cmd and Option combinations are supported on both platforms: Ctrl+A, Ctrl+E, Ctrl+U, Ctrl+W and Alt+`b`/`f`. Other readline keys are not emulated; Ctrl+K and Ctrl+P are app commands.
- **Flow control.** Raw mode disables XON/XOFF, so Ctrl+Q and Ctrl+S reach the app; the test matrix verifies this.
- **Compositors.** Bindings that a desktop compositor commonly intercepts (Super combinations under Hyprland, Ctrl+arrows under macOS) are never used.
- Keymap is configurable in TOML, per context. The same duplicate check runs when the config loads; on a conflict the app keeps the default binding for that command and shows a banner.
- A test matrix covers Ghostty, Kitty, Alacritty, foot and WezTerm on Linux, and Ghostty, Kitty, Alacritty, WezTerm, iTerm2 and Terminal.app on macOS, each bare and inside tmux, recording which bindings arrive intact.

**macOS keyboard:** the keymap is the same as on Linux and needs no terminal setup. Cmd+Left/Right and Cmd+Backspace work where the terminal maps them to Ctrl+A, Ctrl+E and Ctrl+U (Ghostty by default, iTerm2 with its Natural Text Editing preset); Option+Left/Right/Backspace work as Alt+`b`/`f`/Backspace or as Alt+arrows. Ctrl+Left/Right word movement may be taken by Spaces. Apple's Terminal.app is treated as ANSI-only; Ghostty, kitty, WezTerm, Alacritty and iTerm2 are fully supported.

### 6.5 Mouse

- Phase 2: scroll, click to place the cursor, click a glyph to toggle collapse.
- Mouse capture disables the terminal's native text selection. The app documents Shift+drag (which most terminals use to bypass capture) and offers a config toggle to turn capture off.
- No drag-to-move; nodes move with the keyboard.

### 6.6 Palettes, stars and small commands

- **Command palette (Ctrl+P, Phase 1):** every command searchable by name, showing its key binding, recently used first. It is the discoverability layer for everything in this spec, so terminal key limits never hide a feature.
- **Stars (Phase 2):** "Toggle star" stars the bullet you're on: the focused bullet, or the selected ones in block selection. To star the zoom target, toggle it on the zoom heading. Stars appear first in the jump palette and in a Stars view; `is:starred` finds them.
- **Small commands (Phase 1),** with keys in §6.4:
  - Expand all (`0`) and expand to level 1–9 under the current node; level 1 collapses everything below the children
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
  Linux:  ~/.local/share/hyphal/            (honors XDG_DATA_HOME)
  macOS:  ~/Library/Application Support/hyphal/

  outline.db          # the data (WAL mode)
  snapshots/          # automatic backups (§8.5)
  exports/            # default target for scheduled export

Config (both platforms, CLI convention)
  ~/.config/hyphal/config.toml              (honors XDG_CONFIG_HOME)

Sync folder (Phase 5, user-chosen, e.g. inside iCloud Drive or Syncthing)
  <sync-folder>/hyphal/<device-id>.log      # §7.6
```

Separate outlines are separate database files, selected with `--db` or a config default. `hyphal paths` prints every location in use.

### 7.2 Schema (sketch)

```sql
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT);
-- schema_version, format_id (neutral, never the app name),
-- root_id, inbox_id, tui_opened (§4.5), a one-paragraph description of the schema

CREATE TABLE nodes (
  id           TEXT PRIMARY KEY,     -- random, stable forever: 12 chars, lowercase base32
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

CREATE TABLE stars (
  node_id    TEXT PRIMARY KEY REFERENCES nodes(id),
  starred_at TEXT NOT NULL
);

CREATE TABLE view_state (           -- per device; not in history or sync
  node_id   TEXT PRIMARY KEY REFERENCES nodes(id),
  collapsed INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE ui_state (              -- per device; not in history or sync
  key   TEXT PRIMARY KEY,            -- show_completed (default true), zoom, cursor:<zoom-id>, palette_recent
  value TEXT NOT NULL                -- JSON
);

CREATE INDEX nodes_by_parent  ON nodes(parent_id, position);
CREATE INDEX nodes_in_trash   ON nodes(deleted_at) WHERE deleted_at IS NOT NULL;

-- change_sets and ops: see §8.2
```

- **IDs** are random and permanent. They are what the CLI, links and the editor round trip use.
- **Ordering** uses fractional index strings, so moving or reordering a node rewrites one row.
- **Completion:** `is_task` marks an open task (`[ ]`); `completed_at` marks completion for any node. Reopening clears `completed_at` and keeps `is_task`, so a plain bullet never turns into a task by accident.
- **View state** (collapse) and **UI state** (show/hide completed, the current zoom, the cursor per zoom, palette recents) are per device and excluded from history: folding never creates undo steps. Both are included in JSON export so a restore brings back the same view. The filter is never persisted.
- **Timestamps** are RFC 3339 in UTC with milliseconds (`2026-10-17T09:30:00.123Z`), displayed and interpreted in local time (`created:2026-10`). `updated_at` changes when the node's own text, page, task, heading or completion changes, not on moves, stars or changes to descendants; `changed:7d` uses it.
- **Integrity:** `PRAGMA foreign_keys=ON` on every connection. History ops reference nodes by ID without a foreign key, so purging a node never cascades into history (§8.3).
- **File permissions:** the data directory is created `0700`, and the database, snapshots and exports `0600`.
- The schema stays plain enough that anyone opening the file in a SQLite browser understands it in a minute.
- **Formatting** is stored as the markup the user typed, not as separate style rows. The text stays readable in any SQLite browser and exports without conversion; rendering parses it on display.
- **Tags** are not stored separately: they are parsed from bullet text and pages into the in-memory tree at load and on every edit. A tag written in a page counts for the page's bullet.
- **Stars** are user data and part of history; collapse state is not.

### 7.3 Transactions and concurrency

- Every user action is one transaction that updates `nodes` and writes its history together (§8.2).
- **Typing is committed when it seals.** Consecutive typing in one node forms one operation, kept in memory until it seals: after a 2-second pause, on leaving the node, on any structural action, on quit, or 5 seconds after it opened. It is then written once and never changed. A crash loses at most the unsealed typing, 5 seconds at worst.
- Pending typing is flushed and sealed on quit, on SIGHUP (terminal closed) and on SIGTERM, before the app exits.
- WAL mode with `synchronous=FULL`. The extra cost is negligible at this write rate.
- The TUI keeps an in-memory tree for rendering; all writes go through `core` to `store`.
- The CLI opens its own connection with a busy timeout. The TUI polls `PRAGMA data_version` about every 500 ms; when another connection has written, it reloads the whole tree off the UI loop and keeps zoom, cursor and selection by node ID. Incremental reloading is a later optimization if ever needed.
- **Concurrent edits to one node.** Every write bumps the node's `version`. If another connection changes a node while the TUI holds unsealed typing on it, the user's typing is never dropped: it commits as a new change set after the other change, which stays in history and can be reverted, and a message says so. A CLI or agent that must not overwrite newer work passes `--if-version <n>` and gets a conflict error instead. The same rule applies to a page open in `$EDITOR` while `hyphal page set` runs.

### 7.4 Migrations

- Schema versions are numbered and migrations run forward only.
- A snapshot is taken before every migration.
- A database with a newer schema than the app knows opens read-only with a banner, never migrated backwards.

### 7.5 Export and import

| Format | Export | Import |
| --- | --- | --- |
| JSON | Lossless: every node including the Trash, pages, stars, collapse and UI state; IDs, positions and timestamps. History with `--history` | Into an empty database only; on a non-empty one (including the first-run sample outline) the error names `--replace`. `--replace` snapshots the database, then replaces its contents. `--under <node>` imports as new nodes with new IDs instead |
| Markdown | The shared dialect below; one file (`outline.md`, the default) or, with `--split`, one file per top-level node; pages as separate files in `pages/` | New nodes under a chosen parent |
| OPML | Plain text for Workflowy and other outliners (below) | New nodes under a chosen parent (for migrating from Workflowy and other outliners) |

- **Markdown and OPML imports** without `--under` land in one new top-level bullet named after the imported file (`workflowy-export.opml`), so the whole import can be inspected, moved or trashed as a unit. With `--under`, the imported top-level items become children of that node directly.
- JSON export never contains the config file or snapshots.
- Every export records the app and the export time without breaking its format: a top-level `exported_by` field in JSON, `<head>` elements in OPML (after the `<?xml` declaration), and an HTML comment on the first line of Markdown, which importers skip.
- Exports are written atomically (temp file, then rename).
- **Scheduled export** is an optional setting: Markdown or JSON, daily or on quit, into a folder you choose (for example a Git repo), keeping the last N exports. There is no daemon: a daily export runs on quit, or at launch if it is overdue. `hyphal export --scheduled` runs it from a systemd timer, launchd or cron.
- CI checks that database → JSON → database is identical: every exported field is equal, including IDs, positions, timestamps, pages, stars and view state, and history when exported with `--history`.

**Markdown dialect.** Export, import, multi-line paste (§5.5) and the editor round trip (§6.3) share one dialect, close to GitHub-flavored Markdown:

```markdown
- Plain bullet
- [ ] Open task
- [x] Completed task
- ## Heading-style bullet
  - Child, indented two spaces
- Bullet text with its ID ^k3j9x2m4q7tb
```

- Bullet text never starts with `[ ] `, `[x] ` or `## ` (§5.4), so those markers are unambiguous.
- Inline markup is written as stored (§5.8). Export converts color tags to `<span style="color: …">`; the editor round trip and copy (§5.5) keep them as typed.
- The ` ^id` suffix is written by the editor round trip and by `hyphal export --ids`. Only the editor round trip maps IDs back to existing nodes; Markdown import always creates new nodes and ignores the suffix.
- A completed bullet that is not a task exports as `[x]` and re-imports as a completed task. This small loss is documented; JSON and the editor round trip (through `^id`) keep it exact.
- Import accepts any consistent indentation (spaces or tabs) and `-`, `*`, `+` or numbered list markers. Lines that aren't list items become bullets.

**Pages in exports:** Markdown export writes each page as its own `.md` file in a `pages/` folder, and the outline links to it. File names are a slug of the bullet text (lowercase letters, digits and hyphens, at most 50 characters) plus the node ID, so they never collide. The result reads much like an Obsidian vault. JSON export includes pages in full; OPML puts them in `_note`.

**Workflowy OPML import mapping.** Verified against a real Workflowy export in Phase 1 (§12.4).

| Workflowy | Imported as |
| --- | --- |
| Outline text with inline HTML | Bullet text with Markdown-style markup (§5.8) |
| `<b>`, `<i>`, `<s>`, `<code>`, `<a href>` | `**…**`, `*…*`, `~~…~~`, `` `…` ``, `[label](url)` |
| `<span class="colored c-red">` and other text colors | `<red>…</red>`, nearest of the eight colors |
| Background-color spans | `==…==` |
| `_note` | Page |
| `_complete="true"` | Completed bullet (not a task) |
| `#tags` | Tags, unchanged |
| `@mentions`, dates | Plain text |
| Mirrors | Independent copies; the mirror link is lost. The import summary counts them when the export marks them |
| Boards | Plain children; the board layout is lost |

**OPML export** writes each bullet's text with markup removed (colors and highlights dropped, link labels kept), its page as Markdown in `_note`, and completion as `_complete="true"`. It does not convert formatting back to Workflowy HTML; JSON and Markdown are the lossless and readable exports.

### 7.6 Sync (Phase 5)

Multi-device sync ships in Phase 5, and an iPhone app is a separate follow-up project. Nothing sync-specific is built before then; Phase 5 adds it with a schema migration.

**The planned model: per-device logs in a shared folder.** Each device appends its operations to its own file in a user-chosen sync folder (iCloud Drive, Syncthing or similar) and reads the others' files to merge. Devices never write the same file, so the file-sync tool never sees a conflict. No server is involved, and the live database never lives in the sync folder (§8.1).

**What earlier phases already provide,** because the app needs it anyway:

- Random, permanent IDs for every node (§7.2).
- History operations that are self-contained: `after` holds complete field values, not diffs, and a move carries parent and position together (§8.2).
- Soft deletes; undo and revert create new operations and never rewrite history.

**Added in Phase 5:** device IDs, hybrid logical clock timestamps on change sets, a device tie-breaker in sibling positions, per-field last-writer-wins merging, a proven tree-move algorithm for concurrent moves (Kleppmann et al.), the log format and compaction, tombstones for Forget node, and convergence tests. The open details are in §12.3.

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
  id          TEXT PRIMARY KEY,  -- random
  at          TEXT NOT NULL,     -- timestamp (§7.2)
  source      TEXT NOT NULL,     -- 'tui', 'cli', 'editor', 'import', 'restore', 'agent:<name>'
  description TEXT,              -- e.g. 'Moved "Design notes"'
  undo_of     TEXT               -- change set this one undoes (§5.6)
);
CREATE INDEX change_sets_by_at ON change_sets(at);

CREATE TABLE ops (
  id            INTEGER PRIMARY KEY,
  change_set_id TEXT NOT NULL REFERENCES change_sets(id),
  node_id       TEXT NOT NULL,
  kind          TEXT NOT NULL,   -- insert, delete, restore, edit_text, edit_page, move,
                                 -- set_task, set_heading, set_complete, set_star
  before        TEXT,            -- JSON: prior values of the changed fields
  after         TEXT             -- JSON: complete new values, never diffs
);
CREATE INDEX ops_by_node ON ops(node_id);
CREATE INDEX ops_by_change_set ON ops(change_set_id);
-- move carries parent and position together: {parent_id, position}
-- edit_page with before = null creates a page; with after = null deletes it
-- a delete made by a merge (§5.3) records the survivor: after = {deleted: true, merged_into: <id>}
```

- **Current state is authoritative;** history is a record beside it, written in the same transaction. Nothing is rebuilt by replay.
- **Before and after values** make every operation reversible by applying its inverse.
- **Change sets** group the operations of one action: a subtree move, a CLI command, an editor round trip. They are the unit of undo, revert and "what did this agent do?".
- **Coalescing:** consecutive edits to one node's text form a single operation, written once when it seals (§7.3) and never changed.
- **Append-only:** history is never rewritten, except by explicit erasure: `hyphal history purge` and Forget node (§8.4).
- **Revert** of an older change set applies its inverses to fields not changed since. Any skipped fields are reported, never overwritten.
- **History view** (Phase 2): browse a node's past versions and restore one, or list change sets by time and source.

### 8.3 Trash

- Deleting sets `deleted_at` and `deleted_by` (the deleting change set) on every node in the subtree; nothing is removed. The Trash view lists deletions, newest first.
- **Leftovers stay out of sight.** The Trash view hides two kinds of deletion whose content isn't lost: bullets merged away by Backspace or Delete (§5.3–5.3a), whose text now lives in the survivor, and a single empty leaf with no page. They are still soft-deleted, purge by age like any other deletion, and are undoable through history; `hyphal trash list --all` shows them.
- Restore brings back exactly one deletion: the nodes whose `deleted_by` matches. A child deleted separately earlier stays in the Trash.
- A restored subtree returns to its original position among its siblings when that position still exists, otherwise to the end of its parent. If the parent is gone too, it goes under the nearest surviving ancestor, else the root, and the message says where.
- Items older than 30 days are purged for real (configurable). "Empty Trash" purges everything immediately, after a one-line confirmation.
- Purging deletes the node rows with their pages, stars and view state. History ops that mention them stay until `hyphal history purge` removes them; reverting one reports the node as gone.
- After a delete the cursor moves to the next sibling, else the previous one, else the parent. The message names the bullet and counts its descendants.

### 8.4 Retention and erasure

- History is kept in full; nothing is compacted. `hyphal history purge --before <date>` removes older history when you want the space back.
- **Forget node** (Phase 4) erases a node from `nodes` and from history, and records a change set saying a node was forgotten (without its content). Snapshots still contain it until they rotate out, and the command says so.
- Expected growth is a few hundred KB per day of heavy use, about 100 MB a year. This is an estimate to verify in Phase 0.

### 8.5 Snapshots

- Taken with SQLite's online backup API: once a day when the outline has changed (at launch or quit, whichever comes first), and before every migration or restore.
- The last 14 are kept.
- Each snapshot is checked with `PRAGMA quick_check` after it is written; one that fails is deleted and reported. "Latest good snapshot" means the newest that passed.
- `hyphal snapshot list` lists them with their sizes; `hyphal snapshot restore <name>` restores one after snapshotting the current state first, so a restore is itself reversible.
- A restore never renames files over the database (with `-wal` and `-shm` files present that can corrupt it). It copies the snapshot into the live database with the online backup API. It refuses while a TUI holds the lock (§8.6). When the TUI offers a restore itself (after a failed integrity check), it closes its other connections first and reopens afterwards.

### 8.6 Locking

- One TUI per database, enforced with an exclusive `flock` on a lock file next to the database (`outline.db.lock`). It is not an SQLite lock, so CLI writers are never blocked by it, and the kernel releases it if the TUI dies.
- A second TUI opens read-only with a banner. It still follows changes through `PRAGMA data_version`; typing in it changes nothing and shows a hint.
- **Takeover.** The read-only TUI retries the lock on its `data_version` tick. Once the lock is free, its banner changes to "Editing is available · Enter to edit"; pressing Enter takes the lock and switches to normal editing. Nothing changes without that keypress.
- The CLI and `hyphal capture` don't take the lock. They are ordinary writers with a 5-second busy timeout; past it they exit with a `busy` error code (§10.2). SQLite serializes the writes.

## 9. Implementation

### 9.1 Stack

| Area | Choice | Notes |
| --- | --- | --- |
| Language | Go | Simple, fast to compile, one static binary; well suited to agent-driven development |
| UI framework | Bubble Tea v2 (`charm.land/bubbletea/v2`) | Elm-style update/view loop; v2 brings kitty keyboard protocol support and the new renderer. Released; pin the latest v2 minor in Phase 0 |
| Styling and layout | Lip Gloss v2 | Styles from the color tokens (§4.6); rendered rows cached per bullet |
| Components | Bubbles v2 (viewport, list) | Palettes and page view. The outline editor and the page edit mode are custom, sharing one wrap-layout and cursor engine (§9.3); Bubbles' text area can't share it |
| CLI | Cobra | Shared by quick capture, scripting and agents |
| Database | Pure-Go SQLite driver (`modernc.org/sqlite` or `ncruces/go-sqlite3`) | No C toolchain, so cross-builds for Linux and macOS stay simple. Confirm the online backup API when choosing |
| Text | `rivo/uniseg` | Grapheme segmentation and display width from one library, so the cursor and rendering never disagree on CJK and emoji |
| Markdown | goldmark | Page rendering, with source positions for mode switching |
| Syntax highlighting | chroma | Code blocks in page view, mapped to the ANSI colors |
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
│ In-memory tree, domain operations, undo stack, search              │
└─────────────────────────────────┬──────────────────────────────────┘
                                  │ load, commit, revert
┌─────────────────────────────────▼──────────────────────────────────┐
│ store                                                              │
│ SQLite access, history writes, snapshots, import and export        │
└──────────┬──────────────────────┬───────────────────────┬──────────┘
           │ transactions         │ backup API            │ on demand
┌──────────▼─────────┐  ┌─────────▼──────────┐  ┌─────────▼──────────┐
│ outline.db         │  │ Snapshots          │  │ Exports            │
│ Nodes, pages,      │  │ Daily and          │  │ Markdown, JSON,    │
│ history; the only  │  │ pre-migration      │  │ OPML, on demand    │
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
| Cold launch to first frame | < 100 ms | 10,000 nodes; the integrity check runs after the first frame (§8.1) |
| Keypress to frame | < 16 ms | Same, typing in a deep node |
| One action committed (state + history) | < 5 ms on Linux | Same. On macOS, real durability needs `PRAGMA fullfsync`, often 5–20 ms; commits run off the UI loop, so this never delays a frame |
| In-place filter update | < 50 ms per keystroke | 100,000 nodes |
| CLI change visible in a running TUI | < 1 s | Any |

Targets are checked with generated databases: as absolute numbers on fixed benchmark hardware, and in CI only as regressions against a stored baseline, because shared runners are too noisy for absolute thresholds.

### 9.5 Testing

- **Export round trip:** database → JSON → database is identical, for generated and real outlines (property tests with `pgregory.net/rapid`).
- **History tests:** random sequences of operations followed by undo-all restore the original state; reverting any single change set leaves all other changes intact.
- **Crash injection:** kill the process at random points during editing; the database passes `PRAGMA integrity_check` and loses at most the unsealed typing (§7.3).
- **Migration tests:** a fixture database from every past schema version migrates cleanly, with a snapshot taken first.
- **Editor round trip:** scripted edits to the temp file (rename, reorder, delete, add, malformed lines) apply exactly as expected, or not at all.
- **Import tests:** real Workflowy OPML and Markdown exports from Logseq and Obsidian import without loss.
- **Parser fuzzing** with Go's built-in fuzzing for the inline-markup parser, the list importer and OPML.
- **UI tests:** Bubble Tea's `teatest` drives scripted key sequences and compares the rendered screen with golden files.
- **Interaction matrix:** scripted keystroke tests for every row of the Enter, Backspace and Delete tables (§5.2–5.3a) and every binding in §6.4.
- **CLI tests:** every command's `--json` output is checked against golden files.

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

- **Node links (Phase 2):** type `[[` to search and link to any bullet. Links are stored by node ID, so they survive moves and renames, and they display the target's current text. `o` in block selection (or a click) zooms to its target. Markdown export writes Obsidian-style links.
- **Backlinks (Phase 3):** an on-demand floating overlay listing every bullet that links to the current node, each with its breadcrumb.
- **Saved searches (Phase 3):** "Save search" in the command palette stores the current filter query under a name (`Open bugs` = `#bug is:open`). Saved searches are listed in the jump palette next to starred locations; choosing one opens the filter with that query. They are stored in their own small table and included in JSON export.
- **Templates (Phase 3):** any subtree tagged `#template` can be instantiated under the current node with one command, from the command palette.
- Graph view and plugins remain non-goals.

### 10.2 CLI and agent interface

A robust CLI ships in **Phase 1**, for users, scripts and AI agents alike. It is a thin layer over the same `core` operations as the TUI, and it doubles as the build agents' test harness (§9.6). Phase 4 adds `forget`, batch `apply` and agent conveniences on top.

Every `<node>` argument accepts an ID or a path.

```bash
hyphal add "Fix tooltip positioning" [--under <node>] [--after <node> | --before <node>]
hyphal capture [--window]             # one-line capture prompt (§4.7)
hyphal show <node> [--depth N] --json
hyphal tree [--root <node>] --json
hyphal search "tooltip #bug is:open" --json
hyphal edit <node> --text "New text" [--if-version N]
hyphal set <node> [--task | --no-task] [--heading | --no-heading]
hyphal move <node> --under <node> [--after <node> | --before <node> | --index N]
hyphal complete <node>      # and: reopen, star, unstar
hyphal delete <node>        # to Trash
hyphal forget <node>        # Phase 4: erase for good (§8.4); asks for --yes
hyphal trash list [--all]|restore <id>|empty   # <id>: the deleted subtree's top node
hyphal page get <node>      # Markdown to stdout
hyphal page set <node> --file page.md [--if-version N]   # or from stdin
hyphal page delete <node>
hyphal apply --file ops.json          # Phase 4: many operations as one change set
hyphal apply --schema                 # Phase 4: prints the ops.json JSON Schema
hyphal history list [--source agent:claude] --json
hyphal history purge --before <date>
hyphal revert <change-set-id>
hyphal export --format markdown|json|opml [--out <dir>] [--ids] [--history] [--split] [--scheduled]
hyphal import --format json|opml|markdown <file> [--under <node>] [--replace]
hyphal snapshot list|restore <name>
hyphal check                # integrity check (quick_check plus tree invariants)
hyphal paths
```

- Core verbs on nodes are top-level; everything else is grouped by noun (`trash`, `page`, `history`, `snapshot`, and `sync` in Phase 5). Tag commands arrive with tags in Phase 2.
- **Paths** are breadcrumb paths from Home, with `/` between segments: `Projects/ControllerCraft`. A literal `/` in a bullet's text is written `\/` (`Projects/2026\/10 review`). Segments match the bullet's text with markup stripped, exactly first, then case-insensitively. If a segment matches several siblings, the command fails with an `ambiguous` error listing the candidates and their IDs. Path lookups return the ID for later calls.
- **`ops.json`** (Phase 4) is a JSON array of operations, each `{"op": "<command>", ...arguments}`, using the same names and arguments as the commands above (`add`, `edit`, `set`, `move`, `complete`, `delete`, `page_set`, …). An operation can refer to a node added earlier in the batch by a temporary `"ref"`. `hyphal apply --schema` prints the exact schema.
- **Output:** every command supports `--json`, with a stable, versioned envelope: `{"ok": true, "data": …}` or `{"ok": false, "error": {"code": "…", "message": "…", "candidates": [...]}}`. Human-readable output is the default. Export without `--out` writes a single-file format to stdout.
- **Exit statuses:** 0 success, 1 other error, 2 usage error, 3 `not_found` or `ambiguous`, 4 `conflict` (`--if-version` mismatch), 5 `busy` (locked past the 5-second busy timeout, §8.6), 6 `read_only` (newer schema or failed integrity check).
- Every write is one change set tagged with its source: `cli` by default, `--source agent:<name>` for agents, so their changes can be reviewed and reverted as a unit. The source is self-declared: it is for attribution, not a security boundary.
- `hyphal apply` (Phase 4) lets an agent submit a batch of operations atomically; either all apply or none do.
- Writes are safe while the TUI is open; the TUI picks them up within a second (§7.3).
- The database is always explicit: `--db`, then the `HYPHAL_DB` environment variable, then the config default.
- **Phase 4:** `forget`, `apply`, a shipped usage guide for agents, and richer query output. There is no MCP server: agents drive the CLI directly.

## 11. Staged delivery plan

Phase 1 delivers the core of the Workflowy clone (every Phase 1 *Must* row), usable daily on your own imported outline. Phase 2 completes Workflowy parity, Phase 3 adds the Obsidian-inspired features, and agent integration and sync come last.

Six phases, each with an exit gate:

| Phase | Scope | Exit gate |
|---|---|---|
| **0 · Storage and editing spike** | Schema, history, JSON round trip; wrap layout, keymap matrix | Export and import are identical; a crash loses at most the unsealed typing |
| **1 · Minimum lovable Workflowy clone** | Editing rules, block selection, pages, formatting; theme matching, palettes, CLI, Workflowy import | A week of daily use on your own outline; side-by-side calm review passes |
| **2 · Full Workflowy parity** | Tags, node links, search operators, stars; jump palette, history, mouse, page edit mode | Every *Must* row of the parity checklist works |
| **3 · Obsidian-inspired features** | Backlinks, saved searches, templates | Links survive moves and renames and export cleanly |
| **4 · Agent integration** | Batch `apply`, `forget`, agent usage guide, richer queries | Every agent change is reviewable and revertible |
| **5 · Multi-device sync** | Device IDs, HLC merge, per-device logs, compaction; any-folder sync (iCloud Drive, Syncthing) | Two machines edit offline and converge with nothing lost |

Each phase starts only after the previous gate passes.

**Separate follow-up project:** an iPhone app that syncs through the same per-device log format. It is out of scope here; this project only guarantees the format is documented and stable enough to build it.

Future and exploratory, with no phase: direct Wi-Fi sync, collaboration, graph view, plugins.

## 12. Non-goals, decisions and next steps

### 12.1 Non-goals

- No editing of the data outside the app. Exports are copies; the `$EDITOR` round trip is the supported way to edit with another tool.
- No live sync through Git, Syncthing or cloud folders.
- No `@person` mentions and no board (kanban) view in V1.
- No mirrors (one bullet shown live in two places). Every node has exactly one parent; links, tags and saved searches cover the need.
- No daily journal or Today node, and no page-based model: everything lives in the one tree, and a page (§5.7) is an optional attachment to a bullet.
- No Vim-style modal editing or Vim motions.
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
| 7 | Backup | Daily and pre-migration snapshots, last 14 kept; export on demand or on schedule | §8.5, §7.5 |
| 8 | Editor ergonomics | Modeless + Esc block selection; Enter/Backspace tables | §5 |
| 9 | Formatting | Markdown-style markers, concealed when not editing | §5.8 |
| 10 | Text color | `<red>text</red>` with theme-mapped color names; `==highlight==` | §5.8 |
| 11 | Tags | `#tag` with nested `/` tags; no `@person` | §5.9 |
| 12 | Theme | The terminal's 16 ANSI colors only; no theme files read; OSC 11 for light or dark | §4.4 |
| 13 | Tasks | `is_task` plus `completed_at`; exported as checkboxes | §7.2 |
| 14 | Search | In-memory substring scan, no index; filter in P1, jump palette and operators in P2 | §6.2 |
| 15 | External editing | Temp-file round trip only | §6.3 |
| 16 | Configuration | XDG paths, TOML, explicit `--db` | §7.1 |
| 17 | Long-form content | Bullets stay one line; an optional Markdown page per bullet, opened with page zoom. Replaces notes | §5.7 |
| 18 | Language and UI stack | Go with Bubble Tea, Lip Gloss and Bubbles; pure-Go SQLite | §9.1 |
| 19 | Platforms | Linux (Omarchy first) and macOS; no Windows | §6.4, §7.1 |
| 20 | Sync | Nothing sync-specific before Phase 5; then per-device logs in a shared folder, no server | §7.6 |
| 21 | CLI | Full CLI with JSON output in Phase 1; batch `apply` and `forget` in Phase 4; no MCP server | §10.2 |
| 22 | Builders | Built entirely by AI agents from a phased plan; every behavior has an automated test | §9.6 |
| 23 | App name | Only in user-facing places; never in schema, ops, sync log or JSON fields | §1 |
| 24 | Root and Inbox | A fixed Home root; a top-level Inbox found by ID | §4.1, §6.3 |
| 25 | Keymap | One authoritative table per context; every command has a block-selection key; no Alt commands; same keys on Linux and macOS; Esc never zooms out; typing never leaves block selection | §6.4 |
| 26 | Overlays | Backlinks and palettes are floating overlays; no side panes | §4.7 |
| 27 | Undo scope | Per-session stack of TUI-started changes, including `$EDITOR` round trips; CLI and agent changes are reverted | §5.6 |
| 28 | Identity | Split keeps the ID on the first half; merges keep the earlier bullet and retarget stars and links | §5.2–5.3a |
| 29 | Concurrency | Per-node versions; TUI typing is never dropped; `--if-version` for agents | §7.3 |
| 30 | Trash restore | Per deletion, to the original position, else the nearest surviving ancestor (Trash and sync alike) | §8.3 |
| 31 | History policy | Append-only, ops written once when sealed; no compaction; manual purge | §8.2, §8.4 |
| 32 | Sync moves | Moves carry parent and position together; a proven tree-move algorithm in Phase 5 | §7.6 |
| 33 | Markdown | One GFM-style dialect for export, import, paste and the editor round trip | §7.5 |
| 34 | CLI shape | Core verbs top-level, the rest grouped by noun; `--json` envelope and fixed exit codes | §10.2 |
| 35 | Look | Comfortable density, no indent guides by default; semantic color tokens | §4.6 |
| 36 | Mirrors | Never; the outline stays a strict tree, and links, tags and saved searches cover multi-context items | §12.1 |
| 37 | Cut and paste | Pasting the app's own last cut moves the bullets intact (IDs, pages, history) | §5.5 |
| 38 | Page preview | None in the outline; zoom into the page to read it | §5.7 |
| 39 | Trash contents | Merge leftovers and empty leaves are hidden from the Trash view | §8.3 |
| 40 | Block-selection keys | Every bullet command has one single key, no aliases: `[`/`]` move, `D` duplicate, `h` show/hide completed, `0` expand all, `~` zoom to root | §6.4 |
| 41 | Markdown export layout | One `outline.md` plus `pages/` by default; `--split` for one file per top-level node | §7.5 |
| 42 | Feedback | No toasts: one footer row shows a message (until the next key), the filter query, or key hints | §4.5 |
| 43 | Leading markers | `[ ] `, `[x] ` and `## ` at the start of a bullet always convert, so the dialect needs no escapes | §5.4, §7.5 |
| 44 | Clipboard | Local platform tool first; OSC 52 over SSH or when no tool exists | §5.5 |
| 45 | Name | Hyphal; `hyphal` for the binary, directories, window class and `HYPHAL_DB` | §1 |
| 46 | Simplification pass | Cut: sync groundwork before Phase 5, Omarchy theme files, Nerd Font glyphs, history compaction, hourly snapshots, properties, saved-query virtual children, MCP server, OPML formatting export, desktop notifications, persistent undo, FTS5, `node_tags`, in-place history rows, incremental reload | §4–§11 |
| 47 | Chords | Text-editing chords limited to the Workflowy core; Ctrl+Enter is the only kitty-only key; Alt only for word editing | §6.4 |

### 12.3 Still open

- **Color palette size.** Eight named colors is the starting set; confirm against Omarchy themes that each stays legible on light and dark backgrounds.
- **History growth.** History is kept in full; measure real growth in Phase 0 and add a retention default only if it is needed.
- **Sync details** (Phase 5, §7.6): HLC drift handling, log compaction thresholds, and whether page text later merges line by line.

**Open questions by phase.** Each must be settled before its phase starts, as part of that phase's plan.

- **Phase 2**
  - *Tags (§5.9):* how `#` is told apart from Markdown headings in pages, issue numbers (`#123`), `C#`, URL fragments and code; Unicode letters; numeric-only tags; a trailing `/`; display case versus stored lowercase; a rename command; colors repeating with only eight palette colors.
  - *Node links (§10.1):* the stored form of `[[…]]` and how it stays readable in a SQLite browser; Obsidian export (links target files, not nodes); links to trashed or purged nodes; the format of Copy link to node.
  - *Screens:* History view, Stars view, tag list, jump palette result rows; mouse details.
- **Phase 3**
  - *Templates:* stripping `#template` from instances; variables such as the date.
- **Phase 5**
  - *Sync view* layout and keys.

### 12.4 Immediate next steps

1. **Schema spike:** nodes, pages, history and typing coalescing in Go with the chosen SQLite driver. Simulate a week of heavy editing and measure database growth.
2. **Safety tests:** the JSON export round trip, crash injection during typing, and snapshot restore on a clean machine.
3. **Wrap-layout and rendering prototype:** one Bubble Tea outline view with grapheme-aware cursor movement, concealed inline formatting, and the centered reading column.
4. **Reference mockups:** the five screens listed in §4.6, reviewed side by side with Workflowy and with lazygit and btop under three Omarchy themes, one of them light.
5. **Keymap matrix:** record which bindings arrive in each terminal, bare and in tmux, before finalizing §6.4, including how Option+Left/Right/Backspace and Cmd+Left/Right arrive on macOS and that macOS leaves Ctrl+Shift+Up/Down alone.
6. **Workflowy import:** run a real Workflowy OPML export through the importer and list anything lost, including how mirrors appear and whether they can be detected.
7. **Stack checks:** confirm Bubble Tea v2's keyboard-protocol support in the terminal matrix and that `teatest` (or a v2 equivalent) works with v2, and choose the pure-Go SQLite driver after verifying the online backup API; measure fuzzy matching and the in-memory filter over 100,000 nodes.
8. **Implementation plan:** turn this spec into the phased, agent-ready plan, with acceptance criteria per feature (§9.6).

## 13. Configuration reference

`config.toml` (§7.1) is optional; every key has a default. "Edit settings" opens it commented, listing every key below (§4.7).

| Key | Default | Meaning | Section |
| --- | --- | --- | --- |
| `db` | `<data dir>/outline.db` | Default database | §7.1 |
| `column_width` | `90` | Reading column cap, in columns | §4.6 |
| `density` | `"comfortable"` | `comfortable` or `compact` | §4.6 |
| `indent_guides` | `false` | Faint `│` guides | §4.3 |
| `glyphs` | `"auto"` | `auto` (Unicode unless the locale isn't UTF-8), `unicode` or `ascii` | §4.6 |
| `background` | `"auto"` | `auto` (ask the terminal), `light` or `dark` | §4.4 |
| `footer` | `true` | Footer key hints | §4.5 |
| `split_view_min_width` | `0` (off) | Open pages in split view at or above this width | §5.7 |
| `zoom_transition` | `false` | Brief zoom animation | §4.6 |
| `[tag_colors]` | — | Per-tag color overrides | §5.9 |
| `mouse` | `true` | Mouse capture (Phase 2) | §6.5 |
| `[keys.<context>]` | — | Key overrides per context | §6.4 |
| `trash_days` | `30` | Days before the Trash purges | §8.3 |
| `snapshots.keep` | `14` | Daily snapshots kept | §8.5 |
| `export.format` | `"none"` | Scheduled export: `none`, `markdown` or `json` | §7.5 |
| `export.when` | `"daily"` | `daily` or `quit` | §7.5 |
| `export.dir` | `<data dir>/exports` | Scheduled export folder | §7.5 |
| `export.keep` | `10` | Scheduled exports kept | §7.5 |
| `capture.terminal` | Omarchy default terminal via `xdg-terminal-exec` | Command template for `hyphal capture --window` | §4.7 |

- **Invalid config:** at launch, invalid keys fall back to their defaults and a banner names the file and line. On reload after "Edit settings", an invalid file is ignored entirely: the previous config stays in effect, with the same banner. Unknown keys produce a warning, not an error.
