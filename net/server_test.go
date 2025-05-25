package netservice

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/krostar/test"
	"golang.org/x/sync/errgroup"
)

func Test_Serve(t *testing.T) {
	client := &http.Client{Timeout: time.Millisecond * 500}

	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"))
		test.Assert(t, err == nil)
		addr := l.Addr().String()

		var wg errgroup.Group
		wg.Go(func() error {
			srv := newServer(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusTeapot)
			})
			return Serve(srv, l, ServeWithServeErrorTransformer(func(err error) error {
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			}))(ctx)
		})

		resp, err := client.Get("http://" + addr)
		test.Assert(t, err == nil)
		test.Assert(t, resp.StatusCode == http.StatusTeapot)

		cancel()
		test.Assert(t, wg.Wait() == nil)
	})

	t.Run("unable to serve", func(t *testing.T) {
		ctx := t.Context()

		srv := newServer(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})

		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"))
		test.Assert(t, err == nil)

		err = Serve(srv, listenerFail{Listener: l})(ctx)
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to serve listener"))
	})

	t.Run("unable to shutdown", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		srv := newServer(func(_ http.ResponseWriter, r *http.Request) {
			cancel()
			reqCtx, cancelReqCtx := context.WithTimeout(r.Context(), time.Minute)
			defer cancelReqCtx()
			<-reqCtx.Done()
		})

		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"))
		test.Assert(t, err == nil)

		serverErr := make(chan error)
		go func() {
			serverErr <- Serve(srv, l, ServeWithShutdownTimeout(time.Millisecond*100))(ctx)
		}()

		reqErr := make(chan error)
		go func() {
			_, err := client.Get("http://" + l.Addr().String())
			reqErr <- err
		}()

		test.Assert(t, func() bool {
			err := <-reqErr
			return err != nil && errors.Is(err, context.DeadlineExceeded)
		}())
		test.Assert(t, func() bool {
			err := <-serverErr
			return err != nil && strings.Contains(err.Error(), "unable to shut server down")
		}())
	})
}

type listenerFail struct {
	net.Listener
}

func (listenerFail) Accept() (net.Conn, error) { return nil, errors.New("boom") }

func newServer(handler http.HandlerFunc) Server {
	return &http.Server{Handler: handler}
}
