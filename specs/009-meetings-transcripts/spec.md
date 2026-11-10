# Feature Specification: Teams meeting transcripts

**Feature Branch**: `009-meetings-transcripts`

**Created**: 2026-11-10

**Status**: Draft

**Input**: Approved design `2026-11-10-teams-meeting-transcripts/design.md` (kept outside the repo). The signed-in user downloads the Teams transcript of a meeting they were invited to, as a WebVTT file with named speakers, so they can keep it as evidence of who said what.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Save the transcript of a meeting I was invited to (Priority: P1)

The user knows a meeting from their calendar. They ask m365 for that occurrence's transcript and get a WebVTT file, exactly as Teams produced it, in which every line names its speaker. m365 also prints a short summary of the saved file, including the speaker names it found. The transcript text never appears in the output.

**Why this priority**: This is the whole goal. A Teams transcript names its speakers, so it is better evidence than a local recording, which files everything said in the room under the user's name.

**Independent Test**: Against a synthetic meeting service holding one calendar event, one meeting and one attributed transcript, run `m365 meetings transcript --event EVENT --out FILE`. The file holds the transcript byte for byte, the summary lists the fictional speakers, and stdout holds no transcript text.

**Acceptance Scenarios**:

1. **Given** meetings consent, calendar consent and an event whose occurrence has exactly one transcript, **When** the user saves with `--event`, **Then** the file holds that transcript byte for byte, and the summary reports its path, id, start and end times, size and speaker names.
2. **Given** a meeting join URL instead of an event, **When** the user saves with `--join-url` and the meeting has exactly one transcript, **Then** the same result follows.
3. **Given** a meeting with several transcripts, **When** the user saves with `--id` naming one of them, **Then** exactly that transcript is saved.
4. **Given** an existing file at the destination, **When** the user saves without `--overwrite`, **Then** nothing is written and a usage error says to pass `--overwrite`.
5. **Given** an existing file readable by other users, **When** the user saves with `--overwrite`, **Then** the saved file is readable and writable by its owner only.

---

### User Story 2 - See which transcripts a meeting has (Priority: P2)

Before saving, or after being told a meeting has several transcripts, the user lists them with their start and end times, newest first, and picks one by id.

**Why this priority**: A transcription that stops and restarts produces several transcripts, and a recurring series may share one meeting. Listing lets the user choose instead of m365 guessing.

**Independent Test**: With a synthetic meeting holding three transcripts across two occurrences, `m365 meetings transcripts --event EVENT` lists only the transcripts that started within the event's window, and `--join-url` lists all three, newest first.

**Acceptance Scenarios**:

1. **Given** a meeting with transcripts, **When** the user lists by `--event`, **Then** only transcripts that started between 1 hour before the event's start and 1 hour after its end are listed, each with id, start and end times, newest first.
2. **Given** the same meeting, **When** the user lists by `--join-url`, **Then** every transcript of the meeting is listed.
3. **Given** a meeting with no transcripts, **When** the user lists, **Then** an empty list is returned with a count of zero.

---

### User Story 3 - Opt in to meetings consent without changing ordinary sign-in (Priority: P2)

Reading transcripts needs a permission that only a tenant administrator can grant. The user opts in at sign-in with `m365 auth login --meetings`. A plain sign-in keeps asking for exactly what it asks for today, so users whose tenant has not granted the permission keep every existing command.

**Why this priority**: Asking for an administrator-only permission at every sign-in would lock users out of mail, Teams, calendar and files in tenants without that grant.

**Independent Test**: With a synthetic sign-in, a plain login requests today's permissions and reports `meetings: false`. A `--meetings` login requests the two additional permissions and reports `meetings: true`. Meetings commands without meetings consent fail with an authentication error whose hint names `m365 auth login --meetings`.

**Acceptance Scenarios**:

1. **Given** a plain sign-in, **When** the user checks `auth status`, **Then** meetings consent is reported as false, and the requested permissions are unchanged from before this feature.
2. **Given** a sign-in with `--meetings` that grants both permissions, **When** the user checks `auth status`, **Then** meetings consent is reported as true.
3. **Given** a sign-in with `--meetings` whose response does not list the granted permissions, **When** the user checks `auth status`, **Then** meetings consent is reported as true, because the requested permissions count as granted.
4. **Given** `--meetings` together with `--device-code`, **When** the user signs in, **Then** the device-code flow requests the meetings permissions as well.
5. **Given** no meetings consent, **When** the user runs either meetings command, **Then** it fails with an authentication error and the hint `run m365 auth login --meetings`.

---

### User Story 4 - Know exactly why a transcript can't be read (Priority: P3)

When the tenant blocks transcript access, or the transcript has no speaker names, the user gets an error that names the cause precisely enough to take to their IT department. No file is written.

**Why this priority**: Two tenant settings that block reading are off by default, and the service reports both as a generic "forbidden". Without the specific code, the user can't ask for the right fix.

**Independent Test**: For each synthetic refusal, the command exits with the documented class, names the cause, and writes no file.

**Acceptance Scenarios**:

1. **Given** the tenant has Graph access to transcripts turned off, **When** the user lists or saves, **Then** the command exits with the authentication class, names `GraphAccessToTranscriptsDisabled`, and says a Teams administrator controls the setting.
2. **Given** the tenant has speaker attribution turned off, **When** the user saves, **Then** the command exits with the authentication class, names `SpeakerAttributionNotAllowed`, and does not fall back to a transcript without names.
3. **Given** any other refusal, **When** the user lists or saves, **Then** the error includes the service's error code and inner code when present.
4. **Given** a transcript without any speaker names, **When** the user saves, **Then** the command exits with the service class, says the transcript has no speaker names, and writes nothing.

---

### Edge Cases

- Both `--event` and `--join-url`, or neither: a usage error before any service call.
- A join URL that is not an `https` URL: a usage error before any service call.
- `--out` missing on save: a usage error before any service call.
- The event does not exist: not found.
- The event is not an online meeting: not found, saying so.
- No meeting matches the join URL (expired, organised elsewhere, or never a Teams meeting): not found.
- No transcript in scope (for example, transcription was off): not found for save; an empty list for list.
- Several transcripts in scope and no `--id`: a usage error naming `meetings transcripts` and `--id`.
- `--id` names a transcript the meeting doesn't have: not found.
- A join URL containing a single quote: escaped, so it can't change the meeting lookup.
- More than 10 pages of transcripts: a service error naming the cap; nothing is silently dropped.
- A continuation link that is not `https`, or not on the service's own host: a service error, and that link is never requested.
- Transcript content over 10 MiB: a service error, and nothing is written.
- Content that is not WebVTT: a service error, and nothing is written.
- Real-world WebVTT with cue identifiers, multi-line cues, or lines without a speaker next to named lines: saved unchanged; speakers are read from the named lines.
- `--help` on either verb shows that verb's help, never an argument error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: m365 MUST provide a `meetings` namespace with two verbs: `transcripts` (list) and `transcript` (save).
- **FR-002**: Both verbs MUST take exactly one of `--event EVENT_ID` or `--join-url URL`. Otherwise they MUST fail with a usage error before contacting the service.
- **FR-003**: With `--event`, m365 MUST read the event itself and use its online meeting join URL and its scheduled start and end. An event without an online meeting MUST be reported as not found.
- **FR-004**: m365 MUST find the meeting from its join URL. A join URL MUST be an `https` URL, and MUST be escaped so it cannot alter the lookup.
- **FR-005**: With `--event`, only transcripts whose start falls between 1 hour before the event's start and 1 hour after its end are in scope. With `--join-url`, all of the meeting's transcripts are in scope.
- **FR-006**: `transcripts` MUST output a count and items (id, start, end), newest first, in JSON by default and with `--human`.
- **FR-007**: `transcript` MUST save the single in-scope transcript, or the one named by `--id`. With none in scope it MUST report not found. With several in scope and no `--id`, it MUST fail with a usage error that hints `meetings transcripts` and `--id`.
- **FR-008**: `transcript` MUST request the attributed (speaker-named) WebVTT form, and MUST NOT fall back to a form without names.
- **FR-009**: `transcript` MUST save the content byte for byte. `--out` is required. Without `--overwrite`, the file MUST be created exclusively, so an existing path is refused. With `--overwrite`, the saved file MUST be owner-only, even when the previous file was not.
- **FR-010**: `transcript` MUST output path, id, start, end, size in bytes, and the distinct speaker names in order of first appearance. It MUST NOT output any transcript text.
- **FR-011**: Content MUST start with the WebVTT header (a byte-order mark is allowed) and MUST carry at least one speaker name. Otherwise m365 MUST fail with a service error and write nothing.
- **FR-012**: m365 MUST read at most 10 MiB of content, and at most 10 pages of the transcript list. Exceeding either MUST fail with a service error naming the limit.
- **FR-013**: m365 MUST follow a continuation link only when it is `https` and on the service's own host. Any other link MUST fail with a service error and MUST NOT be requested.
- **FR-014**: `m365 auth login --meetings` MUST request the two meetings permissions in addition to today's permissions, MUST combine with `--device-code`, and MUST record meetings consent. A plain `m365 auth login` MUST request exactly today's permissions.
- **FR-015**: Meetings consent MUST be true when the sign-in grants both meetings permissions, or when the sign-in response omits the granted list after a `--meetings` request. `auth status` MUST report meetings consent alongside the existing namespaces.
- **FR-016**: Both verbs MUST require meetings consent. `--event` MUST also require calendar consent. A missing consent MUST fail with the authentication class, and the meetings hint MUST name `m365 auth login --meetings`.
- **FR-017**: A refusal naming `GraphAccessToTranscriptsDisabled` or `SpeakerAttributionNotAllowed` MUST fail with the authentication class, name the code, and say a Teams administrator controls the setting. Any other authentication or permission refusal on a meetings call MUST include the service's error code and inner code when present.
- **FR-018**: Error classes and exit codes MUST be the existing ones: usage 3, authentication 4, service 5, not found 6.
- **FR-019**: Tokens MUST NOT appear in any output, error, file or child process. Error messages MUST pass the existing redaction.
- **FR-020**: Both verbs MUST be available to agents through the existing MCP run tool as reads, not writes.
- **FR-021**: Root help, `meetings` help, each verb's `--help`, `auth` help, the README and `docs/m365.md` MUST describe the verbs, the `--meetings` sign-in, the in-scope rule, the limits and the error codes.
- **FR-022**: Other namespaces' behaviour and error messages MUST be unchanged. The one exception is the shared file save, which becomes owner-only on overwrite (FR-009) for `files download` too.

### Key Entities

- **Meeting**: a Teams online meeting, named by a calendar event or by its join URL.
- **Transcript**: Teams' timed record of what was said in a meeting, made each time transcription runs. It has an id, a start time and an end time. A meeting can have several.
- **Speaker attribution**: the name of the person who said a transcript line, carried on that line.
- **Meetings consent**: the signed-in user's opt-in grant to read meeting transcripts, separate from mail, Teams, calendar and files consent.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Starting from a calendar event id, the user saves a meeting's transcript with one command, and the summary lists every speaker named in the file.
- **SC-002**: 100% of saved transcripts carry at least one speaker name, and are byte-for-byte identical to what the service returned.
- **SC-003**: In every test, no transcript text appears on stdout or stderr, and no token appears in any output, error or file.
- **SC-004**: Each refusal in User Story 4 names its cause, and in 100% of those cases no file is written.
- **SC-005**: A plain sign-in requests exactly the same permissions as before this feature.
- **SC-006**: A command makes at most 13 requests to the service (1 event, 1 lookup, up to 10 list pages, 1 content), reads at most 10 MiB of content, and makes no further requests after a failure.
- **SC-007**: A live check, after `m365 auth login --meetings`, saves the transcript of a real meeting the user was invited to, to a path outside the repository, and reports its speaker names.

## Assumptions

- The tenant administrator has granted the meetings permissions to the user's app registration. Otherwise `--meetings` sign-in stops at an administrator-approval prompt, and a plain sign-in is unaffected.
- The service lets invited attendees, not only the organiser, read a meeting's transcripts and use the join-URL lookup. Microsoft documents the first. The second is undocumented and is confirmed by the live check.
- Meetings expire about 60 days after they end, and expired meetings cannot be read.
- Meetings without a calendar event, ad hoc calls, recordings and attendance reports are out of scope.
- No new request timeout is added; the existing commands have none.
- Fixtures use fictional names and synthetic transcripts only. No real transcript, real name, token or tenant identifier is committed.
