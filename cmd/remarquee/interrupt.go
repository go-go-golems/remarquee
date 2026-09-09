package main

import (
	"context"
	"os"
	"os/signal"
	"time"
)

// executeInterruptible gives cooperative work a cancellation context without
// swallowing SIGINT indefinitely in rmapi's context-free auth/local-file code.
func executeInterruptible(run func(context.Context) error) error {
	interrupts := make(chan os.Signal, 2)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	return runInterruptible(run, interrupts, 2*time.Second, os.Exit)
}

func runInterruptible(run func(context.Context) error, interrupts <-chan os.Signal, grace time.Duration, exit func(int)) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-stop:
			return
		case <-interrupts:
			cancel()
		}
		timer := time.NewTimer(grace)
		defer timer.Stop()
		select {
		case <-stop:
			return
		case <-interrupts:
			exit(130)
		case <-timer.C:
			exit(130)
		}
	}()
	err := run(ctx)
	close(stop)
	<-done
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
