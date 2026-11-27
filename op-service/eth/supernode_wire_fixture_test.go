package eth

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// lokahiTestdata holds the JSON fixtures lokahi, the Rust interop verifier, checks its
// superroot_atTimestamp and supernode_syncStatus answers against.
const lokahiTestdata = "../../rust/lokahi/testdata"

func fixtureHash(b byte) common.Hash {
	return common.Hash(bytes.Repeat([]byte{b}, 32))
}

func fixtureBytes32(b byte) Bytes32 {
	return Bytes32(fixtureHash(b))
}

func fixtureOutput(tag byte) *OutputV0 {
	return &OutputV0{
		StateRoot:                fixtureBytes32(tag + 1),
		MessagePasserStorageRoot: fixtureBytes32(tag + 2),
		BlockHash:                fixtureHash(tag + 3),
	}
}

// fixtureSuperRootAtTimestamp is a verified timestamp of two chains, the second of which replaced
// a denied block: its optimistic output is the denied block's, its super root output the
// replacement's.
func fixtureSuperRootAtTimestamp() SuperRootAtTimestampResponse {
	chainA, chainB := ChainIDFromUInt64(901), ChainIDFromUInt64(902)
	outputA, deniedB, replacementB := fixtureOutput(0xa0), fixtureOutput(0xb0), fixtureOutput(0xc0)
	super := NewSuperV1(1003,
		ChainIDAndOutput{ChainID: chainB, Output: OutputRoot(replacementB)},
		ChainIDAndOutput{ChainID: chainA, Output: OutputRoot(outputA)},
	)
	return SuperRootAtTimestampResponse{
		CurrentL1:                 BlockID{Hash: fixtureHash(0x11), Number: 7},
		CurrentSafeTimestamp:      1002,
		CurrentLocalSafeTimestamp: 1004,
		CurrentFinalizedTimestamp: 1000,
		OptimisticAtTimestamp: map[ChainID]OutputWithRequiredL1{
			chainA: {Output: outputA, OutputRoot: OutputRoot(outputA), RequiredL1: BlockID{Hash: fixtureHash(0x21), Number: 5}},
			chainB: {Output: deniedB, OutputRoot: OutputRoot(deniedB), RequiredL1: BlockID{Hash: fixtureHash(0x22), Number: 6}},
		},
		ChainIDs: []ChainID{chainA, chainB},
		Data: &SuperRootResponseData{
			VerifiedRequiredL1: BlockID{Hash: fixtureHash(0x22), Number: 6},
			Super:              super,
			SuperRoot:          SuperRoot(super),
		},
	}
}

func fixtureL1(tag byte, number uint64) L1BlockRef {
	return L1BlockRef{Hash: fixtureHash(tag), Number: number, ParentHash: fixtureHash(tag + 1), Time: 100 + number}
}

func fixtureL2(tag byte, number uint64) L2BlockRef {
	return L2BlockRef{
		Hash:           fixtureHash(tag),
		Number:         number,
		ParentHash:     fixtureHash(tag + 1),
		Time:           1000 + 2*number,
		L1Origin:       BlockID{Hash: fixtureHash(tag + 2), Number: number / 4},
		SequenceNumber: number % 4,
	}
}

func fixtureSyncStatus(tag byte, l1, unsafe, safe, finalized, localSafe uint64) SyncStatus {
	return SyncStatus{
		CurrentL1:          fixtureL1(tag, l1),
		CurrentL1Finalized: fixtureL1(tag+0x10, l1-4),
		HeadL1:             fixtureL1(tag+0x20, l1+2),
		SafeL1:             fixtureL1(tag+0x30, l1-2),
		FinalizedL1:        fixtureL1(tag+0x10, l1-4),
		UnsafeL2:           fixtureL2(tag+0x40, unsafe),
		SafeL2:             fixtureL2(tag+0x50, safe),
		FinalizedL2:        fixtureL2(tag+0x60, finalized),
		PendingSafeL2:      fixtureL2(tag+0x70, localSafe),
		LocalSafeL2:        fixtureL2(tag+0x70, localSafe),
	}
}

// fixtureSupernodeSyncStatus is the sync status of two chains: the aggregate is the lowest of
// each field.
func fixtureSupernodeSyncStatus() SuperNodeSyncStatusResponse {
	chainA, chainB := ChainIDFromUInt64(901), ChainIDFromUInt64(902)
	statusA := fixtureSyncStatus(0x01, 30, 12, 8, 4, 10)
	statusB := fixtureSyncStatus(0x02, 28, 14, 9, 3, 11)
	return SuperNodeSyncStatusResponse{
		Chains:             map[ChainID]SyncStatus{chainA: statusA, chainB: statusB},
		ChainIDs:           []ChainID{chainA, chainB},
		CurrentL1:          statusB.CurrentL1.ID(),
		SafeTimestamp:      statusA.SafeL2.Time,
		LocalSafeTimestamp: statusA.LocalSafeL2.Time,
		FinalizedTimestamp: statusB.FinalizedL2.Time,
	}
}

// TestSupernodeWireFixtures pins the JSON of op-supernode's superroot_atTimestamp and
// supernode_syncStatus answers in fixtures that lokahi's tests assert its answers against, so a
// change to either side's wire format fails a test. Set WRITE_LOKAHI_FIXTURES=1 to rewrite them.
func TestSupernodeWireFixtures(t *testing.T) {
	for name, value := range map[string]any{
		"superroot_at_timestamp.json": fixtureSuperRootAtTimestamp(),
		"supernode_sync_status.json":  fixtureSupernodeSyncStatus(),
	} {
		t.Run(name, func(t *testing.T) {
			want, err := json.MarshalIndent(value, "", "  ")
			require.NoError(t, err)
			path := filepath.Join(lokahiTestdata, name)
			if os.Getenv("WRITE_LOKAHI_FIXTURES") != "" {
				require.NoError(t, os.WriteFile(path, append(want, '\n'), 0o644))
			}
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			require.JSONEq(t, string(want), string(got))
		})
	}
}
