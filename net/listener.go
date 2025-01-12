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

// NewListener creates a new listener.
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

// ListenAndServeOption allow underlying option to be of type ListenOption or ServeOption.
type ListenAndServeOption any

// ListenAndServe is a shortcut for NewListener and Serve.
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
