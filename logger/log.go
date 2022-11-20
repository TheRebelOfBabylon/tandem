package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/SSSOC-CAN/laniakea/utils"
	"github.com/mattn/go-colorable"
	color "github.com/mgutz/ansi"
	"github.com/rs/zerolog"
)

const (
	logFileNameRoot = "tandem"
	logFileName     = "tandem.log"
	logFileExt      = "log"
)

func InitLogger(consoleOutput bool, logFileDir string, logLevel string, maxLogFileSize uint32, maxLogFiles uint16) (zerolog.Logger, error) {
	// open log file
	var (
		log_file *os.File
		logger   zerolog.Logger
		err      error
	)
	log_file, err = os.OpenFile(path.Join(logFileDir, logFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// try to create AppData/Tandem | Application Support/Tandem or ~/.tandem file if logFileDir is pointing there
		if utils.AppDataDir("tandem", false) == logFileDir {
			err = os.Mkdir(utils.AppDataDir("tandem", false), 0755)
			if err != nil {
				return logger, err
			}
			log_file, err = os.OpenFile(path.Join(logFileDir, logFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return logger, err
			}
		} else {
			return logger, err
		}
	}
	// use modified writer to manage log file size
	modded_file := &moddedFileWriter{
		File:         log_file,
		maxFileSize:  maxLogFileSize * 1000000,
		maxFiles:     maxLogFiles,
		fileNameRoot: logFileNameRoot,
		fileExt:      logFileExt,
		pathToFile:   logFileDir,
	}
	// add console output if option is chosen
	if consoleOutput {
		output := zerolog.NewConsoleWriter()
		if runtime.GOOS == "windows" {
			output.Out = colorable.NewColorableStdout()
		} else {
			output.Out = os.Stderr
		}
		output.FormatLevel = func(i interface{}) string {
			var msg string
			switch v := i.(type) {
			default:
				x := fmt.Sprintf("%v", v)
				switch x {
				case "info":
					msg = color.Color(strings.ToUpper("["+x+"]"), "green")
				case "panic":
					msg = color.Color(strings.ToUpper("["+x+"]"), "red")
				case "fatal":
					msg = color.Color(strings.ToUpper("["+x+"]"), "red")
				case "error":
					msg = color.Color(strings.ToUpper("["+x+"]"), "red")
				case "warn":
					msg = color.Color(strings.ToUpper("["+x+"]"), "yellow")
				case "debug":
					msg = color.Color(strings.ToUpper("["+x+"]"), "yellow")
				case "trace":
					msg = color.Color(strings.ToUpper("["+x+"]"), "magenta")
				}
			}
			return msg + fmt.Sprintf("\t")
		}
		multi := zerolog.MultiLevelWriter(output, modded_file)
		logger = zerolog.New(multi).With().Timestamp().Logger()
	} else {
		logger = zerolog.New(modded_file).With().Timestamp().Logger()
	}
	// set log level
	switch logLevel {
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "TRACE":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	return logger, nil
}
