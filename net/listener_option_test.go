package netservice

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func Test_ListenWithContext(t *testing.T) {
	var o listenOptions
	err := ListenWithContext(context.Background())(&o)
	assert.NilError(t, err)
	assert.Check(t, o.ctx != nil)
}

func Test_ListenWithAddress(t *testing.T) {
	var o listenOptions
	err := ListenWithAddress("net", "addr")(&o)
	assert.NilError(t, err)
	assert.Equal(t, "net", o.network)
	assert.Equal(t, "addr", o.address)
}

func Test_ListenWithKeepAlive(t *testing.T) {
	var o listenOptions
	err := ListenWithKeepAlive(3 * time.Millisecond)(&o)
	assert.NilError(t, err)
	assert.Equal(t, 3*time.Millisecond, o.keepAlive)
}

func Test_ListenWithoutKeepAlive(t *testing.T) {
	var o listenOptions
	o.keepAlive = 10 * time.Second
	err := ListenWithoutKeepAlive()(&o)
	assert.NilError(t, err)
	assert.Equal(t, time.Duration(-1), o.keepAlive)
}

func Test_ListenWithModernTLSConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var o listenOptions
		assert.NilError(t, ListenWithModernTLSConfig("./tls/testdata/cert.crt", "./tls/testdata/cert.key", func(cfg *tls.Config) {
			cfg.ServerName = "foo"
		})(&o))
		assert.Check(t, o.tlsConfig != nil)
		assert.Check(t, o.tlsConfig.MinVersion == tls.VersionTLS13)
	})

	t.Run("ko", func(t *testing.T) {
		var o listenOptions
		assert.ErrorContains(t, ListenWithModernTLSConfig("./notfound", "./notfound")(&o), "unable to create modern tls config")
	})
}

func Test_ListenWithIntermediateTLSConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var o listenOptions
		assert.NilError(t, ListenWithIntermediateTLSConfig("./tls/testdata/cert.crt", "./tls/testdata/cert.key", func(cfg *tls.Config) {
			cfg.ServerName = "foo"
		})(&o))
		assert.Check(t, o.tlsConfig != nil)
		assert.Check(t, o.tlsConfig.MinVersion == tls.VersionTLS12)
	})

	t.Run("ko", func(t *testing.T) {
		var o listenOptions
		assert.ErrorContains(t, ListenWithIntermediateTLSConfig("./notfound", "./notfound")(&o), "unable to create intermediate tls config")
	})
}

func Test_ListenWithTLSConfig(t *testing.T) {
	var o listenOptions
	assert.NilError(t, ListenWithTLSConfig(&tls.Config{ServerName: "foo", MinVersion: tls.VersionTLS12})(&o))
	assert.Check(t, o.tlsConfig != nil, "tls config should not be nil")
	assert.Check(t, o.tlsConfig.ServerName == "foo")
}

func Test_ListenWithSystemdProvidedFileDescriptors(t *testing.T) {
	var o listenOptions
	err := ListenWithSystemdProvidedFileDescriptors()(&o)
	assert.NilError(t, err)
	assert.Check(t, o.useSystemdProvidedFileDescriptor)
}
