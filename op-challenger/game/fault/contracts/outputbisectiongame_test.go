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

func TestOutputBisectionGameContract_CommonTests(t *testing.T) {
	runCommonDisputeGameTests(t, func(t *testing.T) (*batchingTest.AbiBasedRpc, *disputeGameContract) {
		stubRpc, contract := setupOutputBisectionGameTest(t)
		return stubRpc, &contract.disputeGameContract
	})
}

func TestGetBlockRange(t *testing.T) {
	stubRpc, contract := setupOutputBisectionGameTest(t)
	expectedStart := uint64(65)
	expectedEnd := uint64(102)
	stubRpc.SetResponse(fdgAddr, methodGenesisBlockNumber, batching.BlockLatest, nil, []interface{}{new(big.Int).SetUint64(expectedStart)})
	stubRpc.SetResponse(fdgAddr, methodL2BlockNumber, batching.BlockLatest, nil, []interface{}{new(big.Int).SetUint64(expectedEnd)})
	start, end, err := contract.GetBlockRange(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedStart, start)
	require.Equal(t, expectedEnd, end)
}

func TestGetSplitDepth(t *testing.T) {
	stubRpc, contract := setupOutputBisectionGameTest(t)
	expectedSplitDepth := uint64(15)
	stubRpc.SetResponse(fdgAddr, methodSplitDepth, batching.BlockLatest, nil, []interface{}{new(big.Int).SetUint64(expectedSplitDepth)})
	splitDepth, err := contract.GetSplitDepth(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedSplitDepth, splitDepth)
}

func TestGetGenesisOutputRoot(t *testing.T) {
	stubRpc, contract := setupOutputBisectionGameTest(t)
	expectedOutputRoot := common.HexToHash("0x1234")
	stubRpc.SetResponse(fdgAddr, methodGenesisOutputRoot, batching.BlockLatest, nil, []interface{}{expectedOutputRoot})
	genesisOutputRoot, err := contract.GetGenesisOutputRoot(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedOutputRoot, genesisOutputRoot)
}

func TestOutputBisectionGame_UpdateOracleTx(t *testing.T) {
	t.Run("Local", func(t *testing.T) {
		stubRpc, game := setupOutputBisectionGameTest(t)
		data := &faultTypes.PreimageOracleData{
			IsLocal:      true,
			OracleKey:    common.Hash{0xbc}.Bytes(),
			OracleData:   []byte{1, 2, 3, 4, 5, 6, 7},
			OracleOffset: 16,
		}
		claimIdx := uint64(6)
		stubRpc.SetResponse(fdgAddr, methodAddLocalData, batching.BlockLatest, []interface{}{
			data.GetIdent(),
			new(big.Int).SetUint64(claimIdx),
			new(big.Int).SetUint64(uint64(data.OracleOffset)),
		}, nil)
		tx, err := game.UpdateOracleTx(context.Background(), claimIdx, data)
		require.NoError(t, err)
		stubRpc.VerifyTxCandidate(tx)
	})

	t.Run("Global", func(t *testing.T) {
		stubRpc, game := setupOutputBisectionGameTest(t)
		data := &faultTypes.PreimageOracleData{
			IsLocal:      false,
			OracleKey:    common.Hash{0xbc}.Bytes(),
			OracleData:   []byte{1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14, 15},
			OracleOffset: 16,
		}
		claimIdx := uint64(6)
		stubRpc.SetResponse(fdgAddr, methodVMV1, batching.BlockLatest, nil, []interface{}{vmAddr})
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

func setupOutputBisectionGameTest(t *testing.T) (*batchingTest.AbiBasedRpc, *OutputBisectionGameContract) {
	fdgAbi, err := bindings.OutputBisectionGameMetaData.GetAbi()
	require.NoError(t, err)

	vmAbi, err := bindings.MIPSMetaData.GetAbi()
	require.NoError(t, err)
	oracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	require.NoError(t, err)

	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, fdgAbi)
	stubRpc.AddContract(vmAddr, vmAbi)
	stubRpc.AddContract(oracleAddr, oracleAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	game, err := NewOutputBisectionGameContract(fdgAddr, caller)
	require.NoError(t, err)
	return stubRpc, game
}
