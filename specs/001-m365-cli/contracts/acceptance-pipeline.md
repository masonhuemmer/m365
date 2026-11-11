# Acceptance pipeline contract

m365 uses `unclebob/Acceptance-Pipeline-Specification`. Feature files are
product behavior; generated Go tests are projections.

## Inputs

```text
features/auth/*.feature
features/mail/*.feature
features/teams/*.feature
features/attachments/*.feature
features/cli/*.feature
```

Feature files MUST use product language (session, mail, chat, attachment,
dry-run, exit class). They MUST NOT name Go, Cobra, kiota, HTTP paths, or
MIME internals. Parameters for ids, limits, paths, and sizes. Fake Graph
and fake secret store by default.

## Normal flow

```text
gherkin-parser <feature> build/acceptance/ir/<feature>.json
gherkin-ir-dry-checker build/acceptance/ir/<feature>.json build/acceptance/dry/<feature>.json
acceptance-entrypoint-generator build/acceptance/ir/<feature>.json acceptance/generated/<feature>_acceptance_test.go
go test ./acceptance/generated
```

`scripts/acceptance.sh` owns this for developers and CI.

## Generated tests

- Live under `acceptance/generated/`; not edited by hand.
- Use `testing`.
- Load IR deterministically.
- Dispatch through `acceptance/runtime`.
- Fail if a step handler is missing.
- Do not replace unit tests.

## Runtime

- Fresh world per scenario/example.
- Fake clock and fake Graph.
- No sleeps to hide flakes.
- Interactive login steps use a fake browser/callback, not a real gesture, except documented manual quickstart.

## Mutation

`make acceptance-mutation` (`scripts/acceptance-mutation.sh`) runs
`gherkin-mutator` at level `full` on a copy of every feature with an
Examples table, so `features/` never changes. The generator also writes APS
metadata to `build/acceptance/metadata/`. The persistent worker
`acceptance/runner` runs the feature's generated test from a binary
compiled once, with `M365_ACCEPTANCE_IR` naming the mutated IR, and
classifies `test_success`, `test_failure`, `infrastructure_error` (a run
that selects no test, times out or cannot start). A surviving mutation, an
error, or no feature to mutate fails the target. Reports under
`build/acceptance-mutation/`. Hardening workflow: before release and after
acceptance-contract changes; not part of `make verify`.

Generated tests load the parser IR, not the `.feature` text: one execution
per example row (`example_<n>`), Background first, `<name>` placeholders
filled from the row.

Pin APS tooling to a reviewed upstream commit SHA (no release tags).
