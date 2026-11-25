package backend

import (
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// TestPayloadID checks that the payload ID is the same for equal inputs and differs when any
// one input differs, so that two different build requests never share an ID.
func TestPayloadID(t *testing.T) {
	parent := common.HexToHash("0x01")
	attributes := func() *eth.PayloadAttributes {
		return &eth.PayloadAttributes{
			Timestamp:             1234,
			SuggestedFeeRecipient: common.HexToAddress("0x03"),
			PrevRandao:            eth.Bytes32{4},
			Withdrawals:           &types.Withdrawals{},
			ParentBeaconBlockRoot: ptr.New(common.HexToHash("0x02")),
			NoTxPool:              true,
			Transactions:          []eth.Data{{0x02, 0xc0}, {0x02, 0xc1}},
			GasLimit:              ptr.New(eth.Uint64Quantity(30_000_000)),
			EIP1559Params:         &eth.Bytes8{0, 0, 0, 0xfa, 0, 0, 0, 6},
			MinBaseFee:            ptr.New(uint64(1e9)),
		}
	}
	id := computePayloadID(parent, attributes(), engine.PayloadV3)
	require.Equal(t, id, computePayloadID(parent, attributes(), engine.PayloadV3))
	require.Equal(t, byte(engine.PayloadV3), id[0])

	// Attributes without any optional field, as before Canyon. Nil withdrawals hash like an
	// empty list.
	bare := computePayloadID(parent, &eth.PayloadAttributes{}, engine.PayloadV1)
	require.NotEqual(t, id, bare)
	require.Equal(t, bare, computePayloadID(parent, &eth.PayloadAttributes{Withdrawals: &types.Withdrawals{}}, engine.PayloadV1))

	require.NotEqual(t, id, computePayloadID(common.HexToHash("0x05"), attributes(), engine.PayloadV3), "parent")
	require.NotEqual(t, id, computePayloadID(parent, attributes(), engine.PayloadV2), "version")
	for name, edit := range map[string]func(*eth.PayloadAttributes){
		"timestamp":         func(a *eth.PayloadAttributes) { a.Timestamp++ },
		"fee recipient":     func(a *eth.PayloadAttributes) { a.SuggestedFeeRecipient[0]++ },
		"prev randao":       func(a *eth.PayloadAttributes) { a.PrevRandao[0]++ },
		"withdrawals":       func(a *eth.PayloadAttributes) { a.Withdrawals = &types.Withdrawals{{Index: 1}} },
		"beacon root":       func(a *eth.PayloadAttributes) { a.ParentBeaconBlockRoot = nil },
		"no tx pool":        func(a *eth.PayloadAttributes) { a.NoTxPool = false },
		"transaction":       func(a *eth.PayloadAttributes) { a.Transactions[1] = eth.Data{0x02, 0xc2} },
		"transaction count": func(a *eth.PayloadAttributes) { a.Transactions = a.Transactions[:1] },
		"gas limit":         func(a *eth.PayloadAttributes) { *a.GasLimit++ },
		"EIP-1559 params":   func(a *eth.PayloadAttributes) { a.EIP1559Params[7]++ },
		"min base fee":      func(a *eth.PayloadAttributes) { *a.MinBaseFee++ },
	} {
		t.Run(name, func(t *testing.T) {
			attrs := attributes()
			edit(attrs)
			require.NotEqual(t, id, computePayloadID(parent, attrs, engine.PayloadV3))
		})
	}
}
