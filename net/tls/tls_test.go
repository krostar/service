package tlsnetservice

import (
	"crypto/tls"
	"strings"
	"testing"

	"github.com/krostar/test"
)

func Test_ModernConfig(t *testing.T) {
	cfg, err := ModernConfig("./testdata/cert.crt", "./testdata/cert.key", func(cfg *tls.Config) {
		cfg.ServerName = "foo"
	})
	test.Require(t, err == nil)
	test.Assert(t, cfg != nil, "tls config should not be nil")
	test.Assert(t, len(cfg.Certificates) != 0, "tls config should contain certificates")
	test.Assert(t, cfg.ServerName == "foo", "tls server name should be foo")
	test.Assert(t, cfg.MinVersion == tls.VersionTLS13, "tls config min version should be 1.2")

	_, err = ModernConfig("./dont/exists", "./testdata/cert.key")
	test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to load tls key pair"))
}

func Test_IntermediateConfig(t *testing.T) {
	cfg, err := IntermediateConfig("./testdata/cert.crt", "./testdata/cert.key", func(cfg *tls.Config) {
		cfg.ServerName = "foo"
	})
	test.Require(t, err == nil)
	test.Assert(t, cfg != nil, "tls config should not be nil")
	test.Assert(t, len(cfg.Certificates) != 0, "tls config should contain certificates")
	test.Assert(t, cfg.ServerName == "foo", "tls server name should be foo")
	test.Assert(t, cfg.MinVersion == tls.VersionTLS12, "tls config min version should be 1.2")

	_, err = IntermediateConfig("./dont/exists", "./testdata/cert.key")
	test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to load tls key pair"))
}
