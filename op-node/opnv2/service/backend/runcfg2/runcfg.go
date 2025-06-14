// Package runcfg2 replaces the v1 runcfg,
// offering runtime config for multiple chains,
// and no longer watching the previously deprecated protocol-version contract.
package runcfg2

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/node/runcfg"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type RunCfgUpdateRequestEvent struct {
	L1Block eth.BlockID
}

func (ev RunCfgUpdateRequestEvent) String() string {
	return "run-cfg-update-request"
}

type SequencerAuthEntry struct {
	LastL1Update eth.BlockID
	Address      common.Address
}

type RuntimeConfig struct {
	log      log.Logger
	depSet   depset.DependencySet
	cfgs     depset.RollupConfigSetV2
	l1Client runcfg.RuntimeCfgL1Source

	sequencerAuth locks.RWMap[eth.ChainID, SequencerAuthEntry]

	emitter event.Emitter
}

func (r *RuntimeConfig) AttachEmitter(em event.Emitter) {
	r.emitter = em
}

var _ event.AttachEmitter = (*RuntimeConfig)(nil)
var _ event.Deriver = (*RuntimeConfig)(nil)

var _ p2p.GossipChainConfig = (*RuntimeConfig)(nil)

func NewRuntimeConfig(
	logger log.Logger,
	depSet depset.DependencySet,
	cfgs depset.RollupConfigSetV2,
	l1Client runcfg.RuntimeCfgL1Source,
) *RuntimeConfig {
	out := &RuntimeConfig{
		log:      logger,
		depSet:   depSet,
		cfgs:     cfgs,
		l1Client: l1Client,
	}
	for _, chainID := range depSet.Chains() {
		out.sequencerAuth.Set(chainID, SequencerAuthEntry{})
	}
	return out
}

func (r *RuntimeConfig) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case RunCfgUpdateRequestEvent:
		r.Load(ctx, x.L1Block)
		return true
	}
	return false
}

func (r *RuntimeConfig) Load(ctx context.Context, l1Block eth.BlockID) {
	for _, chainID := range r.ChainIDs() {
		rollupCfg := r.cfgs.RollupConfig(chainID)

		// TODO: if we replace this with receipt-parsing,
		// then we can read cached data, since we already do derivation.

		p2pSignerVal, err := r.l1Client.ReadStorageAt(ctx, rollupCfg.L1SystemConfigAddress,
			runcfg.UnsafeBlockSignerAddressSystemConfigStorageSlot, l1Block.Hash)
		if err != nil {
			r.log.Warn("Failed to load params from L1", "chainID", chainID, "err", err)
			continue
		}
		r.sequencerAuth.Set(chainID, SequencerAuthEntry{
			LastL1Update: l1Block,
			Address:      common.BytesToAddress(p2pSignerVal[:]),
		})
	}
}

func (r *RuntimeConfig) ChainIDs() []eth.ChainID {
	return r.depSet.Chains()
}

func (r *RuntimeConfig) P2PSequencerAddress(chainID eth.ChainID) (common.Address, error) {
	entry, ok := r.sequencerAuth.Get(chainID)
	if !ok {
		return common.Address{}, types.ErrUnknownChain
	}
	if entry.Address == (common.Address{}) {
		return common.Address{}, errors.New("not loaded")
	}
	return entry.Address, nil
}

func (r *RuntimeConfig) BlockVersion(chainID eth.ChainID, timestamp uint64) (eth.BlockVersion, error) {
	if !r.depSet.HasChain(chainID) {
		return 0, types.ErrUnknownChain
	}
	rollupCfg := r.cfgs.RollupConfig(chainID)
	if rollupCfg.IsIsthmus(timestamp) {
		return eth.BlockV4, nil
	} else if rollupCfg.IsEcotone(timestamp) {
		return eth.BlockV3, nil
	} else if rollupCfg.IsCanyon(timestamp) {
		return eth.BlockV2, nil
	} else {
		return eth.BlockV1, nil
	}
}
