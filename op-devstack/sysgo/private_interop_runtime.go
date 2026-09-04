package sysgo

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-interop-filter/filter"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	projectiongenesis "github.com/ethereum-optimism/optimism/op-private-interop/genesis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/claimfollow"
)

// The private interop pair's runtime: what runs, in what order, and what is deliberately not wired.
//
// # The topology
//
// EVERY PROCESS BELOW IS A STOCK BINARY. There is no sidecar left. The private chain is a stock
// op-reth and a stock op-node in light-sequencer mode; the rendering is a stock op-reth and the
// supernode's own verifier route; the publisher is op-batcher with its --private-interop flag group;
// and the follow endpoint the private sequencer reads is served by the SUPERNODE, from public data
// it already has, at the sibling route `<base>/<chainID>/claimed`.
//
// That last one is the change this runtime exists to reflect. The standalone claim-follower is
// deleted. Three ratified amendments made it possible (DESIGN.md, "The supernode follow module"):
// origin-copy, so private and rendering L1 origins and sequence numbers are equal by construction;
// privateTerminalParentHash on the claim; and snap-to-commitment in place of the withhold latch. Put
// together, every field of the six-field L2BlockRef the follow protocol demands is public, so the
// endpoint needs no private credential and lives in the judge.
//
// # Construct-last, and why the order below is not arbitrary
//
// Two edges are forced, and one of them is a recorded sharp edge rather than a mere dependency:
//
//  1. THE SUPERNODE MUST BE CONSTRUCTED AND STARTED WITH THE FOLLOW MODULE ENABLED BEFORE THE
//     PRIVATE SEQUENCER'S LIGHTCL POINTS AT `/claimed`. With the module disabled the sibling route
//     is NOT a 404: the per-chain handler mounts its root JSON-RPC at "/", so `/claimed` falls
//     through and answers optimism_syncStatus FOR THE RENDERING -- plausible-looking refs of the
//     wrong chain, which a sequencing LightCL force-resets onto. The ops rule until the upstream
//     oprpc fix lands is "enable the module before pointing any LightCL at the route", and here that
//     rule is an ordering plus assertClaimedRouteIsTheModule, which checks the route is the module
//     and not the fall-through before anything consumes it.
//  2. the batcher dials the rendering for its parent check and the private node for its blocks, so
//     it is last.
//
// # The not-yet state, and the bootstrap deadlock it used to cause
//
// Until the first claim lands the module serves the PRIVATE chain's GENESIS ref — one configured
// hash (derived below from the private network's genesis)
// plus five fields it derives from the rendering's config and the definition of a genesis block.
//
// It is worth recording why, because this runtime is what found it. An earlier module ERRORED until
// the first claim, on the reasoning that a follow-mode sequencer's initial engine reset comes from
// sync.FindL2Heads over real L1 and its own EL, never from the follow source. That reasoning was
// correct and incomplete: THE SEQUENCER IS NOT THE ONLY CONSUMER OF THE op-node THE SOURCE FEEDS.
// The pair's BATCHER polls the same op-node, whose reported CurrentL1 in follow mode has exactly
// one writer — the driver forwards the follow source's `current_l1` verbatim
// (op-node/rollup/driver/driver.go:311-313), because derivation is off. An erroring module left
// that zero, and op-batcher's very first check rejects a status with a zero CurrentL1 outright
// (op-batcher/batcher/sync_actions.go:75-81, "empty BlockRef in sync status") and loads no blocks
// at all. No blocks, no batch; no batch, no claim; no claim, the not-yet state never ended.
//
// Measured on this runtime before the fix: the private chain sequenced to block 126, the batcher
// logged "empty BlockRef in sync status" 486 times, and the rendering never left block 0 — while
// the counterparty's batcher, whose follow source is the supernode's ordinary chain route,
// published 324 transactions. A pair could not bootstrap and the acceptance suite could not pass.
//
// With the genesis ref served from t=0 the private chain starts, sequences, and holds safe at
// genesis while its batcher runs normally; the first claim then moves safe and finalized forward in
// one step.
//
// # P2P severance
//
// The private chain and its rendering share a chain ID and are DIFFERENT CHAINS: same numbers, same
// timestamps, different content, different genesis hash. A stock node that peered across that
// boundary would gossip a conflicting history into a node that has every reason to believe it.
//
// The severance is not a mechanism. connectL2CLPeers and connectL2ELPeers are explicit calls, and
// the light-sequencer runtime makes four of them; this runtime makes them only WITHIN the public
// side, and never across. That is the whole implementation, and it is why the rule is written down
// here rather than enforced by a type: there is nothing to enforce, only something not to do.
//
// The private sequencer also runs CLSync rather than the light-sequencer path's ELSync. ELSync
// exists there so a light sequencer can reorg onto the supernode's replacement block by syncing
// bodies from its paired supernode EL -- a peering this topology forbids -- and an ELSync sequencer
// with no peer simply deadlocks waiting for a payload nobody will send it. What the private chain
// does instead, under snap-to-commitment, is force-reset onto a claimed ref through stock
// consolidation: the claim is the operator's binding public statement, so moving TO it is recovery,
// and a claim naming a block that exists nowhere stalls loudly in sync rather than corrupting
// anything.

// privateInteropRuntime is everything the pair added to the world, kept so the preset layer and the
// invariant checker can reach it.
type privateInteropRuntime struct {
	// Config is the pair's resolved configuration.
	Config PrivateInteropConfig
	// Rendering is the public half's network: same chain ID as the private half, different genesis.
	Rendering *L2Network
	// FollowSource is the URL the private sequencer's op-node polls for its own safe head: the
	// supernode's `<base>/<chainID>/claimed` sibling route. It is kept so a test can assert what the
	// sequencer is actually pointed at, which is the one thing the sharp edge above is about.
	FollowSource string
}

// privateInteropFollowFlags builds the supernode's --private-interop.* flag group for this pair.
//
// It is a *cli.Context over a synthetic flag set rather than a config struct, because that is the
// module's only config door: supernode.initClaimFollow reads the group off config.CLIConfig.RawCtx,
// and everything downstream of that -- ReadCLIConfig, the all-or-nothing Check, Resolve -- is the
// code an operator's flags go through. Inventing a second door would leave the operator's path
// untested and test one nothing runs.
func privateInteropFollowFlags(t devtest.T, privateChainID eth.ChainID, privateGenesisPath string) *cli.Context {
	set := flag.NewFlagSet("private-interop-follow", flag.ContinueOnError)
	for _, f := range claimfollow.Flags {
		t.Require().NoError(f.Apply(set), "registering %s", f.Names()[0])
	}
	for _, kv := range [][2]string{
		{claimfollow.ChainIDFlag.Name, privateChainID.String()},
		{claimfollow.GenesisPathFlag.Name, privateGenesisPath},
		// Scanning from genesis is right here and wrong in production, where an operator enabling
		// the module against a long-lived rendering sets it to the block the first claim landed in
		// rather than walking the whole chain.
		{claimfollow.ScanStartBlockFlag.Name, "0"},
	} {
		t.Require().NoError(set.Set(kv[0], kv[1]), "setting %s", kv[0])
	}
	return cli.NewContext(cli.NewApp(), set, nil)
}

// claimedFollowSourceURL is the sibling route a private chain's stock op-node points
// --l2.follow.source at: the supernode's own per-chain route plus one path segment.
func claimedFollowSourceURL(renderingCL L2CLNode) string {
	return renderingCL.UserRPC() + "/" + claimfollow.DefaultRoute
}

// assertClaimedRouteIsTheModule checks that `/claimed` is the follow module and not the supernode's
// per-chain handler answering through a fall-through.
//
// This guards a recorded sharp edge (DESIGN.md, "Sharp edge ... a DISABLED follow module's route is
// not a 404"): the chain container mounts its root JSON-RPC at "/", so an unregistered sibling
// sub-path does not 404 -- it serves the RENDERING chain's own optimism namespace. A private LightCL
// pointed at that gets plausible refs of the wrong chain and force-resets toward a hash no private
// peer holds. It is a loud stall rather than silent corruption, but it is an expensive way to find
// out that a flag was missing.
//
// Two checks, and both are cheap because the module's surface is deliberately one method wide:
//
//   - optimism_rollupConfig must FAIL. The module serves optimism_syncStatus and nothing else, so a
//     route that answers rollupConfig is the rendering's handler. This check is not time-sensitive
//     and holds for the life of the process.
//   - optimism_syncStatus must SUCCEED and name the PRIVATE chain's genesis hash. The module now
//     serves the private genesis ref until the first claim lands, so "it errors" is no longer the
//     module's signature — but the ref it serves is a sharper one, because the two chains differ in
//     exactly this field. Called here, before any batcher exists and therefore before any claim can
//     have landed, the answer must be the private genesis; the fall-through would report the
//     RENDERING's own status, whose safe head is the rendering's genesis and never the private one.
func assertClaimedRouteIsTheModule(t devtest.T, logger log.Logger, url string, privateGenesisHash common.Hash) {
	require := t.Require()
	rpcCl, err := client.NewRPC(t.Ctx(), logger, url, client.WithLazyDial())
	require.NoError(err, "dialling the claimed follow route %s", url)
	defer rpcCl.Close()

	var cfg any
	require.Error(rpcCl.CallContext(t.Ctx(), &cfg, "optimism_rollupConfig"),
		"%s answered optimism_rollupConfig, so it is the RENDERING chain's handler and not the claim follow module: "+
			"the module serves optimism_syncStatus and nothing else. Pointing the private sequencer here would feed it "+
			"refs of the wrong chain", url)

	var status eth.SyncStatus
	require.NoError(rpcCl.CallContext(t.Ctx(), &status, "optimism_syncStatus"),
		"%s served no sync status; the follow module must answer with the private chain's genesis ref from its "+
			"first tick, because the pair's batcher will not load a block until op-node reports a current_l1", url)
	require.Equal(privateGenesisHash, status.SafeL2.Hash,
		"%s reported safe head %s, which is not the PRIVATE chain's genesis %s -- the route fell through to the "+
			"rendering's own handler. Pointing the private sequencer here would force-reset it onto the wrong chain",
		url, status.SafeL2.Hash, privateGenesisHash)
	require.Equal(privateGenesisHash, status.LocalSafeL2.Hash, "%s local safe head is not the private genesis", url)
	require.Equal(privateGenesisHash, status.FinalizedL2.Hash, "%s finalized head is not the private genesis", url)
	logger.Info("The claimed follow route is the module, not the fall-through",
		"url", url, "private_genesis", privateGenesisHash, "current_l1", status.CurrentL1)
}

// privateInteropBatcherOption turns the devstack's stock batcher configuration into the publisher of
// a private interop pair.
//
// It goes through the CLI config rather than reaching for ConfigurePrivateInterop, so that the
// devstack exercises the same path an operator does: the flag group, its all-or-nothing Check, the
// Resolve, and initPrivateInterop's own assembly of the follower and the range source.
// The alternative -- constructing the terminal seam directly -- would mean the devstack tests a
// component wiring that no deployment uses.
func privateInteropBatcherOption(
	t devtest.T,
	cfg PrivateInteropConfig,
	renderingRollup *rollup.Config,
	privateGenesisPath string,
	renderingRPC string,
	depSetHash common.Hash,
) BatcherOption {
	rollupConfigHash := hashOfJSON(t, renderingRollup, "the rendering's rollup config")
	return func(_ ComponentTarget, c *bss.CLIConfig) {
		// The cadence owns where a range ends, and these two stock knobs would otherwise take that
		// away. A range that ends somewhere its claim did not say it would is a range whose parent
		// check the next one gets wrong -- and one span batch per cadence is the ratified shape, not
		// an approximation of it.
		if c.MaxBlocksPerSpanBatch < int(cfg.MaxBlocksPerRange) {
			c.MaxBlocksPerSpanBatch = int(cfg.MaxBlocksPerRange)
		}
		// Zero disables the duration check entirely (op-batcher/batcher/channel_config.go:24).
		c.MaxChannelDuration = 0
		c.PrivateInterop = bss.PrivateInteropCLIConfig{
			PrivateChainGenesisPath: privateGenesisPath,
			PublicProjectionRPC:     renderingRPC,
			MaxBlocksPerRange:       cfg.MaxBlocksPerRange,
			MaxRangeBytes:           batcherFlags.DefaultPrivateInteropMaxRangeBytes,
			// No L1 confirmation depth, and no L1 view at all: under origin-copy the transformation
			// reuses each private block's OWN L1 origin as the rendering block's epoch, so there is
			// no origin to choose and nothing for a shallow L1 reorg to orphan.

			RollupConfigHash: rollupConfigHash.Hex(),
			DepSetHash:       depSetHash.Hex(),

			GasLimitExport: 500_000,
			GasLimitImport: 500_000,
			GasLimitEvent:  500_000,
			GasLimitClaim:  500_000,
		}
	}
}

// hashOfJSON is the devstack's convention for the claim's two configuration commitments.
//
// The claim commits to WHICH CHAIN and WHICH DEPENDENCY SET it speaks for, and the codec takes both
// as opaque 32-byte values -- neither op-private-interop nor op-node defines how they are computed,
// because nothing yet reads them back. A deterministic hash of the canonical JSON is therefore
// enough for every property the devstack can check (the same configuration hashes the same, a
// different one differs) and is explicitly NOT a claim about the production convention. When one is
// ratified, this is the single place the devstack changes.
func hashOfJSON(t devtest.T, v any, what string) common.Hash {
	encoded, err := json.Marshal(v)
	t.Require().NoErrorf(err, "encoding %s for its claim commitment", what)
	return crypto.Keccak256Hash(encoded)
}

// writePrivateChainGenesis puts the private-chain genesis where consumers can derive the public
// projection from the same artifact that is supplied to the private execution client.
func writePrivateChainGenesis(t devtest.T, genesis *core.Genesis) string {
	dir := t.TempDirWithPrefix("private-interop")
	path := filepath.Join(dir, "genesis.json")
	encoded, err := json.Marshal(genesis)
	t.Require().NoError(err, "encoding the private-chain genesis")
	t.Require().NoError(os.WriteFile(path, encoded, 0o644), "writing the private-chain genesis")
	return path
}

// privateInteropChainIDs names which chain of a two-L2 world becomes the pair.
//
// Chain B, always. The counterparty must be an ordinary public chain -- it is what a fabricated
// import is checked against, and what the trust model rests on -- and picking it by position keeps
// every existing two-L2 test's mental model of "A is normal" intact.
func privateInteropChainIDs() (private, counterparty eth.ChainID) {
	return DefaultL2BID, DefaultL2AID
}

// NewTwoL2PrivateInteropRuntimeWithConfig builds a two-L2 world in which chain B is a private
// interop pair.
//
// Chain A is untouched: an ordinary public chain with a light sequencer, a supernode verifier route
// and a stock batcher, exactly as the light-sequencer preset builds it. That is deliberate and
// load-bearing rather than laziness -- the counterparty is what a fabricated import is checked
// against, so the trust model only says anything if the counterparty is a chain the operator has no
// special relationship with.
func NewTwoL2PrivateInteropRuntimeWithConfig(t devtest.T, delaySeconds uint64, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()
	require.NotNil(cfg.PrivateInterop, "the private interop runtime needs a private interop configuration")
	pi := *cfg.PrivateInterop
	require.NoError(pi.Check(), "invalid private interop configuration")

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	privateID, _ := privateInteropChainIDs()
	batcherKey, err := keys.Secret(devkeys.BatcherRole.Key(privateID.ToBig()))
	require.NoError(err, "deriving the standard batcher key")
	batcherAddr := crypto.PubkeyToAddress(batcherKey.PublicKey)

	// Interop at genesis, always: the projection is only defined over a Lagoon-at-genesis source,
	// because an activation block on the rendering would run the stock network-upgrade bundle and
	// replace the replay messenger. The supernode's activation override below follows suit.
	require.Zero(delaySeconds, "a private interop pair activates interop at genesis; a delayed activation has no projection")
	wb, l1Net, l2ANet, l2BNet := buildTwoL2RuntimeWorld(t, keys, true, 0, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...)

	// Install the private policy on ordinary ETH artifacts before constructing either EL.
	privateGenesis, privateRollup, err := projectiongenesis.ConfigurePrivateGenesis(l2BNet.genesis, l2BNet.rollupCfg)
	require.NoError(err, "configuring the private ETH profile")
	l2BNet.genesis, l2BNet.rollupCfg = privateGenesis, privateRollup
	wb.outL2Genesis[privateID], wb.outL2RollupCfg[privateID] = privateGenesis, privateRollup

	// Share the deployment and keys, but derive the projection's own genesis and rollup config.
	renderingNet := ptr.New(*l2BNet)
	renderingNet.name += "-public-projection"
	renderingNet.genesis, err = projectiongenesis.ProjectGenesisFrom(privateGenesis)
	require.NoError(err, "projecting the private-chain genesis")
	renderingNet.rollupCfg, err = projectiongenesis.ProjectRollupConfigFrom(privateRollup, renderingNet.genesis)
	require.NoError(err, "projecting the private-chain rollup config")
	require.NotEqual(privateRollup.Genesis.L2.Hash, renderingNet.rollupCfg.Genesis.L2.Hash)

	migration := newInteropMigrationState(wb)
	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock()
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClockConfig(t, l1Net, jwtPath, l1Clock, cfg)

	// The four execution clients, named for what they are. Two belong to chain A (a supernode
	// verifier and a sequencer, peered), and two belong to chain B's pair (the rendering and the
	// private chain, NEVER peered -- they share a chain ID and disagree about its content).
	supernodeAEL := startSupernodeEL(t, l2ANet, jwtPath, jwtSecret)
	renderingEL := startL2ELForKey(t, renderingNet, jwtPath, jwtSecret, "rendering", NewELNodeIdentity(0), ResolveMixedL2ELOpts(t)...)
	var seqAEL, privateEL L2ELNode
	var filterProxy *tcpproxy.Proxy
	if cfg.UseInteropFilter {
		// Allocate the stable filter address before either sequencing EL starts. The filter itself
		// starts after the EL RPCs exist, then the proxy is connected atomically.
		filterProxy = tcpproxy.New(t.Logger().New("proxy", "interop-filter"))
		require.NoError(filterProxy.Start())
		t.Cleanup(func() { filterProxy.Close() })
		filterRPC := "http://" + filterProxy.Addr()
		seqAEL = startSupernodeELWithInteropURL(t, l2ANet, "sequencer-a", jwtPath, jwtSecret, filterRPC)
		privateEL = startSupernodeELWithInteropURL(t, l2BNet, "private-sequencer", jwtPath, jwtSecret, filterRPC)
	} else {
		seqAEL = startSequencerEL(t, l2ANet, jwtPath, jwtSecret, NewELNodeIdentity(0), ResolveMixedL2ELOpts(t)...)
		privateEL = startSequencerEL(t, l2BNet, jwtPath, jwtSecret, NewELNodeIdentity(0), ResolveMixedL2ELOpts(t)...)
	}

	var interopFilter *InteropFilter
	if cfg.UseInteropFilter {
		// The filter reads PRIVATE blocks for chain B, but its chain-B rollup config is the public
		// rendering config, and --private-interop.chain-id names chain B so that ingestion transforms
		// the private logs to their canonical rendered positions. No extra emitters: the emitter set
		// is the two standard interop predeploys, the same set the batcher renders with.
		rollupConfigs := map[eth.ChainID]*rollup.Config{
			eth.ChainIDFromBig(l2ANet.RollupConfig().L2ChainID):       l2ANet.RollupConfig(),
			eth.ChainIDFromBig(renderingNet.RollupConfig().L2ChainID): renderingNet.RollupConfig(),
		}
		interopFilter = startInteropFilter(t, "interop-filter",
			[]string{seqAEL.UserRPC(), privateEL.UserRPC()}, rollupConfigs,
			func(c *filter.Config) { c.PrivateInteropChainID = &privateID })
		filterProxy.SetUpstream(ProxyAddr(require, interopFilter.HTTPEndpoint()))
	}

	activationTime := l2ANet.rollupCfg.Genesis.L2Time + delaySeconds
	// Both halves take their genesis timestamp from the same L1 block, so one activation time is
	// correct for the pair as well as for the counterparty. If that ever stopped being true the
	// cross-safe frontier would hang with every component looking healthy, which is the failure mode
	// this whole pairing exists to make impossible.
	require.Equal(l2ANet.rollupCfg.Genesis.L2Time, renderingNet.rollupCfg.Genesis.L2Time,
		"the world's chains must share a genesis timestamp for one interop activation to fit them all")

	depSet, runtimeDepSet := resolveRuntimeDepSet(t, wb, cfg)
	privateGenesisPath := writePrivateChainGenesis(t, l2BNet.genesis)

	// The supernode judges {chain A, the RENDERING} and has no idea a private chain exists. That is
	// the design's central claim about the public side, and it is true here by construction: the
	// only chain-B endpoint it is given is the rendering's.
	supernode, supernodeACL, renderingCL := startTwoL2SharedSupernode(
		t, l1Net, l1EL, l1CL,
		l2ANet, supernodeAEL,
		renderingNet, renderingEL,
		depSet,
		&activationTime,
		cfg.InteropLogBackfillDepth,
		jwtSecret,
		false, // verifier routes only; chain A is sequenced by its light CL
		// The follow module, enabled HERE rather than after the fact: the module has to exist before
		// anything points a LightCL at its route, because a disabled module's route is not a 404.
		privateInteropFollowFlags(t, privateID, privateGenesisPath),
	)

	// Construct-last edge 1, and the sharp edge: the supernode above was built WITH the follow module
	// enabled, so `/claimed` is the module. Check that before anything is pointed at it -- a
	// fall-through here would feed the private sequencer the rendering's own refs, and this is the
	// last moment where "no claim has landed yet" makes the check unambiguous.
	followSource := claimedFollowSourceURL(renderingCL)
	assertClaimedRouteIsTheModule(t, t.Logger().New("component", "claim-follow"), followSource,
		l2BNet.rollupCfg.Genesis.L2.Hash)

	// Chain A: the ordinary light sequencer, following the supernode and peered to it.
	//
	// CLSync, where the stock light-sequencer preset uses ELSync. That preset's ELSync sequencers
	// cannot bootstrap a chain from genesis as the sole producer -- they deadlock in willStartEL
	// waiting for a peer payload (#21164) -- which is why it offers a VN-sequencer handoff to get
	// them started. This runtime cannot use that handoff: it would have to enable sequencing on the
	// supernode's routes, and one of those routes is the RENDERING, a chain that must never sequence
	// anything. So both chains here build their own blocks from genesis, which is what CLSync does.
	l2ACL := startL2CLNode(t, keys, l1Net, l2ANet, l1EL, l1CL, seqAEL, jwtSecret, l2CLNodeStartConfig{
		Key:            "sequencer",
		IsSequencer:    true,
		NoDiscovery:    true,
		EnableReqResp:  true,
		DependencySet:  runtimeDepSet,
		L2FollowSource: supernodeACL.UserRPC(),
		L2CLOptions:    cfg.GlobalL2CLOptions,
	})
	connectL2CLPeers(t, t.Logger(), l2ACL, supernodeACL)
	connectL2ELPeers(t, t.Logger(), supernodeAEL.UserRPC(), seqAEL.UserRPC())

	// Chain B, private half: the same stock op-node, pointed at the supernode's `/claimed` sibling
	// route instead of a same-chain CL. The module is already enabled and checked, which is the
	// ordering the sharp edge demands.
	//
	// CLSync, not ELSync, and no peers at all. See the package comment: ELSync is the light-sequencer
	// preset's answer to reorging onto a supernode replacement over EL p2p, and this chain has
	// neither a peer to sync from nor a replacement to reorg onto.
	privateCL := startL2CLNode(t, keys, l1Net, l2BNet, l1EL, l1CL, privateEL, jwtSecret, l2CLNodeStartConfig{
		Key:            "sequencer",
		IsSequencer:    true,
		NoDiscovery:    true,
		EnableReqResp:  true,
		DependencySet:  runtimeDepSet,
		L2FollowSource: followSource,
		L2CLOptions:    cfg.GlobalL2CLOptions,
	})
	// No connectL2CLPeers and no connectL2ELPeers across the pair. This absence is the severance.

	// Construct-last edge 3: the batchers, which read both chains.
	depSetHash := hashOfJSON(t, runtimeDepSet, "the dependency set")
	piBatcherOpt := privateInteropBatcherOption(
		t, pi, renderingNet.rollupCfg, privateGenesisPath, renderingEL.UserRPC(),
		depSetHash,
	)

	l2ABatcher := startMinimalBatcher(t, keys, l2ANet, l1EL, l2ACL, seqAEL, cfg.BatcherOptions...)
	l2AProposer := startMinimalProposer(t, keys, l2ANet, l1EL, supernodeACL)
	// The pair's batcher loads PRIVATE blocks and posts the RENDERING's batches. Its own rollup
	// config comes from --rollup-rpc, which is the private chain's op-node; the rendering's comes
	// from the file written above, because no node it can reach serves it.
	piBatcher := startMinimalBatcher(t, keys, l2BNet, l1EL, privateCL, privateEL,
		append(append([]BatcherOption{}, cfg.BatcherOptions...), piBatcherOpt)...)
	// The rendering's proposer proposes the rendering's output roots: the rendering is the chain the
	// dependency set names, so it is the chain that settles.
	renderingProposer := startMinimalProposer(t, keys, renderingNet, l1EL, renderingCL)

	runtime := &MultiChainRuntime{
		Keys:          keys,
		Migration:     migration,
		FullConfigSet: wb.outFullCfgSet,
		DependencySet: runtimeDepSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains: map[string]*MultiChainNodeRuntime{
			"l2a": {
				Name:        "l2a",
				Network:     l2ANet,
				EL:          seqAEL,
				CL:          l2ACL,
				SupernodeCL: supernodeACL,
				SupernodeEL: supernodeAEL,
				Batcher:     l2ABatcher,
				Proposer:    l2AProposer,
			},
			"l2b": {
				Name:             "l2b",
				Network:          l2BNet,
				RenderingNetwork: renderingNet,
				EL:               privateEL,
				CL:               privateCL,
				SupernodeCL:      renderingCL,
				SupernodeEL:      renderingEL,
				Batcher:          piBatcher,
				Proposer:         renderingProposer,
			},
		},
		Supernode:     supernode,
		InteropFilter: interopFilter,
		TimeTravel:    timeTravelClock,
		DelaySeconds:  delaySeconds,
		PrivateInterop: &privateInteropRuntime{
			Config:       pi,
			Rendering:    renderingNet,
			FollowSource: followSource,
		},
	}
	// The deterministic block builder, the same one every two-L2 interop preset attaches. The
	// private chain is an ordinary sequenced chain from its own side, so it takes one exactly as
	// chain A does.
	attachTestSequencerToRuntime(t, runtime, "test-sequencer-2l2-private-interop")

	t.Logger().Info("configured a private interop pair",
		"chain_id", privateID,
		"genesis_time", l2BNet.rollupCfg.Genesis.L2Time,
		"activation_time", activationTime,
		"private_genesis", l2BNet.rollupCfg.Genesis.L2.Hash,
		"rendering_genesis", renderingNet.rollupCfg.Genesis.L2.Hash,
		"batcher", batcherAddr,
		"cadence_blocks", pi.MaxBlocksPerRange,
	)
	return runtime
}

// resolveRuntimeDepSet is the dependency-set extraction the two-L2 runtimes share: the static set the
// world built, with the preset's message-expiry override applied if it asked for one.
func resolveRuntimeDepSet(t devtest.T, wb *worldBuilder, cfg PresetConfig) (*depset.StaticConfigDependencySet, depset.DependencySet) {
	require := t.Require()
	var static *depset.StaticConfigDependencySet
	if wb.outFullCfgSet.DependencySet != nil {
		cast, ok := wb.outFullCfgSet.DependencySet.(*depset.StaticConfigDependencySet)
		require.True(ok, "expected static dependency set")
		static = cast
	}
	if cfg.MessageExpiryWindow != nil && static != nil {
		var err error
		static, err = depset.NewStaticConfigDependencySetWithMessageExpiryOverride(
			static.Dependencies(), *cfg.MessageExpiryWindow)
		require.NoError(err, "failed to override message expiry window")
	}
	if static != nil {
		return static, static
	}
	return nil, wb.outFullCfgSet.DependencySet
}
