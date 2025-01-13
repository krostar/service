package tlsnetservice

import (
	"crypto/tls"
	"testing"

	"gotest.tools/v3/assert"
)

func Test_ModernConfig(t *testing.T) {
	cfg, err := ModernConfig("./testdata/cert.crt", "./testdata/cert.key", func(cfg *tls.Config) {
		cfg.ServerName = "foo"
	})
	assert.NilError(t, err)
	assert.Check(t, cfg != nil, "tls config should not be nil")
	assert.Check(t, len(cfg.Certificates) != 0, "tls config should contain certificates")
	assert.Check(t, cfg.ServerName == "foo", "tls server name should be foo")
	assert.Check(t, cfg.MinVersion == tls.VersionTLS13, "tls config min version should be 1.2")

	_, err = ModernConfig("./dont/exists", "./testdata/cert.key")
	assert.ErrorContains(t, err, "unable to load tls key pair")
}

func Test_IntermediateConfig(t *testing.T) {
	cfg, err := IntermediateConfig("./testdata/cert.crt", "./testdata/cert.key", func(cfg *tls.Config) {
		cfg.ServerName = "foo"
	})
	assert.NilError(t, err)
	assert.Check(t, cfg != nil, "tls config should not be nil")
	assert.Check(t, len(cfg.Certificates) != 0, "tls config should contain certificates")
	assert.Check(t, cfg.ServerName == "foo", "tls server name should be foo")
	assert.Check(t, cfg.MinVersion == tls.VersionTLS12, "tls config min version should be 1.2")

	_, err = IntermediateConfig("./dont/exists", "./testdata/cert.key")
	assert.ErrorContains(t, err, "unable to load tls key pair")
}
