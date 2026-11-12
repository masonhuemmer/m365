# Tasks: React to Teams chat messages

**Input**: [spec.md](spec.md), [plan.md](plan.md), [contracts/teams-react.md](contracts/teams-react.md), `.specify/memory/constitution.md`.

**Tests**: MANDATORY (Constitution I). Each slice: SCAFFOLD (zero-value signatures only) → failing tests run and seen to fail on assertions → RED commit (tests and fixtures only) → smallest GREEN → GREEN commit. Locked tests are never edited without Mason's approval.

**Do not**: change `teams.Store` (its stubs are in locked tests); react in a real chat without Mason; push, open a PR or merge without Mason's explicit yes.

## Phase 1: Domain (US1, US4)

- [X] T001 SCAFFOLD `internal/domain/reaction.go`: `Reaction`, `ChatMessage.Reactions`, `ParseReaction`, `ReactionEmoji` returning zero values.
- [X] T002 RED `internal/domain/reaction_test.go`: each name (any case) → its emoji; direct emoji kept (👍, 🎉, 👍🏽, 👩🏽‍💻, 1️⃣); refused with a usage error naming the accepted names: "", "hello", "+1", "1", "!", "a👍", "👍 👍", a control character, 11 code points; `ReactionEmoji`: legacy names → emoji, unicode unchanged, `custom` unchanged; `reactions` omitted from JSON when empty.
- [X] T003 GREEN `reaction.go`.

## Phase 2: Application (US1–US4)

- [X] T004 SCAFFOLD `internal/app/teams/react.go` (`Reactor`, `ReactInput`, `React`) and `react_names.go` (name filling), zero-valued.
- [X] T005 RED `internal/app/teams/react_test.go` and `react_names_test.go`: by IDs → Reactor gets chat, message, emoji, remove flag; output fields per contract; dry run → no Reactor call, `action`; `--last` → newest by created time, skipping yours and system events, with `from`/`created`/`text`; none → not found; you not among members → usage; argument forms → usage, no call; invalid reaction → usage, no call; `--to` resolves through the chat map/find; signed out or no Teams → auth; Reactor error returned unchanged; `Messages` fills a missing user name from members with one `GetChat`, makes none when all are named, leaves `user` empty without a match.
- [X] T006 GREEN `react.go`, `react_names.go`, and the `Messages` call.

## Phase 3: Graph adapter (US1, US3, US4)

- [X] T007 SCAFFOLD `internal/adapters/graph/httpteams_react.go`: `HTTPTeams.React` and the reaction mapping, zero-valued.
- [X] T008 RED `internal/adapters/graph/httpteams_react_test.go` (httptest, synthetic bodies): set → one `POST /chats/{c}/messages/{m}/setReaction` with `{"reactionType":"👍"}`; remove → `unsetReaction`; IDs are path-escaped; 404 → not found; 400 → Graph error; `Messages` maps reactions (legacy `like` → 👍, unicode kept, `custom` kept, user id and display name, created).
- [X] T009 GREEN.

## Phase 4: CLI, MCP and acceptance (US1–US5)

- [X] T010 SCAFFOLD `internal/adapters/cli/teams_react.go` and `internal/adapters/graph/memory_react.go`, zero-valued.
- [X] T011 RED `features/teams/react.feature` (with the `chat-ajay` seed messages as its fixture), `internal/adapters/cli/teams_react_test.go` (forms, `--help`, JSON and `--human`, exit codes), and MCP tests (dry run without opt-in; `--allow` without `teams.react` refused; `--exact-recipients` refuses a display name for `--to` and accepts an email).
- [X] T012 GREEN: CLI verb and help, `teams --help` and `teams messages --help` text, fake React, `isWrite`, exact-recipient policy for react.

## Phase 5: Documentation and gates

- [X] T013 MCP write recipe, `skills/teams-write/SKILL.md`, README.
- [X] T014 `make verify` and `make acceptance-mutation` pass.
- [X] T015 Live check with Mason in a real chat (react, see it in Teams and in `teams messages`, remove).
- [ ] T016 Ask Mason before push, PR or merge.

## Log

- T001–T003 (2026-11-12): SCAFFOLD `2fdbeaf`, RED `c137d29` (4 tests failed on assertions; one JSON-shape test already passed on the scaffold field), GREEN `17d8167`.
- T004–T006: SCAFFOLD `df0bb97`, RED `0d86d6e` (12 of 13 tests failed on assertions; the no-lookup test passed on the scaffold), GREEN `357fb7f`.
- T007–T009: SCAFFOLD `ef2c796`, RED `016d33e` (4 tests failed on assertions), GREEN `c38ed5a`.
- T010–T012: SCAFFOLD `8ecc3fc`, RED `10fc24d` (CLI and MCP tests and every new acceptance scenario failed with `unknown teams verb "react"`; the `teams messages` reactions scenario already passed on the phase 2 GREEN). The locked `TestTeamsReactHumanOutput` wrongly asserted that `--human` output is not JSON; m365 prints indented JSON for map results. Mason approved dropping that one check (`2f47b20`). GREEN `a49a5b6`. `--last` and `--remove` joined `boolFlags`, or `parseMixed` would read the reaction as their value.
- T013 `48bfd67`: README, teams-write MCP recipe and skill. Spec edge case corrected (Graph errors exit 5 with the generic message); plan step 4 (the fake records `ReactionChange` in `Sent`) and step 7 (names are not example values: the mutator's case changes would be equivalent mutations) updated to match.
- T014: first `make verify` failed only at `crap`: `MCPPolicy.check` 16.1 > 15 after the react case. REFACTOR `4b9e98b` moved the exact-recipient switch into `checkExactRecipients`. Then, from a clean `build/`: `make verify` exit 0 (filesize ok 383 files, gosec 0 issues, govulncheck 0 affecting, crap ok 563 functions / 42 baseline); `make acceptance-mutation` exit 0 (unknown-commands 12/12 killed, teams react 4/4 killed), `features/` unchanged.
- T015 (2026-11-12, Mason chose his Notes chat so nobody else sees it): branch build, real Graph. `teams react 48:notes <newest note id> eyes --dry-run` → dry run; real run → `reacted: true`; `teams messages 48:notes` showed 👀 with `user_id` and `created` (no `user`: Graph sent no name and Notes has no member list, as specified); `--remove` → `removed: true`, and the reaction was gone. `--last` in Notes exited 5 because Graph cannot read `48:notes` as a chat, not the specified usage error: RED `a735a47`, fix `84d4ca1` (refused before any Graph call); live recheck exit 3 with the hint. Read-only check of 8 recent chats (counts only, no text or names): 39 reactions, every one with `user_id` and `user`. `--last` was not run in a real chat with other people.
- After the fix, from a clean `build/`: `make verify` exit 0; `make acceptance-mutation` 12/12 and 4/4 killed.

