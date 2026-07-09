package chain_container

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/virtual_node"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func newMessage(chainID, blockNum uint64, logIdx uint32, ts uint64) *messages.Message {
	return &messages.Message{
		Identifier: messages.Identifier{
			Origin:      params.InteropCrossL2InboxAddress,
			ChainID:     eth.ChainIDFromUInt64(chainID),
			BlockNumber: blockNum,
			LogIndex:    logIdx,
			Timestamp:   ts,
		},
		PayloadHash: common.BytesToHash([]byte{byte(blockNum), byte(logIdx), byte(ts)}),
	}
}

func TestL2BlockReceiptsDecode(t *testing.T) {
	m := newMessage(10, 100, 0, 5000)
	blk := &L2Block{ExecMsgs: map[uint32]*messages.Message{0: m}}

	rcpts := blk.Receipts()
	require.Len(t, rcpts, 1)
	require.Len(t, rcpts[0].Logs, 1)

	decoded, err := messages.DecodeExecutingMessageLog(rcpts[0].Logs[0])
	require.NoError(t, err)
	require.Equal(t, m.Identifier.ChainID, decoded.ChainID)
	require.Equal(t, m.Identifier.BlockNumber, decoded.BlockNum)
	require.Equal(t, m.Checksum(), decoded.Checksum)
}

func TestL2BlockOutput(t *testing.T) {
	wRoot := common.HexToHash("0x1234")
	blk := &L2Block{
		Payload: &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: &eth.ExecutionPayload{
				StateRoot:       eth.Bytes32(common.HexToHash("0xdead")),
				WithdrawalsRoot: &wRoot,
				BlockHash:       common.HexToHash("0xabcd"),
			},
		},
	}

	out := blk.Output()
	require.Equal(t, eth.Bytes32(common.HexToHash("0xdead")), out.StateRoot)
	require.Equal(t, eth.Bytes32(wRoot), out.MessagePasserStorageRoot)
	require.Equal(t, common.HexToHash("0xabcd"), out.BlockHash)
}

func TestRandomChainSafeDB(t *testing.T) {
	rc := &RandomChain{
		safeDB: []SafeHeadEntry{
			{L1: eth.BlockID{Number: 100}, L2: eth.BlockID{Number: 5}},
			{L1: eth.BlockID{Number: 110}, L2: eth.BlockID{Number: 8}},
		},
	}

	_, err := rc.L1AtSafeHead(context.Background(), eth.BlockID{Number: 8})
	require.ErrorIs(t, err, virtual_node.ErrVirtualNodeNotRunning)

	require.NoError(t, rc.Start(context.Background()))

	l1, err := rc.L1AtSafeHead(context.Background(), eth.BlockID{Number: 8})
	require.NoError(t, err)
	require.Equal(t, uint64(110), l1.Number)

	_, err = rc.L1AtSafeHead(context.Background(), eth.BlockID{Number: 9})
	require.ErrorIs(t, err, safedb.ErrL1AtSafeHeadNotFound)

	gL1, gL2, err := rc.FirstSafeHeadEntry(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(100), gL1.Number)
	require.Equal(t, uint64(5), gL2.Number)

	aL1, aL2, err := rc.SafeHeadAtL1(context.Background(), 105)
	require.NoError(t, err)
	require.Equal(t, uint64(100), aL1.Number)
	require.Equal(t, uint64(5), aL2.Number)
}

func TestRandomChainL2Provider(t *testing.T) {
	wRoot := common.HexToHash("0x1234")
	blkHash := common.HexToHash("0xfeed")
	rc := &RandomChain{
		safe: 0,
		l2: []L2Block{{
			Ref: eth.L2BlockRef{Hash: blkHash, Number: 0, Time: 1234},
			Payload: &eth.ExecutionPayloadEnvelope{
				ExecutionPayload: &eth.ExecutionPayload{
					StateRoot:       eth.Bytes32(common.HexToHash("0xdead")),
					WithdrawalsRoot: &wRoot,
					BlockHash:       blkHash,
				},
			},
			ExecMsgs: map[uint32]*messages.Message{0: newMessage(10, 0, 0, 1234)},
		}},
	}

	ref, err := rc.L2BlockRefByNumber(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, blkHash, ref.Hash)

	labelRef, err := rc.L2BlockRefByLabel(context.Background(), eth.Safe)
	require.NoError(t, err)
	require.Equal(t, blkHash, labelRef.Hash)

	_, err = rc.L2BlockRefByNumber(context.Background(), 1)
	require.ErrorIs(t, err, ethereum.NotFound)

	out, err := rc.OutputV0AtBlock(context.Background(), blkHash)
	require.NoError(t, err)
	require.Equal(t, blkHash, out.BlockHash)

	info, rcpts, err := rc.FetchReceipts(context.Background(), blkHash)
	require.NoError(t, err)
	require.Equal(t, blkHash, info.Hash())
	require.Len(t, rcpts[0].Logs, 1)

	_, _, err = rc.FetchReceipts(context.Background(), common.HexToHash("0x0"))
	require.ErrorIs(t, err, ethereum.NotFound)
}

func TestRandomChainManagerAccessors(t *testing.T) {
	a := eth.ChainIDFromUInt64(1)
	b := eth.ChainIDFromUInt64(2)
	m := &RandomChainManager{
		chains: map[eth.ChainID]*RandomChain{a: {chainID: a}, b: {chainID: b}},
		order:  []eth.ChainID{a, b},
	}
	m.l1Source = &RandomL1Source{parent: m}

	require.Equal(t, a, m.Chain(a).chainID)
	require.Nil(t, m.Chain(eth.ChainIDFromUInt64(99)))

	chains := m.Chains()
	require.Len(t, chains, 2)
	require.Equal(t, a, chains[0].chainID)
	require.Equal(t, b, chains[1].chainID)

	require.Same(t, m, m.L1Source().parent)
}

func TestRandomChainManagerGenerate(t *testing.T) {
	m := NewRandomChainManager([]byte("seed-abc"))
	m.Generate()

	chains := m.Chains()
	require.GreaterOrEqual(t, len(chains), 2)
	require.LessOrEqual(t, len(chains), 4)

	for _, rc := range chains {
		require.GreaterOrEqual(t, len(rc.l2), 4)
		require.LessOrEqual(t, len(rc.l2), 16)
		require.Zero(t, len(rc.l2)%2, "depths are drawn even")

		for i := 1; i < len(rc.l2); i++ {
			require.Equal(t, rc.l2[i-1].Ref.Hash, rc.l2[i].Ref.ParentHash, "L2 blocks must link")
		}
		for i := 1; i < len(rc.safeDB); i++ {
			require.Greater(t, rc.safeDB[i].L1.Number, rc.safeDB[i-1].L1.Number)
			require.Greater(t, rc.safeDB[i].L2.Number, rc.safeDB[i-1].L2.Number)
		}
		require.Less(t, rc.unsafe, uint64(len(rc.l2)))
		require.Equal(t, rc.safe, rc.safeDB[len(rc.safeDB)-1].L2.Number,
			"SafeDB must reach the safe head")
		require.Greater(t, rc.currentL1, rc.safeDB[0].L1.Number,
			"currentL1 must be above the first SafeDB row")

		require.NoError(t, rc.Start(context.Background()))
		l1, err := rc.L1AtSafeHead(context.Background(), rc.safeDB[0].L2)
		require.NoError(t, err)
		require.Equal(t, rc.safeDB[0].L1.Number, l1.Number)
	}

	// all chains share the one canonical L1
	require.Equal(t, m.l1, chains[0].l1)
	require.Equal(t, chains[0].l1[0].Hash, chains[1].l1[0].Hash)

	// L1Source serves the canonical L1
	ref, err := m.L1Source().L1BlockRefByNumber(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, m.l1[1].Hash, ref.Hash)
	_, err = m.L1Source().L1BlockRefByNumber(context.Background(), 99)
	require.ErrorIs(t, err, ethereum.NotFound)

	// same seed -> same data
	m2 := NewRandomChainManager([]byte("seed-abc"))
	m2.Generate()
	require.Equal(t, chains[0].l2[0].Ref.Hash, m2.Chains()[0].l2[0].Ref.Hash)
}

func TestGeneratedExecutingMessages(t *testing.T) {
	m := NewRandomChainManager([]byte("xmsg"))
	m.Generate()
	chains := m.Chains()
	require.GreaterOrEqual(t, len(chains), 2)

	for _, rc := range chains {
		first := rc.safeDB[0].L2.Number
		for i, blk := range rc.l2 {
			if uint64(i) <= first || uint64(i) > rc.safe {
				require.Empty(t, blk.ExecMsgs)
				continue
			}
			require.Len(t, blk.ExecMsgs, 1)
			msg := blk.ExecMsgs[1]
			require.NotEqual(t, rc.chainID, msg.Identifier.ChainID)
			require.Less(t, msg.Identifier.Timestamp, blk.Ref.Time)

			src := m.Chain(msg.Identifier.ChainID)
			require.NotNil(t, src)
			require.GreaterOrEqual(t, msg.Identifier.BlockNumber, src.safeDB[0].L2.Number,
				"init block must be verifiable on the source chain")
			initLog := src.l2[msg.Identifier.BlockNumber].InitLog
			require.Equal(t, initLog.Address, msg.Identifier.Origin)
			require.Equal(t, uint32(0), msg.Identifier.LogIndex)
			require.Equal(t,
				crypto.Keccak256Hash(messages.LogToMessagePayload(initLog)),
				msg.PayloadHash)
		}
	}

	// init logs must not read as executing-message events
	em, err := messages.DecodeExecutingMessageLog(chains[0].l2[1].InitLog)
	require.NoError(t, err)
	require.Nil(t, em)
}

func TestBreakOneExecMsg(t *testing.T) {
	// Some topologies have no reachable executing message (a shallow chain gates
	// verification before any cross-chain block); find a seed that does.
	type key struct {
		id  eth.ChainID
		blk int
	}
	var m *RandomChainManager
	var before map[key]messages.Message
	var id eth.ChainID
	var ts uint64
	var ok bool
	for s := 0; s < 50 && !ok; s++ {
		m = NewRandomChainManager([]byte{byte(s)})
		m.Generate()
		before = map[key]messages.Message{}
		for _, rc := range m.Chains() {
			for i := range rc.l2 {
				if msg := rc.l2[i].ExecMsgs[1]; msg != nil {
					before[key{rc.chainID, i}] = *msg
				}
			}
		}
		id, ts, ok = m.BreakOneExecMsg()
	}
	require.True(t, ok, "no seed produced a reachable executing message in 50 tries")

	// exactly one message changed (any breaker), on chain id at time ts
	changed := 0
	for _, rc := range m.Chains() {
		for i := range rc.l2 {
			msg := rc.l2[i].ExecMsgs[1]
			if msg == nil {
				continue
			}
			if *msg != before[key{rc.chainID, i}] {
				changed++
				require.Equal(t, id, rc.chainID)
				require.Equal(t, ts, rc.l2[i].Ref.Time)
			}
		}
	}
	require.Equal(t, 1, changed)
}

func TestBreakOneL1Divergence(t *testing.T) {
	var m *RandomChainManager
	var l1Snap []eth.L1BlockRef
	var id eth.ChainID
	var ts uint64
	var ok bool
	for s := 0; s < 20 && !ok; s++ {
		m = NewRandomChainManager([]byte{byte(s)})
		m.Generate()
		l1Snap = make([]eth.L1BlockRef, len(m.l1))
		copy(l1Snap, m.l1)
		id, ts, ok = m.BreakOneL1Divergence()
	}
	require.True(t, ok, "no seed produced a reachable safeDB row in 20 tries")

	// Mutation is chain-local: the canonical L1 the L1Source serves is untouched.
	require.Equal(t, l1Snap, m.l1, "canonical L1 must be unchanged")
	require.LessOrEqual(t, ts, m.MinSafeTimestamp(), "badTS must be reachable")

	// Exactly one safeDB row diverges from canonical, on the returned chain.
	diverged := 0
	for _, rc := range m.Chains() {
		for _, row := range rc.safeDB {
			if row.L1.Hash != l1Snap[row.L1.Number].Hash {
				diverged++
				require.Equal(t, id, rc.chainID, "diverged row must be on the returned chain")
				require.Less(t, row.L1.Number, uint64(len(l1Snap)), "diverged row still indexes a canonical block")
			}
		}
	}
	require.Equal(t, 1, diverged, "exactly one safeDB row must diverge")
}

func TestBreakOneSafeDBFrontGap(t *testing.T) {
	m := NewRandomChainManager([]byte("safedb"))
	m.Generate()

	startTS := m.FirstVerifiableTimestamp()
	before := map[eth.ChainID]uint64{}
	for _, rc := range m.Chains() {
		before[rc.chainID] = rc.safeDB[0].L2.Number
	}

	id, ts, ok := m.BreakOneSafeDBFrontGap()
	require.True(t, ok)
	require.Equal(t, startTS, ts)

	bumped := 0
	for _, rc := range m.Chains() {
		got := rc.safeDB[0].L2.Number
		if got != before[rc.chainID] {
			bumped++
			require.Equal(t, id, rc.chainID)
			require.Equal(t, before[rc.chainID]+1, got)
			if len(rc.safeDB) >= 2 {
				require.Less(t, rc.safeDB[0].L2.Number, rc.safeDB[1].L2.Number)
			}
		}
	}
	require.Equal(t, 1, bumped)
}

func TestReorgL1(t *testing.T) {
	m := NewRandomChainManager([]byte("reorg-test"))
	m.Generate()

	orig := make([]eth.L1BlockRef, len(m.l1))
	copy(orig, m.l1)

	fork := uint64(2)
	m.ReorgL1(fork)

	require.Equal(t, len(orig), len(m.l1), "length must be unchanged")

	for i := range m.l1 {
		require.Equal(t, orig[i].Number, m.l1[i].Number, "Number must be preserved at index %d", i)
		require.Equal(t, orig[i].Time, m.l1[i].Time, "Time must be preserved at index %d", i)
	}

	// Entries before fork are untouched.
	for i := uint64(0); i < fork; i++ {
		require.Equal(t, orig[i].Hash, m.l1[i].Hash, "Hash must be unchanged before fork at index %d", i)
		require.Equal(t, orig[i].ParentHash, m.l1[i].ParentHash, "ParentHash must be unchanged before fork at index %d", i)
	}

	// Entries from fork onward have new hashes.
	for i := fork; i < uint64(len(m.l1)); i++ {
		require.NotEqual(t, orig[i].Hash, m.l1[i].Hash, "Hash must change at index %d", i)
	}

	// ParentHash chain is re-linked from the common ancestor.
	require.Equal(t, m.l1[fork-1].Hash, m.l1[fork].ParentHash, "ParentHash at fork must link to ancestor")
	for i := fork + 1; i < uint64(len(m.l1)); i++ {
		require.Equal(t, m.l1[i-1].Hash, m.l1[i].ParentHash, "ParentHash must link to prior at index %d", i)
	}
}

func TestRandomChainForkchoiceEngine(t *testing.T) {
	m := NewRandomChainManager([]byte("fc-engine"))
	m.Generate()
	rc := m.Chains()[0]
	ctx := context.Background()

	// Before ForkchoiceUpdate, L2BlockRefByLabel uses the index-based path.
	indexRef, err := rc.L2BlockRefByLabel(ctx, eth.Unsafe)
	require.NoError(t, err)
	require.Equal(t, rc.l2[rc.unsafe].Ref.Hash, indexRef.Hash)

	// Use a real block hash for safe/finalized and a synthetic hash for unsafe.
	syntheticHead := common.Hash{0xab}
	realSafeHash := rc.l2[rc.safe].Ref.Hash
	realFinalizedHash := rc.l2[rc.finalized].Ref.Hash

	state := &eth.ForkchoiceState{
		HeadBlockHash:      syntheticHead,
		SafeBlockHash:      realSafeHash,
		FinalizedBlockHash: realFinalizedHash,
	}
	res, err := rc.ForkchoiceUpdate(ctx, state, nil)
	require.NoError(t, err)
	require.Equal(t, eth.ExecutionValid, res.PayloadStatus.Status)

	// After ForkchoiceUpdate, L2BlockRefByLabel reflects the stored refs.
	unsafeRef, err := rc.L2BlockRefByLabel(ctx, eth.Unsafe)
	require.NoError(t, err)
	require.Equal(t, syntheticHead, unsafeRef.Hash, "synthetic head passes through")

	safeRef, err := rc.L2BlockRefByLabel(ctx, eth.Safe)
	require.NoError(t, err)
	require.Equal(t, realSafeHash, safeRef.Hash)

	finalizedRef, err := rc.L2BlockRefByLabel(ctx, eth.Finalized)
	require.NoError(t, err)
	require.Equal(t, realFinalizedHash, finalizedRef.Hash)

	// NewPayload always accepts.
	status, err := rc.NewPayload(ctx, &eth.ExecutionPayload{}, nil)
	require.NoError(t, err)
	require.Equal(t, eth.ExecutionValid, status.Status)
}

func TestChainContainerWiring(t *testing.T) {
	m := NewRandomChainManager([]byte("wire"))
	m.Generate()
	id := m.Chains()[0].chainID

	cc, err := m.ChainContainer(id, "")
	require.NoError(t, err)
	require.Equal(t, id, cc.ID())

	_, err = m.ChainContainer(eth.ChainIDFromUInt64(123456), "")
	require.ErrorIs(t, err, ErrUnknownChain)

	all, err := m.ChainContainers("")
	require.NoError(t, err)
	require.Len(t, all, len(m.Chains()))
	require.Equal(t, id, all[id].ID())
}
