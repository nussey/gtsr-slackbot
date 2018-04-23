package gtsr

import (
	"context"
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
	// ColorDanger is the Slack standard red attachment color
	ColorDanger = "danger"

	DefaultTimeout = time.Minute * 15
)

type GlobalMessenger struct {
	API *slack.Client

	userIds map[string]string

	dms      map[string]*directMessage
	listener *callbackListener
}

func (gm *GlobalMessenger) mapIds(users map[string]*slack.User) {

	gm.userIds = make(map[string]string)

	for k, user := range users {
		gm.userIds[user.Name] = k
	}
}

// A Messenger provides scope, tracks state, and allows the sending
// of messages to the scoped channel
type Messenger struct {
	channel string
	gm      *GlobalMessenger

	lastMessage *OutgoingMessage

	callbackID string
	mailbox    chan string
}

// ChannelName returns the human friendly name of the channel in scope
func (msngr *Messenger) ChannelName() string {
	return msngr.channel
}

// An OutgoingMessage represents a new message waiting to be sent.
// The zero value is not helpful - always get them from Messengers
type OutgoingMessage struct {
	text string

	interactive bool
	callbackID  string
	// Map of interactive element IDs to lables
	elements map[string]string
	actions  []slack.AttachmentAction

	messenger *Messenger
	channel   string

	sent bool
	ts   string
}

func (gm *GlobalMessenger) scope(channel string) *Messenger {
	return &Messenger{
		gm:      gm,
		channel: channel,

		mailbox: make(chan string, 1),
	}
}

// Don't call this, use a regular messenger
func (gm *GlobalMessenger) sendMessage(msg *OutgoingMessage) error {
	if msg.callbackID == "" {
		msg.callbackID = "NOCALLBACK"
	}

	params := slack.PostMessageParameters{
		AsUser: true,
		Attachments: []slack.Attachment{slack.Attachment{
			Actions: msg.actions,
			// CallbackID: randStringRunes(8),
			CallbackID: msg.callbackID,
		}},
	}

	_, ts, err := gm.API.PostMessage(msg.channel, msg.text, params)
	if err != nil {
		return err
	}

	msg.sent = true
	msg.ts = ts

	return nil
}

func (gm *GlobalMessenger) updateMessage(msg *OutgoingMessage, newText string, color string) error {
	if !msg.sent {
		return nil
	}

	// TODO(nussey): make this wwaaayyyy less brittle
	channel := gm.userIds[msg.channel[1:]]
	attach := slack.Attachment{
		Color: color,
		Text:  newText,
	}
	_, _, _, err := gm.API.SendMessageContext(context.Background(), channel, slack.MsgOptionUpdate(msg.ts), slack.MsgOptionText(msg.text, true), slack.MsgOptionAttachments(attach))
	return err
}

func (msngr *Messenger) sendMessage(msg *OutgoingMessage) error {
	callbackID := randStringRunes(8)
	msngr.gm.listener.registerCallback(callbackID, msngr)
	msg.callbackID = callbackID
	return msngr.gm.sendMessage(msg)
}

// UpdateLastMessage replaces the interactive components of the last
// message sent with plain text. This is not required, but is generally
// preferable from a UX perspective.
func (msngr *Messenger) UpdateLastMessage(text string, color string) error {
	return msngr.gm.updateMessage(msngr.lastMessage, text, color)
}

// NewMessage creates a new OutgoingMessage within the scope of the
// Messenger.
func (msngr *Messenger) NewMessage(text string) *OutgoingMessage {
	if msngr.lastMessage != nil {
		msngr.gm.listener.unregisterCallback(msngr.lastMessage.callbackID)
	}

	msg := &OutgoingMessage{
		text:    text,
		channel: msngr.channel,

		messenger: msngr,

		elements: make(map[string]string),
	}

	msngr.lastMessage = msg
	return msg
}

// AwaitResponse blocks unti the last message is responded to - only
// available during conversations. Returns true if the conversation
// should continue, false if not. If true, the second return contains
// the user's answer
func (msngr *Messenger) AwaitResponse() (bool, string) {
	return msngr.AwaitRespondseTimeout(DefaultTimeout)
}

func (msngr *Messenger) AwaitRespondseTimeout(timeout time.Duration) (bool, string) {
	select {
	case resp := <-msngr.mailbox:
		return true, resp
	case <-time.After(timeout):
		return false, "timeout"
	}
}

func (msngr *Messenger) respond(text string, interactive bool) {
	if interactive {
		text = msngr.lastMessage.elements[text]
	}
	// This can probably be much more robust - right now if multiple
	// responses are recieved, all but the first is dropped on the ground
	select {
	case msngr.mailbox <- text:
	default:
	}
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
	msg.elements[id] = label
	action := slack.AttachmentAction{
		Name:  label,
		Text:  label,
		Value: id,
		Type:  "button",
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
		msg.elements[id] = opt
		aopt := slack.AttachmentActionOption{
			Text:  opt,
			Value: id,
		}

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
