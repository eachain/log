// +build !windows

package logutil

import (
	"sync"
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

type colorWriter struct {
	m sync.Mutex
	b []byte
	w log.Writer
}

func (cw *colorWriter) WriteLog(t time.Time, level int, s []byte) {
	cw.m.Lock()
	cw.b = cw.b[:0]
	cw.b = append(cw.b, logColor[level]...)
	cw.b = append(cw.b, s...)
	cw.b = append(cw.b, colorEnd...)
	cw.w.WriteLog(t, level, cw.b)
	cw.m.Unlock()
}

func WithColor(w log.Writer) log.Writer {
	return &colorWriter{w: w}
}
