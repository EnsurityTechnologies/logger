package main

import "github.com/EnsurityTechnologies/logger"

func main() {
	l := logger.New(&logger.LoggerOptions{
		Color: logger.AutoColor,
	})

	l.Debug("Test")
}
