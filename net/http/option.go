package httpnetservice

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"

	tlsnetservice "github.com/krostar/service/net/tls"
)

// ListenAndServeOption allow underlying option to be of type ListenOption or ServeOption.
type ListenAndServeOption any

// ServerOption defines options applier for NewServer.
type ServerOption func(*http.Server) error

// ServerWithErrorLogger sets the error log to the provided logger at error level.
func ServerWithErrorLogger(logger *slog.Logger) ServerOption {
	return func(server *http.Server) error {
		server.ErrorLog = slog.NewLogLogger(logger.Handler(), slog.LevelError)
		return nil
	}
}

// ServerWithModernTLSConfig sets the tls configuration.
// "Modern" is defined based on this website: https://wiki.mozilla.org/Security/Server_Side_TLS.
func ServerWithModernTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ServerOption {
	return func(srv *http.Server) error {
		cfg, err := tlsnetservice.ModernConfig(certFile, keyFile, customizeFunc...)
		if err != nil {
			return fmt.Errorf("unable to create modern tls config: %w", err)
		}
		return ServerWithTLSConfig(cfg)(srv)
	}
}

// ServerWithIntermediateTLSConfig sets the tls configuration.
// "Intermediate" is defined based on this website: https://wiki.mozilla.org/Security/Server_Side_TLS.
func ServerWithIntermediateTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ServerOption {
	return func(srv *http.Server) error {
		cfg, err := tlsnetservice.IntermediateConfig(certFile, keyFile, customizeFunc...)
		if err != nil {
			return fmt.Errorf("unable to create intermediate tls config: %w", err)
		}
		return ServerWithTLSConfig(cfg)(srv)
	}
}

// ServerWithTLSConfig sets the tls configuration.
func ServerWithTLSConfig(cfg *tls.Config) ServerOption {
	return func(srv *http.Server) error {
		srv.TLSConfig = cfg
		return nil
	}
}
