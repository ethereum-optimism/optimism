package multiblock

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const (
	// One-second blocks: a group has to form inside a second, so the tests never wait long for
	// one, and a group that does form is unmistakably several blocks on one timestamp.
	blockTime = 1
	// The largest block group the chain allows.
	maxMultiBlocks = 16
	// Seconds after L2 genesis at which blocks may start sharing a timestamp. Long enough for the
	// devnet to come up, fund the load accounts and produce loaded blocks well before the feature
	// turns on, so the pre-activation chain is observed under the same load as the post-activation
	// one.
	activationOffset = 60
	// Blocks per span batch. Spans this short against groups of two or more make a span boundary
	// falling inside a group likely, which is the case span batch v2's leading sibling bit exists
	// for. Nothing pins a particular span to a particular group; the evidence is that a group that
	// straddles a boundary and fails to derive stalls the verifier's safe head, which is what
	// TestVerifierDerivesMultiBlocksFromBatcher's timeout catches.
	maxBlocksPerSpanBatch = 3

	// Accounts the load runs from. op-reth guarantees each account only a handful of pending
	// transaction slots, so the load is spread over several of them.
	loadAccounts = 6
	// Transactions submitted per block time. Comfortably above one per sibling the sequencer can
	// seal, so a sibling finds transactions waiting for it rather than an empty pool.
	loadTxsPerBlockTime = 40

	// Building a group is bounded by one block time, so a handful of slots is generous.
	siblingTimeout = 90 * time.Second
	// How long the spam has to reach the chain. Generous: the verdict that matters is whether it
	// landed before the activation, not how quickly it landed.
	loadTimeout = 2 * time.Minute
	// Gossip is direct between the two nodes on one host.
	p2pTimeout = 90 * time.Second
	// Deriving needs the batcher to submit and L1 to confirm.
	safeHeadTimeout = 5 * time.Minute
	// How far past the activation an idle chain is observed, in seconds. Long enough that a
	// sequencer that wrongly groups idle blocks cannot slip through.
	idleObservationWindow = 15
	// How far past a block group the loaded chain is observed, in seconds, before its shape is
	// checked. Long enough to cover several block times, so the check sees timestamp steps and
	// successive groups rather than one group in isolation.
	loadObservationWindow = 10
)

// multiBlockSystem is a kona-node sequencer and a kona-node verifier, both on op-reth, on a chain
// that allows up to maxMultiBlocks blocks per timestamp from activationOffset onward.
type multiBlockSystem struct {
	L2Network   *dsl.L2Network
	L2Batcher   *dsl.L2Batcher
	SequencerEL *dsl.L2ELNode
	VerifierEL  *dsl.L2ELNode
	Wallet      *dsl.HDWallet
	FunderL2    *dsl.FunderEOA
}

// newMultiBlockSystem brings the devnet up. The node kinds are pinned rather than read from
// DEVSTACK_L2CL_KIND / DEVSTACK_L2EL_KIND: multi-blocks is a kona-node plus op-reth feature, so the
// package has to run identically in every CI variant.
//
// The sequencer's op-reth is told that a payload is worth sealing once it has collected a handful
// of transactions, or a moment's worth of them; that is what lets the sequencer seal well before
// its slot deadline and fit several blocks into one block time. The verifier keeps the stock build
// settings — it does not build.
func newMultiBlockSystem(t devtest.T) *multiBlockSystem {
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{
				ELKey:       "sequencer-op-reth",
				CLKey:       "sequencer",
				ELKind:      sysgo.MixedL2ELOpReth,
				CLKind:      sysgo.MixedL2CLKona,
				IsSequencer: true,
				OpRethOpts: []sysgo.OpRethOption{
					sysgo.OpRethWithExtraArgs(
						"--rollup.multi-block.min-txs", "8",
						"--rollup.multi-block.min-build-time", "50",
					),
					sysgo.OpRethWithBuilderInterval(25 * time.Millisecond),
				},
			},
			{
				ELKey:  "verifier-op-reth",
				CLKey:  "verifier",
				ELKind: sysgo.MixedL2ELOpReth,
				CLKind: sysgo.MixedL2CLKona,
			},
		},
		DeployerOptions: []sysgo.DeployerOption{
			sysgo.WithUniformL2BlockTimes(blockTime),
			sysgo.WithMultiBlockAtOffset(activationOffset),
			sysgo.WithMaxMultiBlocks(maxMultiBlocks),
		},
		BatcherOptions: []sysgo.BatcherOption{withSmallSpanBatches},
	})

	frontends := presets.NewMixedSingleChainFrontends(t, runtime)
	t.Require().Len(frontends.Nodes, 2, "multi-blocks system must hold a sequencer and a verifier")

	var verifierEL *dsl.L2ELNode
	for _, node := range frontends.Nodes {
		if !node.Spec.IsSequencer {
			verifierEL = node.EL
		}
	}
	t.Require().NotNil(verifierEL, "missing multi-blocks verifier EL")

	return &multiBlockSystem{
		L2Network:   frontends.L2Network,
		L2Batcher:   frontends.L2Batcher,
		SequencerEL: frontends.L2Network.PrimaryEL(),
		VerifierEL:  verifierEL,
		Wallet:      frontends.Wallet,
		FunderL2:    frontends.FunderL2,
	}
}

// withSmallSpanBatches makes the batcher emit span batches holding only a few blocks each, so a
// block group regularly straddles a span, and with it a channel.
func withSmallSpanBatches(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
	cfg.BatchType = derive.SpanBatchType
	cfg.MaxBlocksPerSpanBatch = maxBlocksPerSpanBatch
}

func (sys *multiBlockSystem) RollupConfig() *rollup.Config {
	return sys.L2Network.Escape().RollupConfig()
}

// StartTransferLoad keeps the sequencer's mempool fed with plain transfers until the test ends.
// The arrival rate matters more than the volume: a block group only grows while every sibling finds
// a transaction worth sealing, so a one-shot burst that the first block drains whole would stop the
// group at two blocks.
func (sys *multiBlockSystem) StartTransferLoad(t devtest.T) {
	l2BlockTime := time.Duration(sys.RollupConfig().BlockTime) * time.Second
	senders := loadtest.NewRoundRobin(loadtest.FundEOAs(
		t, eth.ThousandEther, loadAccounts, l2BlockTime, sys.SequencerEL, sys.Wallet, sys.FunderL2))
	recipient := sys.Wallet.NewEOA(sys.SequencerEL).Address()

	spammer := loadtest.SpammerFunc(func(t devtest.T) error {
		_, err := senders.Get().Include(t,
			txplan.WithTo(&recipient),
			txplan.WithValue(eth.GWei(1)),
			txplan.WithGasLimit(params.TxGas))
		return err
	})

	ctx, cancel := context.WithCancel(t.Ctx())
	var wg sync.WaitGroup
	wg.Add(1)
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	go func() {
		defer wg.Done()
		loadtest.NewConstant(l2BlockTime, loadtest.WithBaseRPS(loadTxsPerBlockTime)).
			Run(t.WithCtx(ctx), spammer)
	}()
	t.Logger().Info("Transfer load started", "accounts", loadAccounts, "txsPerBlockTime", loadTxsPerBlockTime)
}
