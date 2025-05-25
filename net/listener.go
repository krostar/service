package netservice

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/krostar/service"
)

// NewListener creates a new network listener with the specified configuration options.
//
// This function provides a flexible way to create network listeners with various
// configuration options including TLS encryption, keep-alive settings, systemd
// socket activation, and custom network parameters.
//
// NewListener applies configuration in the following priority order:
//  1. systemd socket activation (if enabled and available)
//  2. manual network and address configuration
//  3. TLS wrapping (if configured)
//  4. keep-alive settings
//
// When called without options, NewListener provides:
//   - background context for operations
//   - 1-minute TCP keep-alive interval
//   - no TLS encryption
//   - no systemd socket activation
//
// When systemd socket activation is enabled, NewListener will:
//  1. check for systemd-provided file descriptors
//  2. use the first available file descriptor if found
//  3. fall back to manual listener creation if none available
//  4. close unused file descriptors to prevent resource leaks
//
// TLS is automatically applied when a TLS configuration is provided:
//
// NewListener returns errors for:
//   - invalid option configurations
//   - network binding failures
//   - TLS configuration errors
//   - systemd socket activation failures
//
// Always close listeners when done to free system resources.
func NewListener(opts ...ListenOption) (net.Listener, error) {
	o := listenOptions{
		ctx:       context.Background(),
		keepAlive: time.Minute,
	}
	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return nil, fmt.Errorf("unable to apply option: %w", err)
		}
	}

	var listener net.Listener

	if o.useSystemdProvidedFileDescriptor {
		systemdListeners, err := GetSystemdListeners(true)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve systemd listeners: %w", err)
		}

		if len(systemdListeners) > 0 {
			listener = systemdListeners[0]
			for i := 1; i < len(systemdListeners); i++ {
				_ = systemdListeners[i].Close() //nolint:errcheck // we don't need the remaining listener, we don't really care about errors here
			}
		}
	}

	if listener == nil && o.network != "" && o.address != "" {
		lc := net.ListenConfig{
			KeepAlive: o.keepAlive,
			KeepAliveConfig: net.KeepAliveConfig{
				Enable:   o.keepAlive > 0,
				Interval: o.keepAlive,
			},
		}

		l, err := lc.Listen(o.ctx, o.network, o.address)
		if err != nil {
			return nil, fmt.Errorf("unable to listen on %s: %w", o.address, err)
		}

		listener = l
	}

	if listener == nil {
		return nil, errors.New("no listener configured")
	}

	if o.tlsConfig != nil && strings.HasPrefix(listener.Addr().Network(), "tcp") {
		listener = tls.NewListener(listener, o.tlsConfig)
	}

	return listener, nil
}

// ListenAndServeOption represents configuration options for the ListenAndServe function.
//
// This type allows ListenAndServe to accept both ListenOption and ServeOption
// configurations in a single function call, providing a unified interface for
// configuring both listener creation and server management.
//
// Supported Option Types:
//   - ListenOption: controls listener creation (address, TLS, keep-alive, systemd)
//   - ServeOption: controls server behavior (shutdown timeout, error handling)
//
// The interface{} type (any) is used to allow flexibility while maintaining
// type safety through runtime checks. Invalid option types will cause a panic
// with a descriptive error message during function execution.
type ListenAndServeOption any

// ListenAndServe creates a listener and serves a server on it in a single operation.
//
// This is a convenience function that combines NewListener and Serve into a single
// call, providing the most common pattern for creating network services. It accepts
// both ListenOption and ServeOption configurations and automatically handles the
// listener creation and server lifecycle management.
//
// ListenAndServe performs the following steps:
//  1. separates ListenOption and ServeOption configurations
//  2. creates a new listener using NewListener with ListenOptions
//  3. creates a server runner using Serve with ServeOptions
//  4. returns a service.RunFunc that manages the complete lifecycle
//
// Returns a service.RunFunc that can be used with service.Run() to manage
// the complete server lifecycle including graceful shutdown.
//
// Errors from listener creation are returned immediately when the RunFunc is executed.
// The listener is created when the RunFunc executes, not when ListenAndServe is called.
func ListenAndServe(server Server, opts ...ListenAndServeOption) service.RunFunc {
	var (
		lopts []ListenOption
		sopts []ServeOption
	)
	for _, opt := range opts {
		switch o := opt.(type) {
		case ListenOption:
			lopts = append(lopts, o)
		case ServeOption:
			sopts = append(sopts, o)
		default:
			panic(fmt.Sprintf("unknown option type %T", opt))
		}
	}

	return func(ctx context.Context) error {
		listener, err := NewListener(append(lopts, ListenWithContext(ctx))...)
		if err != nil {
			return err
		}
		return Serve(server, listener, sopts...)(ctx)
	}
}
