package log_test

import (
	"bytes"
	"fmt"

	"github.com/eachain/log"
)

func ExampleLogger() {
	log.SetFlags(log.Lshortfile)
	var buf bytes.Buffer
	log.SetOutput(log.NewLogger(&buf))
	log.Info("Hello world")
	fmt.Println(&buf)
	// prints: example_test.go:14: [INFO] Hello world
}
