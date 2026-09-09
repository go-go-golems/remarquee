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
	// Pass one cancellation context through Cobra, but also bound shutdown when
	// rmapi's context-free authentication cannot observe that cancellation.
	return runInterruptible(run, interrupts, 2*time.Second, os.Exit)
}

// runInterruptible supplies the top-level context passed through the verbs;
// it does not replace their cooperative context cancellation. The first SIGINT
// cancels that context. A second SIGINT or expiry of grace forces exit with 130.
//
// The watchdog is needed because api.AuthHttpCtx belongs to the rmapi dependency:
// its "Ctx" holds an HTTP client and tokens, not a context.Context. It prompts
// on stdin and exchanges tokens without accepting the caller's context, so
// canceling our top-level context alone cannot unblock those operations.
// Revisit this fallback when rmapi supports cancellable authentication and the
// remaining long-running work cooperates with cancellation. The injected run,
// signal channel, grace period, and exit function make this policy testable.
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
