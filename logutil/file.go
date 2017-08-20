package logutil

import (
	"os"
	"path/filepath"
	"time"
)

type FileWriter struct {
	filename string
	fp       *os.File
}

func (fw *FileWriter) WriteLog(t time.Time, level int, s []byte) {
	fw.fp.Write(s)
}

func (fw *FileWriter) Filename() string {
	return fw.filename
}

func (fw *FileWriter) SwapFile(new *os.File) *os.File {
	fw.fp, new = new, fw.fp
	return new
}

func NewFileWriter(filename string) *FileWriter {
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, 755)
	if err != nil {
		panic(err)
	}

	fp, err := os.OpenFile(filename,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644)
	if err != nil {
		panic(err)
	}

	return &FileWriter{filename: filename, fp: fp}
}
