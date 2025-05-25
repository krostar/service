package netservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	tlsnetservice "github.com/krostar/service/net/tls"
)

type listenOptions struct {
	// ctx is the context used for listener creation operations.
	// It controls timeouts and cancellation for the listening process.
	ctx context.Context //nolint:containedctx // ctx here is used for the Listen function and provided as an option

	// network specifies the network type (e.g., "tcp", "tcp4", "tcp6", "unix").
	// This determines the addressing scheme and protocol used.
	network string

	// address specifies the local address to bind to.
	// Format depends on the network type (e.g., ":8080", "127.0.0.1:8080", "/tmp/socket").
	address string

	// keepAlive configures TCP keep-alive behavior for the listener.
	// Positive values enable keep-alive with the specified interval.
	// Zero uses system defaults. Negative values disable keep-alive.
	keepAlive time.Duration

	// tlsConfig holds TLS configuration for secure connections.
	// When set, the listener will automatically wrap connections with TLS.
	tlsConfig *tls.Config

	// useSystemdProvidedFileDescriptor indicates whether to use systemd socket activation.
	// When true, the listener will attempt to use file descriptors provided by systemd.
	useSystemdProvidedFileDescriptor bool
}

// ListenOption defines a function that configures listener creation options.
//
// ListenOptions follow the functional options pattern, allowing flexible and
// extensible configuration of network listeners. Options can be combined and
// applied in any order to customize listener behavior.
//
// ListenOptions can return errors for invalid configurations. These errors
// are propagated during listener creation, allowing early detection of
// configuration problems.
type ListenOption func(*listenOptions) error

// ListenWithContext sets the context used for listener creation and lifecycle management.
//
// The context is typically derived from the application's root context or
// a service-specific context that can be canceled during shutdown.
//
// The context is used only during listener creation. Once the listener is
// created and returned, the context is no longer used. For controlling the
// lifecycle of the server that uses this listener, pass the context to
// the service.Run() function instead.
func ListenWithContext(ctx context.Context) ListenOption {
	return func(o *listenOptions) error {
		o.ctx = ctx
		return nil
	}
}

// ListenWithAddress sets the network type and address for the listener.
//
// This option configures where and how the listener will accept connections.
// The network parameter determines the protocol and addressing scheme, while
// the address parameter specifies the exact location to bind to.
func ListenWithAddress(network, address string) ListenOption {
	return func(o *listenOptions) error {
		o.network, o.address = network, address
		return nil
	}
}

// ListenWithKeepAlive configures TCP keep-alive behavior for the listener.
//
// TCP keep-alive is a mechanism to detect dead connections and prevent
// resource leaks. It sends periodic probe packets to verify that the
// remote peer is still reachable and responsive.
//
// When enabled, TCP keep-alive:
//   - sends probe packets at the specified interval
//   - detects unresponsive connections automatically
//   - allows the system to clean up dead connections
//   - helps prevent connection exhaustion attacks
//
// Keep-alive behavior may vary slightly between operating systems:
//   - Linux: Uses SO_KEEPALIVE with configurable parameters
//   - Windows: Uses SO_KEEPALIVE with system defaults
//   - macOS: Uses SO_KEEPALIVE with BSD-style configuration
func ListenWithKeepAlive(keepAlive time.Duration) ListenOption {
	return func(o *listenOptions) error {
		o.keepAlive = keepAlive
		return nil
	}
}

// ListenWithoutKeepAlive explicitly disables TCP keep-alive for the listener.
//
// This option turns off TCP keep-alive probes, meaning the system will not
// automatically detect or clean up dead connections. Use this when:
//   - you have application-level keep-alive mechanisms
//   - connection lifetime is very short (request-response patterns)
//   - you want to minimize network overhead
//   - testing scenarios where you need predictable connection behavior
//
// Disabling keep-alive can lead to:
//   - connection leaks if clients disconnect ungracefully
//   - resource exhaustion under certain failure scenarios
//   - slower recovery from transient network issues
func ListenWithoutKeepAlive() ListenOption {
	return func(o *listenOptions) error {
		o.keepAlive = -1
		return nil
	}
}

// ListenWithModernTLSConfig configures the listener with modern TLS security settings.
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
func ListenWithModernTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ListenOption {
	return func(o *listenOptions) error {
		cfg, err := tlsnetservice.ModernConfig(certFile, keyFile, customizeFunc...)
		if err != nil {
			return fmt.Errorf("unable to create modern tls config: %w", err)
		}
		return ListenWithTLSConfig(cfg)(o)
	}
}

// ListenWithIntermediateTLSConfig configures the listener with intermediate TLS security settings.
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
func ListenWithIntermediateTLSConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) ListenOption {
	return func(o *listenOptions) error {
		cfg, err := tlsnetservice.IntermediateConfig(certFile, keyFile, customizeFunc...)
		if err != nil {
			return fmt.Errorf("unable to create intermediate tls config: %w", err)
		}
		return ListenWithTLSConfig(cfg)(o)
	}
}

// ListenWithTLSConfig sets a custom TLS configuration for the listener.
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
func ListenWithTLSConfig(cfg *tls.Config) ListenOption {
	return func(o *listenOptions) error {
		o.tlsConfig = cfg
		return nil
	}
}

// ListenWithSystemdProvidedFileDescriptors enables systemd socket activation support.
//
// When enabled, the listener will attempt to use file descriptors provided by systemd
// through socket activation instead of creating new sockets. This enables:
//   - zero-downtime deployments and restarts
//   - privilege separation (systemd creates privileged sockets)
//   - consistent socket configuration across service restarts
//   - integration with systemd's dependency management
//
// How socket activation works:
// 1. systemd creates and binds the socket before starting the service
// 2. systemd passes file descriptors to the service via environment variables
// 3. the service inherits the already-bound socket instead of creating a new one
// 4. systemd can restart the service without dropping connections
//
// systemd sets these environment variables:
//   - LISTEN_PID: Process ID that should inherit the descriptors
//   - LISTEN_FDS: Number of file descriptors passed
//   - LISTEN_FDNAMES: Names of the file descriptors (optional)
//
// If systemd file descriptors are not available:
//   - the listener falls back to using ListenWithAddress settings
//   - no error is returned (graceful degradation)
func ListenWithSystemdProvidedFileDescriptors() ListenOption {
	return func(o *listenOptions) error {
		o.useSystemdProvidedFileDescriptor = true
		return nil
	}
}
