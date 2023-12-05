package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func Test_RunFunc(t *testing.T) {
	called := false
	assert.NilError(t, RunFunc(func(ctx context.Context) error {
		called = true
		return nil
	}).Run(context.Background()))
	assert.Equal(t, called, true)
}

func Test_Run(t *testing.T) {
	t.Run("runner ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()

		assert.NilError(t, Run(ctx,
			RunFunc(func(ctx context.Context) error {
				assert.Check(t, ctx.Err() == nil)
				<-ctx.Done()
				return nil
			}),
			RunFunc(func(ctx context.Context) error {
				assert.Check(t, ctx.Err() == nil)
				<-ctx.Done()
				return nil
			}),
		))
	})

	t.Run("one runner stopping unexpectedly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := Run(ctx,
			RunFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}),
			RunFunc(func(ctx context.Context) error {
				return nil
			}),
		)
		assert.ErrorIs(t, err, ErrUnexpectedReturn)
		assert.ErrorContains(t, err, "runner 2")
	})

	t.Run("one runner failing unexpectedly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		runErr := errors.New("boom")

		err := Run(ctx,
			RunFunc(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}),
			RunFunc(func(ctx context.Context) error {
				return runErr
			}),
		)
		assert.ErrorIs(t, err, ErrUnexpectedReturn)
		assert.ErrorIs(t, err, runErr)
		assert.ErrorContains(t, err, "runner 2")
	})

	t.Run("one runner failing after stop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second)
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
		assert.ErrorIs(t, err, runErr)
		assert.ErrorContains(t, err, "runner 1")
	})

	t.Run("multiple runner failing after stop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second)
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
		assert.ErrorIs(t, err, runErr1)
		assert.ErrorIs(t, err, runErr2)
	})
}

func Test_runnerError(t *testing.T) {
	err := errors.New("boom")
	rerr := runnerError(0, err)

	assert.ErrorIs(t, rerr, err)
	assert.Error(t, rerr, "(runner 1) boom")
}
