package p2p

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestLocalSignerSetupCreatesFreshSigner(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	setup := NewLocalSignerSetup(privateKey)

	first, err := setup.SetupSigner(t.Context())
	require.NoError(t, err)
	_, err = first.SignBlockV1(t.Context(), eth.ChainIDFromUInt64(901), common.Hash{0x1})
	require.NoError(t, err)
	require.NoError(t, first.Close())

	second, err := setup.SetupSigner(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, second.Close()) })
	require.NotSame(t, first, second)
	_, err = second.SignBlockV1(t.Context(), eth.ChainIDFromUInt64(901), common.Hash{0x2})
	require.NoError(t, err, "closing one node's signer must not poison the next lifecycle")
}
