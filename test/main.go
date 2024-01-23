package main

import (
	"io"
	"os"

	"github.com/EnsurityTechnologies/logger"
)

func main() {
	fp, err := os.OpenFile("log.txt",
		os.O_APPEND|os.O_CREATE, 0644)
		
		
	if err != nil {
		panic(err)
	}
	l := logger.New(&logger.LoggerOptions{
		Level:  logger.Debug,
		Color:  []logger.ColorOption{logger.AutoColor, logger.ColorOff},
		Output: []io.Writer{logger.DefaultOutput, fp},
	})
	l.Debug("Test")
	l.Info("Test")
}
