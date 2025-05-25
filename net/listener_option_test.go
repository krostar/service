package netservice

import (
	"crypto/tls"
	"strings"
	"testing"
	"time"

	"github.com/krostar/test"
)

func Test_ListenWithContext(t *testing.T) {
	var o listenOptions
	test.Require(t, ListenWithContext(t.Context())(&o) == nil)
	test.Assert(t, o.ctx != nil)
}

func Test_ListenWithAddress(t *testing.T) {
	var o listenOptions
	test.Require(t, ListenWithAddress("net", "addr")(&o) == nil)
	test.Assert(t, o.network == "net")
	test.Assert(t, o.address == "addr")
}

func Test_ListenWithKeepAlive(t *testing.T) {
	var o listenOptions
	test.Require(t, ListenWithKeepAlive(3*time.Millisecond)(&o) == nil)
	test.Assert(t, o.keepAlive == 3*time.Millisecond)
}

func Test_ListenWithoutKeepAlive(t *testing.T) {
	var o listenOptions
	o.keepAlive = 10 * time.Second
	test.Require(t, ListenWithoutKeepAlive()(&o) == nil)
	test.Assert(t, o.keepAlive == time.Duration(-1))
}

func Test_ListenWithModernTLSConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var o listenOptions
		test.Assert(t, ListenWithModernTLSConfig("./tls/testdata/cert.crt", "./tls/testdata/cert.key", func(cfg *tls.Config) {
			cfg.ServerName = "foo"
		})(&o) == nil)
		test.Assert(t, o.tlsConfig != nil)
		test.Assert(t, o.tlsConfig.MinVersion == tls.VersionTLS13)
	})

	t.Run("ko", func(t *testing.T) {
		var o listenOptions
		err := ListenWithModernTLSConfig("./notfound", "./notfound")(&o)
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to create modern tls config"))
	})
}

func Test_ListenWithIntermediateTLSConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var o listenOptions
		test.Assert(t, ListenWithIntermediateTLSConfig("./tls/testdata/cert.crt", "./tls/testdata/cert.key", func(cfg *tls.Config) {
			cfg.ServerName = "foo"
		})(&o) == nil)
		test.Assert(t, o.tlsConfig != nil)
		test.Assert(t, o.tlsConfig.MinVersion == tls.VersionTLS12)
	})

	t.Run("ko", func(t *testing.T) {
		var o listenOptions
		err := ListenWithIntermediateTLSConfig("./notfound", "./notfound")(&o)
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to create intermediate tls config"))
	})
}

func Test_ListenWithTLSConfig(t *testing.T) {
	var o listenOptions
	test.Assert(t, ListenWithTLSConfig(&tls.Config{ServerName: "foo", MinVersion: tls.VersionTLS12})(&o) == nil)
	test.Assert(t, o.tlsConfig != nil, "tls config should not be nil")
	test.Assert(t, o.tlsConfig.ServerName == "foo")
}

func Test_ListenWithSystemdProvidedFileDescriptors(t *testing.T) {
	var o listenOptions
	err := ListenWithSystemdProvidedFileDescriptors()(&o)
	test.Assert(t, err == nil)
	test.Assert(t, o.useSystemdProvidedFileDescriptor)
}
