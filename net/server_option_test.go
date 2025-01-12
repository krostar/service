package netservice

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func Test_ServeWithGracefulTimeout(t *testing.T) {
	var o serveOptions
	ServeWithShutdownTimeout(time.Second)(&o)
	assert.Equal(t, o.shutdownTimeout, time.Second)
}

func Test_ServeWithServeErrorTransformer(t *testing.T) {
	var o serveOptions
	ServeWithServeErrorTransformer(func(err error) error {
		return err
	})(&o)
	assert.Check(t, o.serveErrorTransformer != nil)
	assert.Check(t, o.shutdownErrorTransformer == nil)
}

func Test_ServeWithShutdownErrorTransformer(t *testing.T) {
	var o serveOptions
	ServeWithShutdownErrorTransformer(func(err error) error {
		return err
	})(&o)
	assert.Check(t, o.serveErrorTransformer == nil)
	assert.Check(t, o.shutdownErrorTransformer != nil)
}
