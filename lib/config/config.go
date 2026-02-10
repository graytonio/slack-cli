package config

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"

	"github.com/graytonio/slack-cli/lib/cookieextracter"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/spf13/viper"
)

var home, _ = os.UserHomeDir()

type SmartSection struct {
	SectionName  string `mapstructure:"section"`
	ReExpression string `mapstructure:"re"`
}

type FavoriteChannel struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

type Config struct {
	Workspace         string                            `mapstructure:"workspace"`
	SlackCredentials  *cookieextracter.SlackCredentials `mapstructure:"credentials"`
	SavedChannels     map[string]string                 `mapstructure:"channel_cache"`
	SavedUsers        map[string]string                 `mapstructure:"users_cache"`
	SmartSections     []SmartSection                    `mapstructure:"smart_sections"`
	FavoriteChannels  []FavoriteChannel                 `mapstructure:"favorite_channels"`
}

var config = Config{
	SlackCredentials: &cookieextracter.SlackCredentials{},
	SavedChannels:    make(map[string]string),
	SavedUsers:       make(map[string]string),
	SmartSections:    []SmartSection{},
	FavoriteChannels: []FavoriteChannel{},
}

var SlackClient *slack.Client
var SlackHTTPClient *http.Client

func init() {
	log.SetOutput(io.Discard)
	viper.SetDefault("workspace", "")

	viper.SetConfigFile(path.Join(home, ".config/slackcli.yaml"))
	viper.SafeWriteConfigAs(path.Join(home, ".config/slackcli.yaml"))
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(err)
	}

	if config.SlackCredentials.Cookie == "" || config.SlackCredentials.UserToken == "" {
		if config.Workspace == "" {
			fmt.Println("workspace not configured please set in config file ~/.config/slackcli.yaml")
			os.Exit(1)
		}

		fmt.Printf("Fetching credentials for %s make sure slack app is quit\n", config.Workspace)
		loadCredentials()
	}

	initSlackClient()
}

func SetLogLevel() {
	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func AddUserCache(name string, id string) {
	config.SavedUsers[name] = id
	viper.Set("users_cache", config.SavedUsers)
	viper.WriteConfig()
}

func AddChannelCache(name string, id string) {
	config.SavedChannels[name] = id
	viper.Set("channel_cache", config.SavedChannels)
	viper.WriteConfig()
}

func loadCredentials() {
	creds, err := cookieextracter.GetSlackCredentials(config.Workspace)
	if err != nil {
		panic(err)
	}

	viper.SetConfigFile(path.Join(home, ".config/slackcli.yaml"))
	viper.SafeWriteConfigAs(path.Join(home, ".config/slackcli.yaml"))
	viper.Set("credentials.cookie", creds.Cookie)
	viper.Set("credentials.token", creds.UserToken)
	viper.WriteConfig()
	viper.Unmarshal(&config)
}

func initSlackClient() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	cookieURL, _ := url.Parse("https://slack.com")
	jar.SetCookies(cookieURL, []*http.Cookie{
		{
			Name:  "d",
			Value: GetConfig().SlackCredentials.Cookie,
		},
	})

	SlackHTTPClient = &http.Client{
		Jar: jar,
	}

	SlackClient = slack.New(GetConfig().SlackCredentials.UserToken, slack.OptionHTTPClient(SlackHTTPClient))
}

func SaveFavorites(favs []FavoriteChannel) {
	config.FavoriteChannels = favs
	viper.Set("favorite_channels", favs)
	viper.WriteConfig()
}

func GetConfig() *Config {
	return &config
}
