package logging

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/rs/zerolog"
)

var (
	formatLvlFunc = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %s |", i))
	}
	ErrInvalidLogLevel = errors.New("invalid log level")
	consoleWriter      = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: formatLvlFunc, TimeLocation: time.UTC}
)

type Logger struct {
	zerolog.Logger
}

func NewLogger() Logger {
	return Logger{
		Logger: zerolog.New(consoleWriter).With().Timestamp().Logger(),
	}
}

// setLogLevel sets the global log level
func (log Logger) setLogLevel(lvl string) (Logger, error) {
	var newLogger Logger
	switch lvl {
	case "debug":
		newLogger.Logger = log.Level(zerolog.DebugLevel)
	case "info":
		newLogger.Logger = log.Level(zerolog.InfoLevel)
	case "error":
		newLogger.Logger = log.Level(zerolog.ErrorLevel)
	default:
		return log, fmt.Errorf("%w: %s", ErrInvalidLogLevel, lvl)
	}
	return newLogger, nil
}

// Configure configure the logger
func (log Logger) Configure(logCfg config.Log) (Logger, error) {
	// set the log level
	newLogger, err := log.setLogLevel(logCfg.Level)
	if err != nil {
		return log, err
	}
	// set a file writer
	if logCfg.LogFilePath != "" {
		logFile, err := os.OpenFile(logCfg.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return log, fmt.Errorf("failed to open log file: %w", err)
		}
		newLogger.Logger = newLogger.Output(zerolog.MultiLevelWriter(consoleWriter, logFile))
	}
	return newLogger, nil
}
