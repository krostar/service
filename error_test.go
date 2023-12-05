package service

import (
	"testing"

	"gotest.tools/v3/assert"
)

func Test_sentinelError_Error(t *testing.T) {
	assert.Equal(t, sentinelError("foo").Error(), "foo")
}
