# Graph mapping and fake server: meetings

Only `internal/adapters/graph` knows these paths and shapes. Every request goes through the existing `HTTPClient.request`. That means the existing bearer and refresh handling, one retry after a forced refresh on 401, and no new headers except `Prefer` on the event read.

## Requests

### 1. Event (with `--event` only)

```text
GET {base}/me/events/{url.PathEscape(eventID)}?$select=start,end,isOnlineMeeting,onlineMeeting
Prefer: outlook.timezone="UTC"
```

Fields read: `isOnlineMeeting` (bool), `onlineMeeting.joinUrl` (string), `start.dateTime` and `end.dateTime` (UTC, no offset, optional fraction).

### 2. Meeting lookup

```text
GET {base}/me/onlineMeetings?$filter={url.QueryEscape("JoinWebUrl eq '" + join with ' doubled + "'")}
```

Fields read: `value[0].id`. An empty `value` means not found.

### 3. Transcript list

```text
GET {base}/me/onlineMeetings/{url.PathEscape(meetingID)}/transcripts
then GET {@odata.nextLink} while present, for at most 10 pages in total
```

- Fields read from each item: `id`, `createdDateTime` and `endDateTime`.
- A link is requested only if it is `https` and its host equals the base host.

### 4. Content (`transcript` only)

```text
GET {base}/me/onlineMeetings/{mid}/transcripts/{url.PathEscape(tid)}/content?$format=text/vtt
```

The body is read through `io.LimitReader(body, 10 MiB + 1)`. Reading 10 MiB + 1 bytes means the content is over the cap.

## Error mapping (`mapMeetingsError`)

Every non-2xx status on requests 1 to 4 goes through `mapMeetingsError`:

```json
{"error":{"code":"Forbidden","message":"<ignored>","innerError":{"code":"GraphAccessToTranscriptsDisabled"}}}
```

The rules are in [command-catalog.md](command-catalog.md), rows 10 to 12. Everything else goes to the existing `MapGraphError`: 404 and the not-found codes give not_found, and other statuses give the service class.

## Fake server (tests and acceptance)

The fake is an `httptest.NewTLSServer`. Its `srv.Client()` is set as `HTTPClient.Client`, and its URL as `HTTPClient.Base`. The bearer token is `synthetic-token`. The fake counts requests, records each path and query, and records the bytes it writes for content.

| Route | Fixture behaviour |
|---|---|
| `GET /me/events/evt-online` | an online event, `start` `2030-01-15T15:00:00.0000000`, `end` `2030-01-15T15:30:00.0000000`, `joinUrl` `https://teams.example.test/l/meetup-join/abc` |
| `GET /me/events/evt-offline` | `isOnlineMeeting: false` |
| `GET /me/events/evt-missing` | 404 `ErrorItemNotFound` |
| `GET /me/onlineMeetings?$filter=…` | matches the decoded filter literal. A known URL gives `[{"id":"mtg-1"}]`; others give `[]` |
| `GET /me/onlineMeetings/mtg-1/transcripts` | the window for `evt-online` is `14:00Z`–`16:30Z`, inclusive. Transcripts are created at `13:59:59Z` (outside), `14:00Z` (inside, start edge), `15:05Z` (inside), `16:30Z` (inside, end edge) and `16:30:01Z` (outside), in the subset each test needs |
| `…/transcripts` paged | `@odata.nextLink` back to the TLS host with `?$skiptoken=N`; one variant serves 11 pages |
| `…/transcripts/{tid}/content` | a synthetic WebVTT (below); variants are non-VTT, no voice tags, 10 MiB + 2 bytes, and 403 with each inner code |

### Synthetic WebVTT fixture

Fictional names only. It has a BOM, a cue identifier, a multi-line cue, a `NOTE` block, a class on a voice tag, and a cue with no speaker:

```text
﻿WEBVTT

NOTE synthetic fixture

1
00:00:01.000 --> 00:00:03.000
<v Avery Example>Opening the review.</v>

00:00:03.500 --> 00:00:06.000 
<v.loud Blake Sample>Two lines here,
and a second line.</v>

00:00:06.500 --> 00:00:07.000
(inaudible)

00:00:07.500 --> 00:00:09.000
<v Avery Example>Back to the agenda.</v>
```

The expected `speakers` are `["Avery Example","Blake Sample"]`. The saved file equals the fixture byte for byte.

### Recording-transport test (continuation links)

The same pattern as `page_token_host_test.go`. The `HTTPClient` has `Base: "https://graph.microsoft.com/v1.0"` and a `RoundTripper` that records origins. The first page returns each of these as `@odata.nextLink`:

- `https://graph.microsoft.com.attacker.example/...`
- `https://evilgraph.microsoft.com/...`
- `http://graph.microsoft.com/...`

Each one must give the service error, and the transport must show exactly one origin, `https://graph.microsoft.com`.
