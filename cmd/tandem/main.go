package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/SSSOC-CAN/laniakea/utils"
	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/TheRebelOfBabylon/tandem/logger"
	u "github.com/TheRebelOfBabylon/tandem/utils"
)

func main() {
	configDir := flag.String("config", utils.AppDataDir("tandem", false), "Directory of the toml config file")
	verFlag := flag.Bool("version", false, "Display current tandem version")
	logConsoleOutFlag := flag.Bool("consoleoutput", true, "Print logs to the console")
	flag.Parse()
	if *verFlag {
		fmt.Fprintf(os.Stdout, "tandem version %s\n", u.AppVersion)
		os.Exit(0)
	}
	_, err := config.InitConfig(*configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not initialize configuration: %v\n", err)
		os.Exit(1)
	}
	_, err = logger.InitLogger(*logConsoleOutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not initialize logger: %v\n", err)
		os.Exit(1)
	}
}
