package main

import (
	"github.com/TheRebelOfBabylon/tandem/logging"
	"github.com/TheRebelOfBabylon/tandem/signal"
)

func main() {
	// Initialize logging
	logger := logging.NewLogger()

	// Initialize signal handler
	interruptHandler := signal.NewInterruptHandler(logger.With().Str("module", "interruptHandler").Logger())

	// Read Config

	// Configure Logging

	// Initialize Connection to Database

	// Start/Initialize HTTP Server
	<-interruptHandler.ShutdownDoneChannel()
	logger.Info().Msg("shutdown complete")
}
