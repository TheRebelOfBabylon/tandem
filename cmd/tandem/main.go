package main

import (
	"flag"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/logging"
	"github.com/TheRebelOfBabylon/tandem/signal"
)

var (
	cfgFilePath = flag.String("config", "tandem.toml", "path to the TOML config file")
)

func main() {
	// parse command line flags
	flag.Parse()

	// initialize logging
	logger := logging.NewLogger()

	// read and validate config
	cfg, err := config.ReadConfig(*cfgFilePath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to read config file")
	}
	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("failed to validate config")
	}

	// configure Logging
	logger, err = logger.SetLogLevel(cfg.Log.Level)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to set log level")
	}
	logger.Info().Msgf("using log level %s", logger.GetLevel().String())

	// initialize signal handler
	interruptHandler := signal.NewInterruptHandler(logger.With().Str("module", "interruptHandler").Logger())

	// initialize Connection to Database

	// Start/Initialize HTTP Server
	<-interruptHandler.ShutdownDoneChannel()
	logger.Info().Msg("shutdown complete")
}
