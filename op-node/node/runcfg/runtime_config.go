package runcfg

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// DefaultSignerGracePeriod is how long the node continues to accept blocks from
// a previous unsafe block signer after detecting a signer rotation on L1.
// The grace period ends early if a block from the new signer is verified.
// The 3-hour value is picked arbitrarily: long enough to give operators time
// to complete a key rotation across infrastructure, but short enough to stop
// accepting payloads from a retired signer within a reasonable window.
const DefaultSignerGracePeriod = 3 * time.Hour

// UnsafeBlockSignerAddressSystemConfigStorageSlot is the storage slot identifier of the unsafeBlockSigner
// `address` storage value in the SystemConfig L1 contract. Computed as `keccak256("systemconfig.unsafeblocksigner")`
var UnsafeBlockSignerAddressSystemConfigStorageSlot = common.HexToHash("0x65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08")

type RuntimeCfgL1Source interface {
	ReadStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockHash common.Hash) (common.Hash, error)
}

type ReadonlyRuntimeConfig interface {
	P2PSequencerAddress() common.Address
}

// RuntimeConfig maintains runtime-configurable options.
// These options are loaded based on initial loading + updates for every subsequent L1 block.
// Only the *latest* values are maintained however, the runtime config has no concept of chain history,
// does not require any archive data, and may be out of sync with the rollup derivation process.
type RuntimeConfig struct {
	mu sync.RWMutex

	log log.Logger

	l1Client  RuntimeCfgL1Source
	rollupCfg *rollup.Config

	// l1Ref is the current source of the data,
	// if this is invalidated with a reorg the data will have to be reloaded.
	l1Ref eth.L1BlockRef

	runtimeConfigData
}

// runtimeConfigData is a flat bundle of configurable data, easy and light to copy around.
type runtimeConfigData struct {
	p2pBlockSignerAddr common.Address

	// prevP2PBlockSignerAddr holds the previous signer during a grace period after rotation.
	prevP2PBlockSignerAddr common.Address
	// signerChangeTime is when the signer rotation was detected.
	signerChangeTime time.Time
}

var _ p2p.GossipRuntimeConfig = (*RuntimeConfig)(nil)

func NewRuntimeConfig(log log.Logger, l1Client RuntimeCfgL1Source, rollupCfg *rollup.Config) *RuntimeConfig {
	return &RuntimeConfig{
		log:       log,
		l1Client:  l1Client,
		rollupCfg: rollupCfg,
	}
}

func (r *RuntimeConfig) P2PSequencerAddress() common.Address {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.p2pBlockSignerAddr
}

func (r *RuntimeConfig) PreviousP2PSequencerAddress() common.Address {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if time.Since(r.signerChangeTime) > DefaultSignerGracePeriod {
		return common.Address{}
	}
	return r.prevP2PBlockSignerAddr
}

// ConfirmCurrentSigner is called on every validly-signed block to confirm
// the current signer is in use. If a grace period is active (i.e. a previous
// signer is still being accepted), the previous signer is cleared.
func (r *RuntimeConfig) ConfirmCurrentSigner() {
	// Guard: check the previous signer and exit early if it's not set.
	r.mu.RLock()
	hasPrev := r.prevP2PBlockSignerAddr != (common.Address{})
	r.mu.RUnlock()
	if !hasPrev {
		return
	}

	// Otherwise, take the write lock and clear the previous signer.
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.prevP2PBlockSignerAddr == (common.Address{}) {
		return
	}
	if r.p2pBlockSignerAddr == (common.Address{}) {
		r.log.Warn("confirmed current signer but address is nil")
		return
	}
	r.log.Info("new signer confirmed in use, ending grace period",
		"current", r.p2pBlockSignerAddr, "previous", r.prevP2PBlockSignerAddr)
	r.prevP2PBlockSignerAddr = common.Address{}
}

// Load resets the runtime configuration by fetching the latest config data from L1 at the given L1 block.
// Load is safe to call concurrently, but will lock the runtime configuration modifications only,
// and will thus not block other Load calls with possibly alternative L1 block views.
func (r *RuntimeConfig) Load(ctx context.Context, l1Ref eth.L1BlockRef) error {
	p2pSignerVal, err := r.l1Client.ReadStorageAt(ctx, r.rollupCfg.L1SystemConfigAddress, UnsafeBlockSignerAddressSystemConfigStorageSlot, l1Ref.Hash)
	if err != nil {
		return fmt.Errorf("failed to fetch unsafe block signing address from system config: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.l1Ref = l1Ref
	r.rotateSigner(common.BytesToAddress(p2pSignerVal[:]))
	r.log.Info("loaded new runtime config values!", "p2p_seq_address", r.p2pBlockSignerAddr)
	return nil
}

// rotateSigner updates the current signer address and, if it changed,
// starts a grace period during which the previous signer is still accepted.
// If the signer changes again before the grace period expires, only the most
// recent previous signer is retained.
// Must be called with r.mu held.
func (r *RuntimeConfig) rotateSigner(newAddr common.Address) {
	if r.p2pBlockSignerAddr != (common.Address{}) && r.p2pBlockSignerAddr != newAddr {
		r.prevP2PBlockSignerAddr = r.p2pBlockSignerAddr
		r.signerChangeTime = time.Now()
		r.log.Info("p2p signer rotated, grace period started for previous signer",
			"previous", r.p2pBlockSignerAddr, "new", newAddr, "grace_period", DefaultSignerGracePeriod)
	}
	r.p2pBlockSignerAddr = newAddr
}
