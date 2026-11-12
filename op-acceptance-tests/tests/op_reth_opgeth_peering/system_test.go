package op_reth_opgeth_peering

import (
	"fmt"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	// l2BlockTimeSeconds keeps the runs short.
	l2BlockTimeSeconds = 1
	// senderSyncAttempts bounds, in 2s polls, how long the sender may take to catch up with the
	// chain once peered.
	senderSyncAttempts = 60
)

// followerKinds are the two execution clients of the pair; a matrix over them runs the op-reth
// cell first.
var followerKinds = []sysgo.MixedL2ELKind{sysgo.MixedL2ELOpReth, sysgo.MixedL2ELOpGeth}

// otherKind returns the execution client kind that pairs with kind.
func otherKind(kind sysgo.MixedL2ELKind) sysgo.MixedL2ELKind {
	if kind == sysgo.MixedL2ELOpReth {
		return sysgo.MixedL2ELOpGeth
	}
	return sysgo.MixedL2ELOpReth
}

// followerCell is one cell of a matrix over followerKinds: the follower that acts, and the
// subtest name.
type followerCell struct {
	Kind sysgo.MixedL2ELKind
	Name string
}

func (cell followerCell) String() string {
	return cell.Name
}

// followerCells returns a cell per follower kind, named by nameFormat applied to the acting kind
// and the other kind.
func followerCells(nameFormat string) []followerCell {
	cells := make([]followerCell, len(followerKinds))
	for i, kind := range followerKinds {
		cells[i] = followerCell{Kind: kind, Name: fmt.Sprintf(nameFormat, kind, otherKind(kind))}
	}
	return cells
}

type pairNode struct {
	CL   *dsl.L2CLNode
	EL   *dsl.L2ELNode
	Kind sysgo.MixedL2ELKind
}

// pairSystem is an op-reth sequencer with an op-reth follower and an op-geth follower. The sender
// follower has the receiver follower as its only execution-layer peer, through a basic peering it
// dialed, while the receiver keeps the trusted peering with the sequencer that the mixed runtime
// sets up. A transaction submitted to the sender can only reach the sequencer through the
// receiver, so its inclusion proves it crossed the peering under test. The sender's consensus
// client is peered with the sequencer's alone, so the sender follows the chain, and it is on the
// chain by the time the system is returned. The op-reth nodes get pending and basefee sub-pools of
// burstPoolMaxSizeMB and burstPoolMaxCount, and op-geth a pool of burstPoolMaxCount.
type pairSystem struct {
	FunderL2  *dsl.FunderEOA
	Receiver  pairNode
	Sender    pairNode
	Sequencer pairNode
	Wallet    *dsl.HDWallet
}

// follower returns the pair's follower of the given kind.
func (sys *pairSystem) follower(kind sysgo.MixedL2ELKind) pairNode {
	if sys.Sender.Kind == kind {
		return sys.Sender
	}
	return sys.Receiver
}

// withBurstPoolSlots lets op-geth's pool hold burstPoolMaxCount transactions of one slot each,
// the size of every transaction these tests sign.
func withBurstPoolSlots(ethCfg *ethconfig.Config, _ *node.Config) error {
	ethCfg.TxPool.GlobalSlots = burstPoolMaxCount
	return nil
}

func newPairSystem(t devtest.T, sender sysgo.MixedL2ELKind) *pairSystem {
	clKind := sysgo.ResolveMixedL2CLKind()
	followerSpec := func(kind sysgo.MixedL2ELKind) sysgo.MixedSingleChainNodeSpec {
		return sysgo.MixedSingleChainNodeSpec{
			CLKey:  string(kind) + "-follower",
			CLKind: clKind,
			ELKey:  string(kind) + "-follower",
			ELKind: kind,
			// The runtime peers every other node with every other as trusted; the sender is kept
			// off that mesh and peered below, with the receiver as its only execution-layer peer.
			IsolateFromL2P2P: kind == sender,
		}
	}
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		// op-geth refuses Karst blocks, which the default dev genesis activates.
		DeployerOptions: []sysgo.DeployerOption{
			sysgo.WithJovianAtGenesis,
			sysgo.WithUniformL2BlockTimes(l2BlockTimeSeconds),
		},
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				CLKey:       "sequencer",
				CLKind:      clKind,
				ELKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				IsSequencer: true,
			},
			followerSpec(sysgo.MixedL2ELOpReth),
			followerSpec(sysgo.MixedL2ELOpGeth),
		},
		OpGethOptions: []geth.GethOption{withBurstPoolSlots},
		OpRethOptions: []sysgo.OpRethOption{
			sysgo.OpRethWithExtraArgs(
				fmt.Sprintf("--txpool.basefee-max-count=%d", burstPoolMaxCount),
				fmt.Sprintf("--txpool.basefee-max-size=%d", burstPoolMaxSizeMB),
				fmt.Sprintf("--txpool.pending-max-count=%d", burstPoolMaxCount),
				fmt.Sprintf("--txpool.pending-max-size=%d", burstPoolMaxSizeMB),
			),
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 3, "expected the sequencer and the two followers")

	sys := &pairSystem{FunderL2: frontends.FunderL2, Wallet: frontends.Wallet}
	for _, node := range frontends.Nodes {
		pair := pairNode{CL: node.CL, EL: node.EL, Kind: node.Spec.ELKind}
		switch {
		case node.Spec.IsSequencer:
			sys.Sequencer = pair
		case node.Spec.ELKind == sender:
			sys.Sender = pair
		default:
			sys.Receiver = pair
		}
	}

	sys.Sender.CL.ConnectPeer(sys.Sequencer.CL)
	sys.Sender.CL.IsP2PConnected(sys.Sequencer.CL)
	sys.Sender.EL.PeerWithBasic(sys.Receiver.EL)
	t.Require().Len(sys.Sender.EL.Peers(), 1, "the sender must have the receiver as its only execution-layer peer")
	// The sender started behind the chain. It catches up through its consensus-layer peering with
	// the sequencer, or through the receiver, its only execution-layer peer, when its consensus
	// client leaves the gap to execution-layer sync. A test may plan transactions against the
	// sender's state or have it admit them only once it is on the chain.
	sys.Sender.EL.Reached(eth.Unsafe, sys.Sequencer.EL.BlockRefByLabel(eth.Unsafe).Number, senderSyncAttempts)
	return sys
}

func skipUnsupported(t devtest.T) {
	sysgo.SkipOnOpGeth(t, "the pair's sequencer is this tree's op-reth")
}
