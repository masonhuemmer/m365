# Quickstart: Teams meeting transcripts

How to prove the feature works. The automated scenarios use synthetic fixtures only ([contracts/graph-mapping.md](contracts/graph-mapping.md)). The live check runs once, by hand, against a real meeting, and nothing from it enters the repository.

## Prerequisites

- A worktree on `009-meetings-transcripts`, with the pinned tools in `.tools/` (`scripts/install-tools.sh`, or a symlink to the main checkout's `.tools`).
- No account is needed for the automated runs.
- For the live check: an app registration whose tenant has granted admin consent for `OnlineMeetings.Read` and `OnlineMeetingTranscript.Read.All`, and a Teams meeting from the last 60 days that the user was invited to and that had transcription on.

## Automated validation

| Scenario (spec) | Command | Expected |
|---|---|---|
| WebVTT rules, speaker names | `go test ./internal/domain/ -run 'WebVTT\|VoiceNames'` | pass; the fixture gives `["Avery Example","Blake Sample"]` |
| Consent flag and hint (US3) | `go test ./internal/app/auth/ ./internal/adapters/graph/ -run 'Meetings\|Blob\|Login'` | pass; a plain login's scope list is unchanged |
| Selection and checks (US1, US2) | `go test ./internal/app/meetings/` | pass; window edges inside, ±1 s outside; several → usage with hint |
| Graph mapping, caps, links, tenant codes (US4, SC-006) | `go test ./internal/adapters/graph/ -run 'Meetings\|Transcript'` | pass; request count ≤ 13; no request after a failure |
| Owner-only save (FR-009, FR-022) | `go test ./internal/adapters/fs/` | pass on Unix; the mode assertion is skipped on Windows |
| CLI streams, exit codes, help, MCP (FR-018–FR-021) | `go test ./internal/adapters/cli/ -run 'Meetings\|Login\|Status\|MCP'` | pass; no transcript text on stdout or stderr |
| Gherkin | `make acceptance` | `features/meetings/transcripts.feature` passes |
| Full gate | `make verify` then `make acceptance-mutation` | both exit 0 (the mutation target does not run a mutator; see plan) |

## Live check (SC-007)

Run it once, after the automated validation passes. Use a scratch directory outside the repository, and never commit what it produces.

1. Sign in with meetings consent:

   ```sh
   m365 auth login --meetings
   m365 auth status            # namespaces.meetings is true
   ```

   **Stop if** sign-in stops at an administrator-approval prompt. That means the tenant has not granted the transcript scope. Report it. A plain `m365 auth login` is unaffected.

2. Find the event id:

   ```sh
   m365 calendar list --start DAY_START --end DAY_END   # RFC 3339 with offset
   ```

3. List the transcripts for that occurrence:

   ```sh
   m365 meetings transcripts --event EVENT_ID
   ```

   - **Stop if** the error names `GraphAccessToTranscriptsDisabled` or `SpeakerAttributionNotAllowed`. Report the code, because a Teams administrator owns the setting.
   - **Stop and return to design if** the result is `no meeting found for that join URL`, or a 403 without a tenant code. The undocumented attendee lookup has failed.
   - If `count` is 0, transcription was off for that meeting. Ask the user to name another meeting.

4. Save the transcript:

   ```sh
   SCRATCH="$(mktemp -d)"
   m365 meetings transcript --event EVENT_ID --out "$SCRATCH/meeting.vtt"
   ls -l "$SCRATCH/meeting.vtt"       # -rw-------
   head -c 6 "$SCRATCH/meeting.vtt"   # WEBVTT
   ```

   Add `--id` if step 3 listed several transcripts.

5. Report the `speakers` array from the summary to the user. Don't paste transcript text into notes, commits or issues. The file stays in the scratch directory, and the user decides where it goes next.

## Expected failure outputs (synthetic)

These match the error table in [command-catalog.md](contracts/command-catalog.md). For example, with no meetings consent:

```json
{"class":"auth","message":"missing meetings consent","hint":"run m365 auth login --meetings"}
```

The exit code is 4.
