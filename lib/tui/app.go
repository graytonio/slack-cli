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
	focusThread
	focusThreadInput
)

const channelListWidth = 30

// AppModel is the root Bubble Tea model for the TUI.
type AppModel struct {
	channels   ChannelListModel
	chatView   ChatViewModel
	input      InputModel
	threadView ThreadViewModel
	favorites  FavoritesModel
	userCache  *UserCache
	emojiCache *EmojiCache

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
	ec := NewEmojiCache()
	return AppModel{
		channels:   NewChannelListModel(),
		chatView:   NewChatViewModel(uc, ec),
		input:      NewInputModel(),
		threadView: NewThreadViewModel(uc, ec),
		favorites:  NewFavoritesModel(),
		userCache:  uc,
		emojiCache: ec,
		focus:      focusChannels,
		statusText: "Loading channels...",
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(fetchChannels(), fetchEmoji(), tickCmd())
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
			if m.focus == focusThread || m.focus == focusThreadInput {
				m.threadView.Close()
				m.setFocus(focusMessages)
				m.updateSizes()
				return m, nil
			}
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

	case EmojiLoadedMsg:
		if msg.Err == nil {
			m.emojiCache.SetCustom(msg.Emojis)
			// Re-render chat with emoji resolved
			if len(m.chatView.messages) > 0 {
				content := m.chatView.renderMessages()
				m.chatView.viewport.SetContent(content)
			}
		}
		return m, nil

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
		if m.threadView.visible {
			var tcmd tea.Cmd
			m.threadView, tcmd = m.threadView.Update(msg)
			return m, tea.Batch(cmd, tcmd)
		}
		return m, cmd

	case MessageSentMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Send error: %v", msg.Err)
			return m, nil
		}
		m.statusText = fmt.Sprintf("Connected to #%s", m.channelName)
		return m, fetchMessages(m.channelID)

	case ThreadOpenMsg:
		m.threadView.Open(msg.ChannelID, msg.ThreadTS, m.channelName)
		m.setFocus(focusThread)
		m.updateSizes()
		return m, fetchThreadReplies(msg.ChannelID, msg.ThreadTS)

	case ThreadRepliesLoadedMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Error loading thread: %v", msg.Err)
		}
		var cmd tea.Cmd
		m.threadView, cmd = m.threadView.Update(msg)
		return m, cmd

	case NewThreadRepliesMsg:
		var cmd tea.Cmd
		m.threadView, cmd = m.threadView.Update(msg)
		return m, cmd

	case ThreadReplySentMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Reply error: %v", msg.Err)
			return m, nil
		}
		m.statusText = fmt.Sprintf("Connected to #%s", m.channelName)
		return m, fetchThreadReplies(msg.ChannelID, msg.ThreadTS)

	case TickMsg:
		cmds = append(cmds, tickCmd())
		if m.channelID != "" {
			latestTS := m.chatView.LatestTimestamp()
			cmds = append(cmds, pollMessages(m.channelID, latestTS))
		}
		if m.threadView.visible {
			latestTS := m.threadView.LatestTimestamp()
			cmds = append(cmds, pollThreadReplies(m.threadView.channelID, m.threadView.threadTS, latestTS))
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
	case focusThread, focusThreadInput:
		var cmd tea.Cmd
		m.threadView, cmd = m.threadView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Calculate right-side dimensions first so channel panel can match
	rightWidth := m.width - channelListWidth - 4 // account for borders
	if rightWidth < 10 {
		rightWidth = 10
	}

	// Split right side between chat and thread if thread is visible
	chatWidth := rightWidth
	threadWidth := 0
	if m.threadView.visible {
		threadWidth = rightWidth * 40 / 100
		chatWidth = rightWidth - threadWidth - 2 // -2 for thread border
		if chatWidth < 10 {
			chatWidth = 10
		}
		if threadWidth < 10 {
			threadWidth = 10
		}
	}

	inputHeight := 3
	chatHeight := m.height - inputHeight - 3 - 3 // -3 for input border, -1 top padding, -1 status, -1 help bar
	if chatHeight < 3 {
		chatHeight = 3
	}

	// Channel list panel (left) — height matches chat + input + their borders
	channelBorder := unfocusedBorder
	if m.focus == focusChannels {
		channelBorder = focusedBorder
	}
	channelPanelHeight := chatHeight + inputHeight + 2 // +2 for input border top/bottom
	channelPanel := channelBorder.
		Width(channelListWidth).
		Height(channelPanelHeight).
		Render(m.channels.View())

	// Chat viewport (right top)
	chatBorder := unfocusedBorder
	if m.focus == focusMessages {
		chatBorder = focusedBorder
	}

	chatPanel := chatBorder.
		Width(chatWidth).
		Height(chatHeight).
		Render(m.chatView.View())

	// Input panel (right bottom)
	inputBorder := unfocusedBorder
	if m.focus == focusInput {
		inputBorder = focusedBorder
	}
	inputPanel := inputBorder.
		Width(chatWidth).
		Height(inputHeight).
		Render(m.input.View())

	// Compose chat side
	chatSide := lipgloss.JoinVertical(lipgloss.Left, chatPanel, inputPanel)

	var mainLayout string
	if m.threadView.visible {
		// Thread panel
		threadBorder := unfocusedBorder
		if m.focus == focusThread || m.focus == focusThreadInput {
			threadBorder = focusedBorder
		}
		threadPanel := threadBorder.
			Width(threadWidth).
			Height(channelPanelHeight). // match channel panel height
			Render(m.threadView.View())

		rightSide := lipgloss.JoinHorizontal(lipgloss.Top, chatSide, threadPanel)
		mainLayout = lipgloss.JoinHorizontal(lipgloss.Top, channelPanel, rightSide)
	} else {
		mainLayout = lipgloss.JoinHorizontal(lipgloss.Top, channelPanel, chatSide)
	}

	// Status bar
	statusBar := statusBarStyle.
		Width(m.width).
		Render(m.statusText)

	// Help bar
	helpText := "Ctrl+K Filter  Ctrl+L Channels  Ctrl+N New Message  Ctrl+F Favorites  Ctrl+A Add Fav  Alt+N Jump  Tab Next Panel"
	if m.threadView.visible {
		helpText += "  Esc Close Thread"
	} else if m.focus == focusMessages {
		helpText += "  Enter Open Thread"
	}
	helpBar := helpBarStyle.
		Width(m.width).
		Render(helpText)

	base := "\n" + mainLayout + "\n" + statusBar + "\n" + helpBar

	// Overlay favorites panel if visible
	if m.favorites.visible {
		overlay := m.favorites.View()
		base = placeOverlay(m.width, m.height, overlay, base)
	}

	return base
}

func (m *AppModel) cycleFocus(forward bool) {
	panels := []focusPanel{focusChannels, focusMessages, focusInput}
	if m.threadView.visible {
		panels = append(panels, focusThread, focusThreadInput)
	}
	n := len(panels)
	// Find current index
	cur := 0
	for i, p := range panels {
		if p == m.focus {
			cur = i
			break
		}
	}
	if forward {
		cur = (cur + 1) % n
	} else {
		cur = (cur + n - 1) % n
	}
	m.focus = panels[cur]
	m.applyFocus()
}

func (m *AppModel) setFocus(f focusPanel) {
	m.focus = f
	m.applyFocus()
}

func (m *AppModel) applyFocus() {
	m.input.Blur()
	m.threadView.BlurAll()
	switch m.focus {
	case focusInput:
		m.input.Focus()
	case focusThread:
		m.threadView.FocusViewport()
	case focusThreadInput:
		m.threadView.FocusInput()
	}
}

func (m *AppModel) switchToChannel(channelID, channelName string) tea.Cmd {
	m.channelID = channelID
	m.channelName = channelName
	m.chatView.SetChannel(channelName)
	m.input.SetChannel(channelID)
	m.statusText = fmt.Sprintf("Loading #%s...", channelName)
	if m.threadView.visible {
		m.threadView.Close()
		m.updateSizes()
	}
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

	chatWidth := rightWidth
	threadWidth := 0
	if m.threadView.visible {
		threadWidth = rightWidth * 40 / 100
		chatWidth = rightWidth - threadWidth - 2
		if chatWidth < 10 {
			chatWidth = 10
		}
		if threadWidth < 10 {
			threadWidth = 10
		}
	}

	inputHeight := 1
	chatHeight := m.height - inputHeight - 10 // borders, status, help bar, top padding
	if chatHeight < 3 {
		chatHeight = 3
	}

	// Channel height derived from chat + input so they stay aligned
	channelHeight := chatHeight + inputHeight + 2 // +2 for input border
	if channelHeight < 3 {
		channelHeight = 3
	}

	m.channels.SetSize(channelListWidth-2, channelHeight)
	m.chatView.SetSize(chatWidth, chatHeight)
	m.input.SetWidth(chatWidth)

	if m.threadView.visible {
		m.threadView.SetSize(threadWidth, channelHeight)
	}
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
