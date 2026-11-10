# Implementation Plan: Teams meeting transcripts

**Branch**: `009-meetings-transcripts` | **Date**: 2026-11-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/009-meetings-transcripts/spec.md`, and the approved design `2026-11-10-teams-meeting-transcripts/design.md` (kept outside the repo, based on `main@4b160ac`). This plan is based on `main@4877baf`, which includes the page-link host fix (PR #53).

**Note**: Design only. This command adds no application source and no `tasks.md`.

## Summary

Add a `meetings` namespace with two read verbs. `meetings transcripts` lists a meeting's transcripts, and `meetings transcript` saves one as an attributed WebVTT file. The user names the meeting with `--event EVENT_ID` or `--join-url URL`.

- A new app package, `internal/app/meetings`, owns the port and the use cases: consent, flag checks, the event-window selection, the WebVTT checks, and the save. It imports no other namespace.
- A new Graph adapter, `HTTPMeetings`, does the work against Graph:
  - reads the event in UTC;
  - finds the meeting with the `JoinWebUrl` filter;
  - lists transcripts through `@odata.nextLink`, at most 10 pages, following only `https` links on the Graph host;
  - reads the content with `$format=text/vtt`, at most 10 MiB;
  - names the tenant error codes.
- Sign-in gains the opt-in `--meetings` flag. It adds `OnlineMeetings.Read` and `OnlineMeetingTranscript.Read.All` to the requested scopes and sets a fifth consent flag, `meetings`. A plain login is unchanged.
- The shared `fs.WriteFile` now creates the file exclusively, and makes it owner-only on overwrite. This change gets its own RED test.

## Technical Context

**Language/Version**: Go, module `github.com/masonhuemmer/m365`. The toolchain comes from the `go.mod` toolchain line (`go1.26.10` on `main`).

**Primary Dependencies**: No new ones. The existing stdlib `flag`, `net/http`, `net/url`, `encoding/json`, `regexp` and `os`, plus `golang.org/x/oauth2`, which is already used for sign-in. No Graph SDK.

**Storage**: No new runtime storage. The session blob gains an additive `meetings` boolean, so older blobs read as `false`. Transcripts are written only to the path the user names, through the existing write port.

**Testing**:
- Go `testing`.
- App unit tests use a fake `meetings.Store`.
- Graph contract tests use an `httptest.NewTLSServer` fake that counts requests. Continuation links must be `https`, so a TLS server is required.
- The page-link tests use a recording `RoundTripper` with the live base URL, the same pattern as `page_token_host_test.go`.
- CLI tests use fake `Deps`.
- Acceptance tests use APS Gherkin under `features/meetings/`, with step handlers that drive the real HTTP adapter against a TLS fake, like `reply_attach_steps.go`.
- Fixtures are synthetic WebVTT with fictional names.

**Target Platform**: The macOS and Linux CLI and the in-process MCP server (`m365 mcp serve`). The Windows CI job runs `go test ./...`, so file-mode assertions skip on Windows.

**Project Type**: A single-module local CLI.

**Performance Goals**: A Graph request budget, not a latency target. The work is bounded by the Graph round trips. See [Performance contract](#performance-contract).

**Constraints**:
- Exit classes stay `0/3/4/5/6`, and the error JSON schema is unchanged.
- JSON is the default output.
- No transcript text appears on stdout or stderr.
- No token appears in any output, error, file or child process.
- No new timeout.
- A plain login requests exactly today's scopes.
- Other namespaces' messages are unchanged. The one exception is that `files download` now saves owner-only on overwrite (FR-022).

**Scale/Scope**: One namespace, two verbs and one login flag. Four flags on the verbs: `--event`, `--join-url`, `--id` and `--out`. `--overwrite` already exists. One additive consent key.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. TDD. PASS.**
  - Each slice runs SCAFFOLD → locked RED → GREEN, the pattern spec 006 used. SCAFFOLD adds zero-value signatures only, so that RED fails on its assertions, not on compilation. RED holds only the tests and fixtures, and is committed after the failure is seen. GREEN is the smallest change that passes it.
  - The shared `fs.WriteFile` fix is its own slice, with its own RED commit, before any meetings code.
  - A locked test that turns out wrong stops the work for Mason's direction.
- [x] **II. Clean Code. PASS.** Every new file is under 250 lines (estimates in [Source Code](#source-code-implement-time)).
  - `cli/run.go` is 235 lines on `main`. It gains three `Deps` fields and one `case`, so it ends at about 241.
  - `cli/help.go` grows from 165 to about 185 lines.
  - No file crosses 250 lines, so no refactoring note is needed.
  - `cmd/m365/main.go` stays wiring only.
- [x] **III. Smallest Sufficient Design. PASS.**
  - There is no in-memory `MeetingsAPI` fake for `M365_FAKE=1`. The TLS fake in the tests is enough, so in fake mode `meetings` reports "meetings not available".
  - There is no shared transport change: the content format is chosen with `$format=text/vtt` (research R1).
  - There is no shared error-mapper change: the meetings adapter has its own small mapper (design decision 6).
  - There is no `meetings list` verb, no unattributed fallback and no `--out -`.
- [x] **IV. Testing Standards. PASS.**
  - App rules are unit-tested with a fake port.
  - The Graph mapping is tested against a TLS fake server, and the page-link host rule with a recording transport.
  - The CLI streams, exit codes and help are tested through `cli.Run`.
  - Gherkin covers the save, the list, consent and one tenant refusal.
  - Time comes from the event and the transcripts, never from the clock, so the tests are deterministic.
  - All fixtures are synthetic. The commands for each suite are in [Test commands](#test-commands) and in `quickstart.md`.
- [x] **V. CLI Experience Consistency. PASS.**
  - Each verb has tests for its empty state (an empty list, or not found on save), for validation (`--event`/`--join-url`, the `https` check, `--out`), for an auth failure (meetings consent, calendar consent, a tenant code), for Graph errors (other 403, 404, a bad body, the caps), and for the happy path.
  - Both verbs handle `--help`.
  - The flags follow the existing conventions: `--out` and `--overwrite` as in `files download`, `--id` as an explicit choice.
  - The caps are in the help and the docs, so nothing is truncated silently.
- [x] **VI. Performance. PASS.** The metrics, thresholds and measurement are in [Performance contract](#performance-contract). The transcript list has no documented page size. Its limit is the 10-page cap, which is explicit, visible, and fails loudly.
- [x] **VII. Domain Isolation. PASS, with a note.**
  - `internal/app/meetings` imports only `app/auth` and `domain`, and an isolation test enforces it.
  - The meetings adapter reads the event itself, so `app/calendar` and `cli/calendar.go` stay untouched.
  - Graph URLs, OData and JSON stay in `internal/adapters/graph`.
  - **Note:** the core edits are `run.go` registration, `auth` consent, `help.go`, `main.go` wiring and the shared `fs.WriteFile`. The `fs` change alters `files download` overwrite behaviour, but it edits the core write adapter, not the `files` namespace.
- [x] **VIII. Secrets, Auth, Least Privilege. PASS, with a note.**
  - The two scopes are the least Graph allows. `OnlineMeetings.Read` is the read-only lookup scope, and `OnlineMeetingTranscript.Read.All` is the only delegated transcript scope.
  - Both scopes are opt-in with `--meetings`, so tenants without admin consent keep every existing command (ADR 0004 in the workstream notes).
  - The meetings verbs check only meetings consent, plus calendar consent for `--event`, because that path reads an event.
  - **Note:** m365 keeps one session token for every namespace, and its single scope list predates this feature (spec 002). So a `--meetings` login also requests the existing scopes. A meetings-only login was rejected because it would replace the shared session and cut off mail and Teams.
  - Fixtures use the fake token `synthetic-token`. Tokens stay in the existing `HTTPClient`, refresher and keychain. Error messages hold Graph codes only, never Graph messages, and they pass `redact`.
- [x] **Engineering Constraints. PASS, with a note.** `make verify` covers the new code. No dependency is added.
  - **Pre-existing gap:** `make acceptance-mutation` (`scripts/acceptance-mutation.sh`) runs the acceptance runner and prints `test_success`, but never runs a mutator. This feature runs the target, but does not fix it. Mason decides whether that becomes its own task.
- [x] **Development Workflow. PASS.** Specify (done) → this check → SCAFFOLD/RED/GREEN per slice → `make verify` → review → live check.

No new constitution exceptions.

**Post-design re-check (after Phase 1): PASS.**
- The contracts keep the error schema and add `meetings` to `auth status`.
- The data model keeps Graph shapes in the adapter, and the domain has no I/O.
- The fake-Graph contract counts requests against the budget.
- The quickstart's live check writes outside the repository.

## Project Structure

### Documentation (this feature)

```text
specs/009-meetings-transcripts/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── command-catalog.md
│   ├── graph-mapping.md
│   ├── json-transcripts.schema.json
│   └── json-transcript.schema.json
├── checklists/requirements.md
└── tasks.md                 # /speckit-tasks, not this command
```

### Source Code (implement time)

```text
internal/domain/session.go               # + MeetingsConsented
internal/domain/meeting.go               # new (~35): Transcript, TranscriptList, SavedTranscript
internal/domain/webvtt.go                # new (~45): IsWebVTT, VoiceNames
internal/app/auth/ports.go               # + Blob.Meetings, Status mapping, RequireMeetings (with hint)
internal/app/meetings/ports.go           # new (~50): Store port, Target, Occurrence
internal/app/meetings/transcripts.go     # new (~150): List, Save, target checks, window, choose
internal/app/meetings/isolation_test.go  # imports only app/auth and domain
internal/adapters/fs/fs.go               # exclusive create; owner-only overwrite
internal/adapters/graph/oauth.go         # MeetingsScopes; PKCEConfig(..., extra ...string)
internal/adapters/graph/oauth_loopback.go# blobFromToken(ctx, tok, requested); RealLogin extra scopes
internal/adapters/graph/device.go        # pass requested scopes; RealDeviceLogin extra scopes
internal/adapters/graph/httpmeetings.go  # new (~180): HTTPMeetings: event, lookup, list, content
internal/adapters/graph/meetingserrors.go# new (~50): tenant codes, other 401/403, fallback to MapGraphError
internal/adapters/cli/meetings.go        # new (~130): runMeetings, two verbs, per-verb --help
internal/adapters/cli/run.go             # Deps.Meetings, MeetingsLogin, MeetingsDeviceLogin; case "meetings"
internal/adapters/cli/auth.go            # login flags --device-code and --meetings in any order; nsMap meetings
internal/adapters/cli/help.go            # meetingsHelp; root help namespace line and login line
cmd/m365/main.go                         # wire HTTPMeetings and the two meetings login functions
features/meetings/transcripts.feature    # save, list, consent, a tenant refusal, help
acceptance/steps/meetings_steps.go       # TLS fake, synthetic fixtures, file checks
README.md, docs/m365.md                  # meetings section, --meetings sign-in, caps, error codes
specs/001-m365-cli/contracts/json-stdout.schema.json  # + namespaces.meetings (additive, as spec 002 did)
```

**Structure Decision**: The same single module. One new app package and one new Graph adapter file beside the existing namespaces. Changes elsewhere are limited to core registration, auth consent, help, wiring and the shared file writer. `app/calendar`, `cli/calendar.go`, `app/files` and `cli/files.go` are not edited.

## Graph mapping (adapter only)

The full mapping and the fake routes are in [contracts/graph-mapping.md](contracts/graph-mapping.md). In short:

| Step | Request | Notes |
|---|---|---|
| Event | `GET /me/events/{id}?$select=start,end,isOnlineMeeting,onlineMeeting` with `Prefer: outlook.timezone="UTC"` | `--event` only |
| Lookup | `GET /me/onlineMeetings?$filter=` + escaped `JoinWebUrl eq '…'` | `'` doubled per OData |
| List | `GET /me/onlineMeetings/{mid}/transcripts`, then each `@odata.nextLink` | at most 10 pages; `https` and the Graph host only |
| Content | `GET /me/onlineMeetings/{mid}/transcripts/{tid}/content?$format=text/vtt` | at most 10 MiB + 1 byte read |

## Performance contract

The metrics are measured with the TLS fake server, which counts requests and bytes served. Each is a test in `internal/adapters/graph` or `internal/app/meetings`.

| Metric | Threshold | Measured by |
|---|---|---|
| Graph requests per command | at most 13 with `--event` (1 + 1 + 10 + 1), at most 12 with `--join-url` | the fake serves exactly 10 pages and one content body; the test asserts the count |
| Requests after a failure | 0 | each refusal test asserts that no further request arrives after the failing one |
| Content bytes read | at most 10 MiB + 1 | the fake serves 10 MiB + 2 bytes; the adapter refuses, and the fake records bytes written to the connection |
| List pages | at most 10; an 11th link fails before it is requested | the fake serves 11 pages; the request count is 10 list requests |

A breach of any threshold is a defect and blocks completion. Latency and binary size are not affected in any meaningful way, so no target is set for them.

## Test commands

| Suite | Command |
|---|---|
| Domain and app unit tests | `go test ./internal/domain/ ./internal/app/meetings/ ./internal/app/auth/` |
| Shared file writer | `go test ./internal/adapters/fs/` |
| Graph contract tests | `go test ./internal/adapters/graph/ -run 'Meetings\|Transcript\|Blob\|Login'` |
| CLI and MCP | `go test ./internal/adapters/cli/ -run 'Meetings\|Login\|Status\|MCP'` |
| Acceptance | `make acceptance` (generates and runs `features/meetings/`) |
| Race | `go test -race ./internal/app/meetings/ ./internal/adapters/graph/ ./internal/adapters/cli/` |
| Full gate | `make verify`, then `make acceptance-mutation` (see the pre-existing gap above) |

## Out of This Plan

- `/speckit-tasks` and `/speckit-implement`.
- Unattributed transcripts, `/metadataContent`, `.docx`, ad hoc calls, recordings, attendance reports.
- A `meetings list` verb, or online-meeting fields in `calendar` output.
- Changing the shared `MapGraphError` or `request()`, timeouts, or the other code-versus-docs gaps from research.
- Fixing `scripts/acceptance-mutation.sh`.
- An in-memory fake for `M365_FAKE=1`, or MCP recipe text for meetings. `m365_help namespace=meetings` returns the help.
- Admitting a transcript anywhere, or a release.

## Risks And Tradeoffs

- **The attendee lookup is undocumented.** The docs don't say whether an invited attendee may use the `JoinWebUrl` filter. If the live check returns no meeting or a 403 for a meeting the user only attended, stop and return to design.
- **`$format=text/vtt` instead of an `Accept` header** (research R1). Both are documented for `text/vtt`. If the live service ignores `$format`, the fix is a small `Accept` option on the shared request, behind its own RED test.
- **The event's join URL may not match `JoinWebUrl` exactly**, for example because of encoding. The live check confirms it. `--join-url` is the fallback.
- **Admin consent may not cover the transcript scope.** If so, `auth login --meetings` stops at an administrator-approval prompt, and a plain login is unaffected.
- **Tenant switches.** A 403 naming `GraphAccessToTranscriptsDisabled` or `SpeakerAttributionNotAllowed` means stop and report the code.
- **A plain login after `--meetings`** may drop the flag, depending on which scopes Entra returns. The help says to pass `--meetings` whenever meetings are needed.
- **The event window is a rule of thumb.** A transcript that starts more than 1 hour after the scheduled end needs `--id`. The output always shows `created` and `ended`, and a test pins both window edges.
- **Real WebVTT may differ from the documented examples**, with cue identifiers, multi-line cues, `NOTE` blocks, or guest and phone-caller names. The fixtures model all of these, and the live file confirms them.
- **Owner-only overwrite changes `files download`.** This is intended (FR-022), and Windows skips the mode assertion.

## Complexity Tracking

No new constitution exceptions.
