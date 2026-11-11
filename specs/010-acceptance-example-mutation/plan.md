# Implementation Plan: Acceptance-example mutation gate

**Branch**: `010-acceptance-example-mutation` | **Date**: 2026-11-11 | **Spec**: [spec.md](spec.md)

## Summary

Make the APS example-mutation stage real and retire EX-EC-002. The acceptance runtime runs the parser IR (Examples, Background, placeholders) instead of re-reading `.feature` text; the generator writes APS metadata; `acceptance/runner` becomes the persistent NDJSON worker the APS `gherkin-mutator` drives; a new outline feature states unknown-command errors as examples; `make acceptance-mutation` runs the mutator on every feature with Examples.

## Technical Context

- **Language**: Go (toolchain from `go.mod`). No new dependency.
- **APS**: pinned `accaa33` in `.tools/aps`; `gherkin-parser`, `gherkin-ir-dry-checker`, `gherkin-mutator` from `scripts/install-tools.sh`. Contracts: `parser-spec.md` (IR), `acceptance-generator.md` (metadata), `mutator-spec.md` (worker protocol, levels, manifest).
- **IR**: `{name, background?: [step], scenarios: [{name, steps: [{keyword, text}], examples: [{key: value}]}]}`, already produced by `scripts/acceptance.sh` into `build/acceptance/ir/<name>.json`.

## Design

1. **Runtime (`acceptance/runtime`)**. `RunFeature(t, featurePath)` keeps its signature (the locked generator test pins the call). It resolves the IR path from the feature path with the generator's name rule (`features/cli/help.feature` → `build/acceptance/ir/cli_help.json`, relative to the same root), unless `M365_ACCEPTANCE_IR` names a file. It expands each scenario into executions (one per example row; one with no row), prepends Background, substitutes `<key>` from the row, checks every expanded step for a handler before running (SDO-566), then runs each execution in a fresh `World` as subtest `<scenario>` or `<scenario>/example_<n>`. New file `ir.go`; `Parse`/`checkFeature` stay for their locked tests.
2. **Generator (`cmd/acceptance-entrypoint-generator`)**. `run` also writes metadata to `build/acceptance/metadata/<feature-metadata-name>.json` (lowercase, runs of non-alphanumerics → `-`). `implementation_hash` is `sha256:` over the generated test file's bytes; `generated_files` lists it. New file `metadata.go`. `generate` and its locked tests are unchanged.
3. **Worker (`acceptance/runner`)**. Flags `-bin` (test binary), `-test` (test name), `-dir` (working directory for the binary). For each stdin line (job: `id`, `feature_json`, `timeout`) it runs `bin -test.run ^<test>$ -test.count=1` in `dir` with `M365_ACCEPTANCE_IR=<abs feature_json>`, and writes one response line (`id`, `outcome`, `output`, `error`, `duration` ns). Classification: exit 0 with the test run → `test_success`; exit non-zero with a `--- FAIL` line → `test_failure`; otherwise (start failure, timeout, `no tests to run`, malformed job) → `infrastructure_error`. Diagnostics go to stderr only. Logic in `worker.go`, `main.go` stays wiring.
4. **Gate (`scripts/acceptance-mutation.sh`)**. Runs `scripts/acceptance.sh`, builds `build/acceptance-mutation/generated.test` (`go test -c ./acceptance/generated`) and the worker once, then for each feature with an `Examples:` line copies it to `build/acceptance-mutation/<name>/` and runs `gherkin-mutator --feature <copy> --generated-dir build/acceptance --work-dir build/acceptance-mutation/<name> --level full --runner-worker "<worker> -bin … -test TestAcceptance_<name> -dir acceptance/generated"`. Fails when no feature has Examples, or when any mutator run exits non-zero. Prints each mutator summary.
5. **Feature `features/cli/unknown-commands.feature`**. One Scenario Outline over `args | code | class | message`, using the steps `I run`, `exit code` and a new `the error is <class>: <message>` (exact JSON error on stderr). The message names the unknown word, so a mutated argument changes it. Not-found errors were left out: their messages do not echo the id, so a mutated id would survive.

## Constitution Check

- [x] **I. TDD. PASS.** Runtime, generator metadata and worker each run SCAFFOLD → locked RED → smallest GREEN. No locked test is edited: `strict_test.go`/`step_args_test.go` keep testing `Parse`; `generate_test.go` keeps its pinned source; `runner/main_test.go` only compiles. The gate script and the outline feature are verified by running the gate, including a planted surviving cell (recorded in tasks.md).
- [x] **II. Clean Code. PASS.** New files: `runtime/ir.go` (~120 lines), `generator/metadata.go` (~60), `runner/worker.go` (~120). Each under 250.
- [x] **III. Smallest Sufficient Design. PASS.** No BDD framework, no regex step DSL: placeholders are substituted into step text, so existing handlers serve outlines unchanged. No equivalent-mutation filter.
- [x] **IV. Testing Standards. PASS.** This feature is the APS example-mutation stage IV names. Unit tests use synthetic IR and a fake job runner; no network.
- [x] **V. CLI Experience Consistency. PASS.** No CLI change. The outline asserts the existing usage-error contract.
- [x] **VI. Performance. PASS.** No runtime path of the CLI changes. The gate compiles the test binary once and keeps the worker hot (APS persistent mode).
- [x] **VII. Domain Isolation. PASS.** All changes are in acceptance tooling (`acceptance/`, `cmd/acceptance-entrypoint-generator`, `scripts/`).
- [x] **VIII. Secrets. PASS.** No secrets, tokens or real data; the worker passes only a file path in its environment.
- [x] **Engineering Constraints. PASS** once EX-EC-002 is removed: `make acceptance-mutation` is a real blocking acceptance-example mutation target.

## Complexity Tracking

None. EX-EC-002 is deleted by this feature. `Parse` and `checkFeature` remain: `RunFeature` uses them when a feature has no IR file (a test calling it before `scripts/acceptance.sh`).

## Out of This Plan

- Converting existing scenarios to outlines (they are locked acceptance tests).
- Adding the gate to `make verify` or CI.
- Project-specific equivalent-mutation filters.
