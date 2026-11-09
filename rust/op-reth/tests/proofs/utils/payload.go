package utils

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// nextBlockMinBaseFee is the Jovian minimum base fee the attributes carry; any value the chain
// accepts does for a block that is only executed, never sealed.
const nextBlockMinBaseFee = uint64(10)

// NextPayloadAttributes returns the attributes of a block built on parent that carries exactly
// rawTxs and nothing from the transaction pool, one block time after parent with parent's gas
// limit, fee recipient and randomness.
func NextPayloadAttributes(parent eth.BlockInfo, rawTxs ...[]byte) eth.PayloadAttributes {
	gasLimit := eth.Uint64Quantity(parent.GasLimit())
	minBaseFee := nextBlockMinBaseFee
	var prevRandao eth.Bytes32
	copy(prevRandao[:], parent.MixDigest().Bytes())
	var zeroEIP1559Params eth.Bytes8
	txs := make([]eth.Data, len(rawTxs))
	for i, raw := range rawTxs {
		txs[i] = eth.Data(raw)
	}
	return eth.PayloadAttributes{
		EIP1559Params:         &zeroEIP1559Params,
		GasLimit:              &gasLimit,
		MinBaseFee:            &minBaseFee,
		NoTxPool:              true,
		ParentBeaconBlockRoot: parent.ParentBeaconRoot(),
		PrevRandao:            prevRandao,
		SuggestedFeeRecipient: parent.Coinbase(),
		Timestamp:             eth.Uint64Quantity(parent.Time() + 1),
		Transactions:          txs,
	}
}

// SignedTransaction signs the transaction that plan describes without sending it and returns its
// encoding.
func SignedTransaction(t devtest.T, plan txplan.Option) []byte {
	tx := txplan.NewPlannedTx(plan)
	signed, err := tx.Signed.Eval(t.Ctx())
	require.NoError(t, err, "transaction must sign")
	raw, err := signed.MarshalBinary()
	require.NoError(t, err, "transaction must encode")
	return raw
}

// SignedTransfer signs a transfer of amount from sender to recipient without sending it and
// returns its encoding. The transfer takes sender's next nonce unless opts say otherwise.
func SignedTransfer(t devtest.T, sender *dsl.EOA, recipient common.Address, amount eth.ETH, opts ...txplan.Option) []byte {
	return SignedTransaction(t, txplan.Combine(sender.PlanTransfer(recipient, amount), txplan.Combine(opts...)))
}
