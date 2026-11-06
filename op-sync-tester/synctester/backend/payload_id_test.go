package backend

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// TestPayloadIDMatchesOpGeth is a differential test against op-geth's BuildPayloadArgs.Id().
// Delete it when go.mod switches to upstream go-ethereum, which lacks the OP fields.
func TestPayloadIDMatchesOpGeth(t *testing.T) {
	parent := common.HexToHash("0x01")

	// No optional fields set.
	bedrock := miner.BuildPayloadArgs{Parent: parent, Version: engine.PayloadV1}
	require.Equal(t, bedrock.Id(), computePayloadID(parent, &eth.PayloadAttributes{}, engine.PayloadV1))

	// Every optional field set.
	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 7, Gas: 21_000}),
		types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(10), Nonce: 8, Gas: 21_000}),
	}
	var rawTxs []eth.Data
	for _, tx := range txs {
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)
		rawTxs = append(rawTxs, raw)
	}
	beaconRoot := common.HexToHash("0x02")
	feeRecipient := common.HexToAddress("0x03")
	eip1559Params := eth.Bytes8{0, 0, 0, 0xfa, 0, 0, 0, 6}
	gasLimit, minBaseFee := uint64(30_000_000), uint64(1e9)
	jovian := miner.BuildPayloadArgs{
		Parent:        parent,
		Timestamp:     1234,
		FeeRecipient:  feeRecipient,
		Random:        common.Hash{4},
		Withdrawals:   types.Withdrawals{},
		BeaconRoot:    &beaconRoot,
		Version:       engine.PayloadV3,
		NoTxPool:      true,
		Transactions:  txs,
		GasLimit:      &gasLimit,
		EIP1559Params: eip1559Params[:],
		MinBaseFee:    &minBaseFee,
	}
	require.Equal(t, jovian.Id(), computePayloadID(parent, &eth.PayloadAttributes{
		Timestamp:             1234,
		SuggestedFeeRecipient: feeRecipient,
		PrevRandao:            eth.Bytes32{4},
		Withdrawals:           &types.Withdrawals{},
		ParentBeaconBlockRoot: &beaconRoot,
		NoTxPool:              true,
		Transactions:          rawTxs,
		GasLimit:              ptr.New(eth.Uint64Quantity(gasLimit)),
		EIP1559Params:         &eip1559Params,
		MinBaseFee:            &minBaseFee,
	}, engine.PayloadV3))
}
