package netservice

import (
	"time"
)

type serveOptions struct {
	gracefulTimeout          time.Duration
	serveErrorTransformer    func(error) error
	shutdownErrorTransformer func(error) error
}

// ServeOption defines options applier for the server.
type ServeOption func(*serveOptions)

// ServerWithGracefulTimeout sets the provider timeout for graceful shutdown.
func ServerWithGracefulTimeout(timeout time.Duration) ServeOption {
	return func(o *serveOptions) {
		o.gracefulTimeout = timeout
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
