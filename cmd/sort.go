package cmd

import (
	"errors"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/graytonio/slack-cli/lib/slackutils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sortCmd)
}

var sortCmd = &cobra.Command{
	Use:   "sort [section] [expression]",
	Short: "Sort through all channels and sort them into a category based on a regex",
	Long:  "Runs a filter over all the channels in the sidebar and moves them to a specified category. If no arguments are passed every filter configured will be executed. If only a section name is passed then that configured section will be run. If a section name and a regex expression are passed then that expression will be used and matched channels will be moved into the specified section",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.WithField("args", args).WithField("length", len(args)).Debug("sorting channels")

		// Run all configured filters
		if len(args) == 0 {
			for _, s := range config.GetConfig().SmartSections {
				err := slackutils.ExecuteSmartSection(s.SectionName, s.ReExpression)
				if err != nil {
					return err
				}
			}
			return nil
			// Run Specific Filter
		} else if len(args) == 1 {
			for _, s := range config.GetConfig().SmartSections {
				if s.SectionName == args[0] {
					return slackutils.ExecuteSmartSection(args[0], s.ReExpression)
				}
			}
			// Run AdHock Filter
		} else if len(args) == 2 {
			return slackutils.ExecuteSmartSection(args[0], args[1])
		}

		return errors.New("no config found")
	},
}
