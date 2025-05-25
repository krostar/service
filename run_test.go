package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/krostar/test"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func Test_RunFunc(t *testing.T) {
	called := false
	test.Require(t, RunFunc(func(context.Context) error {
		called = true
		return nil
	}).Run(t.Context()) == nil)
	test.Assert(t, called)
}

func Test_Run(t *testing.T) {
	t.Run("runner ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(time.Millisecond * 100)
			cancel()
		}()

		test.Assert(t, Run(ctx,
			RunFunc(func(ctx context.Context) error {
				test.Assert(t, ctx.Err() == nil)
				<-ctx.Done()
				return nil
			}),
			RunFunc(func(ctx context.Context) error {
				test.Assert(t, ctx.Err() == nil)
				<-ctx.Done()
				return nil
			}),
		) == nil)
	})

	t.Run("one runner stopping unexpectedly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		err := Run(ctx,
			RunFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}),
			RunFunc(func(context.Context) error {
				return nil
			}),
		)
		test.Assert(t, err != nil && errors.Is(err, ErrUnexpectedReturn) && strings.Contains(err.Error(), "runner 2"))
	})

	t.Run("one runner failing unexpectedly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		runErr := errors.New("boom")

		err := Run(ctx,
			RunFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}),
			RunFunc(func(context.Context) error {
				return runErr
			}),
		)
		test.Assert(t, err != nil && errors.Is(err, ErrUnexpectedReturn) && errors.Is(err, runErr) && strings.Contains(err.Error(), "runner 2"))
	})

	t.Run("one runner failing after stop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(time.Millisecond * 100)
			cancel()
		}()

		runErr := errors.New("boom")

		err := Run(ctx,
			RunFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return runErr
			}),
			RunFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}),
		)
		test.Assert(t, err != nil && errors.Is(err, runErr) && strings.Contains(err.Error(), "runner 1"))
	})

	t.Run("multiple runner failing after stop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(time.Millisecond * 100)
			cancel()
		}()

		runErr1 := errors.New("boom")
		runErr2 := errors.New("papam")

		err := Run(ctx,
			RunFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return runErr1
			}),
			RunFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return runErr2
			}),
		)
		test.Assert(t, err != nil && errors.Is(err, runErr1) && errors.Is(err, runErr2))
	})
}

func Test_runnerError(t *testing.T) {
	anError := errors.New("boom")
	err := runnerError(0, anError)

	test.Assert(t, err != nil && errors.Is(err, anError) && strings.Contains(err.Error(), "(runner 1) boom"))
}
