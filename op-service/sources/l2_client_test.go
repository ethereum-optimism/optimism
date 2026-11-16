package sources

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// testRPCBlock returns a verifiable RPC block with the given raw transactions.
func testRPCBlock(txs RawTransactions) *RPCBlock {
	block := &RPCBlock{
		RPCHeader: RPCHeader{
			UncleHash: types.EmptyUncleHash,
			Number:    123,
			Time:      1_700_000_000,
			BaseFee:   (*hexutil.Big)(big.NewInt(7)),
			TxHash:    types.DeriveSha(txs, trie.NewStackTrie(nil)),
		},
		Transactions: txs,
	}
	block.Hash = block.computeBlockHash()
	return block
}

// newL2ClientServing returns an L2Client whose RPC answers one block call, made with method and
// block argument arg, with block. The test fails if that call is not made.
func newL2ClientServing(t *testing.T, block *RPCBlock, method string, arg any) *L2Client {
	m := new(mockRPC)
	m.On("CallContext", mock.Anything, new(*RPCBlock), method, []any{arg, true}).Run(func(args mock.Arguments) {
		*args[1].(**RPCBlock) = block
	}).Return([]error{nil}).Once()
	t.Cleanup(func() { m.AssertExpectations(t) })
	s, err := NewL2Client(m, nil, nil, L2ClientSimpleConfig(&rollup.Config{}, false, 10, 10))
	require.NoError(t, err)
	return s
}

// TestL2ClientInfoAndTxsRejectsOpTxs pins that the L2Client accessors decoding a whole block
// body reject a Lagoon-shaped L2 block — L1-info deposit, user tx, trailing post-exec tx — and
// blocks holding only one of the two OP Stack transaction types, by their type bytes, while the
// user-tx accessor serves the Lagoon-shaped block.
func TestL2ClientInfoAndTxsRejectsOpTxs(t *testing.T) {
	ctx := t.Context()

	deposit, err := (&optypes.DepositTx{
		SourceHash: common.Hash{0x01},
		From:       common.Address{0x11},
		Value:      big.NewInt(0),
		Gas:        1_000_000,
		Data:       []byte{0xde, 0xad},
	}).MarshalBinary()
	require.NoError(t, err)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	chainID := big.NewInt(10)
	to := common.Address{0x42}
	userTx := types.MustSignNewTx(key, types.LatestSignerForChainID(chainID), &types.DynamicFeeTx{
		ChainID: chainID, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(5), Gas: 21_000, To: &to, Value: big.NewInt(1),
	})
	user, err := userTx.MarshalBinary()
	require.NoError(t, err)
	postExec, err := (&optypes.PostExecTx{Data: []byte{0xc2, 0x80, 0x80}}).MarshalBinary()
	require.NoError(t, err)

	block := testRPCBlock(RawTransactions{deposit, user, postExec})

	for _, rc := range []struct {
		name  string
		block *RPCBlock
	}{
		{"lagoon block", block},
		{"deposit only", testRPCBlock(RawTransactions{deposit, user})},
		{"post-exec only", testRPCBlock(RawTransactions{user, postExec})},
	} {
		rejected := rc.block
		byNumber := hexutil.EncodeUint64(uint64(rejected.Number))
		rejecting := []struct {
			name   string
			method string
			arg    any
			read   func(*L2Client) error
		}{
			{"HeaderAndTxsByHash", "eth_getBlockByHash", rejected.Hash, func(s *L2Client) error {
				_, _, err := s.HeaderAndTxsByHash(ctx, rejected.Hash)
				return err
			}},
			{"InfoAndTxsByHash", "eth_getBlockByHash", rejected.Hash, func(s *L2Client) error {
				_, _, err := s.InfoAndTxsByHash(ctx, rejected.Hash)
				return err
			}},
			{"InfoAndTxsByNumber", "eth_getBlockByNumber", byNumber, func(s *L2Client) error {
				_, _, err := s.InfoAndTxsByNumber(ctx, uint64(rejected.Number))
				return err
			}},
			{"InfoAndTxsByLabel", "eth_getBlockByNumber", string(eth.Unsafe), func(s *L2Client) error {
				_, _, err := s.InfoAndTxsByLabel(ctx, eth.Unsafe)
				return err
			}},
		}
		for _, tc := range rejecting {
			t.Run(rc.name+"/"+tc.name, func(t *testing.T) {
				require.ErrorIs(t, tc.read(newL2ClientServing(t, rejected, tc.method, tc.arg)), ErrOpTxInBody)
			})
		}
	}

	t.Run("InfoAndUserTxsByHash", func(t *testing.T) {
		s := newL2ClientServing(t, block, "eth_getBlockByHash", block.Hash)
		info, txs, err := s.InfoAndUserTxsByHash(ctx, block.Hash)
		require.NoError(t, err)
		require.Equal(t, block.Hash, info.Hash())
		require.Len(t, txs, 1)
		require.Equal(t, userTx.Hash(), txs[0].Hash())
	})

	t.Run("decodes a body without OP Stack txs", func(t *testing.T) {
		userOnly := testRPCBlock(RawTransactions{user})
		s := newL2ClientServing(t, userOnly, "eth_getBlockByHash", userOnly.Hash)
		_, txs, err := s.InfoAndTxsByHash(ctx, userOnly.Hash)
		require.NoError(t, err)
		require.Len(t, txs, 1)
		require.Equal(t, userTx.Hash(), txs[0].Hash())
	})
}
