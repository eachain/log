# log 一个基于Golang的日志模块

## 说明

**1. 该日志模块为标准库log的一个扩展实现。**

**2. 全局使用同一份日志模块(与其他实现的区别)。**

*注: 多数项目中，为方便排查问题，都会全局使用同一个日志句柄，所以在本模块内直接做成单例，方便使用。*

## 基本功能

有`Debug`、`Info`、`Notice`、`Warn`、`Error`、`Panic`、`Fatal`共7个级别的日志。

## 使用

### 1. 输出到控制台

```go
package main

import "github.com/eachain/log"

func main() {
	log.SetFlags(log.Lshortfile)
	log.Info("Hello eachain log!")
	// prints:
	// test.go:7: Hello eachain log!
}
```

### 2. 写到文件

```go
package main

import (
	"os"

	"github.com/eachain/log"
)

func main() {
	file, err := os.OpenFile("eachain.log",
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644)
	if err != nil {
		log.Fatal("open log file error: %v", err)
	}
	defer file.Close()

	log.SetOutput(log.NewLogger(file))
	log.Info("Hello eachain log!")
}
```

**可见，和标准库的用法几乎一模一样。同时写到文件和控制台:**

```go
log.SetOutput(log.NewLogger(io.MultiWriter(os.Stdout, file)))
```

**所以，你应该自己实现具体的写文件操作。**

## 高级用法

### 1. 带颜色输出(暂不支持Windows系统)

```go
package main

import (
	"os"

	"github.com/eachain/log"
	"github.com/eachain/log/logutil"
)

func main() {
	log.SetOutput(logutil.WithColor(log.NewLogger(os.Stdout)))
	log.Info("Hello eachain log!")
}
```

**现在，有些奇怪想法来了：同时输出到控制台和文件，控制台带颜色(醒目)，文件不带颜色(方便排查历史问题)**

```go
package main

import (
	"os"

	"github.com/eachain/log"
	"github.com/eachain/log/logutil"
)

func main() {
	file, _ := os.OpenFile("eachain.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer file.Close()

	log.SetOutput(logutil.MultiLogger(
		logutil.WithColor(log.NewLogger(os.Stdout)),
		log.NewLogger(file),
	))
	log.Info("Hello eachain log!")
}
```

### 2. Logger接口

```go
type Logger interface {
    Log(t time.Time, level int, s string)
}
```
只要实现了Logger接口，就可以无限嵌套，形成一条链，像io.Reader，像http.Handler，都是可以不断地在接口上加功能，形成强大的功能。

比如，你可以根据文件大小，依次生成新文件；也可以根据日期生成新文件；还可以将不同level的日志打到不同文件中去。

将上面的功能组合，可得：将日志带颜色打印到控制台；将日志带缓存打印到文件；不同level的日志打到不同的文件中；每天生成新文件。

见下面组合式应用：

```go
package main

import (
	glog "log"
	"os"
	"path/filepath"
	"time"

	"github.com/eachain/log"
	"github.com/eachain/log/logutil"
)

// - - - - - - - - - - - - - - - - - - - -

type fileLogger struct {
	prefix string
	date   string
	file   *os.File
}

func (fl *fileLogger) Log(t time.Time, level int, s string) {
	newDate := t.Format("060102")
	if fl.date != newDate && fl.file != nil {
		fl.file.Close()
		fl.file = nil
	}
	if fl.file == nil {
		err := os.MkdirAll(filepath.Dir(fl.prefix), 0775)
		if err != nil {
			glog.Printf("file logger: makedir error: %v", err)
			return
		}

		file, err := os.OpenFile(fl.prefix+"-"+newDate+".log",
			os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
		if err != nil {
			glog.Printf("file logger: create log file error: %v", err)
			return
		}
		fl.file = file
		fl.date = newDate
	}

	fl.file.WriteString(s)
}

func newFileLogger(prefix string) log.Logger {
	return &fileLogger{prefix: prefix}
}

// - - - - - - - - - - - - - - - - - - - -

type levelLogger struct {
	info log.Logger
	err  log.Logger
}

func (ll *levelLogger) Log(t time.Time, level int, s string) {
	ll.info.Log(t, level, s)
	if level >= log.Lerror {
		ll.err.Log(t, level, s)
	}
}

func newLevelLogger(info, err log.Logger) log.Logger {
	return &levelLogger{info: info, err: err}
}

// - - - - - - - - - - - - - - - - - - - -

func main() {
	log.SetFlags(log.Flags() | log.Lmodule | log.Lshortfile)
	log.SetOutput(logutil.MultiLogger(
		logutil.WithColor(log.NewLogger(os.Stdout)),
		logutil.WithCache(newLevelLogger(
			newFileLogger("eachain-info"),
			newFileLogger("eachain-err"),
		), 1024),
	))

	log.Debug("debug log")
	log.Info("info log")
	log.Notice("notice log")
	log.Warn("warn log")
	log.Error("error log")
}
```

综上，按自身需求实现对日志的操作，你可以做到大多数日志模块能完成的事情。


