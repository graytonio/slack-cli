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
	width       int
	height      int
	ready       bool
}

func NewChatViewModel(userCache *UserCache) ChatViewModel {
	return ChatViewModel{
		userCache: userCache,
	}
}

func (m ChatViewModel) Init() tea.Cmd {
	return nil
}

func (m ChatViewModel) Update(msg tea.Msg) (ChatViewModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case MessagesLoadedMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.channelID = msg.ChannelID
		// Messages come in reverse chronological order; reverse them
		m.messages = reverseMessages(msg.Messages)
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

func (m ChatViewModel) renderMessages() string {
	if len(m.messages) == 0 {
		return "No messages"
	}

	wrapStyle := lipgloss.NewStyle().Width(m.width)

	var sb strings.Builder
	for _, msg := range m.messages {
		username := msg.User
		if name, ok := m.userCache.Get(msg.User); ok {
			username = name
		}

		text := m.replaceLinks(msg.Text)
		text = m.replaceMentions(text)

		ts := formatTimestamp(msg.Timestamp)
		line := fmt.Sprintf("%s %s: %s",
			timestampStyle.Render("["+ts+"]"),
			usernameStyle.Render(username),
			text,
		)
		sb.WriteString(wrapStyle.Render(line) + "\n")
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
