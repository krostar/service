package netservice

import (
	"testing"
	"time"

	"github.com/krostar/test"
)

func Test_ServeWithGracefulTimeout(t *testing.T) {
	var o serveOptions
	ServeWithShutdownTimeout(time.Second)(&o)
	test.Assert(t, o.shutdownTimeout == time.Second)
}

func Test_ServeWithServeErrorTransformer(t *testing.T) {
	var o serveOptions
	ServeWithServeErrorTransformer(func(err error) error {
		return err
	})(&o)
	test.Assert(t, o.serveErrorTransformer != nil)
	test.Assert(t, o.shutdownErrorTransformer == nil)
}

func Test_ServeWithShutdownErrorTransformer(t *testing.T) {
	var o serveOptions
	ServeWithShutdownErrorTransformer(func(err error) error {
		return err
	})(&o)
	test.Assert(t, o.serveErrorTransformer == nil)
	test.Assert(t, o.shutdownErrorTransformer != nil)
}
