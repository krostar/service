package netservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	tlsnetservice "github.com/krostar/service/net/tls"
)

type listenOptions struct {
	ctx context.Context //nolint:containedctx // ctx here is used for the Listen function and provided as an option

	network string
	address string

	keepAlive time.Duration
	tlsConfig *tls.Config

	useSystemdProvidedFileDescriptor bool
}

// ListenOption defines options applier for the listener.
type ListenOption func(*listenOptions) error

// ListenWithContext sets the context provided to net.Listen().
func ListenWithContext(ctx context.Context) ListenOption {
	return func(o *listenOptions) error {
		o.ctx = ctx //nolint:fatcontext // we want to provide that context as an option to Listen
		return nil
	}
}

// ListenWithAddress sets the address provided to net.Listen().
func ListenWithAddress(network, address string) ListenOption {
	return func(o *listenOptions) error {
		o.network, o.address = network, address
		return nil
	}
}

// ListenWithKeepAlive sets keepalive period.
func ListenWithKeepAlive(keepAlive time.Duration) ListenOption {
	return func(o *listenOptions) error {
		o.keepAlive = keepAlive
		return nil
	}
}

// ListenWithoutKeepAlive disables the keepalive on the listener.
func ListenWithoutKeepAlive() ListenOption {
	return func(o *listenOptions) error {
		o.keepAlive = -1
		return nil
	}
}

// ListenWithModernTLSConfig sets the tls configuration for tls.NewListener.
// "Modern" is defined based on this website: https://wiki.mozilla.org/Security/Server_Side_TLS.
func ListenWithModernTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ListenOption {
	return func(o *listenOptions) error {
		cfg, err := tlsnetservice.ModernConfig(certFile, keyFile, customizeFunc...)
		if err != nil {
			return fmt.Errorf("unable to create modern tls config: %w", err)
		}
		return ListenWithTLSConfig(cfg)(o)
	}
}

// ListenWithIntermediateTLSConfig sets the tls configuration for tls.NewListener.
// "Intermediate" is defined based on this website: https://wiki.mozilla.org/Security/Server_Side_TLS.
func ListenWithIntermediateTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ListenOption {
	return func(o *listenOptions) error {
		cfg, err := tlsnetservice.IntermediateConfig(certFile, keyFile, customizeFunc...)
		if err != nil {
			return fmt.Errorf("unable to create intermediate tls config: %w", err)
		}
		return ListenWithTLSConfig(cfg)(o)
	}
}

// ListenWithTLSConfig sets the tls configuration for tls.NewListener.
func ListenWithTLSConfig(cfg *tls.Config) ListenOption {
	return func(o *listenOptions) error {
		o.tlsConfig = cfg
		return nil
	}
}

// ListenWithSystemdProvidedFileDescriptors tries to use systemd provided fds if they are provided.
func ListenWithSystemdProvidedFileDescriptors() ListenOption {
	return func(o *listenOptions) error {
		o.useSystemdProvidedFileDescriptor = true
		return nil
	}
}
