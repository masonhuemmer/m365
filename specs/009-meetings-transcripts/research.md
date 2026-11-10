# Research: Teams meeting transcripts

Phase 0 for [plan.md](plan.md). The Graph facts come from the task research outside the repo (Q1–Q7, with citations to the Microsoft Graph docs). This file records only the implementation decisions they lead to. The Technical Context has no NEEDS CLARIFICATION items left.

## R1. How to ask for the attributed WebVTT

- **Decision**: `GET …/transcripts/{tid}/content?$format=text/vtt`.
- **Rationale**:
  - The v1.0 transcript-content page documents `$format` as an optional query parameter that selects `text/vtt`.
  - WebVTT is also the default format, so the parameter only makes the request explicit.
  - The speaker-attribution setting refuses `text/vtt` requests with `SpeakerAttributionNotAllowed`. The command must surface that refusal, and it never asks for the unattributed format.
  - A query parameter needs no change to the shared `HTTPClient.request`, which has no header option today (constitution III).
- **Alternatives considered**:
  - An `Accept: text/vtt` header. This is the form the docs' examples use, but it needs a header parameter on the shared request, which five call sites use. It is held in reserve: if the live check shows that `$format` is ignored, it gets its own RED test.
  - Sending no format at all, and relying on the default. Rejected, because the design asks for an explicit request.

## R2. Reading the event's occurrence window

- **Decision**: `GET /me/events/{id}?$select=start,end,isOnlineMeeting,onlineMeeting`, with the header `Prefer: outlook.timezone="UTC"`.
  - `start.dateTime` and `end.dateTime` are read as UTC with the layout `2006-01-02T15:04:05.9999999`. They have no offset, and the fraction is optional.
  - An event with `isOnlineMeeting: false`, or with an empty `onlineMeeting.joinUrl`, is not found ("event is not an online meeting").
- **Rationale**:
  - Without the `Prefer` header, Graph returns times in a zone that depends on the request, and the existing calendar decoder drops the zone.
  - Asking for UTC makes the window deterministic. Transcript times (`createdDateTime`, `endDateTime`) are UTC with an offset (`Z`), so both sides compare in one zone.
  - An occurrence id from `calendar list` (calendarView) returns that occurrence's own start and end.
- **Alternatives considered**: Reuse `calendar.Store.GetEvent`. Rejected, because it imports another namespace's port (VII), and its decoder discards both the online-meeting fields and the zone.

## R3. Finding the meeting from a join URL

- **Decision**: `GET /me/onlineMeetings?$filter=` + `url.QueryEscape("JoinWebUrl eq '" + strings.ReplaceAll(join, "'", "''") + "'")`.
  - An empty `value` array is not found ("no meeting found for that join URL").
  - The first item's `id` is used, because a join URL names one meeting.
  - `--join-url` must parse as an absolute URL with scheme `https` and a host. Anything else is a usage error before any request.
- **Rationale**:
  - This is the documented delegated lookup.
  - OData escapes a single quote by doubling it, so a quote in the URL can't end the string literal.
  - `url.QueryEscape` of the whole expression matches the existing `conversationId` filter in `HTTPClient.Thread`, which works against live Graph.
- **Alternatives considered**: A `--meeting ID` flag. Rejected: nobody has a raw meeting id to hand (design decision 3).

## R4. Which continuation links to follow

- **Decision**: The meetings adapter follows `@odata.nextLink` only when `url.Parse` succeeds, the scheme is `https`, and the host equals the host of `HTTPClient.Base`, compared case-insensitively.
  - Any other link fails with a service error ("refused a transcript list link outside Microsoft Graph"), and that link is not requested.
  - After 10 pages, a remaining link fails with "more than 10 pages of transcripts", and it is not requested.
- **Rationale**: FR-013 requires `https` in every case. The shared `allowedNext`, as fixed in PR #53, also accepts the base's own scheme so that `http` test servers work. That is right for page tokens, but looser than FR-013. The check here is three lines.
- **Alternatives considered**: Reuse `allowedNext`. Rejected for the reason above. Changing `allowedNext` again is out of scope.

## R5. Which Graph errors to name

- **Decision**: `mapMeetingsError(status, body)` in the graph adapter:
  - It decodes `error.code` and `error.innerError.code`. Go's JSON decoding ignores key case, so `innererror` also matches.
  - An inner code of `GraphAccessToTranscriptsDisabled` gives the auth class, the message `GraphAccessToTranscriptsDisabled: Graph access to meeting transcripts is turned off for this tenant`, and the hint `a Teams administrator controls this setting`.
  - An inner code of `SpeakerAttributionNotAllowed` gives the auth class, the message `SpeakerAttributionNotAllowed: speaker attribution is turned off for this tenant`, and the same hint.
  - Any other 401 or 403 gives the auth class, with the message `missing consent or session (code: C, inner: I)`. Each part appears only when present.
  - Everything else goes to the existing `MapGraphError`, which maps 404 to not found and other statuses to the service class.
- **Rationale**:
  - The top-level code for both tenant switches is just `Forbidden`.
  - The message carries codes only, never Graph's free-text message, which can hold request ids or tenant details. The CLI's `redact` still runs on it.
  - The shared mapper stays unchanged, because other namespaces' messages must not change (FR-022).
- **Alternatives considered**: Extend `MapGraphError` for every namespace. Rejected: out of scope (design, Not doing).

## R6. How to find the speaker names

- **Decision**: `domain.VoiceNames(vtt []byte) []string` finds every WebVTT voice span opening tag, `<v` plus optional `.class` parts, then whitespace, then the annotation, then `>`. It trims each name and drops empty ones. It returns the distinct names in order of first appearance.
  - Only cue text is read: the header block, `NOTE`, `STYLE` and `REGION` blocks, and each cue's id and timing lines are skipped. A voice tag in a comment must not count as attribution (implementation review, Codex finding 5).
  - `domain.IsWebVTT(b)` accepts an optional UTF-8 BOM, then `WEBVTT`, followed by the end of input, a space, a tab, `\r` or `\n`.
- **Rationale**:
  - Teams writes `<v Name>text</v>`.
  - The WebVTT spec allows classes (`<v.loud Name>`).
  - Matching the tag anywhere in a cue payload covers multi-line cues, cue identifiers and `NOTE` blocks, without a full WebVTT parser.
  - Lines without a voice tag are kept, because the file is saved byte for byte.
  - Zero names fails the save (FR-011).
- **Alternatives considered**: A WebVTT parser library. Rejected: a new dependency for one regular expression (III).

## R7. Choosing a transcript

- **Decision**:
  - `--event`: in scope means `createdDateTime` is between `start − 1h` and `end + 1h`, inclusive at both ends.
  - `--join-url`: every transcript is in scope.
  - Sorting is by `createdDateTime`, newest first; equal times sort by `id`.
  - `--id` is matched against all of the meeting's transcripts, not only those in scope, so an explicit choice overrides the window. A missing id is not found, and no content request is sent.
  - The save:
    - with exactly one transcript in scope, saves it;
    - with none, fails as not found ("no transcript in scope; transcription may have been off");
    - with several and no `--id`, fails with a usage error, the hint `run m365 meetings transcripts, then pass --id`, and no content request.
  - A `createdDateTime` that doesn't parse is a service error ("invalid transcript").
- **Rationale**:
  - The spec says `--id` names a transcript "the meeting" has (spec Edge Cases).
  - The design keeps the window as a rule of thumb and lets the output show `created` and `ended`.
  - Listing before reading content also supplies `created` and `ended` for the save summary.
- **Alternatives considered**: Limit `--id` to in-scope transcripts. Rejected: it would make the documented workaround for late-started transcripts depend on `--join-url`.

## R8. Opt-in sign-in plumbing

- **Decision**:
  - `graph.MeetingsScopes = []string{"OnlineMeetings.Read", "OnlineMeetingTranscript.Read.All"}`.
  - `PKCEConfig(clientID, tenant, redirect string, extra ...string)` appends `extra` to the existing list.
  - `RealLogin` and `RealDeviceLogin` take `extra ...string` and pass it on.
  - `cli.Deps` gains `MeetingsLogin` and `MeetingsDeviceLogin`, which `main.go` wires with `MeetingsScopes`.
  - `runLogin` accepts `--device-code` and `--meetings` in any order, each at most once, and picks one of the four login functions. A nil function gives the existing auth error, "login not available".
  - `blobFromToken(ctx, tok, requested []string)` sets `Meetings` when the granted `scope` contains both `OnlineMeetings.Read` and `OnlineMeetingTranscript.Read.All`, or when `scope` is empty and `requested` contains both.
  - The refresher keeps the stored flags, as it does today.
- **Rationale**:
  - A plain login still calls `PKCEConfig` with no extra scopes, so its request is unchanged (SC-005).
  - The refresh grant sends no `scope`, so refreshing keeps the consented set.
  - Substring matching follows the existing flag style, and `OnlineMeetings.ReadWrite` would also satisfy the read check.
  - Two more `Deps` fields avoid changing `auth.LoginFn`, which every fake and test uses.
- **Alternatives considered**:
  - Add an options parameter to `auth.LoginFn`. Rejected: it touches every login fake for one flag.
  - Always request the scopes. Rejected: every login would stop at an admin-approval prompt in tenants without the grant (ADR 0004).

## R9. Saving the file

- **Decision**: In `fs.WriteFile`:
  - without `overwrite`, `os.OpenFile(path, O_WRONLY|O_CREATE|O_EXCL, 0o600)`, and an existing file gives the unchanged usage message `destination exists; pass --overwrite`;
  - with `overwrite`, `O_WRONLY|O_CREATE|O_TRUNC` with `0o600`, then `f.Chmod(0o600)` before writing;
  - the existing `MkdirAll` and the `unwritable path` message stay.
- **Rationale**:
  - Exclusive create removes the gap between checking and writing.
  - `os.WriteFile` keeps an existing file's mode, so a 0644 file stays 0644 after an overwrite. The explicit `Chmod` fixes that.
  - The messages stay the same, so `files download` keeps its error contract.
- **Alternatives considered**: Write to a temp file and rename it. Rejected: it changes the inode, the directory permission needs, and Windows rename semantics, for no requirement.
- **Implementation review (2026-11-10, approved)**: with `overwrite`, an existing destination that `Lstat` shows is not a regular file (a symlink, FIFO or device such as `/dev/stdout`) is refused before any change. The file is opened without `O_TRUNC` and truncated only after `Chmod(0o600)` succeeds, so a failed overwrite keeps the old content. A swap between `Lstat` and `OpenFile` remains possible; it would need someone else with write access to the destination's directory.
- **Windows**: `O_EXCL` works there, but `Chmod` only toggles read-only, so the mode assertion runs only on Unix.

## R10. The acceptance harness

- **Decision**: `acceptance/steps/meetings_steps.go` starts an `httptest.NewTLSServer` with synthetic routes and wires `HTTPMeetings{HTTPClient: &graph.HTTPClient{Base: srv.URL, Token: "synthetic-token", Client: srv.Client()}}` into `cli.Deps`. It signs in with a fake `MeetingsLogin`, runs `m365 meetings …`, and asserts on the exit code, the stdout JSON and the saved file. The steps write files only under `t.TempDir()`.
- **Rationale**: Acceptance then drives the real adapter, as `reply_attach_steps.go` does. A TLS server is needed because continuation links must be `https`.
- **Alternatives considered**: An in-memory `MeetingsAPI` in `graph.Memory`. Rejected: an extra fake with no requirement behind it (III), and it would not exercise the request mapping.
