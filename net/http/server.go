package httpnetservice

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/krostar/service"
	netservice "github.com/krostar/service/net"
)

// NewServer creates a new server with sensible default values, customizable through options.
func NewServer(handler http.Handler, opts ...ServerOption) (*http.Server, error) {
	server := &http.Server{
		Handler:                      handler,
		DisableGeneralOptionsHandler: true,
		ReadTimeout:                  5 * time.Second,
		IdleTimeout:                  3 * time.Minute,
		MaxHeaderBytes:               1 << 13, // 8ko
	}

	for _, opt := range opts {
		if err := opt(server); err != nil {
			return nil, fmt.Errorf("unable to apply server option: %w", err)
		}
	}

	return server, nil
}

// Serve is the equivalent of calling netservice.Serve with the server error transformer
// configured to skip normal errors returned by the http server.
func Serve(server *http.Server, listener net.Listener, opts ...netservice.ServeOption) service.RunFunc {
	opts = append(opts, netservice.ServeWithServeErrorTransformer(func(err error) error {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}))
	return netservice.Serve(server, listener, opts...)
}

// ListenAndServe creates a new server and start listening on it.
func ListenAndServe(handler http.Handler, opts ...ListenAndServeOption) service.RunFunc {
	var (
		sropts []ServerOption
		lopts  []netservice.ListenOption
		sopts  []netservice.ServeOption
	)
	for _, opt := range opts {
		switch o := opt.(type) {
		case ServerOption:
			sropts = append(sropts, o)
		case netservice.ListenOption:
			lopts = append(lopts, o)
		case netservice.ServeOption:
			sopts = append(sopts, o)
		default:
			panic(fmt.Sprintf("unknown option type %T", opt))
		}
	}

	return func(ctx context.Context) error {
		server, err := NewServer(handler, sropts...)
		if err != nil {
			return err
		}

		listener, err := netservice.NewListener(append(lopts, netservice.ListenWithContext(ctx))...)
		if err != nil {
			return err
		}

		return Serve(server, listener, sopts...)(ctx)
	}
}
