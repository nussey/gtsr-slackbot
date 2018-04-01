package helptext

import (
	"github.com/nussey/gtsr-slackbot/gtsr"
)

type HelpTextBot struct {
}

var networkDriveText = "*On Windows*: \n• Open a File Explorer window. \n• Right-click on ‘This PC’ and then select ‘Map Network Drive...’.  \n• Enter ‘\\\\mefile4.me.gatech.edu\\Research\\GTSR’ into the ‘Folder:’ field and then click ‘Finish’ (or hit Enter). \n• Enter your GT Prism ID as ‘AD\\<username>’ (e.g. ‘AD\\gburdell3’) and your password. \n\n*On OSX*:  \n• From the desktop, click ‘Go’ in the menu bar above all and then ‘Connect to Server’. \n• Enter ‘cifs://mefile4.me.gatech.edu/Research/GTSR’ into the ‘Server Address:’ field and then click ‘Connect’ (or hit Enter). \n• Enter your GT Prism ID (e.g. ‘gburdell3’) and your password."

// TODO(nussey) maybe link out to the wiki one day

func (ht *HelpTextBot) Init() *gtsr.PluginConfig {
	faq := &gtsr.ConvoTopic{
		Name:  "FAQ",
		Label: "Frequently Asked Questions",

		Action: ht.FAQ,
	}

	return &gtsr.PluginConfig{
		Name:        "Help Text",
		Description: "Let users get basic help information without bothering people",
		Version:     "1.0",

		FeatureConvo: true,
		Topics:       []*gtsr.ConvoTopic{faq},

		FeatureChron: false,
		Jobs:         []*gtsr.CronJob{},
	}

}

func (ht *HelpTextBot) Teardown() {

}

func (ht *HelpTextBot) ParseMessage(msg string, messenger *gtsr.Messenger) error {
	if Match_NetworkDrive(msg) {
		// TODO(nussey): send an etherial message first asking if they are curious
		return messenger.NewMessage(networkDriveText).Send()
	}

	return nil
}

func Match_NetworkDrive(msg string) bool {
	// TODO(nussey): actually scan the words and see if they were asking about the network drive
	return true
}

func (ht *HelpTextBot) FAQ(usr gtsr.User) error {

	return nil
}
