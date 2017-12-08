package main

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/uoregon-libraries/gopkg/logger"
)

var isDone int32

func catchInterrupts() {
	var sigInt = make(chan os.Signal, 1)
	signal.Notify(sigInt, syscall.SIGINT)
	signal.Notify(sigInt, syscall.SIGTERM)
	go func() {
		for range sigInt {
			if done() {
				logger.Errorf("Force-interrupt detected; some jobs may need to be manually cleaned up")
				os.Exit(1)
			}

			logger.Warnf("Interrupt detected; attempting to clean up.  Another signal will immediately end the process.")
			quit()
		}
	}()
}

func quit() {
	atomic.StoreInt32(&isDone, 1)
}

func done() bool {
	return atomic.LoadInt32(&isDone) == 1
}
