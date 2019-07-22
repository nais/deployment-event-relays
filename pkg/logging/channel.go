package logging

import (
	"fmt"
)

type chanLogger struct {
	C chan string
}

func NewChanLogger() *chanLogger {
	return &chanLogger{
		C: make(chan string, 1024),
	}
}

func (d *chanLogger) Print(v ...interface{}) {
	d.C <- fmt.Sprint(v...)
}

func (d *chanLogger) Printf(format string, v ...interface{}) {
	d.C <- fmt.Sprintf(format, v...)
}

func (d *chanLogger) Println(v ...interface{}) {
	d.C <- fmt.Sprintln(v...)
}
