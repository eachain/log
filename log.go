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

// - - - - - - - - - logger - - - - - - - - -

type Logger interface {
	Log(t time.Time, level int, s []byte)
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

func formatHeader(buf *bytes.Buffer, t time.Time, file string, line int, lvl int) {
	if global.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if global.flag&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			buf.WriteByte('-')
			itoa(buf, int(month), 2)
			buf.WriteByte('-')
			itoa(buf, day, 2)
			buf.WriteByte(' ')
		}
		if global.flag&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			buf.WriteByte(':')
			itoa(buf, min, 2)
			buf.WriteByte(':')
			itoa(buf, sec, 2)
			if global.flag&Lmicroseconds != 0 {
				buf.WriteByte('.')
				itoa(buf, t.Nanosecond()/1e6, 3)
			}
			buf.WriteByte(' ')
		}
	}
	if global.flag&Llevel != 0 {
		buf.WriteString(levels[lvl])
		buf.WriteByte(' ')
	}
	if global.flag&Lmodule != 0 {
		buf.WriteByte('[')
		buf.WriteString(moduleOf(file))
		buf.WriteByte(']')
		buf.WriteByte(' ')
	}
	if global.flag&(Lshortfile|Llongfile) != 0 {
		if global.flag&Lshortfile != 0 {
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

func output(lvl int, calldepth int, s string) {
	now := time.Now() // get this early.
	var file string
	var line int
	if global.flag&(Lshortfile|Llongfile) != 0 {
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
	}
	global.mu.Lock()
	global.buf.Reset()
	formatHeader(&global.buf, now, file, line, lvl)
	global.buf.WriteString(s)
	if len(s) > 0 && s[len(s)-1] != '\n' {
		global.buf.WriteByte('\n')
	}
	global.out.Log(now, lvl, global.buf.Bytes())
	global.mu.Unlock()
}

func Debug(format string, v ...interface{}) {
	if Ldebug < global.level {
		return
	}
	output(Ldebug, global.calldepth, fmt.Sprintf(format, v...))
}

func Info(format string, v ...interface{}) {
	if Linfo < global.level {
		return
	}
	output(Linfo, global.calldepth, fmt.Sprintf(format, v...))
}

func Notice(format string, v ...interface{}) {
	if Lnotice < global.level {
		return
	}
	output(Lnotice, global.calldepth, fmt.Sprintf(format, v...))
}

func Warn(format string, v ...interface{}) {
	if Lwarn < global.level {
		return
	}
	output(Lwarn, global.calldepth, fmt.Sprintf(format, v...))
}

func Error(format string, v ...interface{}) {
	if Lerror < global.level {
		return
	}
	output(Lerror, global.calldepth, fmt.Sprintf(format, v...))
}

func Panic(format string, v ...interface{}) {
	if Lpanic < global.level {
		return
	}
	s := fmt.Sprintf(format, v...)
	output(Lpanic, global.calldepth, s)
	panic(s)
}

func Fatal(format string, v ...interface{}) {
	if Lfatal < global.level {
		return
	}
	output(Lfatal, global.calldepth, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// - - - - - - - - - std logger - - - - - - - - -

type writerLogger struct {
	w io.Writer
}

func (sl writerLogger) Log(t time.Time, level int, s []byte) {
	sl.w.Write(s)
}

func NewLogger(w io.Writer) Logger {
	return writerLogger{w: w}
}

// - - - - - - - - - setting - - - - - - - - -

type setting struct {
	mu        sync.Mutex   // just for buf
	buf       bytes.Buffer // for accumulating text to write
	flag      int
	level     int
	out       Logger
	calldepth int
}

var global *setting

func init() {
	global = &setting{
		flag:      LstdFlags,
		level:     Linfo,
		out:       NewLogger(os.Stderr),
		calldepth: 2,
	}
}

func Flags() int {
	return global.flag
}

func SetFlags(flag int) {
	global.flag = flag
}

func SetLevel(level int) {
	global.level = level
}

func SetOutput(logger Logger) {
	global.out = logger
}

func SetCallDepth(depth int) {
	global.calldepth = depth
}

func CallDepth() int {
	return global.calldepth
}
