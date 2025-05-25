package service

import (
	"testing"

	"github.com/krostar/test"
)

func Test_sentinelError_Error(t *testing.T) {
	test.Assert(t, sentinelError("foo").Error() == "foo")
}
