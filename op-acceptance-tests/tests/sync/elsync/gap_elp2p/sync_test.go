package gap_elp2p

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
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

	// We do NOT need to resend the same FCU just because peers have connected;
	// the EL continues syncing toward the last forkchoice target asynchronously.
	//
	// Once the EL has downloaded and validated the required data,
	// a subsequent FCU call (even with the same target) may immediately return VALID.
	//
	// In practice, after peers are established, one or two FCU calls
	// typically observe VALID — though this depends on the EL’s sync progress
	// and network conditions.
	attempts := 3

	// Retry a few times until the first EL Sync is complete
	sys.L2ELB.FinishedELSync(sys.L2EL, targetNum, 0, 0)

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

// TestELP2PFCUUnavailableHash verifies that when an Execution Layer (EL) client
// receives a Forkchoice Update (FCU) with an unknown head hash (invalid or
// non-existent) during EL syncing, it remains in the "SYNCING" state and does
// not advance its canonical chain.
//
// In this scenario, the node is EL syncing, and the target forkchoice head hash
// does not exist in any connected EL peers. When the EL processes an FCU with
// such an unknown hash, it attempts to fetch the corresponding block from peers
// once. If the block cannot be retrieved, the EL reports SYNCING. The EL will not
// retry automatically, but each subsequent FCU with the same unknown hash will
// trigger another one-time fetch attempt, again resulting in SYNCING.
//
// This behavior ensures that the EL client safely handles invalid or unknown
// forkchoice targets by consistently reporting SYNCING for each FCU attempt
// and by avoiding advancement of the chain on invalid data.
func TestELP2PFCUUnavailableHash(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	logger := t.Logger()
	genesis := sys.L2ELB.BlockRefByNumber(0)

	// Advance few blocks to make sure reference node advanced
	sys.L2CL.Advanced(types.LocalUnsafe, 10, 30)

	sys.L2CLB.Stop()

	// At this point, L2ELB has no ELP2P, and L2CL connection
	startNum := sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number

	// Peer to confirm EL Syncing is working
	sys.L2ELB.PeerWith(sys.L2EL)

	// Trigger EL Sync to valid hash
	targetNum := startNum + 3
	attempts := 5
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, 0, 0, nil).WaitUntilValid(attempts)
	// head advanced
	sys.L2ELB.UnsafeHead().NumEqualTo(targetNum)
	logger.Info("Canonical chain advanced", "number", targetNum)

	unsafeHashInvalid := common.MaxHash // must be non-existent invalid hash
	// We retry FCU using the invalid hash
	// The ELP2P enabled L2EL will ask other peers but fail to fetch the block with the invalid hash
	// Example logs from L2EL(geth)
	//  "Fetching the unknown forkchoice head from network"  hash=0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
	//  "Fetching batch of headers"
	//  "Could not retrieve unknown head from peers"
	sys.L2ELB.ForkchoiceUpdateRaw(unsafeHashInvalid, genesis.Hash, genesis.Hash, nil).Retry(attempts).ResultAllSyncing()

	sys.L2ELB.UnsafeHead().NumEqualTo(targetNum)
	logger.Info("Canonical chain not advanced", "number", targetNum)

	t.Cleanup(func() {
		sys.L2CLB.Start()
		sys.L2ELB.DisconnectPeerWith(sys.L2EL)
	})
}

// TestSafeDoesNotAdvanceWhenUnsafeIsSyncing_NoELP2P verifies Engine API semantics
// where ForkchoiceUpdate (FCU) validates the unsafe target first and, if the unsafe
// head is not directly appendable (e.g., there is a gap), FCU returns SYNCING and
// exits early without updating the safe head—even when the provided safe hash is
// independently appendable.
//
// The presence or absence of EL P2P is not the core factor here. Disabling EL P2P
// in this test simply makes the gap persist so the condition is observable. The
// key behavior is that FCU's unsafe-first check causes an early return, so the safe
// head is not bumped when the unsafe target cannot be immediately synced.
//
// This validates that safe head updates are contingent on the unsafe target passing
// appendability/sync checks first, per Engine API behavior.
func TestSafeDoesNotAdvanceWhenUnsafeIsSyncing_NoELP2P(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	logger := t.Logger()

	// Advance few blocks to make sure reference node advanced
	sys.L2CL.Advanced(types.LocalUnsafe, 10, 30)

	sys.L2CLB.Stop()

	// At this point, L2ELB has no ELP2P, and L2CL connection
	startNum := sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number

	// Try to advance non canonical chain
	targetNum := startNum + 1
	logger.Info("NewPayload", "target", targetNum)
	sys.L2ELB.NewPayload(sys.L2EL, targetNum).IsValid()

	// FCU to advance unsafe and safe normally, promoting non canonical chain to canonical
	logger.Info("ForkchoiceUpdate", "target", targetNum)
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, targetNum, 0, nil).IsValid()
	sys.L2ELB.UnsafeHead().NumEqualTo(targetNum)
	sys.L2ELB.SafeHead().NumEqualTo(targetNum)
	logger.Info("Canonical chain advanced for unsafe and safe", "number", targetNum)

	// Try to advance non canonical chain
	safeTargetNum := startNum + 2
	logger.Info("NewPayload", "target", safeTargetNum)
	sys.L2ELB.NewPayload(sys.L2EL, safeTargetNum).IsValid()

	// FCU safe normally, but target unsafe which cannot be synced because of the gap
	unsafeTargetNum := safeTargetNum + 5
	attempts := 5
	logger.Info("ForkchoiceUpdate", "safeTarget", safeTargetNum, "unsafeTarget", unsafeTargetNum)
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, unsafeTargetNum, safeTargetNum, 0, nil).Retry(attempts).ResultAllSyncing()
	sys.L2ELB.UnsafeHead().NumEqualTo(targetNum)
	sys.L2ELB.SafeHead().NumEqualTo(targetNum)
	logger.Info("Canonical chain not advanced for unsafe and safe", "number", targetNum)

	// Try to advance non canonical chain
	safeTargetNum = startNum + 3
	logger.Info("NewPayload", "target", safeTargetNum)
	sys.L2ELB.NewPayload(sys.L2EL, safeTargetNum).IsValid()

	// FCU safe normally, but target unsafe which cannot be synced because of the gap
	unsafeTargetNum = safeTargetNum + 6
	logger.Info("ForkchoiceUpdate", "safeTarget", safeTargetNum, "unsafeTarget", unsafeTargetNum)
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, unsafeTargetNum, safeTargetNum, 0, nil).Retry(attempts).ResultAllSyncing()
	sys.L2ELB.UnsafeHead().NumEqualTo(targetNum)
	sys.L2ELB.SafeHead().NumEqualTo(targetNum)
	logger.Info("Canonical chain not advanced for unsafe and safe", "number", targetNum)

	// Enable EL P2P to update both unsafe and safe at once using EL Sync
	sys.L2ELB.PeerWith(sys.L2EL)
	logger.Info("ForkchoiceUpdate", "safeTarget", safeTargetNum, "unsafeTarget", unsafeTargetNum)
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, unsafeTargetNum, safeTargetNum, 0, nil).WaitUntilValid(attempts)
	sys.L2ELB.UnsafeHead().NumEqualTo(unsafeTargetNum)
	sys.L2ELB.SafeHead().NumEqualTo(safeTargetNum)
	logger.Info("Canonical chain advanced for unsafe and safe", "safeTarget", safeTargetNum, "unsafeTarget", unsafeTargetNum)

	t.Cleanup(func() {
		sys.L2CLB.Start()
		sys.L2ELB.DisconnectPeerWith(sys.L2EL)
	})
}

// TestInvalidPayloadThroughCLP2P verifies that invalid L2 payloads propagated via
// CL P2P (simulated with admin_postUnsafePayload) do not advance either the CL or EL.
//
// The test first confirms normal progress on a valid target, then exercises three
// invalid cases and asserts no advancement on both sides (unsafe head remains at
// startNum+1):
//
//  1. CL-detectable invalidity (bad block hash):
//     The payload is mutated (e.g., StateRoot) without updating BlockHash.
//     The CL rejects it immediately (hash mismatch) and does not relay it to the EL.
//
//  2. EL-only invalidity (bad state root):
//     The payload's BlockHash is recomputed so the CL relays it via engine_newPayload,
//     but the EL rejects it during execution due to an invalid state root.
//
//  3. EL-only invalidity via invalid parent:
//     A new payload builds on a previously rejected block (an invalid parent),
//     causing the EL to reject it as referencing an invalid ancestor.
//
// In all scenarios, both CL and EL remain at the same head height, confirming that
// invalid payloads—whether rejected at the CL or EL—do not advance the chain.
func TestInvalidPayloadThroughCLP2P(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	logger := t.Logger()
	require := t.Require()
	ctx := t.Ctx()

	// Advance few blocks to make sure reference node advanced
	sys.L2CL.Advanced(types.LocalUnsafe, 4, 30)

	// At this point, L2ELB has no ELP2P, and L2CL connection
	startNum := sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number

	// We check L2ELB can be advanced using the valid payload first
	attempts := 3
	targetNum := startNum + 1
	sys.L2CLB.SignalTarget(sys.L2EL, targetNum)
	sys.L2ELB.Reached(eth.Unsafe, targetNum, attempts)
	logger.Info("Canonical chain advanced", "number", targetNum)

	// Assume sequencer crafted invalid payload and broadcasted via P2P
	// Simulate the situation using the admin_postUnsafePayload API

	// Scenario 1: Invalid Payload can be checked at the CL side
	targetNum = startNum + 2
	payload := sys.L2EL.PayloadByNumber(targetNum)
	// inject fault to the payload
	payload.ExecutionPayload.StateRoot = eth.Bytes32{}
	logger.Info("Injected fault to payload but not updated hash")
	// Post invalid payload with the fault that can be checked at the CL side
	require.Error(sys.L2CLB.Escape().RollupAPI().PostUnsafePayload(ctx, payload))
	// ex) op-node error msg: "payload has bad block hash"
	// CL will not send the payload but drop immediately due to hash mismatch
	// EL did not advance
	sys.L2ELB.UnsafeHead().NumEqualTo(startNum + 1)
	// CL did not advance
	sys.L2CLB.UnsafeHead().NumEqualTo(startNum + 1)

	// Scenario 2: Invalid Payload can be only checked at the EL side
	// Make sure to update the block hash included at payload to CL relay the payload to the EL
	newHash, ok := payload.CheckBlockHash()
	require.False(ok)
	logger.Info("Injected fault to payload", "newHash", newHash, "prevHash", payload.ExecutionPayload.BlockHash)
	payload.ExecutionPayload.BlockHash = newHash
	_, ok = payload.CheckBlockHash()
	require.True(ok)
	// L2CLB will relay the payload to the L2ELB using the engine_newPayload
	// L2ELB will return INVALID because payload is invalid, due to wrong stateRoot
	// ex) op-geth will call InsertBlockWithoutSetHead() to execute the payload while engine_newPayload
	// Post invalid payload with the fault that can be only checked at the EL side
	sys.L2CLB.PostUnsafePayload(payload)
	// ex) op-geth error msg: "ignoring bad block: invalid merkle root"
	sys.L2CLB.NotAdvanced(types.LocalUnsafe, attempts)
	sys.L2ELB.NotAdvanced(eth.Unsafe)
	// EL did not advance
	sys.L2ELB.UnsafeHead().NumEqualTo(startNum + 1)
	// CL did not advance
	sys.L2CLB.UnsafeHead().NumEqualTo(startNum + 1)

	// Scenario 3: Invalid Payload can be only checked at the EL side, invalid because invalid parent
	// Try to build on top of previously rejected block with block number startNum + 2
	targetNum = startNum + 3
	payload2 := sys.L2EL.PayloadByNumber(targetNum)
	payload2.ExecutionPayload.ParentHash = payload.ExecutionPayload.BlockHash
	newHash, ok = payload2.CheckBlockHash()
	require.False(ok)
	logger.Info("Updated payload parent to invalid payload", "newHash", newHash)
	payload2.ExecutionPayload.BlockHash = newHash
	_, ok = payload.CheckBlockHash()
	require.True(ok)
	// Post invalid payload with the fault that can be only checked at the EL side
	sys.L2CLB.PostUnsafePayload(payload)
	// ex) op-geth error msg: "ignoring bad block: links to previously rejected block"
	sys.L2CLB.NotAdvanced(types.LocalUnsafe, attempts)
	sys.L2ELB.NotAdvanced(eth.Unsafe)
	// EL did not advance
	sys.L2ELB.UnsafeHead().NumEqualTo(startNum + 1)
	// CL did not advance
	sys.L2CLB.UnsafeHead().NumEqualTo(startNum + 1)
}
