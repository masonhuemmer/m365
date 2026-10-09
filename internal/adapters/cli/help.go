package cli

const rootHelp = `m365 — Microsoft 365 CLI for one signed-in user

Usage: m365 <namespace> <verb> [flags]

Namespaces:
  auth      sign in, status, logout (core, not a workload)
  mail      Outlook mail
  teams     Teams chats (alias: chat)
  calendar  own calendars and events
  files     own OneDrive
  mcp       stdio MCP for agents (serve)

Output: JSON on stdout by default. --human for text. Diagnostics on stderr.
Exit classes: 0 success; 3 usage/config; 4 auth; 5 service; 6 not-found.

List limits: --top default 10 (mail list, calendar list), 20 (teams list, calendar calendars, files list); max 50.
Attachments: max 10 MiB per file, max 10 files per send/reply.
Upload: max 100 MiB per file.
`

const calendarHelp = `m365 calendar — own calendars and events

Verbs: calendars, list, get, create, update, delete, free
Create: --subject --when 'tomorrow at 1:30 pm' --until --duration (default 30 minutes) --dry-run
Free: --when (default tomorrow) --duration (default 30 minutes) --hours 9:00-17:00 --top (default 5, max 20)
Working hours 09:00-17:00 local; 30-minute grid. --dry-run does not apply to free.
Flags: --calendar --start --end --top (events default 10, calendars default 20, max 50) --page-token
Window for list: default now through +7 days.
Output: JSON (default) or --human.
`

const filesHelp = `m365 files — own OneDrive

Verbs: root, list, get, download, upload, create-folder, delete, move
Flags: --folder --top (default 20, max 50) --page-token --out --overwrite --file --name --dry-run
Upload: max 100 MiB per file. Download requires --out; existing file refused without --overwrite.
Output: JSON (default) or --human. No file bytes on stdout.
`

const mailListHelp = `m365 mail list — list messages

Flags: --folder (default inbox) --unread --search --top (default 10, max 50) --page-token
Well-known folders: inbox, sentitems, drafts, all
Output: JSON (default) or --human. Exit 0 empty list.
`

const mailWatchHelp = `m365 mail watch — poll one folder for incremental message changes

Flags: --folder (default inbox) --include-existing --classify
Classification target: --target-address and --target-name (repeatable).
The first poll records a quiet baseline unless --include-existing is set.
Folder-scoped only: the mailbox-wide all value is not supported.
JSON output is one body-free mail.changed object per line; empty polls write no lines.
Experimental response classification is disabled by default and transfers a bounded text thread to the configured provider.
Classified output is mail.response_classified; actionable is a routing hint, not authorization to reply.
The cursor is committed only after the complete delta round is emitted.
`

const mailSendHelp = `m365 mail send — send mail

Required: --to --subject --body or --body-file
Optional: --cc --attach (repeatable) --html --format md --dry-run --preview --note-to-self
Plain text keeps paragraphs and line breaks (sent as HTML).
--format md converts a markdown subset to HTML: # / ## / ### headings, **bold**, - / * / 1. lists, [label](url), inline code in backticks, fenced code.
--html sends the body as HTML unchanged. Not both.
--preview shows the message as text in a box and never sends; not with --json.
--note-to-self sends to the signed-in mailbox (default subject Note to self). Do not combine with --to.
Attachment caps: 10 MiB per file, 10 files. --top N/A.
Output modes: --json (default) --human
`

const mailReplyHelp = `m365 mail reply — reply to a message

Required: MESSAGE_ID --body or --body-file
Optional: --all --attach (repeatable) --html --format md --dry-run --preview
--format md converts a markdown subset to HTML: # / ## / ### headings, **bold**, - / * / 1. lists, [label](url), inline code in backticks, fenced code.
--preview shows the message as text in a box and never sends; not with --json.
Plain text keeps paragraphs and line breaks. --html sends the body as HTML unchanged.
Both go in the Graph reply comment, so the quoted thread stays below.
--attach includes local files on reply and --all; all files are read before sending.
Unreadable files or Graph rejection fail without a text-only fallback.
Attachment caps: 10 MiB per file, 10 files (local limits; Graph may reject the payload).
Output modes: --json (default) --human
`

const teamsListHelp = `m365 teams list — list chats

Flags: --top (default 20, max 50) --page-token
Output: JSON (default) or --human.
`

const teamsMessagesHelp = `m365 teams messages — read messages in a chat

Required: CHAT_ID (from teams list or teams find)
Flags: --top (default 20, max 50) --page-token --include-system
Each item has from: the sender's display name, or the app name for bot posts.
System events (member added, call ended) are skipped unless --include-system.
Output: JSON (default) or --human.
`

const teamsFindHelp = `m365 teams find — find a chat by person or group

Examples: teams find Ajay
          teams find --group NOC
Flags: --group --top (default 10, max 20)
Person query prefers 1:1. Matching yourself returns Notes (48:notes); Graph list omits it.
Unique 1:1 hits are remembered so later --to skips a full scan.
--group matches group topic/members.
Scan ceiling 10 pages of 50 chats; incomplete=true if not finished.
Empty list exit 0. Never sends.
Output: JSON (default) or --human.
`

const teamsSendHelp = `m365 teams send — send a chat message

Required: CHAT_ID or --to Ajay, plus --text or --text-file
Examples: teams send --to Ajay --text ping --dry-run
Optional: --attach (repeatable) --html --format md --dry-run --preview --note-to-self
--preview shows the message as text in a box and never sends; not with --json.
--note-to-self posts to Teams Notes (48:notes). Do not combine with --to or a chat id.
Plain text keeps paragraphs and line breaks (sent as HTML).
Type @Name (@Ajay, @Ajay Mathew) to mention a member of that chat; --dry-run shows mentions and unresolved_mentions. A name matching two members fails; an unknown @name stays plain text.
--format md converts a markdown subset to HTML (headings, **bold**, lists, links, inline code in backticks, fenced code). --html posts the body as HTML already. Not both.
--to and chat id together exit 3. Attachment caps: 10 MiB per file, 10 files.
--attach uploads each file to your OneDrive ("Microsoft Teams Chat Files"), gives every other chat member read access (nobody is emailed), and posts the message with a file card. Notes (--note-to-self) keeps the file private. A member with no email address fails the send before anything is uploaded; --dry-run shows share_with.
Output modes: --json (default) --human
`
