package contracts

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestVMContract_Oracle(t *testing.T) {
	vmAbi, err := bindings.MIPSMetaData.GetAbi()
	require.NoError(t, err)

	stubRpc := batchingTest.NewAbiBasedRpc(t, vmAddr, vmAbi)
	vmContract, err := NewVMContract(vmAddr, batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize))
	require.NoError(t, err)

	stubRpc.SetResponse(vmAddr, methodOracle, rpcblock.Latest, nil, []interface{}{oracleAddr})

	oracleContract, err := vmContract.Oracle(context.Background())
	require.NoError(t, err)
	tx, err := oracleContract.AddGlobalDataTx(types.NewPreimageOracleData(common.Hash{byte(preimage.Keccak256KeyType)}.Bytes(), make([]byte, 20), 0))
	require.NoError(t, err)
	// This test doesn't care about all the tx details, we just want to confirm the contract binding is using the
	// correct address
	require.Equal(t, &oracleAddr, tx.To)
}
