package node

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/go-multierror"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

var (
	// UnsafeBlockSignerAddressSystemConfigStorageSlot is the storage slot identifier of the unsafeBlockSigner
	// `address` storage value in the SystemConfig L1 contract.
	UnsafeBlockSignerAddressSystemConfigStorageSlot = common.Hash{31: 103}
)

type RuntimeCfgL1Source interface {
	ReadStorageAt(ctx context.Context, address common.Address, storageSlot common.Hash, blockHash common.Hash) (common.Hash, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
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

// Update the runtime config with any new L1 information from the given L1 block.
// The update may be ignored if it's older than the current state.
// If it's newer with a gap, or if there is a reorg, then it forces a full configuration reload.
func (r *RuntimeConfig) Update(ctx context.Context, l1Ref eth.L1BlockRef) error {
	if r.l1Ref.Hash == l1Ref.Hash {
		r.log.Debug("skipping already processed L1 block", "skipped", l1Ref)
		return nil
	}
	if r.l1Ref.Number < l1Ref.Number {
		r.log.Debug("skipping update with older L1 block", "current", r.l1Ref, "skipped", l1Ref)
		return nil
	}
	if r.l1Ref.Hash == l1Ref.ParentHash {
		// apply update that fits on top
		_, receipts, err := r.l1Client.FetchReceipts(ctx, l1Ref.Hash)
		if err != nil {
			return fmt.Errorf("failed to fetch receipts to update runtime config with latest changes of %s: %w", l1Ref, err)
		}

		runtimeConfigCopy := r.runtimeConfigData
		if err := parseReceipts(r.rollupCfg, receipts, &runtimeConfigCopy); err != nil {
			return fmt.Errorf("failed to parse receipts to update runtime config with latest changes of %s: %w", l1Ref, err)
		}

		// concurrent updates / races are fine, as long as the data is consistent with its reference of origin in L1
		r.mu.Lock()
		r.l1Ref = l1Ref
		r.runtimeConfigData = runtimeConfigCopy
		r.mu.Unlock()

		return nil
	} else {
		// fully reload
		r.log.Warn("skipped or reorged on runtime config, reloading now", "old", r.l1Ref, "new", l1Ref)
		return r.Load(ctx, l1Ref)
	}
}

// parseReceipts iterates receipts of the SystemConfig to find runtime configuration updates
func parseReceipts(cfg *rollup.Config, receipts types.Receipts, dest *runtimeConfigData) error {
	var result error
	for i, rec := range receipts {
		if rec.Status != types.ReceiptStatusSuccessful {
			continue
		}
		for j, log := range rec.Logs {
			if log.Address == cfg.L1SystemConfigAddress && len(log.Topics) > 0 && log.Topics[0] == derive.ConfigUpdateEventABIHash {
				if err := parseEvent(log, dest); err != nil {
					result = multierror.Append(result, fmt.Errorf("malformatted L1 system sysCfg log in receipt %d, log %d: %w", i, j, err))
				}
			}
		}
	}
	return result
}

// parseEvent is handles event logs originating from the L1 system config to update the runtime configuration
func parseEvent(ev *types.Log, dest *runtimeConfigData) error {
	if len(ev.Topics) != 3 {
		return fmt.Errorf("expected 3 event topics (event identity, indexed version, indexed updateType), got %d", len(ev.Topics))
	}
	if ev.Topics[0] != derive.ConfigUpdateEventABIHash {
		return fmt.Errorf("invalid deposit event selector: %s, expected %s", ev.Topics[0], derive.DepositEventABIHash)
	}

	// indexed 0
	version := ev.Topics[1]
	if version != derive.ConfigUpdateEventVersion0 {
		return fmt.Errorf("unrecognized L1 sysCfg update event version: %s", version)
	}
	// indexed 1
	updateType := ev.Topics[2]
	switch updateType {
	case derive.SystemConfigUpdateUnsafeBlockSigner:
		if len(ev.Data) != 32*3 {
			return fmt.Errorf("expected 32*3 bytes in unsafe block signer address update, but got %d bytes", len(ev.Data))
		}
		if x := common.BytesToHash(ev.Data[:32]); x != (common.Hash{31: 32}) {
			return fmt.Errorf("expected offset to point to length location, but got %s", x)
		}
		if x := common.BytesToHash(ev.Data[32:64]); x != (common.Hash{31: 32}) {
			return fmt.Errorf("expected length of 1 bytes32, but got %s", x)
		}
		if !bytes.Equal(ev.Data[64:64+12], make([]byte, 12)) {
			return fmt.Errorf("expected version 0 unsafe block signer address with zero padding, but got %x", ev.Data)
		}
		dest.p2pBlockSignerAddr.SetBytes(ev.Data[64+12:])
		return nil
	default:
		// ignore all other events, e.g. system config updates related to L2 derivation
		return nil
	}
}
