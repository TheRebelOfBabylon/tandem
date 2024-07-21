package logging

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	formatLvlFunc = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %s |", i))
	}
	ErrInvalidLogLevel = errors.New("invalid log level")
)

type Logger struct {
	zerolog.Logger
}

func NewLogger() Logger {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, FormatLevel: formatLvlFunc, TimeLocation: time.UTC}

	return Logger{
		Logger: zerolog.New(output).With().Timestamp().Logger(),
	}
}

// SetLogLevel sets the global log level
func (log Logger) SetLogLevel(lvl string) (Logger, error) {
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
