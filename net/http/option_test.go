package httpnetservice

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
)

func Test_ServerWithErrorLogger(t *testing.T) {
	var srv http.Server
	assert.NilError(t, ServerWithErrorLogger(slog.Default())(&srv))
	assert.Check(t, srv.ErrorLog != nil)
}

func Test_ServerWithModernTLSConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var srv http.Server
		assert.NilError(t, ServerWithModernTLSConfig("../tls/testdata/cert.crt", "../tls/testdata/cert.key", func(cfg *tls.Config) {
			cfg.ServerName = "foo"
		})(&srv))
		assert.Check(t, srv.TLSConfig != nil)
		assert.Check(t, srv.TLSConfig.MinVersion == tls.VersionTLS13)
	})

	t.Run("ko", func(t *testing.T) {
		var srv http.Server
		assert.ErrorContains(t, ServerWithModernTLSConfig("./notfound", "./notfound")(&srv), "unable to create modern tls config")
	})
}

func Test_ServerWithIntermediateTLSConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var srv http.Server
		assert.NilError(t, ServerWithIntermediateTLSConfig("../tls/testdata/cert.crt", "../tls/testdata/cert.key", func(cfg *tls.Config) {
			cfg.ServerName = "foo"
		})(&srv))
		assert.Check(t, srv.TLSConfig != nil)
		assert.Check(t, srv.TLSConfig.MinVersion == tls.VersionTLS12)
	})

	t.Run("ko", func(t *testing.T) {
		var srv http.Server
		assert.ErrorContains(t, ServerWithIntermediateTLSConfig("./notfound", "./notfound")(&srv), "unable to create intermediate tls config")
	})
}

func Test_ServerWithTLSConfig(t *testing.T) {
	var srv http.Server
	assert.NilError(t, ServerWithTLSConfig(&tls.Config{ServerName: "foo", MinVersion: tls.VersionTLS12})(&srv))
	assert.Check(t, srv.TLSConfig != nil, "tls config should not be nil")
	assert.Check(t, srv.TLSConfig.ServerName == "foo")
}
