package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Setup configures global log level and output
func Setup(level string, disable bool) {
	if disable {
		log.SetOutput(io.Discard)
		return
	} else {
		f, _ := os.Create("/tmp/crypto-tracker.log")
		log.SetOutput(f)
	}

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

func Log() *logrus.Logger {
	return log
}
