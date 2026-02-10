# TUI Guide

The interactive TUI provides a full Slack browsing experience from your terminal. Launch it with:

```bash
slack-cli tui
```

## Layout

The interface is split into panels:

```
┌──────────┬─────────────────────┬──────────────┐
│          │                     │              │
│ Channels │     Messages        │   Thread     │
│          │                     │  (optional)  │
│          │                     │              │
│          ├─────────────────────┤              │
│          │ Input               │              │
└──────────┴─────────────────────┴──────────────┘
 Status bar
 Help bar
```

- **Channels** (left) — scrollable, filterable list of your workspace channels
- **Messages** (center) — chat messages for the selected channel
- **Input** (center bottom) — text input for sending messages
- **Thread** (right, when open) — thread replies shown in a side panel

## Navigation

Press **Tab** / **Shift+Tab** to cycle focus between panels. The focused panel has a highlighted border.

### Channel List

| Key | Action |
|-----|--------|
| `j` / `Down` | Move selection down |
| `k` / `Up` | Move selection up |
| `Enter` | Open selected channel |
| `/` | Start filtering channels by name |
| `Esc` | Cancel filter |

### Messages

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor to next message |
| `k` / `Up` | Move cursor to previous message |
| `Enter` | Open thread (on messages with replies) |

The selected message is indicated by a `▎` marker and highlighted timestamp/username. Messages with thread replies display a 💬 icon between the timestamp and author name.

### Input

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| Standard text editing keys | Edit message text |

### Thread Panel

When you press Enter on a message with replies, the thread panel opens on the right side of the screen.

| Key | Action |
|-----|--------|
| `j` / `Down` | Scroll down through replies |
| `k` / `Up` | Scroll up through replies |
| `Esc` | Close thread panel |

When focused on the thread input:

| Key | Action |
|-----|--------|
| `Enter` | Send reply to thread |

## Global Shortcuts

These shortcuts work regardless of which panel is focused:

| Key | Action |
|-----|--------|
| `Ctrl+K` | Jump to channel list and start filtering |
| `Ctrl+L` | Focus channel list |
| `Ctrl+N` | Focus message input |
| `Ctrl+F` | Toggle favorites overlay |
| `Ctrl+A` | Add/remove current channel from favorites |
| `Alt+1`–`Alt+9` | Jump to favorite channel by slot number |
| `Tab` | Focus next panel |
| `Shift+Tab` | Focus previous panel |
| `Ctrl+C` | Quit |

## Favorites

You can save up to 9 channels as favorites for quick access:

1. Open a channel
2. Press **Ctrl+A** to add it to your favorites
3. Press **Alt+1** through **Alt+9** to jump to a favorite by slot
4. Press **Ctrl+F** to open the favorites overlay and manage them

Favorited channels show a slot number badge in the channel list.

## Features

- **Live updates** — new messages and thread replies are polled automatically
- **User resolution** — user IDs are resolved to display names
- **Emoji support** — standard Unicode emoji and custom workspace emoji are rendered inline
- **Clickable links** — URLs are rendered as clickable terminal hyperlinks (in supported terminals)
- **Mentions** — `@user` mentions are highlighted
- **Thread indicators** — messages with replies show a 💬 icon so you can spot active threads at a glance
