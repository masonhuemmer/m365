# Implementation Plan: m365 CLI v1

**Branch**: `main` (setup-plan JSON reported `001-m365-cli`; working tree is `main`) | **Date**: 2026-09-16 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-m365-cli/spec.md`

**Note**: This plan is design only. It does not add application source, `go.mod`, `tasks.md`, MCP, CI workflow files, or a shim around the old Python CLI.

## Summary

Replace the local m365 command surface with a Go 1.25+ CLI module. `cmd/m365` is wiring only. Domain types and use cases stay free of CLI, HTTP, SDK, and OS imports. A thin Graph HTTP adapter maps Microsoft 365 resources to spec entities. Auth is delegated PKCE with a localhost callback; tokens live in macOS Keychain when they fit `go-keyring`, otherwise in a `0600` session file (constitution VIII equivalent). Tests use an in-process fake Graph (`httptest`) and synthetic fixtures. Acceptance follows Gherkin → APS IR → generated tests, separate from unit tests.

## Technical Context

**Language/Version**: Go 1.25+ (`go 1.25` in the future `go.mod`). Intended module path: `github.com/masonhuemmer/m365`. Do not write `go.mod` in this planning step.

**Primary Dependencies**: Go standard library (`flag`, `net/http`, `encoding/json`, `os`, `testing`, `httptest`); `golang.org/x/oauth2` for PKCE authorize/exchange only inside the auth adapter; `github.com/zalando/go-keyring` for macOS Keychain. No Cobra, no Microsoft Graph SDK, no kiota, no MSAL.

**Storage**: Tokens in macOS Keychain via `go-keyring` when the blob fits. If Keychain `Set` fails because the item is too large, persist the same blob in `$XDG_STATE_HOME/m365/session.json` defaulting to `~/.local/state/m365/session.json`, mode `0600` (constitution VIII equivalent). Logout MUST delete both. World-readable token files are forbidden. User config file for Client ID and Tenant ID names only (no secret values in repo). Watch checkpoint JSON under the user state directory, mode `0600`, high-watermark ids only.

**Testing**: Go `testing`; table-driven unit tests; `httptest` fake Graph; generated acceptance tests from Gherkin via APS; `gosec`; `govulncheck`; race; coverage; CRAP.

**Target Platform**: macOS CLI invoked by a person or a local agent. Not a service, container, or MCP server.

**Project Type**: Single-module local CLI.

**Performance Goals**: SC-014 — default-limit `mail list` and `teams list` complete in under 15 seconds after a usable session exists. CI measures this against the fake Graph (must finish well under 15s; guards retry/paging bugs). Live timing is optional manual, not a unit test.

**Constraints**: Exit classes `0/3/4/5/6`; JSON stdout default; no secrets in output, fixtures, logs, or this plan; mail and Teams consent independent; list `--top` default 10/20 max 50; attach 10 MiB × 10 files; no bytes on stdout; no MCP, calendar, OneDrive, generic Graph shell, overnight monitor, or `--guard`.

**Scale/Scope**: Two workload namespaces (`mail`, `teams`) plus core `auth`. `chat` is a CLI alias for `teams`. About 15 verbs. Registration seam for those two namespaces only.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Test-Driven Development (NON-NEGOTIABLE). PASS WITH EXCEPTION EX-I-001.** Every behavior slice starts with a RED test (unit and/or generated acceptance). RED is a separate commit of failing tests and required fixtures only. GREEN is the smallest production change. Refactor only on GREEN. Locked RED tests are not rewritten without human approval. Historical US1–US10 git checkpoints (T097–T119) are waived per Complexity Tracking; future slices MUST follow separate RED then GREEN commits.
- [x] **II. Clean Code: Small, Single-Purpose Modules. PASS.** `cmd/m365` is wiring. Files SHOULD stay under 250 lines; over 250 needs a plan/task note; over 500 MUST split. No util grab-bags. No shared mutable globals except immutable process config and composition-root wiring.
- [x] **III. Smallest Sufficient Design. PASS.** Stdlib `flag.FlagSet` plus a small CLI registry; thin HTTP Graph adapter; two-namespace registration only. No plugin marketplace, no unused calendar/OneDrive, no MCP, no generic provider SDK. Three similar lines beat a premature abstraction.
- [x] **IV. Testing Standards. PASS.** Unit tests use ports/fakes; no live Microsoft Graph in unit tests. Contract tests cover flags, stdout/stderr/exit, and Graph mapping against synthetic fixtures or `httptest`. Acceptance is Gherkin → APS → generated tests under `acceptance/`, separate from unit tests. Time, HTTP, and randomness injected. Assert observable behavior.
- [x] **V. CLI Experience Consistency. PASS.** Spec and [contracts/command-catalog.md](contracts/command-catalog.md) lock streams, JSON/human, exit classes, `--help`, dry-run, visible limits, and secret redaction.
- [x] **VI. Performance: Measurable Expectations. PASS.** SC-014: metric = wall-clock time for default-limit inbox list and default-limit chat list; threshold = 15 seconds; CI method = acceptance/command test against fake Graph with a 15s deadline (typically milliseconds). Live mailbox timing is not a unit test and is not required for CI. Graph list calls always send an explicit page size. Silent full-mailbox dumps are forbidden.
- [x] **VII. Domain Isolation and Namespace Extensibility. PASS.** `internal/app/mail` and `internal/app/teams` MUST NOT import each other. Shared auth, config, transport, output, and errors live in core packages. CLI surface `m365 <namespace> <verb> [flags]`. Graph URLs/JSON/SDK types stay in `internal/adapters/graph`. `chat` alias is CLI routing only. Namespace registration for mail and teams is required product architecture, not speculative.
- [x] **VIII. Secrets, Auth, and Least Privilege. PASS.** Delegated PKCE only. Client ID and Tenant ID from env or user config — values NEVER in plan, research, contracts, source, tests, logs, or fixtures. Tokens in macOS Keychain when they fit; otherwise a `0600` session file (see Storage). Mail scopes MUST NOT be required to run Teams, and vice versa. Fixtures use fake tokens and synthetic bodies.
- [x] **Engineering Constraints. PASS.** Clean architecture as above. Makefile named targets: `fmt`, `vet`, `unit`, `race`, `coverage`, `gosec`, `govulncheck`, `acceptance`, `acceptance-mutation`, `crap`, aggregated by `verify`. Exception process: written human approval. Bug fixes do not include surrounding refactors. Top-N/paging visible.
- [x] **Development Workflow & Quality Gates. PASS.** Specify (done) → Constitution Check (this section) → RED commit → GREEN commit → optional REFACTOR → review citing principles → completion only when named gates pass.

v1 MCP absence is not a violation. Written exceptions: EX-I-001 and EX-EC-001 (Complexity Tracking). EX-EC-002 was retired on 2026-11-11 by spec 010.

**Post-design re-check (after Phase 1): PASS.** `research.md`, `data-model.md`, `contracts/`, and `quickstart.md` keep Graph/OAuth/keychain at adapters, domain entities free of SDK fields, fake Graph as the CI boundary, and SC-014 measurement on fakes. Token store: Keychain-with-0600-fallback (T126). Exception EX-I-001 was then the only waiver; EX-EC-001 (CRAP baseline) was added on 2026-09-27.

## Project Structure

### Documentation (this feature)

```text
specs/001-m365-cli/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── command-catalog.md
│   ├── json-stdout.schema.json
│   ├── json-error.schema.json
│   ├── json-watch-event.schema.json
│   ├── fake-graph.md
│   └── acceptance-pipeline.md
└── tasks.md                 # Future /speckit-tasks output, not created by this plan
```

### Source Code (repository root)

```text
.
├── cmd/m365/                    # main wiring only: config, keychain, graph, CLI, shutdown
├── cmd/acceptance-entrypoint-generator/  # APS IR → acceptance/generated tests
├── internal/
│   ├── config/                  # env + user config file; Client ID/Tenant ID names, never values in repo
│   ├── domain/                  # Session, mail, teams, attachment, result, errors — no CLI/HTTP/SDK/OS
│   ├── app/
│   │   ├── auth/                # login/status/logout use cases + ports
│   │   ├── mail/                # mail use cases + ports (MUST NOT import teams)
│   │   └── teams/               # teams use cases + ports (MUST NOT import mail)
│   └── adapters/
│       ├── cli/                 # flag.FlagSet registry, chat alias, JSON/human, exit mapping
│       ├── graph/               # HTTP + JSON mapping; fake server for tests
│       ├── keychain/            # Keychain token store; 0600 FileStore fallback when blob too large
│       ├── fs/                  # local attach read / save write
│       └── watchstate/          # 0600 checkpoint file
├── features/                    # Gherkin: auth, mail, teams, attachments, cli
├── acceptance/
│   ├── generated/               # generated tests; do not edit by hand
│   ├── runtime/
│   ├── steps/
│   └── runner/
├── testdata/                    # synthetic files and JSON fixtures only
├── Makefile
└── scripts/
    ├── acceptance.sh
    ├── acceptance-mutation.sh
    └── crap.sh
```

**Structure Decision**: Single Go module at repo root. Thin `cmd/m365`. Domain independent. Use cases own ports. Adapters implement ports. Interfaces live next to consumers (`internal/app/mail`, `internal/app/teams`, `internal/app/auth`).

## Implementation Workflow

1. Gherkin under `features/` for the slice.
2. Parse to APS JSON IR; optional dry-check; generate `acceptance/generated/`.
3. Write the focused RED unit/use-case/CLI test; confirm expected failure.
4. RED commit (tasks T097, T102, T104, T106, T108, T110, T112, T114, T116, T118): failing tests and required fixtures only; record the commit id in task notes.
5. Smallest production code to GREEN; GREEN commit (T098, T103, T105, T107, T109, T111, T113, T115, T117, T119) with no RED-test edits.
6. Refactor only while GREEN.
7. `make verify`. `make acceptance-mutation` before release or after acceptance-contract changes.

## Acceptance Pipeline Plan

Use `unclebob/Acceptance-Pipeline-Specification` (pin a reviewed commit SHA; upstream has no release tags).

```text
features/*.feature
  -> gherkin-parser
  -> build/acceptance/ir/*.json
  -> gherkin-ir-dry-checker
  -> acceptance-entrypoint-generator
  -> acceptance/generated/*_acceptance_test.go
  -> go test ./acceptance/generated
```

Mutation: `gherkin-mutator` plus `acceptance/runner` classifying `test_success` / `test_failure` / `infrastructure_error`. Details: [contracts/acceptance-pipeline.md](contracts/acceptance-pipeline.md).

Gherkin uses product language (session, mail, chat, attachment, dry-run), not HTTP status codes or SDK types.

## Make Targets

```make
fmt:                  # gofmt -l; fail if dirty
vet:                  # go vet ./...
unit:                 # go test ./...  (excludes ./acceptance/generated)
race:                 # go test -race ./...
coverage:             # go test -coverprofile=coverage.out ./...
gosec:                # gosec ./...
govulncheck:          # govulncheck ./...
acceptance:           # scripts/acceptance.sh
acceptance-mutation:  # scripts/acceptance-mutation.sh
crap:                 # scripts/crap.sh — CRAP > 15 fails unless in the baseline (EX-EC-001)
gomod:                # scripts/gomod.sh — go.mod go line unchanged, go mod tidy clean
filesize:             # scripts/filesize.sh — over 250 lines needs a note, over 500 fails
verify: gomod filesize fmt vet unit race coverage gosec govulncheck acceptance crap
```

CI MUST run `make verify`. Do not add Jenkins/GitHub workflow files in this plan.

Pin `gosec` (`v2.25.0`) and `govulncheck` (`v1.6.0`) in `scripts/install-tools.sh`. Pin `unclebob/Acceptance-Pipeline-Specification` to a reviewed commit SHA in that script and record the SHA in task notes (T003). No in-repo Gherkin substitute without a written exception.

Graph list/get/thread/messages adapter tests (T032, T041) MUST use `NewFakeServer` (`httptest`) per contracts/fake-graph.md. Memory-only tests do not satisfy those tasks.

SC-014 CI: T100 — default-limit `mail list` and `teams list` against the fake Graph finish in under 15 seconds.

Production token store: T099 — `internal/adapters/keychain/keyring.go` via `github.com/zalando/go-keyring`; unit tests keep the in-memory fake. T126 — live `cmd/m365` (not `M365_FAKE=1`) tries Keychain first; if `Set` fails because the blob is too large, persist in `FileStore` at `LiveSessionPath()` mode `0600`; logout deletes both.

## Files To Add At Implement Time (not this command)

- `go.mod` / `go.sum` with module path `github.com/masonhuemmer/m365` (adjust only if Origin requires a different path).
- Packages listed in the source tree above.
- `Makefile`, `scripts/*`, `.gitignore` (binaries, coverage, IR, `.env`, key material).
- `.env.example` with empty `M365_CLIENT_ID=` and `M365_TENANT_ID=` placeholders only.

## Out of This Plan

- Application code and `go.mod` creation.
- `/speckit-tasks` and `/speckit-implement`.
- Wrapping the old Python m365 CLI.
- Copying live mailbox or chat content into fixtures.
- MCP, calendar, OneDrive, generic Graph shell, overnight monitor, `--guard`.

## Risks And Tradeoffs

- **Stdlib flags vs Cobra**: `flag.FlagSet` is enough for two namespaces; custom `--help` text must still name caps and limits (locked tests will prove it).
- **No Graph SDK**: HTTP mapping is extra adapter code but keeps types from leaking inward.
- **PKCE loopback on macOS**: Primary path; device-code/WAM stay out unless loopback cannot work.
- **APS untagged upstream**: pin by SHA.
- **SC-014 on fakes**: CI cannot prove live-network 15s; it proves no unbounded paging/retry. Live timing remains optional and manual.

## Complexity Tracking

### EX-I-001 — Historical RED/GREEN git checkpoints (T097–T119)

- **Principle**: I. Test-Driven Development — RED MUST be a separate git commit containing only the failing test and fixtures.
- **Rationale**: US1–US10 RED and GREEN were mixed in the first implement. Splitting that history would require a rewrite and would not produce inspectable RED evidence.
- **Scope**: Git checkpoints T097–T119 only (US1–US10). Does not waive writing RED tests, locking tests, or separate RED then GREEN commits for later work.
- **Retirement**: Phase 14+ and any new slice (including T124–T126) MUST use a RED commit then a GREEN commit. This exception MUST NOT be cited as precedent.
- **Human approval**: 2026-09-16, analyze remediation (C1).

### File-size notes

Constitution II: a Go file over 250 lines needs a refactoring note here; over 500 it must be split. Test files count too (owner's decision, 2026-09-27); generated acceptance tests do not. `scripts/filesize.sh` (in `make verify`) fails on a file over 250 lines without a note below, on any file over 500, and on a stale note. Retirement for every note: split the file the next time a change touches it, then delete its note.

- `internal/adapters/graph/memory.go`: the in-memory fake store for mail and Teams together (calendar and files are already in `memory_cal.go` and `memory_files.go`). Split into `memory_mail.go` and `memory_teams.go`, keeping `Seed` and `Memory` here.
- `internal/app/mail/watch.go`: the mail watch use case plus its classification-input and revision-state helpers. Move classification normalisation (`buildClassificationInput`, the `normalize*` and `truncateUTF` helpers) and revision bookkeeping (`normalizeRevisionState`, `touchRevision`, `pruneRevisions`) into their own files.
- `acceptance/steps/mail_watch_steps.go`: mail watch acceptance steps for both plain watch and response classification. Split the classification steps into `mail_classify_steps.go`.
- `internal/app/mail/watch_test.go`: tests for the mail watch use case, including classification and revision state. Split with `watch.go`: classification and revision tests into their own files.
- `internal/adapters/cli/mail_watch_test.go`: CLI mail watch tests for plain watch and response classification. Split the classification tests into `mail_classify_test.go`.
- `internal/adapters/jev/client_test.go`: jev client tests. Split request-building tests from response-mapping tests.
- `internal/adapters/graph/httpmail.go`: the shared Graph HTTP client (`bearer`, `request`, `do`, paging tokens) plus the mail read calls. Move the client and paging helpers into `httpclient.go`.

### EX-EC-001 — CRAP baseline for functions that predate a working gate

- **Principle**: Engineering Constraints — quality gates MUST block; the complexity/CRAP gate fails on CRAP > 15 unless justified (`crap:` target above).
- **Rationale**: `scripts/crap.sh` never failed until PR #22 (`gocyclo … || true`). When a real gate was turned on, 49 existing functions scored over 15. Fixing all of them first is a large refactor outside any feature.
- **Scope**: Exactly the entries in `scripts/crap-baseline.txt`, each at no more than its recorded score. The gate fails on any other function over 15 and on any baselined score that rises.
- **Retirement**: Fix a function when a change touches it. The gate fails when an entry is stale (its function now scores 15 or less, or is gone), so the list only shrinks. This exception MUST NOT be cited as precedent for new functions.
- **Human approval**: 2026-09-27, the owner chose "Ratchet baseline" for the CRAP gate (session answer before PR #22).
