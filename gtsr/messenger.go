package gtsr

import (
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

type GlobalMessenger struct {
	API *slack.Client
}

type Messenger struct {
	channel string
	GlobalMessenger
}

func (msngr *Messenger) Channel() string {
	return msngr.channel
}

type OutgoingMessage struct {
	text    string
	actions []slack.AttachmentAction

	messenger *GlobalMessenger
	channel   string
}

type IncomingMessage struct {
	Text string

	channel   string
	timestamp string

	sb *SlackBot
}

func (inmsg *IncomingMessage) TimeStamp() time.Time {
	millis, _ := strconv.Atoi(strings.Split(inmsg.timestamp, ".")[0])
	return time.Unix(int64(millis), 0)
}

func (inmsg *IncomingMessage) Channel() string {
	return inmsg.sb.Channels[inmsg.channel].Name
}

func (inmsg *IncomingMessage) AddReaction(react string) error {
	return inmsg.sb.API.AddReaction(react, slack.ItemRef{
		Channel:   inmsg.channel,
		Timestamp: inmsg.timestamp,
	})
}

func (gm *GlobalMessenger) NewMessage(text string, channel string) *OutgoingMessage {
	return &OutgoingMessage{
		text:    text,
		channel: channel,

		messenger: gm,
	}
}

func (msnger *Messenger) NewMessage(text string) *OutgoingMessage {
	return &OutgoingMessage{
		text:    text,
		channel: msnger.channel,

		messenger: &msnger.GlobalMessenger,
	}
}

func (msg *OutgoingMessage) Send() error {
	params := slack.PostMessageParameters{
		AsUser: true,
		Attachments: []slack.Attachment{slack.Attachment{
			Actions:    msg.actions,
			CallbackID: randStringRunes(8),
		}},
	}

	_, _, err := msg.messenger.API.PostMessage(msg.channel, msg.text, params)

	return err
}

func (msg *OutgoingMessage) AddButton(label string, id string) {
	action := slack.AttachmentAction{
		Name: label,
		Text: label,
		Type: "button",
	}
	msg.actions = append(msg.actions, action)
}

func (msg *OutgoingMessage) AddDropdown(label string, options []string) {
	action := slack.AttachmentAction{
		Name: label,
		Text: label,
		Type: "select",
	}

	for _, opt := range options {
		aopt := slack.AttachmentActionOption{
			Text:  opt,
			Value: opt,
		}

		action.Options = append(action.Options, aopt)
	}

	msg.actions = append(msg.actions, action)
}

func (gm *GlobalMessenger) Scope(channel string) *Messenger {
	return &Messenger{
		GlobalMessenger: *gm,
		channel:         channel,
	}
}
