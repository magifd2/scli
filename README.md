# scli

A terminal-based Slack client that operates as a user (not a bot).
Read channels, send messages, search, and manage DMs — all without leaving your terminal.

## Features

- Read channel and DM messages with thread expansion
- Post messages and reply to threads (supports `\n` for newlines)
- Upload files to channels
- List unread channels and DMs
- Search messages across the workspace
- Multiple workspace support
- Color output with `--json` and `--no-color` flags
- Token storage via OS keychain, environment variables, or `.env` files

## Installation

### Build from source

```sh
git clone https://github.com/magifd2/scli.git
cd scli
make build          # builds for the current platform → dist/scli
make build-all      # cross-compiles for all target platforms
```

Requirements: Go 1.26+, `make`

### First-time setup

See [docs/setup.md](docs/setup.md) for step-by-step instructions on creating a Slack app and authenticating.

```sh
scli auth login
```

## Commands

| Command | Description |
|---------|-------------|
| `scli auth login` | Authenticate with Slack (OAuth 2.0 PKCE) |
| `scli auth logout` | Remove stored credentials |
| `scli auth list` | Show authenticated workspaces |
| `scli channel list` | List channels you are a member of |
| `scli channel read <channel>` | Read messages from a channel |
| `scli dm list` | List open DM conversations |
| `scli dm read <user>` | Read DM messages |
| `scli dm send <user> <message>` | Send a direct message |
| `scli post <channel> <message>` | Post a message to a channel |
| `scli search <query>` | Search messages in the workspace |
| `scli unread` | Show channels and DMs with unread messages |
| `scli user list` | List workspace members |
| `scli workspace list` | List configured workspaces |
| `scli workspace use <name>` | Switch default workspace |

### Common flags

```
--workspace, -w   Workspace name (default: "default")
--json            Output raw JSON
--no-color        Disable ANSI color codes
```

### channel read / dm read options

```
-n, --limit N     Number of messages to fetch (default: 20)
--unread          Show only messages since last read
--thread <ts>     Show a specific thread by message timestamp
```

### post options

```
--file <path>     Attach a file to the message
--thread <ts>     Reply in a thread
```

### search options

```
-n, --limit N     Maximum number of results (default: 20)
--asc             Sort results oldest first (default: newest first)
```

## Multiple workspaces

```sh
scli auth login --workspace personal
scli auth login --workspace work
scli workspace use work
scli channel read #general --workspace personal
```

## License

MIT — see [LICENSE](LICENSE)
