package cmd

import (
	"errors"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(aliasCmd)
}

var aliasCmd = &cobra.Command{
	Use:   "alias <user|channel> name id",
	Short: "Save a channel or user to reference by name",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "user":
			config.AddUserCache(args[1], args[2])
		case "channel":
			config.AddChannelCache(args[1], args[2])
		default:
			return errors.New("valid save types are user or channel")
		}

		return nil
	},
}
