package crossdomain_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/stretchr/testify/require"
)

func FuzzVersionedNonce(f *testing.F) {
	f.Fuzz(func(t *testing.T, _nonce []byte, _version uint16) {
		inputNonce := new(big.Int).SetBytes(_nonce)

		// Clamp nonce to uint240
		if inputNonce.Cmp(crossdomain.NonceMask) > 0 {
			inputNonce = new(big.Int).Set(crossdomain.NonceMask)
		}
		// Clamp version to 0 or 1
		_version = _version % 2

		inputVersion := new(big.Int).SetUint64(uint64(_version))
		encodedNonce := crossdomain.EncodeVersionedNonce(inputNonce, inputVersion)

		decodedNonce, decodedVersion := crossdomain.DecodeVersionedNonce(encodedNonce)

		require.Equal(t, decodedNonce.Uint64(), inputNonce.Uint64())
		require.Equal(t, decodedVersion.Uint64(), inputVersion.Uint64())
	})
}
