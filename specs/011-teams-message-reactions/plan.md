# Implementation Plan: React to Teams chat messages

**Branch**: `011-teams-message-reactions` | **Date**: 2026-11-12 | **Spec**: [spec.md](spec.md) | **Contract**: [contracts/teams-react.md](contracts/teams-react.md)

## Summary

Add `m365 teams react` (add or remove an emoji reaction on one chat message, picked by IDs or by `--last`) and a `reactions` field in `teams messages`. Graph: `POST /chats/{id}/messages/{id}/setReaction|unsetReaction` under the existing `ChatMessage.Send` scope.

## Technical Context

- **Language**: Go (toolchain from `go.mod`). No new dependency.
- **Graph**: v1.0 `setReaction`/`unsetReaction` (204), `chatMessage.reactions[]` with `reactionType`, `displayName`, `createdDateTime`, `user.user.{id,displayName}`. Messages are listed by last change.
- **Scopes**: unchanged. `ChatMessage.Send` is already in every login (`internal/adapters/graph/oauth.go`).

## Design

1. **Domain (`internal/domain/reaction.go`, new)**. `Reaction{Emoji, UserID, User, Created}` (JSON `emoji`, `user_id`, `user`, `created`); `ChatMessage.Reactions []Reaction` (`reactions`, omitted when empty). `ParseReaction(s)` returns the emoji for a name (like, heart, laugh, surprised, sad, angry, eyes; case-insensitive) or for a direct emoji (at most 10 code points; no letter, space or control character; at least one symbol or enclosing-mark code point), else a usage error that lists the names. `ReactionEmoji(graphType)` maps Graph's legacy names to the same emoji and returns any other value unchanged.
2. **Application (`internal/app/teams/react.go`, new)**. Port `Reactor{React(ctx, chatID, messageID, emoji string, remove bool) error}`, separate from `Store` so the locked test stubs that implement `Store` are untouched. `React(ctx, st Store, r Reactor, m ChatMap, sess, ReactInput)`: requires a Teams session; validates the argument form; parses the reaction; resolves `--to` with the existing recipient lookup (`--exact-recipient` as in send); for `--last`, `GetChat` gives your member ID (member email = session account) and one `Messages` page (top 50, no system events) gives the newest message, by created time, from anyone else. Dry run returns the plan; otherwise calls the Reactor. `Messages` then fills missing reaction user names from one `GetChat` when any reaction lacks a name (`react_names.go`).
3. **Graph adapter**. `HTTPTeams.React` posts `{"reactionType": emoji}` to `/chats/{chat}/messages/{msg}/setReaction` or `unsetReaction` (`httpteams_react.go`, new). `graphChatMsg` gains `reactions`; `toChatMessage` maps each to `domain.Reaction` through `ReactionEmoji`. Errors keep the existing classification (404 → not found, 4xx/5xx → Graph error).
4. **Fake (`internal/adapters/graph/memory_react.go`, new)**. `Memory.React`/`TeamsAPI.React` record each set or unset in `Sent` as a `ReactionChange`; the chat and message must exist (else not found). `Seed` gets messages in `chat-ajay`: an older one from Ajay, a newer one from you with Ajay's 👍 (no name, to exercise name filling).
5. **CLI (`internal/adapters/cli/teams_react.go`, new)**. `teams react` flags `--to`, `--last`, `--remove`, `--dry-run`, `--exact-recipient`; positionals per the contract. The Reactor is the Teams dependency when it implements `teams.Reactor`, else a Graph error (wiring fault). Help in `help.go`; `teams --help` lists it; `teams messages --help` mentions `reactions`.
6. **MCP**. `teams react` is a write in `isWrite` (dry run without `write_opt_in`, `--allow teams.react`). With `--exact-recipients`, `--to` must be an exact email or user ID and sets `--exact-recipient`. One recipe line in `mcp_write_recipes.go`; the same line in `skills/teams-write/SKILL.md`; README section.
7. **Acceptance (`features/teams/react.feature`)**. A Scenario Outline over message and emoji given directly (names are not example values: the mutator's case changes would be equivalent mutations, since names ignore case), and scenarios for a name, `--to Ajay --last`, `--remove`, dry run (nothing sent), an invalid reaction (exit 3, `the error is`), bad argument forms, and `teams messages` reactions with a filled name.

## Test commands

- Unit and contract: `make unit` (or `go test ./internal/...`), race: `make race`.
- Acceptance: `make acceptance`; example mutation: `make acceptance-mutation`.
- Everything: `make verify`.

## Performance (Principle VI)

- Metric: Graph requests per command, counted by the fake Graph server in contract tests.
- Thresholds: react by IDs = 1; `--last` = 3 (chat, one message page, set) plus the existing `--to` lookup; `teams messages` = 1, or 2 when a reaction lacks a user name.

## Constitution Check

- [x] **I. TDD. PASS.** Each slice: SCAFFOLD → locked RED → smallest GREEN. No locked test edited: the new port avoids changing `Store`, whose stubs live in locked tests.
- [x] **II. Clean Code. PASS.** New files each under 150 lines. `memory.go` (483) gains only the seed call and stays under 500; `help.go`, `teams.go` stay under 250.
- [x] **III. Smallest Sufficient Design. PASS.** Chats only; no channel support; one port method; the name list is a map.
- [x] **IV. Testing Standards. PASS.** Domain and app tests with fakes; Graph contract tests on an httptest server with synthetic bodies; generated acceptance tests from the feature.
- [x] **V. CLI Experience Consistency. PASS.** Same flags as send (`--to`, `--dry-run`, `--exact-recipient`), JSON and `--human`, exit codes 0/3/4/5/6, `--help`, the 50-message limit stated.
- [x] **VI. Performance. PASS.** Metric and thresholds above; one bounded page for `--last`.
- [x] **VII. Domain Isolation. PASS.** Graph URLs and JSON stay in the adapter; the port speaks "react to a message".
- [x] **VIII. Secrets, Auth, Least Privilege. PASS.** No new scope; fake tokens and synthetic bodies only.

## Complexity Tracking

None.

## Out of This Plan

- Channel messages and replies; reacting to several messages at once; custom (image) reactions.
