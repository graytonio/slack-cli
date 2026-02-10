package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/slack-go/slack"
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
	var sb strings.Builder
	for _, msg := range m.messages {
		username := msg.User
		if name, ok := m.userCache.Get(msg.User); ok {
			username = name
		}

		ts := formatTimestamp(msg.Timestamp)
		line := fmt.Sprintf("%s %s: %s",
			timestampStyle.Render("["+ts+"]"),
			usernameStyle.Render(username),
			msg.Text,
		)
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func (m ChatViewModel) resolveUnknownUsers() []tea.Cmd {
	seen := make(map[string]bool)
	var cmds []tea.Cmd
	for _, msg := range m.messages {
		if msg.User == "" || seen[msg.User] {
			continue
		}
		if _, ok := m.userCache.Get(msg.User); !ok {
			seen[msg.User] = true
			cmds = append(cmds, resolveUser(msg.User))
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
