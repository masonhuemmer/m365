# Contract: `m365 teams react` and reactions in `teams messages`

## Forms

```
m365 teams react CHAT_ID MESSAGE_ID REACTION [--remove] [--dry-run]
m365 teams react CHAT_ID --last REACTION [--remove] [--dry-run]
m365 teams react --to NAME --last REACTION [--exact-recipient] [--remove] [--dry-run]
```

REACTION: `like` 👍, `heart` ❤️, `laugh` 😆, `surprised` 😮, `sad` 😢, `angry` 😡, `eyes` 👀 (any case), or an emoji.

`--last`: the newest message, by created time, that is not a system event and not yours, among the first 50 messages Graph returns.

## Output (JSON; `--human` prints the same fields)

| Case | Fields |
| --- | --- |
| Added | `chat_id`, `message_id`, `reaction`, `reacted: true` |
| Removed | `chat_id`, `message_id`, `reaction`, `removed: true` |
| Dry run | `dry_run: true`, `action` (`react` or `remove`), `chat_id`, `message_id`, `reaction` |
| With `--last` | also `from`, `created`, `text` of the picked message |

## Errors

| Exit | When |
| --- | --- |
| 3 usage | reaction is not a name or an emoji; wrong argument form (`--to` without `--last`, `--last` with a message ID, `--to` with a chat ID, missing IDs); `--last` cannot tell which members are you |
| 4 auth | not signed in, or signed in without Teams |
| 5 Graph | Graph refuses the reaction or fails |
| 6 not found | chat or message does not exist; `--last` finds no message from someone else; `--to` finds no chat |

## `teams messages`

Each item may carry `reactions: [{emoji, user_id, user?, created}]`. Graph's legacy names map to the emoji above; `custom` stays `custom`. `user` comes from Graph, else from the chat's members; it is left out when neither has it. The field is absent when a message has no reactions.

## MCP

`teams react` is a write: without `write_opt_in` it runs as a dry run; `--allow` must name `teams.react`; with `--exact-recipients`, `--to` must be an exact email or user ID.
