# Testing

Implements §9.5 and §9.6: every behavior is checkable by `go test`; no manual verification is an acceptance criterion.

## 1. Commands

| Make target | Runs |
| --- | --- |
| `make test` | `go test -race -count=1 ./...` |
| `make check` | gofmt check, `go vet`, `golangci-lint run`, `make test`, `make fuzz-short` |
| `make fuzz-short` | each `Fuzz*` target for 10 s (`-fuzztime=10s`) |
| `make fuzz-long` | each `Fuzz*` target for 10 min (nightly CI) |
| `make bench` | `go test -run=^$ -bench=. -benchmem ./...` with generated DBs |
| `make golden-update` | `go test ./... -update` (only when a card changes rendering; explain each change) |
| `make crash` | crash-injection suite (`-tags crash`, ~2 min) |

## 2. CI (GitHub Actions, P0-01)

- `ci.yml` on every PR and push to `main`: matrix `ubuntu-latest`, `macos-latest`; Go from `go.mod`; `make check`; `make crash` on Linux only.
- `nightly.yml`: `make fuzz-long`, `make bench` compared to the stored baseline (`testdata/bench/baseline.txt`) with `benchstat`; fail if any tracked benchmark regresses > 20 % with p < 0.05 (§9.4). Runs on schedule and on manual dispatch.
- Release workflow (P1-50): GoReleaser on tags.

## 3. Test layers

| Layer | Where | Style |
| --- | --- | --- |
| Leaf unit + fuzz | each leaf package | Table tests; `Fuzz*` for markup, dialect, opml, fracindex, layout (§9.5) |
| Store | `internal/store` | Real SQLite in `t.TempDir()`; migration fixtures in `internal/store/testdata/legacy/` |
| Property | `internal/store`, `internal/core` | `rapid`: JSON round trip, undo-all, revert isolation (§9.5) |
| Core rules | `internal/core` (`core_test`) | Fixture notation (CONTRACTS §11): `before`, action, `after`, `hint` |
| CLI | `internal/cli` | Run the Cobra root in-process with args; compare `--json` output to goldens in `internal/cli/testdata/golden/`. Timestamps and IDs normalized (§6 below) |
| TUI model | `internal/tui` | `tuitest.Harness`: keys in, assert notation + footer + banner (§4) |
| TUI screens | `internal/tui` | `teatest` v2 full program; golden screens in `internal/tui/testdata/golden/*.txt` (ANSI stripped) and `*.ansi` (with styles) |
| Crash | `internal/store/crashtest` (build tag `crash`) | Subprocess killed with SIGKILL (§5 below) |
| Bench | packages + `internal/testgen` | Generated DBs: 10k and 100k nodes |
| Arch | `internal/archtest` | Import rules (ARCHITECTURE §2), app-name placement scan (CONTRACTS §1) |

### Core rule test shape

```go
func TestEnter(t *testing.T) {
    cases := []struct{ name, before, after, hint string }{
        {"start of bullet inserts above",
         "- ‸Alpha\n- Beta",
         "- \n- ‸Alpha\n- Beta", ""},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            f := testgen.MustParseOutline(t, c.before)
            p := core.PlanEnter(f.Ctx(), f.Cursor(), f.Selection())
            got := f.ApplyAndFormat(t, p) // applies to tree, sets cursor from p.Cursor
            testgen.Equal(t, c.after, got); testgen.Equal(t, c.hint, p.Hint)
        })
    }
}
```

Use only the stdlib `testing` package plus small helpers in `testgen` (`testgen.Equal`, `testgen.Golden`); no assertion library dependency.

## 4. `tuitest.Harness`

```go
h := tuitest.New(t, tuitest.Options{
    Outline: "- Alpha‸\n- Beta",   // fixture notation
    Width: 100, Height: 30,          // default 100×30
    Background: "dark",              // no OSC 11 reply simulation; "light" to force
    Glyphs: "unicode",
    Clock: tuitest.FixedClock("2026-10-24T09:00:00Z"),
    Config: "",                      // TOML text
    Kitty: false,                    // whether ctrl+enter can arrive
})
h.Keys("enter", "tab", "H", "i")     // key strings as Bubble Tea v2 prints them; "type:hello" types runes
h.Paste("line one\n  line two")
h.Advance(3 * time.Second)           // fake clock; fires ticks → typing seals
h.AssertOutline("- Alpha\n  - Hi‸\n- Beta")
h.AssertFooter("Enter new · Tab indent · Esc select · Ctrl+P commands · F1 help")
h.AssertMessage("Moved \"Alpha\" to Trash · Ctrl+Z undoes")
h.AssertBanner("")                   // none
h.AssertState(tuitest.TextEditing)   // TextEditing, BlockSelection, FilterInput, PageView, Overlay, Trash
h.Screen()                           // rendered text (ANSI stripped) for ad-hoc checks
h.DB()                               // *store.DB to assert history rows
```

- The harness uses a temp DB, a synchronous committer, a fake clock, a fake clipboard (records writes, returns configurable reads), a fake `$EDITOR` (`h.Editor(func(path string) error)` edits the temp file), and a fake URL opener.
- It drives `Model.Update` directly with constructed messages; it does not need a PTY.
- `teatest` is used only for full-screen goldens and startup/terminal behavior (OSC 11 reply parsing, focus events, resize).

## 5. Crash injection (§9.5, Phase 0 gate)

- `internal/store/crashtest` builds a helper binary (`go build` in `TestMain`) that opens a DB, then loops: random structural actions via `core` (synchronous commits) and typing bursts through `Outline.TypeInto` with a fake clock that seals per the rules. After each commit it appends the change set ID and the expected durable state hash to a side file (`fsync`ed).
- The parent kills the child with SIGKILL at random times (uniform 5–500 ms) for N=200 iterations (CI) / 2,000 (nightly).
- After each kill: `PRAGMA integrity_check` = `ok`; `hyphal check` invariants pass; every change set recorded in the side file as committed is present; unsealed typing is the only possible loss (the DB state equals the state after the last committed change set).

## 6. Golden files

- Location: `testdata/golden/` next to the test. Names: `<TestName>.golden` (`.json`, `.txt`, `.ansi`).
- Normalization before comparison: IDs → `<id1>`, `<id2>`… in order of appearance; timestamps → `<ts>`; durations → `<dur>`; temp paths → `<tmp>`; version string → `<version>`.
- Update with `-update` only in cards that change the output; list every changed golden with a one-line reason in the PR (§9.6).
- Screen goldens: fixed 100×30 unless the test is about size; also run the light-theme and ASCII variants where the card says.

## 7. Generators (`internal/testgen`)

- `testgen.Outline(r *rapid.T | *rand.Rand, o OutlineOpts) *store.Dump`: random trees (depth up to 12, fan-out skewed), random markup in text (using the markup grammar), pages (10 %), stars, collapsed state, trashed subtrees, completion, tasks, headings, unicode (CJK, emoji, combining marks, RTL).
- `testgen.Actions(...)`: random sequences of planner calls valid for the current tree (for undo/revert properties and crash tests).
- `testgen.BenchDB(tb, n int) string`: builds and caches (under `$TMPDIR/hyphal-bench/`) a DB with n nodes for benchmarks.
- `testgen.ParseOutline` / `FormatOutline`: fixture notation (CONTRACTS §11).

## 8. Interaction matrix (§9.5)

Every row of the §5.2, §5.3 and §5.3a tables and every binding in §6.4 has a named test. P1-55 adds `internal/tui/matrix_test.go` that enumerates `keymap.Defaults()` and fails if any binding lacks a test tagged with its command ID (`// covers: toggle_complete@block`). Tests carry such tags in a comment; the meta-test greps the test files.

## 9. Performance (§9.4)

| Benchmark | Package | Target (fixed hardware) |
| --- | --- | --- |
| `BenchmarkColdLaunch10k` (open + LoadAll + first View) | tui | < 100 ms |
| `BenchmarkKeypressDeepNode10k` (Update + View for a typed rune) | tui | < 16 ms |
| `BenchmarkCommitAction10k` | store | < 5 ms (Linux) |
| `BenchmarkFilter100k` (one keystroke) | core | < 50 ms |
| `BenchmarkCLIChangeVisible` (CLI commit → TUI poll sees it) | tui | < 1 s |
| `BenchmarkCLILoad100k` (CLI open + LoadAll) | core | measured in P0-20; target set in report |

Absolute targets are checked by the human on fixed hardware (`make bench` prints a table); CI checks only regressions against the baseline (§9.4).
