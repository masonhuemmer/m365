# Tasks: React to Teams chat messages

**Input**: [spec.md](spec.md), [plan.md](plan.md), [contracts/teams-react.md](contracts/teams-react.md), `.specify/memory/constitution.md`.

**Tests**: MANDATORY (Constitution I). Each slice: SCAFFOLD (zero-value signatures only) → failing tests run and seen to fail on assertions → RED commit (tests and fixtures only) → smallest GREEN → GREEN commit. Locked tests are never edited without Mason's approval.

**Do not**: change `teams.Store` (its stubs are in locked tests); react in a real chat without Mason; push, open a PR or merge without Mason's explicit yes.

## Phase 1: Domain (US1, US4)

- [ ] T001 SCAFFOLD `internal/domain/reaction.go`: `Reaction`, `ChatMessage.Reactions`, `ParseReaction`, `ReactionEmoji` returning zero values.
- [ ] T002 RED `internal/domain/reaction_test.go`: each name (any case) → its emoji; direct emoji kept (👍, 🎉, 👍🏽, 👩🏽‍💻, 1️⃣); refused with a usage error naming the accepted names: "", "hello", "+1", "1", "!", "a👍", "👍 👍", a control character, 11 code points; `ReactionEmoji`: legacy names → emoji, unicode unchanged, `custom` unchanged; `reactions` omitted from JSON when empty.
- [ ] T003 GREEN `reaction.go`.

## Phase 2: Application (US1–US4)

- [ ] T004 SCAFFOLD `internal/app/teams/react.go` (`Reactor`, `ReactInput`, `React`) and `react_names.go` (name filling), zero-valued.
- [ ] T005 RED `internal/app/teams/react_test.go` and `react_names_test.go`: by IDs → Reactor gets chat, message, emoji, remove flag; output fields per contract; dry run → no Reactor call, `action`; `--last` → newest by created time, skipping yours and system events, with `from`/`created`/`text`; none → not found; you not among members → usage; argument forms → usage, no call; invalid reaction → usage, no call; `--to` resolves through the chat map/find; signed out or no Teams → auth; Reactor error returned unchanged; `Messages` fills a missing user name from members with one `GetChat`, makes none when all are named, leaves `user` empty without a match.
- [ ] T006 GREEN `react.go`, `react_names.go`, and the `Messages` call.

## Phase 3: Graph adapter (US1, US3, US4)

- [ ] T007 SCAFFOLD `internal/adapters/graph/httpteams_react.go`: `HTTPTeams.React` and the reaction mapping, zero-valued.
- [ ] T008 RED `internal/adapters/graph/httpteams_react_test.go` (httptest, synthetic bodies): set → one `POST /chats/{c}/messages/{m}/setReaction` with `{"reactionType":"👍"}`; remove → `unsetReaction`; IDs are path-escaped; 404 → not found; 400 → Graph error; `Messages` maps reactions (legacy `like` → 👍, unicode kept, `custom` kept, user id and display name, created).
- [ ] T009 GREEN.

## Phase 4: CLI, MCP and acceptance (US1–US5)

- [ ] T010 SCAFFOLD `internal/adapters/cli/teams_react.go` and `internal/adapters/graph/memory_react.go`, zero-valued.
- [ ] T011 RED `features/teams/react.feature` (with the `chat-ajay` seed messages as its fixture), `internal/adapters/cli/teams_react_test.go` (forms, `--help`, JSON and `--human`, exit codes), and MCP tests (dry run without opt-in; `--allow` without `teams.react` refused; `--exact-recipients` refuses a display name for `--to` and accepts an email).
- [ ] T012 GREEN: CLI verb and help, `teams --help` and `teams messages --help` text, fake React, `isWrite`, exact-recipient policy for react.

## Phase 5: Documentation and gates

- [ ] T013 MCP write recipe, `skills/teams-write/SKILL.md`, README.
- [ ] T014 `make verify` and `make acceptance-mutation` pass.
- [ ] T015 Live check with Mason in a real chat (react, see it in Teams and in `teams messages`, remove).
- [ ] T016 Ask Mason before push, PR or merge.

## Log
