package slackutils

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

// TODO Very slow and time consuming see if can fetch data from local machine instead
func GetChannelIDByName(name string) (*slack.Channel, error) {
	var (
		list []slack.Channel
		cursor string
		err error
		limiter = rate.NewLimiter(rate.Every(5*time.Second), 30)
	)
	for {
		limiter.Wait(context.Background())
		list, cursor, err = config.SlackClient.GetConversations(&slack.GetConversationsParameters{
			Types: []string{"public_channel", "private_channel"},
			Cursor: cursor,
			Limit: 200,
			ExcludeArchived: true,
		})
		if err != nil {
			return nil, err
		}

		fmt.Println(len(list))
		if cursor == "dGVhbTpDNFdGMDA4VFk=" {
			fmt.Println("Beginning of list")
		}

		for _, c := range list {
			if strings.EqualFold(c.Name, name) {
				return &c, nil
			}
		}

		if cursor == "" {
			return nil, nil
		}
	}
}

var (
	ErrUserNotFound = errors.New("user not found")
)

// TODO Super slow to process try to find data locally or process request faster
func GetUserIDByName(name string) (*slack.User, error) {
	users, err := config.SlackClient.GetUsers()
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.Profile.DisplayName == name {
			return &u, nil
		}
	}

	return nil, ErrUserNotFound
}