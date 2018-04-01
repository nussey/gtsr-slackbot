package gtsr

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/nlopes/slack"
)

var rngRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

// TODO(nussey): Add authentication layer for user permissions

const (
	DM      = 'D'
	ChanMsg = 'C'
)

type SlackBot struct {
	APIKey string

	API     *slack.Client
	RTM     *slack.RTM
	running bool

	Users    map[string]*slack.User
	Channels map[string]*slack.Channel

	GM *GlobalMessenger

	Plugins []SlackPlugin
}

type SlackPlugin interface {
	Init() *PluginConfig // Set up the plugin
	Teardown()

	ParseMessage(*IncomingMessage, *Messenger) error
}

type PluginConfig struct {
	Name        string
	Description string
	Version     string

	FeatureConvo bool
	Topics       []*ConvoTopic

	FeatureChron bool
	Jobs         []*CronJob
}

type CronJob struct {
	Name     string
	Interval time.Duration

	// All cron actions must be fully threadsafe
	Action func(string) error
}

type ConvoTopic struct {
	Name  string
	Label string

	// All action functions must be fully threadsafe
	Action func(User) error
}

type User struct {
	Name   string
	Handle string
	Perms  Permissions
}

type Permissions struct {
	Admin       bool
	Exec        bool
	SubteamLead bool
}

func InitSlack(key string) *SlackBot {
	// TODO(nussey): Move off of the global RNG
	rand.Seed(time.Now().UnixNano())

	bot := &SlackBot{
		APIKey: key,
		API:    slack.New(key),
	}
	bot.RTM = bot.API.NewRTM()
	bot.GM = &GlobalMessenger{
		API: bot.API,
	}

	return bot
}

func (sb *SlackBot) AddPlugin(plugin SlackPlugin) {
	plugin.Init()

	sb.Plugins = append(sb.Plugins, plugin)
}

func (sb *SlackBot) ServeSlack() error {
	go sb.RTM.ManageConnection()

	// NewMessage("test").Send("testing", sb.API)
	// TODO MAKE SURE THE MESSAGE IS NOT COMING FROM MYSELF

	for msg := range sb.RTM.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello

		case *slack.ConnectedEvent:
			sb.fetchChannels()
			sb.fetchUsers()

		case *slack.MessageEvent:
			fmt.Println(ev.Channel)
			if ev.User == sb.RTM.GetInfo().User.ID {
				continue
			}

			msgType := ev.Channel[0]

			if msgType == ChanMsg {
				sb.parseMessage(ev)
			}
			if msgType == DM {
				sb.dispatchConversation(ev)
			}

		case *slack.PresenceChangeEvent:
			// Ignore presence change events

		case *slack.LatencyReport:
			// Ignore incoming latency reports

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			return nil

		default:
			// Ignore other events..
		}
	}
	return nil
}

func (sb *SlackBot) parseMessage(ev *slack.MessageEvent) {
	channel := sb.Channels[ev.Channel]
	scopedMessenger := sb.GM.Scope(channel.Name)

	msg := &IncomingMessage{
		Text: ev.Text,

		channel:   ev.Channel,
		timestamp: ev.Timestamp,

		sb: sb,
	}

	for _, plugin := range sb.Plugins {
		// TODO(nussey): Handle errors
		plugin.ParseMessage(msg, scopedMessenger)
	}
}

func (sb *SlackBot) dispatchConversation(ev *slack.MessageEvent) {

}

func (sb *SlackBot) fetchUsers() error {
	sb.Users = make(map[string]*slack.User)
	users := sb.RTM.GetInfo().Users

	for user := range users {
		sb.Users[users[user].ID] = &users[user]
	}

	return nil
}

func (sb *SlackBot) fetchChannels() error {
	sb.Channels = make(map[string]*slack.Channel)
	chans := sb.RTM.GetInfo().Channels

	for channel := range chans {
		sb.Channels[chans[channel].ID] = &chans[channel]
	}

	return nil
}

func randStringRunes(n int) string {
	// TODO(nussey): Move off of the global RNG
	b := make([]rune, n)
	for i := range b {
		b[i] = rngRunes[rand.Intn(len(rngRunes))]
	}
	return string(b)
}
