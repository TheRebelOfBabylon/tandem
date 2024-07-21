package logging

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	formatLvlFunc = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-5s|", i))
	}
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
