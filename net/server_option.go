package netservice

import (
	"time"
)

type serveOptions struct {
	// shutdownTimeout specifies the maximum duration to wait for graceful shutdown.
	// If the server doesn't shut down within this time, the shutdown is considered failed.
	// A zero value means no timeout (wait indefinitely).
	shutdownTimeout time.Duration

	// shutdownErrorTransformer allows customization of how shutdown errors are handled.
	// This can be used to ignore certain expected errors or add additional context.
	shutdownErrorTransformer func(error) error

	// serveErrorTransformer allows customization of how serving errors are handled.
	// This is commonly used to ignore expected errors like http.ErrServerClosed.
	serveErrorTransformer func(error) error
}

// ServeOption defines a function that can modify serveOptions.
//
// ServeOptions are used to customize the behavior of the Serve function,
// allowing fine-grained control over server lifecycle management, error handling,
// and shutdown behavior.
//
// Options follow the functional options pattern, making them composable and
// easy to use. Multiple options can be passed to Serve() and they will be
// applied in order.
type ServeOption func(*serveOptions)

// ServeWithShutdownTimeout sets the maximum duration to wait for graceful server shutdown.
//
// When the context is canceled, the server's Shutdown() method is called with a context
// that will timeout after the specified duration. If the server doesn't shut down
// gracefully within this time, the shutdown is considered failed and an error is returned.
//
// A timeout of 0 means no timeout - the shutdown will wait indefinitely for the server
// to stop gracefully. This is generally not recommended for production use as it can
// cause the application to hang if the server doesn't shut down properly.
func ServeWithShutdownTimeout(timeout time.Duration) ServeOption {
	return func(o *serveOptions) { o.shutdownTimeout = timeout }
}

// ServeWithServeErrorTransformer sets a function to transform errors returned by server.Serve().
//
// This is commonly used to filter out expected errors that should not be treated as failures.
// For example, http.ErrServerClosed is returned by HTTP servers during normal shutdown
// and should typically be ignored.
//
// The transformer function receives the original error and should return:
//   - nil to ignore the error (treat as successful shutdown)
//   - the original error unchanged to preserve the error
//   - a wrapped or modified error to add context or change the error type
func ServeWithServeErrorTransformer(f func(error) error) ServeOption {
	return func(o *serveOptions) { o.serveErrorTransformer = f }
}

// ServeWithShutdownErrorTransformer sets a function to transform errors returned by server.Shutdown().
//
// This allows customization of how shutdown errors are handled. Shutdown errors typically
// indicate that the server failed to stop gracefully within the configured timeout,
// or that there was an issue during the shutdown process.
//
// The transformer function receives the shutdown error and should return:
//   - nil to ignore the error (treat shutdown as successful)
//   - the original error unchanged to preserve the error
//   - a wrapped or modified error to add context or change the error type
func ServeWithShutdownErrorTransformer(f func(error) error) ServeOption {
	return func(o *serveOptions) {
		o.shutdownErrorTransformer = f
	}
}
