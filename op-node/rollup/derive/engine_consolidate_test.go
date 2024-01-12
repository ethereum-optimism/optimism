package derive

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	validParentHash       = common.HexToHash("0x123")
	validTimestamp        = eth.Uint64Quantity(123)
	validParentBeaconRoot = common.HexToHash("0x456")
	validPrevRandao       = eth.Bytes32(common.HexToHash("0x789"))
	validGasLimit         = eth.Uint64Quantity(1000)
	validWithdrawals      = types.Withdrawals{}
)

type args struct {
	envelope   *eth.ExecutionPayloadEnvelope
	attrs      *eth.PayloadAttributes
	parentHash common.Hash
}

func ecotoneArgs() args {
	return args{
		envelope: &eth.ExecutionPayloadEnvelope{
			ParentBeaconBlockRoot: &validParentBeaconRoot,
			ExecutionPayload: &eth.ExecutionPayload{
				ParentHash:  validParentHash,
				Timestamp:   validTimestamp,
				PrevRandao:  validPrevRandao,
				GasLimit:    validGasLimit,
				Withdrawals: &validWithdrawals,
			},
		},
		attrs: &eth.PayloadAttributes{
			Timestamp:             validTimestamp,
			PrevRandao:            validPrevRandao,
			GasLimit:              &validGasLimit,
			ParentBeaconBlockRoot: &validParentBeaconRoot,
			Withdrawals:           &validWithdrawals,
		},
		parentHash: validParentHash,
	}
}

func canyonArgs() args {
	args := ecotoneArgs()
	args.attrs.ParentBeaconBlockRoot = nil
	args.envelope.ParentBeaconBlockRoot = nil
	return args
}

func bedrockArgs() args {
	args := ecotoneArgs()
	args.attrs.Withdrawals = nil
	args.envelope.ExecutionPayload.Withdrawals = nil
	return args
}

func ecotoneNoParentBeaconBlockRoot() args {
	args := ecotoneArgs()
	args.envelope.ParentBeaconBlockRoot = nil
	return args
}

func mismatchedParentHashArgs() args {
	args := ecotoneArgs()
	args.parentHash = common.HexToHash("0xabc")
	return args
}

func createMistmatchedPrevRandao() args {
	args := ecotoneArgs()
	args.attrs.PrevRandao = eth.Bytes32(common.HexToHash("0xabc"))
	return args
}

func createMismatchedGasLimit() args {
	args := ecotoneArgs()
	val := eth.Uint64Quantity(2000)
	args.attrs.GasLimit = &val
	return args
}

func createNilGasLimit() args {
	args := ecotoneArgs()
	args.attrs.GasLimit = nil
	return args
}

func createMistmatchedTimestamp() args {
	args := ecotoneArgs()
	val := eth.Uint64Quantity(2000)
	args.attrs.Timestamp = val
	return args
}

func TestAttributesMatch(t *testing.T) {
	rollupCfg := &rollup.Config{}

	tests := []struct {
		shouldMatch bool
		args        args
	}{
		{
			shouldMatch: true,
			args:        ecotoneArgs(),
		},
		{
			shouldMatch: true,
			args:        canyonArgs(),
		},
		{
			shouldMatch: true,
			args:        bedrockArgs(),
		},
		{
			shouldMatch: false,
			args:        mismatchedParentHashArgs(),
		},
		{
			shouldMatch: false,
			args:        ecotoneNoParentBeaconBlockRoot(),
		},
		{
			shouldMatch: false,
			args:        createMistmatchedPrevRandao(),
		},
		{
			shouldMatch: false,
			args:        createMismatchedGasLimit(),
		},
		{
			shouldMatch: false,
			args:        createNilGasLimit(),
		},
		{
			shouldMatch: false,
			args:        createMistmatchedTimestamp(),
		},
	}

	for _, test := range tests {
		err := AttributesMatchBlock(rollupCfg, test.args.attrs, test.args.parentHash, test.args.envelope, testlog.Logger(t, log.LvlInfo))
		if test.shouldMatch {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestWithdrawalsMatch(t *testing.T) {
	tests := []struct {
		attrs       *types.Withdrawals
		block       *types.Withdrawals
		shouldMatch bool
	}{
		{
			attrs:       nil,
			block:       nil,
			shouldMatch: true,
		},
		{
			attrs:       &types.Withdrawals{},
			block:       nil,
			shouldMatch: false,
		},
		{
			attrs:       nil,
			block:       &types.Withdrawals{},
			shouldMatch: false,
		},
		{
			attrs:       &types.Withdrawals{},
			block:       &types.Withdrawals{},
			shouldMatch: true,
		},
		{
			attrs: &types.Withdrawals{
				{
					Index: 1,
				},
			},
			block:       &types.Withdrawals{},
			shouldMatch: false,
		},
		{
			attrs: &types.Withdrawals{
				{
					Index: 1,
				},
			},
			block: &types.Withdrawals{
				{
					Index: 2,
				},
			},
			shouldMatch: false,
		},
	}

	for _, test := range tests {
		err := checkWithdrawalsMatch(test.attrs, test.block)

		if test.shouldMatch {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
