package netservice

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/krostar/service"
)

// Server defines the interface for network services that can be managed by this package.
//
// This interface is designed to work with servers that:
//   - accept connections through a net.Listener
//   - support graceful shutdown via context cancellation
//   - follow the "standard" Go server pattern
//
// The interface is compatible with many standard library servers including:
//   - *http.Server
//   - *grpc.Server
//   - custom TCP/UDP servers
//
// Serve should:
//   - block until the server stops or encounters an error
//   - handle listener closure gracefully
//   - be safe to call Shutdown() concurrently with Serve()
//
// Shutdown should:
//   - initiate graceful shutdown of the server
//   - respect the provided context timeout
//   - return quickly if the server is already stopped
//   - be safe to call multiple times
type Server interface {
	Serve(listener net.Listener) error
	Shutdown(ctx context.Context) error
}

// Serve creates a service.Runner that manages the lifecycle of a network server.
//
// This function wraps a Server and net.Listener into a Runner that can be managed
// by the service package. It handles the complexity of graceful shutdown, error
// transformation, and proper resource cleanup.
//
// The returned Runner will:
//  1. start serving the server on the provided listener in a separate goroutine
//  2. monitor for context cancellation to initiate graceful shutdown
//  3. apply error transformations to distinguish expected vs unexpected errors
//  4. ensure the listener is closed when the server stops
//  5. respect shutdown timeouts to prevent hanging
//
// Shutdown Process: when the context is canceled:
//  1. server.Shutdown() is called with a timeout context
//  2. the function waits for the server to stop serving
//  3. if shutdown times out, the server is considered failed
//  4. the listener is closed to free system resources
//
// Errors are categorized as:
//   - expected: Transformed by ServeErrorTransformer (e.g., http.ErrServerClosed -> nil)
//   - unexpected: Server stopped without context cancellation, failures during graceful shutdown process
//
// Configuration Options:
//   - ServeWithShutdownTimeout: Set maximum time to wait for graceful shutdown
//   - ServeWithServeErrorTransformer: Transform serving errors (e.g., ignore expected errors)
//   - ServeWithShutdownErrorTransformer: Transform shutdown errors
//
// The returned Runner is safe to use concurrently with other Runners in service.Run().
// However, the same Runner instance should not be used in multiple service.Run() calls.
func Serve(server Server, listener net.Listener, opts ...ServeOption) service.RunFunc {
	return func(ctx context.Context) error {
		o := serveOptions{
			shutdownTimeout:          time.Second * 30,                     // 30 seconds for graceful shutdown
			serveErrorTransformer:    func(err error) error { return err }, // pass through all errors
			shutdownErrorTransformer: func(err error) error { return err }, // pass through all errors
		}
		for _, opt := range opts {
			opt(&o)
		}

		cerr := make(chan error) // channel to receive errors from the serving goroutine
		defer func() { close(cerr) }()

		go func() { // server goroutine
			defer listener.Close() //nolint:errcheck // listener probably will complain, we don't care

			// start serving - this should block until server stops
			if err := server.Serve(listener); err != nil {
				// transform the error according to user configuration
				cerr <- o.serveErrorTransformer(fmt.Errorf("unable to serve listener: %w", err))
				return
			}

			cerr <- nil
		}()

		select {
		case err := <-cerr: // server stopped unexpectedly (without context cancellation)
			// even if err is nil, this indicates the server exited prematurely
			// which is typically not expected for long-running services
			return fmt.Errorf("server stopped serving abruptly: %w", err)

		case <-ctx.Done(): // context was canceled - initiate graceful shutdown
			// create a separate context for shutdown to prevent inheritance of cancellation
			// this gives the server time to shut down gracefully even if the original context is canceled
			shutdownCtx := context.Background()
			if o.shutdownTimeout > 0 {
				var cancel context.CancelFunc
				shutdownCtx, cancel = context.WithTimeout(context.Background(), o.shutdownTimeout)
				defer cancel()
			}

			if err := server.Shutdown(shutdownCtx); err != nil { //nolint:contextcheck // we don't want to provide the function's context to give some time to the server to gracefully shut down
				// shutdown failed - combine the shutdown error with any serving error
				shutdownErr := o.shutdownErrorTransformer(fmt.Errorf("unable to shut server down: %w", err))
				servingErr := <-cerr // wait for serving goroutine to finish
				return errors.Join(shutdownErr, servingErr)
			}

			return <-cerr
		}
	}
}
