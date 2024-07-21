package signal

import (
	"log"
	"os"
	"os/signal"
	"syscall"
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
}

// NewInterruptHanlder instatiates a new interrupt handler
func NewInterruptHandler() *InterruptHandler {
	ih := &InterruptHandler{
		interruptChan: make(chan os.Signal),
		quitChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
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
			log.Println("already shutting down...")
			return
		}
		isShutdown = true
		log.Println("beginning shutdown sequence...")
		close(i.quitChan)
	}
	for {
		select {
		case signal := <-i.interruptChan:
			log.Printf("received %v", signal)
			shutdown()
		case <-i.quitChan:
			log.Println("gracefully shutting down...")
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
