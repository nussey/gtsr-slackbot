package sysadmin

import (
	"strings"

	"github.com/nussey/gtsr-slackbot/gtsr"
)

type SysAdminBot struct {
}

func (sa *SysAdminBot) Init() *gtsr.PluginConfig {
	return &gtsr.PluginConfig{
		Name:        "SysAdmin Bot",
		Description: "Helps plugin developers see what is going on inside clippy",
		Version:     "1.0",

		FeatureConvo: false,
		Topics:       []*gtsr.ConvoTopic{},

		FeatureChron: false,
		Jobs:         []*gtsr.CronJob{},
	}

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
