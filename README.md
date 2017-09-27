# log 一个基于Golang的日志模块

## 说明

**该日志模块为标准库log的一个扩展实现。**

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

	log.SetOutput(log.NewWriter(file))
	log.Info("Hello eachain log!")
}
```

**可见，和标准库的用法几乎一模一样。同时写到文件和控制台:**

```go
log.SetOutput(log.NewWriter(io.MultiWriter(os.Stdout, file)))
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
	log.SetOutput(logutil.WithColor(log.NewWriter(os.Stdout)))
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
	file, _ := os.OpenFile("eachain.log",
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer file.Close()

	log.SetOutput(logutil.MultiWriter(
		logutil.WithColor(log.NewWriter(os.Stdout)),
		log.NewWriter(file),
	))
	log.Info("Hello eachain log!")
}
```

### 2. Writer接口

```go
type Writer interface {
    WriteLog(t time.Time, level int, s []byte)
}
```

**(注意: 如果要异步处理日志，请先复制一份`s []byte`再异步，见`https://github.com/eachain/log/blob/master/logutil/cached.go`，因为日志用的公用缓存，不同步处理，会被后面的日志覆盖式修改。)**

**(注意: WriteLog必须自己保证线程安全，如log.NewMutexWriter)**

只要实现了Writer接口，就可以无限嵌套，形成一条链，像io.Reader，像http.Handler，都是可以不断地在接口上加功能，形成强大的功能。

比如，你可以根据文件大小，依次生成新文件；也可以根据日期生成新文件；还可以将不同level的日志打到不同文件中去。

将上面的功能组合，可得：日志带颜色带缓存打印到控制台，同时打印到文件，不同level的日志打到不同的文件中，每天生成新文件。

见下面组合式应用：

```go
package main

import (
	glog "log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/eachain/log"
	"github.com/eachain/log/logutil"
)

// - - - - - - - - - - - - - - - - - - - -

type fileWriter struct {
	sync.Mutex
	prefix string
	date   string
	file   *os.File
}

func (fw *fileWriter) changeFile(t time.Time) {
	newDate := t.Format("060102")
	if fw.date != newDate && fw.file != nil {
		fw.file.Close()
		fw.file = nil
	}
	if fw.file == nil {
		err := os.MkdirAll(filepath.Dir(fw.prefix), 0775)
		if err != nil {
			glog.Printf("file logger: makedir error: %v", err)
			return
		}

		file, err := os.OpenFile(fw.prefix+"-"+newDate+".log",
			os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
		if err != nil {
			glog.Printf("file logger: create log file error: %v", err)
			return
		}
		fw.file = file
		fw.date = newDate
	}
}

func (fw *fileWriter) WriteLog(t time.Time, level int, s []byte) {
	fw.Lock()
	fw.changeFile(t)
	fw.file.Write(s)
	fw.Unlock()
}

func newFileWriter(prefix string) log.Writer {
	return &fileWriter{prefix: prefix}
}

// - - - - - - - - - - - - - - - - - - - -

type levelWriter struct {
	info log.Writer
	err  log.Writer
}

func (lw *levelWriter) WriteLog(t time.Time, level int, s []byte) {
	lw.info.WriteLog(t, level, s)
	if level >= log.Lerror {
		lw.err.WriteLog(t, level, s)
	}
}

func newLevelWriter(info, err log.Writer) log.Writer {
	return &levelWriter{info: info, err: err}
}

// - - - - - - - - - - - - - - - - - - - -

func main() {
	log.SetFlags(log.Flags() | log.Lmodule | log.Lshortfile)
	log.SetOutput(logutil.MultiWriter(
		logutil.WithColor(logutil.WithCache(log.NewWriter(os.Stdout), 1024)),
		newLevelWriter(
			newFileWriter("eachain-info"),
			newFileWriter("eachain-err"),
		),
	))

	log.Debug("debug log")
	log.Info("info log")
	log.Notice("notice log")
	log.Warn("warn log")
	log.Error("error log")
	time.Sleep(time.Second)
}
```

综上，按自身需求实现对日志的操作，你可以做到大多数日志模块能完成的事情。

-----

## 特殊服务

### 1. 目前仅支持linux环境(大多服务器是linux的嘛)

```go
package main

import (
	"time"

	"github.com/eachain/log"
	"github.com/eachain/log/logutil"
)

func main() {
	log.SetOutput(logutil.WithInotify(logutil.NewFileWriter("eachain.log")))

	for i := 1; i <= 60; i++ {
		log.Info("line %v", i)
		time.Sleep(time.Second)
	}
}
```

`WithInotify`可以监视文件的移动删除事件，如果输出日志的文件被移动或删除，它会马上新建一个文件，把输出转向新文件。

对，它是面向那些用logrotate转储文件用的。

logrotate的文件转储方式中，copytruncate是肯定会丢日志的，所以不应该用它; compress方式是先mv，再gzip压缩，它是有可能丢日志的，但机率极其低; 用其它方式转储的是不会丢的。

用Inotify方式还有个限制，就是日志输出文件不变的情况下，如果*os.File会变，比如上面按天打印到不同日志文件中，是不适合用Inotify的，或者你自己按需自行实现了。
