package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/slack-go/slack"
)

// ThreadViewModel displays a thread's messages in a side panel.
type ThreadViewModel struct {
	viewport    viewport.Model
	input       textinput.Model
	messages    []slack.Message
	channelID   string
	threadTS    string
	channelName string
	userCache   *UserCache
	emojiCache  *EmojiCache
	visible     bool
	focusInput  bool
	width       int
	height      int
	ready       bool
}

func NewThreadViewModel(userCache *UserCache, emojiCache *EmojiCache) ThreadViewModel {
	ti := textinput.New()
	ti.Placeholder = "Reply in thread..."
	ti.Prompt = "> "
	ti.CharLimit = 4000

	return ThreadViewModel{
		userCache:  userCache,
		emojiCache: emojiCache,
		input:      ti,
	}
}

func (m *ThreadViewModel) Open(channelID, threadTS, channelName string) {
	m.channelID = channelID
	m.threadTS = threadTS
	m.channelName = channelName
	m.visible = true
	m.messages = nil
	m.focusInput = false
	if m.ready {
		m.viewport.SetContent("Loading thread...")
		m.viewport.GotoTop()
	}
}

func (m *ThreadViewModel) Close() {
	m.visible = false
	m.messages = nil
	m.input.Blur()
	m.input.Reset()
}

func (m ThreadViewModel) Update(msg tea.Msg) (ThreadViewModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.focusInput {
			switch msg.String() {
			case "enter":
				text := m.input.Value()
				if text == "" || m.channelID == "" || m.threadTS == "" {
					return m, nil
				}
				m.input.Reset()
				return m, tea.Batch(
					sendThreadReply(m.channelID, m.threadTS, text),
					func() tea.Msg {
						return StatusMsg{Text: "Sending reply..."}
					},
				)
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
		// Focus is on thread viewport
		switch msg.String() {
		case "up", "k":
			m.viewport.ScrollUp(1)
			return m, nil
		case "down", "j":
			m.viewport.ScrollDown(1)
			return m, nil
		}
		return m, nil

	case ThreadRepliesLoadedMsg:
		if msg.Err != nil {
			m.viewport.SetContent(fmt.Sprintf("Error loading thread: %v", msg.Err))
			return m, nil
		}
		m.messages = msg.Messages
		content := m.renderMessages()
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()
		cmds = append(cmds, m.resolveUnknownUsers()...)
		return m, tea.Batch(cmds...)

	case NewThreadRepliesMsg:
		if msg.Err != nil || msg.ThreadTS != m.threadTS || len(msg.Messages) == 0 {
			return m, nil
		}
		// Merge new messages (avoid duplicates by timestamp)
		existing := make(map[string]bool)
		for _, em := range m.messages {
			existing[em.Timestamp] = true
		}
		added := 0
		for _, nm := range msg.Messages {
			if !existing[nm.Timestamp] {
				m.messages = append(m.messages, nm)
				added++
			}
		}
		if added == 0 {
			return m, nil
		}
		content := m.renderMessages()
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()
		cmds = append(cmds, m.resolveUnknownUsers()...)
		return m, tea.Batch(cmds...)

	case UserResolvedMsg:
		if msg.Err == nil {
			m.userCache.Set(msg.UserID, msg.Name)
			if m.visible && len(m.messages) > 0 {
				content := m.renderMessages()
				m.viewport.SetContent(content)
			}
		}
		return m, nil
	}

	return m, nil
}

func (m ThreadViewModel) View() string {
	if !m.visible {
		return ""
	}

	header := threadHeaderStyle.Render("Thread in #" + m.channelName)

	inputView := m.input.View()

	return header + "\n" + m.viewport.View() + "\n" + inputView
}

func (m *ThreadViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	headerHeight := 1
	inputHeight := 1
	vpHeight := height - headerHeight - inputHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	if !m.ready {
		m.viewport = viewport.New(width, vpHeight)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = vpHeight
	}
	m.input.Width = width - 4
}

func (m *ThreadViewModel) FocusInput() {
	m.focusInput = true
	m.input.Focus()
}

func (m *ThreadViewModel) FocusViewport() {
	m.focusInput = false
	m.input.Blur()
}

func (m *ThreadViewModel) BlurAll() {
	m.focusInput = false
	m.input.Blur()
}

func (m ThreadViewModel) LatestTimestamp() string {
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[len(m.messages)-1].Timestamp
}

func (m ThreadViewModel) renderMessages() string {
	if len(m.messages) == 0 {
		return "No replies"
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
		text = m.emojiCache.Replace(text)

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

func (m ThreadViewModel) replaceMentions(text string) string {
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

func (m ThreadViewModel) replaceLinks(text string) string {
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

func (m ThreadViewModel) resolveUnknownUsers() []tea.Cmd {
	seen := make(map[string]bool)
	var cmds []tea.Cmd
	for _, msg := range m.messages {
		if msg.User != "" && !seen[msg.User] {
			if _, ok := m.userCache.Get(msg.User); !ok {
				seen[msg.User] = true
				cmds = append(cmds, resolveUser(msg.User))
			}
		}
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
