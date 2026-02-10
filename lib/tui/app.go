package tui

import (
	"fmt"
	"strings"

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
	favorites FavoritesModel
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
		favorites:  NewFavoritesModel(),
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
		// When favorites overlay is open, route all keys there (except ctrl+c)
		if m.favorites.visible && msg.String() != "ctrl+c" {
			if msg.String() == "ctrl+f" {
				m.favorites.visible = false
				return m, nil
			}
			var cmd tea.Cmd
			var sel *ChannelSelectedMsg
			m.favorites, cmd, sel = m.favorites.Update(msg)
			if sel != nil {
				fetchCmd := m.switchToChannel(sel.ChannelID, sel.ChannelName)
				if cmd != nil {
					return m, tea.Batch(cmd, fetchCmd)
				}
				return m, fetchCmd
			}
			if cmd != nil {
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+f":
			m.favorites.visible = !m.favorites.visible
			return m, nil
		case "ctrl+a":
			if m.channelID != "" {
				if m.favorites.IsFavorite(m.channelID) {
					m.favorites.Remove(m.channelID)
					m.statusText = fmt.Sprintf("Removed #%s from favorites", m.channelName)
				} else {
					if m.favorites.Add(m.channelID, m.channelName) {
						m.statusText = fmt.Sprintf("Added #%s to favorites", m.channelName)
					} else {
						m.statusText = "Favorites full (max 9)"
					}
				}
				m.syncFavBadges()
				return m, m.favorites.persistCmd()
			}
			return m, nil
		case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7", "alt+8", "alt+9":
			idx := int(msg.String()[len("alt+")] - '1')
			if sel := m.favorites.GetSlot(idx); sel != nil {
				m.favorites.visible = false
				return m, m.switchToChannel(sel.ID, sel.Name)
			}
			return m, nil
		case "tab":
			m.cycleFocus(true)
			return m, nil
		case "shift+tab":
			m.cycleFocus(false)
			return m, nil
		case "ctrl+k":
			m.setFocus(focusChannels)
			// Send "/" to activate the list's built-in filter
			filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
			var cmd tea.Cmd
			m.channels, cmd = m.channels.Update(filterMsg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		case "ctrl+n":
			m.setFocus(focusInput)
			return m, nil
		case "ctrl+l":
			m.setFocus(focusChannels)
			return m, nil
		case "esc":
			if m.focus == focusInput {
				m.setFocus(focusMessages)
				return m, nil
			}
			if m.focus == focusChannels {
				// Let escape pass through to the channel list so it can
				// cancel an active filter before we handle it here.
				break
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
		m.syncFavBadges()
		return m, tea.Batch(cmds...)

	case FavoritesSavedMsg:
		m.syncFavBadges()
		return m, nil

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
		Height(m.height - 4). // -1 status bar, -1 help bar, -2 borders
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
	chatHeight := m.height - inputHeight - 3 - 2 // -3 for input border, -1 status, -1 help bar
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

	// Help bar
	helpBar := helpBarStyle.
		Width(m.width).
		Render("Ctrl+K Filter  Ctrl+L Channels  Ctrl+N New Message  Ctrl+F Favorites  Ctrl+A Add Fav  Alt+N Jump  Tab Next Panel")

	base := mainLayout + "\n" + statusBar + "\n" + helpBar

	// Overlay favorites panel if visible
	if m.favorites.visible {
		overlay := m.favorites.View()
		base = placeOverlay(m.width, m.height, overlay, base)
	}

	return base
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

func (m *AppModel) switchToChannel(channelID, channelName string) tea.Cmd {
	m.channelID = channelID
	m.channelName = channelName
	m.chatView.SetChannel(channelName)
	m.input.SetChannel(channelID)
	m.statusText = fmt.Sprintf("Loading #%s...", channelName)
	m.setFocus(focusMessages)
	return fetchMessages(channelID)
}

func (m *AppModel) syncFavBadges() {
	slots := make(map[string]int)
	for i, f := range m.favorites.items {
		if f.ID != "" {
			slots[f.ID] = i + 1
		}
	}
	m.channels.UpdateFavSlots(slots)
}

func (m *AppModel) updateSizes() {
	rightWidth := m.width - channelListWidth - 6 // borders + padding
	if rightWidth < 10 {
		rightWidth = 10
	}

	inputHeight := 1
	chatHeight := m.height - inputHeight - 9 // borders, status, help bar, padding
	if chatHeight < 3 {
		chatHeight = 3
	}

	channelHeight := m.height - 6
	if channelHeight < 3 {
		channelHeight = 3
	}

	m.channels.SetSize(channelListWidth-2, channelHeight)
	m.chatView.SetSize(rightWidth, chatHeight)
	m.input.SetWidth(rightWidth)
}

// placeOverlay renders the overlay string centered on top of the base string.
func placeOverlay(width, height int, overlay, base string) string {
	overlayLines := strings.Split(overlay, "\n")
	baseLines := strings.Split(base, "\n")

	// Pad base to full height if needed
	for len(baseLines) < height {
		baseLines = append(baseLines, "")
	}

	overlayWidth := 0
	for _, line := range overlayLines {
		w := lipgloss.Width(line)
		if w > overlayWidth {
			overlayWidth = w
		}
	}
	overlayHeight := len(overlayLines)

	// Center position
	startX := (width - overlayWidth) / 2
	startY := (height - overlayHeight) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for i, overlayLine := range overlayLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}

		baseLine := baseLines[y]
		baseW := lipgloss.Width(baseLine)

		// Pad base line if shorter than startX
		if baseW < startX {
			baseLine += strings.Repeat(" ", startX-baseW)
			baseW = startX
		}

		// Build the merged line: left of overlay + overlay + right of overlay
		// Use rune-aware slicing via measuring visible width
		left := truncateToWidth(baseLine, startX)
		right := ""
		rightStart := startX + lipgloss.Width(overlayLine)
		if rightStart < baseW {
			right = skipToWidth(baseLine, rightStart)
		}

		baseLines[y] = left + overlayLine + right
	}

	return strings.Join(baseLines, "\n")
}

// truncateToWidth returns the prefix of s that fits within maxWidth visible columns.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(s)
	result := ""
	w := 0
	for _, r := range runes {
		next := result + string(r)
		w = lipgloss.Width(next)
		if w > maxWidth {
			break
		}
		result = next
	}
	// Pad if we're short
	if w < maxWidth {
		result += strings.Repeat(" ", maxWidth-w)
	}
	return result
}

// skipToWidth returns the suffix of s starting at the given visible column.
func skipToWidth(s string, startWidth int) string {
	if startWidth <= 0 {
		return s
	}
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		w = lipgloss.Width(string(runes[:i+1]))
		if w >= startWidth {
			return string(runes[i+1:])
		}
		_ = r
	}
	return ""
}
