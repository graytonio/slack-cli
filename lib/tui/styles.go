package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Panel borders
	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	unfocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	// Channel list
	channelListStyle = lipgloss.NewStyle().Padding(0, 1)

	// Chat header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	// Message formatting
	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	usernameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33"))

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	// Input prompt
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("62"))
)
