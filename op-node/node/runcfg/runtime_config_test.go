package runcfg

/*
Behaviors tested in this file:

- When Load detects a signer change (non-zero old → different new), the previous signer
  address is preserved and accessible via PreviousP2PSequencerAddress.
- PreviousP2PSequencerAddress returns zero after the grace period expires.
- ConfirmCurrentSigner clears the previous signer immediately.
- When Load sets the signer for the first time (zero → new), no grace period is started.
- When Load reloads the same signer, the grace period state is not disturbed.
*/

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/stretchr/testify/require"
)

type stubL1Source struct {
	storageValues map[common.Hash]common.Hash
}

func (s *stubL1Source) ReadStorageAt(_ context.Context, _ common.Address, slot common.Hash, _ common.Hash) (common.Hash, error) {
	return s.storageValues[slot], nil
}

func addrToHash(addr common.Address) common.Hash {
	var h common.Hash
	copy(h[12:], addr[:])
	return h
}

func newTestRuntimeConfig(t *testing.T, source *stubL1Source) *RuntimeConfig {
	return NewRuntimeConfig(testlog.Logger(t, log.LevelDebug), source, &rollup.Config{})
}

func TestSignerGracePeriod_ChangeStartsGracePeriod(t *testing.T) {
	addrA := common.HexToAddress("0xAAAA")
	addrB := common.HexToAddress("0xBBBB")
	source := &stubL1Source{storageValues: map[common.Hash]common.Hash{
		UnsafeBlockSignerAddressSystemConfigStorageSlot: addrToHash(addrA),
	}}
	rc := newTestRuntimeConfig(t, source)

	// Initial load: addrA
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{1}, Number: 1}))
	require.Equal(t, addrA, rc.P2PSequencerAddress())
	require.Equal(t, common.Address{}, rc.PreviousP2PSequencerAddress(), "no previous on first load")

	// Rotate signer to addrB
	source.storageValues[UnsafeBlockSignerAddressSystemConfigStorageSlot] = addrToHash(addrB)
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{2}, Number: 2}))
	require.Equal(t, addrB, rc.P2PSequencerAddress())
	require.Equal(t, addrA, rc.PreviousP2PSequencerAddress(), "previous signer should be addrA during grace period")
}

func TestSignerGracePeriod_FirstLoadNoGracePeriod(t *testing.T) {
	addrA := common.HexToAddress("0xAAAA")
	source := &stubL1Source{storageValues: map[common.Hash]common.Hash{
		UnsafeBlockSignerAddressSystemConfigStorageSlot: addrToHash(addrA),
	}}
	rc := newTestRuntimeConfig(t, source)

	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{1}, Number: 1}))
	require.Equal(t, addrA, rc.P2PSequencerAddress())
	require.Equal(t, common.Address{}, rc.PreviousP2PSequencerAddress(),
		"going from zero to first address should not start a grace period")
}

func TestSignerGracePeriod_ConfirmClearsPrevious(t *testing.T) {
	addrA := common.HexToAddress("0xAAAA")
	addrB := common.HexToAddress("0xBBBB")
	source := &stubL1Source{storageValues: map[common.Hash]common.Hash{
		UnsafeBlockSignerAddressSystemConfigStorageSlot: addrToHash(addrA),
	}}
	rc := newTestRuntimeConfig(t, source)

	// Load A, then rotate to B
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{1}, Number: 1}))
	source.storageValues[UnsafeBlockSignerAddressSystemConfigStorageSlot] = addrToHash(addrB)
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{2}, Number: 2}))

	require.Equal(t, addrA, rc.PreviousP2PSequencerAddress())
	rc.ConfirmCurrentSigner()
	require.Equal(t, common.Address{}, rc.PreviousP2PSequencerAddress(),
		"after confirmation, previous signer should be cleared")
}

func TestSignerGracePeriod_SameSignerNoChange(t *testing.T) {
	addrA := common.HexToAddress("0xAAAA")
	addrB := common.HexToAddress("0xBBBB")
	source := &stubL1Source{storageValues: map[common.Hash]common.Hash{
		UnsafeBlockSignerAddressSystemConfigStorageSlot: addrToHash(addrA),
	}}
	rc := newTestRuntimeConfig(t, source)

	// Load A, rotate to B
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{1}, Number: 1}))
	source.storageValues[UnsafeBlockSignerAddressSystemConfigStorageSlot] = addrToHash(addrB)
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{2}, Number: 2}))

	require.Equal(t, addrA, rc.PreviousP2PSequencerAddress())

	// Reload with same signer B — should not disturb grace period
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{3}, Number: 3}))
	require.Equal(t, addrB, rc.P2PSequencerAddress())
	require.Equal(t, addrA, rc.PreviousP2PSequencerAddress(),
		"reloading same signer should not clear the grace period")
}

// TestSignerGracePeriod_Expiry covers the scenario where a signer rotation is
// detected but the new signer never produces a confirmed block (i.e.
// ConfirmCurrentSigner is never called). After the grace period elapses,
// PreviousP2PSequencerAddress must report zero so that gossip rejects blocks
// from the (now retired) previous signer. The current signer address must be
// unaffected by expiry.
func TestSignerGracePeriod_Expiry(t *testing.T) {
	addrA := common.HexToAddress("0xAAAA")
	addrB := common.HexToAddress("0xBBBB")
	source := &stubL1Source{storageValues: map[common.Hash]common.Hash{
		UnsafeBlockSignerAddressSystemConfigStorageSlot: addrToHash(addrA),
	}}
	rc := newTestRuntimeConfig(t, source)

	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{1}, Number: 1}))
	source.storageValues[UnsafeBlockSignerAddressSystemConfigStorageSlot] = addrToHash(addrB)
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{2}, Number: 2}))

	require.Equal(t, addrB, rc.P2PSequencerAddress(),
		"current signer should be the new signer during grace period")
	require.Equal(t, addrA, rc.PreviousP2PSequencerAddress(),
		"previous signer should be the old signer during grace period")

	// Simulate grace period expiry without any ConfirmCurrentSigner call,
	// modelling the case where the new signer never produces a verified block.
	rc.mu.Lock()
	rc.signerChangeTime = time.Now().Add(-DefaultSignerGracePeriod - time.Second)
	rc.mu.Unlock()

	require.Equal(t, addrB, rc.P2PSequencerAddress(),
		"current signer should remain the new signer after grace period expires")
	require.Equal(t, common.Address{}, rc.PreviousP2PSequencerAddress(),
		"previous signer should be zero after grace period expires (so gossip rejects blocks from it)")
}

func TestSignerGracePeriod_DoubleRotation(t *testing.T) {
	addrA := common.HexToAddress("0xAAAA")
	addrB := common.HexToAddress("0xBBBB")
	addrC := common.HexToAddress("0xCCCC")
	source := &stubL1Source{storageValues: map[common.Hash]common.Hash{
		UnsafeBlockSignerAddressSystemConfigStorageSlot: addrToHash(addrA),
	}}
	rc := newTestRuntimeConfig(t, source)

	// A → B
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{1}, Number: 1}))
	source.storageValues[UnsafeBlockSignerAddressSystemConfigStorageSlot] = addrToHash(addrB)
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{2}, Number: 2}))
	require.Equal(t, addrA, rc.PreviousP2PSequencerAddress())

	// B → C (before grace period for A expires)
	source.storageValues[UnsafeBlockSignerAddressSystemConfigStorageSlot] = addrToHash(addrC)
	require.NoError(t, rc.Load(context.Background(), eth.L1BlockRef{Hash: common.Hash{3}, Number: 3}))
	require.Equal(t, addrC, rc.P2PSequencerAddress())
	require.Equal(t, addrB, rc.PreviousP2PSequencerAddress(),
		"double rotation: previous should be B (the most recent old signer), not A")
}
