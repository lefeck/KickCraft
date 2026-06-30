package main

import (
	"log"
	"os"

	"github.com/kickcraft/cmd"
	"github.com/kickcraft/logger"
)

func main() {
	if err := cmd.Execute(); err != nil {
		logger.Error("Application error: %v", err)
		os.Exit(1)
	}
}

func init() {
	// Initialize logger
	logger.Init()
	log.SetOutput(logger.GetWriter())
	log.SetFlags(0)
}
