package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Focus panel identifiers
type focusPanel int

const (
	focusChannels focusPanel = iota
	focusMessages
	focusInput
)

const channelListWidth = 30

// AppModel is the root Bubble Tea model for the TUI.
type AppModel struct {
	channels  ChannelListModel
	chatView  ChatViewModel
	input     InputModel
	userCache *UserCache

	focus       focusPanel
	statusText  string
	channelID   string
	channelName string
	width       int
	height      int
	ready       bool
}

func NewAppModel() AppModel {
	uc := NewUserCache()
	return AppModel{
		channels:   NewChannelListModel(),
		chatView:   NewChatViewModel(uc),
		input:      NewInputModel(),
		userCache:  uc,
		focus:      focusChannels,
		statusText: "Loading channels...",
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(fetchChannels(), tickCmd())
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.cycleFocus(true)
			return m, nil
		case "shift+tab":
			m.cycleFocus(false)
			return m, nil
		case "esc":
			if m.focus == focusInput {
				m.setFocus(focusMessages)
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateSizes()
		return m, nil

	case ChannelsLoadedMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Error loading channels: %v", msg.Err)
			return m, nil
		}
		m.statusText = fmt.Sprintf("Loaded %d channels", len(msg.Channels))
		var cmd tea.Cmd
		m.channels, cmd = m.channels.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case ChannelSelectedMsg:
		m.channelID = msg.ChannelID
		m.channelName = msg.ChannelName
		m.chatView.SetChannel(msg.ChannelName)
		m.input.SetChannel(msg.ChannelID)
		m.statusText = fmt.Sprintf("Loading #%s...", msg.ChannelName)
		m.setFocus(focusMessages)
		return m, fetchMessages(msg.ChannelID)

	case MessagesLoadedMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			m.statusText = fmt.Sprintf("Connected to #%s", m.channelName)
		}
		var cmd tea.Cmd
		m.chatView, cmd = m.chatView.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case NewMessagesMsg:
		var cmd tea.Cmd
		m.chatView, cmd = m.chatView.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case UserResolvedMsg:
		var cmd tea.Cmd
		m.chatView, cmd = m.chatView.Update(msg)
		return m, cmd

	case MessageSentMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Send error: %v", msg.Err)
			return m, nil
		}
		m.statusText = fmt.Sprintf("Connected to #%s", m.channelName)
		return m, fetchMessages(m.channelID)

	case TickMsg:
		cmds = append(cmds, tickCmd())
		if m.channelID != "" {
			latestTS := m.chatView.LatestTimestamp()
			cmds = append(cmds, pollMessages(m.channelID, latestTS))
		}
		return m, tea.Batch(cmds...)

	case StatusMsg:
		m.statusText = msg.Text
		return m, nil
	}

	// Route to focused panel
	switch m.focus {
	case focusChannels:
		var cmd tea.Cmd
		m.channels, cmd = m.channels.Update(msg)
		cmds = append(cmds, cmd)
	case focusMessages:
		var cmd tea.Cmd
		m.chatView, cmd = m.chatView.Update(msg)
		cmds = append(cmds, cmd)
	case focusInput:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Channel list panel (left)
	channelBorder := unfocusedBorder
	if m.focus == focusChannels {
		channelBorder = focusedBorder
	}
	channelPanel := channelBorder.
		Width(channelListWidth).
		Height(m.height - 3). // -1 for status bar, -2 for borders
		Render(m.channels.View())

	// Chat viewport (right top)
	chatBorder := unfocusedBorder
	if m.focus == focusMessages {
		chatBorder = focusedBorder
	}
	rightWidth := m.width - channelListWidth - 4 // account for borders
	if rightWidth < 10 {
		rightWidth = 10
	}

	inputHeight := 3
	chatHeight := m.height - inputHeight - 3 - 1 // -3 for input border, -1 for status
	if chatHeight < 3 {
		chatHeight = 3
	}

	chatPanel := chatBorder.
		Width(rightWidth).
		Height(chatHeight).
		Render(m.chatView.View())

	// Input panel (right bottom)
	inputBorder := unfocusedBorder
	if m.focus == focusInput {
		inputBorder = focusedBorder
	}
	inputPanel := inputBorder.
		Width(rightWidth).
		Height(inputHeight).
		Render(m.input.View())

	// Compose right side
	rightSide := lipgloss.JoinVertical(lipgloss.Left, chatPanel, inputPanel)

	// Compose main layout
	mainLayout := lipgloss.JoinHorizontal(lipgloss.Top, channelPanel, rightSide)

	// Status bar
	statusBar := statusBarStyle.
		Width(m.width).
		Render(m.statusText)

	return mainLayout + "\n" + statusBar
}

func (m *AppModel) cycleFocus(forward bool) {
	if forward {
		m.focus = (m.focus + 1) % 3
	} else {
		m.focus = (m.focus + 2) % 3
	}
	m.applyFocus()
}

func (m *AppModel) setFocus(f focusPanel) {
	m.focus = f
	m.applyFocus()
}

func (m *AppModel) applyFocus() {
	m.input.Blur()
	if m.focus == focusInput {
		m.input.Focus()
	}
}

func (m *AppModel) updateSizes() {
	rightWidth := m.width - channelListWidth - 6 // borders + padding
	if rightWidth < 10 {
		rightWidth = 10
	}

	inputHeight := 1
	chatHeight := m.height - inputHeight - 8 // borders, status, padding
	if chatHeight < 3 {
		chatHeight = 3
	}

	channelHeight := m.height - 5
	if channelHeight < 3 {
		channelHeight = 3
	}

	m.channels.SetSize(channelListWidth-2, channelHeight)
	m.chatView.SetSize(rightWidth, chatHeight)
	m.input.SetWidth(rightWidth)
}
