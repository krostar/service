package httpnetservice

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	netservice "github.com/krostar/service/net"
)

func Test_NewServer(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		called := false
		srv, err := NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			called = true
		}))
		assert.Check(t, err)
		assert.Check(t, srv.Handler != nil)
		assert.Check(t, srv.ReadTimeout > 0)
		assert.Check(t, srv.MaxHeaderBytes > 1024)

		srv.Handler.ServeHTTP(nil, nil)
		assert.Check(t, called)
	})

	t.Run("ko", func(t *testing.T) {
		anError := errors.New("boom")
		_, err := NewServer(nil, func(*http.Server) error {
			return anError
		})
		assert.Check(t, errors.Is(err, anError))
		assert.ErrorContains(t, err, "unable to apply server option")
	})
}

func Test_Serve(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	l, err := netservice.NewListener(netservice.ListenWithAddress("tcp", "localhost:0"))
	assert.NilError(t, err)
	addr := l.Addr().String()

	var wg errgroup.Group
	wg.Go(func() error {
		srv := &http.Server{Handler: http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})}
		return Serve(srv, l)(ctx)
	})

	client := &http.Client{Timeout: time.Millisecond * 500}
	resp, err := client.Get("http://" + addr)
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusTeapot)

	cancel()
	assert.NilError(t, wg.Wait())
}

func Test_ListenAndServe(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		var wg errgroup.Group
		wg.Go(func() error {
			return ListenAndServe(
				http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusTeapot) }),
				netservice.ListenWithAddress("tcp", "localhost:0"),
				netservice.ServeWithShutdownTimeout(time.Second),
				ServerWithModernTLSConfig("../tls/testdata/cert.crt", "../tls/testdata/cert.key"),
			)(ctx)
		})

		time.Sleep(time.Millisecond * 100)
		cancel()

		assert.NilError(t, wg.Wait())
	})

	t.Run("ko", func(t *testing.T) {
		t.Run("invalid option type", func(t *testing.T) {
			cpanic := make(chan string, 1)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						cpanic <- r.(string)
					} else {
						cpanic <- ""
					}
				}()
				_ = ListenAndServe(
					http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusTeapot) }),
					42,
				)(context.Background())
			}()

			assert.Check(t, strings.Contains(<-cpanic, "unknown option type"))
		})

		t.Run("unable to create server", func(t *testing.T) {
			anError := errors.New("boom")

			err := ListenAndServe(
				http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusTeapot) }),
				ServerOption(func(*http.Server) error { return anError }),
			)(context.Background())

			assert.Check(t, errors.Is(err, anError))
		})

		t.Run("unable to create listener", func(t *testing.T) {
			l, err := netservice.NewListener(netservice.ListenWithAddress("tcp", "localhost:0"))
			assert.NilError(t, err)

			err = ListenAndServe(
				http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusTeapot) }),
				netservice.ListenWithAddress(l.Addr().Network(), l.Addr().String()),
			)(context.Background())
			assert.ErrorContains(t, err, "unable to listen")
		})
	})
}
