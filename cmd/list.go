package cmd

import (
	"fmt"
	"strings"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/graytonio/slack-cli/lib/slackutils"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var channelListLimit int
var channelListChunkSize int
var channelListOutputFormat string

func init() {
	listCmd.PersistentFlags().IntVarP(&channelListLimit, "limit", "l", 500, "How many messages to return total")
	listCmd.PersistentFlags().IntVarP(&channelListChunkSize, "chunk", "c", 100, "How many messages to fetch at a time. Helpful for optimizing large fetches")
	listCmd.PersistentFlags().StringVar(&channelListOutputFormat, "format", "${user_id}: ${text}", "Format to output messages in")
	rootCmd.AddCommand(listCmd)
}

func TSprintf(format string, params map[string]interface{}) string {
	for key, val := range params {
		format = strings.Replace(format, "${"+key+"}", fmt.Sprintf("%s", val), -1)
	}
	return format
}

var listCmd = &cobra.Command{
	Use:   "list <channel>",
	Short: "List messages in a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := slackutils.ParseChannelTarget(args[0])
		if err != nil {
			return err
		}

		total := channelListLimit
		cursor := ""

		for total > 0  {
			resp, err := config.SlackClient.GetConversationHistory(&slack.GetConversationHistoryParameters{
				ChannelID:          target,
				Limit:              channelListChunkSize,
				Cursor:             cursor,
				IncludeAllMetadata: true,
			})
			if err != nil {
				return err
			}
      cursor = resp.ResponseMetadata.Cursor
 
			for _, m := range resp.Messages {
				fmt.Println(TSprintf(channelListOutputFormat, map[string]any{
					"user_id":   m.User,
					"text":      m.Text,
					"timestamp": m.Timestamp,
				}))
			}

      if cursor == "" {
        break
      }

			total = total - channelListChunkSize
		}

		return nil
	},
}
