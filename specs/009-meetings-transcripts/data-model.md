# Data Model: Teams meeting transcripts

The entities come from the spec's Key Entities. Graph JSON shapes stay in the adapter ([contracts/graph-mapping.md](contracts/graph-mapping.md)), and the types below use the workload's own terms.

## Domain (`internal/domain`)

### Transcript (`meeting.go`)

| Field | Type | JSON | Rule |
|---|---|---|---|
| ID | string | `id` | not empty; opaque, never parsed |
| Created | string | `created` | RFC 3339 UTC instant, from Graph `createdDateTime` |
| Ended | string | `ended` | RFC 3339 UTC instant from Graph `endDateTime`; may be empty while transcription runs |

### TranscriptList (`meeting.go`)

| Field | Type | JSON | Rule |
|---|---|---|---|
| Count | int | `count` | equals `len(Items)` |
| Items | []Transcript | `items` | newest `Created` first, ties by `ID`; never `null` (an empty list is `[]`) |

### SavedTranscript (`meeting.go`)

| Field | Type | JSON | Rule |
|---|---|---|---|
| Path | string | `path` | the `--out` value as given |
| ID, Created, Ended | string | `id`, `created`, `ended` | copied from the chosen Transcript |
| Bytes | int | `bytes` | length of the content written |
| Speakers | []string | `speakers` | at least one; distinct; order of first appearance |

The summary has no field for transcript text.

### WebVTT rules (`webvtt.go`), pure functions

- `IsWebVTT(b []byte) bool`: an optional UTF-8 BOM (`EF BB BF`), then `WEBVTT`, then the end of input or one of space, tab, `\r`, `\n`.
- `VoiceNames(b []byte) []string`: every voice span opening tag `<v(\.[^\s>]+)*\s+([^>]+)>` in cue text. Blocks are split at blank lines. The header block, `NOTE`, `STYLE` and `REGION` blocks, and a cue's id and timing lines are skipped. Each name is trimmed, empty names are dropped, and the distinct names are returned in order of first appearance.

### Session (`session.go`)

The session gains `MeetingsConsented bool`. The four existing flags are unchanged.

## Auth (`internal/app/auth`)

- `Blob.Meetings bool` with JSON key `meetings`. Stored blobs without the key read as `false`.
- `Status` maps `Blob.Meetings` to `Session.MeetingsConsented`.
- `RequireMeetings(sess)` runs the signed-in and usable checks from `Require`, then fails with the auth class, the message `missing meetings consent` and the hint `run m365 auth login --meetings`.

### The meetings consent flag

| Login | Granted `scope` in the token response | `meetings` |
|---|---|---|
| plain | anything | true only if the granted scope names both meetings scopes |
| `--meetings` | names both meetings scopes | true |
| `--meetings` | omitted or empty | true (the requested scopes count as granted) |
| `--meetings` | names one or neither | false |
| any | not signed in | false |

The token refresher never changes the flag.

## Meetings app (`internal/app/meetings`)

### Port

```text
Store
  EventMeeting(ctx, eventID)            -> Occurrence, error
  FindMeeting(ctx, joinURL)             -> meetingID string, error
  Transcripts(ctx, meetingID)           -> []domain.Transcript, error   # all pages, capped in the adapter
  TranscriptContent(ctx, meetingID, id) -> []byte, error                # capped in the adapter
```

`Occurrence` holds `JoinURL string`, `Start time.Time` and `End time.Time`, all UTC.

### Target (the CLI's input)

| Field | Rule |
|---|---|
| EventID | exactly one of EventID or JoinURL is set |
| JoinURL | an absolute `https` URL with a host |
| ID | optional; the chosen transcript id (save only) |
| Out | required for save |
| Overwrite | passed to the write port |

### Use cases

`List(ctx, st, sess, t) (TranscriptList, error)` and `Save(ctx, st, sess, t, write) (SavedTranscript, error)` run these steps in order. Each step either passes or ends the command.

1. `RequireMeetings(sess)` → auth.
2. Check the target. Both `--event` and `--join-url`, or neither, is a usage error; so is a join URL that isn't `https`. On save, a missing `--out` is a usage error.
3. With `--event`, `RequireCalendar(sess)` → auth.
4. Find the meeting. With `--event`, call `EventMeeting`, then `FindMeeting(occ.JoinURL)`. With `--join-url`, call `FindMeeting` directly.
5. `Transcripts(meetingID)`, then sort them newest first.
6. Scope. With `--event`, keep the transcripts with `occ.Start − 1h ≤ created ≤ occ.End + 1h`. With `--join-url`, keep them all.
7. **List** returns the in-scope transcripts here.
8. **Save** chooses a transcript:
   - with `--id`, the transcript with that id in the full list, or not found;
   - otherwise exactly one in scope, or not found when there are none, or a usage error with a hint when there are several.
9. `TranscriptContent(meetingID, chosen.ID)`.
10. If `IsWebVTT` fails, return a service error.
11. If `VoiceNames` is empty, return a service error. Nothing is written.
12. `write(Out, content, Overwrite)`.
13. Return `SavedTranscript`.

No step reads the clock. Graph times that don't parse are a service error.

## Graph adapter (`internal/adapters/graph`)

`HTTPMeetings struct{ *HTTPClient }` implements `meetings.Store`. Its private decode types (event, meeting list, transcript list) never leave the adapter. The caps live here:

| Cap | Value | Failure |
|---|---|---|
| List pages | 10 | service error: `more than 10 pages of transcripts`; the 11th link is not requested |
| Continuation link | `https`, host equal to the base host | service error: `refused a transcript list link outside Microsoft Graph`; the link is not requested |
| Content | 10 MiB (10 485 760 bytes) | service error: `transcript is larger than the 10 MiB limit`; at most one byte over the cap is read |

## State transitions

None. Both verbs are reads. The only side effect is the file that `Save` writes in step 12.
