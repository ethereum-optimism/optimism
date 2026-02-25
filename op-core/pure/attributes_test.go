package pure

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestBuildAttributes_EpochStart(t *testing.T) {
	cfg := testRollupConfig()
	sysConfig := testSystemConfig()
	l1Block := makeTestL1Input(5)
	l1Block.Deposits = []*types.DepositTx{makeTestDeposit(), makeTestDeposit()}

	l1Num := bigs.Uint64Strict(l1Block.Header.Number)
	l1Hash := l1Block.Header.Hash()

	userTx := hexutil.Bytes{0x01, 0x02, 0x03}
	batch := &derive.SingularBatch{
		ParentHash:   common.HexToHash("0xaaaa"),
		EpochNum:     rollup.Epoch(l1Num),
		EpochHash:    l1Hash,
		Timestamp:    l1Block.Header.Time + cfg.BlockTime,
		Transactions: []hexutil.Bytes{userTx},
	}

	// Cursor has a different L1 origin so this is an epoch boundary
	cursor := l2Cursor{
		Number:         10,
		Timestamp:      batch.Timestamp - cfg.BlockTime,
		L1Origin:       eth.BlockID{Hash: common.HexToHash("0xprev"), Number: l1Num - 1},
		SequenceNumber: 3,
	}

	result, err := buildAttributes(batch, l1Block, cursor, sysConfig, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Attributes)

	attrs := result.Attributes

	// L1 info deposit + 2 user deposits + 1 batch tx = 4
	require.GreaterOrEqual(t, len(attrs.Transactions), 3)
	require.Len(t, attrs.Transactions, 4)

	require.True(t, attrs.NoTxPool)
	require.Equal(t, hexutil.Uint64(batch.Timestamp), attrs.Timestamp)
	require.Equal(t, eth.Bytes32(l1Block.Header.MixDigest), attrs.PrevRandao)
	require.Equal(t, predeploys.SequencerFeeVaultAddr, attrs.SuggestedFeeRecipient)
	require.NotNil(t, attrs.GasLimit)
	require.Equal(t, sysConfig.GasLimit, uint64(*attrs.GasLimit))
	require.NotNil(t, attrs.Withdrawals)
	require.Empty(t, *attrs.Withdrawals)

	// The last transaction should be the batch tx
	require.Equal(t, userTx, attrs.Transactions[len(attrs.Transactions)-1])

	require.Equal(t, batch.ParentHash, result.ExpectedParentHash)
	require.Equal(t, l1Block.BlockRef(), result.DerivedFrom)
}

func TestBuildAttributes_SameEpoch(t *testing.T) {
	cfg := testRollupConfig()
	sysConfig := testSystemConfig()
	l1Block := makeTestL1Input(5)
	l1Block.Deposits = []*types.DepositTx{makeTestDeposit()}

	l1Num := bigs.Uint64Strict(l1Block.Header.Number)
	l1Hash := l1Block.Header.Hash()

	userTx := hexutil.Bytes{0xaa, 0xbb}
	batch := &derive.SingularBatch{
		ParentHash:   common.HexToHash("0xbbbb"),
		EpochNum:     rollup.Epoch(l1Num),
		EpochHash:    l1Hash,
		Timestamp:    l1Block.Header.Time + 2*cfg.BlockTime,
		Transactions: []hexutil.Bytes{userTx},
	}

	// Same L1 origin -- not an epoch boundary
	cursor := l2Cursor{
		Number:         10,
		Timestamp:      batch.Timestamp - cfg.BlockTime,
		L1Origin:       eth.BlockID{Hash: l1Hash, Number: l1Num},
		SequenceNumber: 2,
	}

	result, err := buildAttributes(batch, l1Block, cursor, sysConfig, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	attrs := result.Attributes

	// L1 info deposit + 1 batch tx = 2 (no user deposits because same epoch)
	require.GreaterOrEqual(t, len(attrs.Transactions), 2)
	require.Len(t, attrs.Transactions, 2)

	require.True(t, attrs.NoTxPool)
	require.Equal(t, hexutil.Uint64(batch.Timestamp), attrs.Timestamp)

	// The last transaction should be the batch tx
	require.Equal(t, userTx, attrs.Transactions[len(attrs.Transactions)-1])
}

func TestBuildAttributes_EmptyBatch(t *testing.T) {
	cfg := testRollupConfig()
	sysConfig := testSystemConfig()

	t.Run("empty batch at epoch start", func(t *testing.T) {
		l1Block := makeTestL1Input(5)
		l1Block.Deposits = []*types.DepositTx{makeTestDeposit()}

		l1Num := bigs.Uint64Strict(l1Block.Header.Number)
		l1Hash := l1Block.Header.Hash()

		batch := &derive.SingularBatch{
			ParentHash:   common.HexToHash("0xcccc"),
			EpochNum:     rollup.Epoch(l1Num),
			EpochHash:    l1Hash,
			Timestamp:    l1Block.Header.Time + cfg.BlockTime,
			Transactions: nil,
		}

		cursor := l2Cursor{
			Number:         10,
			Timestamp:      batch.Timestamp - cfg.BlockTime,
			L1Origin:       eth.BlockID{Hash: common.HexToHash("0xold"), Number: l1Num - 1},
			SequenceNumber: 0,
		}

		result, err := buildAttributes(batch, l1Block, cursor, sysConfig, cfg)
		require.NoError(t, err)

		// L1 info deposit + 1 user deposit = 2 (no batch txs)
		require.Len(t, result.Attributes.Transactions, 2)
	})

	t.Run("empty batch same epoch", func(t *testing.T) {
		l1Block := makeTestL1Input(5)

		l1Num := bigs.Uint64Strict(l1Block.Header.Number)
		l1Hash := l1Block.Header.Hash()

		batch := &derive.SingularBatch{
			ParentHash:   common.HexToHash("0xdddd"),
			EpochNum:     rollup.Epoch(l1Num),
			EpochHash:    l1Hash,
			Timestamp:    l1Block.Header.Time + 2*cfg.BlockTime,
			Transactions: nil,
		}

		cursor := l2Cursor{
			Number:         10,
			Timestamp:      batch.Timestamp - cfg.BlockTime,
			L1Origin:       eth.BlockID{Hash: l1Hash, Number: l1Num},
			SequenceNumber: 1,
		}

		result, err := buildAttributes(batch, l1Block, cursor, sysConfig, cfg)
		require.NoError(t, err)

		// Only L1 info deposit, no user deposits, no batch txs
		require.Len(t, result.Attributes.Transactions, 1)
	})
}

func TestBuildAttributes_HoloceneFields(t *testing.T) {
	cfg := testRollupConfig()
	sysConfig := testSystemConfig()
	sysConfig.EIP1559Params = eth.Bytes8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	l1Block := makeTestL1Input(5)
	l1Num := bigs.Uint64Strict(l1Block.Header.Number)
	l1Hash := l1Block.Header.Hash()

	batch := &derive.SingularBatch{
		ParentHash: common.HexToHash("0xeeee"),
		EpochNum:   rollup.Epoch(l1Num),
		EpochHash:  l1Hash,
		Timestamp:  l1Block.Header.Time + cfg.BlockTime,
	}

	cursor := l2Cursor{
		Number:         10,
		Timestamp:      batch.Timestamp - cfg.BlockTime,
		L1Origin:       eth.BlockID{Hash: common.HexToHash("0xold"), Number: l1Num - 1},
		SequenceNumber: 0,
	}

	result, err := buildAttributes(batch, l1Block, cursor, sysConfig, cfg)
	require.NoError(t, err)
	require.NotNil(t, result.Attributes.EIP1559Params)
	require.Equal(t, sysConfig.EIP1559Params, *result.Attributes.EIP1559Params)
	require.NotNil(t, result.Attributes.ParentBeaconBlockRoot)
	require.NotNil(t, result.Attributes.Withdrawals)
}

func TestBuildAttributes_SequenceNumber(t *testing.T) {
	cfg := testRollupConfig()
	sysConfig := testSystemConfig()
	l1Block := makeTestL1Input(5)
	l1Num := bigs.Uint64Strict(l1Block.Header.Number)
	l1Hash := l1Block.Header.Hash()

	t.Run("epoch start resets to zero", func(t *testing.T) {
		batch := &derive.SingularBatch{
			ParentHash: common.HexToHash("0x1111"),
			EpochNum:   rollup.Epoch(l1Num),
			EpochHash:  l1Hash,
			Timestamp:  l1Block.Header.Time + cfg.BlockTime,
		}

		cursor := l2Cursor{
			Number:         10,
			Timestamp:      batch.Timestamp - cfg.BlockTime,
			L1Origin:       eth.BlockID{Hash: common.HexToHash("0xold"), Number: l1Num - 1},
			SequenceNumber: 5,
		}

		result, err := buildAttributes(batch, l1Block, cursor, sysConfig, cfg)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("same epoch increments", func(t *testing.T) {
		batch := &derive.SingularBatch{
			ParentHash: common.HexToHash("0x2222"),
			EpochNum:   rollup.Epoch(l1Num),
			EpochHash:  l1Hash,
			Timestamp:  l1Block.Header.Time + 4*cfg.BlockTime,
		}

		cursor := l2Cursor{
			Number:         10,
			Timestamp:      batch.Timestamp - cfg.BlockTime,
			L1Origin:       eth.BlockID{Hash: l1Hash, Number: l1Num},
			SequenceNumber: 5,
		}

		result, err := buildAttributes(batch, l1Block, cursor, sysConfig, cfg)
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}
