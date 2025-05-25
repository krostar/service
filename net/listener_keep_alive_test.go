package netservice

import (
	"net"
	"testing"
	"time"

	"github.com/krostar/test"
)

func Test_NewListener_keepalive(t *testing.T) {
	t.Run("without keepalive", func(t *testing.T) {
		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithoutKeepAlive())
		test.Require(t, err == nil && l != nil)

		activated, _ := tcpGetKeepAliveSockOPT(t, l)
		test.Assert(t, !activated, "keepalive should not be set with normal listener")
		test.Assert(t, l.Close() == nil)
	})

	t.Run("with keepalive", func(t *testing.T) {
		l, err := NewListener(ListenWithAddress("tcp", "localhost:0"), ListenWithKeepAlive(17*time.Second))
		test.Require(t, err == nil && l != nil)

		activated, period := tcpGetKeepAliveSockOPT(t, l)
		test.Assert(t, activated && period == 17, "keepalive should have been set")
		test.Assert(t, l.Close() == nil)
	})
}

func tcpGetKeepAliveSockOPT(t *testing.T, l net.Listener) (bool, int) {
	t.Helper()
	go func() {
		conn, err := net.Dial("tcp", l.Addr().String())
		if test.Assert(t, err == nil) {
			test.Assert(t, conn.Close() == nil)
		}
	}()

	conn, err := l.Accept()
	test.Require(t, err == nil && conn != nil)

	var (
		activated bool
		secs      int
	)

	r, err := conn.(*net.TCPConn).SyscallConn()
	test.Require(t, err == nil && r != nil)
	test.Assert(t, r.Control(func(fd uintptr) {
		activated, secs = getKeepAliveConfig(t, int(fd))
	}) == nil)

	test.Assert(t, conn.Close() == nil)

	return activated, secs
}
