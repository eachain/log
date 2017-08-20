package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// - - - - - - - - - flags - - - - - - - - -

const (
	Ldate         = 1 << iota // the date in the local time zone: 2006-01-02
	Ltime                     // the time in the local time zone: 15:04:05
	Lmicroseconds             // microsecond resolution: 01:23:23.123.  assumes Ltime.
	Llongfile                 // full file name and line number: /a/b/c/d.go:23
	Lshortfile                // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                      // if Ldate or Ltime is set, use UTC rather than the local time zone
	Lmodule                   // module name
	Llevel                    // the level of the log

	LstdFlags = Ldate | Ltime | Lmicroseconds | Llevel // initial values for the standard logger
)

// - - - - - - - - - levels - - - - - - - - -

const (
	Ldebug = iota
	Linfo
	Lnotice
	Lwarn
	Lerror
	Lpanic
	Lfatal
)

var levels = []string{
	"[DEBUG]",
	"[INFO]",
	"[NOTICE]",
	"[WARN]",
	"[ERROR]",
	"[PANIC]",
	"[FATAL]",
}

type Writer interface {
	WriteLog(t time.Time, level int, s []byte)
}

type writer struct {
	w io.Writer
}

func (wr writer) WriteLog(t time.Time, level int, s []byte) {
	wr.w.Write(s)
}

// NewWriter returns a Writer.
// It is w's responsibility to write currently safe.
func NewWriter(w io.Writer) Writer {
	return writer{w: w}
}

// - - - - - - - - - logger - - - - - - - - -

type Logger struct {
	pool      *sync.Pool // a buf pool
	flag      int
	level     int
	out       Writer
	calldepth int
}

func NewLogger(w Writer, flag int, level int) *Logger {
	return &Logger{
		pool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(nil)
			},
		},
		flag:      flag,
		level:     level,
		out:       w,
		calldepth: 2,
	}
}

func (l *Logger) Flags() int {
	return l.flag
}

func (l *Logger) SetFlags(flag int) {
	l.flag = flag
}

func (l *Logger) SetLevel(level int) {
	l.level = level
}

func (l *Logger) SetOutput(w Writer) {
	l.out = w
}

func (l *Logger) SetCallDepth(depth int) {
	l.calldepth = depth
}

func (l *Logger) CallDepth() int {
	return l.calldepth
}

func itoa(buf *bytes.Buffer, i int, wid int) {
	var u uint = uint(i)
	if u == 0 && wid <= 1 {
		buf.WriteByte('0')
		return
	}

	// Assemble decimal in reverse order.
	var b [32]byte
	bp := len(b)
	for ; u > 0 || wid > 0; u /= 10 {
		bp--
		wid--
		b[bp] = byte(u%10) + '0'
	}

	// avoid slicing b to avoid an allocation.
	for bp < len(b) {
		buf.WriteByte(b[bp])
		bp++
	}
}

func moduleOf(file string) string {
	pos := strings.LastIndex(file, "/")
	if pos != -1 {
		pos1 := strings.LastIndex(file[:pos], "/src/")
		if pos1 != -1 {
			return file[pos1+5 : pos]
		}
	}
	return "UNKNOWN"
}

func (l *Logger) formatHeader(buf *bytes.Buffer, t time.Time, file string, line int, lvl int) {
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if l.flag&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			buf.WriteByte('-')
			itoa(buf, int(month), 2)
			buf.WriteByte('-')
			itoa(buf, day, 2)
			buf.WriteByte(' ')
		}
		if l.flag&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			buf.WriteByte(':')
			itoa(buf, min, 2)
			buf.WriteByte(':')
			itoa(buf, sec, 2)
			if l.flag&Lmicroseconds != 0 {
				buf.WriteByte('.')
				itoa(buf, t.Nanosecond()/1e6, 3)
			}
			buf.WriteByte(' ')
		}
	}
	if l.flag&Llevel != 0 {
		buf.WriteString(levels[lvl])
		buf.WriteByte(' ')
	}
	if l.flag&Lmodule != 0 {
		buf.WriteByte('[')
		buf.WriteString(moduleOf(file))
		buf.WriteByte(']')
		buf.WriteByte(' ')
	}
	if l.flag&(Lshortfile|Llongfile) != 0 {
		if l.flag&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		buf.WriteString(file)
		buf.WriteByte(':')
		itoa(buf, line, -1)
		buf.WriteString(": ")
	}
}

func (l *Logger) output(lvl int, s string) {
	now := time.Now() // get this early.
	var file string
	var line int
	if l.flag&(Lshortfile|Llongfile) != 0 {
		var ok bool
		_, file, line, ok = runtime.Caller(l.calldepth)
		if !ok {
			file = "???"
			line = 0
		}
	}
	buf := l.pool.Get().(*bytes.Buffer)
	buf.Reset()
	l.formatHeader(buf, now, file, line, lvl)
	buf.WriteString(s)
	if len(s) > 0 && s[len(s)-1] != '\n' {
		buf.WriteByte('\n')
	}
	l.out.WriteLog(now, lvl, buf.Bytes())
	l.pool.Put(buf)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if Ldebug < l.level {
		return
	}
	l.output(Ldebug, fmt.Sprintf(format, v...))
}

func (l *Logger) Info(format string, v ...interface{}) {
	if Linfo < l.level {
		return
	}
	l.output(Linfo, fmt.Sprintf(format, v...))
}

func (l *Logger) Notice(format string, v ...interface{}) {
	if Lnotice < l.level {
		return
	}
	l.output(Lnotice, fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if Lwarn < l.level {
		return
	}
	l.output(Lwarn, fmt.Sprintf(format, v...))
}

func (l *Logger) Error(format string, v ...interface{}) {
	if Lerror < l.level {
		return
	}
	l.output(Lerror, fmt.Sprintf(format, v...))
}

func (l *Logger) Panic(format string, v ...interface{}) {
	if Lpanic < l.level {
		return
	}
	s := fmt.Sprintf(format, v...)
	l.output(Lpanic, s)
	panic(s)
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	if Lfatal < l.level {
		return
	}
	l.output(Lfatal, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// - - - - - - - - - std logger - - - - - - - - -

type mutexWriter struct {
	m sync.Mutex
	w io.Writer
}

func (mw *mutexWriter) WriteLog(t time.Time, level int, s []byte) {
	mw.m.Lock()
	mw.w.Write(s)
	mw.m.Unlock()
}

// NewMutexWriter returns a currently safe writer.
func NewMutexWriter(w io.Writer) Writer {
	return writer{w: w}
}

var std *Logger

func init() {
	std = NewLogger(NewMutexWriter(os.Stderr), LstdFlags, Linfo)
	std.SetCallDepth(std.CallDepth() + 1)
}

func Flags() int {
	return std.Flags()
}

func SetFlags(flag int) {
	std.SetFlags(flag)
}

func SetLevel(level int) {
	std.SetLevel(level)
}

func SetOutput(w Writer) {
	std.SetOutput(w)
}

func SetCallDepth(depth int) {
	std.SetCallDepth(depth)
}

func CallDepth() int {
	return std.CallDepth()
}

func Debug(format string, v ...interface{}) {
	std.Debug(format, v...)
}

func Info(format string, v ...interface{}) {
	std.Info(format, v...)
}

func Notice(format string, v ...interface{}) {
	std.Notice(format, v...)
}

func Warn(format string, v ...interface{}) {
	std.Warn(format, v...)
}

func Error(format string, v ...interface{}) {
	std.Error(format, v...)
}

func Panic(format string, v ...interface{}) {
	std.Panic(format, v...)
}

func Fatal(format string, v ...interface{}) {
	std.Fatal(format, v...)
}
