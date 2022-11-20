package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/SSSOC-CAN/laniakea/intercept"
	"github.com/SSSOC-CAN/laniakea/utils"
	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/logger"
	u "github.com/TheRebelOfBabylon/tandem/utils"
)

func main() {
	configDir := flag.String("config", utils.AppDataDir("tandem", false), "Directory of the toml config file")
	verFlag := flag.Bool("version", false, "Display current tandem version")
	flag.Parse()
	if *verFlag {
		fmt.Fprintf(os.Stdout, "tandem version %s\n", u.AppVersion)
		os.Exit(0)
	}
	interceptor, err := intercept.InitInterceptor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not initialize signal interceptor: %v\n", err)
		os.Exit(1)
	}
	cfg, err := config.InitConfig(*configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not initialize configuration: %v\n", err)
		os.Exit(1)
	}
	log, err := logger.InitLogger(&cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not initialize logger: %v\n", err)
		os.Exit(1)
	}
	interceptor.Logger = &log
	log.Info().Msg("testing info")
	log.Debug().Msg("testing debug")
	log.Warn().Msg("testing warn")
	log.Error().Msg("testing error")
	log.Trace().Msg("testing trace")
	//log.Fatal().Msg("testing fatal")
	go func() {
		time.Sleep(5 * time.Second)
		interceptor.RequestShutdown()
	}()
	<-interceptor.ShutdownChannel()
}