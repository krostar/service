package netservice

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krostar/test"
	"golang.org/x/sync/errgroup"
)

func Test_NewListener(t *testing.T) {
	t.Run("no configuration", func(t *testing.T) {
		l, err := NewListener()
		test.Assert(t, err != nil && strings.Contains(err.Error(), "no listener configured"))
		test.Assert(t, l == nil)
	})

	t.Run("no tls", func(t *testing.T) {
		var l net.Listener
		{
			var err error
			l, err = NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithSystemdProvidedFileDescriptors())
			test.Require(t, err == nil, "%v", err)
		}

		go func() {
			conn, err := net.Dial("tcp4", l.Addr().String())
			test.Assert(t, err == nil)
			_, err = io.WriteString(conn, "hello world")
			test.Assert(t, err == nil)
			test.Assert(t, conn.Close() == nil)
		}()

		conn, err := l.Accept()
		test.Assert(t, err == nil)

		read, err := io.ReadAll(conn)
		test.Assert(t, err == nil)
		test.Assert(t, string(read) == "hello world")
		test.Assert(t, conn.Close() == nil)
		test.Assert(t, l.Close() == nil)
	})

	t.Run("tls", func(t *testing.T) {
		var l net.Listener
		{
			var err error
			l, err = NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithIntermediateTLSConfig("./tls/testdata/cert.crt", "./tls/testdata/cert.key"))
			test.Assert(t, err == nil)
		}

		go func() {
			rootCAPEM, err := os.ReadFile("./tls/testdata/ca.crt")
			test.Assert(t, err == nil)
			rootCAs := x509.NewCertPool()
			test.Assert(t, rootCAs.AppendCertsFromPEM(rootCAPEM))

			conn, err := tls.Dial(l.Addr().Network(), l.Addr().String(), &tls.Config{
				RootCAs: rootCAs, ServerName: "foo.bar", MinVersion: tls.VersionTLS12,
			})
			test.Assert(t, err == nil)

			_, err = io.WriteString(conn, "hello world")
			test.Assert(t, err == nil)
			test.Assert(t, conn.Close() == nil)
		}()

		conn, err := l.Accept()
		test.Assert(t, err == nil)

		read, err := io.ReadAll(conn)
		test.Assert(t, err == nil)
		test.Assert(t, string(read) == "hello world")
		test.Assert(t, conn.Close() == nil)
		test.Assert(t, l.Close() == nil)
	})

	t.Run("with systemd sockets enabled and provided", func(t *testing.T) {
		emulateSystemdProvidingFileDescriptors(t, 2, false)
		setupSystemdEnv(t, func(t *testing.T) {
			t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()))
			t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "2")
		})
		listener, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithSystemdProvidedFileDescriptors())
		test.Assert(t, err == nil)
		test.Assert(t, listener.Addr().Network() == "unix")
		test.Assert(t, listener.Close() == nil)
	})

	t.Run("with systemd sockets enabled but not provided", func(t *testing.T) {
		listener, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithSystemdProvidedFileDescriptors())
		test.Assert(t, err == nil)
		test.Assert(t, listener.Addr().Network() == "tcp")
		test.Assert(t, listener.Close() == nil)
	})

	t.Run("with systemd sockets enabled and wrongly provided", func(t *testing.T) {
		emulateSystemdProvidingFileDescriptors(t, 2, false)
		setupSystemdEnv(t, func(t *testing.T) {
			t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()+1))
			t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "2")
		})
		listener, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithSystemdProvidedFileDescriptors())
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to retrieve systemd listeners"))
		test.Assert(t, listener == nil)
	})

	t.Run("bad option", func(t *testing.T) {
		_, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithIntermediateTLSConfig("dont/exist", "./tls/testdata/cert.key"))
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to apply option"))
	})

	t.Run("unable to listen", func(t *testing.T) {
		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"))
		test.Assert(t, err == nil)
		_, err = NewListener(ListenWithAddress(l.Addr().Network(), l.Addr().String()))
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to listen"))
		test.Assert(t, l.Close() == nil)
	})
}

func Test_ListenAndServe(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		var wg errgroup.Group
		wg.Go(func() error {
			return ListenAndServe(
				newServer(func(rw http.ResponseWriter, _ *http.Request) { rw.WriteHeader(http.StatusTeapot) }),
				ListenWithAddress("tcp", "localhost:0"),
				ServeWithServeErrorTransformer(func(err error) error {
					if errors.Is(err, http.ErrServerClosed) {
						return nil
					}
					return err
				}),
			)(ctx)
		})

		time.Sleep(time.Millisecond * 100)
		cancel()

		test.Assert(t, wg.Wait() == nil)
	})

	t.Run("ko", func(t *testing.T) {
		ctx := t.Context()

		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"))
		test.Assert(t, err == nil)

		err = ListenAndServe(nil, ListenWithAddress(l.Addr().Network(), l.Addr().String()))(ctx)
		test.Assert(t, err != nil && strings.Contains(err.Error(), "unable to listen"))

		test.Assert(t, l.Close() == nil)
	})
}
