package sysadmin

import (
	"strings"

	"github.com/nussey/gtsr-slackbot/gtsr"
)

type SysAdminBot struct {
}

func (sa *SysAdminBot) Init() *gtsr.PluginConfig {
	debug := &gtsr.ConvoTopic{
		ID:          "debug",
		Label:       "Debugger",
		Permissions: &gtsr.Permissions{},

		Action: sa.debugger,
	}

	poker := &gtsr.CronJob{
		ID:   "poker",
		Name: "Developer Poker",

		Spec:   "@every 15m",
		Action: sa.poke,
	}

	return &gtsr.PluginConfig{
		Name:        "SysAdmin Bot",
		Description: "Helps plugin developers see what is going on inside clippy",
		Version:     "1.0",

		FeatureConvo: true,
		Topics:       []*gtsr.ConvoTopic{debug},

		FeatureCron: true,
		Jobs:        []*gtsr.CronJob{poker},
	}

}

func (sa *SysAdminBot) debugger(messenger *gtsr.Messenger) error {
	msg := messenger.NewMessage("What's up hackerman?")
	msg.AddButton("Ping").AddButton("Pong")
	msg.AddDropdown("Foobar", []string{"bar", "foo"})
	msg.Send()

	cont, rsp := messenger.AwaitResponse()
	if !cont {
		messenger.NewMessage("Really? You ignoring me?").Send()
		return nil
	}

	messenger.UpdateLastMessage(rsp+", really?", gtsr.ColorGood)
	messenger.NewMessage("See? Now I can do stuff with your response, including ask another question").Send()
	return nil
}

func (sa *SysAdminBot) poke(gm *gtsr.GlobalMessenger) error {
	gm.NewConversation("nussey", func(messenger *gtsr.Messenger) error {
		return messenger.NewMessage("CODE FASTER!").Send()
	})

	return nil
}

func (sa *SysAdminBot) Teardown() {

}

func (sa *SysAdminBot) ParseMessage(msg *gtsr.IncomingMessage, messenger *gtsr.Messenger) error {
	if match_ping(msg.Text) {
		// TODO(nussey): send an etherial message first asking if they are curious
		return messenger.NewMessage("pong").Send()
	}

	return nil
}

func match_ping(msg string) bool {
	// TODO(nussey): actually scan the words and see if they were asking about the network drive
	msg = strings.ToLower(msg)
	return msg == "ping"
}
