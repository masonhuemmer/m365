# Tasks: Teams meeting transcripts

**Input**: Design documents from `/specs/009-meetings-transcripts/`

**Prerequisites**: plan.md, spec.md, research.md (R1–R10), data-model.md, contracts/ (command-catalog.md, graph-mapping.md, json-transcripts.schema.json, json-transcript.schema.json), quickstart.md, `.specify/memory/constitution.md`, and the approved design `~/.work/m365/2026-11-10-teams-meeting-transcripts/design.md`

**Tests**: MANDATORY per Constitution I. Every slice runs the same steps:
1. A SCAFFOLD commit, if new symbols are needed.
2. Write the failing tests.
3. Run them and confirm that every new failure is an assertion failure, with no compile, import or fixture errors.
4. A RED commit with only the tests, fixtures and Gherkin.
5. The smallest GREEN change.
6. A GREEN commit.

Locked tests are never edited afterwards without Mason's approval. If a locked test turns out wrong, stop and ask.

- **Assertions**: exact strings and exact JSON in new tests. No `strings.Contains` on messages, except when asserting that something is absent.
- **Fixtures**: synthetic only, with fictional names (`Avery Example`, `Blake Sample`), the host `teams.example.test`, and the token `synthetic-token`. No live Graph calls, real names, real transcript text, tokens or tenant ids.

**Scaffold convention**: the one approved in spec 006 (T003(d)). A SCAFFOLD commit adds only types, fields and function signatures whose bodies return zero values (or `nil` errors). It adds no behaviour, no routing and no tests.

**Do not**:
- edit `internal/app/calendar`, `internal/app/files`, `internal/adapters/cli/calendar.go` or `internal/adapters/cli/files.go`;
- change the shared `MapGraphError`, `HTTPClient.request` or `allowedNext`;
- add a dependency, a timeout, a `meetings list` verb, an unattributed fallback, `--out -`, or an in-memory meetings fake for `M365_FAKE=1`;
- fix `scripts/acceptance-mutation.sh`;
- push, open a PR, or merge without Mason's explicit yes.

**Toolchain**: from the `go.mod` toolchain line (`go1.26.10`). The pinned tools are reached through the untracked `.tools` symlink, which is never staged. The gate is `make verify`.

**Organization**: Setup and Foundational block every story. US1–US4 map to spec User Stories 1–4. The MVP for the live check is Setup + Foundational + US1 + US3: US1 saves, and US3 is needed to sign in with meetings consent.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no incomplete dependencies)
- **[Story]**: `[US1]`–`[US4]`, on user-story phases only
- Every task names its exact file paths

## Path Conventions

`internal/domain/`, `internal/app/auth/`, `internal/app/meetings/` (new), `internal/adapters/fs/`, `internal/adapters/graph/`, `internal/adapters/cli/`, `cmd/m365/`, `acceptance/steps/`, `features/meetings/` (new)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Check the worktree, record the baseline, and find any locked test that this feature would break.

- [X] T001 In `~/Git/worktrees/m365/009-meetings-transcripts`, confirm that the branch is `009-meetings-transcripts` at `6b1356b`, on top of `main@4877baf`. Confirm that `git status --short` shows only `?? .tools`. Confirm the repository-local git identity with `git config user.name` and `git config user.email`.
- [X] T002 Record the baseline in the Notes section of specs/009-meetings-transcripts/tasks.md:
  - the result of `make verify`;
  - the line counts from `wc -l internal/adapters/cli/run.go internal/adapters/cli/help.go internal/adapters/cli/auth.go internal/app/auth/ports.go internal/adapters/graph/oauth_loopback.go internal/adapters/fs/fs.go cmd/m365/main.go`.
- [X] T003 Find the existing tests that the planned behaviour changes would break, using a scratch edit of internal/adapters/cli/auth.go, internal/adapters/fs/fs.go and internal/adapters/cli/help.go that is never committed:
  - add `"meetings": false` to `nsMap` in internal/adapters/cli/auth.go;
  - make `fs.WriteFile` in internal/adapters/fs/fs.go use `O_EXCL`, plus `Chmod(0o600)` on overwrite;
  - add a `meetings` line to `rootHelp` in internal/adapters/cli/help.go;
  - run `go test ./...`, record every failing test by name in Notes, then `git restore` all three files.

  If any test fails, STOP and ask Mason to approve editing it, and record the approval text and date in Notes. If none fails, record "no locked test affected".

**Checkpoint**: The baseline is green and recorded, and the list of affected tests is recorded (or approved).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The private shared save (FR-009, FR-022), the pure WebVTT rules, the domain types, and the meetings consent flag that every verb checks. No meetings verb exists yet.

**⚠️ CRITICAL**: No user-story work until this phase is complete.

### Slice F1: owner-only shared save (its own RED, before any meetings code)

- [X] T004 Write RED tests in internal/adapters/fs/write_private_test.go (new file; do not edit fs_test.go). Each case uses `t.TempDir()`, and every mode check is skipped when `runtime.GOOS == "windows"`.
  - (a) A file written with mode `0o644`, then overwritten with `WriteFile(p, []byte("new"), true)`: the content is `new` and `Mode().Perm()` is `0o600`. **This case must fail at baseline.**
  - (b) An existing file and `overwrite=false`: the error has class `usage` and message exactly `destination exists; pass --overwrite`, and the content is unchanged. This is a guard; it may pass at baseline.
  - (c) A new path: the file is created with mode `0o600`. This is a guard.
- [X] T005 Run `go test ./internal/adapters/fs/`. Confirm that (a) fails on the mode assertion (`0644` ≠ `0600`), then make a RED commit with T004 only.
- [X] T006 GREEN in internal/adapters/fs/fs.go `WriteFile`, per research R9:
  - keep the empty-path check and `MkdirAll`;
  - without overwrite, `os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)`, and map `errors.Is(err, fs.ErrExist)` to the unchanged message `destination exists; pass --overwrite`;
  - with overwrite, use `os.O_WRONLY|os.O_CREATE|os.O_TRUNC` with `0o600`, then `f.Chmod(0o600)` before writing;
  - every other error stays `unwritable path: PATH`;
  - close the file and check the error from `Close`.
- [X] T007 Run `go test ./internal/adapters/fs/ ./internal/adapters/cli/ ./internal/app/files/`, then make a GREEN commit with T006 only.

### Slice F2: domain types, WebVTT rules and the meetings consent flag

- [X] T008 SCAFFOLD in internal/domain/meeting.go, internal/domain/webvtt.go, internal/domain/session.go and internal/app/auth/ports.go, then a SCAFFOLD commit:
  - internal/domain/meeting.go: `Transcript{ID, Created, Ended string}` with JSON `id`, `created`, `ended`; `TranscriptList{Count int; Items []Transcript}` with JSON `count`, `items`; `SavedTranscript{Path, ID, Created, Ended string; Bytes int; Speakers []string}` with JSON `path`, `id`, `created`, `ended`, `bytes`, `speakers`.
  - internal/domain/webvtt.go: `IsWebVTT(b []byte) bool` returning `false`, and `VoiceNames(b []byte) []string` returning `nil`.
  - internal/domain/session.go: `MeetingsConsented bool`.
  - internal/app/auth/ports.go: `Blob.Meetings bool` with JSON `meetings`, and `RequireMeetings(sess domain.Session) error` returning `nil`.
- [X] T009 [P] Write RED tests in internal/domain/webvtt_test.go:
  - `IsWebVTT` is true for `WEBVTT`, `WEBVTT\n`, `WEBVTT \n`, `WEBVTT\t`, `WEBVTT\r\n`, and `\xEF\xBB\xBFWEBVTT\n`.
  - It is false for ``, `WEBVTTX`, ` WEBVTT`, `{"value":[]}`, and `webvtt\n`.
  - `VoiceNames` on the exact synthetic fixture in contracts/graph-mapping.md (BOM, `NOTE`, cue id `1`, a multi-line cue, `<v.loud Blake Sample>`, an unnamed cue) returns exactly `[]string{"Avery Example", "Blake Sample"}`.
  - `<v   Pat Doe  >x</v>` returns `["Pat Doe"]`.
  - `<v.a.b Pat Doe>` returns `["Pat Doe"]`.
  - `<v >x</v>` and `<v>x</v>` return length 0.
  - `<b>x</b> <i>y</i>` returns length 0.
  - Put the fixture in internal/domain/testdata/meeting.vtt, with a real BOM and LF line endings, and read it with `os.ReadFile`.
- [X] T010 [P] Write RED tests in internal/app/auth/meetings_consent_test.go:
  - `Status` maps a stored `Blob{Usable: true, Meetings: true}` to `MeetingsConsented == true`.
  - `json.Unmarshal` of `{"account":"a","usable":true,"mail":true}` into `Blob` gives `Meetings == false`.
  - `RequireMeetings(domain.Session{})` gives class `auth`, message `not signed in`.
  - A signed-in, usable session without meetings gives class `auth`, message exactly `missing meetings consent`, and `(*domain.Error).Hint` exactly `run m365 auth login --meetings`.
  - With meetings consent, it returns `nil`.
- [X] T011 Run `go test ./internal/domain/ ./internal/app/auth/`. Confirm that every new failure is an assertion failure, then make a RED commit with T009–T010 and the testdata file.
- [X] T012 [P] Implement internal/domain/webvtt.go per data-model.md: "an optional UTF-8 BOM (`EF BB BF`), then `WEBVTT`, then the end of input or one of space, tab, `\r`, `\n`". `VoiceNames` uses the regular expression `<v(\.[^\s>]+)*\s+([^>]+)>`, trims each name, drops empty ones, and keeps "distinct; order of first appearance". Stdlib only.
- [X] T013 [P] Implement `Status` mapping and `RequireMeetings` in internal/app/auth/ports.go. Reuse `Require(sess, false, false)` for the signed-in checks, then return `&domain.Error{Class: domain.ClassAuth, Message: "missing meetings consent", Hint: "run m365 auth login --meetings"}`.
- [X] T014 Run `go test ./internal/domain/ ./internal/app/auth/`, then make a GREEN commit with T012–T013.

**Checkpoint**: Overwrite is owner-only, the WebVTT rules pass, and meetings consent can be required. No CLI change yet.

---

## Phase 3: User Story 1 — Save the transcript of a meeting I was invited to (Priority: P1) 🎯 MVP

**Goal**: `m365 meetings transcript (--event ID | --join-url URL) --out FILE [--id TID] [--overwrite]` saves the attributed WebVTT byte for byte and prints the summary. This phase builds everything both verbs share: the target checks, the event read, the meeting lookup, the paged list with its link rule and page cap, the event window, and the content cap. Spec US4 scenario 4 (no speaker names) is covered here, because a save must never write an unattributed file (SC-002).

**Independent Test**: Against the TLS fake, which holds one online event, one meeting and one attributed transcript, `m365 meetings transcript --event evt-online --out FILE` exits 0. The file equals the fixture byte for byte, stdout is the summary with `speakers` `["Avery Example","Blake Sample"]`, and neither stdout nor stderr contains `Opening the review.`.

### Scaffold and tests for User Story 1 (MANDATORY)

- [X] T015 [US1] SCAFFOLD in internal/app/meetings/ports.go, internal/app/meetings/transcripts.go, internal/adapters/graph/httpmeetings.go and internal/adapters/cli/run.go, then a SCAFFOLD commit:
  - internal/app/meetings/ports.go: `Store` with `EventMeeting(ctx, eventID string) (Occurrence, error)`, `FindMeeting(ctx, joinURL string) (string, error)`, `Transcripts(ctx, meetingID string) ([]domain.Transcript, error)` and `TranscriptContent(ctx, meetingID, transcriptID string) ([]byte, error)`; `Occurrence{JoinURL string; Start, End time.Time}`; `Target{EventID, JoinURL, ID, Out string; Overwrite bool}`.
  - internal/app/meetings/transcripts.go: `Save(ctx, st Store, sess domain.Session, t Target, write func(string, []byte, bool) error) (domain.SavedTranscript, error)`, returning zero values.
  - internal/adapters/graph/httpmeetings.go: `type HTTPMeetings struct{ *HTTPClient }`, whose four methods return zero values.
  - internal/adapters/cli/run.go: the field `Meetings meetings.Store` in `Deps` only, with no routing.
- [X] T016 [P] [US1] Write the isolation guard internal/app/meetings/isolation_test.go, copying `TestCalendarDoesNotImportOthers` in internal/app/calendar/isolation_test.go. Forbid `internal/app/mail`, `internal/app/teams`, `internal/app/calendar`, `internal/app/files` and `internal/adapters`. It is a guard and may pass at scaffold.
- [X] T017 [P] [US1] Write RED app tests in internal/app/meetings/save_test.go:
  - The fake `Store` records each call in order, returns fixtures, and fails the test on any unexpected call. The fake `write` records `(path, bytes, overwrite)`.
  - The occurrence is 2030-01-15 `15:00Z`–`15:30Z`, with `JoinURL` `https://teams.example.test/l/meetup-join/abc`. The content is internal/domain/testdata/meeting.vtt (read via `../../domain/testdata/meeting.vtt`).
  - Each case asserts the class, the exact message and hint, and the exact call list:
    - No meetings consent: `missing meetings consent`, hint `run m365 auth login --meetings`, zero store calls.
    - Both `EventID` and `JoinURL`, or neither: usage `use exactly one of --event or --join-url`, zero calls.
    - `JoinURL` of `http://teams.example.test/x`, `teams.example.test/x` or `https:///x`: usage `--join-url must be an https URL`, zero calls.
    - No `Out`: usage `--out is required`, zero calls.
    - `--event` without calendar consent: auth `missing calendar consent`, zero calls.
    - Happy `--event`, where the only transcript is `t1` created `15:05:00Z`, ended `15:29:00Z`:
      - calls `EventMeeting(evt)`, `FindMeeting(joinURL)`, `Transcripts(mtg-1)`, `TranscriptContent(mtg-1, t1)`, in that order;
      - `write` gets `Out`, the exact fixture bytes and `Overwrite`;
      - the result is `SavedTranscript{Path: Out, ID: "t1", Created: "2030-01-15T15:05:00Z", Ended: "2030-01-15T15:29:00Z", Bytes: len(fixture), Speakers: ["Avery Example","Blake Sample"]}`.
    - Happy `--join-url`: no `EventMeeting` call.
    - Window edges: transcripts at `13:59:59Z` and `16:30:01Z` only gives not_found `no transcript in scope; transcription may have been off` and no `TranscriptContent` call. Transcripts at `14:00:00Z` and `16:30:00Z` gives usage `several transcripts in scope`, hint `run m365 meetings transcripts, then pass --id`, and no content call.
    - `--id` naming the `13:59:59Z` transcript (outside the window) saves that transcript.
    - `--id t9`, which does not exist: not_found `no transcript with that id`, and no content call.
    - Content `{"value":[]}`: service `transcript is not WebVTT`, and `write` is not called.
    - Content `WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nno speaker\n`: service `transcript has no speaker names`, and `write` is not called.
    - `write` returning usage `destination exists; pass --overwrite`: that same error comes back.
    - `Created` of `yesterday`: service `invalid transcript`.
    - An error from the store's `Transcripts` (`domain.Service("x")`) comes back unchanged, with no later calls.
- [X] T018 [P] [US1] Write RED Graph contract tests in internal/adapters/graph/httpmeetings_test.go. Use an `httptest.NewTLSServer` fake with `HTTPClient{Base: srv.URL, Token: "synthetic-token", Client: srv.Client()}`; it counts requests and records method, path, raw query and the `Prefer` header. Routes and fixtures follow contracts/graph-mapping.md.
  - `EventMeeting("evt-online")`:
    - sends `GET /me/events/evt-online`, whose decoded query has `$select` `start,end,isOnlineMeeting,onlineMeeting`;
    - sends `Prefer` exactly `outlook.timezone="UTC"`;
    - returns `Start` `2030-01-15T15:00:00Z` and `End` `2030-01-15T15:30:00Z` (UTC) and the join URL.
  - `evt-offline` gives not_found `event is not an online meeting`; `evt-missing` (404) gives not_found `not found`.
  - `FindMeeting("https://teams.example.test/l/meetup-join/a'b")` sends a decoded `$filter` exactly `JoinWebUrl eq 'https://teams.example.test/l/meetup-join/a''b'`, and returns `mtg-1`. An unknown URL gives not_found `no meeting found for that join URL`.
  - `Transcripts("mtg-1")` over 3 pages (each `@odata.nextLink` is `srv.URL + "/me/onlineMeetings/mtg-1/transcripts?$skiptoken=N"`) returns every item, mapped from `createdDateTime` and `endDateTime`, in 3 requests.
  - Over 11 pages: service `more than 10 pages of transcripts`, after exactly 10 list requests.
  - `TranscriptContent("mtg-1","t1")` sends the decoded query `$format` = `text/vtt` and returns the fixture byte for byte.
  - A body of exactly 10 485 760 bytes is returned whole.
  - A body of 10 485 762 bytes gives service `transcript is larger than the 10 MiB limit`.
  - No error message from these calls contains `synthetic-token`.
- [X] T019 [P] [US1] Write a RED link-rule test in internal/adapters/graph/meetings_link_test.go. Use the pattern of page_token_host_test.go, with new names (`meetingsRecorder`): `Base` `https://graph.microsoft.com/v1.0`, and a `RoundTripper` that records the origin of each request and answers the first request with `{"value":[],"@odata.nextLink":LINK}`. For each `LINK` in `https://graph.microsoft.com.attacker.example/v1.0/x`, `https://evilgraph.microsoft.com/v1.0/x` and `http://graph.microsoft.com/v1.0/x`:
  - `Transcripts` gives service `refused a transcript list link outside Microsoft Graph`;
  - the recorded origins are exactly `["https://graph.microsoft.com"]`.
- [X] T020 [P] [US1] Write a RED budget test in internal/adapters/graph/meetings_budget_test.go. It runs `meetings.Save` with `HTTPMeetings` against the TLS fake. The fake holds the online event, `mtg-1`, 10 pages with one in-window transcript on page 10, and the fixture content.
  - The save succeeds after exactly 13 requests.
  - The same with `--join-url` takes exactly 12.
  - With content of 10 485 762 bytes, it fails after 13 requests, and no 14th arrives (SC-006).
- [X] T021 [P] [US1] Write RED CLI tests in internal/adapters/cli/meetings_save_test.go. `Deps` holds a fake `meetings.Store`, a session with all five consents (stored as `Blob{Usable:true, Mail:true, Teams:true, Calendar:true, Files:true, Meetings:true}` through `Store.Put`), and `Write: fs.WriteFile`.
  - `m365 meetings transcript --event evt-online --out DIR/m.vtt` exits 0. Stdout decodes to exactly the keys `path,id,created,ended,bytes,speakers`, with the values from T017's happy case. The file equals the fixture. Neither stdout nor stderr contains `Opening the review.`.
  - `--human` prints the same object, indented by two spaces.
  - Exit codes: both selectors exit 3, an unknown `--id` exits 6, and an existing file without `--overwrite` exits 3 with stderr JSON exactly `{"class":"usage","message":"destination exists; pass --overwrite"}`.
  - `m365 meetings`, `m365 meetings --help` and `m365 meetings transcript --help` print `meetingsHelp`, exactly the text in contracts/command-catalog.md, exit 0, and make no store calls.
  - With `Deps.Meetings == nil`: usage `meetings not available`.
  - `rootHelp` contains the line `  meetings  Teams meeting transcripts`.
  - Over MCP, `m365_run` with `namespace=meetings`, `verb=transcript` and flags `{"event":"evt-online","out":P}`, and no write opt-in, returns the summary JSON (no `dry_run` key), and the file exists (FR-020). Use the existing helpers `connectMCP` and `toolText` (internal/adapters/cli/mcp_test.go) and `testDeps`, as in `TestMCPRunChatAliasAndDownloadPathOnly`.
- [X] T022 [P] [US1] Write Gherkin features/meetings/save.feature, plus the step handlers acceptance/steps/meetings_harness.go and acceptance/steps/meetings_save_steps.go.
  - The harness starts a TLS fake from a `meetingsFixture` struct (events, meetings by join URL, transcript pages, and content by id as `{status int; body []byte}`), so that later stories configure it without editing the harness. It wires `HTTPMeetings{HTTPClient: &graph.HTTPClient{Base: srv.URL, Token: "synthetic-token", Client: srv.Client()}}`, signs in with a fake `Login` returning all five consents, writes only under `w.T.TempDir()`, and skips mode checks on Windows.
  - Scenarios:
    - save by event: the file equals the fixture, and stdout lists both speakers;
    - save by join URL;
    - several transcripts with `--id`;
    - an existing file without `--overwrite` exits 3;
    - overwriting a `0644` file leaves it `0600`;
    - a transcript without speaker names exits 5 and no file is written.
  - Step text has no arguments (`runtime.Parse` rejects them).
- [X] T023 [US1] Run `go test ./internal/app/meetings/ ./internal/adapters/graph/ ./internal/adapters/cli/` and `make acceptance`. Confirm that every new failure is an assertion failure (T016 may pass), then make a RED commit with T016–T022 only.

### Implementation for User Story 1

- [X] T024 [US1] Implement `Save` in internal/app/meetings/transcripts.go, following data-model.md "Use cases" steps 1–13 in order. Use unexported helpers `checkTarget`, `resolve` (returns the meeting id and an optional window), `inScope` (with `--event`, `occ.Start − 1h ≤ created ≤ occ.End + 1h`, inclusive), `sortNewest` (newest `Created` first, ties by `ID`) and `choose`. Parse times with `time.RFC3339Nano`; a time that does not parse is service `invalid transcript`. The use case never reads the clock. Keep the file under 250 lines.
- [X] T025 [US1] Implement `HTTPMeetings` in internal/adapters/graph/httpmeetings.go, per contracts/graph-mapping.md:
  - Every request goes through `c.request(ctx, http.MethodGet, url, nil, "", prefer)`. The prefer value is `outlook.timezone="UTC"` for the event read and empty otherwise.
  - A status of 400 or above reads the body and returns `MapGraphError(status, body)` for now (US4 replaces this).
  - Event times use the layout `2006-01-02T15:04:05.9999999` in UTC.
  - The lookup escapes the URL as research R3 describes.
  - The list loop allows at most 10 pages. A link is followed only if `u.Scheme == "https"` and `strings.EqualFold(u.Host, baseHost)`; the check runs before the request.
  - Content is read through `io.LimitReader(res.Body, 10<<20+1)`.
  - Decode failures give service `invalid event`, `invalid meeting list` or `invalid transcript`.
  - Keep the file under 250 lines.
- [X] T026 [US1] Implement internal/adapters/cli/meetings.go:
  - `runMeetings(args, d, format)`: print help for no args, `help`, `--help` or `-h`. Fail with usage `meetings not available` when `d.Meetings == nil`.
  - Read the session, then dispatch the verb `transcript`. An unknown verb gives usage `unknown meetings verb "X"`.
  - `transcript` returns help when `hasHelp(args)`. Its flags are `--event`, `--join-url`, `--id`, `--out` and `--overwrite`, parsed with `parseMixed`. `d.Write` falls back to `fs.WriteFile`, and success goes through `success`.
  - Add `case "meetings": return runMeetings(rest, d, format)` in internal/adapters/cli/run.go.
- [X] T027 [US1] Add the `meetingsHelp` constant (exact text from contracts/command-catalog.md) and the root help namespace line `  meetings  Teams meeting transcripts` in internal/adapters/cli/help.go.
- [X] T028 [US1] Wire `d.Meetings = &graph.HTTPMeetings{HTTPClient: httpc}` in the live branch of cmd/m365/main.go only. The `M365_FAKE` branch leaves it nil.
- [X] T029 [US1] Run `go test ./...` and `make acceptance` (features/meetings/save.feature), then make a GREEN commit with T024–T028 (internal/app/meetings/, internal/adapters/graph/httpmeetings.go, internal/adapters/cli/, cmd/m365/main.go).

**Checkpoint**: The save works end to end against the fake, with the caps, the link rule and the request budget enforced. Live use still needs US3's `--meetings` sign-in.

---

## Phase 4: User Story 2 — See which transcripts a meeting has (Priority: P2)

**Goal**: `m365 meetings transcripts (--event ID | --join-url URL)` lists the in-scope transcripts, newest first.

**Independent Test**: A fake meeting with three transcripts across two occurrences. `--event` lists only those in the window, `--join-url` lists all three newest first, and a meeting with none gives `{"count":0,"items":[]}`.

### Scaffold and tests for User Story 2 (MANDATORY)

- [ ] T030 [US2] SCAFFOLD, then a SCAFFOLD commit: `List(ctx, st Store, sess domain.Session, t Target) (domain.TranscriptList, error)`, returning a zero value, in internal/app/meetings/transcripts.go.
- [ ] T031 [P] [US2] Write RED app tests in internal/app/meetings/list_test.go, using a fake store in the style of T017:
  - Transcripts at `14:00:00Z`, `15:05:00Z`, `13:59:59Z` and `16:30:01Z`, with `--event`, give exactly `Count 2`, items `15:05` then `14:00`.
  - With `--join-url`, they give all four, newest first.
  - Equal `Created` sorts by `ID` ascending.
  - No transcripts gives `Count 0`, and `json.Marshal` of the result is exactly `{"count":0,"items":[]}`.
  - The consent and target errors are the same as T017's. `Out` is not required. List never calls `TranscriptContent`.
- [ ] T032 [P] [US2] Write a RED budget test in internal/adapters/graph/meetings_list_budget_test.go: `meetings.List` with `--event` over 10 pages takes exactly 12 requests, and with `--join-url` exactly 11.
- [ ] T033 [P] [US2] Write RED CLI tests in internal/adapters/cli/meetings_list_test.go:
  - `m365 meetings transcripts --event evt-online` prints exactly `{"count":1,"items":[{"id":"t1","created":"2030-01-15T15:05:00Z","ended":"2030-01-15T15:29:00Z"}]}` plus a newline.
  - The empty case prints `{"count":0,"items":[]}`.
  - `--human` prints the same object indented.
  - `m365 meetings transcripts --help` prints `meetingsHelp` with exit 0.
  - Both selectors exit 3.
- [ ] T034 [P] [US2] Write Gherkin features/meetings/list.feature and acceptance/steps/meetings_list_steps.go, using the US1 harness without editing it. Scenarios: list by event (window only), list by join URL (all, newest first), and an empty meeting gives count 0.
- [ ] T035 [US2] Run `go test ./internal/app/meetings/ ./internal/adapters/graph/ ./internal/adapters/cli/` and `make acceptance`, confirm the assertion failures, then make a RED commit with T031–T034 (the list tests and features/meetings/list.feature).

### Implementation for User Story 2

- [ ] T036 [US2] Implement `List` in internal/app/meetings/transcripts.go, reusing `checkTarget`, `resolve`, `inScope` and `sortNewest`. `Items` is never `nil`.
- [ ] T037 [US2] Add the verb `transcripts` to `runMeetings` in internal/adapters/cli/meetings.go, with flags `--event` and `--join-url`, and `hasHelp` handling.
- [ ] T038 [US2] Run `go test ./...` and `make acceptance`, then make a GREEN commit with T036–T037 (internal/app/meetings/transcripts.go, internal/adapters/cli/meetings.go).

**Checkpoint**: Both verbs work against the fake.

---

## Phase 5: User Story 3 — Opt in to meetings consent without changing ordinary sign-in (Priority: P2)

**Goal**: `m365 auth login --meetings` (with or without `--device-code`) requests the two meetings scopes and records `meetings` consent. A plain login is unchanged, and `auth status` reports `meetings`.

**Independent Test**: A plain login requests exactly today's scopes and reports `meetings: false`. A `--meetings` login uses the meetings login function and reports `meetings: true`. Meetings verbs without the consent exit 4 with the hint.

### Scaffold and tests for User Story 3 (MANDATORY)

- [ ] T039 [US3] SCAFFOLD in internal/adapters/graph/oauth.go, internal/adapters/graph/oauth_loopback.go, internal/adapters/graph/device.go and internal/adapters/cli/run.go, then a SCAFFOLD commit:
  - `var MeetingsScopes = []string{}` in internal/adapters/graph/oauth.go;
  - a variadic `extra ...string` on `PKCEConfig`, `RealLogin` and `RealDeviceLogin`, unused for now;
  - a `requested []string` parameter on `blobFromToken`, unused for now, with both callers (oauth_loopback.go and device.go) passing `cfg.Scopes`;
  - `MeetingsLogin` and `MeetingsDeviceLogin auth.LoginFn` in `Deps` in internal/adapters/cli/run.go.
- [ ] T040 [P] [US3] Write RED tests in internal/adapters/graph/meetings_login_test.go. They call `blobFromToken` directly with tokens whose `AccessToken` is `""`, so that `fetchMe` never calls live Graph.
  - `PKCEConfig("c","t","")` returns exactly `["openid","offline_access","profile","Mail.Read","Mail.Send","Chat.Read","Chat.ReadWrite","ChatMessage.Send","Calendars.ReadWrite","Files.ReadWrite"]` (SC-005).
  - `PKCEConfig("c","t","", MeetingsScopes...)` returns that list plus `OnlineMeetings.Read` and `OnlineMeetingTranscript.Read.All`.
  - `MeetingsScopes` is exactly those two.
  - `blobFromToken` sets `Meetings` as follows:
    - scope `"Mail.Read OnlineMeetings.Read OnlineMeetingTranscript.Read.All"` gives `true` with either requested list;
    - scope `"OnlineMeetings.Read"` alone gives `false`;
    - scope `""` with the requested list including both gives `true`;
    - scope `""` with the plain list gives `false`;
    - scope `"https://graph.microsoft.com/OnlineMeetings.Read https://graph.microsoft.com/OnlineMeetingTranscript.Read.All"` gives `true`.
- [ ] T041 [P] [US3] Write RED CLI tests in internal/adapters/cli/meetings_login_test.go (do not edit auth_test.go):
  - `auth login --meetings` calls only `MeetingsLogin`.
  - `auth login --device-code --meetings` and `auth login --meetings --device-code` call only `MeetingsDeviceLogin`.
  - A plain `auth login` calls only `Login`.
  - `auth login --meetings --meetings` gives usage `unknown login flag "--meetings"`.
  - A nil `MeetingsLogin` gives auth `login not available`.
  - After a login whose blob has `Meetings: true`, `auth status` `namespaces` decodes to exactly `{"mail":true,"teams":true,"calendar":true,"files":true,"meetings":true}`.
  - After a plain login, `meetings` is `false`.
  - Without meetings consent, `meetings transcript --join-url https://teams.example.test/x --out P` (the US1 verb, so this test doesn't depend on US2) exits 4 with stderr exactly `{"class":"auth","message":"missing meetings consent","hint":"run m365 auth login --meetings"}`.
  - `rootHelp` contains `Sign in: m365 auth login [--device-code] [--meetings]`.
  - Over MCP, `m365_status` shows `meetings`.
- [ ] T042 [P] [US3] Write Gherkin features/meetings/consent.feature and acceptance/steps/meetings_consent_steps.go. Scenarios:
  - a plain login reports meetings false;
  - a meetings login reports meetings true;
  - a meetings verb without meetings consent exits 4.
- [ ] T043 [US3] Run `go test ./internal/adapters/graph/ ./internal/adapters/cli/` and `make acceptance`, confirm the assertion failures, then make a RED commit with T040–T042 (the login tests and features/meetings/consent.feature).

### Implementation for User Story 3

- [ ] T044 [US3] In internal/adapters/graph/oauth.go, set `MeetingsScopes = []string{"OnlineMeetings.Read", "OnlineMeetingTranscript.Read.All"}`, and make `PKCEConfig` append `extra` to a copy of the base list.
- [ ] T045 [US3] In internal/adapters/graph/oauth_loopback.go, `blobFromToken` sets `Meetings` from the granted scope (`strings.Contains` for both names), or from `requested` when `scope == ""`, independent of `AccessToken`. `RealLogin` passes `extra` to `PKCEConfig`. In internal/adapters/graph/device.go, `RealDeviceLogin` does the same. The other flags are unchanged.
- [ ] T046 [US3] In internal/adapters/cli/auth.go:
  - `runLogin` loops over the flags, accepting `--device-code` and `--meetings` each at most once; anything else gives usage `unknown login flag "X"`.
  - It picks `Login`, `DeviceLogin`, `MeetingsLogin` or `MeetingsDeviceLogin`.
  - `nsMap` adds `"meetings": st.MeetingsConsented`.
- [ ] T047 [US3] Add the root help line `Sign in: m365 auth login [--device-code] [--meetings]` in internal/adapters/cli/help.go. In cmd/m365/main.go, wire `d.MeetingsLogin = graph.RealLogin(cfg.ClientID, cfg.TenantID, graph.MeetingsScopes...)` and `d.MeetingsDeviceLogin = graph.RealDeviceLogin(cfg.ClientID, cfg.TenantID, graph.MeetingsScopes...)` in the live branch only.
- [ ] T048 [US3] Add the property `"meetings": { "type": "boolean" }` to `namespaces` in specs/001-m365-cli/contracts/json-stdout.schema.json. This is additive, as spec 002 did, and the only edit to a 001 file.
- [ ] T049 [US3] Run `go test ./...` and `make acceptance`, then make a GREEN commit with T044–T048 (internal/adapters/graph/, internal/adapters/cli/, cmd/m365/main.go, specs/001-m365-cli/contracts/json-stdout.schema.json).

**Checkpoint**: MVP complete (US1 + US3). The live check (T062) can run now.

---

## Phase 6: User Story 4 — Know exactly why a transcript can't be read (Priority: P3)

**Goal**: Tenant refusals name `GraphAccessToTranscriptsDisabled` or `SpeakerAttributionNotAllowed` with exit 4, and other 401 and 403 errors carry Graph's codes. No file is written. (Scenario 4, no speaker names, is covered in US1.)

**Independent Test**: For each synthetic refusal, the command exits with the documented class, names the cause, and writes no file.

### Scaffold and tests for User Story 4 (MANDATORY)

- [ ] T050 [US4] SCAFFOLD, then a SCAFFOLD commit: `func mapMeetingsError(status int, body []byte) error` in internal/adapters/graph/meetingserrors.go, returning `MapGraphError(status, body)`.
- [ ] T051 [P] [US4] Write RED table tests in internal/adapters/graph/meetingserrors_test.go for `mapMeetingsError`, asserting the class, exact message and exact hint:
  - 403 with `{"error":{"code":"Forbidden","message":"tenant detail","innerError":{"code":"GraphAccessToTranscriptsDisabled"}}}` gives auth `GraphAccessToTranscriptsDisabled: Graph access to meeting transcripts is turned off for this tenant`, hint `a Teams administrator controls this setting`.
  - The same with the key `innererror` gives the same result.
  - 403 with inner `SpeakerAttributionNotAllowed` gives auth `SpeakerAttributionNotAllowed: speaker attribution is turned off for this tenant`, with the same hint.
  - 403 with code `Forbidden` and inner `Other` gives auth `missing consent or session (code: Forbidden, inner: Other)`.
  - 401 with code `InvalidAuthenticationToken` only gives `missing consent or session (code: InvalidAuthenticationToken)`.
  - 403 with an empty body gives `missing consent or session`.
  - 404 gives not_found `not found`.
  - 500 gives service `microsoft 365 service error`.
  - No message contains `tenant detail`.
- [ ] T052 [P] [US4] Write RED adapter tests in internal/adapters/graph/meetings_refusal_test.go against the TLS fake:
  - A list that returns 403 `GraphAccessToTranscriptsDisabled` gives that auth error, and no further request is made.
  - Content that returns 403 `SpeakerAttributionNotAllowed` gives that auth error after exactly one content request; there is no retry and no other format (FR-008).
  - An event read that returns 403 with inner `Other` gives the coded message.
- [ ] T053 [P] [US4] Write RED CLI tests in internal/adapters/cli/meetings_refusal_test.go, using `HTTPMeetings` against a TLS fake in the test:
  - `meetings transcript --event evt-online --out P` with a content 403 `SpeakerAttributionNotAllowed` exits 4. Stderr is exactly `{"class":"auth","message":"SpeakerAttributionNotAllowed: speaker attribution is turned off for this tenant","hint":"a Teams administrator controls this setting"}`. `P` does not exist.
  - `meetings transcripts` with a list 403 `GraphAccessToTranscriptsDisabled` exits 4 with that message.
- [ ] T054 [P] [US4] Write Gherkin features/meetings/refusals.feature and acceptance/steps/meetings_refusal_steps.go, using the US1 harness's fixture struct. Scenarios:
  - Graph access off: the list exits 4 and stderr names the code;
  - speaker attribution off: the save exits 4, stderr names the code, and no file is written.
- [ ] T055 [US4] Run `go test ./internal/adapters/graph/ ./internal/adapters/cli/` and `make acceptance`, confirm the assertion failures, then make a RED commit with T051–T054 (the refusal tests and features/meetings/refusals.feature).

### Implementation for User Story 4

- [ ] T056 [US4] Implement `mapMeetingsError` in internal/adapters/graph/meetingserrors.go per research R5:
  - Decode `error.code` and `error.innerError.code`.
  - The two tenant codes give `&domain.Error{Class: auth, Message, Hint}`.
  - Any other 401 or 403 gives the coded auth message, listing only the parts present.
  - Everything else goes to `MapGraphError`.
  - Graph's `message` is never used.
- [ ] T057 [US4] In internal/adapters/graph/httpmeetings.go, replace the four `MapGraphError(status, body)` calls with `mapMeetingsError(status, body)`. The shared mapper is not touched.
- [ ] T058 [US4] Run `go test ./...` and `make acceptance`, then make a GREEN commit with T056–T057 (internal/adapters/graph/meetingserrors.go, internal/adapters/graph/httpmeetings.go).

**Checkpoint**: Every refusal in the command catalog names its cause.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T059 [P] Document the feature in README.md and docs/m365.md:
  - the `meetings` namespace, its two verbs and both selectors;
  - the `--meetings` sign-in and why it is opt-in (admin-consented scopes);
  - the event-window rule and `--id`;
  - the 10-page and 10 MiB caps;
  - the tenant error codes, and that a Teams administrator owns them;
  - that `--out` is required, saves are owner-only, and `files download` overwrites are now owner-only too.

  Commit as `docs: meetings transcripts`. Docs are not behaviour, so no RED is needed.
- [ ] T060 Run the gates (Makefile, scripts/filesize.sh) and record their output in Notes of specs/009-meetings-transcripts/tasks.md:
  - `make verify`;
  - `make acceptance-mutation` (note that it runs no mutator);
  - `go test -race ./internal/app/meetings/ ./internal/adapters/graph/ ./internal/adapters/cli/`;
  - `sh scripts/filesize.sh`;
  - the final `wc -l` of every file touched.

  Any file over 250 lines needs a note under "### File-size notes" in specs/001-m365-cli/plan.md. Any file over 500 lines must be split first.
- [ ] T061 Review the branch diff against `main` and save the reports in ~/.work/m365/2026-11-10-teams-meeting-transcripts/reviews/:
  - run `/code-review` against `main` (Standards and Spec);
  - run an independent Codex adversarial review of the branch diff through the `codex:codex-rescue` agent;
  - save both reports in `~/.work/m365/2026-11-10-teams-meeting-transcripts/reviews/`;
  - fix confirmed findings with their own RED/GREEN commits, never by editing locked tests.
- [ ] T062 Live check (SC-007), per quickstart.md "Live check":
  - Build to a scratch path outside the repository: `go build -o "$SCRATCH/m365" ./cmd/m365`.
  - Ask Mason to run `! "$SCRATCH/m365" auth login --meetings`.
  - Then run steps 2–5 against the meeting he named in the task. Save it to `$SCRATCH`, never the repository. Report only the `speakers` array and the file mode.
  - **Stop** and report if the sign-in stops at admin approval, if a tenant code appears, or if the lookup finds no meeting or returns a 403 without a tenant code. The last case means returning to design.
  - Record the outcome in Notes without real names or transcript text. Write "speaker count N, matched the expected attendees: yes/no".
- [ ] T063 Run every row of quickstart.md's "Automated validation" table, and mark each pass in Notes.
- [ ] T064 Ask Mason before any push, PR or merge, and merge only on his explicit yes. Update `~/.work/m365/2026-11-10-teams-meeting-transcripts/task.md` with the commits, gates, reviews and the live-check outcome.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (T001–T003)** has no dependencies. T003 may stop the work for Mason's approval.
- **Foundational (T004–T014)** depends on Setup. F1 (T004–T007) comes before F2 (T008–T014). Both block every story.
- **US1 (T015–T029)** depends on Foundational.
- **US2 (T030–T038)** depends on US1, because it reuses the helpers in `transcripts.go`, the adapter and the harness.
- **US3 (T039–T049)** depends on Foundational only, so it can run before or after US1 and US2. It touches `run.go` and `help.go` too, so don't run it in parallel with US1's GREEN.
- **US4 (T050–T058)** depends on US1, because it changes the adapter's error path.
- **Polish (T059–T064)** depends on every story. The live check T062 needs US1 + US3 at least.

### Within Each Story

SCAFFOLD commit → the [P] test tasks (different files) → the RED commit after confirming the assertion failures → the GREEN tasks in order → the GREEN commit.

## Parallel Examples

```text
Foundational F2 tests:  T009 (internal/domain/webvtt_test.go) ‖ T010 (internal/app/auth/meetings_consent_test.go)
Foundational F2 GREEN:  T012 (internal/domain/webvtt.go) ‖ T013 (internal/app/auth/ports.go)
US1 tests:              T016 ‖ T017 ‖ T018 ‖ T019 ‖ T020 ‖ T021 ‖ T022   (seven files)
US2 tests:              T031 ‖ T032 ‖ T033 ‖ T034
US3 tests:              T040 ‖ T041 ‖ T042
US4 tests:              T051 ‖ T052 ‖ T053 ‖ T054
```

## Implementation Strategy

1. **MVP**: Setup → Foundational → US1 → US3. Stop and validate with `make verify`, then run the live check (T062), which proves the goal: a named-speaker transcript saved from a real meeting.
2. **Increment**: US2 (list), then US4 (named refusals). Each ends with a GREEN commit and `make verify`.
3. **Finish**: Polish (docs, gates, reviews), then Mason's yes before any push.

## Notes

- Pre-existing, not fixed here:
  - `scripts/acceptance-mutation.sh` runs no mutator;
  - `TestDeviceLoginPromptsAndReturnsSession` (internal/adapters/graph/device_test.go) uses the access token `tok`, so `fetchMe` sends a request to live `graph.microsoft.com/v1.0/me` during `go test`. The new tests avoid this by using an empty `AccessToken`.
- This list refines plan.md's single `features/meetings/transcripts.feature` and `acceptance/steps/meetings_steps.go` into one feature and one step file per story, plus a shared harness, so that a later story never edits a locked file.
- Baseline, the test-impact scan, approvals, gate output and the live-check outcome are recorded below as the work proceeds.

### Log

- **T001 (2026-11-10)**:
  - Branch `009-meetings-transcripts` at `3a54f61` (spec `68b3d34`, plan `6b1356b`, tasks `3a54f61`), on `main@4877baf`.
  - `git status` shows only `?? .tools`.
  - The git identity comes from the repository config (`.git/config`): Mason Huemmer, a GitHub noreply address.
- **T002 baseline**:
  - `make verify` exit 0 in 31 s. govulncheck: 0 affecting. Acceptance: 66 advisory dry-check findings in 15 features, all pass. CRAP: ok (500 functions, 44 baselined).
  - Line counts: run.go 235, help.go 165, cli/auth.go 75, app/auth/ports.go 105, oauth_loopback.go 155, fs.go 45, main.go 96.
  - Gates that shape the code: every Go file, tests included, at most 250 lines (scripts/filesize.sh); CRAP at most 15 per function (scripts/crap.sh).
- **T003**: The scratch change (`meetings` in `nsMap`, a root help line, `fs.WriteFile` with `O_EXCL`, `O_TRUNC` and `Chmod`) followed by `go test ./...` gave no failures. No locked test is affected, so no approval is needed. The files are restored.
- **F1**: RED `29d6cb4`. Only `TestWriteFileOverwriteIsOwnerOnly` failed (`mode 644, want 600`); the two guards passed. GREEN in the next commit; fs, cli and files tests pass.
- **F2**: SCAFFOLD `3adff06`, RED `809a30c` (all failures were assertions; `TestOlderBlobHasNoMeetingsConsent` passed as a guard), then GREEN. domain, auth and cli tests pass.
- **US1**:
  - SCAFFOLD `a0c4776`. RED `fb0ff19`: every failure was an assertion or the expected unknown namespace. The isolation guard passed. One extra test from the plan's performance contract, `TestTranscriptContentReadsAtMostOneByteOverTheCap` (read count at most 10 MiB + 1), was added before the RED commit.
  - Approved edit: gosec G302 flagged the deliberate `0o644` in the locked step at acceptance/steps/meetings_save_steps.go:54. **Mason approved on 2026-11-10** a comment-only `#nosec G302` with a reason, committed alone as `21a2fa3`.
  - GREEN `e0325e0`. The CRAP ratchet flagged `Run` (internal/adapters/cli/run.go) rising from 21.6 to 22.6 because of the new `case`. Per the baseline's rule ("fix a function when you change it"), GREEN routes workload namespaces through `namespaceRunners()`, built per call to avoid an initialization cycle through `runMCP`. `Run` is now under its baselined score.
  - `make verify` exit 0: gosec 0 issues, govulncheck 0 affecting, crap ok (525 functions, 44 baselined). run.go 238 lines, help.go 180.

