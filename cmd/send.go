package cmd

import (
	"io"
	"strings"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/graytonio/slack-cli/lib/slackutils"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sendCmd)
}

func parseToArg(arg string) (string, error) {
	if strings.HasPrefix(arg, "@") {
		logrus.WithField("target", arg).WithField("type", "user").Debug("looking up user")
		user := strings.TrimPrefix(arg, "@")
		uID, ok := config.GetConfig().SavedUsers[user]
		if ok {
			return uID, nil
		}

		u, err := slackutils.GetUserByName(user)
		if err != nil {
			logrus.WithError(err).Debug("could not find user")
			return "", err
		}

		logrus.WithField("id", u.ID).Debug("found user")
		return u.ID, nil
	} else if strings.HasPrefix(arg, "#") {
		logrus.WithField("target", arg).WithField("type", "channel_name").Debug("looking up channel")
		channel := strings.TrimPrefix(arg, "#")
		c, err := slackutils.GetChannelByName(channel)
		if err != nil {
			return "", err
		}

		logrus.WithField("id", c.ID).Debug("found channel")
		return c.ID, nil
	} else {
		logrus.WithField("target", arg).WithField("type", "channel_id").Debug("sending to channel")
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

		to, err := parseToArg(args[0])
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