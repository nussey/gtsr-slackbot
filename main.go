package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/nussey/gtsr-slackbot/gtsr"
	"github.com/nussey/gtsr-slackbot/plugins/helptext"
)

const keyFileLocation = "./keys.json"

type KeysFile struct {
	SlackAPIKey string
}

// Uptime plugin

func main() {
	raw, err := ioutil.ReadFile(keyFileLocation)
	if err != nil {
		panic(err)
	}

	var keys = &KeysFile{}
	json.Unmarshal(raw, keys)

	fmt.Println(keys.SlackAPIKey)
	bot := gtsr.InitSlack(keys.SlackAPIKey)
	bot.AddPlugin(&helptext.HelpTextBot{})
	bot.ServeSlack()
}
