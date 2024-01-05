package contracts

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestFaultDisputeGameContract_CommonTests(t *testing.T) {
	runCommonDisputeGameTests(t, func(t *testing.T) (*batchingTest.AbiBasedRpc, *disputeGameContract) {
		stubRpc, contract := setupFaultDisputeGameTest(t)
		return stubRpc, &contract.disputeGameContract
	})
}

func TestGetProposals(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	agreedIndex := big.NewInt(5)
	agreedBlockNum := big.NewInt(6)
	agreedRoot := common.Hash{0xaa}
	disputedIndex := big.NewInt(7)
	disputedBlockNum := big.NewInt(8)
	disputedRoot := common.Hash{0xdd}
	agreed := contractProposal{
		Index:         agreedIndex,
		L2BlockNumber: agreedBlockNum,
		OutputRoot:    agreedRoot,
	}
	disputed := contractProposal{
		Index:         disputedIndex,
		L2BlockNumber: disputedBlockNum,
		OutputRoot:    disputedRoot,
	}
	expectedAgreed := Proposal{
		L2BlockNumber: agreed.L2BlockNumber,
		OutputRoot:    agreed.OutputRoot,
	}
	expectedDisputed := Proposal{
		L2BlockNumber: disputed.L2BlockNumber,
		OutputRoot:    disputed.OutputRoot,
	}
	stubRpc.SetResponse(fdgAddr, methodProposals, batching.BlockLatest, []interface{}{}, []interface{}{
		agreed, disputed,
	})
	actualAgreed, actualDisputed, err := game.GetProposals(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedAgreed, actualAgreed)
	require.Equal(t, expectedDisputed, actualDisputed)
}

func TestFaultDisputeGame_UpdateOracleTx(t *testing.T) {
	t.Run("Local", func(t *testing.T) {
		stubRpc, game := setupFaultDisputeGameTest(t)
		data := &faultTypes.PreimageOracleData{
			IsLocal:      true,
			OracleKey:    common.Hash{0xbc}.Bytes(),
			OracleData:   []byte{1, 2, 3, 4, 5, 6, 7},
			OracleOffset: 16,
		}
		claimIdx := uint64(6)
		stubRpc.SetResponse(fdgAddr, methodAddLocalData, batching.BlockLatest, []interface{}{
			data.GetIdent(),
			faultTypes.NoLocalContext,
			new(big.Int).SetUint64(uint64(data.OracleOffset)),
		}, nil)
		tx, err := game.UpdateOracleTx(context.Background(), claimIdx, data)
		require.NoError(t, err)
		stubRpc.VerifyTxCandidate(tx)
	})

	t.Run("Global", func(t *testing.T) {
		stubRpc, game := setupFaultDisputeGameTest(t)
		data := &faultTypes.PreimageOracleData{
			IsLocal:      false,
			OracleKey:    common.Hash{0xbc}.Bytes(),
			OracleData:   []byte{1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14, 15},
			OracleOffset: 16,
		}
		claimIdx := uint64(6)
		stubRpc.SetResponse(fdgAddr, methodVMV0, batching.BlockLatest, nil, []interface{}{vmAddr})
		stubRpc.SetResponse(vmAddr, methodOracle, batching.BlockLatest, nil, []interface{}{oracleAddr})
		stubRpc.SetResponse(oracleAddr, methodLoadKeccak256PreimagePart, batching.BlockLatest, []interface{}{
			new(big.Int).SetUint64(uint64(data.OracleOffset)),
			data.GetPreimageWithoutSize(),
		}, nil)
		tx, err := game.UpdateOracleTx(context.Background(), claimIdx, data)
		require.NoError(t, err)
		stubRpc.VerifyTxCandidate(tx)
	})
}

func setupFaultDisputeGameTest(t *testing.T) (*batchingTest.AbiBasedRpc, *FaultDisputeGameContract) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)

	vmAbi, err := bindings.MIPSMetaData.GetAbi()
	require.NoError(t, err)
	oracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	require.NoError(t, err)

	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, fdgAbi)
	stubRpc.AddContract(vmAddr, vmAbi)
	stubRpc.AddContract(oracleAddr, oracleAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	game, err := NewFaultDisputeGameContract(fdgAddr, caller)
	require.NoError(t, err)
	return stubRpc, game
}
