# Hyphal implementation plan — agent operating manual

Audience: AI coding agents that build Hyphal. Humans are not the intended readers. Be literal: when a card says "must", it is an acceptance criterion; when it says "suggested", you may deviate if you record why in the PR.

## 1. Documents and precedence

| File | Contains | Load when |
| --- | --- | --- |
| `SPEC.md` | Product behavior. Section numbers (§) are stable and cited everywhere | Always, for the § cited by your card |
| `plan/README.md` | This manual: workflow, Definition of Done, gap protocol | Always |
| `plan/ARCHITECTURE.md` | Repo layout, packages, import rules, Go interfaces, runtime model | Always |
| `plan/CONTRACTS.md` | Wire and data formats: schema, IDs, positions, op payloads, JSON export, CLI envelope, error codes, test notation | When your card touches data, storage, CLI or tests |
| `plan/TESTING.md` | Test strategy, harnesses, golden files, generators, benchmarks, CI | Always |
| `plan/DECISIONS.md` | Gap resolutions made by the planner (D-xxx) and by agents (A-xxx) | Always (it is short); append to it per §5 below |
| `plan/phase-N.md` | Task cards for phase N | Your current phase only |
| `plan/reports/` | Measurement and gate reports written by agents | When a card asks |

Precedence when two documents disagree:

1. A `DECISIONS.md` entry that explicitly names the conflicting text wins.
2. `SPEC.md` wins for behavior.
3. `CONTRACTS.md` wins for formats and names.
4. `ARCHITECTURE.md` wins for structure.
5. The task card wins for scope.

If the conflict is not covered by rule 1, follow the gap protocol (§5).

## 2. Phase rules

- Phases run in order 0 → 5. A phase's cards may start only after the previous phase's gate card is merged **and** its human gate (if any) is signed off in `plan/reports/phase-N-gate.md` by the human (a line `Signed off: <date>` written by the human, never by an agent).
- Phases 2–4 begin with **Decision cards** (type `decision`). Their output is a proposal in `plan/proposals/`; the human picks options; the agent then writes the outcome into `DECISIONS.md` and updates the phase file's affected cards. No implementation card that depends on a decision card may start before that.
- Phase 5 has only an outline. Its first card is to write the full Phase 5 plan.

## 3. Task card format

```
### P1-12 · Enter rules engine
Type: impl | spike | decision | gate | human
Packages: internal/core            ← the only packages you may change (plus tests/testdata)
Depends on: P1-07, P0-10           ← all must be merged first
Parallel-safe with: P1-13          ← optional hint
Spec: §5.2, §5.4                   ← read these sections in full before starting
Human review: no | yes             ← yes = do not self-merge (§4)

Goal: one paragraph.
Deliverables: files, exported identifiers, commands.
Notes: algorithms, pitfalls, decisions already made.
Acceptance criteria:
  AC1 <observable behavior>  → test: TestName (package)
  AC2 ...
Out of scope: things that belong to other cards.
```

Every AC must map to at least one automated test whose name is listed in the PR body. "No manual verification" (§9.6): if you believe an AC cannot be automated, stop and ask (§5).

Card types:

- `impl`: production code and tests.
- `spike`: measurement or evaluation; output is a report in `plan/reports/` plus any reusable code (benchmarks, generators). Spikes still merge code through PRs.
- `decision`: write a proposal with options and a recommendation; stop for the human.
- `gate`: verify a phase exit gate with automated checks and a report; stop for human sign-off where the spec requires a human checkpoint.
- `human`: work only a human can do (real terminals, a real Workflowy export, daily use). The agent prepares tooling and instructions, then stops.

## 4. Workflow per card

1. **Pick** a card whose dependencies are all merged. Find merged cards with `gh pr list --state merged --search "in:title [P1-" --limit 200`. A card is taken if an open PR or a branch `card/<id>-*` exists; pick another.
2. **Branch** from up-to-date `main`: `card/p1-12-enter-rules` (lowercase card ID + slug). One card per branch. Use a git worktree when running in parallel with other agents.
3. **Read** the cited spec sections in full, `ARCHITECTURE.md`, `TESTING.md`, the relevant `CONTRACTS.md` sections and `DECISIONS.md`.
4. **Write tests first** for each AC where practical. Name tests after behavior (`TestEnter_EmptyNestedBulletOutdents`).
5. **Implement** within the card's packages. If you must change another package (for example a missing accessor in `core` while working in `tui`), keep it minimal, additive and listed in the PR as "cross-package change". Never change a contract in `CONTRACTS.md` without `Human review: yes`.
6. **Run locally**: `make check` (gofmt, go vet, golangci-lint, `go test -race ./...`, short fuzz). All must pass.
7. **Commit** in small logical commits. Message format:
   ```
   P1-12: Implement Enter rules (§5.2)

   <what and why, 1–5 lines>

   Co-Authored-By: <your attribution line, if your environment requires one>
   ```
8. **Open a PR** titled `[P1-12] Enter rules engine`. Body template:
   ```
   Card: P1-12   Spec: §5.2, §5.4
   ## Acceptance criteria
   - [x] AC1 … → TestEnter_StartOfBulletInsertsAbove
   ## Decisions
   - A-017 (new): … / none
   ## Golden changes
   - testdata/golden/…: why it changed / none
   ## Cross-package changes
   - none
   ```
9. **Merge**: squash-merge yourself once CI is green, unless the card says `Human review: yes` or the PR adds/changes golden screens for the calm review, the schema, a `CONTRACTS.md` format, or a dependency in `go.mod`. In those cases leave the PR open, request review, and pick another card.
10. Delete the branch after merge.

### Definition of Done (every card)

- All ACs have passing tests named in the PR.
- `make check` passes; CI passes on Linux and macOS.
- No `TODO` without a card ID (`// TODO(P2-05): …`).
- Exported identifiers have doc comments that cite the spec section where behavior is defined (`// Split implements Enter mid-text (§5.2).`).
- New or changed goldens are explained in the PR (§9.6: "golden files are reviewed, not regenerated blindly").
- `DECISIONS.md` updated if you made a gap decision.
- No new dependency unless the card or `ARCHITECTURE.md §3` lists it; otherwise ask.

## 5. Spec-gap protocol

When the spec is silent, ambiguous or self-contradictory:

1. Check `DECISIONS.md`. If an entry covers it, follow it.
2. Classify the gap.
   - **Stop and ask the human** if the gap changes: a *Must* row in the parity checklist (§3), the database schema, an op payload, the JSON export format, the CLI command surface/flags/envelope/exit codes, or a default key binding.
   - **Decide and continue** otherwise: file names, wording of messages, internal APIs, edge cases not covered by the spec's tables, performance tactics.
3. When you decide, choose the option closest to the product principles (§2 of the spec), prefer the Workflowy behavior when one exists, and prefer the option that loses no data.
4. Append an entry to `DECISIONS.md`:
   ```
   ### A-017 · Short title (P1-12, 2026-11-03)
   Context: <spec § and the gap>
   Decision: <what you did>
   Alternatives: <briefly>
   ```
   Use the next free `A-` number. On a merge conflict in `DECISIONS.md`, renumber yours.
5. Cite the entry ID in a code comment at the place it applies and in the PR body.

To ask the human: stop work on the card, push your branch, open a draft PR titled `[P1-12][QUESTION] …` whose body states the question, the options, and your recommendation. Then pick another card.

## 6. Human checkpoints (never agent acceptance criteria)

From §9.6 and §11. Agents prepare tooling and stop:

| Checkpoint | Phase | Agent's part |
| --- | --- | --- |
| Keymap matrix in real terminals | 0 | Build `hyphal-keyprobe`, write instructions, ingest the recorded JSON into a report |
| Comparison-set evaluation (§3) | 0–1 | Write an evaluation checklist in `plan/reports/` |
| Reference mockups + calm review (§4.6) | 1 | Produce the five golden screens and a viewing script |
| Real Workflowy OPML import (§12.4.6) | 1 | Importer + `hyphal import --dry-run`-style loss report; the human supplies the file at `testdata/private/` (gitignored) or a redacted copy for `testdata/opml/` |
| A week of daily use (§11) | 1 | Release build, a feedback template in `plan/reports/phase-1-dogfood.md` |

## 7. General rules for agents

- Never touch the user's real data directory. Tests use `t.TempDir()`. The test harness sets `XDG_DATA_HOME`, `XDG_CONFIG_HOME`, `HOME` and `HYPHAL_DB` to temp paths.
- Never write the app name into the schema, op kinds, JSON field names or sync formats (§1). The only allowed places are listed in `CONTRACTS.md §1`.
- Never add network access. No telemetry, no update checks.
- Do not regenerate goldens with `-update` unless the card changes rendering; explain every changed golden.
- Keep `go.mod` tidy; pin exact versions.
- Prefer clarity over cleverness: the code will be maintained by other agents.
- If a card turns out to be larger than ~1,500 changed lines (excluding testdata), split it: finish a coherent part, open the PR, and add a follow-up card `P1-12b` at the end of the phase file with the remaining ACs (an additive edit to the phase file is allowed without review).
