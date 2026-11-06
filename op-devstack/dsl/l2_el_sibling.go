package dsl

import (
	"github.com/ethereum-optimism/optimism/op-core/eip1559"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BuildSiblingBlock has the engine build a block that can replace the canonical block at number:
// it extends the same parent from the attributes that block was built from, so it carries the
// same transactions and timestamp, and differs in its prevRandao and so in its hash. The payload
// is returned without being inserted. The build is started by a forkchoice update that names the
// parent as head with zero safe and finalized hashes, since reth 2.6 refuses a nonzero safe or
// finalized hash above the head it is given; naming a canonical ancestor leaves the head where
// it is, which is required.
func (el *L2ELNode) BuildSiblingBlock(number uint64) *eth.ExecutionPayloadEnvelope {
	headBefore := el.BlockRefByLabel(eth.Unsafe)
	replaced := el.PayloadByNumber(number)
	parent := replaced.ExecutionPayload.ParentHash
	attrs := siblingPayloadAttributes(replaced)
	el.log.Info("Building a sibling block", "number", number, "replaced", replaced.ExecutionPayload.BlockHash, "parent", parent, "node", el)
	started := el.ForkchoiceUpdateRaw(parent, common.Hash{}, common.Hash{}, attrs)
	started.IsValid()
	el.require.NotNilf(started.Result.PayloadID, "the engine must start building on %s", parent)
	sibling := el.GetPayload(eth.PayloadInfo{ID: *started.Result.PayloadID, Timestamp: uint64(attrs.Timestamp)})
	el.require.Equal(parent, sibling.ExecutionPayload.ParentHash, "the sibling must extend the replaced block's parent")
	el.require.Equal(number, uint64(sibling.ExecutionPayload.BlockNumber), "the sibling must sit at the replaced block's height")
	el.require.NotEqual(replaced.ExecutionPayload.BlockHash, sibling.ExecutionPayload.BlockHash, "the sibling must differ from the block it replaces")
	el.require.Equal(headBefore, el.BlockRefByLabel(eth.Unsafe), "building the sibling must not move the head")
	el.log.Info("Built a sibling block", "number", number, "sibling", sibling.ExecutionPayload.BlockHash, "replaced", replaced.ExecutionPayload.BlockHash, "node", el)
	return sibling
}

// siblingPayloadAttributes returns the attributes that rebuild payload on its parent with a
// different prevRandao. They force the payload's transactions in and take nothing from the
// transaction pool, and they carry the fee recipient, gas limit and EIP-1559 parameters the
// payload was built with, which an execution engine requires to match the active forks.
func siblingPayloadAttributes(payload *eth.ExecutionPayloadEnvelope) *eth.PayloadAttributes {
	block := payload.ExecutionPayload
	gasLimit := block.GasLimit
	attrs := &eth.PayloadAttributes{
		ParentBeaconBlockRoot: payload.ParentBeaconBlockRoot,
		PrevRandao:            eth.Bytes32(crypto.Keccak256Hash(block.PrevRandao[:])),
		SuggestedFeeRecipient: block.FeeRecipient,
		Timestamp:             block.Timestamp,
		Withdrawals:           block.Withdrawals,

		GasLimit:     &gasLimit,
		NoTxPool:     true,
		Transactions: block.Transactions,
	}
	// Holocene puts the EIP-1559 parameters into extraData, Jovian adds the minimum base fee;
	// before Holocene extraData is empty and the attributes carry neither.
	if len(block.ExtraData) > 0 {
		denominator, elasticity, minBaseFee := eip1559.DecodeJovianExtraData(block.ExtraData)
		params := eth.Bytes8(eip1559.EncodeHolocene1559Params(denominator, elasticity))
		attrs.EIP1559Params = &params
		attrs.MinBaseFee = minBaseFee
	}
	return attrs
}
