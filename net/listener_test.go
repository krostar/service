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
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

func Test_NewListener(t *testing.T) {
	t.Run("no configuration", func(t *testing.T) {
		l, err := NewListener()
		assert.ErrorContains(t, err, "no listener configured")
		assert.Check(t, l == nil)
	})

	t.Run("no tls", func(t *testing.T) {
		var l net.Listener
		{
			var err error
			l, err = NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithSystemdProvidedFileDescriptors())
			assert.NilError(t, err)
		}

		go func() {
			conn, err := net.Dial("tcp4", l.Addr().String())
			assert.NilError(t, err)
			_, err = io.WriteString(conn, "hello world")
			assert.NilError(t, err)
			assert.NilError(t, conn.Close())
		}()

		conn, err := l.Accept()
		assert.NilError(t, err)

		read, err := io.ReadAll(conn)
		assert.NilError(t, err)
		assert.Equal(t, "hello world", string(read))

		assert.NilError(t, conn.Close())
		assert.NilError(t, l.Close())
	})

	t.Run("tls", func(t *testing.T) {
		var l net.Listener
		{
			var err error
			l, err = NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithIntermediateTLSConfig("./tls/testdata/cert.crt", "./tls/testdata/cert.key"))
			assert.NilError(t, err)
		}

		go func() {
			rootCAPEM, err := os.ReadFile("./tls/testdata/ca.crt")
			assert.NilError(t, err)
			rootCAs := x509.NewCertPool()
			assert.Check(t, rootCAs.AppendCertsFromPEM(rootCAPEM))

			conn, err := tls.Dial(l.Addr().Network(), l.Addr().String(), &tls.Config{
				RootCAs: rootCAs, ServerName: "foo.bar", MinVersion: tls.VersionTLS12,
			})
			assert.NilError(t, err)

			_, err = io.WriteString(conn, "hello world")
			assert.NilError(t, err)
			assert.NilError(t, conn.Close())
		}()

		conn, err := l.Accept()
		assert.NilError(t, err)

		read, err := io.ReadAll(conn)
		assert.NilError(t, err)
		assert.Equal(t, "hello world", string(read))

		assert.NilError(t, conn.Close())
		assert.NilError(t, l.Close())
	})

	t.Run("with systemd sockets enabled and provided", func(t *testing.T) {
		emulateSystemdProvidingFileDescriptors(t, 2, false)
		setupSystemdEnv(t, func(t *testing.T) {
			t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()))
			t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "2")
		})
		listener, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithSystemdProvidedFileDescriptors())
		assert.Check(t, err)
		assert.Check(t, listener.Addr().Network() == "unix")
		assert.Check(t, listener.Close())
	})

	t.Run("with systemd sockets enabled but not provided", func(t *testing.T) {
		listener, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithSystemdProvidedFileDescriptors())
		assert.Check(t, err)
		assert.Check(t, listener.Addr().Network() == "tcp")
		assert.Check(t, listener.Close())
	})

	t.Run("with systemd sockets enabled and wrongly provided", func(t *testing.T) {
		emulateSystemdProvidingFileDescriptors(t, 2, false)
		setupSystemdEnv(t, func(t *testing.T) {
			t.Setenv(_systemdSocketActivationEnvExpectedProgramIDKey, strconv.Itoa(os.Getpid()+1))
			t.Setenv(_systemdSocketActivationEnvNumberOfFileDescriptorsKey, "2")
		})
		listener, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithSystemdProvidedFileDescriptors())
		assert.ErrorContains(t, err, "unable to retrieve systemd listeners")
		assert.Check(t, listener == nil)
	})

	t.Run("bad option", func(t *testing.T) {
		_, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithIntermediateTLSConfig("dont/exist", "./tls/testdata/cert.key"))
		assert.ErrorContains(t, err, "unable to apply option")
	})

	t.Run("unable to listen", func(t *testing.T) {
		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"))
		assert.NilError(t, err)
		_, err = NewListener(ListenWithAddress(l.Addr().Network(), l.Addr().String()))
		assert.ErrorContains(t, err, "unable to listen")
		assert.NilError(t, l.Close())
	})
}

func Test_ListenAndServe(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

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

		assert.NilError(t, wg.Wait())
	})

	t.Run("ko", func(t *testing.T) {
		ctx := context.Background()

		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"))
		assert.NilError(t, err)

		assert.ErrorContains(t, ListenAndServe(nil, ListenWithAddress(l.Addr().Network(), l.Addr().String()))(ctx), "unable to listen")
		assert.NilError(t, l.Close())
	})
}
