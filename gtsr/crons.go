package gtsr

import (
	"github.com/robfig/cron"
)

// A CronJob representes a specific task to be executed at a given interval.
// These jobs need not be bound to the recieving of messages or DMs. Use cases
// may include a Calendar integration, attendance trigger, etc.
type CronJob struct {
	// ID of the job
	ID string

	// Human friendly name of the job
	Name string
	// Spec defining when to take action
	// +---------------- minute (0 - 59)
	// |  +------------- hour (0 - 23)
	// |  |  +---------- day of month (1 - 31)
	// |  |  |  +------- month (1 - 12)
	// |  |  |  |  +---- day of week (0 - 7) (Sunday=0 or 7)
	// |  |  |  |  |
	// *  *  *  *  *  command to be executed
	Spec string

	// Action to be performed every Interval amount of time
	// All cron actions must be fully threadsafe
	Action func(*GlobalMessenger) error
}

func (sb *SlackBot) initCron() {
	c := cron.New()

	for _, job := range sb.crons {
		c.AddFunc(job.Spec, func() {
			job.Action(sb.gm)
		})
	}

	c.Start()

	sb.scheduler = c
}
