package netservice

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func Test_ServerWithGracefulTimeout(t *testing.T) {
	var o serveOptions
	ServerWithShutdownTimeout(time.Second)(&o)
	assert.Equal(t, o.shutdownTimeout, time.Second)
}

func Test_ServerWithServeErrorTransformer(t *testing.T) {
	var o serveOptions
	ServerWithServeErrorTransformer(func(err error) error {
		return err
	})(&o)
	assert.Check(t, o.serveErrorTransformer != nil)
	assert.Check(t, o.shutdownErrorTransformer == nil)
}

func Test_ServerWithShutdownErrorTransformer(t *testing.T) {
	var o serveOptions
	ServerWithShutdownErrorTransformer(func(err error) error {
		return err
	})(&o)
	assert.Check(t, o.serveErrorTransformer == nil)
	assert.Check(t, o.shutdownErrorTransformer != nil)
}
