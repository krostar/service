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
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

func Test_NewListener(t *testing.T) {
	t.Run("no tls", func(t *testing.T) {
		var l net.Listener
		{
			var err error
			l, err = NewListener(context.Background(), "localhost:0")
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
			l, err = NewListener(context.Background(), "localhost:0", ListenWithIntermediateTLSConfig("./testdata/cert.crt", "./testdata/cert.key"))
			assert.NilError(t, err)
		}

		go func() {
			rootCAPEM, err := os.ReadFile("./testdata/ca.crt")
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

	t.Run("bad option", func(t *testing.T) {
		_, err := NewListener(context.Background(), "localhost:0", ListenWithIntermediateTLSConfig("dont/exist", "./testdata/cert.key"))
		assert.ErrorContains(t, err, "unable to apply option")
	})

	t.Run("unable to listen", func(t *testing.T) {
		l, err := NewListener(context.Background(), "localhost:0")
		assert.NilError(t, err)
		_, err = NewListener(context.Background(), l.Addr().String())
		assert.ErrorContains(t, err, "unable to listen")
		assert.NilError(t, l.Close())
	})
}

func Test_ListenAndServe(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		var wg errgroup.Group
		wg.Go(func() error {
			return ListenAndServe("localhost:0",
				newServer(func(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(http.StatusTeapot) }),
				ServerWithServeErrorTransformer(func(err error) error {
					if errors.Is(err, http.ErrServerClosed) {
						return nil
					}
					return err
				}),
				ListenWithNetwork("tcp"),
			)(ctx)
		})

		time.Sleep(time.Second * 2)
		cancel()

		assert.NilError(t, wg.Wait())
	})

	t.Run("ko", func(t *testing.T) {
		ctx := context.Background()

		l, err := NewListener(ctx, "localhost:0")
		assert.NilError(t, err)

		assert.ErrorContains(t, ListenAndServe(l.Addr().String(), nil)(ctx), "unable to listen")
	})
}
