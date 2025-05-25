//go:build darwin

package netservice

import (
	"syscall"
	"testing"

	"github.com/krostar/test"
)

func getKeepAliveConfig(t *testing.T, fd int) (bool, int) {
	activated, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
	test.Assert(t, err == nil)

	secs, err := syscall.GetsockoptInt(fd, syscall.IPPROTO_TCP, 0x101)
	test.Assert(t, err == nil)

	return activated > 0, secs
}
