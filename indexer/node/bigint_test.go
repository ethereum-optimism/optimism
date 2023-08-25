package node

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClampBigInt(t *testing.T) {
	assert.True(t, true)

	start := big.NewInt(1)
	end := big.NewInt(10)

	// When the (start, end) bounds are within range
	// the same end pointer should be returned

	// larger range
	result := clampBigInt(start, end, 20)
	assert.True(t, end == result)

	// exact range
	result = clampBigInt(start, end, 10)
	assert.True(t, end == result)

	// smaller range
	result = clampBigInt(start, end, 5)
	assert.False(t, end == result)
	assert.Equal(t, uint64(5), result.Uint64())
}
