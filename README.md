# Rocket.Chat TUI

Terminal User Interface for [Rocket.Chat](https://rocket.chat) built with [Bubbletea](https://github.com/charmbracelet/bubbletea).

> Fork from [RocketChat/rocketchat-tui](https://github.com/RocketChat/rocketchat-tui) with stability fixes, async operations, and UX improvements.

## Features

- Login with email/password or cached token
- Real-time messaging via WebSocket (DDP)
- Channel, group, and DM support
- Slash commands with fuzzy search (`/`)
- `@mention` autocomplete with member list
- Message pagination and history loading
- Unread message badges on sidebar
- Connection health check with auto-reconnect
- Connection status indicator

## Quick Start

Prerequisites: [Git](https://git-scm.com/) and [Go 1.17+](https://go.dev/)

```bash
git clone https://github.com/nerdseverino/rocketchat-tui.git
cd rocketchat-tui
go get
```

Create a `.env` file:

```
PROD_SERVER_URL=https://your-rocketchat-server.com
DEV_SERVER_URL=http://localhost:3000
```

Run:

```bash
# Development (localhost)
go run main.go

# Production server
go run main.go -prod

# Custom server
go run main.go -url=https://your-server.com

# With debug logging (writes to debug.log)
go run main.go -prod -debug
```

## Keybindings

| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit |
| `Ctrl+L` | Logout |
| `Ctrl+↑/↓` | Navigate channels |
| `Ctrl+←/→` | Navigate message pages |
| `Enter` | Select channel / Send message |
| `Esc` | Toggle typing mode |
| `/` | Slash commands |
| `@` | Mention members |

## Project Structure

```
├── main.go                  # Entrypoint, flags, env loading
├── cache/cache.go           # BoltDB token cache
├── keyBindings/keyBindings.go # Keyboard shortcuts
├── styles/styles.go         # Lipgloss styling
└── ui/
    ├── model.go             # Global state, Init/Update/View
    ├── view.go              # UI rendering (sidebar, chat, login)
    ├── listsView.go         # List delegates (messages, channels, commands)
    ├── chatControllers.go   # Slash commands, @mentions, key handling
    ├── channelControllers.go # Channel switching, subscriptions
    ├── messageControllers.go # Send/receive, history, health check
    ├── userControllers.go   # Login/logout, token management
    └── utils.go             # Helpers
```

## Changes from Upstream

- Fixed message listener dying silently on non-active room messages
- Fixed panic on empty message history and subscription list
- Fixed data race on `messageHistory` (concurrent goroutine access)
- Made `loadHistory` and `fetchPastMessages` async (non-blocking UI)
- Added connection health check with auto-reconnect (30s interval)
- Added graceful shutdown (BoltDB + WebSocket cleanup)
- Added unread message count badges on sidebar
- Added connection status indicator in bottom bar
- Replaced `panic()` calls with proper error handling
- Added bounds checks on channel selection

## License

See [upstream repository](https://github.com/RocketChat/rocketchat-tui) for license information.
