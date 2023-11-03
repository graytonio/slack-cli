package cmd

import (
	"io"
	"strings"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/graytonio/slack-cli/lib/slackutils"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
}

func parseToArg(arg string) (string, error) {
	if strings.HasPrefix(arg, "@") {
		user := strings.TrimPrefix(arg, "@")
		uID, ok := config.GetConfig().SavedUsers[user]
		if ok {
			return uID, nil
		}

		u, err := slackutils.GetUserByName(user)
		if err != nil {
			return "", err
		}

		return u.ID, nil
	} else if strings.HasPrefix(arg, "#") {
		channel := strings.TrimPrefix(arg, "#")
		c, err := slackutils.GetChannelByName(channel)
		if err != nil {
			return "", err
		}

		return c.ID, nil
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