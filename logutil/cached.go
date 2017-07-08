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
	msg   string
}

type cachedLogger struct {
	logger log.Logger
	ch     chan *record
	pool   sync.Pool
}

func (cl *cachedLogger) Log(t time.Time, level int, s string) {
	rd := cl.pool.Get().(*record)
	rd.t = t
	rd.level = level
	rd.msg = s
	select {
	case cl.ch <- rd:
	default:
		glog.Printf("cached logger: miss log, level: %v, message: %v", level, s)
	}
}

func WithCache(logger log.Logger, cacheSize int) log.Logger {
	cl := &cachedLogger{
		logger: logger,
		ch:     make(chan *record, cacheSize),
		pool: sync.Pool{
			New: func() interface{} {
				return &record{}
			},
		},
	}
	go func() {
		for rd := range cl.ch {
			cl.logger.Log(rd.t, rd.level, rd.msg)
			cl.pool.Put(rd)
		}
	}()
	return cl
}
