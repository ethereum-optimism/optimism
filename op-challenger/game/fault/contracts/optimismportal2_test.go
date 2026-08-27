package contracts

import (
	"context"
	"testing"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

var portalAddr = common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")

func setupOptimismPortal2Test(t *testing.T) (*batchingTest.AbiBasedRpc, *OptimismPortal2Contract) {
	stubRpc := batchingTest.NewAbiBasedRpc(t, portalAddr, snapshots.LoadOptimismPortal2ABI())
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	return stubRpc, NewOptimismPortal2Contract(contractMetrics.NoopContractMetrics, portalAddr, caller)
}

func TestOptimismPortal2_GetProvenWithdrawals(t *testing.T) {
	stubRpc, portal := setupOptimismPortal2Test(t)
	block := rpcblock.ByNumber(42)
	proofs := []WithdrawalProof{
		{WithdrawalHash: common.Hash{0xaa}, ProofSubmitter: common.Address{0x01}},
		{WithdrawalHash: common.Hash{0xbb}, ProofSubmitter: common.Address{0x02}},
	}
	expected := []ProvenWithdrawal{
		{DisputeGameProxy: common.Address{0xcc}, Timestamp: 100},
		{},
	}
	for i, proof := range proofs {
		stubRpc.SetResponse(portalAddr, methodProvenWithdrawals, block,
			[]interface{}{proof.WithdrawalHash, proof.ProofSubmitter},
			[]interface{}{expected[i].DisputeGameProxy, expected[i].Timestamp})
	}

	actual, err := portal.GetProvenWithdrawals(context.Background(), block, proofs)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestOptimismPortal2_DeleteProvenWithdrawalTx(t *testing.T) {
	stubRpc, portal := setupOptimismPortal2Test(t)
	proof := WithdrawalProof{WithdrawalHash: common.Hash{0xaa}, ProofSubmitter: common.Address{0xbb}}
	stubRpc.SetResponse(portalAddr, methodDeleteProvenWithdrawal, rpcblock.Latest,
		[]interface{}{proof.WithdrawalHash, proof.ProofSubmitter}, nil)

	candidate, err := portal.DeleteProvenWithdrawalTx(proof)
	require.NoError(t, err)
	require.Equal(t, &portalAddr, candidate.To)
	stubRpc.VerifyTxCandidate(candidate)
}

func TestOptimismPortal2_DecodeWithdrawalProvenExtension1(t *testing.T) {
	_, portal := setupOptimismPortal2Test(t)
	proof := WithdrawalProof{WithdrawalHash: common.Hash{0xaa}, ProofSubmitter: common.Address{0xbb}}

	t.Run("Valid", func(t *testing.T) {
		actual, err := portal.DecodeWithdrawalProvenExtension1(&ethTypes.Log{
			Address: portalAddr,
			Topics: []common.Hash{
				portal.WithdrawalProvenExtension1Topic(),
				proof.WithdrawalHash,
				common.BytesToHash(proof.ProofSubmitter.Bytes()),
			},
		})
		require.NoError(t, err)
		require.Equal(t, proof, actual)
	})

	t.Run("WrongEvent", func(t *testing.T) {
		_, err := portal.DecodeWithdrawalProvenExtension1(&ethTypes.Log{
			Address: portalAddr,
			Topics:  []common.Hash{{0xde, 0xad}},
		})
		require.ErrorIs(t, err, batching.ErrUnknownEvent)
	})
}
