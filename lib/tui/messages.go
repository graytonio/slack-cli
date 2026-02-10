package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/graytonio/slack-cli/lib/config"
	"github.com/graytonio/slack-cli/lib/slackutils"
	"github.com/slack-go/slack"
)

// --- Message types ---

// ChannelsLoadedMsg is sent when channels have been fetched from the API.
type ChannelsLoadedMsg struct {
	Channels []slack.Channel
	Err      error
}

// ChannelSelectedMsg is sent when the user selects a channel.
type ChannelSelectedMsg struct {
	ChannelID   string
	ChannelName string
}

// MessagesLoadedMsg is sent when messages have been fetched for a channel.
type MessagesLoadedMsg struct {
	Messages  []slack.Message
	ChannelID string
	Err       error
}

// NewMessagesMsg is sent when polling finds new messages.
type NewMessagesMsg struct {
	Messages  []slack.Message
	ChannelID string
	Err       error
}

// UserResolvedMsg is sent when a user ID has been resolved to a display name.
type UserResolvedMsg struct {
	UserID string
	Name   string
	Err    error
}

// MessageSentMsg is sent after a message has been sent.
type MessageSentMsg struct {
	ChannelID string
	Err       error
}

// TickMsg triggers polling for new messages.
type TickMsg struct{}

// StatusMsg updates the status bar text.
type StatusMsg struct {
	Text string
}

// EmojiLoadedMsg is sent when custom workspace emoji have been fetched.
type EmojiLoadedMsg struct {
	Emojis map[string]string
	Err    error
}

// FavoritesSavedMsg is sent after favorites have been persisted to config.
type FavoritesSavedMsg struct{}

// ThreadOpenMsg is sent when the user presses Enter on a threaded message.
type ThreadOpenMsg struct {
	ChannelID string
	ThreadTS  string
}

// ThreadRepliesLoadedMsg is sent when thread replies have been fetched.
type ThreadRepliesLoadedMsg struct {
	Messages  []slack.Message
	ChannelID string
	ThreadTS  string
	Err       error
}

// NewThreadRepliesMsg is sent when polling finds new thread replies.
type NewThreadRepliesMsg struct {
	Messages  []slack.Message
	ChannelID string
	ThreadTS  string
	Err       error
}

// ThreadReplySentMsg is sent after a thread reply has been sent.
type ThreadReplySentMsg struct {
	ChannelID string
	ThreadTS  string
	Err       error
}

// --- Commands ---

func fetchChannels() tea.Cmd {
	return func() tea.Msg {
		channels, err := slackutils.GetAllConversations()
		return ChannelsLoadedMsg{Channels: channels, Err: err}
	}
}

func fetchMessages(channelID string) tea.Cmd {
	return func() tea.Msg {
		resp, err := config.SlackClient.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID:          channelID,
			Limit:              50,
			IncludeAllMetadata: true,
		})
		if err != nil {
			return MessagesLoadedMsg{ChannelID: channelID, Err: err}
		}
		return MessagesLoadedMsg{Messages: resp.Messages, ChannelID: channelID}
	}
}

func pollMessages(channelID, latestTS string) tea.Cmd {
	return func() tea.Msg {
		if channelID == "" {
			return NewMessagesMsg{}
		}
		resp, err := config.SlackClient.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID:          channelID,
			Oldest:             latestTS,
			Limit:              100,
			IncludeAllMetadata: true,
		})
		if err != nil {
			return NewMessagesMsg{ChannelID: channelID, Err: err}
		}
		return NewMessagesMsg{Messages: resp.Messages, ChannelID: channelID}
	}
}

func resolveUser(userID string) tea.Cmd {
	return func() tea.Msg {
		user, err := config.SlackClient.GetUserInfo(userID)
		if err != nil {
			return UserResolvedMsg{UserID: userID, Err: err}
		}
		name := user.Profile.DisplayName
		if name == "" {
			name = user.RealName
		}
		if name == "" {
			name = user.Name
		}
		return UserResolvedMsg{UserID: userID, Name: name}
	}
}

func sendMessage(channelID, text string) tea.Cmd {
	return func() tea.Msg {
		_, _, _, err := config.SlackClient.SendMessage(channelID, slack.MsgOptionText(text, false))
		return MessageSentMsg{ChannelID: channelID, Err: err}
	}
}

func fetchEmoji() tea.Cmd {
	return func() tea.Msg {
		emojis, err := config.SlackClient.GetEmoji()
		return EmojiLoadedMsg{Emojis: emojis, Err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

func fetchThreadReplies(channelID, threadTS string) tea.Cmd {
	return func() tea.Msg {
		msgs, _, _, err := config.SlackClient.GetConversationReplies(&slack.GetConversationRepliesParameters{
			ChannelID: channelID,
			Timestamp: threadTS,
			Inclusive: true,
			Limit:     200,
		})
		if err != nil {
			return ThreadRepliesLoadedMsg{ChannelID: channelID, ThreadTS: threadTS, Err: err}
		}
		return ThreadRepliesLoadedMsg{Messages: msgs, ChannelID: channelID, ThreadTS: threadTS}
	}
}

func pollThreadReplies(channelID, threadTS, latestTS string) tea.Cmd {
	return func() tea.Msg {
		if channelID == "" || threadTS == "" {
			return NewThreadRepliesMsg{}
		}
		msgs, _, _, err := config.SlackClient.GetConversationReplies(&slack.GetConversationRepliesParameters{
			ChannelID: channelID,
			Timestamp: threadTS,
			Oldest:    latestTS,
			Limit:     100,
		})
		if err != nil {
			return NewThreadRepliesMsg{ChannelID: channelID, ThreadTS: threadTS, Err: err}
		}
		return NewThreadRepliesMsg{Messages: msgs, ChannelID: channelID, ThreadTS: threadTS}
	}
}

func sendThreadReply(channelID, threadTS, text string) tea.Cmd {
	return func() tea.Msg {
		_, _, _, err := config.SlackClient.SendMessage(channelID,
			slack.MsgOptionText(text, false),
			slack.MsgOptionTS(threadTS),
		)
		return ThreadReplySentMsg{ChannelID: channelID, ThreadTS: threadTS, Err: err}
	}
}
