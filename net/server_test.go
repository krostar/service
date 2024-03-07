package netservice

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

func Test_Serve(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		l, err := NewListener(ctx, "localhost:0")
		assert.NilError(t, err)
		addr := l.Addr().String()

		var wg errgroup.Group
		wg.Go(func() error {
			srv := newServer(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusTeapot)
			})
			return Serve(srv, l, ServerWithServeErrorTransformer(func(err error) error {
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			}))(ctx)
		})

		resp, err := http.DefaultClient.Get("http://" + addr)
		assert.NilError(t, err)
		assert.Equal(t, resp.StatusCode, http.StatusTeapot)

		cancel()
		assert.NilError(t, wg.Wait())
	})

	t.Run("unable to serve", func(t *testing.T) {
		ctx := context.Background()

		srv := newServer(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})

		l, err := NewListener(ctx, "localhost:0")
		assert.NilError(t, err)

		assert.ErrorContains(t, Serve(srv, listenerFail{Listener: l})(ctx), "unable to serve listener")
	})

	t.Run("unable to shutdown", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		srv := newServer(func(_ http.ResponseWriter, r *http.Request) {
			cancel()
			reqCtx, cancelReqCtx := context.WithTimeout(r.Context(), time.Second*2)
			defer cancelReqCtx()
			<-reqCtx.Done()
		})

		l, err := NewListener(context.Background(), "localhost:0")
		assert.NilError(t, err)

		var wg errgroup.Group
		wg.Go(func() error {
			err := Serve(srv, l, ServerWithShutdownTimeout(time.Second))(ctx)
			return err
		})
		wg.Go(func() error {
			_, err := http.DefaultClient.Get("http://" + l.Addr().String())
			return err
		})

		assert.ErrorContains(t, wg.Wait(), "unable to shut server down")
	})
}

type listenerFail struct {
	net.Listener
}

func (listenerFail) Accept() (net.Conn, error) { return nil, errors.New("boom") }

func newServer(handler http.HandlerFunc) Server {
	return &http.Server{Handler: handler}
}
