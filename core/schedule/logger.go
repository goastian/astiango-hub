package schedule

import (
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/robfig/cron/v3"
)

type CronLogger struct {
	interfaces.Logger
}

func (l *CronLogger) Info(msg string, keysAndValues ...interface{}) {
	l.Infof("%s %v", msg, keysAndValues)
}

func (l *CronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.Errorf("%s %v %v", msg, err, keysAndValues)
}

func NewCronLogger() cron.Logger {
	return &CronLogger{
		Logger: utils.NewLogger("Cron"),
	}
}
