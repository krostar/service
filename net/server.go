package netservice

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/multierr"

	"github.com/krostar/service"
)

// Server defines methods to serve and stop a network service.
type Server interface {
	Serve(listener net.Listener) error
	Shutdown(ctx context.Context) error
}

// Serve returns a runner that serves the server through the provided listener.
// On context cancellation, the server tries to gracefully shutdown.
// Once the graceful timeout is reached the shutdown context is canceled.
func Serve(server Server, listener net.Listener, opts ...ServeOption) service.RunFunc {
	return func(ctx context.Context) error {
		o := serveOptions{
			gracefulTimeout:          time.Second * 15,
			serveErrorTransformer:    func(err error) error { return err },
			shutdownErrorTransformer: func(err error) error { return err },
		}
		for _, opt := range opts {
			opt(&o)
		}

		cerr := make(chan error)
		go func() {
			if err := server.Serve(listener); err != nil {
				cerr <- o.serveErrorTransformer(fmt.Errorf("unable to serve listener: %w", err))
				return
			}
			cerr <- nil
		}()

		select {
		case err := <-cerr:
			return err
		case <-ctx.Done():
			shutdownCtx := context.Background()
			if o.gracefulTimeout > 0 {
				var cancel context.CancelFunc
				shutdownCtx, cancel = context.WithTimeout(context.Background(), o.gracefulTimeout)
				defer cancel()
			}

			if err := server.Shutdown(shutdownCtx); err != nil { //nolint:contextcheck // we don't want to provide the function's context to give some time to the server to gracefully shut down
				return multierr.Combine(o.shutdownErrorTransformer(fmt.Errorf("unable to shut server down: %w", err)), <-cerr)
			}
		}

		return <-cerr
	}
}

// ListenAndServe is a shortcut for NewListener and Serve.
func ListenAndServe(ctx context.Context, address string, server Server, opts ...any) error {
	var (
		lopts []ListenerOption
		sopts []ServeOption
	)
	for _, opt := range opts {
		if lopt, ok := opt.(ListenerOption); ok && lopt != nil {
			lopts = append(lopts, lopt)
		}
		if sopt, ok := opt.(ServeOption); ok && sopt != nil {
			sopts = append(sopts, sopt)
		}
	}

	listener, err := NewListener(ctx, address, lopts...)
	if err != nil {
		return err
	}

	return Serve(server, listener, sopts...)(ctx)
}
