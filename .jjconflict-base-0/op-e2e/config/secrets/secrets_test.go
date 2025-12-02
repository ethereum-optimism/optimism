package secrets

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSecrets(t *testing.T) {
	addresses := DefaultSecrets.Addresses()
	require.Equal(t, addresses.Proposer, common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"))
	require.Equal(t, addresses.Batcher, common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"))
}
