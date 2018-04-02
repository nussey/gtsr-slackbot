package gtsr

import (
	"github.com/nlopes/slack"
)

type conversation struct {
	InConversation bool
	msngr          *Messenger
}

// ConvoAction describes the function signature needed to act
// as a conversation entry point
type ConvoAction func(string) error

// A ConvoTopic is a possible topic of conversation to be registered
// by a plugin
type ConvoTopic struct {
	// Unique ID for the topic - only alphanumeric
	ID string
	// Human friendly name for the conversation topic - this what the
	// user will see
	Label string

	// The entry point for the conversation.
	// All action functions must be fully threadsafe.
	Action ConvoAction
}

func (sb *SlackBot) dispatchConversation(ev *slack.MessageEvent) {
	user := sb.dms[ev.Channel].Name

	if _, ok := sb.converations[user]; !ok {
		sb.newConversation(user)
	}

	msg := sb.converations[user].msngr.NewMessage("test")
	msg.AddDropdown("How can I help?", []string{"abc", "xyz"}).Send()
}

func (sb *SlackBot) newConversation(user string) {
	convo := &conversation{
		InConversation: false,
		msngr:          sb.gm.scope("@" + user),
	}

	sb.converations[user] = convo
}
