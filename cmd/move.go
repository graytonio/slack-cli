package cmd

import (
	"github.com/graytonio/slack-cli/lib/slackutils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(moveCmd)
}

var moveCmd = &cobra.Command{
	Use: "move <channel> <section>",
	Short: "Move a channel to a new section",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		channel := args[0]
		section := args[1]
		return slackutils.MoveChannelToSection(channel, section)
	},
}
