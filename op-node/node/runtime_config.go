package node

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	// UnsafeBlockSignerAddressSystemConfigStorageSlot is the storage slot identifier of the unsafeBlockSigner
	// `address` storage value in the SystemConfig L1 contract. Computed as `keccak256("systemconfig.unsafeblocksigner")`
	UnsafeBlockSignerAddressSystemConfigStorageSlot = common.HexToHash("0x65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08")
)

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

// Load resets the runtime configuration by fetching the latest config data from L1 at the given L1 block.
// Load is safe to call concurrently, but will lock the runtime configuration modifications only,
// and will thus not block other Load calls with possibly alternative L1 block views.
func (r *RuntimeConfig) Load(ctx context.Context, l1Ref eth.L1BlockRef) error {
	val, err := r.l1Client.ReadStorageAt(ctx, r.rollupCfg.L1SystemConfigAddress, UnsafeBlockSignerAddressSystemConfigStorageSlot, l1Ref.Hash)
	if err != nil {
		return fmt.Errorf("failed to fetch unsafe block signing address from system config: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.l1Ref = l1Ref
	r.p2pBlockSignerAddr = common.BytesToAddress(val[:])
	r.log.Info("loaded new runtime config values!", "p2p_seq_address", r.p2pBlockSignerAddr)
	return nil
}
