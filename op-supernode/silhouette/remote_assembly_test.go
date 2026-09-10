package silhouette

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

type testRemoteInteropSource struct {
	status *Status
	blocks map[uint64]*InteropBlock
}

func (s *testRemoteInteropSource) Status(context.Context) (*Status, error) { return s.status, nil }
func (s *testRemoteInteropSource) InteropBlock(_ context.Context, number uint64) (*InteropBlock, error) {
	return s.blocks[number], nil
}
func (*testRemoteInteropSource) MarkDenied(context.Context, uint64, common.Hash) error { return nil }
func (*testRemoteInteropSource) PruneDenied(context.Context, map[uint64][]common.Hash) error {
	return nil
}

func TestRemoteAssemblyWaitsForAuthorityRewindBeforeMirroringReplacement(t *testing.T) {
	sink, db := newSinkHarness(t)
	old := makeInteropChain(20, 31, common.HexToHash("0x1000"), 0x11)
	seed := make([]proofbatch.BlockExport, 0, len(old))
	for n := uint64(20); n <= 31; n++ {
		seed = append(seed, interopExport(old[n]))
	}
	require.NoError(t, sink.Accept(seed, old[20].ParentHash))

	// The L1 reorg keeps blocks through 23, then the standalone EL has only re-derived through 29
	// on the replacement branch. The former algorithm asked it for old block 30 forever.
	replacement := make(map[uint64]*InteropBlock)
	for n := uint64(20); n <= 23; n++ {
		replacement[n] = old[n]
	}
	parent := old[23].Hash
	for n := uint64(24); n <= 29; n++ {
		hash := common.BytesToHash([]byte{0x22, byte(n)})
		replacement[n] = &InteropBlock{
			Number: hexutil.Uint64(n), Timestamp: hexutil.Uint64(1_000 + n),
			ParentHash: parent, Hash: hash, ExecMsgsKnown: true,
		}
		parent = hash
	}
	oldest, head := hexutil.Uint64(20), hexutil.Uint64(29)
	source := &testRemoteInteropSource{
		status: &Status{OldestFact: &oldest, HeadFact: &head, CanonicalHead: eth.BlockID{Number: 29, Hash: replacement[29].Hash}},
		blocks: replacement,
	}
	assembly := &RemoteAssembly{ChainID: eth.ChainIDFromUInt64(902), Source: source, log: testlog.Logger(t, 3), sink: sink}

	// The mirror must leave the old seals intact until stock interop applies its normal rewind plan.
	require.NoError(t, assembly.sync(context.Background()))
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, eth.BlockID{Number: 31, Hash: old[31].Hash}, latest)
	seal, err := db.FindSealedBlock(24)
	require.NoError(t, err)
	require.Equal(t, old[24].Hash, seal.Hash)

	// Once stock interop has rewound to the shared head, the mirror appends the replacement branch.
	require.NoError(t, db.Rewind(eth.BlockID{Number: 23, Hash: old[23].Hash}))
	require.NoError(t, assembly.sync(context.Background()))
	latest, ok = db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, eth.BlockID{Number: 29, Hash: replacement[29].Hash}, latest)
	seal, err = db.FindSealedBlock(24)
	require.NoError(t, err)
	require.Equal(t, replacement[24].Hash, seal.Hash, "the replacement branch must be mirrored")
}

func makeInteropChain(first, last uint64, parent common.Hash, marker byte) map[uint64]*InteropBlock {
	out := make(map[uint64]*InteropBlock, last-first+1)
	for n := first; n <= last; n++ {
		hash := common.BytesToHash([]byte{marker, byte(n)})
		out[n] = &InteropBlock{
			Number: hexutil.Uint64(n), Timestamp: hexutil.Uint64(1_000 + n),
			ParentHash: parent, Hash: hash, ExecMsgsKnown: true,
		}
		parent = hash
	}
	return out
}

func interopExport(block *InteropBlock) proofbatch.BlockExport {
	return proofbatch.BlockExport{
		Number: uint64(block.Number), Timestamp: uint64(block.Timestamp), Hash: block.Hash, Logs: block.Logs,
	}
}
