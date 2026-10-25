# Phase 3 · Obsidian-inspired features

Spec: §11 row 3, §10.1. Scope: backlinks, saved searches, templates.

Exit gate (automated, P3-06): **backlinks list every linking bullet and page; a saved search reopens with the same results; a template instantiates with new IDs.**

## Dependencies

```text
P3-01 templates decision ─▶ P3-05 templates
P3-02 backlinks core ─▶ P3-03 backlinks overlay
P3-04 saved searches (schema v2)
P3-06 gate
```

### P3-01 · Decide: templates
Type: decision · Spec: §10.1 (templates), §12.3 (Phase 3)

Write `plan/proposals/p3-01-templates.md`:
1. Whether `#template` is stripped from the instance's top bullet (and from descendants that carry it).
2. Variables (`{{date}}`, `{{time}}`, `{{title}}` prompt?) with syntax that the markup grammar leaves literal, and the date format (local time, §7.2).
3. Choosing a template: the palette command "Insert template…" lists subtrees tagged `#template` (the top tagged bullet is the template; its children are the content? or the whole subtree including the top?). Placement: as children of the current node, or as next siblings.
4. Pages in templates: copied (like Duplicate).

Stop for the human; afterwards write D-entries and update P3-05.

### P3-02 · Backlinks index
Type: impl · Packages: `internal/core` · Depends on: P2-07 · Spec: §10.1 (backlinks), §5.7 ("pages appear in backlinks")
Deliverables: `Backlinks(tree, id) []Backlink{Source NodeID, InPage bool, Snippet string}` covering links in bullet text and page text; incremental maintenance on edits; excludes trashed sources.
Acceptance criteria:
- AC1 Links from bullets and pages are both listed → `TestBacklinks_BulletsAndPages`.
- AC2 Editing a source to remove the link updates the index → `TestBacklinks_Update`.
- AC3 Trashed sources excluded; restored ones return → `TestBacklinks_Trash`.

### P3-03 · Backlinks overlay
Type: impl · Packages: `internal/tui`, `internal/keymap` · Depends on: P3-02 · Spec: §10.1, §4.7 ("Backlinks is a floating overlay"), §4.6 (overlay look) · Human review: yes (new key binding)
Deliverables: palette command "Backlinks" (plus a block-selection key proposed in the PR, satisfying §6.4 rules — `b` is unavailable by the Alt rule (§6.4: `b`, `f`, Backspace do nothing in block selection), so propose `l` or palette-only); floating overlay listing linking bullets with breadcrumbs and page snippets; Enter zooms to the source (page sources open the page in view mode scrolled to the link); Esc closes.
Acceptance criteria:
- AC1 Overlay lists every linking bullet and page with breadcrumb → `TestBacklinksOverlay` (golden).
- AC2 Enter zooms to the source → `TestBacklinksOverlay_Enter`.

### P3-04 · Saved searches
Type: impl · Packages: `internal/store`, `internal/core`, `internal/tui`, `internal/cli` · Depends on: P2-09, P2-12 · Spec: §10.1 (saved searches), §7.4, §7.5 · Human review: yes (schema, JSON export format)
Deliverables:
- Schema v2 migration: `CREATE TABLE saved_searches (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, query TEXT NOT NULL, created_at TEXT NOT NULL, position TEXT NOT NULL)`. Snapshot before migrating (§7.4). Changes to saved searches are user data: record them in history as a new op family? The spec doesn't say; propose in the PR. Default: no history, like view state, but included in JSON export (§10.1).
- JSON export `format_version` 2 with `saved_searches`; import accepts 1 and 2 (CONTRACTS §6).
- Palette "Save search" (asks for a name in an overlay input, prefilled from the query) while a filter is active; saved searches listed in the jump palette next to starred locations; choosing one opens the filter with that query; palette "Delete saved search".
- CLI: none in the spec; propose `hyphal search --saved NAME` in the PR as a `[QUESTION]` if wanted.
Acceptance criteria:
- AC1 Migration v1→v2 on a v1 fixture DB, snapshot first → `TestMigrate_V2` (fixture in `internal/store/testdata/legacy/v1.db`, committed by this card).
- AC2 Saved search reopens with the same results → `TestSavedSearch_Reopen`.
- AC3 JSON round trip includes saved searches; v1 files still import → `TestJSONRoundTrip_V2`, `TestImportJSON_V1`.

### P3-05 · Templates
Type: impl · Packages: `internal/core`, `internal/tui` · Depends on: P3-01, P2-05 · Spec: §10.1 (templates)
Deliverables per P3-01. Minimum ACs:
- AC1 Instantiating a template creates new IDs, copies pages, one change set → `TestTemplate_Instantiate`.
- AC2 Variables expand per P3-01 → `TestTemplate_Variables`.

### P3-06 · Phase 3 gate
Type: gate · Depends on: P3-02…P3-05
Deliverables: `plan/reports/phase-3-gate.md` mapping the three gate criteria to `TestBacklinksOverlay`, `TestSavedSearch_Reopen`, `TestTemplate_Instantiate`. Human sign-off line.
