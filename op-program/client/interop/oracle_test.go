package interop

import (
	"math/rand"
	"testing"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	interopTypes "github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	l2Types "github.com/ethereum-optimism/optimism/op-program/client/l2/types"
	"github.com/ethereum-optimism/optimism/op-program/client/mpt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/mock"
)

func TestConsolidateOracle_NoConsolidatedData(t *testing.T) {
	chainID := uint64(48294)
	rng := rand.New(rand.NewSource(1))

	t.Run("BlockByHash", func(t *testing.T) {
		mock := new(OracleMock)
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		block, _ := testutils.RandomBlock(rng, 1)
		mock.On("BlockByHash", block.Hash(), eth.ChainIDFromUInt64(chainID)).Return(block)
		actual := oracle.BlockByHash(block.Hash(), eth.ChainIDFromUInt64(chainID))
		require.Equal(t, block, actual)
		mock.AssertExpectations(t)
	})
	t.Run("OutputByRoot", func(t *testing.T) {
		mock := new(OracleMock)
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		root := common.Hash{0xaa}
		output := testutils.RandomOutputV0(rng)
		mock.On("OutputByRoot", root, eth.ChainIDFromUInt64(chainID)).Return(output)
		actual := oracle.OutputByRoot(root, eth.ChainIDFromUInt64(chainID))
		require.Equal(t, output, actual)
		mock.AssertExpectations(t)
	})
	t.Run("BlockDataByHash", func(t *testing.T) {
		mock := new(OracleMock)
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		block, _ := testutils.RandomBlock(rng, 1)
		mock.On("BlockDataByHash", block.Hash(), block.Hash(), eth.ChainIDFromUInt64(chainID)).Return(block)
		actual := oracle.BlockDataByHash(block.Hash(), block.Hash(), eth.ChainIDFromUInt64(chainID))
		require.Equal(t, block, actual)
		mock.AssertExpectations(t)
	})
	t.Run("ReceiptsByBlockHash", func(t *testing.T) {
		mock := new(OracleMock)
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		block, receipts := testutils.RandomBlock(rng, 1)
		mock.On("ReceiptsByBlockHash", block.Hash(), eth.ChainIDFromUInt64(chainID)).Return(block, types.Receipts(receipts))
		actual, actualReceipts := oracle.ReceiptsByBlockHash(block.Hash(), eth.ChainIDFromUInt64(chainID))
		require.Equal(t, block, actual)
		require.Equal(t, types.Receipts(receipts), actualReceipts)
		mock.AssertExpectations(t)
	})
	t.Run("NodeByHash", func(t *testing.T) {
		mock := new(OracleMock)
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		node := []byte{12, 3, 4}
		hash := common.Hash{0xaa}
		mock.On("NodeByHash", hash, eth.ChainIDFromUInt64(chainID)).Return(node)
		actual := oracle.NodeByHash(hash, eth.ChainIDFromUInt64(chainID))
		require.Equal(t, node, actual)
		mock.AssertExpectations(t)
	})
	t.Run("CodeByHash", func(t *testing.T) {
		mock := new(OracleMock)
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		code := []byte{12, 3, 4}
		hash := common.Hash{0xaa}
		mock.On("CodeByHash", hash, eth.ChainIDFromUInt64(chainID)).Return(code)
		actual := oracle.CodeByHash(hash, eth.ChainIDFromUInt64(chainID))
		require.Equal(t, code, actual)
		mock.AssertExpectations(t)
	})
	t.Run("TransitionStateByRoot", func(t *testing.T) {
		mock := new(OracleMock)
		ts := &interopTypes.TransitionState{SuperRoot: []byte{0xbb}}
		oracle := NewConsolidateOracle(mock, ts)
		root := common.Hash{0xaa}
		actual := oracle.TransitionStateByRoot(root)
		require.Equal(t, ts, actual)
		mock.AssertExpectations(t)
	})
	t.Run("Hinter", func(t *testing.T) {
		mock := new(OracleMock)
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		mock.On("Hinter").Return(new(OracleHinterStub))
		actual := oracle.Hinter()
		require.Equal(t, new(OracleHinterStub), actual)
		mock.AssertExpectations(t)
	})
}

func TestConsolidateOracle_WithConsolidatedData(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(48294)
	rng := rand.New(rand.NewSource(1))
	block, receipts := testutils.RandomBlock(rng, 1)
	mock := new(OracleMock)
	defer mock.AssertExpectations(t)

	t.Run("BlockByHash", func(t *testing.T) {
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		db := oracle.KeyValueStore()
		storeBlock(t, db, block, receipts)

		actual := oracle.BlockByHash(block.Hash(), chainID)
		require.Equal(t, block.Hash(), actual.Hash())

		require.Equal(t, len(block.Transactions()), len(actual.Transactions()))
		for i := range block.Transactions() {
			require.Equal(t, block.Transactions()[i].Hash(), actual.Transactions()[i].Hash())
		}
	})
	t.Run("ReceiptsByBlockHash", func(t *testing.T) {
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		db := oracle.KeyValueStore()
		storeBlock(t, db, block, receipts)

		actual, actualReceipts := oracle.ReceiptsByBlockHash(block.Hash(), chainID)
		require.Equal(t, block.Hash(), actual.Hash())
		require.Equal(t, len(receipts), len(actualReceipts))
		for i := range receipts {
			// compare only consensus fields
			a, err := receipts[i].MarshalBinary()
			require.NoError(t, err)
			b, err := actualReceipts[i].MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, a, b)
		}
	})
	t.Run("OutputByRoot", func(t *testing.T) {
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		db := oracle.KeyValueStore()
		key := common.Hash{0xaa}
		dbKey := preimage.Keccak256Key(key).PreimageKey()
		output := testutils.RandomOutputV0(rng).Marshal()
		require.NoError(t, db.Put(dbKey[:], output))

		actual := oracle.OutputByRoot(key, chainID)
		require.Equal(t, output, actual.Marshal())
	})
	t.Run("NodeByHash", func(t *testing.T) {
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		db := oracle.KeyValueStore()
		key := common.Hash{0xaa}
		require.NoError(t, db.Put(key[:], []byte{1, 2, 3}))
		storeBlock(t, db, block, receipts)

		actual := oracle.NodeByHash(key, chainID)
		require.Equal(t, []byte{1, 2, 3}, actual)
	})
	t.Run("CodeByHash", func(t *testing.T) {
		oracle := NewConsolidateOracle(mock, &interopTypes.TransitionState{})
		db := oracle.KeyValueStore()
		key := common.Hash{0xaa}
		require.NoError(t, db.Put(key[:], []byte{1, 2, 3}))
		storeBlock(t, db, block, receipts)

		actual := oracle.CodeByHash(key, chainID)
		require.Equal(t, []byte{1, 2, 3}, actual)
	})
}

func storeBlock(t *testing.T, kv l2.KeyValueStore, block *types.Block, receipts types.Receipts) {
	opaqueRcpts, err := eth.EncodeReceipts(receipts)
	require.NoError(t, err)
	_, nodes := mpt.WriteTrie(opaqueRcpts)
	for _, node := range nodes {
		key := preimage.Keccak256Key(crypto.Keccak256Hash(node)).PreimageKey()
		require.NoError(t, kv.Put(key[:], node))
	}

	opaqueTxs, err := eth.EncodeTransactions(block.Transactions())
	require.NoError(t, err)
	_, txsNodes := mpt.WriteTrie(opaqueTxs)
	for _, p := range txsNodes {
		key := preimage.Keccak256Key(crypto.Keccak256Hash(p)).PreimageKey()
		require.NoError(t, kv.Put(key[:], p))
	}

	headerRlp, err := rlp.EncodeToBytes(block.Header())
	require.NoError(t, err)
	key := preimage.Keccak256Key(block.Hash()).PreimageKey()
	require.NoError(t, kv.Put(key[:], headerRlp))
}

type OracleMock struct {
	mock.Mock
}

var _ l2.Oracle = &OracleMock{}

func (o *OracleMock) BlockByHash(blockHash common.Hash, chainID eth.ChainID) *gethTypes.Block {
	args := o.Called(blockHash, chainID)
	return args.Get(0).(*gethTypes.Block)
}

func (o *OracleMock) OutputByRoot(root common.Hash, chainID eth.ChainID) eth.Output {
	args := o.Called(root, chainID)
	return args.Get(0).(eth.Output)
}

func (o *OracleMock) BlockDataByHash(agreedBlockHash, blockHash common.Hash, chainID eth.ChainID) *gethTypes.Block {
	args := o.Called(agreedBlockHash, blockHash, chainID)
	return args.Get(0).(*gethTypes.Block)
}

func (o *OracleMock) TransitionStateByRoot(root common.Hash) *interopTypes.TransitionState {
	args := o.Called(root)
	return args.Get(0).(*interopTypes.TransitionState)
}

func (o *OracleMock) ReceiptsByBlockHash(blockHash common.Hash, chainID eth.ChainID) (*gethTypes.Block, gethTypes.Receipts) {
	args := o.Called(blockHash, chainID)
	return args.Get(0).(*gethTypes.Block), args.Get(1).(gethTypes.Receipts)
}

func (o *OracleMock) NodeByHash(nodeHash common.Hash, chainID eth.ChainID) []byte {
	args := o.Called(nodeHash, chainID)
	return args.Get(0).([]byte)
}

func (o *OracleMock) CodeByHash(codeHash common.Hash, chainID eth.ChainID) []byte {
	args := o.Called(codeHash, chainID)
	return args.Get(0).([]byte)
}

func (o *OracleMock) Hinter() l2Types.OracleHinter {
	args := o.Called()
	return args.Get(0).(l2Types.OracleHinter)
}

type OracleHinterStub struct {
}

var _ l2Types.OracleHinter = &OracleHinterStub{}

func (o *OracleHinterStub) HintBlockExecution(parentBlockHash common.Hash, attr eth.PayloadAttributes, chainID eth.ChainID) {
}

func (o *OracleHinterStub) HintWithdrawalsRoot(blockHash common.Hash, chainID eth.ChainID) {
}
