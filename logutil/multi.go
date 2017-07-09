package logutil

import (
	"time"

	"github.com/eachain/log"
)

type multiLogger struct {
	loggers []log.Logger
}

func (ml *multiLogger) Log(t time.Time, level int, s []byte) {
	for _, l := range ml.loggers {
		l.Log(t, level, s)
	}
}

func MultiLogger(logger ...log.Logger) log.Logger {
	return &multiLogger{loggers: logger}
}
