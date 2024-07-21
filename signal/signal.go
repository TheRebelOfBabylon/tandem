package signal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
)

var (
	signalsToCatch = []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}
)

type InterruptHandler struct {
	interruptChan chan os.Signal
	quitChan      chan struct{}
	doneChan      chan struct{}
	logger        zerolog.Logger
}

// NewInterruptHanlder instantiates a new interrupt handler
func NewInterruptHandler(logger zerolog.Logger) *InterruptHandler {
	ih := &InterruptHandler{
		interruptChan: make(chan os.Signal),
		quitChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
		logger:        logger,
	}
	signal.Notify(ih.interruptChan, signalsToCatch...)
	go ih.mainHandler()
	return ih
}

// mainHanlder is the main interrupt handler routine for catching signals
func (i *InterruptHandler) mainHandler() {
	var isShutdown bool
	shutdown := func() {
		if isShutdown {
			i.logger.Warn().Msg("already shutting down...")
			return
		}
		isShutdown = true
		i.logger.Info().Msg("beginning shutdown sequence...")
		close(i.quitChan)
	}
	for {
		select {
		case signal := <-i.interruptChan:
			i.logger.Info().Msgf("received %v", signal)
			shutdown()
		case <-i.quitChan:
			i.logger.Info().Msg("gracefully shutting down...")
			signal.Stop(i.interruptChan)
			close(i.doneChan)
			return
		}
	}
}

// ShutdownDoneChannel returns the channel used to signal the end of the shutdown sequence
func (i *InterruptHandler) ShutdownDoneChannel() <-chan struct{} {
	return i.doneChan
}
