package dsl

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/eip1559"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func siblingSourcePayload(extraData []byte) *eth.ExecutionPayloadEnvelope {
	beaconRoot := common.HexToHash("0xbeac")
	return &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockHash:    common.HexToHash("0xb10c"),
			BlockNumber:  42,
			ExtraData:    extraData,
			FeeRecipient: common.HexToAddress("0xfee"),
			GasLimit:     30_000_000,
			ParentHash:   common.HexToHash("0x9a"),
			PrevRandao:   eth.Bytes32(common.HexToHash("0x1")),
			Timestamp:    1_700_000_000,
			Transactions: []eth.Data{{0x7e, 0x01}, {0x02, 0x02}},
			Withdrawals:  &types.Withdrawals{},
		},
		ParentBeaconBlockRoot: &beaconRoot,
	}
}

func TestSiblingPayloadAttributesRebuildTheBlockWithAnotherPrevRandao(t *testing.T) {
	t.Parallel()
	payload := siblingSourcePayload(nil)
	attrs := siblingPayloadAttributes(payload)
	block := payload.ExecutionPayload
	require.Equal(t, block.Timestamp, attrs.Timestamp)
	require.NotEqual(t, block.PrevRandao, attrs.PrevRandao)
	require.Equal(t, block.FeeRecipient, attrs.SuggestedFeeRecipient)
	require.Equal(t, block.Withdrawals, attrs.Withdrawals)
	require.Equal(t, payload.ParentBeaconBlockRoot, attrs.ParentBeaconBlockRoot)
	require.Equal(t, block.Transactions, attrs.Transactions)
	require.True(t, attrs.NoTxPool, "the sibling must carry the replaced block's transactions and nothing from the pool")
	require.Equal(t, block.GasLimit, *attrs.GasLimit)
}

func TestSiblingPayloadAttributesCarryTheExtraDataParameters(t *testing.T) {
	t.Parallel()
	minBaseFee := uint64(7)
	for _, tc := range []struct {
		name          string
		extraData     []byte
		eip1559Params *eth.Bytes8
		minBaseFee    *uint64
	}{
		{name: "pre-holocene"},
		{name: "holocene", extraData: eip1559.EncodeHoloceneExtraData(250, 6), eip1559Params: siblingParams(250, 6)},
		{name: "jovian", extraData: eip1559.EncodeJovianExtraData(250, 6, minBaseFee), eip1559Params: siblingParams(250, 6), minBaseFee: &minBaseFee},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			attrs := siblingPayloadAttributes(siblingSourcePayload(tc.extraData))
			require.Equal(t, tc.eip1559Params, attrs.EIP1559Params)
			require.Equal(t, tc.minBaseFee, attrs.MinBaseFee)
		})
	}
}

func siblingParams(denominator, elasticity uint64) *eth.Bytes8 {
	params := eth.Bytes8(eip1559.EncodeHolocene1559Params(denominator, elasticity))
	return &params
}
