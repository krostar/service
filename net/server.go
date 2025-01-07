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
func Serve(server Server, listener net.Listener, opts ...ServeOption) service.RunFunc {
	return func(ctx context.Context) error {
		o := serveOptions{
			shutdownTimeout:          time.Second * 30,
			serveErrorTransformer:    func(err error) error { return err },
			shutdownErrorTransformer: func(err error) error { return err },
		}
		for _, opt := range opts {
			opt(&o)
		}

		cerr := make(chan error)
		go func() {
			defer listener.Close() //nolint:errcheck // listener probably will complain, we don't care
			if err := server.Serve(listener); err != nil {
				cerr <- o.serveErrorTransformer(fmt.Errorf("unable to serve listener: %w", err))
				return
			}
			cerr <- nil
		}()

		select {
		case err := <-cerr: // server exit without asking, even if err is nil it should be considered an error
			return fmt.Errorf("server stopped serving abruptly: %w", err)
		case <-ctx.Done():
			shutdownCtx := context.Background()
			if o.shutdownTimeout > 0 {
				var cancel context.CancelFunc
				shutdownCtx, cancel = context.WithTimeout(context.Background(), o.shutdownTimeout)
				defer cancel()
			}

			if err := server.Shutdown(shutdownCtx); err != nil { //nolint:contextcheck // we don't want to provide the function's context to give some time to the server to gracefully shut down
				return multierr.Combine(o.shutdownErrorTransformer(fmt.Errorf("unable to shut server down: %w", err)), <-cerr)
			}
		}

		return <-cerr
	}
}
