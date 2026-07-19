package blocks_stream

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	sequencerELKey = "sequencer-op-reth"
	validatorELKey = "validator-op-reth"
)

type blocksStreamControl interface {
	PauseBlocksStream()
	ResumeBlocksStream()
}

type blocksStreamSystem struct {
	t             devtest.T
	SourceEL      *dsl.L2ELNode
	SourceCL      *dsl.L2CLNode
	ValidatorEL   *dsl.L2ELNode
	ValidatorCL   *dsl.L2CLNode
	L1CL          *dsl.L1CLNode
	Batcher       *dsl.L2Batcher
	Funder        *dsl.Funder
	Wallet        *dsl.HDWallet
	TestSequencer *dsl.TestSequencer
	blocksStream  blocksStreamControl
}

func newBlocksStreamSystem(t devtest.T) *blocksStreamSystem {
	return newBlocksStreamSystemWithTestSequencer(t, false)
}

func newBlocksStreamReorgSystem(t devtest.T) *blocksStreamSystem {
	return newBlocksStreamSystemWithTestSequencer(t, true)
}

func newBlocksStreamSystemWithTestSequencer(t devtest.T, withTestSequencer bool) *blocksStreamSystem {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		WithTestSequencer: withTestSequencer,
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       sequencerELKey,
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLOpNode,
				IsSequencer: true,
			},
			{
				ELKey:             validatorELKey,
				CLKey:             "validator",
				ELKind:            sysgo.MixedL2ELOpReth,
				CLKind:            sysgo.MixedL2CLKona,
				BlocksSourceELKey: sequencerELKey,
				IsolateFromL2P2P:  true,
			},
		},
	})
	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "blocks stream system requires a sequencer and validator")

	var sourceEL, validatorEL *dsl.L2ELNode
	var sourceCL, validatorCL *dsl.L2CLNode
	for _, node := range frontends.Nodes {
		switch node.Spec.ELKey {
		case sequencerELKey:
			sourceEL = node.EL
			sourceCL = node.CL
		case validatorELKey:
			validatorEL = node.EL
			validatorCL = node.CL
		}
	}
	t.Require().NotNil(sourceEL, "missing blocks stream source EL")
	t.Require().NotNil(sourceCL, "missing blocks stream source CL")
	t.Require().NotNil(validatorEL, "missing blocks stream validator EL")
	t.Require().NotNil(validatorCL, "missing blocks stream validator CL")

	var streamControl blocksStreamControl
	for _, node := range runtime.Nodes {
		if node.Spec.ELKey != sequencerELKey {
			continue
		}
		var ok bool
		streamControl, ok = node.EL.(blocksStreamControl)
		t.Require().True(ok, "blocks stream source EL is not controllable")
		break
	}
	t.Require().NotNil(streamControl, "missing blocks stream control")
	if withTestSequencer {
		t.Require().NotNil(frontends.TestSequencer, "missing test sequencer")
	}

	wallet := dsl.NewHDWallet(t, devkeys.TestMnemonic, 30)
	return &blocksStreamSystem{
		t:             t,
		SourceEL:      sourceEL,
		SourceCL:      sourceCL,
		ValidatorEL:   validatorEL,
		ValidatorCL:   validatorCL,
		L1CL:          frontends.L1CL,
		Batcher:       frontends.L2Batcher,
		Funder:        dsl.NewFunder(wallet, frontends.FaucetL2, sourceEL),
		Wallet:        wallet,
		TestSequencer: frontends.TestSequencer,
		blocksStream:  streamControl,
	}
}

func (s *blocksStreamSystem) PauseBlocksFeed() {
	s.t.Logger().Info("Pausing sequencer blocks feed")
	s.blocksStream.PauseBlocksStream()
}

func (s *blocksStreamSystem) ResumeBlocksFeed() {
	s.t.Logger().Info("Resuming sequencer blocks feed")
	s.blocksStream.ResumeBlocksStream()
}

func (s *blocksStreamSystem) ReplaceUnsafeBlock(original eth.L2BlockRef) eth.L2BlockRef {
	s.t.Require().NotNil(s.TestSequencer, "test sequencer is required to replace an unsafe block")
	s.SourceCL.StopSequencer()
	s.TestSequencer.SequenceBlock(s.t, s.SourceEL.ChainID(), original.ParentHash)
	s.SourceEL.ReorgExact(original, 30)

	replacement := s.SourceEL.BlockRefByNumber(original.Number)
	s.t.Require().Equal(original.ParentHash, replacement.ParentHash, "replacement must fork from the original block's parent")
	s.t.Require().NotEqual(original.Hash, replacement.Hash, "test sequencer must replace the original unsafe block")

	// Extend the replacement branch so clients exercise replay of a suffix rather than only a
	// same-height tip replacement. It also gives asynchronous proofs-history indexing an
	// unambiguous new high-water mark that tests can await before shutting down op-reth.
	s.TestSequencer.SequenceBlock(s.t, s.SourceEL.ChainID(), replacement.Hash)
	s.SourceEL.Reached(eth.Unsafe, replacement.Number+1, 30)
	s.SourceEL.WaitForProofsStoreBlock(replacement.Number + 1)
	return replacement
}
