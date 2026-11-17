package sysgo

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2CLNode interface {
	stack.Lifecycle
	UserRPC() string
}

type L2CLConfig struct {
	// SyncMode for op-node, if this is a sequencer. Kona accepts the recognized
	// modes as compatibility inputs and uses its native adaptive EL sync behavior.
	SequencerSyncMode nodeSync.Mode
	// SyncMode for op-node, if this is a verifier. As above, this does not select
	// a native sync mode in Kona.
	VerifierSyncMode nodeSync.Mode
	// VerifierL1Confs delays L1 input to derivation, independently of sequencing.
	VerifierL1Confs uint64

	// SafeDBPath is the path to the safe DB to use. Disabled if empty.
	SafeDBPath string

	IsSequencer bool

	// ConductorRPC configures this node to use the conductor at the given RPC
	// endpoint. Empty disables conductor integration.
	ConductorRPC        string
	ConductorRPCTimeout time.Duration

	// NoDiscovery is the flag to enable/disable discovery
	NoDiscovery bool

	FollowSource string

	// OffsetELSafe retracts safe and finalized from the EL-sync tip by floor(OffsetELSafe / L2BlockTime) blocks.
	OffsetELSafe time.Duration

	// AltDAServerURL is the DA server the node reads alt-DA inputs from. Empty disables alt-DA.
	AltDAServerURL string
}

func L2CLSequencer() L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.IsSequencer = true
	})
}

func L2CLFollowSource(source string) L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.FollowSource = source
	})
}

func DefaultL2CLConfig() *L2CLConfig {
	return &L2CLConfig{
		SequencerSyncMode: nodeSync.CLSync,
		VerifierSyncMode:  nodeSync.CLSync,
		SafeDBPath:        "",
		IsSequencer:       false,
		NoDiscovery:       false,
		FollowSource:      "",
	}
}

type L2CLOption interface {
	Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig)
}

type L2CLOptionFn func(p devtest.T, target ComponentTarget, cfg *L2CLConfig)

var _ L2CLOption = L2CLOptionFn(nil)

func (fn L2CLOptionFn) Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig) {
	fn(p, target, cfg)
}

// L2CLOptionBundle a list of multiple L2CLOption, to all be applied in order.
type L2CLOptionBundle []L2CLOption

var _ L2CLOption = L2CLOptionBundle(nil)

func (l L2CLOptionBundle) Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig) {
	for _, opt := range l {
		p.Require().NotNil(opt, "cannot Apply nil L2CLOption")
		opt.Apply(p, target, cfg)
	}
}

// startSequencerAtUnsafeHead starts the sequencer of the rollup node behind
// rollupClient at the node's current unsafe head, and returns the sync status
// it started from. The RPC can be up before the node's initial engine reset
// publishes a head, and derivation or gossip can move the head between the read
// and the start call, so it reads a fresh head on every attempt. A start call
// can time out while the node still applies it, so an attempt that fails while
// the sequencer is active counts as started.
func startSequencerAtUnsafeHead(ctx context.Context, logger log.Logger, rollupClient *sources.RollupClient) (*eth.SyncStatus, error) {
	return retry.Do(ctx, 90, retry.Fixed(500*time.Millisecond), func() (*eth.SyncStatus, error) {
		status, err := rollupClient.SyncStatus(ctx)
		if err != nil {
			return nil, err
		}
		if status == nil || status.UnsafeL2.Hash == (common.Hash{}) {
			logger.Info("Waiting for the sequencer to publish its unsafe head")
			return nil, errors.New("sequencer has not published its unsafe head yet")
		}
		if err := rollupClient.StartSequencer(ctx, status.UnsafeL2.Hash); err != nil {
			if active, activeErr := rollupClient.SequencerActive(ctx); activeErr == nil && active {
				return status, nil
			}
			logger.Info("Waiting to start the sequencer at its current head", "err", err)
			return nil, err
		}
		return status, nil
	})
}
