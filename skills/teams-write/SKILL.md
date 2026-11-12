---
name: teams-write
description: Write Teams chat messages as HTML or converted markdown so they render as paragraphs, lists, and links.
---

# teams-write

Write a Teams chat message that renders: paragraphs, lists, links. No session required to read this recipe.

Use --html with real HTML, or --format md with the documented subset (# / ## / ###, **bold**, - / * / 1. lists, [label](url), fenced code, blank-line paragraphs). --format md converts that subset to HTML. --html posts the body as HTML already. Plain text keeps its paragraphs and line breaks.

  teams send --to Ajay --format md --text 'The change is in UAT.

- Rollback is the previous chart.

See [the ticket](https://example.com/ticket).' --dry-run
  teams send --to Ajay --html --text '<p>The change is in UAT.</p><ul><li>Rollback is the previous chart.</li></ul>' --dry-run

MCP (flags.text string, not text-file=-):

  m365_run namespace=teams verb=send flags to=Ajay format=md text='The change is in UAT.

- Rollback is the previous chart.'
  m365_run namespace=teams verb=send flags to=Ajay html=true text='<p>The change is in UAT.</p><ul><li>Rollback is the previous chart.</li></ul>'

Mention someone by typing @Name in the text: @Ajay, @Ajay Mathew or @ajay.mathew. Only an @ at the start of a word counts, and the name is matched against the members of that chat, so emails and code are left alone. Dry-run shows mentions (who gets notified) and unresolved_mentions (an @name that matched nobody, left as plain text). Two members with the same first name fail before sending: write the full name. @everyone is not supported. Mentions are skipped inside code and links.

Teams: --attach PATH (repeatable) uploads each file to your OneDrive (Microsoft Teams Chat Files), gives every other chat member read access without emailing them, and posts the message with a file card. Dry-run lists the files and share_with (who gets access): check it first. A chat member with no email address makes the send fail before anything is uploaded. Notes (--note-to-self) keeps the file private to you. If sharing or posting fails, nothing is reported as sent; the error names any file already uploaded.

  teams send --to Ajay --text 'Report attached.' --attach ~/report.pdf --dry-run

MCP: flags attach=[/path/report.pdf]; requires write_opt_in for the real send.

Group chats: teams create --name NAME --member EMAIL_OR_ID (repeat, at least 2) starts a named group chat and teams rename CHAT_ID --name NAME renames one (group chats only). Members are exact emails or user IDs, never display names, and you are added automatically. Dry-run shows the name and members (rename shows old_name): check it, because create makes a new chat on every real run. Neither sends a message; post with teams send and the returned id. MCP: write_opt_in is required, --allow takes teams.create and teams.rename, and the user must have signed in with Chat.ReadWrite.

  teams create --name 'NOC Weekly' --member ajay@example.com --member jarrett@example.com --dry-run
  teams rename CHAT_ID --name 'NOC Daily' --dry-run

Hide a chat: teams hide CHAT_ID removes it from your own list only (teams unhide undoes it). Nothing is deleted, other members still see it, and a new message in the chat shows it again; teams list skips hidden chats unless --include-hidden (then they carry hidden: true), which is how to find the id to unhide; non-admins cannot delete chats. Dry-run shows the chat name and type. MCP: write_opt_in is required and --allow takes teams.hide and teams.unhide.

  teams hide CHAT_ID --dry-run

React to a message: teams react CHAT_ID MESSAGE_ID REACTION adds an emoji reaction; REACTION is like, heart, laugh, surprised, sad, angry, eyes, or an emoji such as 🎉. --remove takes it back. teams react --to Ajay --last eyes reacts to the newest message someone else sent, among the 50 messages Graph returns first; teams react CHAT_ID --last works the same in a known chat. Message IDs come from teams messages, which also lists each message's reactions. Dry-run shows chat_id, message_id and reaction (with --last also from, created and text): check it is the message you mean. MCP: write_opt_in is required, --allow takes teams.react, and with --exact-recipients --to must be an exact email or user ID.

  teams react --to Ajay --last eyes --dry-run

Do not post one run-on --text string with markdown left unconverted. Adaptive Cards and Graph beta markdown are out of scope.
Happy-path examples stay dry-run (write_opt_in false).
