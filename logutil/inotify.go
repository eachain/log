// +build linux

package logutil

import (
	glog "log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/eachain/log"
)

type FileSwitcher interface {
	log.Writer

	// Filename returns the file abs path
	Filename() string
	// SwapFile returns the old file before
	SwapFile(new *os.File) (old *os.File)
}

// 特殊服务, 仅支持linux系统
type inotifyWriter struct {
	fs  FileSwitcher
	buf []byte
}

func (iw *inotifyWriter) inotify(cb func()) <-chan struct{} {
	const events = syscall.IN_MOVE | syscall.IN_MOVE_SELF |
		syscall.IN_DELETE | syscall.IN_DELETE_SELF

	fd, err := syscall.InotifyInit()
	if err != nil {
		glog.Printf("inotify init error: %v", err)
		return nil
	}
	wd, err := syscall.InotifyAddWatch(fd, iw.fs.Filename(), events)
	if err != nil {
		glog.Printf("inotify add watch error: %v", err)
		syscall.Close(fd)
		return nil
	}

	done := make(chan struct{})
	go func() {
		syscall.Read(fd, iw.buf)
		cb()
		syscall.InotifyRmWatch(fd, uint32(wd))
		syscall.Close(fd)
		close(done)
	}()
	return done
}

func (iw *inotifyWriter) inotifyCb() {
	fp, err := os.OpenFile(iw.fs.Filename(),
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644)
	if err != nil {
		glog.Printf("open log file error: %v", err)
		return
	}

	old := iw.fs.SwapFile(fp)
	syscall.Dup2(int(fp.Fd()), int(old.Fd()))
	time.Sleep(time.Second)
	old.Close()
}

func (iw *inotifyWriter) run() {
	for {
		done := iw.inotify(iw.inotifyCb)
		if done != nil {
			<-done
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (iw *inotifyWriter) WriteLog(t time.Time, level int, s []byte) {
	iw.fs.WriteLog(t, level, s)
}

func WithInotify(fs FileSwitcher) log.Writer {
	dir := filepath.Dir(fs.Filename())
	err := os.MkdirAll(dir, 755)
	if err != nil {
		panic(err)
	}

	fp, err := os.OpenFile(fs.Filename(),
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644)
	if err != nil {
		panic(err)
	}

	old := fs.SwapFile(fp)
	if old != nil {
		old.Close()
	}

	iw := &inotifyWriter{
		fs:  fs,
		buf: make([]byte, 1024),
	}
	go iw.run()
	return iw
}
