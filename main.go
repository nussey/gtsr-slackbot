package main

import (
	"fmt"

	"github.com/nlopes/slack"
)

func main() {
	fmt.Println("Foobar")

	api := slack.New("xoxb-289010713271-SeWO3xt6keY6udA8kkPIkyc6")
	users, err := api.GetUsers()
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	for _, user := range users {
		fmt.Printf("ID: %s, Fullname: %s, Email: %s\n", user.ID, user.Profile.RealName, user.Profile.Email)
	}

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "U2PVAD9B7"))

}
