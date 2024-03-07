package netservice

import (
	"time"
)

type serveOptions struct {
	shutdownTimeout          time.Duration
	shutdownErrorTransformer func(error) error
	serveErrorTransformer    func(error) error
}

// ServeOption defines options applier for the server.
type ServeOption func(*serveOptions)

// ServerWithShutdownTimeout sets the provider timeout to shut down.
func ServerWithShutdownTimeout(timeout time.Duration) ServeOption {
	return func(o *serveOptions) {
		o.shutdownTimeout = timeout
	}
}

// ServerWithServeErrorTransformer sets a function to transform serve errors.
func ServerWithServeErrorTransformer(f func(error) error) ServeOption {
	return func(o *serveOptions) {
		o.serveErrorTransformer = f
	}
}

// ServerWithShutdownErrorTransformer sets a function to transform shutdown errors.
func ServerWithShutdownErrorTransformer(f func(error) error) ServeOption {
	return func(o *serveOptions) {
		o.shutdownErrorTransformer = f
	}
}
