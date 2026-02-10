package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/slack-go/slack"
)

// channelItem implements list.Item for the channel list.
type channelItem struct {
	channel slack.Channel
}

func (i channelItem) Title() string       { return "#" + i.channel.Name }
func (i channelItem) Description() string { return i.channel.Topic.Value }
func (i channelItem) FilterValue() string { return i.channel.Name }

// ChannelListModel wraps a bubbles list for channels.
type ChannelListModel struct {
	list     list.Model
	channels []slack.Channel
	width    int
	height   int
}

func NewChannelListModel() ChannelListModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Channels"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()

	return ChannelListModel{list: l}
}

func (m ChannelListModel) Init() tea.Cmd {
	return nil
}

func (m ChannelListModel) Update(msg tea.Msg) (ChannelListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ChannelsLoadedMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.channels = msg.Channels
		items := make([]list.Item, len(msg.Channels))
		for i, ch := range msg.Channels {
			items[i] = channelItem{channel: ch}
		}
		m.list.SetItems(items)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "enter" {
			if item, ok := m.list.SelectedItem().(channelItem); ok {
				return m, func() tea.Msg {
					return ChannelSelectedMsg{
						ChannelID:   item.channel.ID,
						ChannelName: item.channel.Name,
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ChannelListModel) View() string {
	return m.list.View()
}

func (m *ChannelListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
}

func (m ChannelListModel) SelectedChannel() (slack.Channel, bool) {
	if item, ok := m.list.SelectedItem().(channelItem); ok {
		return item.channel, true
	}
	return slack.Channel{}, false
}
