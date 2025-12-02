package rpc

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: add test for dependencySetV1 after devstack support is added to the QueryAPI

func TestRPCLocalUnsafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	client := sys.Supervisor.Escape()

	t.Run("fails with invalid chain ID", func(gt devtest.T) {
		_, err := client.QueryAPI().LocalUnsafe(context.Background(), eth.ChainIDFromUInt64(100))
		require.Error(t, err, "expected LocalUnsafe to fail with raw chain ID")
	})

	for _, chainID := range []eth.ChainID{sys.L2ChainA.ChainID(), sys.L2ChainB.ChainID()} {
		t.Run(fmt.Sprintf("succeeds with valid chain ID %d", chainID), func(gt devtest.T) {
			safe, err := client.QueryAPI().LocalUnsafe(context.Background(), chainID)
			require.NoError(t, err)
			assert.Greater(t, safe.Number, uint64(0))
			assert.Len(t, safe.Hash, 32)
		})
	}
}

func TestRPCCrossSafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	client := sys.Supervisor.Escape()

	t.Run("fails with invalid chain ID", func(gt devtest.T) {
		_, err := client.QueryAPI().CrossSafe(context.Background(), eth.ChainIDFromUInt64(100))
		require.Error(t, err, "expected CrossSafe to fail with invalid chain")
	})

	for _, chainID := range []eth.ChainID{sys.L2ChainA.ChainID(), sys.L2ChainB.ChainID()} {
		t.Run(fmt.Sprintf("succeeds with valid chain ID %d", chainID), func(gt devtest.T) {
			blockPair, err := client.QueryAPI().CrossSafe(context.Background(), chainID)
			require.NoError(t, err)
			assert.Greater(t, blockPair.Derived.Number, uint64(0))
			assert.Len(t, blockPair.Derived.Hash, 32)

			assert.Greater(t, blockPair.Source.Number, uint64(0))
			assert.Len(t, blockPair.Source.Hash, 32)
		})
	}
}

func TestRPCFinalized(gt *testing.T) {
	gt.Skip()
	t := devtest.ParallelT(gt)

	sys := presets.NewSimpleInterop(t)
	client := sys.Supervisor.Escape()

	t.Run("fails with invalid chain ID", func(gt devtest.T) {
		_, err := client.QueryAPI().Finalized(context.Background(), eth.ChainIDFromUInt64(100))
		require.Error(t, err, "expected Finalized to fail with invalid chain")
	})

	for _, chainID := range []eth.ChainID{sys.L2ChainA.ChainID(), sys.L2ChainB.ChainID()} {
		t.Run(fmt.Sprintf("succeeds with valid chain ID %d", chainID), func(gt devtest.T) {
			safe, err := client.QueryAPI().Finalized(context.Background(), chainID)
			require.NoError(t, err)
			assert.Greater(t, safe.Number, uint64(0))
			assert.Len(t, safe.Hash, 32)
		})
	}
}

func TestRPCFinalizedL1(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	client := sys.Supervisor.Escape()
	t.Run("succeeds to get finalized L1 block", func(gt devtest.T) {
		block, err := client.QueryAPI().FinalizedL1(context.Background())
		require.NoError(t, err)
		assert.Greater(t, block.Number, uint64(0))
		assert.Less(t, block.Time, uint64(time.Now().Unix()+5))
		assert.Len(t, block.Hash, 32)
	})
}

func TestRPCSuperRootAtTimestamp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	client := sys.Supervisor.Escape()

	t.Run("fails with invalid timestamp", func(gt devtest.T) {
		_, err := client.QueryAPI().SuperRootAtTimestamp(context.Background(), 0)
		require.Error(t, err)
	})

	t.Run("succeeds with valid timestamp", func(gt devtest.T) {
		timeNow := uint64(time.Now().Unix())
		root, err := client.QueryAPI().SuperRootAtTimestamp(context.Background(), hexutil.Uint64(timeNow-90))
		require.NoError(t, err)
		assert.Len(t, root.SuperRoot, 32)
		assert.Len(t, root.Chains, 2)

		for _, chain := range root.Chains {
			assert.Len(t, chain.Canonical, 32)
			assert.Contains(t, []eth.ChainID{sys.L2ChainA.ChainID(), sys.L2ChainB.ChainID()}, chain.ChainID)
		}
	})
}

func TestRPCAllSafeDerivedAt(gt *testing.T) {
	t := devtest.ParallelT(gt)

	sys := presets.NewSimpleInterop(t)
	client := sys.Supervisor.Escape()

	t.Run("fails with invalid L1 block hash", func(gt devtest.T) {
		_, err := client.QueryAPI().AllSafeDerivedAt(context.Background(), eth.BlockID{
			Number: 100,
			Hash:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		})
		require.Error(t, err)
	})

	t.Run("succeeds with valid synced L1 block hash", func(gt devtest.T) {
		sync, err := client.QueryAPI().SyncStatus(context.Background())
		require.NoError(t, err)

		allSafe, err := client.QueryAPI().AllSafeDerivedAt(context.Background(), eth.BlockID{
			Number: sync.MinSyncedL1.Number,
			Hash:   sync.MinSyncedL1.Hash,
		})
		require.NoError(t, err)

		require.Equal(t, 2, len(allSafe))
		for key, value := range allSafe {
			require.Contains(t, []eth.ChainID{sys.L2ChainA.ChainID(), sys.L2ChainB.ChainID()}, key)
			require.Len(t, value.Hash, 32)
		}
	})
}

func TestRPCCrossDerivedToSource(gt *testing.T) {
	t := devtest.ParallelT(gt)

	sys := presets.NewSimpleInterop(t)
	client := sys.Supervisor.Escape()

	t.Run("fails with invalid chain ID", func(gt devtest.T) {
		_, err := client.QueryAPI().CrossDerivedToSource(context.Background(), eth.ChainIDFromUInt64(100), eth.BlockID{Number: 25})
		require.Error(t, err, "expected CrossDerivedToSource to fail with invalid chain")
	})

	safe, err := client.QueryAPI().CrossSafe(context.Background(), sys.L2ChainA.ChainID())
	require.NoError(t, err)

	t.Run(fmt.Sprintf("succeeds with valid chain ID %d", sys.L2ChainA.ChainID()), func(gt devtest.T) {
		source, err := client.QueryAPI().CrossDerivedToSource(
			context.Background(),
			sys.L2ChainA.ChainID(),
			eth.BlockID{
				Number: safe.Derived.Number,
				Hash:   safe.Derived.Hash,
			},
		)
		require.NoError(t, err)
		assert.Greater(t, source.Number, uint64(0))
		assert.Len(t, source.Hash, 32)
		assert.Equal(t, source.Number, safe.Source.Number)
		assert.Equal(t, source.Hash, safe.Source.Hash)
	})

}

func TestRPCCheckAccessList(gt *testing.T) {
	t := devtest.ParallelT(gt)

	sys := presets.NewSimpleInterop(t)
	client := sys.Supervisor.Escape()
	ctx := sys.T.Ctx()

	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	eventLoggerAddress := alice.DeployEventLogger()
	sys.L2ChainB.CatchUpTo(sys.L2ChainA)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	_, initReceipt := alice.SendInitMessage(
		interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(3), rng.Intn(10)),
	)

	logToAccess := func(chainID eth.ChainID, log *gethTypes.Log, timestamp uint64) types.Access {
		msgPayload := make([]byte, 0)
		for _, topic := range log.Topics {
			msgPayload = append(msgPayload, topic.Bytes()...)
		}
		msgPayload = append(msgPayload, log.Data...)

		msgHash := crypto.Keccak256Hash(msgPayload)
		args := types.ChecksumArgs{
			BlockNumber: log.BlockNumber,
			Timestamp:   timestamp,
			LogIndex:    uint32(log.Index),
			ChainID:     chainID,
			LogHash:     types.PayloadHashToLogHash(msgHash, log.Address),
		}
		return args.Access()
	}

	blockRef := sys.L2ChainA.PublicRPC().BlockRefByNumber(initReceipt.BlockNumber.Uint64())

	var accessEntries []types.Access
	for _, evLog := range initReceipt.Logs {
		accessEntries = append(accessEntries, logToAccess(alice.ChainID(), evLog, blockRef.Time))
	}

	cloneAccessEntries := func() []types.Access {
		clone := make([]types.Access, len(accessEntries))
		copy(clone, accessEntries)
		return clone
	}

	sys.L2ChainB.WaitForBlock()

	t.Run("succeeds with valid access list", func(gt devtest.T) {
		accessList := types.EncodeAccessList(cloneAccessEntries())
		timestamp := uint64(time.Now().Unix())
		ed := types.ExecutingDescriptor{
			Timestamp: timestamp,
			ChainID:   bob.ChainID(),
		}

		err := client.QueryAPI().CheckAccessList(ctx, accessList, types.LocalUnsafe, ed)
		require.NoError(t, err, "CheckAccessList should succeed with valid access list and chain ID")
	})

	t.Run("fails with invalid chain ID", func(gt devtest.T) {
		accessList := types.EncodeAccessList(cloneAccessEntries())
		timestamp := uint64(time.Now().Unix())
		ed := types.ExecutingDescriptor{
			Timestamp: timestamp,
			ChainID:   eth.ChainIDFromUInt64(99999999),
		}

		err := client.QueryAPI().CheckAccessList(ctx, accessList, types.LocalUnsafe, ed)
		require.Error(t, err, "CheckAccessList should fail with invalid chain ID")
	})

	t.Run("fails with invalid timestamp", func(gt devtest.T) {
		accessList := types.EncodeAccessList(cloneAccessEntries())
		ed := types.ExecutingDescriptor{
			Timestamp: blockRef.Time - 1,
			ChainID:   bob.ChainID(),
		}

		err := client.QueryAPI().CheckAccessList(ctx, accessList, types.LocalUnsafe, ed)
		require.Error(t, err, "CheckAccessList should fail with invalid timestamp")
	})

	t.Run("fails with conflicting data - log index mismatch", func(gt devtest.T) {
		entries := cloneAccessEntries()
		entries[0].LogIndex = 10
		accessList := types.EncodeAccessList(entries)
		timestamp := uint64(time.Now().Unix())
		ed := types.ExecutingDescriptor{
			Timestamp: timestamp,
			ChainID:   bob.ChainID(),
		}

		err := client.QueryAPI().CheckAccessList(ctx, accessList, types.LocalUnsafe, ed)
		require.Error(t, err, "CheckAccessList should fail with conflicting log index")
	})

	t.Run("fails with conflicting data - invalid block number", func(gt devtest.T) {
		entries := cloneAccessEntries()
		entries[0].BlockNumber = entries[0].BlockNumber - 1
		accessList := types.EncodeAccessList(entries)
		timestamp := uint64(time.Now().Unix())
		ed := types.ExecutingDescriptor{
			Timestamp: timestamp,
			ChainID:   bob.ChainID(),
		}

		err := client.QueryAPI().CheckAccessList(ctx, accessList, types.LocalUnsafe, ed)
		require.Error(t, err, "CheckAccessList should fail with invalid block number")
	})

	t.Run("fails with conflicting data - invalid checksum", func(gt devtest.T) {
		entries := cloneAccessEntries()
		// Corrupt the checksum
		entries[0].Checksum[10] ^= 0xFF
		accessList := types.EncodeAccessList(entries)
		timestamp := uint64(time.Now().Unix())
		ed := types.ExecutingDescriptor{
			Timestamp: timestamp,
			ChainID:   bob.ChainID(),
		}

		err := client.QueryAPI().CheckAccessList(ctx, accessList, types.LocalUnsafe, ed)
		require.Error(t, err, "CheckAccessList should fail with invalid checksum")
	})

	t.Run("fails with safety violation", func(gt devtest.T) {
		accessList := types.EncodeAccessList(cloneAccessEntries())
		timestamp := uint64(time.Now().Unix())
		ed := types.ExecutingDescriptor{
			Timestamp: timestamp,
			ChainID:   bob.ChainID(),
		}

		err := client.QueryAPI().CheckAccessList(ctx, accessList, types.Finalized, ed)
		require.Error(t, err, "CheckAccessList should fail due to safety level violation")
	})
}
