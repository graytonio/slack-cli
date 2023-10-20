package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
}

func parseToArg(arg string) (string, error) {
	if strings.HasPrefix(arg, "@") {
		user, ok := config.GetConfig().SavedUsers[strings.TrimPrefix(arg, "@")]
		if !ok {
			return "", fmt.Errorf("user %s not found in cache", arg)
		}
		return user, nil
	} else if strings.HasPrefix(arg, "#") {
		channel, ok := config.GetConfig().SavedChannels[strings.TrimPrefix(arg, "#")]
		if !ok {
			return "", fmt.Errorf("user %s not found in cache", arg)
		}
		return channel, nil
	} else {
		return arg, nil
	}
}

var sendCmd = &cobra.Command{
	Use: "send <to> <message>",
	Short: "Send a message to a channel",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := args[1]

		if message == "-" {
			stdin, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			message = string(stdin)
		}

		to, err := parseToArg( args[0])
		if err != nil {
			return err
		}
		
		_, _, _, err = config.SlackClient.SendMessage(to, slack.MsgOptionText(message, false))
		if err != nil {
			return err
		}

		return nil
	},
}