package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/slack-go/slack"
)

var (
	userMentionRe = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)
	slackLinkRe   = regexp.MustCompile(`<(https?://[^|>]+)(?:\|([^>]+))?>`)
)

// ChatViewModel displays messages in a scrollable viewport.
type ChatViewModel struct {
	viewport    viewport.Model
	messages    []slack.Message
	channelID   string
	channelName string
	userCache   *UserCache
	emojiCache  *EmojiCache
	cursor      int
	lineOffsets []int // starting line number of each message in rendered content
	width       int
	height      int
	ready       bool
}

func NewChatViewModel(userCache *UserCache, emojiCache *EmojiCache) ChatViewModel {
	return ChatViewModel{
		userCache:  userCache,
		emojiCache: emojiCache,
	}
}

func (m ChatViewModel) Init() tea.Cmd {
	return nil
}

func (m ChatViewModel) Update(msg tea.Msg) (ChatViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				content := m.renderMessages()
				m.viewport.SetContent(content)
				m.syncViewportToCursor()
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.messages)-1 {
				m.cursor++
				content := m.renderMessages()
				m.viewport.SetContent(content)
				m.syncViewportToCursor()
			}
			return m, nil
		case "enter":
			if sel := m.SelectedMessage(); sel != nil && sel.ReplyCount > 0 {
				threadTS := sel.Timestamp
				if sel.ThreadTimestamp != "" {
					threadTS = sel.ThreadTimestamp
				}
				return m, func() tea.Msg {
					return ThreadOpenMsg{ChannelID: m.channelID, ThreadTS: threadTS}
				}
			}
			return m, nil
		}

	case MessagesLoadedMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.channelID = msg.ChannelID
		// Messages come in reverse chronological order; reverse them
		m.messages = reverseMessages(msg.Messages)
		m.cursor = len(m.messages) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
		content := m.renderMessages()
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()

		// Resolve unknown users
		cmds = append(cmds, m.resolveUnknownUsers()...)
		return m, tea.Batch(cmds...)

	case NewMessagesMsg:
		if msg.Err != nil || msg.ChannelID != m.channelID || len(msg.Messages) == 0 {
			return m, nil
		}
		newMsgs := reverseMessages(msg.Messages)
		m.messages = append(m.messages, newMsgs...)
		content := m.renderMessages()
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()
		cmds = append(cmds, m.resolveUnknownUsers()...)
		return m, tea.Batch(cmds...)

	case UserResolvedMsg:
		if msg.Err == nil {
			m.userCache.Set(msg.UserID, msg.Name)
			content := m.renderMessages()
			m.viewport.SetContent(content)
		}
		return m, nil
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// SelectedMessage returns the message at the cursor position.
func (m *ChatViewModel) SelectedMessage() *slack.Message {
	if len(m.messages) == 0 || m.cursor < 0 || m.cursor >= len(m.messages) {
		return nil
	}
	return &m.messages[m.cursor]
}

// syncViewportToCursor adjusts viewport scroll so the cursor stays visible.
// Must be called after renderMessages() so that lineOffsets is up to date.
func (m *ChatViewModel) syncViewportToCursor() {
	if len(m.messages) == 0 || len(m.lineOffsets) == 0 {
		return
	}
	idx := m.cursor
	if idx >= len(m.lineOffsets) {
		idx = len(m.lineOffsets) - 1
	}

	cursorLine := m.lineOffsets[idx]

	// Determine how many lines this message occupies
	cursorEnd := cursorLine + 1
	if idx+1 < len(m.lineOffsets) {
		cursorEnd = m.lineOffsets[idx+1]
	}

	// If cursor message is above the viewport, scroll up to it
	if cursorLine < m.viewport.YOffset {
		m.viewport.SetYOffset(cursorLine)
	}
	// If cursor message extends below the viewport, scroll down
	if cursorEnd > m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(cursorEnd - m.viewport.Height)
	}
}

func (m ChatViewModel) View() string {
	if m.channelName == "" {
		return "Select a channel to view messages"
	}
	header := headerStyle.Render("#" + m.channelName)
	return header + "\n" + m.viewport.View()
}

func (m *ChatViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	headerHeight := 1
	vpHeight := height - headerHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	if !m.ready {
		m.viewport = viewport.New(width, vpHeight)
		m.viewport.SetContent("Select a channel to view messages")
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = vpHeight
	}
}

func (m *ChatViewModel) SetChannel(name string) {
	m.channelName = name
}

func (m ChatViewModel) LatestTimestamp() string {
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[len(m.messages)-1].Timestamp
}

func (m *ChatViewModel) renderMessages() string {
	if len(m.messages) == 0 {
		m.lineOffsets = nil
		return "No messages"
	}

	wrapWidth := m.width - 1 // reserve 1 column for cursor marker/spacer
	wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
	m.lineOffsets = make([]int, len(m.messages))
	currentLine := 0

	var sb strings.Builder
	for i, msg := range m.messages {
		m.lineOffsets[i] = currentLine

		username := msg.User
		if name, ok := m.userCache.Get(msg.User); ok {
			username = name
		}

		text := m.replaceLinks(msg.Text)
		text = m.replaceMentions(text)
		text = m.emojiCache.Replace(text)

		ts := formatTimestamp(msg.Timestamp)
		tsStyle := timestampStyle
		unStyle := usernameStyle
		if i == m.cursor {
			tsStyle = cursorTimestampStyle
			unStyle = cursorUsernameStyle
		}

		threadIcon := " "
		if msg.ReplyCount > 0 {
			threadIcon = threadIndicatorStyle.Render("💬") + " "
		}

		line := fmt.Sprintf("%s %s%s: %s",
			tsStyle.Render("["+ts+"]"),
			threadIcon,
			unStyle.Render(username),
			text,
		)

		rendered := wrapStyle.Render(line)

		// Prepend cursor marker or spacer for alignment
		if i == m.cursor {
			rendered = cursorMarker.Render("▎") + rendered
		} else {
			rendered = " " + rendered
		}

		sb.WriteString(rendered + "\n")
		currentLine += strings.Count(rendered, "\n") + 1
	}
	return sb.String()
}

func (m ChatViewModel) replaceMentions(text string) string {
	return userMentionRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := userMentionRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		if name, ok := m.userCache.Get(sub[1]); ok {
			return mentionStyle.Render("@" + name)
		}
		return match
	})
}

func (m ChatViewModel) replaceLinks(text string) string {
	return slackLinkRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := slackLinkRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		url := sub[1]
		label := url
		if len(sub) >= 3 && sub[2] != "" {
			label = sub[2]
		}
		return osc8Link(url, linkStyle.Render(label))
	})
}

// osc8Link wraps text in an OSC 8 hyperlink escape sequence.
// Terminals that support it render a clickable link; others show the text as-is.
func osc8Link(url, text string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
}

func (m ChatViewModel) resolveUnknownUsers() []tea.Cmd {
	seen := make(map[string]bool)
	var cmds []tea.Cmd
	for _, msg := range m.messages {
		// Resolve message author
		if msg.User != "" && !seen[msg.User] {
			if _, ok := m.userCache.Get(msg.User); !ok {
				seen[msg.User] = true
				cmds = append(cmds, resolveUser(msg.User))
			}
		}
		// Resolve users mentioned in message text
		for _, match := range userMentionRe.FindAllStringSubmatch(msg.Text, -1) {
			uid := match[1]
			if !seen[uid] {
				if _, ok := m.userCache.Get(uid); !ok {
					seen[uid] = true
					cmds = append(cmds, resolveUser(uid))
				}
			}
		}
	}
	return cmds
}

func reverseMessages(msgs []slack.Message) []slack.Message {
	n := len(msgs)
	reversed := make([]slack.Message, n)
	for i, m := range msgs {
		reversed[n-1-i] = m
	}
	return reversed
}

func formatTimestamp(ts string) string {
	// Slack timestamps are "epoch.seq" e.g. "1234567890.123456"
	// Just show HH:MM from the epoch part
	parts := strings.SplitN(ts, ".", 2)
	if len(parts) == 0 {
		return ts
	}

	var epoch int64
	fmt.Sscanf(parts[0], "%d", &epoch)
	if epoch == 0 {
		return ts
	}

	t := fmt.Sprintf("%02d:%02d",
		(epoch%86400)/3600,
		(epoch%3600)/60,
	)
	return t
}
