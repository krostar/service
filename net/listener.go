package netservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/krostar/service"
)

// NewListener creates a new listener.
func NewListener(ctx context.Context, address string, opts ...ListenerOption) (net.Listener, error) {
	o := listenerOptions{
		network:   "tcp4",
		keepAlive: time.Minute,
	}
	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return nil, fmt.Errorf("unable to apply option: %w", err)
		}
	}

	lc := net.ListenConfig{
		KeepAlive: o.keepAlive,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   o.keepAlive > 0,
			Interval: o.keepAlive,
		},
	}

	l, err := lc.Listen(ctx, o.network, address)
	if err != nil {
		return nil, fmt.Errorf("unable to listen on %s: %w", address, err)
	}

	if o.tlsConfig != nil {
		l = tls.NewListener(l, o.tlsConfig)
	}

	return l, err
}

// ListenAndServeOption allow underlying option to be of type ListenerOption or ServeOption.
type ListenAndServeOption any

// ListenAndServe is a shortcut for NewListener and Serve.
func ListenAndServe(address string, server Server, opts ...ListenAndServeOption) service.RunFunc {
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

	return func(ctx context.Context) error {
		listener, err := NewListener(ctx, address, lopts...)
		if err != nil {
			return err
		}
		return Serve(server, listener, sopts...)(ctx)
	}
}
