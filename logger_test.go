package logger

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	fp, err := os.OpenFile("log.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	l := New(&LoggerOptions{
		Level:  DebugLevel,
		Color:  []ColorOption{AutoColor, ColorOff},
		Output: []io.Writer{DefaultOutput, fp},
	})
	l.Debug("Test")
	l.Info("Test")
}

func TestDefaultLog(t *testing.T) {

	l := NewDefaultLog(nil, "test", DebugLevel, "./", 1)
	for {
		l.Info("Test message")
		time.Sleep(1 * time.Minute)
	}

}

func TestEventLog(t *testing.T) {
	OnError(nil).Error("Error not occured")
	OnError(nil).Fatal("Error not occured")
	OnError(fmt.Errorf("testing error")).Error("Error occured")
	WithFields("test", 10).Info("testing with fileds")
}
