// +build !windows

package logutil

import (
	"time"

	"github.com/eachain/log"
)

const colorEnd = "\033[0m"

var logColor = []string{
	log.Ldebug:  "\033[37m",
	log.Linfo:   "\033[32m",
	log.Lnotice: "\033[33m",
	log.Lwarn:   "\033[35m",
	log.Lerror:  "\033[31m",
	log.Lpanic:  "\033[1;31m",
	log.Lfatal:  "\033[1;31m",
}

type colorLogger struct {
	b []byte
	l log.Logger
}

func (cl *colorLogger) Log(t time.Time, level int, s string) {
	cl.b = cl.b[:0]
	cl.b = append(cl.b, logColor[level]...)
	cl.b = append(cl.b, s...)
	cl.b = append(cl.b, colorEnd...)
	cl.l.Log(t, level, string(cl.b))
}

func WithColor(logger log.Logger) log.Logger {
	return &colorLogger{l: logger}
}
