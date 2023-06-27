package kvstore

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPreimageSourceSplitter(t *testing.T) {
	localResult := []byte{1}
	globalResult := []byte{2}
	local := func(key common.Hash) ([]byte, error) { return localResult, nil }
	global := func(key common.Hash) ([]byte, error) { return globalResult, nil }
	splitter := NewPreimageSourceSplitter(local, global)

	tests := []struct {
		name      string
		keyPrefix byte
		expected  []byte
	}{
		{"Local", byte(preimage.LocalKeyType), localResult},
		{"Keccak", byte(preimage.Keccak256KeyType), globalResult},
		{"Generic", byte(3), globalResult},
		{"Reserved", byte(4), globalResult},
		{"Application", byte(255), globalResult},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := common.Hash{0xff}
			key[0] = test.keyPrefix
			res, err := splitter.Get(key)
			require.NoError(t, err)
			require.Equal(t, test.expected, res)
		})
	}
}
