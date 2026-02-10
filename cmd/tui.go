package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/graytonio/slack-cli/lib/tui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(tuiCmd)
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open interactive TUI for browsing and sending messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(tui.NewAppModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		return nil
	},
}
