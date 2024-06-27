package attributes

import (
	"math/rand" // nosemgrep
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

var (
	validParentHash       = common.HexToHash("0x123")
	validTimestamp        = eth.Uint64Quantity(123)
	validParentBeaconRoot = common.HexToHash("0x456")
	validPrevRandao       = eth.Bytes32(common.HexToHash("0x789"))
	validGasLimit         = eth.Uint64Quantity(1000)
	validWithdrawals      = types.Withdrawals{}
	validFeeRecipient     = predeploys.SequencerFeeVaultAddr
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
				ParentHash:   validParentHash,
				Timestamp:    validTimestamp,
				PrevRandao:   validPrevRandao,
				GasLimit:     validGasLimit,
				Withdrawals:  &validWithdrawals,
				FeeRecipient: validFeeRecipient,
			},
		},
		attrs: &eth.PayloadAttributes{
			Timestamp:             validTimestamp,
			PrevRandao:            validPrevRandao,
			GasLimit:              &validGasLimit,
			ParentBeaconBlockRoot: &validParentBeaconRoot,
			Withdrawals:           &validWithdrawals,
			SuggestedFeeRecipient: validFeeRecipient,
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

func ecotoneUnexpectedParentBeaconBlockRoot() args {
	args := ecotoneArgs()
	args.attrs.ParentBeaconBlockRoot = nil
	return args
}

func ecotoneMismatchParentBeaconBlockRoot() args {
	args := ecotoneArgs()
	h := common.HexToHash("0xabc")
	args.attrs.ParentBeaconBlockRoot = &h
	return args
}

func ecotoneMismatchParentBeaconBlockRootPtr() args {
	args := ecotoneArgs()
	cpy := *args.attrs.ParentBeaconBlockRoot
	args.attrs.ParentBeaconBlockRoot = &cpy
	return args
}

func ecotoneNilParentBeaconBlockRoots() args {
	args := ecotoneArgs()
	args.attrs.ParentBeaconBlockRoot = nil
	args.envelope.ParentBeaconBlockRoot = nil
	return args
}

func mismatchedParentHashArgs() args {
	args := ecotoneArgs()
	args.parentHash = common.HexToHash("0xabc")
	return args
}

func createMismatchedPrevRandao() args {
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

func createMismatchedTimestamp() args {
	args := ecotoneArgs()
	val := eth.Uint64Quantity(2000)
	args.attrs.Timestamp = val
	return args
}

func createMismatchedFeeRecipient() args {
	args := ecotoneArgs()
	args.attrs.SuggestedFeeRecipient = common.Address{0xde, 0xad}
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
			args:        ecotoneUnexpectedParentBeaconBlockRoot(),
		},
		{
			shouldMatch: false,
			args:        ecotoneMismatchParentBeaconBlockRoot(),
		},
		{
			shouldMatch: true,
			args:        ecotoneMismatchParentBeaconBlockRootPtr(),
		},
		{
			shouldMatch: true,
			args:        ecotoneNilParentBeaconBlockRoots(),
		},
		{
			shouldMatch: false,
			args:        createMismatchedPrevRandao(),
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
			args:        createMismatchedTimestamp(),
		},
		{
			shouldMatch: false,
			args:        createMismatchedFeeRecipient(),
		},
	}

	for i, test := range tests {
		err := AttributesMatchBlock(rollupCfg, test.args.attrs, test.args.parentHash, test.args.envelope, testlog.Logger(t, log.LevelInfo))
		if test.shouldMatch {
			require.NoError(t, err, "fail %d", i)
		} else {
			require.Error(t, err, "fail %d", i)
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

func TestGetMissingTxnHashes(t *testing.T) {
	depositTxs := make([]*types.Transaction, 5)

	for i := 0; i < len(depositTxs); i++ {
		rng := rand.New(rand.NewSource(1234 + int64(i)))
		safeDeposit := testutils.GenerateDeposit(testutils.RandomHash(rng), rng)
		depositTxs[i] = types.NewTx(safeDeposit)
	}

	tests := []struct {
		safeTransactions            []hexutil.Bytes
		unsafeTransactions          []hexutil.Bytes
		expectedSafeMissingHashes   []common.Hash
		expectedUnsafeMissingHashes []common.Hash
		expectErr                   bool
	}{
		{
			safeTransactions:            []hexutil.Bytes{},
			unsafeTransactions:          []hexutil.Bytes{depositTxToBytes(t, depositTxs[0])},
			expectedSafeMissingHashes:   []common.Hash{depositTxs[0].Hash()},
			expectedUnsafeMissingHashes: []common.Hash{},
			expectErr:                   false,
		},
		{
			safeTransactions:            []hexutil.Bytes{depositTxToBytes(t, depositTxs[0])},
			unsafeTransactions:          []hexutil.Bytes{},
			expectedSafeMissingHashes:   []common.Hash{},
			expectedUnsafeMissingHashes: []common.Hash{depositTxs[0].Hash()},
			expectErr:                   false,
		},
		{
			safeTransactions: []hexutil.Bytes{
				depositTxToBytes(t, depositTxs[0]),
			},
			unsafeTransactions: []hexutil.Bytes{
				depositTxToBytes(t, depositTxs[0]),
				depositTxToBytes(t, depositTxs[1]),
				depositTxToBytes(t, depositTxs[2]),
			},
			expectedSafeMissingHashes: []common.Hash{
				depositTxs[1].Hash(),
				depositTxs[2].Hash(),
			},
			expectedUnsafeMissingHashes: []common.Hash{},
			expectErr:                   false,
		},
		{
			safeTransactions: []hexutil.Bytes{
				depositTxToBytes(t, depositTxs[0]),
				depositTxToBytes(t, depositTxs[1]),
				depositTxToBytes(t, depositTxs[2]),
			},
			unsafeTransactions: []hexutil.Bytes{
				depositTxToBytes(t, depositTxs[0]),
			},
			expectedSafeMissingHashes: []common.Hash{},
			expectedUnsafeMissingHashes: []common.Hash{
				depositTxs[1].Hash(),
				depositTxs[2].Hash(),
			},
			expectErr: false,
		},
		{
			safeTransactions: []hexutil.Bytes{
				depositTxToBytes(t, depositTxs[0]),
				depositTxToBytes(t, depositTxs[1]),
				depositTxToBytes(t, depositTxs[2]),
			},
			unsafeTransactions: []hexutil.Bytes{
				depositTxToBytes(t, depositTxs[2]),
				depositTxToBytes(t, depositTxs[3]),
				depositTxToBytes(t, depositTxs[4]),
			},
			expectedSafeMissingHashes: []common.Hash{
				depositTxs[3].Hash(),
				depositTxs[4].Hash(),
			},
			expectedUnsafeMissingHashes: []common.Hash{
				depositTxs[0].Hash(),
				depositTxs[1].Hash(),
			},
			expectErr: false,
		},
		{
			safeTransactions:            []hexutil.Bytes{{1, 2, 3}},
			unsafeTransactions:          []hexutil.Bytes{},
			expectedSafeMissingHashes:   []common.Hash{},
			expectedUnsafeMissingHashes: []common.Hash{},
			expectErr:                   true,
		},
		{
			safeTransactions:            []hexutil.Bytes{},
			unsafeTransactions:          []hexutil.Bytes{{1, 2, 3}},
			expectedSafeMissingHashes:   []common.Hash{},
			expectedUnsafeMissingHashes: []common.Hash{},
			expectErr:                   true,
		},
	}

	for _, test := range tests {
		missingSafeHashes, missingUnsafeHashes, err := getMissingTxnHashes(
			testlog.Logger(t, log.LevelError),
			test.safeTransactions,
			test.unsafeTransactions,
		)

		if test.expectErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.ElementsMatch(t, test.expectedSafeMissingHashes, missingSafeHashes)
			require.ElementsMatch(t, test.expectedUnsafeMissingHashes, missingUnsafeHashes)
		}
	}
}

func depositTxToBytes(t *testing.T, tx *types.Transaction) hexutil.Bytes {
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err)

	return txBytes
}
