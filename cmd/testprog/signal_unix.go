package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func handleSignals() {
	// handle SIGINT and SIGTERM.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range c {
			switch sig {
			case syscall.SIGINT:
				fmt.Println("\x1b[33mgot SIGINT, ignored\x1b[0m")
				_interrupted = true
			case syscall.SIGTERM:
				fmt.Println("\x1b[31mgot SIGTERM\x1b[0m")
				os.Exit(1)
			}
		}
	}()
}
