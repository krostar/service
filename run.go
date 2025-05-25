// Package service provides utilities for managing long-running goroutines and services
// with graceful shutdown capabilities.
//
// The core functionality revolves around the Runner interface and the Run function,
// which allows you to start multiple goroutines concurrently and manage their lifecycle.
// When any runner stops (either due to context cancellation or unexpected termination),
// all other runners are signaled to stop as well.
//
// The Run function distinguishes between expected and unexpected termination:
//   - expected: Runner stops due to context cancellation (returns context.Canceled or nil)
//   - unexpected: Runner stops on its own or returns a non-cancellation error
//
// Unexpected termination triggers ErrUnexpectedReturn and causes all other runners to stop.
//
// Network Services: this package includes specialized support for network services in the net subpackage,
// providing HTTP server management, TLS configuration, and systemd integration.
// See the net subpackage for HTTP server utilities and examples.
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Runner defines the interface for long-running services that can be managed by this package.
//
// Implementations should:
//   - block until the context is canceled or an error occurs
//   - return context.Canceled or nil when stopped due to context cancellation
//   - return a non-nil error only for unexpected failures
//   - consider they will be only be run once
type Runner interface {
	Run(ctx context.Context) error
}

// RunFunc is an adapter that allows ordinary functions to be used as Runners.
// This is the most common way to create ad-hoc runners without defining a new type.
type RunFunc func(ctx context.Context) error

// Run implements the Runner interface by calling the underlying function.
func (f RunFunc) Run(ctx context.Context) error { return f(ctx) }

// Run starts all runners concurrently and blocks until all have stopped.
//
// The function creates a cancellable context that is passed to all runners.
// When any runner stops (for any reason), the context is canceled, signaling
// all other runners to stop gracefully.
//
// Behavior:
//   - runners are expected to run indefinitely until the context is canceled.
//   - any runner that stops on its own is considered to have failed unexpectedly,
//     which will cause all other runners to be stopped.
//
// Error Handling:
//
// Run returns an error in the following cases:
//   - a runner stopped unexpectedly (with or without an error): ErrUnexpectedReturn is returned
//   - a runner returned an error after context cancellation (excluding context.Canceled)
//   - multiple runners returned errors
func Run(ctx context.Context, runner Runner, runners ...Runner) error {
	runners = append([]Runner{runner}, runners...)

	// create a cancellable context that will be canceled when any runner stops
	// this ensures that when one service fails or stops, all others are signaled to stop
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// track errors from each runner separately to provide detailed error reporting
	runnerErrs := make([]error, len(runners))

	var wg sync.WaitGroup
	wg.Add(len(runners))

	// start each runner in its own goroutine
	for runnerIdx, runner := range runners {
		go func(runnerIdx int, runner Runner) {
			defer func() {
				wg.Done()
				// cancel context to signal all other runners to stop
				// this ensures fail-fast behavior: if any service stops, they all stop
				cancel()
			}()

			var runnerErr error

			defer func() { // add panic recovery to prevent a single runner panic from crashing the entire application
				if r := recover(); r != nil {
					runnerErrs[runnerIdx] = runnerError(runnerIdx, ErrUnexpectedReturn, ErrRunnerPanic, fmt.Errorf("panic: %v", r))
				}
			}()

			if err := runner.Run(runCtx); err != nil {
				if runCtx.Err() == nil { // runner quit unexpectedly with error
					runnerErr = runnerError(runnerIdx, ErrUnexpectedReturn, err)
				} else if !errors.Is(err, context.Canceled) { // we ask for exit and runner quit in error
					runnerErr = runnerError(runnerIdx, err)
				}
			} else {
				if runCtx.Err() == nil { // runner quit unexpectedly without error
					runnerErr = runnerError(runnerIdx, ErrUnexpectedReturn)
				}
			}

			runnerErrs[runnerIdx] = runnerErr
		}(runnerIdx, runner)
	}

	wg.Wait()

	return errors.Join(runnerErrs...)
}

// runnerError creates a descriptive error for a specific runner with 1-based indexing.
// This helps identify which specific service failed when multiple services are running.
func runnerError(runnerIdx int, errs ...error) error {
	return fmt.Errorf("(runner %d) %w", runnerIdx+1, errors.Join(errs...))
}
