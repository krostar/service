package httpnetservice

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/krostar/test"
	"golang.org/x/sync/errgroup"

	netservice "github.com/krostar/service/net"
)

func Test_NewServer(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		called := false
		srv, err := NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			called = true
		}))
		test.Assert(t, err == nil)
		test.Assert(t, srv.Handler != nil)
		test.Assert(t, srv.ReadTimeout > 0)
		test.Assert(t, srv.MaxHeaderBytes > 1024)

		srv.Handler.ServeHTTP(nil, nil)
		test.Assert(t, called)
	})

	t.Run("ko", func(t *testing.T) {
		anError := errors.New("boom")
		_, err := NewServer(nil, func(*http.Server) error { return anError })
		test.Assert(t, err != nil && errors.Is(err, anError) && strings.Contains(err.Error(), "unable to apply server option"))
	})
}

func Test_Serve(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	l, err := netservice.NewListener(netservice.ListenWithAddress("tcp", "localhost:0"))
	test.Require(t, err == nil && l != nil)
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
	test.Require(t, err == nil && resp != nil)
	test.Assert(t, resp.StatusCode == http.StatusTeapot)

	cancel()
	test.Assert(t, wg.Wait() == nil)
}

func Test_ListenAndServe(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

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

		test.Assert(t, wg.Wait() == nil)
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
				)(t.Context())
			}()

			test.Assert(t, strings.Contains(<-cpanic, "unknown option type"))
		})

		t.Run("unable to create server", func(t *testing.T) {
			anError := errors.New("boom")

			err := ListenAndServe(
				http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusTeapot) }),
				ServerOption(func(*http.Server) error { return anError }),
			)(t.Context())

			test.Assert(t, err != nil && errors.Is(err, anError))
		})

		t.Run("unable to create listener", func(t *testing.T) {
			l, err := netservice.NewListener(netservice.ListenWithAddress("tcp", "localhost:0"))
			test.Require(t, err == nil && l != nil)

			err = ListenAndServe(
				http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusTeapot) }),
				netservice.ListenWithAddress(l.Addr().Network(), l.Addr().String()),
			)(t.Context())
			test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to listen"))
		})
	})
}
