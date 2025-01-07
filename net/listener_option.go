package netservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"
)

type listenerOptions struct {
	ctx context.Context //nolint:containedctx // ctx here is used for the Listen function and provided as an option

	network string
	address string

	keepAlive time.Duration
	tlsConfig *tls.Config

	useSystemdProvidedFileDescriptor bool
}

// ListenerOption defines options applier for the listener.
type ListenerOption func(*listenerOptions) error

// ListenWithContext sets the context provided to net.Listen().
func ListenWithContext(ctx context.Context) ListenerOption {
	return func(o *listenerOptions) error {
		o.ctx = ctx //nolint:fatcontext // we want to provide that context as an option to Listen
		return nil
	}
}

// ListenWithAddress sets the address provided to net.Listen().
func ListenWithAddress(network, address string) ListenerOption {
	return func(o *listenerOptions) error {
		o.network, o.address = network, address
		return nil
	}
}

// ListenWithKeepAlive sets keepalive period.
func ListenWithKeepAlive(keepAlive time.Duration) ListenerOption {
	return func(o *listenerOptions) error {
		o.keepAlive = keepAlive
		return nil
	}
}

// ListenWithoutKeepAlive disables the keepalive on the listener.
func ListenWithoutKeepAlive() ListenerOption {
	return func(o *listenerOptions) error {
		o.keepAlive = -1
		return nil
	}
}

// ListenWithModernTLSConfig sets the tls configuration for tls.NewListener.
// "Modern" is defined based on this website: https://wiki.mozilla.org/Security/Server_Side_TLS.
func ListenWithModernTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ListenerOption {
	return func(o *listenerOptions) error {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("unable to load tls key pair: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates:     []tls.Certificate{cert},
			MinVersion:       tls.VersionTLS13,
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
		}

		for _, f := range customizeFunc {
			f(tlsConfig)
		}

		return ListenWithTLSConfig(tlsConfig)(o)
	}
}

// ListenWithIntermediateTLSConfig sets the tls configuration for tls.NewListener.
// "Intermediate" is defined based on this website: https://wiki.mozilla.org/Security/Server_Side_TLS.
func ListenWithIntermediateTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ListenerOption {
	return func(o *listenerOptions) error {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("unable to load tls key pair: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			},
			MinVersion:       tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
		}

		for _, f := range customizeFunc {
			f(tlsConfig)
		}

		return ListenWithTLSConfig(tlsConfig)(o)
	}
}

// ListenWithTLSConfig sets the tls configuration for tls.NewListener.
func ListenWithTLSConfig(cfg *tls.Config) ListenerOption {
	return func(o *listenerOptions) error {
		o.tlsConfig = cfg
		return nil
	}
}

// ListenWithSystemdProvidedFileDescriptors tries to use systemd provided fds if they are provided.
func ListenWithSystemdProvidedFileDescriptors() ListenerOption {
	return func(o *listenerOptions) error {
		o.useSystemdProvidedFileDescriptor = true
		return nil
	}
}
