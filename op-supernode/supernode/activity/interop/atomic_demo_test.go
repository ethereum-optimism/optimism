package interop

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Atomic demo regression fixtures. These exercise the actual graph builder;
// they do not simulate EVM execution or transaction-pool admission.
func TestAtomicDemoLogOrdering(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		execA, initA, execB, initB uint32
		cycle                      bool
	}{
		{"both_emit_before_validate", 1, 0, 1, 0, false},
		{"A_emits_first_B_validates_first", 1, 0, 0, 1, false},
		{"both_validate_before_emit", 0, 1, 0, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := buildCycleGraph(testTS, map[eth.ChainID]map[uint32]*messages.ExecutingMessage{
				testChainA: {tc.execA: {ChainID: testChainB, LogIdx: tc.initB, Timestamp: testTS}},
				testChainB: {tc.execB: {ChainID: testChainA, LogIdx: tc.initA, Timestamp: testTS}},
			})
			if tc.cycle {
				require.ErrorIs(t, checkCycle(graph), ErrCycle)
			} else {
				require.NoError(t, checkCycle(graph))
			}
		})
	}
}

// A requests B, consumes B's first result, then sends a second request to B.
// All A logs belong to one transaction, and all B logs to another transaction.
func TestAtomicDemoRepeatedRoundTrip(t *testing.T) {
	// A: request1(0), validate result1(1), request2(2), validate result2(3), completed(4).
	// B: validate request1(0), result1(1), validate request2(2), result2(3), validate completed(4).
	graph := buildCycleGraph(testTS, map[eth.ChainID]map[uint32]*messages.ExecutingMessage{
		testChainA: {
			1: {ChainID: testChainB, LogIdx: 1, Timestamp: testTS},
			3: {ChainID: testChainB, LogIdx: 3, Timestamp: testTS},
		},
		testChainB: {
			0: {ChainID: testChainA, LogIdx: 0, Timestamp: testTS},
			2: {ChainID: testChainA, LogIdx: 2, Timestamp: testTS},
			4: {ChainID: testChainA, LogIdx: 4, Timestamp: testTS},
		},
	})
	require.NoError(t, checkCycle(graph))
}

func TestAtomicDemoConstructMutualSignedReferences(t *testing.T) {
	// Fix block templates and application event payloads before signing either tx.
	// Each event represents a distinct leg of an agreed bundle, not an ExecutingMessage.
	logA := &types.Log{Address: common.HexToAddress("0x1001"), Topics: []common.Hash{crypto.Keccak256Hash([]byte("Leg(bytes32)"))}, Data: crypto.Keccak256([]byte("bundle-1-leg-A")), Index: 0, BlockNumber: 100}
	logB := &types.Log{Address: common.HexToAddress("0x1002"), Topics: logA.Topics, Data: crypto.Keccak256([]byte("bundle-1-leg-B")), Index: 0, BlockNumber: 200}
	messageFor := func(l *types.Log, chain eth.ChainID) *messages.Message {
		return &messages.Message{
			Identifier:  messages.Identifier{Origin: l.Address, BlockNumber: l.BlockNumber, LogIndex: uint32(l.Index), Timestamp: testTS, ChainID: chain},
			PayloadHash: crypto.Keccak256Hash(messages.LogToMessagePayload(l)),
		}
	}
	mA, mB := messageFor(logA, testChainA), messageFor(logB, testChainB)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sign := func(chain eth.ChainID, peer *messages.Message, nonce uint64) *types.Transaction {
		id := chain.ToBig()
		tx := types.NewTx(&types.DynamicFeeTx{ChainID: id, Nonce: nonce, Gas: 200000, GasFeeCap: big.NewInt(1000000000), GasTipCap: big.NewInt(1), To: &logA.Address,
			AccessList: types.AccessList{{Address: predeploys.CrossL2InboxAddr, StorageKeys: messages.EncodeAccessList([]messages.Access{peer.Access()})}},
		})
		signed, err := types.SignTx(tx, types.LatestSignerForChainID(id), key)
		require.NoError(t, err)
		return signed
	}
	txA, txB := sign(testChainA, mB, 0), sign(testChainB, mA, 0)
	// Attach real tx hashes as receipt metadata after signing: message commitments stay fixed.
	logA.TxHash, logB.TxHash = txA.Hash(), txB.Hash()
	logA.BlockHash, logB.BlockHash = common.HexToHash("0xaabb"), common.HexToHash("0xccdd")
	require.Equal(t, mA.Checksum(), messageFor(logA, testChainA).Checksum())
	require.Equal(t, mB.Checksum(), messageFor(logB, testChainB).Checksum())
	for _, pair := range []struct {
		tx   *types.Transaction
		peer *messages.Message
	}{{txA, mB}, {txB, mA}} {
		accesses, err := messages.DecodeAccessList(pair.tx.AccessList())
		require.NoError(t, err)
		require.Equal(t, []messages.Access{pair.peer.Access()}, accesses)
	}
	// Changing one tx's nonce changes its hash, but not the other's reference.
	require.NotEqual(t, txA.Hash(), sign(testChainA, mB, 1).Hash())
	require.Equal(t, mA.Access(), messageFor(logA, testChainA).Access())
	// The commitment still binds application data and location.
	logA.Data[0] ^= 1
	require.NotEqual(t, mA.Checksum(), messageFor(logA, testChainA).Checksum())
	logA.Data[0] ^= 1
	logA.Index++
	require.NotEqual(t, mA.Checksum(), messageFor(logA, testChainA).Checksum())
}
