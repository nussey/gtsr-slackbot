package ryanbot

import (
	"regexp"

	"github.com/nussey/gtsr-slackbot/gtsr"
)

type RyanBot struct {
}

var hmmreg = regexp.MustCompile(`([Hh])+([Mm])+`)

func (rb *RyanBot) Init() *gtsr.PluginConfig {
	return &gtsr.PluginConfig{
		Name:        "Ryan Bot",
		Description: "Allows clippy to immitate Ryan",
		Version:     "1.0",

		FeatureConvo: false,
		Topics:       []*gtsr.ConvoTopic{},

		FeatureChron: false,
		Jobs:         []*gtsr.CronJob{},
	}

}

func (rb *RyanBot) Teardown() {

}

func (rb *RyanBot) ParseMessage(msg *gtsr.IncomingMessage, messenger *gtsr.Messenger) error {
	if match_Hmm(msg.Text) {
		return msg.AddReaction("hmm")
	}

	return nil
}

func match_Hmm(text string) bool {
	return hmmreg.MatchString(text)
}
