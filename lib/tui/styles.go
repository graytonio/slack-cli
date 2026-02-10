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

	mentionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33"))

	linkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Underline(true)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	// Input prompt
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("62"))

	// Help bar
	helpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	// Favorites overlay
	favOverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(40)

	favTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	favSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Bold(true)

	favItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	favEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	favHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	favBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)
)
