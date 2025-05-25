// Package tlsnetservice provides TLS configuration utilities for secure network services.
//
// This package offers pre-configured TLS settings that follow Mozilla's TLS configuration
// recommendations for modern and intermediate compatibility. It simplifies the process
// of setting up secure TLS connections with appropriate cipher suites, protocol versions,
// and security parameters.
//
// The package provides two main security configuration levels:
//   - Modern: TLS 1.3 only with the strongest available ciphers
//   - Intermediate: TLS 1.2+ with broad compatibility while maintaining security
//
// For more information about TLS best practices, see:
// https://wiki.mozilla.org/Security/Server_Side_TLS
package tlsnetservice

import (
	"crypto/tls"
	"fmt"
)

// ModernConfig creates a TLS configuration optimized for modern security standards.
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
func ModernConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load tls key pair: %w", err)
	}

	cfg := &tls.Config{
		Certificates:     []tls.Certificate{cert},
		MinVersion:       tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
	}

	for _, f := range customizeFunc {
		f(cfg)
	}

	return cfg, nil
}

// IntermediateConfig creates a TLS configuration balancing security and compatibility.
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
func IntermediateConfig(certFile, keyFile string, customizeFunc ...func(*tls.Config)) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load tls key pair: %w", err)
	}

	cfg := &tls.Config{
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
		f(cfg)
	}

	return cfg, nil
}
