package tlsnetservice

import (
	"crypto/tls"
	"fmt"
)

// ModernConfig creates a tls configuration.
// "Modern" is defined based on this website: https://wiki.mozilla.org/Security/Server_Side_TLS.
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

// IntermediateConfig creates a tls configuration.
// "Intermediate" is defined based on this website: https://wiki.mozilla.org/Security/Server_Side_TLS.
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
