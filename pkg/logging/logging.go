package logging

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

type standardLogger struct {
	logger *log.Logger
	level  log.Level
}

func (d *standardLogger) Print(v ...interface{}) {
	d.logger.Log(d.level, v...)
}

func (d *standardLogger) Printf(format string, v ...interface{}) {
	d.logger.Logf(d.level, format, v...)
}

func (d *standardLogger) Println(v ...interface{}) {
	d.logger.Logln(d.level, v...)
}

/*
var _ log.StdLogger = &standardLogger{}

const FATAL = 234

func (d *standardLogger) Fatal(v ...interface{}) {
	d.logger.Log(d.level, v...)
	os.Exit(FATAL)
}

func (d *standardLogger) Fatalf(format string, v ...interface{}) {
	d.logger.Logf(d.level, format, v...)
	os.Exit(FATAL)
}

func (d *standardLogger) Fatalln(v ...interface{}) {
	d.logger.Logln(d.level, v...)
	os.Exit(FATAL)
}

func (d *standardLogger) Panic(v ...interface{}) {
	d.logger.Log(d.level, v...)
	panic(fmt.Sprint(v...))
}

func (d *standardLogger) Panicf(format string, v ...interface{}) {
	d.logger.Logf(d.level, format, v...)
	panic(fmt.Sprint(v...))
}

func (d *standardLogger) Panicln(v ...interface{}) {
	d.logger.Logln(d.level, v...)
	panic(fmt.Sprint(v...))
}
*/

func ConstLevel(level, format string) (*standardLogger, error) {
	l := &standardLogger{}
	l.logger = log.New()

	err := Apply(l.logger, level, format)
	if err != nil {
		return nil, err
	}

	l.level = l.logger.GetLevel()

	return l, nil
}

func Apply(logger *log.Logger, level, format string) error {
	var err error
	var formatter log.Formatter

	switch format {
	case "json":
		formatter = jsonFormatter()
	case "text":
		formatter = textFormatter()
	default:
		return fmt.Errorf("log format '%s' is not recognized", format)
	}

	lv, err := log.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("while setting log level: %s", err)
	}

	logger.SetFormatter(formatter)
	logger.SetLevel(lv)

	return nil
}

func textFormatter() log.Formatter {
	return &log.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
	}
}

func jsonFormatter() log.Formatter {
	return &log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
}
