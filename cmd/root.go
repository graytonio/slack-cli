package cmd

import (
	"fmt"
	"os"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "slack-cli",
	Short: "Terminal based slack interface",
  }

  func init() {
	cobra.OnInitialize(func() {
		config.SetLogLevel()
	})

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable debug logging")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
  }
  
  func Execute() {
	if err := rootCmd.Execute(); err != nil {
	  fmt.Fprintln(os.Stderr, err)
	  os.Exit(1)
	}
  }