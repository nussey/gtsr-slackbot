package gtsr

import "github.com/nlopes/slack"

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

type Message struct {
	Text    string
	Actions []slack.AttachmentAction

	messenger *GlobalMessenger
	channel   string
}

func (gm *GlobalMessenger) NewMessage(text string, channel string) *Message {
	return &Message{
		Text:    text,
		channel: channel,

		messenger: gm,
	}
}

func (msnger *Messenger) NewMessage(text string) *Message {
	return &Message{
		Text:    text,
		channel: msnger.channel,

		messenger: &msnger.GlobalMessenger,
	}
}

func (msg *Message) Send() error {
	params := slack.PostMessageParameters{
		AsUser: true,
		Attachments: []slack.Attachment{slack.Attachment{
			Actions:    msg.Actions,
			CallbackID: randStringRunes(8),
		}},
	}

	_, _, err := msg.messenger.API.PostMessage(msg.channel, msg.Text, params)

	return err
}

func (msg *Message) AddButton(label string, id string) {
	action := slack.AttachmentAction{
		Name: label,
		Text: label,
		Type: "button",
	}
	msg.Actions = append(msg.Actions, action)
}

func (msg *Message) AddDropdown(label string, options []string) {
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

	msg.Actions = append(msg.Actions, action)
}

func (gm *GlobalMessenger) Scope(channel string) *Messenger {
	return &Messenger{
		GlobalMessenger: *gm,
		channel:         channel,
	}
}
