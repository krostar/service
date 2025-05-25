//go:build aix || freebsd || linux || netbsd

package netservice

import (
	"syscall"
	"testing"

	"github.com/krostar/test"
)

func getKeepAliveConfig(t *testing.T, fd int) (bool, int) {
	activated, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
	test.Assert(t, err == nil)

	secs, err := syscall.GetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL)
	test.Assert(t, err == nil)

	return activated > 0, secs
}
