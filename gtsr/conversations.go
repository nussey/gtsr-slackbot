package gtsr

import (
	"sync"

	"github.com/nlopes/slack"
)

const convoQueueSize = 10

type directMessage struct {
	mutex *sync.Mutex

	currentConvo *conversation
	convoQueue   chan *conversation
}

func (dm *directMessage) manageDM() {
	for {
		// block until a conversation enters the queue
		convo := <-dm.convoQueue

		// Mark it the current conversation
		dm.mutex.Lock()
		dm.currentConvo = convo
		dm.mutex.Unlock()

		// Walk through the script
		convo.script(convo.msngr)
		dm.mutex.Lock()
		dm.currentConvo = nil
		dm.mutex.Unlock()
	}
}

// LOCK BEFORE YOU USE THIS
func (dm *directMessage) queueEmpty() bool {
	return len(dm.convoQueue) == 0
}

type conversation struct {
	msngr *Messenger

	script ConvoAction
}

// ConvoAction describes the function signature needed to act
// as a conversation entry point
type ConvoAction func(*Messenger) error

// A ConvoTopic is a possible topic of conversation to be registered
// by a plugin
type ConvoTopic struct {
	// Unique ID for the topic - only alphanumeric
	ID string
	// Human friendly name for the conversation topic - this what the
	// user will see
	Label string

	// The entry point for the conversation.
	// All action functions must only access global datastores in a threadsafe fasion
	Action ConvoAction

	// Standin - implement later
	Permissions *Permissions
}

func (sb *SlackBot) smalltalk(msngr *Messenger) error {
	msg := msngr.NewMessage(helpText)
	msg.AddDropdown("Topics", sb.SortedConvoTopics()).AddButton("Cancel").Send()
	cont, rsp := msngr.AwaitResponse()
	if !cont {
		msngr.NewMessage("We can finish this conversation some other time!").Send()
		return nil
	}
	// TODO(nussey) Match against keywords and do fuzzy find

	if rsp == "Cancel" {
		return msngr.UpdateLastMessage("No problem! Let me know if I can help you later.", ColorDanger)
	}

	var topic *ConvoTopic
	var ok bool
	if topic, ok = sb.topics[rsp]; !ok {
		msngr.UpdateLastMessage("I'm sorry, I am not sure what you mean by that :disappointed:", ColorWarning)
		return nil
	}
	sb.gm.NewConversation(msngr.ChannelName(), topic.Action)
	msngr.UpdateLastMessage(rsp, ColorGood)
	return nil
}

func (sb *SlackBot) dispatchConversation(ev *slack.MessageEvent) error {
	user := sb.ims[ev.Channel].Name

	dm := sb.gm.dms[user]

	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	if dm.currentConvo != nil {
		dm.currentConvo.msngr.respond(ev.Text, false)
		return nil
	}

	sb.gm.NewConversation(user, sb.smalltalk)

	return nil
}

// Start a new conversation with a user
// If the user is alreay having a conversation with the slackbot,
// this gets added to the queue
func (gm *GlobalMessenger) NewConversation(user string, script ConvoAction) {
	if string(user[0]) == "@" {
		user = user[1:]
	}
	convo := &conversation{
		msngr:  gm.scope("@" + user),
		script: script,
	}

	gm.dms[user].convoQueue <- convo

}
