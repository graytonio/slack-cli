package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel wraps a text input for sending messages.
type InputModel struct {
	textInput textinput.Model
	channelID string
	width     int
}

func NewInputModel() InputModel {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Prompt = "> "
	ti.CharLimit = 4000

	return InputModel{textInput: ti}
}

func (m InputModel) Init() tea.Cmd {
	return nil
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			text := m.textInput.Value()
			if text == "" || m.channelID == "" {
				return m, nil
			}
			m.textInput.Reset()
			return m, tea.Batch(
				sendMessage(m.channelID, text),
				func() tea.Msg {
					return StatusMsg{Text: "Sending message..."}
				},
			)
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	return m.textInput.View()
}

func (m *InputModel) Focus() tea.Cmd {
	return m.textInput.Focus()
}

func (m *InputModel) Blur() {
	m.textInput.Blur()
}

func (m *InputModel) SetChannel(id string) {
	m.channelID = id
}

func (m *InputModel) SetWidth(width int) {
	m.width = width
	m.textInput.Width = width - 4 // Account for prompt and padding
}
