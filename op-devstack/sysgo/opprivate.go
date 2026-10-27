package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

// OPPrivateSettlementChainID is the settlement chain ID of the Skeleton private chain. There is
// no settlement chain yet: any ID other than 0 and the chain's own ID makes the chain private.
const OPPrivateSettlementChainID = 1337

// opPrivateSequencingWindow is an effectively infinite sequencing window (in L1 blocks): a
// private chain never posts batches. It is not math.MaxUint64 because derivation adds to it and
// op-node multiplies it by the L1 block time.
const opPrivateSequencingWindow = 1 << 32

// OPPrivateNode is one private-chain node: a kona-node CL and an op-reth EL.
type OPPrivateNode struct {
	EL *OpReth
	CL *KonaNode
}

// OPPrivateSkeletonRuntime is L1 and one OP Private private chain, deployed by op-deployer with
// test accounts funded in genesis, and no batcher, proposer or challenger. The sequencer runs in
// follow mode with nothing to follow. The RPC node follows the sequencer.
type OPPrivateSkeletonRuntime struct {
	Keys      devkeys.Keys
	L1Network *L1Network
	L1EL      *L1Geth
	L1CL      *L1CLNode
	L2Network *L2Network

	Sequencer OPPrivateNode
	RPCNode   OPPrivateNode
}

// NewOPPrivateSkeletonRuntime starts the OP Private Skeleton devnet.
func NewOPPrivateSkeletonRuntime(t devtest.T, cfg PresetConfig) *OPPrivateSkeletonRuntime {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err)
	jwtPath, jwtSecret := writeJWTSecret(t)

	opts := append([]DeployerOption{
		WithDevFeatureEnabled(devfeatures.OPPrivateFlag),
		// The preset has a single L2 chain, so a global override sets it on that chain.
		func(_ devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
			builder.WithGlobalOverride("settlementChainId", uint64(OPPrivateSettlementChainID))
		},
		WithSequencingWindow(opPrivateSequencingWindow),
	}, cfg.DeployerOptions...)
	var l1EL *L1Geth
	var l1CL *L1CLNode
	_, l1Net, l2Net, _, _ := buildSingleChainWorld(t, keys, false, cfg.LocalContractArtifactsPath, nil,
		func(l1Net *L1Network) (*L1Geth, *L1CLNode) {
			l1EL, l1CL = startInProcessL1WithClockConfig(t, l1Net, jwtPath, clock.SystemClock, cfg)
			return l1EL, l1CL
		}, opts...)

	seqEL := startMixedOpRethNode(t, l2Net, "sequencer", jwtPath, jwtSecret, nil, cfg.OpRethOptions...)
	seqCL := startMixedKonaNode(t, keys, l1Net, l2Net, l1EL, l1CL, seqEL, "sequencer", "sequencer", true, nil)
	rpcEL := startMixedOpRethNode(t, l2Net, "rpc", jwtPath, jwtSecret, nil, cfg.OpRethOptions...)
	rpcCL := startMixedKonaNode(t, keys, l1Net, l2Net, l1EL, l1CL, rpcEL, "rpc", "rpc", false, nil,
		"KONA_NODE_L2_FOLLOW_SOURCE="+seqCL.UserRPC())

	connectL2ELPeers(t, t.Logger(), seqEL.UserRPC(), rpcEL.UserRPC())
	connectL2CLPeers(t, t.Logger(), seqCL, rpcCL)

	return &OPPrivateSkeletonRuntime{
		Keys:      keys,
		L1Network: l1Net,
		L1EL:      l1EL,
		L1CL:      l1CL,
		L2Network: l2Net,
		Sequencer: OPPrivateNode{EL: seqEL, CL: seqCL},
		RPCNode:   OPPrivateNode{EL: rpcEL, CL: rpcCL},
	}
}
