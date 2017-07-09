package logutil

import (
	glog "log"
	"sync"
	"time"

	"github.com/eachain/log"
)

type record struct {
	t     time.Time
	level int
	msg   []byte
}

type cachedWriter struct {
	wr   log.Writer
	ch   chan *record
	pool sync.Pool
}

func (cw *cachedWriter) WriteLog(t time.Time, level int, s []byte) {
	rd := cw.pool.Get().(*record)
	rd.t = t
	rd.level = level
	rd.msg = append(rd.msg[:0], s...)
	select {
	case cw.ch <- rd:
	default:
		glog.Printf("cached logger: miss log, level: %v, message: %v", level, s)
	}
}

func WithCache(w log.Writer, cacheSize int) log.Writer {
	cw := &cachedWriter{
		wr: w,
		ch: make(chan *record, cacheSize),
		pool: sync.Pool{
			New: func() interface{} {
				return &record{}
			},
		},
	}
	go func() {
		for rd := range cw.ch {
			cw.wr.WriteLog(rd.t, rd.level, rd.msg)
			cw.pool.Put(rd)
		}
	}()
	return cw
}
