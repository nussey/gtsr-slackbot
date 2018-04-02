package gtsr

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

// Several methods will accept a color. RGB codes can be given
// in the form of Hex values (#FFFFFF-#000000), but these short
// cuts can be used directly as well
const (
	// ColorGood is the Slack standard green attachment color
	ColorGood = "good"
	// ColorWarning is the Slack standard orange attachment color
	ColorWarning = "warning"
	// ColorDanger is the Slack standard green attachment color
	ColorDanger = "danger"
)

type globalMessenger struct {
	API *slack.Client
}

// A Messenger provides scope, tracks state, and allows the sending
// of messages to the scoped channel
type Messenger struct {
	channel string
	gm      *globalMessenger

	mailbox chan string
}

// Channel returns the human friendly name of the channel in scope
func (msngr *Messenger) Channel() string {
	return msngr.channel
}

// An OutgoingMessage represents a new message waiting to be sent.
// The zero value is not helpful - always get them from Messengers
type OutgoingMessage struct {
	text        string
	callbacks   map[string]string
	actions     []slack.AttachmentAction
	interactive bool

	messenger *Messenger
	channel   string
}

func (gm *globalMessenger) scope(channel string) *Messenger {
	return &Messenger{
		gm:      gm,
		channel: channel,
	}
}

func (gm *globalMessenger) sendMessage(msg *OutgoingMessage, callbackID string) error {
	if callbackID == "" {
		callbackID = "NOCALLBACK"
	}

	fmt.Println(callbackID)
	params := slack.PostMessageParameters{
		AsUser: true,
		Attachments: []slack.Attachment{slack.Attachment{
			Actions: msg.actions,
			// CallbackID: randStringRunes(8),
			CallbackID: callbackID,
		}},
	}

	_, _, err := gm.API.PostMessage(msg.channel, msg.text, params)

	return err
}

func (msngr *Messenger) sendMessage(msg *OutgoingMessage) error {
	callbackID := randStringRunes(8)
	return msngr.gm.sendMessage(msg, callbackID)
}

// UpdateLastMessage replaces the interactive components of the last
// message sent with plain text. This is not required, but is generally
// preferable from a UX perspective.
func (msngr *Messenger) UpdateLastMessage(text string, color string) error {

	return nil
}

// NewMessage creates a new OutgoingMessage within the scope of the
// Messenger.
func (msngr *Messenger) NewMessage(text string) *OutgoingMessage {
	return &OutgoingMessage{
		text:    text,
		channel: msngr.channel,

		messenger: msngr,

		callbacks: make(map[string]string),
	}
}

// AwaitResponse blocks unti the last message is responded to - only
// available during conversations. Returns true if the conversation
// should continue, false if not. If true, the second return contains
// the user's answer
func (msngr *Messenger) AwaitResponse() (bool, string) {

	return false, ""
}

// Send generates metadata and sends the OutgoingMessage to slack. It can
// possibly return an error
func (msg *OutgoingMessage) Send() error {
	return msg.messenger.sendMessage(msg)
}

// AddButton creates an interactive button attachment on the message.
// label is the text that will appear on the button. The original
// message pointer is returned to allow method chaining
func (msg *OutgoingMessage) AddButton(label string) *OutgoingMessage {
	msg.interactive = true

	id := randStringRunes(8)
	msg.callbacks[id] = label
	action := slack.AttachmentAction{
		Name: label,
		Text: id,
		Type: "button",
	}
	msg.actions = append(msg.actions, action)

	return msg
}

// AddDropdown creates an interactive dropdown menu on the message
// label is the default text that will appear before a selection is
// made. The original message pointer is returned to allow
// method chaining
func (msg *OutgoingMessage) AddDropdown(label string, options []string) *OutgoingMessage {
	msg.interactive = true

	action := slack.AttachmentAction{
		Name: label,
		Text: label,
		Type: "select",
	}

	for _, opt := range options {
		id := randStringRunes(8)
		msg.callbacks[id] = opt
		aopt := slack.AttachmentActionOption{
			Text:  opt,
			Value: id,
		}

		fmt.Println("Id: " + id + ", Name:" + opt)
		action.Options = append(action.Options, aopt)
	}

	msg.actions = append(msg.actions, action)

	return msg
}

// An IncomingMessage models a message just recieved from Slack
type IncomingMessage struct {
	// The textual contents of the message
	Text string

	channel   string
	timestamp string

	sb *SlackBot
}

// TimeStamp returns the Go time.Time version of the message
// timestamp. This representation is NOT garunteed to be unique
// among messages
func (inmsg *IncomingMessage) TimeStamp() time.Time {
	millis, _ := strconv.Atoi(strings.Split(inmsg.timestamp, ".")[0])
	return time.Unix(int64(millis), 0)
}

// Channel returns the human readable name of the channel of the
// IncomingMessage was sent in/to
func (inmsg *IncomingMessage) Channel() string {
	return inmsg.sb.channels[inmsg.channel].Name
}

// AddReaction makes the SlackBot add react reaction to the
// recieved message. The react should be specified without :
func (inmsg *IncomingMessage) AddReaction(react string) error {
	return inmsg.sb.api.AddReaction(react, slack.ItemRef{
		Channel:   inmsg.channel,
		Timestamp: inmsg.timestamp,
	})
}
