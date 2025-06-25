package slackutils

import (
	"encoding/json"
	"errors"
	"regexp"
	"slices"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

var (
	ErrSectionNotFound = errors.New("section not found")
)

type ChannelSection struct {
	ID                   string `json:"channel_section_id"`
	Name                 string `json:"name"`
	Type                 string `json:"standard"`
	Emoji                string `json:"emoji"`
	NextChannelSectionID string `json:"next_channel_section_id"`
	ChannelIdsPage       struct {
		ChannelIDs []string `json:"channel_ids"`
	} `json:"channel_ids_page"`
}

// List Channel Sections
func GetChannelSections() ([]ChannelSection, error) {
	body, code, err := RawSlackRequestJSON("GET", "users.channelSections.list", nil, nil)
	if err != nil {
		return nil, err
	}

	if code != 200 {
		return nil, errors.New(string(body))
	}

	var rawBody = struct {
		ChannelSections []ChannelSection `json:"channel_sections"`
	}{}

	err = json.Unmarshal(body, &rawBody)
	if err != nil {
		return nil, err
	}

	return rawBody.ChannelSections, nil
}

// Create a new channel section
func CreateSection(name string, emoji string) (string, error) {
	body, _, err := RawSlackRequestFormData("POST", "users.channelSections.create", map[string]string{
		"name":  name,
		"emoji": emoji,
	})
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type GetSectionResponse struct {
	OK      bool           `json:"ok"`
	Section ChannelSection `json:"channel_section"`
}

// Get a list of channels in a section
func GetSectionChannels(sectionID string) ([]string, error) {
	body, _, err := RawSlackRequestJSON("GET", "users.channelSections.get", nil, map[string]string{
		"channel_section_id": sectionID,
	})
	if err != nil {
		return nil, err
	}

	response := GetSectionResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response.Section.ChannelIdsPage.ChannelIDs, nil
}

type moveChannelPayload struct {
	ChannelSectionID string   `json:"channel_section_id"`
	ChannelIDs       []string `json:"channel_ids"`
}

// Move a channel from one section to another
func MoveChannelToSection(channelName string, toSectionName string) error {
	fromSection, err := GetSectionOfChannelName(channelName)
	if err != nil && !errors.Is(err, ErrChannelSectionNotFound) {
		return err
	}

	toSection, err := GetSectionByName(toSectionName)
	if err != nil {
		return err
	}

	channel, err := GetChannelByName(channelName)
	if err != nil {
		return err
	}

	insert := []moveChannelPayload{
		{
			ChannelSectionID: toSection.ID,
			ChannelIDs:       []string{channel.ID},
		},
	}

	insertEncoded, err := json.Marshal(insert)
	if err != nil {
		return err
	}

	logrus.WithField("action", "insert").WithField("section", toSectionName).Debugf("%s", insertEncoded)
	payload := map[string]string{
		"insert": string(insertEncoded),
	}

	if fromSection != nil {
		remove := []moveChannelPayload{
			{
				ChannelSectionID: fromSection.ID,
				ChannelIDs:       []string{channel.ID},
			},
		}

		removeEncoded, err := json.Marshal(remove)
		if err != nil {
			return err
		}

		logrus.WithField("action", "remove").WithField("section", fromSection.Name).Debugf("%s", removeEncoded)
		payload["remove"] = string(removeEncoded)
	}

	body, code, err := RawSlackRequestFormData("POST", "users.channelSections.channels.bulkUpdate", payload)
	if err != nil {
		return err
	}

	if code != 200 {
		return errors.New(string(body))
	}

	return nil
}

func ExecuteSmartSection(sectionName string, re string) error {
	_, err := CreateSection(sectionName, "")
	if err != nil {
		return err
	}

	exp, err := regexp.Compile(re)
	if err != nil {
		return err
	}

	channels, err := GetAllConversations()
	if err != nil {
		return err
	}

	logrus.WithField("expression", re).Debug("checking for any channels matching regex")
	channelsToMove := []slack.Channel{}
	for _, c := range channels {
		if exp.Match([]byte(c.Name)) {
			logrus.WithField("channel", c.Name).Debug("matched channel")
			channelsToMove = append(channelsToMove, c)
		}
	}

	logrus.Debug("getting destination section details")
	section, err := GetSectionByName(sectionName)
	if err != nil {
		return err
	}

	return BulkChannelMove(channelsToMove, section.ID)
}

func BulkChannelMove(channels []slack.Channel, sectionID string) error {
	logrus.WithField("section", sectionID).Debug("moving channels in bulk")
	sections, err := GetChannelSections()
	if err != nil {
		return err
	}

	actionData := map[string]map[string][]string{
		"remove": make(map[string][]string),
		"insert": make(map[string][]string),
	}

	// Build Required Actions
	for _, c := range channels {
		// Get where channel is currently
		fromSection, err := localGetChannelSection(sections, c)
		if err != nil && !errors.Is(err, ErrChannelSectionNotFound) {
			return err
		}

		current_name := "channels"
		if fromSection != nil {
			current_name = fromSection.Name
		}

		logrus.WithField("channel", c.Name).WithField("current_section", current_name).Debug("identified matched channel")

		// Do not move a section that is already in the right section
		if fromSection != nil && fromSection.ID == sectionID {
			logrus.WithField("channel", c.Name).Debug("no action needed")
			continue
		}

		// Add the channel to the right section
		logrus.WithField("channel", c.Name).WithField("action", "insert").WithField("section", sectionID).Debug("adding channel to section")
		actionData["insert"][sectionID] = append(actionData["insert"][sectionID], c.ID)

		// If channel is in another section remove it from there
		if fromSection != nil {
			logrus.WithField("channel", c.Name).WithField("action", "remove").WithField("section", fromSection.ID).Debug("removing channel from section")
			actionData["remove"][fromSection.ID] = append(actionData["remove"][fromSection.ID], c.ID)
		}
	}

	payloadData := map[string][]moveChannelPayload{
		"remove": reduceActionMap(actionData["remove"]),
		"insert": reduceActionMap(actionData["insert"]),
	}

	if payloadData["insert"] == nil {
		return nil
	}

	payload := make(map[string]string)

	insertEncoded, err := json.Marshal(payloadData["insert"])
	if err != nil {
		return err
	}
	payload["insert"] = string(insertEncoded)

	if payloadData["remove"] != nil {
		removeEncoded, err := json.Marshal(payloadData["remove"])
		if err != nil {
			return err
		}
		payload["remove"] = string(removeEncoded)
	}

	logrus.WithField("payload", payload).Debug("sending request")

	body, code, err := RawSlackRequestFormData("POST", "users.channelSections.channels.bulkUpdate", payload)
	if err != nil {
		return err
	}

	logrus.WithField("response", string(body)).Debug("request sent")

	if code != 200 {
		return errors.New(string(body))
	}

	return nil
}

func reduceActionMap(action map[string][]string) (payload []moveChannelPayload) {
	for sectionID, channels := range action {
		payload = append(payload, moveChannelPayload{ChannelSectionID: sectionID, ChannelIDs: channels})
	}
	return payload
}

func localGetChannelSection(sections []ChannelSection, c slack.Channel) (*ChannelSection, error) {
	for _, s := range sections {
		if slices.Contains(s.ChannelIdsPage.ChannelIDs, c.ID) {
			return &s, nil
		}
	}
	return nil, ErrChannelSectionNotFound
}
