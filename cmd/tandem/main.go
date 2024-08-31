package main

import (
	"flag"
	"slices"
	"strings"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/filter"
	"github.com/TheRebelOfBabylon/tandem/ingester"
	"github.com/TheRebelOfBabylon/tandem/logging"
	"github.com/TheRebelOfBabylon/tandem/signal"
	"github.com/TheRebelOfBabylon/tandem/storage"
	"github.com/TheRebelOfBabylon/tandem/websocket"
)

type Module interface {
	Start() error
	Stop() error
}

var (
	cfgFilePath = flag.String("config", "tandem.toml", "path to the TOML config file")
	modules     = []Module{}
	stopModules = func(logger logging.Logger) {
		slices.Reverse(modules) // we shut down in reverse order
		for _, m := range modules {
			if err := m.Stop(); err != nil {
				logger.Fatal().Err(err).Msg("failed to safely shutdown")
			}
		}
	}
	startModules = func(logger logging.Logger) {
		for _, m := range modules {
			if err := m.Start(); err != nil {
				logger.Fatal().Err(err).Msg("failed to start modules")
			}
		}
	}
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

	// initialize ingester
	logger.Info().Msg("initializing ingester...")
	ingest := ingester.NewIngester(logger.With().Str("module", "ingester").Logger())
	modules = append(modules, ingest)

	// initialize connection to storage backend
	logger.Info().Msg("initializing connection to storage backend...")
	storageBackend, err := storage.Connect(cfg.Storage, logger.With().Str("module", "storageBackend").Logger(), ingest.SendToDBChannel())
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to storage backend")
	}
	modules = append(modules, storageBackend)

	// initialize filter manager
	logger.Info().Msg("initializing filter manager...")
	filterManager := filter.NewFilterManager(ingest.SendToFilterManager(), storageBackend, logger.With().Str("module", "filterManager").Logger())
	modules = append(modules, filterManager)

	// initialize websocket handler
	logger.Info().Msg("initializing websocket server...")
	wsHandler := websocket.NewWebsocketServer(cfg.HTTP, logger.With().Str("module", "websocketServer").Logger(), ingest.SendToWSHandlerChannel(), filterManager.SendChannel())
	modules = append(modules, wsHandler)

	// ingester and websocket handler now communicating bi-directionally
	ingest.SetRecvChannel(wsHandler.SendChannel())

	// start modules
	logger.Info().Msg("starting modules...")
	startModules(logger)

	// hang until we shutdown
	<-interruptHandler.ShutdownDoneChannel()

	// shutdown
	stopModules(logger)
	logger.Info().Msg("shutdown complete")
}
