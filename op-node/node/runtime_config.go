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

	// RequiredProtocolVersionStorageSlot is the storage slot that the required protocol version is stored at.
	// Computed as: `bytes32(uint256(keccak256("superchainconfig.required")) - 1)`
	RequiredProtocolVersionStorageSlot = common.HexToHash("0xeb75c045af8a14461fe3a1e7ecc4e2c45da8179cf73cdd5fb4341229e6c4b28c")

	// RecommendedProtocolVersionStorageSlot is the storage slot that the recommended protocol version is stored at.
	// Computed as: `bytes32(uint256(keccak256("superchainconfig.recommended")) - 1)`
	RecommendedProtocolVersionStorageSlot = common.HexToHash("0xff71b5ccd5f08cedb6bb6490bfb79ab59028e2767e73551429ec8ac04184336a")
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

type ProtocolVersion [32]byte

// runtimeConfigData is a flat bundle of configurable data, easy and light to copy around.
type runtimeConfigData struct {
	p2pBlockSignerAddr common.Address

	// superchain protocol version signals
	recommended ProtocolVersion
	required    ProtocolVersion
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

func (r *RuntimeConfig) RequiredProtocolVersion() ProtocolVersion {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.required
}

func (r *RuntimeConfig) RecommendedProtocolVersion() ProtocolVersion {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.recommended
}

// Load resets the runtime configuration by fetching the latest config data from L1 at the given L1 block.
// Load is safe to call concurrently, but will lock the runtime configuration modifications only,
// and will thus not block other Load calls with possibly alternative L1 block views.
func (r *RuntimeConfig) Load(ctx context.Context, l1Ref eth.L1BlockRef) error {
	p2pSignerVal, err := r.l1Client.ReadStorageAt(ctx, r.rollupCfg.L1SystemConfigAddress, UnsafeBlockSignerAddressSystemConfigStorageSlot, l1Ref.Hash)
	if err != nil {
		return fmt.Errorf("failed to fetch unsafe block signing address from system config: %w", err)
	}
	// The superchain protocol version data is optional; only applicable to rollup configs that specify a SuperchainConfig address.
	var requiredProtVersion, recommendedProtoVersion ProtocolVersion
	if r.rollupCfg.SuperchainConfigAddress != (common.Address{}) {
		requiredVal, err := r.l1Client.ReadStorageAt(ctx, r.rollupCfg.SuperchainConfigAddress, RequiredProtocolVersionStorageSlot, l1Ref.Hash)
		if err != nil {
			return fmt.Errorf("required-protocol-version value failed to load from the superchain config: %w", err)
		}
		requiredProtVersion = ProtocolVersion(requiredVal)
		recommendedVal, err := r.l1Client.ReadStorageAt(ctx, r.rollupCfg.SuperchainConfigAddress, RecommendedProtocolVersionStorageSlot, l1Ref.Hash)
		if err != nil {
			return fmt.Errorf("recommended-protocol-version value failed to load from the superchain config: %w", err)
		}
		recommendedProtoVersion = ProtocolVersion(recommendedVal)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.l1Ref = l1Ref
	r.p2pBlockSignerAddr = common.BytesToAddress(p2pSignerVal[:])
	r.required = requiredProtVersion
	r.recommended = recommendedProtoVersion
	r.log.Info("loaded new runtime config values!", "p2p_seq_address", r.p2pBlockSignerAddr)
	return nil
}
