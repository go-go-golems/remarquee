package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestInterruptHelperProcess(t *testing.T) {
	mode := os.Getenv("RMQ_INTERRUPT_HELPER")
	if mode == "" {
		return
	}
	err := executeInterruptible(func(ctx context.Context) error {
		fmt.Println("ready")
		<-ctx.Done()
		fmt.Println("cancel-observed")
		if mode == "cooperative" {
			return ctx.Err()
		}
		select {} // Deliberately uncooperative; the signal watchdog must exit.
	})
	if errors.Is(err, context.Canceled) {
		os.Exit(130)
	}
	os.Exit(1)
}

func TestActualSIGINT(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Process.Signal(os.Interrupt)")
	}
	for _, mode := range []string{"cooperative", "timeout", "second-interrupt"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInterruptHelperProcess$")
			cmd.Env = append(os.Environ(), "RMQ_INTERRUPT_HELPER="+mode)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			scanner := bufio.NewScanner(out)
			if !scanner.Scan() || scanner.Text() != "ready" {
				_ = cmd.Wait()
				t.Fatalf("child not ready: %s", stderr.String())
			}
			started := time.Now()
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				t.Fatal(err)
			}
			if !scanner.Scan() || scanner.Text() != "cancel-observed" {
				_ = cmd.Wait()
				t.Fatalf("no cancellation: %s", stderr.String())
			}
			if mode == "second-interrupt" {
				if err := cmd.Process.Signal(os.Interrupt); err != nil {
					t.Fatal(err)
				}
			}
			err = cmd.Wait()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 130 {
				t.Fatalf("exit=%v stderr=%s", err, stderr.String())
			}
			elapsed := time.Since(started)
			if elapsed > 4*time.Second {
				t.Fatalf("interrupt took %s", elapsed)
			}
			if mode == "timeout" && elapsed < 1900*time.Millisecond {
				t.Fatalf("did not allow grace period: %s", elapsed)
			}
			if mode != "timeout" && elapsed >= 1900*time.Millisecond {
				t.Fatalf("waited for fallback instead of promptly exiting: %s", elapsed)
			}
			if strings.Contains(stderr.String(), "Usage:") {
				t.Fatal(stderr.String())
			}
		})
	}
}

func TestInterruptibleNormalCompletion(t *testing.T) {
	want := errors.New("ordinary failure")
	err := runInterruptible(func(ctx context.Context) error {
		if ctx.Err() != nil {
			t.Fatal(ctx.Err())
		}
		return want
	}, make(chan os.Signal), time.Second, func(int) { t.Error("unexpected forced exit") })
	if !errors.Is(err, want) {
		t.Fatal(err)
	}
}
