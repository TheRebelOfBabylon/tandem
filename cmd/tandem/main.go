package main

import (
	"flag"
	"strings"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/ingester"
	"github.com/TheRebelOfBabylon/tandem/logging"
	"github.com/TheRebelOfBabylon/tandem/signal"
	"github.com/TheRebelOfBabylon/tandem/storage"
	"github.com/TheRebelOfBabylon/tandem/websocket"
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
	logger.Info().Msgf("reading configuration file %s...", *cfgFilePath)
	cfg, err := config.ReadConfig(*cfgFilePath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to read config file")
	}
	logger.Info().Msg("validating configuration file...")
	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("failed to validate config")
	}

	// configure Logging
	logger, err = logger.Configure(cfg.Log)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to set log level")
	}
	logger.Info().Msgf("using log level %s", strings.ToUpper(logger.GetLevel().String()))

	// initialize signal handler
	interruptHandler := signal.NewInterruptHandler(logger.With().Str("module", "interruptHandler").Logger())

	// initialize connection to storage backend
	logger.Info().Msg("initializing connection to storage backend...")
	strorageBackend, err := storage.Connect(cfg.Storage, logger.With().Str("module", "storageBackend").Logger())
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to storage backend")
	}
	defer strorageBackend.Close()

	// initialize ingester
	ingest := ingester.NewIngester(logger.With().Str("module", "ingester").Logger())
	defer ingest.Close()

	// initialize websocket handler
	wsHandler := websocket.NewMainHandler(logger.With().Str("module", "websocketHandler").Logger(), ingest.SendChannel())
	defer wsHandler.Close()

	// ingester and websocket handler now communicating bi-directionally
	ingest.SetRecvChannel(wsHandler.SendChannel())
	// TODO - Start HTTP Server
	<-interruptHandler.ShutdownDoneChannel()
	logger.Info().Msg("shutdown complete")
}
