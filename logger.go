package logger

import (
	"io"
	"os"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	log *logrus.Logger
}

// Setup configures the logger based on options in the config.json.
func Setup(logFile string) (*Logger, error) {
	logger := Logger {
		log : logrus.New(),
	}
	//logger.log.Formatter = &logrus.TextFormatter{DisableColors: true}
	//.log.SetLevel(logrus.InfoLevel)
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}
		mw := io.MultiWriter(os.Stderr, f)
		logger.log.Out = mw
	}
	return &logger, nil
}

func (logger *Logger)Debug(args ...interface{}) {
	logger.log.Debug(args...)
}

func (logger *Logger)Debugf(format string, args ...interface{}) {
	logger.log.Debugf(format, args...)
}

func (logger *Logger)Info(args ...interface{}) {
	logger.log.Info(args...)
}

func (logger *Logger)Infof(format string, args ...interface{}) {
	logger.log.Infof(format, args...)
}

func (logger *Logger)Error(args ...interface{}) {
	logger.log.Error(args...)
}

func (logger *Logger)Errorf(format string, args ...interface{}) {
	logger.log.Errorf(format, args...)
}

func (logger *Logger)Warn(args ...interface{}) {
	logger.log.Warn(args...)
}

func (logger *Logger)Warnf(format string, args ...interface{}) {
	logger.log.Warnf(format, args...)
}

func (logger *Logger)Fatal(args ...interface{}) {
	logger.log.Fatal(args...)
}

func (logger *Logger)Fatalf(format string, args ...interface{}) {
	logger.log.Fatalf(format, args...)
}

func (logger *Logger)WithFields(fields logrus.Fields) *logrus.Entry {
	return logger.log.WithFields(fields)
}

func (logger *Logger)Writer() *io.PipeWriter {
	return logger.log.Writer()
}