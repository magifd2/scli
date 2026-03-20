# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-03-20

### Added

**Authentication**
- OAuth 2.0 PKCE flow with local HTTPS callback server (self-signed certificate)
- `--manual` flag for headless environments (prints auth URL, reads redirect URL from stdin)
- Token storage: OS keychain → environment variables → `.env` files → `config.json`
- Multiple workspace support (`--workspace` flag, `scli workspace use`)

**Channel commands**
- `scli channel list` — list channels you are a member of
- `scli channel read <channel>` — read messages with thread expansion
  - `--limit`, `--unread`, `--thread` flags
  - Thread replies automatically nested under parent messages

**Direct message commands**
- `scli dm list` — list open DM conversations (includes bots/apps)
- `scli dm read <user>` — read DM messages
- `scli dm send <user> <message>` — send a DM

**Post command**
- `scli post <channel> <message>` — post a message
  - `--thread` flag for thread replies
  - `--file` flag for file attachments (uses `files.getUploadURLExternal` API)
  - `\n` and `\t` escape sequences in message text

**Search command**
- `scli search <query>` — search messages across the workspace
  - `--limit` and `--asc` flags

**Unread summary**
- `scli unread` — show channels and DMs with unread messages
  - Falls back to `conversations.history` for channels where `unread_count` is inaccurate (bot/webhook-only channels)

**User command**
- `scli user list` — list workspace members

**Output**
- Color-formatted text output with auto TTY detection
- `--json` flag for machine-readable output
- `--no-color` flag

**Infrastructure**
- Cross-compilation for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- `make check`: lint (`golangci-lint`), test, build-all, security scan (`govulncheck`)
- Git pre-commit and pre-push hooks
- Design documents and setup guide in English and Japanese

[1.0.0]: https://github.com/magifd2/scli/releases/tag/v1.0.0
