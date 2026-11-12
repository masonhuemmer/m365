# Feature Specification: React to Teams chat messages

**Feature Branch**: `011-teams-message-reactions`

**Created**: 2026-11-12

**Status**: Draft

**Input**: Mason: "add a feature that allows me to append emojis to team messages so I could do a thumbs up or eyes icon to show I see your message or like it." Design answers (2026-11-12): pick a message by chat and message ID, or with `--to NAME --last`; accept short names and any emoji; `--remove` undoes a reaction; `teams messages` shows each message's reactions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - React to a message by its IDs (Priority: P1)

Mason reads a chat with `m365 teams messages`, picks a message, and adds 👍 or 👀 to it without opening Teams.

**Why this priority**: It is the request itself, and every other story builds on it.

**Independent Test**: `m365 teams react CHAT_ID MESSAGE_ID like` against the fake Graph exits 0 and reports `reacted: true` with `reaction: "👍"`; the fake records one reaction.

**Acceptance Scenarios**:

1. **Given** a chat message, **When** Mason runs `teams react CHAT_ID MESSAGE_ID like`, **Then** the message gets 👍 and stdout is `{chat_id, message_id, reaction: "👍", reacted: true}`.
2. **Given** a chat message, **When** Mason runs `teams react CHAT_ID MESSAGE_ID 🎉`, **Then** the message gets 🎉.
3. **Given** any target, **When** Mason adds `--dry-run`, **Then** stdout shows `dry_run: true`, `action: "react"` and the reaction, and nothing changes.
4. **Given** the reaction `hello` or `+1`, **When** Mason runs the command, **Then** it exits 3 (usage) and names the accepted reactions; nothing is sent.

---

### User Story 2 - React to the newest message from someone else (Priority: P1)

Mason wants to acknowledge Ajay's latest message with 👀 in one command: `m365 teams react --to Ajay --last eyes`.

**Why this priority**: It is the everyday "I saw it" case, and Mason asked for it.

**Independent Test**: In a fake chat where Ajay wrote an older message and Mason wrote the newest one, `teams react --to Ajay --last eyes` reacts to Ajay's message, not Mason's, and reports its sender and time.

**Acceptance Scenarios**:

1. **Given** a 1:1 chat with Ajay, **When** Mason runs `teams react --to Ajay --last eyes`, **Then** the newest message that Ajay (not Mason, not a system event) sent gets 👀, and stdout adds `from`, `created` and `text` of that message.
2. **Given** a chat ID, **When** Mason runs `teams react CHAT_ID --last like`, **Then** the same rule picks the message.
3. **Given** a chat where nobody else has written among the 50 messages Graph returns first, **When** Mason runs `--last`, **Then** it exits 6 (not found) and nothing is sent.
4. **Given** `--to` without `--last`, `--last` with a message ID, or `--to` with a chat ID, **When** Mason runs the command, **Then** it exits 3 (usage) and nothing is sent.

---

### User Story 3 - Remove a reaction (Priority: P2)

Mason reacted with the wrong emoji and takes it back: `teams react CHAT_ID MESSAGE_ID like --remove`.

**Why this priority**: It makes a mistake fixable from the CLI; it is less frequent than adding.

**Independent Test**: After a reaction in the fake, the same command with `--remove` reports `removed: true` and the fake no longer holds that reaction.

**Acceptance Scenarios**:

1. **Given** Mason's 👍 on a message, **When** he runs the command with `--remove`, **Then** Graph unsets 👍 and stdout has `removed: true`.
2. **Given** `--remove --dry-run`, **Then** stdout has `action: "remove"` and nothing changes.

---

### User Story 4 - See reactions when reading messages (Priority: P2)

`teams messages` shows who reacted with what, so Mason or an agent can check before reacting.

**Why this priority**: Mason asked for it; it also confirms a reaction from the CLI.

**Independent Test**: A fake message with a 👍 from Ajay shows `reactions: [{emoji: "👍", user_id, user: "Ajay Kumar", created}]` in `teams messages` output.

**Acceptance Scenarios**:

1. **Given** a message with reactions, **When** Mason runs `teams messages CHAT_ID`, **Then** each message lists its reactions with the emoji, the reacting user's ID and name, and the time.
2. **Given** a reaction Graph reports by a legacy name such as `like`, **Then** the output shows its emoji (👍).
3. **Given** a reaction without a user name from Graph, **Then** the name comes from the chat's members; with no match, `user` is left out and `user_id` stays.
4. **Given** a message without reactions, **Then** the `reactions` field is absent.

---

### User Story 5 - Agents react only when allowed (Priority: P2)

An agent using the m365 MCP server can react only with the user's write opt-in, like every other write.

**Why this priority**: A reaction is visible to the other people in the chat.

**Independent Test**: Over MCP, `teams react` without `write_opt_in` runs as a dry run; with opt-in and `--allow` not naming `teams.react`, it is refused.

**Acceptance Scenarios**:

1. **Given** an MCP run of `teams react` without `write_opt_in`, **Then** the result is a dry run and nothing changes.
2. **Given** `--allow` without `teams.react`, **When** an agent opts in, **Then** the run is refused (usage).
3. **Given** an MCP server started with `--exact-recipients`, **When** an agent passes `--to` with a display name, **Then** the run is refused; an exact email or user ID is accepted.

---

### Edge Cases

- Names are case-insensitive (`Like`, `EYES`).
- A ZWJ or skin-tone emoji (👩🏽‍💻, 👍🏽) is accepted; text, numbers alone, punctuation and spaces are refused before any Graph call.
- An emoji that Teams does not support: Graph refuses it, and the command exits 5 (Graph error) with Graph's message.
- A message or chat that does not exist: exit 6 (not found).
- Reacting twice with the same emoji: Graph decides; m365 sends one request each time.
- `--last` in Notes (`48:notes`) or a chat where m365 cannot find you among the members: exit 3, with a hint to pass a message ID.
- Graph lists messages by last change, so `--last` compares created times within the first 50 messages; a message older than all of those is not considered.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `m365 teams react` MUST add a reaction to one chat message, addressed as `CHAT_ID MESSAGE_ID REACTION`, `CHAT_ID --last REACTION`, or `--to NAME --last REACTION`.
- **FR-002**: REACTION MUST accept, case-insensitively, the names like 👍, heart ❤️, laugh 😆, surprised 😮, sad 😢, angry 😡 and eyes 👀, and any emoji given directly; it MUST refuse text before any Graph call.
- **FR-003**: `--last` MUST pick the newest message, by created time, that is neither a system event nor sent by the signed-in user, among the first 50 messages Graph returns; the limit MUST be stated in help.
- **FR-004**: `--remove` MUST remove the given reaction instead of adding it.
- **FR-005**: `--dry-run` MUST report the chat, message, reaction and action and change nothing.
- **FR-006**: Output MUST be JSON by default and human-readable with `--human`; exit codes MUST follow the CLI contract (0, 3 usage, 4 auth, 5 Graph, 6 not found).
- **FR-007**: `teams messages` MUST list each message's reactions as emoji, user ID, user name when known, and created time; legacy reaction names MUST map to their emoji.
- **FR-008**: Reactions MUST need no permission beyond what m365 already requests (`ChatMessage.Send`).
- **FR-009**: Over MCP, `teams react` MUST be a guarded write: dry run without `write_opt_in`, subject to `--allow teams.react` and to `--exact-recipients` for `--to`.
- **FR-010**: `teams react --help` MUST name the forms, the reaction names, `--last`'s limit, `--remove`, `--dry-run` and the output modes.

### Key Entities

- **Reaction**: an emoji on a chat message from one user: emoji, user ID, user name (optional), created time.
- **Reaction target**: a chat ID and a message ID, given or picked by `--last`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Reacting by IDs takes one Graph request; `--last` adds at most two (the chat and one page of messages), plus the existing `--to` lookup.
- **SC-002**: `teams messages` makes at most one extra Graph request per run, and none when every reaction already has a user name.
- **SC-003**: Every invalid reaction and every invalid argument combination exits 3 without a Graph write.
- **SC-004**: A live check in a real chat shows the reaction in Teams and in `teams messages`.

## Assumptions

- Chats only; channel messages are out of scope because m365 has no channel commands.
- Graph accepts the emoji as the `reactionType` string; Teams decides which emoji it supports.
- The signed-in user's chat membership email matches the session account, as in `teams send --exact-recipient`.
