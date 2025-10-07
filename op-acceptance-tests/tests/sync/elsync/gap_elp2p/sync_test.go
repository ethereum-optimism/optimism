package gap_elp2p

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
)

// TestL2ELP2PCanonicalChainAdvancedByFCU verifies the interaction between NewPayload,
// ForkchoiceUpdate (FCU), and ELP2P/EL sync in a multi-node L2 test network.
//
// Scenario
//   - Start a single-chain, multi-node system without ELP2P connectivity for L2ELB.
//   - Advance the reference node (L2EL) so it is ahead of L2ELB.
//
// Expectations covered by this test
//
//	NewPayload without parents present:
//	  - Does NOT trigger EL sync.
//	  - Returns SYNCING for future blocks (startNum+3/5/4/6).
//
//	NewPayload on a non canonical chain with available state:
//	  - Can extend a non canonical chain (startNum+1 then +2) and returns VALID.
//	  - These non canonical chain blocks are retrievable by hash but remain non-canonical
//	    (BlockRefByNumber returns NotFound) until FCU marks them valid.
//
//	FCU promoting non canonical chain to canonical:
//	  - FCU to startNum+2 marks the previously imported non canonical blocks valid
//	    and advances L2ELB canonical head to startNum+2.
//
//	FCU targeting a block that cannot yet be validated (missing ancestors):
//	  - Triggers EL sync on L2EL (skeleton/backfill logs), returns SYNCING,
//	    and does not advance the head while ELP2P is still unavailable.
//
//	Enabling ELP2P and eventual validation:
//	  - After peering L2ELB with L2EL, FCU to startNum+4 eventually becomes VALID
//	    once EL sync completes; the test waits for canonicalization and confirms head advances.
//	  - Subsequent gaps (to startNum+6, then +8) are resolved by FCU with
//	    WaitUntilValid, advancing the canonical head each time.
//
//	NewPayload still does not initiate EL sync:
//	  - A NewPayload to startNum+10 returns SYNCING and the block remains unknown by number
//	    until an FCU is issued, which initially returns SYNCING.
//
// Insights
//   - NewPayload alone never initiates EL sync, but can build a non canonical chain if state exists.
//   - FCU is the mechanism that (a) promotes non canonical chain blocks to canonical when they are
//     already fully validated, and (b) triggers EL sync when ancestors are missing.
//   - Previously submitted NewPayloads that returned SYNCING are not retained to automatically
//     assemble a non canonical chain later.
//   - With ELP2P enabled, repeated FCU attempts eventually validate and advance the canonical chain.
func TestL2ELP2PCanonicalChainAdvancedByFCU(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()

	// Advance few blocks to make sure reference node advanced
	sys.L2CL.Advanced(types.LocalUnsafe, 10, 30)

	sys.L2CLB.Stop()

	// At this point, L2ELB has no ELP2P, and L2CL connection
	startNum := sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number

	// NewPayload does not trigger the EL Sync
	// Example logs from L2EL(geth)
	//  New skeleton head announced
	//  Ignoring payload with missing parent
	targetNum := startNum + 3
	sys.L2ELB.NewPayload(sys.L2EL, targetNum).IsSyncing()

	// NewPayload does not trigger the EL Sync
	// Example logs from L2EL(geth)
	//  New skeleton head announced
	//  Ignoring payload with missing parent
	targetNum = startNum + 5
	sys.L2ELB.NewPayload(sys.L2EL, targetNum).IsSyncing()

	// NewPayload does not trigger the EL Sync
	// Example logs from L2EL(geth)
	//  New skeleton head announced
	//  Ignoring payload with missing parent
	targetNum = startNum + 4
	sys.L2ELB.NewPayload(sys.L2EL, targetNum).IsSyncing()

	// NewPayload can extend non canonical chain because L2EL has state for startNum and can validate payload
	// Example logs from L2EL(geth)
	//  Inserting block without sethead
	//  Persisted trie from memory database
	//  Imported new potential chain segment
	targetNum = startNum + 1
	sys.L2ELB.NewPayload(sys.L2EL, targetNum).IsValid()
	logger.Info("Non canonical chain advanced", "number", targetNum)

	// NewPayload can extend non canonical chain because L2EL has state for startNum+1 and can validate payload
	// Example logs from L2EL(geth)
	//  Inserting block without sethead
	//  Persisted trie from memory database
	//  Imported new potential chain segment
	targetNum = startNum + 2
	sys.L2ELB.NewPayload(sys.L2EL, targetNum).IsValid()
	logger.Info("Non canonical chain advanced", "number", targetNum)

	// Non canonical chain can be fetched via blockhash
	blockRef := sys.L2EL.BlockRefByNumber(targetNum)
	nonCan := sys.L2ELB.BlockRefByHash(blockRef.Hash)
	require.Equal(uint64(targetNum), nonCan.Number)
	require.Equal(blockRef.Hash, nonCan.Hash)
	// Still targetNum block is non canonicalized
	_, err := sys.L2ELB.Escape().L2EthClient().BlockRefByNumber(t.Ctx(), targetNum)
	require.ErrorIs(err, ethereum.NotFound)

	// Previously inserted payloads are not used to make non-canonical chain automatically
	blockRef = sys.L2EL.BlockRefByNumber(startNum + 3)
	_, err = sys.L2ELB.Escape().EthClient().BlockRefByHash(t.Ctx(), blockRef.Hash)
	require.ErrorIs(err, ethereum.NotFound)
	blockRef = sys.L2EL.BlockRefByNumber(startNum + 5)
	_, err = sys.L2ELB.Escape().EthClient().BlockRefByHash(t.Ctx(), blockRef.Hash)
	require.ErrorIs(err, ethereum.NotFound)

	// No FCU yet so head not advanced yet
	require.Equal(startNum, sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// NewPayload does not trigger the EL Sync
	// Example logs from L2EL(geth)
	//  New skeleton head announced
	//  Ignoring payload with missing parent
	targetNum = startNum + 6
	sys.L2ELB.NewPayload(sys.L2EL, targetNum).IsSyncing()

	// FCU marks startNum + 2 as valid, promoting non canonical blocks to canonical blocks
	// Example logs from L2EL(geth)
	//  Extend chain
	//  Chain head was updated
	targetNum = startNum + 2
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).IsValid()
	logger.Info("Canonical chain advanced", "number", targetNum)

	// Head advanced, canonical head bumped
	require.Equal(uint64(targetNum), sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// FCU to target block which cannot be validated, triggers EL Sync but ELP2P not yet available
	// Example logs from L2EL(geth)
	//  New skeleton head announced
	//  created initial skeleton subchain
	//  Starting reverse header sync cycle
	//  Block synchronisation started
	//  Backfilling with the network
	targetNum = startNum + 3
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).IsSyncing()

	// head not advanced
	require.Equal(uint64(startNum+2), sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// FCU to target block which cannot be validated
	// Example logs from L2EL(geth)
	//  New skeleton head announced
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).IsSyncing()

	// head not advanced
	require.Equal(uint64(startNum+2), sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// FCU to target block which cannot be validated
	// Example logs from L2EL(geth)
	//  New skeleton head announced
	targetNum = startNum + 4
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).IsSyncing()

	// head not advanced
	require.Equal(uint64(startNum+2), sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// Finally peer for enabling ELP2P
	sys.L2ELB.PeerWith(sys.L2EL)

	// We allow three attempts. Most of the time, two attempts are enough
	// At first attempt, L2EL starts EL Sync, returing SYNCING.
	// Before second attempt, L2EL finishes EL Sync, and updates targetNum as canonical
	// At second attempt, L2EL returns VALID since targetNum is already canonical
	attempts := 3

	// FCU to target block which can be eventually validated, because ELP2P enabled
	// Example logs from L2EL(geth)
	//  New skeleton head announced
	//  Backfilling with the network
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).IsSyncing()

	// Wait until L2EL finishes EL Sync and canonicalizes until targetNum
	sys.L2ELB.Reached(eth.Unsafe, targetNum, 3)

	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).WaitUntilValid(attempts)
	logger.Info("Canonical chain advanced", "number", targetNum)

	// head advanced
	require.Equal(uint64(targetNum), sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// FCU to target block which can be eventually validated, because ELP2P enabled
	// Example logs from L2EL(geth)
	//  "Restarting sync cycle" reason="chain gapped, head: 4, newHead: 6"
	targetNum = startNum + 6
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).WaitUntilValid(attempts)
	logger.Info("Canonical chain advanced", "number", targetNum)

	// head advanced
	require.Equal(uint64(targetNum), sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// FCU to target block which can be eventually validated, because ELP2P enabled
	// Example logs from L2EL(geth)
	//  "Restarting sync cycle" reason="chain gapped, head: 6, newHead: 8"
	targetNum = startNum + 8
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).WaitUntilValid(attempts)
	logger.Info("Canonical chain advanced", "number", targetNum)

	// head advanced
	require.Equal(uint64(targetNum), sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number)

	// NewPayload does not trigger EL Sync
	targetNum = startNum + 10
	sys.L2ELB.NewPayload(sys.L2EL, targetNum).IsSyncing()
	_, err = sys.L2ELB.Escape().L2EthClient().BlockRefByNumber(t.Ctx(), targetNum)
	require.ErrorIs(err, ethereum.NotFound)
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).IsSyncing()

	t.Cleanup(func() {
		sys.L2CLB.Start()
		sys.L2ELB.DisconnectPeerWith(sys.L2EL)
	})
}
