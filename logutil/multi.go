package logutil

import (
	"time"

	"github.com/eachain/log"
)

type multiWriter struct {
	ws []log.Writer
}

func (mw *multiWriter) WriteLog(t time.Time, level int, s []byte) {
	for _, w := range mw.ws {
		w.WriteLog(t, level, s)
	}
}

func MultiWriter(wr ...log.Writer) log.Writer {
	return &multiWriter{ws: wr}
}
