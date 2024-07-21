package main

import (
	"log"

	"github.com/TheRebelOfBabylon/tandem/signal"
)

func main() {
	// Initialize SIGKILL handler
	interruptHandler := signal.NewInterruptHandler()

	// Read Config

	// Setup Logging

	// Initialize Connection to Database

	// Start/Initialize HTTP Server
	<-interruptHandler.ShutdownDoneChannel()
	log.Println("shutdown complete")
}
