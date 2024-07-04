package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/EnsurityTechnologies/logger"
)

func main() {
	fp, err := os.OpenFile("log.txt",
		os.O_APPEND|os.O_CREATE, 0644)

	if err != nil {
		panic(err)
	}
	l := logger.New(&logger.LoggerOptions{
		Level:       logger.Debug,
		Color:       []logger.ColorOption{logger.AutoColor, logger.ColorOff},
		Output:      []io.Writer{logger.DefaultOutput, fp},
		KeepNumDays: 1,
	})
	l.Debug("Test")
	l.Info("Test")

	t := time.Now()
	numDays := 1
	then := t.AddDate(0, 0, (numDays+1)*(-1))
	fmt.Printf("Time now : %v\n", t)
	fmt.Printf("Time then : %v\n", then)
}
