package e2e_tests

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func genMPTProof(t *testing.T, outboxRoot common.Hash, evt *bindings.CrossL2OutboxMessagePassed, cl *ethclient.Client) []byte {
	storageKey := crypto.Keccak256Hash(evt.MessageRoot[:], new(common.Hash).Bytes()) // first storage slot
	t.Logf("requesting proof for storage key: %s", storageKey)

	var getProofResponse *eth.AccountResult
	err := cl.Client().CallContext(context.Background(), &getProofResponse, "eth_getProof", predeploys.CrossL2OutboxAddr, []common.Hash{storageKey}, "latest")
	require.NoError(t, err, "must build storage proof")
	require.Equal(t, outboxRoot, getProofResponse.StorageHash, "outbox storage hash must match what proof is generated for")
	require.Len(t, getProofResponse.StorageProof, 1, "need storage proof")
	msgProofEntry := getProofResponse.StorageProof[0]

	// Just concatenate all RLP nodes of the MPT tree, top to bottom (as the eth_getProof should return).
	// The precompile reads it as a stream based
	var proofData []byte
	for i, mptNode := range msgProofEntry.Proof {
		proofData = append(proofData, mptNode...)
		t.Logf("proof node %d: %s    - hash: %s", i, mptNode, crypto.Keccak256Hash(mptNode))
	}

	return proofData
}
