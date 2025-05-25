package httpnetservice

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/krostar/test"
)

func Test_ServerWithErrorLogger(t *testing.T) {
	var srv http.Server
	test.Assert(t, ServerWithErrorLogger(slog.Default())(&srv) == nil)
	test.Assert(t, srv.ErrorLog != nil)
}

func Test_ServerWithModernTLSConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var srv http.Server
		test.Assert(t, ServerWithModernTLSConfig("../tls/testdata/cert.crt", "../tls/testdata/cert.key", func(cfg *tls.Config) {
			cfg.ServerName = "foo"
		})(&srv) == nil)
		test.Assert(t, srv.TLSConfig != nil)
		test.Assert(t, srv.TLSConfig.MinVersion == tls.VersionTLS13)
	})

	t.Run("ko", func(t *testing.T) {
		var srv http.Server
		err := ServerWithModernTLSConfig("./notfound", "./notfound")(&srv)
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to create modern tls config"))
	})
}

func Test_ServerWithIntermediateTLSConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var srv http.Server
		test.Assert(t, ServerWithIntermediateTLSConfig("../tls/testdata/cert.crt", "../tls/testdata/cert.key", func(cfg *tls.Config) {
			cfg.ServerName = "foo"
		})(&srv) == nil)
		test.Assert(t, srv.TLSConfig != nil)
		test.Assert(t, srv.TLSConfig.MinVersion == tls.VersionTLS12)
	})

	t.Run("ko", func(t *testing.T) {
		var srv http.Server
		err := ServerWithIntermediateTLSConfig("./notfound", "./notfound")(&srv)
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to create intermediate tls config"))
	})
}

func Test_ServerWithTLSConfig(t *testing.T) {
	var srv http.Server
	test.Assert(t, ServerWithTLSConfig(&tls.Config{ServerName: "foo", MinVersion: tls.VersionTLS12})(&srv) == nil)
	test.Assert(t, srv.TLSConfig != nil, "tls config should not be nil")
	test.Assert(t, srv.TLSConfig.ServerName == "foo")
}
