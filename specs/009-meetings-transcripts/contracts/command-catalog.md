# Command catalog: meetings

Streams and conventions are the existing ones. Result data goes to stdout as JSON by default, or as indented JSON with `--human`. Errors go to stderr as `{"class","message","hint"?}`, and the exit code follows the class: 3 usage, 4 auth, 5 service, 6 not found. Global flags (`--json`, `--human`, `--verbose`) behave as in every other namespace.

## `m365 auth login [--device-code] [--meetings]`

| Flags | Login function | Scopes requested |
|---|---|---|
| none | `Login` | today's list, unchanged |
| `--device-code` | `DeviceLogin` | today's list, unchanged |
| `--meetings` | `MeetingsLogin` | today's list + `OnlineMeetings.Read` + `OnlineMeetingTranscript.Read.All` |
| `--device-code --meetings`, in either order | `MeetingsDeviceLogin` | same as `--meetings` |

- Any other flag, or a repeated flag, gives a usage error `unknown login flag "X"`.
- The output keeps the existing shape. `namespaces` gains the key `meetings`.

## `m365 auth status`

`namespaces` is `{"mail","teams","calendar","files","meetings"}`, all booleans. The other fields are unchanged. Over MCP, `m365_status` shows the same key.

## `m365 meetings transcripts`

Lists a meeting's transcripts.

```text
m365 meetings transcripts (--event EVENT_ID | --join-url URL)
```

| Case | Exit | Output |
|---|---|---|
| Transcripts in scope | 0 | `{"count":N,"items":[{"id","created","ended"}…]}`, newest first ([schema](json-transcripts.schema.json)) |
| None in scope | 0 | `{"count":0,"items":[]}` |
| `--help`, or `meetings --help` | 0 | `meetingsHelp` |

## `m365 meetings transcript`

Saves one transcript.

```text
m365 meetings transcript (--event EVENT_ID | --join-url URL) --out FILE [--id TRANSCRIPT_ID] [--overwrite]
```

| Case | Exit | Output |
|---|---|---|
| Saved | 0 | `{"path","id","created","ended","bytes","speakers":[…]}` ([schema](json-transcript.schema.json)) |
| `--help` | 0 | `meetingsHelp` |

stdout never holds transcript text. The file holds the content byte for byte, with mode `0600` on Unix.

## Errors (both verbs unless marked)

Checked in this order. The first failure ends the command.

| # | Condition | Class | Message (hint) |
|---|---|---|---|
| 1 | Not signed in, or the session is unusable | auth | `not signed in` |
| 2 | No meetings consent | auth | `missing meetings consent` (`run m365 auth login --meetings`) |
| 3 | Both `--event` and `--join-url`, or neither | usage | `use exactly one of --event or --join-url` |
| 4 | `--join-url` is not an absolute `https` URL | usage | `--join-url must be an https URL` |
| 5 | `transcript` without `--out` | usage | `--out is required` |
| 6 | `--event` without calendar consent | auth | `missing calendar consent` |
| 7 | The event does not exist | not_found | `not found` |
| 8 | The event is not an online meeting | not_found | `event is not an online meeting` |
| 9 | No meeting matches the join URL | not_found | `no meeting found for that join URL` |
| 10 | Tenant: Graph access to transcripts is off | auth | `GraphAccessToTranscriptsDisabled: Graph access to meeting transcripts is turned off for this tenant` (`a Teams administrator controls this setting`) |
| 11 | Tenant: speaker attribution is off (`transcript`) | auth | `SpeakerAttributionNotAllowed: speaker attribution is turned off for this tenant` (`a Teams administrator controls this setting`) |
| 12 | Any other 401 or 403 on a meetings request | auth | `missing consent or session (code: C, inner: I)`, each part only when present |
| 13 | More than 10 list pages | service | `more than 10 pages of transcripts` |
| 14 | A continuation link that is not `https`, or not on the Graph host | service | `refused a transcript list link outside Microsoft Graph` |
| 15 | An unparseable Graph body or time | service | `invalid event`, `invalid meeting list` or `invalid transcript` |
| 16 | `--id` not among the meeting's transcripts (`transcript`) | not_found | `no transcript with that id` |
| 17 | No transcript in scope (`transcript`) | not_found | `no transcript in scope; transcription may have been off` |
| 18 | Several in scope and no `--id` (`transcript`) | usage | `several transcripts in scope` (`run m365 meetings transcripts, then pass --id`) |
| 19 | Content over 10 MiB (`transcript`) | service | `transcript is larger than the 10 MiB limit` |
| 20 | Content is not WebVTT (`transcript`) | service | `transcript is not WebVTT` |
| 21 | Content has no speaker names (`transcript`) | service | `transcript has no speaker names` |
| 22 | The file exists and there is no `--overwrite` (`transcript`) | usage | `destination exists; pass --overwrite` |
| 23 | The path can't be written (`transcript`) | usage | `unwritable path: FILE` |
| 24 | With `--overwrite`, the destination exists and is not a regular file (`transcript`) | usage | `destination is not a regular file: FILE` |

- In rows 10 to 24, no file is written. Rows 1 to 6 make no request.
- Errors pass the existing `redact`. Messages hold Graph codes, never Graph's free-text messages.

## Help text: `meetingsHelp`

`m365 meetings`, `m365 meetings --help` and `m365 meetings <verb> --help` all print the same text:

```text
m365 meetings — Teams meeting transcripts you were invited to

Verbs: transcripts (list), transcript (save one)
Name the meeting: --event EVENT_ID (from calendar list) or --join-url URL (https). Exactly one.
--event lists transcripts that started from 1 hour before the event to 1 hour after it ended.
--join-url lists all of the meeting's transcripts.
Save: --out FILE (required) --id TRANSCRIPT_ID (when several) --overwrite
The file is the attributed WebVTT, byte for byte, owner-only. No transcript text on stdout.
Limits: 10 pages of transcripts; 10 MiB of content.
Needs: m365 auth login --meetings (admin-consented scopes); --event also needs calendar consent.
Tenant codes (exit 4): GraphAccessToTranscriptsDisabled, SpeakerAttributionNotAllowed.
Output: JSON (default) or --human.
```

Root help gains one namespace line (`meetings  Teams meeting transcripts`) and the login line `Sign in: m365 auth login [--device-code] [--meetings]`. `m365 auth` with no verb still prints root help.

## MCP

- `m365_run namespace=meetings verb=transcripts|transcript` dispatches through the same `Run`.
- Neither verb is in `isWrite`, so neither gets the dry-run gate, the same as `files download`.
- The `--read-only` and `--allow` policies treat both verbs as reads.
- `auth login --meetings` stays a terminal-only command, because `auth login` is already forbidden over MCP.
