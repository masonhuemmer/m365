# Tasks: Acceptance-example mutation gate

**Input**: [spec.md](spec.md), [plan.md](plan.md), `.specify/memory/constitution.md`, APS `accaa33` (`.tools/aps`).

**Tests**: MANDATORY (Constitution I). Each slice: SCAFFOLD (zero-value signatures only) → failing tests run and seen to fail on assertions → RED commit (tests only) → smallest GREEN → GREEN commit. Locked tests are never edited without Mason's approval.

**Do not**: edit any locked test; convert an existing feature; add the gate to `make verify`; push, open a PR or merge without Mason's explicit yes.

## Phase 1: Runtime runs the IR (US2)

- [X] T001 SCAFFOLD `acceptance/runtime/ir.go`: IR types, `irPathFor`, `loadIR`, `expand`, `substitute` returning zero values.
- [X] T002 RED `acceptance/runtime/ir_test.go`: IR path from feature path and env override; expansion (no examples → one execution named by the scenario; N rows → `example_1..N`); Background first; placeholder substitution; missing placeholder fails naming it; unmatched expanded steps all listed.
- [X] T003 GREEN `ir.go`, and `RunFeature` runs the expanded IR.

## Phase 2: Generator metadata (US1)

- [X] T004 SCAFFOLD `cmd/acceptance-entrypoint-generator/metadata.go`: `metadataName`, `writeMetadata` zero-valued.
- [X] T005 RED `metadata_test.go`: name mapping (APS examples); one file per feature with schema_version 1, feature_path, ir_path, `sha256:` hash of the generated file, hash_scope, generated_files.
- [X] T006 GREEN, and `run` writes metadata to `build/acceptance/metadata`.

## Phase 3: Runner worker (US1)

- [X] T007 SCAFFOLD `acceptance/runner/worker.go`: job/response types, `serve`, `classify` zero-valued.
- [X] T008 RED `worker_test.go`: one response line per job line with the job id; malformed line → infrastructure_error; classify: pass → test_success, `--- FAIL` → test_failure, `no tests to run`/start error/timeout → infrastructure_error; nothing but JSON on stdout.
- [X] T009 GREEN `worker.go`, `main.go` wiring (`-bin`, `-test`, `-dir`).

## Phase 4: Gate and examples (US1, US3)

- [X] T010 `features/cli/unknown-commands.feature` (RED: its new step has no handler) and the `the error is` step (GREEN); `make acceptance` runs its rows.
- [X] T011 `scripts/acceptance-mutation.sh` per plan step 4; run it: all mutations killed; `features/` unchanged.
- [X] T012 Planted-survivor check: a scratch column no step reads makes the gate exit non-zero (evidence in the Log; scratch change not committed).
- [X] T013 Delete EX-EC-002 from `specs/001-m365-cli/plan.md` (section and line 48); update README/plan notes on the target.
- [X] T014 `make verify` passes; `make acceptance-mutation` passes.
- [ ] T015 Ask Mason before push, PR or merge.

## Log

- T001–T003 (2026-11-11): SCAFFOLD `6c2f89e`, RED `7443c9a` (10 tests failed on assertions), GREEN `f88fdbc`. `make acceptance` runs all 81 existing scenarios from the IR (81 `Scenario:` lines, 81 subtests). Locked `strict_test.go`/`step_args_test.go` unchanged and passing.
- T004–T006: SCAFFOLD `554d04c`, RED `12f9e15` (4 tests; the commit was amended once, before any GREEN work, for gofmt whitespace only), GREEN `60f58fe`. 38 metadata files in `build/acceptance/metadata/`. Locked `generate_test.go` unchanged and passing.
- T007–T009: SCAFFOLD `c84d9ff`, RED `9da7202` (6 tests, including a helper-process run of the real exec path), GREEN `de5a79e`. Locked `runner/main_test.go` unchanged.
- T010: RED `3acb90a` (all three rows expanded with their values and failed only on the missing `the error is` handler), GREEN `ba23609`. Not-found rows were dropped after a probe: their messages do not echo the id, so a mutated id would survive.
- T011 `00e5fa9`: `make acceptance-mutation` → `total=12 killed=12 survived=0 errors=0`, exit 0, twice; `features/` unchanged. First draft ran at level `hard` on the real feature with committed stamps, but the mutator rewrites the manifest's `tested_at` on every run, so the gate would dirty `features/` each time; changed (amended before push) to level `full` on a copy.
- T012: planted column `note` (read by no step) → `total=15 killed=12 survived=3`, gate exit non-zero; restored. No feature with Examples → `acceptance-mutation: no feature has an Examples table; nothing to mutate`, exit 1. Worker pointed at a missing test → `errors=12`, mutator exit 1.
- T013: EX-EC-002 section removed from `specs/001-m365-cli/plan.md` (a one-line retirement note stays at the exceptions list); `contracts/acceptance-pipeline.md` describes the real gate.
- T014: first `make verify` failed only at `crap`: the generator's `run` now scores 14.4 against a 20.0 baseline, so the stale entry was removed (`c698a2c`, EX-EC-001 ratchet).
- Follow-up (needs Mason): `Parse` and `checkFeature` in `acceptance/runtime` are now used only by their locked tests; retiring them means deleting locked tests.
- T014: after `c698a2c`, `make verify` exit 0 (filesize ok, gosec 0 issues, govulncheck 0 affecting, acceptance ok, crap ok with 42 baseline entries); `make acceptance-mutation` exit 0 (`total=12 killed=12`).
