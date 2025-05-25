// Package httpnetservice provides HTTP server configuration options for the service library.
//
// This package extends the HTTP server functionality with convenient configuration
// options that integrate with the service lifecycle management. It provides options
// for server timeouts, TLS configuration, error logging, and other HTTP-specific
// settings that are commonly needed in production applications.
//
// # Server Configuration Options
//
// The package provides ServerOption functions that configure various aspects of
// HTTP servers:
//   - Timeout settings (read, write, idle)
//   - TLS configuration with security presets
//   - Error logging integration
//   - Custom TLS configurations
package httpnetservice

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"

	tlsnetservice "github.com/krostar/service/net/tls"
)

// ListenAndServeOption represents configuration options for HTTP ListenAndServe operations.
//
// This type allows the ListenAndServe function to accept a mix of different option types:
//   - ServerOption: configure HTTP server behavior (timeouts, TLS, logging)
//   - netservice.ListenOption: configure listener creation (address, TLS, keep-alive)
//   - netservice.ServeOption: configure server lifecycle (shutdown timeout, error handling)
//
// Type safety: while this uses any (interface{}) for flexibility, the implementation performs
// runtime type checking to ensure only valid option types are accepted. Invalid
// types will cause a panic with a descriptive error message.
type ListenAndServeOption any

// ServerOption defines a function that configures an HTTP server.
//
// ServerOptions follow the functional options pattern, allowing flexible and
// extensible configuration of HTTP servers. Each option can modify specific
// aspects of the server configuration and can return an error if the
// configuration is invalid.
type ServerOption func(*http.Server) error

// ServerWithErrorLogger configures HTTP server error logging using structured logging.
//
// This option integrates the HTTP server's error logging with Go's structured
// logging (slog) framework, providing consistent log formatting and better
// observability for server errors.
//
// The HTTP server will log errors at the ERROR level using the provided slog.Logger:
//   - connection errors and timeouts
//   - TLS handshake failures
//   - protocol violations
//   - internal server errors
//
// If no error logger is configured, the HTTP server uses Go's default
// log package, which may not integrate well with your application's
// logging strategy.
func ServerWithErrorLogger(logger *slog.Logger) ServerOption {
	return func(server *http.Server) error {
		server.ErrorLog = slog.NewLogLogger(logger.Handler(), slog.LevelError)
		return nil
	}
}

// ServerWithModernTLSConfig configures the HTTP server with modern TLS security settings.
//
// This option applies Mozilla's "Modern" TLS configuration to the HTTP server,
// providing the highest level of security by using only TLS 1.3 with the
// strongest available cipher suites. This configuration prioritizes security
// over compatibility.
//
// Modern TLS configuration provides:
//   - TLS 1.3 only (no fallback to older versions)
//   - strongest available cipher suites
//   - perfect Forward Secrecy (PFS)
//   - protection against downgrade attacks
//   - modern elliptic curves
//
// Compatible with modern clients only, not compatible with:
//   - Internet Explorer (any version)
//   - Chrome < 70, Firefox < 63, Safari < 12.1
//   - Legacy systems and older TLS libraries
//
// Choose ModernTLS when:
//   - You control the client environment
//   - You can require modern TLS support
//   - Building new APIs for modern applications
//
// For broader compatibility, use ServerWithIntermediateTLSConfig instead.
//
// Based on Mozilla's Server Side TLS recommendations:
// https://wiki.mozilla.org/Security/Server_Side_TLS
func ServerWithModernTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ServerOption {
	return func(srv *http.Server) error {
		cfg, err := tlsnetservice.ModernConfig(certFile, keyFile, customizeFunc...)
		if err != nil {
			return fmt.Errorf("unable to create modern tls config: %w", err)
		}
		return ServerWithTLSConfig(cfg)(srv)
	}
}

// ServerWithIntermediateTLSConfig configures the HTTP server with intermediate TLS security settings.
//
// This option applies Mozilla's "Intermediate" TLS configuration to the HTTP server,
// balancing security and compatibility. It provides strong security while maintaining
// compatibility with a broader range of clients including older browsers and legacy systems.
//
// Intermediate TLS configuration provides:
//   - TLS 1.2+ support (includes TLS 1.3 when available)
//   - Carefully selected cipher suites prioritizing AEAD ciphers
//   - Forward secrecy for all supported cipher suites
//   - Protection against known TLS vulnerabilities
//   - Modern elliptic curves
//
// Compatible with most clients:
//   - All modern browsers (Chrome 27+, Firefox 10+, Safari 7+)
//   - Internet Explorer 11 on Windows 7+
//   - Most mobile browsers and applications
//   - Legacy systems with TLS 1.2 support
//   - Load balancers and proxies
//
// Choose IntermediateTLS when:
//   - You need to support a wide range of clients
//   - Legacy browser support is required
//   - You're migrating from clear protocols gradually
//   - You're unsure about your client requirements
//   - Building public-facing web services
//
// For maximum security with modern clients only, use ServerWithModernTLSConfig instead.
//
// Based on Mozilla's Server Side TLS recommendations:
// https://wiki.mozilla.org/Security/Server_Side_TLS
func ServerWithIntermediateTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ServerOption {
	return func(srv *http.Server) error {
		cfg, err := tlsnetservice.IntermediateConfig(certFile, keyFile, customizeFunc...)
		if err != nil {
			return fmt.Errorf("unable to create intermediate tls config: %w", err)
		}
		return ServerWithTLSConfig(cfg)(srv)
	}
}

// ServerWithTLSConfig sets a custom TLS configuration for the HTTP server.
//
// This option allows you to provide a fully customized tls.Config for maximum
// control over TLS behavior. Use this when the predefined Modern or Intermediate
// configurations don't meet your specific security or compatibility requirements.
//
// The provided tls.Config is used as-is without validation. Ensure:
//   - at least one certificate is configured or GetCertificate is set
//   - TLS version ranges are sensible (MinVersion <= MaxVersion)
//   - cipher suites are appropriate for your security requirements
//   - client authentication settings match your application needs
func ServerWithTLSConfig(cfg *tls.Config) ServerOption {
	return func(srv *http.Server) error {
		srv.TLSConfig = cfg
		return nil
	}
}
