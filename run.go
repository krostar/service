package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/multierr"
)

// Runner defines how runners are handled.
type Runner interface {
	Run(ctx context.Context) error
}

// RunFunc type is an adapter to allow the use of functions as Runner.
// RunFunc(f) is a Runner that calls f as Run() function.
type RunFunc func(ctx context.Context) error

// Run implements Runner.
func (f RunFunc) Run(ctx context.Context) error { return f(ctx) }

// Run starts all runners and blocks until all runners returned.
// Runners are expected to stop only due to context cancellation reasons.
// This mean that context.Canceled on runners is not considered an error.
//
// Run returns an error if:
//   - a runner returned unexpectedly (with or without error) ; in that case ErrUnexpectedReturn is returned
//   - after being stopped, a runner returned an error that is not context.Canceled
func Run(ctx context.Context, runner Runner, runners ...Runner) error {
	runners = append([]Runner{runner}, runners...)

	// wrap root context with a cancel func that is called when any runner stops.
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	runnerErrs := make([]error, len(runners))

	var wg sync.WaitGroup
	wg.Add(len(runners))

	for runnerIdx, runner := range runners {
		go func(runnerIdx int, runner Runner) {
			defer func() {
				wg.Done()
				cancel() // make all other runner quit
			}()

			var runnerErr error

			if err := runner.Run(runCtx); err != nil {
				if runCtx.Err() == nil { // runner quit unexpectedly with error
					runnerErr = runnerError(runnerIdx, ErrUnexpectedReturn, err)
				} else if !errors.Is(err, context.Canceled) { // runner quit in error but not because it was canceled
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

	return multierr.Combine(runnerErrs...)
}

func runnerError(runnerIdx int, errs ...error) error {
	return fmt.Errorf("(runner %d) %w", runnerIdx+1, multierr.Combine(errs...))
}
