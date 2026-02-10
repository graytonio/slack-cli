//go:build darwin

package cookieextracter

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

type SlackCredentials struct {
	Cookie    string `mapstructure:"cookie"`
	UserToken string `mapstructure:"token"`
}

var (
	ErrSlackCookieNotFound = errors.New("could not find slack cookie")
	ErrSlackTokenNotFound  = errors.New("could not find slack user token")
)

// TODO local cache
func GetSlackCredentials(workspace string) (*SlackCredentials, error) {
	data := SlackCredentials{}
	conf, err := GetDarwinConfig()
	if err != nil {
		return nil, err
	}

	cookies, err := GetSlackCookies(conf)
	if err != nil {
		return nil, err
	}

	for _, c := range cookies {
		if c.Name != "d" {
			continue
		}

		data.Cookie = c.Value
		break
	}

	if data.Cookie == "" {
		return nil, ErrSlackCookieNotFound
	}

	logrus.WithField("cookie", data.Cookie).Debug("extrated cookie")
	tokens, err := GetSlackUserTokens(conf.LevelDBPath)
	if err != nil {
		return nil, err
	}

	for _, team := range tokens.Teams {
		if team.Domain != workspace {
			continue
		}

		data.UserToken = team.Token
		break
	}

	if data.UserToken == "" {
		return nil, ErrSlackTokenNotFound
	}

	logrus.WithFields(logrus.Fields{
		"tokens": fmt.Sprintf("%+v", tokens),
	}).Debug("got local user tokens")

	return &data, nil
}
