# m365

A command-line tool for **your** Microsoft 365 account: Outlook, Teams, calendar, and OneDrive.

JSON on stdout by default. Add `--human` if you want plain text. Writes stay dry-run until you drop `--dry-run` (or, for agents, set `write_opt_in`).

Current release: **0.5.0**.

## Install

### macOS (Homebrew)

```sh
brew tap masonhuemmer/tap
brew install m365
m365 --version    # 0.5.0
```

Upgrade later with `brew update && brew upgrade m365`.

To build the latest `main` instead of a numbered release:

```sh
brew install --HEAD masonhuemmer/tap/m365
```

### Windows (Scoop)

```powershell
scoop bucket add masonhuemmer https://github.com/masonhuemmer/scoop-bucket
scoop install m365
m365 --version
```

### Windows (WinGet)

The first community package is in review: [microsoft/winget-pkgs#439501](https://github.com/microsoft/winget-pkgs/pull/439501). After that merges:

```powershell
winget install JacobHuemmer.m365
```

Until then, use Scoop or a zip from [Releases](https://github.com/masonhuemmer/m365/releases).

### From source

Go 1.25+:

```sh
git clone https://github.com/masonhuemmer/m365.git
cd m365
make install
```

## Sign in

You need an Entra app registration that you are allowed to use (delegated, as yourself). Create the configuration directory:

```sh
mkdir -p ~/.config/m365
```

Create or update `~/.config/m365/config.json` with your app's IDs, preserving any existing settings:

```json
{
  "client_id": "YOUR_CLIENT_ID",
  "tenant_id": "YOUR_TENANT_ID"
}
```

If `XDG_CONFIG_HOME` is set, use `$XDG_CONFIG_HOME/m365/config.json` instead.

Alternatively, export `M365_CLIENT_ID` and `M365_TENANT_ID` in your shell; these override the values in `config.json`. The CLI does not automatically load `.env` files.

### Windows paths

`~` means `%USERPROFILE%` (usually `C:\Users\<name>`). Create the config folder in PowerShell:

```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\.config\m365"
```

Then put `config.json` in `%USERPROFILE%\.config\m365\` (or `%XDG_CONFIG_HOME%\m365\` if you set it). Setting `M365_CLIENT_ID` and `M365_TENANT_ID` as environment variables works too and needs no file.

| What | Where on Windows |
|---|---|
| Config | `%USERPROFILE%\.config\m365\config.json` |
| Tokens | Windows Credential Manager |
| Session file (used when the token is too big for Credential Manager, which limits a secret to 2,560 bytes) | `%LOCALAPPDATA%\m365\session.json`, readable only by you |
| Watch, mail-watch and chat-map state | `%USERPROFILE%\.local\state\m365\` (or `%XDG_STATE_HOME%\m365\`) |

Then, in a real terminal (browser login):

```sh
m365 auth login
m365 auth status
```

`signed_in` and `session_usable` should be true, with the namespaces you granted (`mail`, `teams`, `calendar`, `files`). Tokens go in the macOS keychain, with a file fallback if needed.

Do not run `auth login` through MCP. If status is not usable, stop and log in from a terminal.

## Everyday use

```
m365 <area> <command> [flags]
```

`--help` on any command does not need a session. `chat` is an alias for `teams`.

| Area | What it does |
| --- | --- |
| `auth` | `login`, `status`, `logout` |
| `mail` | list, read, send, reply |
| `teams` | find a chat, read, send |
| `calendar` | list, free slots, create |
| `files` | OneDrive list, download, upload |
| `mcp` | `serve` for agents |

Exit codes: `0` ok, `3` usage/config, `4` sign-in, `5` Microsoft Graph, `6` not found.

Preview any send with `--dry-run` first.

### Mail

```sh
m365 mail list --unread --top 10 --human
m365 mail get MESSAGE_ID
m365 mail send --to you@example.com --subject 'Status' --body 'In UAT.' --dry-run
m365 mail send --note-to-self --body 'Remember this.' --dry-run
```

HTML mail (paragraphs, lists, links): `--html` and a real HTML `--body`, or `--format md`. Replies with `--html` go out as HTML, not a jammed comment.

Add `--preview` to see the message as a text box before sending (never sends). A send whose body has format problems (broken HTML, leftover markdown, literal `\n`) fails with exit 3; `--dry-run` lists them.

### Teams

```sh
m365 teams find Ajay
m365 teams send --to Ajay --text 'Looking into this.' --dry-run
m365 teams send --note-to-self --text 'Scratch note.' --dry-run
```

`--note-to-self` is **Chat with yourself** (`48:notes`), not a hidden 1:1. Do not combine it with `--to` or a chat id.

For a list or a link, use `--format md` (a small markdown subset becomes HTML) or `--html` with HTML already written.

After a unique `teams find` or a listed 1:1, later `--to Ajay` remembers the chat id so it does not scan the whole list again.

### Calendar and files

```sh
m365 calendar list --human
m365 calendar free --when tomorrow
m365 calendar create --when 'tomorrow at 1:30 pm' --subject 'Sync' --dry-run
m365 files list --human
```

## Agents (MCP)

```sh
m365 mcp serve
```

Stdio JSON-RPC. Three tools: `m365_status`, `m365_help`, `m365_run`. Six recipe prompts: `mail-search`, `teams-find`, `calendar`, `files`, `mail-write`, `teams-write`. Do not pass `--human`. Writes through `m365_run` stay dry-run unless `write_opt_in` is true.

### Running under autonomous agents

```sh
m365 mcp serve --read-only
m365 mcp serve --allow teams.send,mail.send --exact-recipients
```

`--read-only` rejects every `m365_run` call with `write_opt_in: true`, including calls to read verbs. `--allow` restricts real writes to the listed `namespace.verb` pairs; unlisted writes still preview without opt-in. Without `--allow`, the existing opt-in behavior is unchanged. Writes remain dry-run by default in every mode. `--exact-recipients` rejects name-based `--to` values for sends. Use an exact email address or Entra user ID for Teams `--to`, or a chat ID as the Teams positional argument (or `--to`). Mail `--to` and `--cc` require email addresses. Teams exact matching scans 1:1 chat members and fails if no exact match or more than one chat matches.

`--allow=` denies all real workload writes. These guards apply to workload writes; commands that save or download files can still write local paths. `--exact-recipients` validates mail send `--to` and `--cc` and Teams send `--to`; it rejects mail reply `--all` because those recipients come from the original thread. Plain mail reply remains available. Calendar attendees are not checked by `--exact-recipients`. To message yourself with exact matching enabled, use `--note-to-self` instead of your own email address or user ID. Teams exact matching (email or user ID) requires the signed-in account to be identifiable among the chat members; otherwise it fails closed, so pass a chat ID if your account email differs from the address Graph reports for you.

To check the flags with an authenticated test account, call `m365_run` with `namespace=mail`, `verb=send`, and `flags={"to":"you@example.com","subject":"Test","body":"Test"}`. With `--read-only`, adding `write_opt_in=true` returns a usage error. With `--allow teams.send`, the same opt-in mail send returns a usage error; without opt-in it returns a dry-run preview. With `--allow mail.send`, opt-in permits the send. With `--exact-recipients`, a Teams send to a display name returns a usage error, while an exact email or chat ID can resolve and send when opted in. The automated acceptance scenarios use fake Graph data and do not send external messages.

This repo has no `m365 skill` command. Copy `skills/<topic>/SKILL.md` into an agent skill root if you want files on disk. `skills/writing-style` is a voice guide for drafts that go out under your name; edit its examples to match how you write.

```text
Cursor    .cursor/skills/<topic>/SKILL.md
Claude    .claude/skills/<topic>/SKILL.md
Codex     .codex/skills/<topic>/SKILL.md
Grok      .grok/skills/<topic>/SKILL.md
OpenCode  .opencode/skills/<topic>/SKILL.md
```

Full CLI contract for agents: [docs/m365.md](docs/m365.md).

## Experimental: mail watch

`mail watch` is one folder-scoped poll, then exit. Classification is off unless you enable it in `~/.config/m365/config.json`. Keep `TYPESAFE_API_KEY` (or the alias `TYPESAFE_AI_TOKEN`) in the environment, never in that file. An `actionable` event is not permission to send; still `--dry-run`, then an explicit send.

## Develop

```sh
sh scripts/install-tools.sh
make verify
```

Both use the Go version on `go.mod`'s `toolchain` line (Go downloads it if needed), so local results match CI. To try a newer installed Go, run with `GOTOOLCHAIN=local`.

## License

MIT
