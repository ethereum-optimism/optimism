package multithreaded

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// Unit test splitmix64 based on Apache Commons RNG unit tests
// See: https://github.com/apache/commons-rng/blob/df772c2f5b0644a71398e925206039a2ae516ab2/commons-rng-core/src/test/java/org/apache/commons/rng/core/source64/SplitMix64Test.java
func TestSplitmix64(t *testing.T) {
	expectedSequence := []uint64{
		0x4141302768c9e9d0, 0x64df48c4eab51b1a, 0x4e723b53dbd901b3, 0xead8394409dd6454,
		0x3ef60e485b412a0a, 0xb2a23aee63aecf38, 0x6cc3b8933c4fa332, 0x9c9e75e031e6fccb,
		0x0fddffb161c9f30f, 0x2d1d75d4e75c12a3, 0xcdcf9d2dde66da2e, 0x278ba7d1d142cfec,
		0x4ca423e66072e606, 0x8f2c3c46ebc70bb7, 0xc9def3b1eeae3e21, 0x8e06670cd3e98bce,
		0x2326dee7dd34747f, 0x3c8fff64392bb3c1, 0xfc6aa1ebe7916578, 0x3191fb6113694e70,
		0x3453605f6544dac6, 0x86cf93e5cdf81801, 0x0d764d7e59f724df, 0xae1dfb943ebf8659,
		0x012de1babb3c4104, 0xa5a818b8fc5aa503, 0xb124ea2b701f4993, 0x18e0374933d8c782,
		0x2af8df668d68ad55, 0x76e56f59daa06243, 0xf58c016f0f01e30f, 0x8eeafa41683dbbf4,
		0x7bf121347c06677f, 0x4fd0c88d25db5ccb, 0x99af3be9ebe0a272, 0x94f2b33b74d0bdcb,
		0x24b5d9d7a00a3140, 0x79d983d781a34a3c, 0x582e4a84d595f5ec, 0x7316fe8b0f606d20,
	}

	var currentSeed uint64 = 0x1a2b3c4d5e6f7531
	for i := 0; i < len(expectedSequence); i++ {
		expectedOutput := expectedSequence[i]
		actual := splitmix64(currentSeed)
		require.Equal(t, expectedOutput, actual, fmt.Sprintf("Failed at index %d", i))
		currentSeed += 0x9e3779b97f4a7c15
	}
}
