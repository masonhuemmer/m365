# Tasks: Acceptance-example mutation gate

**Input**: [spec.md](spec.md), [plan.md](plan.md), `.specify/memory/constitution.md`, APS `accaa33` (`.tools/aps`).

**Tests**: MANDATORY (Constitution I). Each slice: SCAFFOLD (zero-value signatures only) → failing tests run and seen to fail on assertions → RED commit (tests only) → smallest GREEN → GREEN commit. Locked tests are never edited without Mason's approval.

**Do not**: edit any locked test; convert an existing feature; add the gate to `make verify`; push, open a PR or merge without Mason's explicit yes.

## Phase 1: Runtime runs the IR (US2)

- [ ] T001 SCAFFOLD `acceptance/runtime/ir.go`: IR types, `irPathFor`, `loadIR`, `expand`, `substitute` returning zero values.
- [ ] T002 RED `acceptance/runtime/ir_test.go`: IR path from feature path and env override; expansion (no examples → one execution named by the scenario; N rows → `example_1..N`); Background first; placeholder substitution; missing placeholder fails naming it; unmatched expanded steps all listed.
- [ ] T003 GREEN `ir.go`, and `RunFeature` runs the expanded IR.

## Phase 2: Generator metadata (US1)

- [ ] T004 SCAFFOLD `cmd/acceptance-entrypoint-generator/metadata.go`: `metadataName`, `writeMetadata` zero-valued.
- [ ] T005 RED `metadata_test.go`: name mapping (APS examples); one file per feature with schema_version 1, feature_path, ir_path, `sha256:` hash of the generated file, hash_scope, generated_files.
- [ ] T006 GREEN, and `run` writes metadata to `build/acceptance/metadata`.

## Phase 3: Runner worker (US1)

- [ ] T007 SCAFFOLD `acceptance/runner/worker.go`: job/response types, `serve`, `classify` zero-valued.
- [ ] T008 RED `worker_test.go`: one response line per job line with the job id; malformed line → infrastructure_error; classify: pass → test_success, `--- FAIL` → test_failure, `no tests to run`/start error/timeout → infrastructure_error; nothing but JSON on stdout.
- [ ] T009 GREEN `worker.go`, `main.go` wiring (`-bin`, `-test`, `-dir`).

## Phase 4: Gate and examples (US1, US3)

- [ ] T010 `features/cli/exit-codes.feature`: the exit-code outline; `make acceptance` runs its rows.
- [ ] T011 `scripts/acceptance-mutation.sh` per plan step 4; run it: all mutations killed; commit the stamped feature.
- [ ] T012 Planted-survivor check: a scratch column no step reads makes the gate exit non-zero (evidence in the Log; scratch change not committed).
- [ ] T013 Delete EX-EC-002 from `specs/001-m365-cli/plan.md` (section and line 48); update README/plan notes on the target.
- [ ] T014 `make verify` passes; `make acceptance-mutation` passes.
- [ ] T015 Ask Mason before push, PR or merge.

## Log
