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

// ServeWithShutdownTimeout sets the provider timeout to shut down.
func ServeWithShutdownTimeout(timeout time.Duration) ServeOption {
	return func(o *serveOptions) {
		o.shutdownTimeout = timeout
	}
}

// ServeWithServeErrorTransformer sets a function to transform serve errors.
func ServeWithServeErrorTransformer(f func(error) error) ServeOption {
	return func(o *serveOptions) {
		o.serveErrorTransformer = f
	}
}

// ServeWithShutdownErrorTransformer sets a function to transform shutdown errors.
func ServeWithShutdownErrorTransformer(f func(error) error) ServeOption {
	return func(o *serveOptions) {
		o.shutdownErrorTransformer = f
	}
}
