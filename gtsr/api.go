// Copyright Alex Nussey
// All Rights Reserved

// Package gtsr provides a wrapper around the Slack API
//
// About
//
// The gtsr (temporary name) package is designed to be an extremely
// easy to use wrapper around the slack API to enable rapid development
// of relatively complex "Slack Bots" plugins. It currently takes
// advantage of the Real Time Messenger (RTM) and Slack Web APIS,
// though the Subscriptions model may be of use in the future. Currently,
// the github.com/nlopes/slack package serves as a middle man for
// authentication and masrhalling.
//
// The current UX implementation is built around the idea of a Bot User. All
// user interactions will take place through regular messages and dms involving
// the bot. Consuming this API allows the bot user to implement 3 core concepts:
// message responses, direct message "conversations", and time triggered
// "cron jobs".
//
// Usage
//
// The consumer of this API must implement at least two distinct sections of code
// to get up and running. First, plugin(s) should be built corresponding to the
// spec/interfaces described below. Second, a SlackBot should be instantiated with
// all of the custom plugins and provided with the necessary credentials. Hosting
// the slack bot requires a public facing IP (or a tunnel program such as ngrok) to
// be able to recieve callbacks for interactive messages
package gtsr

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/nlopes/slack"
	"github.com/robfig/cron"
)

var rngRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

const (
	dm      = 'D'
	chanMsg = 'C'
)

const helpText = "Hi, I'm Clippy, your Solar Racing Assistant! What can I help you with today?"

// A SlackBot is the top level entity for the gtsr slack API.
// The zero value is useless - get a SlackBot from InitSlack()
type SlackBot struct {
	apikey string
	token  string

	api       *slack.Client
	rtm       *slack.RTM
	scheduler *cron.Cron
	running   bool

	users    map[string]*slack.User
	channels map[string]*slack.Channel
	ims      map[string]*slack.User

	gm *GlobalMessenger

	plugins []SlackPlugin

	topics map[string]*ConvoTopic
	crons  map[string]*CronJob
}

// SlackPlugin defins a common method interface all plugins
// must conform to
type SlackPlugin interface {
	// Init should return the accurate and complete configuration for
	// the plugin
	Init() *PluginConfig // Set up the plugin
	// Teardown gives the developer the opportunity to destruct their
	// plugin. May be useful is caching in front of a persistent
	// datastore of some kind
	Teardown()

	// ParseMessage is called for every new message sent in a
	// channel the SlackBot is a member of
	ParseMessage(*IncomingMessage, *Messenger) error
}

// A PluginConfig describes the attributes of a plugin, which features
// it uses, and where to send requests for those features
type PluginConfig struct {
	// Full name of the plugin, this may includes spaces and special characters
	Name string
	// Description of the plugin's functionality
	Description string
	// Current version of the plugin (1.0, 2.3 format)
	Version string

	// Enables the Conversation feature for this plugin
	FeatureConvo bool
	// List of available topics of conversation - must be non
	// empty if the feature is enabled
	Topics []*ConvoTopic

	// Enables the Cron Job feature for this plugin
	FeatureCron bool
	// List of registered Cron Jobs for this plugin - must be
	// non empty if the feature is enabled
	Jobs []*CronJob
}

// A Permissions struct enumerates which permissions are available within
// the current scope. This will be exanded on later
type Permissions struct {
	Admin       bool
	Exec        bool
	SubteamLead bool
}

// InitSlack is the main entry point to the gtsr slack API. The key parameter
// expects a valid slack API key with all the necessary scopes, and
// verificationToken is the shared secret provided by slack to verify
// the authenticity of interactive message callbacks
func InitSlack(key string, verificationToken string) *SlackBot {
	// TODO(nussey): Move off of the global RNG
	rand.Seed(time.Now().UnixNano())

	bot := &SlackBot{
		apikey: key,
		token:  verificationToken,
		api:    slack.New(key),

		topics: make(map[string]*ConvoTopic),
		crons:  make(map[string]*CronJob),
	}

	bot.rtm = bot.api.NewRTM()
	bot.gm = &GlobalMessenger{
		API: bot.api,
		dms: make(map[string]*directMessage),
		listener: &callbackListener{
			callbacks: make(map[string]*Messenger),
			mutex:     &sync.Mutex{},
		},
	}

	return bot
}

// AddPlugin registeres a plugin with the Slack Bot. Make
// all of these calls before ServeSlack()
func (sb *SlackBot) AddPlugin(plugin SlackPlugin) {
	if sb.running {
		panic("Register plugins before starting the Slack Bot")
	}

	config := plugin.Init()
	if config.FeatureConvo {
		for _, topic := range config.Topics {
			if _, ok := sb.topics[topic.Label]; ok {
				panic("Can't load multiple plugins that use the same conversation label")
			}
			sb.topics[topic.Label] = topic
		}
	}

	if config.FeatureCron {
		for _, cron := range config.Jobs {
			if _, ok := sb.crons[cron.ID]; ok {
				panic("Can't load multiple plugins that use the same cron ID")
			}
			sb.crons[cron.ID] = cron
		}
	}

	sb.plugins = append(sb.plugins, plugin)
}

func (sb *SlackBot) SortedConvoTopics() []string {
	var labels []string
	for k := range sb.topics {
		labels = append(labels, k)
	}

	sort.Strings(labels)

	return labels
}

func (sb *SlackBot) refreshData() {
	sb.fetchChannels()
	sb.fetchUsers()
	sb.fetchIMs()

	sb.initDms()

	sb.gm.mapIds(sb.ims)
}

// ServeSlack is a blocking function that handles all network transactions
// for the Slack Bot instance
func (sb *SlackBot) ServeSlack() error {
	// TODO(nussey) fix race condition
	if sb.running {
		panic("There is already an instance of this Slack Bot running! Create a new instance to run two concurrently!")
	}

	sb.running = true

	go sb.rtm.ManageConnection()
	go sb.handleInteractiveMessages()

	sb.initCron()

	for msg := range sb.rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			sb.refreshData()

		case *slack.ConnectedEvent:
			sb.refreshData()
			fmt.Println("We are off!")

		case *slack.MessageEvent:
			if ev.User == sb.rtm.GetInfo().User.ID {
				continue
			}
			// Completely ignore threads for now
			// TODO(nussey): support threads
			if ev.ThreadTimestamp != "" {
				continue
			}

			msgType := ev.Channel[0]

			if msgType == chanMsg {
				sb.parseMessage(ev)
			}
			if msgType == dm {
				sb.dispatchConversation(ev)
			}

		case *slack.ChannelJoinedEvent:
			sb.refreshData()

		case *slack.IMCreatedEvent:
			sb.refreshData()

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
	channel := sb.channels[ev.Channel]
	scopedMessenger := sb.gm.scope(channel.Name)

	msg := &IncomingMessage{
		Text: ev.Text,

		channel:   ev.Channel,
		timestamp: ev.Timestamp,

		sb: sb,
	}

	for _, plugin := range sb.plugins {
		// TODO(nussey): much better error handling
		err := plugin.ParseMessage(msg, scopedMessenger)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (sb *SlackBot) fetchUsers() {
	sb.users = make(map[string]*slack.User)
	users := sb.rtm.GetInfo().Users

	for user := range users {
		sb.users[users[user].ID] = &users[user]
	}
}

func (sb *SlackBot) fetchChannels() {
	sb.channels = make(map[string]*slack.Channel)
	chans := sb.rtm.GetInfo().Channels

	for channel := range chans {
		sb.channels[chans[channel].ID] = &chans[channel]
	}
}

func (sb *SlackBot) fetchIMs() {
	sb.ims = make(map[string]*slack.User)
	ims := sb.rtm.GetInfo().IMs

	for im := range ims {
		sb.ims[ims[im].ID] = sb.users[ims[im].User]
	}
}

func randStringRunes(n int) string {
	// TODO(nussey): Move off of the global RNG
	b := make([]rune, n)
	for i := range b {
		b[i] = rngRunes[rand.Intn(len(rngRunes))]
	}
	return string(b)
}

func (sb *SlackBot) initDms() {
	for _, user := range sb.users {
		if _, ok := sb.gm.dms[user.Name]; !ok {
			sb.gm.dms[user.Name] = &directMessage{
				mutex: &sync.Mutex{},

				currentConvo: nil,
				convoQueue:   make(chan *conversation, convoQueueSize),
			}
			go sb.gm.dms[user.Name].manageDM()
		}
	}
}
