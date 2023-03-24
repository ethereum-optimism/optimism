package batcher

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"
)

// TestBatchSubmitter_SendTransaction tests the driver's
// [SendTransaction] external facing function.
func TestBatchSubmitter_SendTransaction(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	txMgr := mocks.TxManager{}
	batcherInboxAddress := common.HexToAddress("0x42000000000000000000000000000000000000ff")
	chainID := big.NewInt(1)
	sender := common.HexToAddress("0xdeadbeef")
	bs := BatchSubmitter{
		Config: Config{
			log:  log,
			From: sender,
			Rollup: &rollup.Config{
				L1ChainID:         chainID,
				BatchInboxAddress: batcherInboxAddress,
			},
		},
		txMgr: &txMgr,
	}
	txData := []byte{0x00, 0x01, 0x02}

	gasTipCap := big.NewInt(136)
	gasFeeCap := big.NewInt(137)
	gas := uint64(1337)

	// Candidate gas should be calculated with [core.IntrinsicGas]
	intrinsicGas, err := core.IntrinsicGas(txData, nil, false, true, true, false)
	require.NoError(t, err)
	candidate := txmgr.TxCandidate{
		To:       batcherInboxAddress,
		TxData:   txData,
		From:     sender,
		GasLimit: intrinsicGas,
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     0,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gas,
		To:        &batcherInboxAddress,
		Data:      txData,
	})
	txHash := tx.Hash()

	expectedReceipt := types.Receipt{
		Type:              1,
		PostState:         []byte{},
		Status:            uint64(1),
		CumulativeGasUsed: gas,
		TxHash:            txHash,
		GasUsed:           gas,
	}

	txMgr.On("CraftTx", mock.Anything, candidate).Return(tx, nil)
	txMgr.On("Send", mock.Anything, tx).Return(&expectedReceipt, nil)

	receipt, err := bs.SendTransaction(context.Background(), tx.Data())
	require.NoError(t, err)
	require.Equal(t, receipt, &expectedReceipt)
}
