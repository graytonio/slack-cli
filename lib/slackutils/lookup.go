package slackutils

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/graytonio/slack-cli/lib/config"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func ParseChannelTarget(arg string) (string, error) {
	if strings.HasPrefix(arg, "@") {
		logrus.WithField("target", arg).WithField("type", "user").Debug("looking up user")
		user := strings.TrimPrefix(arg, "@")
		uID, ok := config.GetConfig().SavedUsers[user]
		if ok {
			return uID, nil
		}

		u, err := GetUserByName(user)
		if err != nil {
			logrus.WithError(err).Debug("could not find user")
			return "", err
		}

		logrus.WithField("id", u.ID).Debug("found user")
		return u.ID, nil
	} else if strings.HasPrefix(arg, "#") {
		logrus.WithField("target", arg).WithField("type", "channel_name").Debug("looking up channel")
		channel := strings.TrimPrefix(arg, "#")
		c, err := GetChannelByName(channel)
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

// Get the Definition of a channel section by name
func GetSectionByName(name string) (*ChannelSection, error) {
	sections, err := GetChannelSections()
	if err != nil {
		return nil, err
	}

	for _, s := range sections {
		if s.Name == name {
			return &s, nil
		}
	}

	return nil, ErrSectionNotFound
}

func GetSectionOfChannelName(channel string) (*ChannelSection, error) {
	c, err := GetChannelByName(channel)
	if err != nil {
		return nil, err
	}

	sections, err := GetChannelSections()
	if err != nil {
		return nil, err
	}

	for _, s := range sections {
		logrus.WithFields(logrus.Fields{
			"section_name":     s.Name,
			"section_id":       s.ID,
			"section_channels": s.ChannelIdsPage.ChannelIDs,
			"channel":          channel,
		}).Debug("checking section for channel")
		if slices.Contains(s.ChannelIdsPage.ChannelIDs, c.ID) {
			return &s, nil
		}
	}

	return nil, ErrChannelSectionNotFound
}

// Lookup channel object by name
func GetChannelByName(name string) (channel *slack.Channel, err error) {
	channels, err := GetAllConversations()
	if err != nil {
		return nil, err
	}

	for _, c := range channels {
		if c.Name == name {
			return &c, nil
		}
	}

	return nil, ErrChannelNotFound
}

// FIXME Too slow (cache search users)
// Lookup a user by their display name
func GetUserByName(name string) (*slack.User, error) {
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

type userBootResponseData struct {
	Channels []slack.Channel `json:"channels"`
}

func GetAllConversations() (channels []slack.Channel, err error) {
	body, _, err := RawSlackRequestJSON("POST", "client.userBoot", nil, nil)
	if err != nil {
		return nil, err
	}

	data := userBootResponseData{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data.Channels, nil
}

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrChannelNotFound        = errors.New("channel not found")
	ErrChannelSectionNotFound = errors.New("section for channel not found")
)
