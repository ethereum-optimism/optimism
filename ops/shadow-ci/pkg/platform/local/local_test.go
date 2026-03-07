package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePathComponent(t *testing.T) {
	assert.NoError(t, validatePathComponent("abc123"))
	assert.NoError(t, validatePathComponent("pipeline-42"))

	assert.Error(t, validatePathComponent("../etc"))
	assert.Error(t, validatePathComponent("foo/bar"))
	assert.Error(t, validatePathComponent(".."))
	assert.Error(t, validatePathComponent("a/../b"))
}
