package sysgo

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	opnodeconfig "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	snconfig "github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
)

// The silhouette seam in devstack: what it takes to run one of a preset's chains as a proof-carried
// chain, plus a second supernode that consumes its proofs.
//
// Four things here are not obvious, and every one of them is a property of silhouette rather than of
// the harness.
//
// Sequencing belongs to P's LightCL node. The silhouette supernode is verifier-only and its entire
// access to P's public history is the proof stream on L1.
//
// P uses the ordinary op-batcher pipeline, with only its terminal encoding and destination changed.
// It loads real blocks and follows the stock reorg/retry/txmgr path, then emits transaction-stripped
// proof batches to the proof-batch inbox.
//
// The devstack L1's params.ChainConfig is written to a FILE and named in the verifier config.
// silhouette.L1ChainConfig knows the four public networks and refuses everything else by design
// (ruling G5 D3): the value it reads decides whether an L1 block's excess blob gas is priced under
// Cancun or Prague, for the L1-info transaction's blob-base-fee field, and inventing that for an
// unknown L1 would put a fabricated number in a consensus-relevant transaction. A devstack L1 has a
// private chain ID, so naming the file is the only way in — and the file's chain ID must match the
// config's, which TestSilhouetteRefusesWrongL1ChainConfig asserts by breaking it on purpose.
//
// The ANCHOR is P's own genesis. silhouette.Assemble requires the virtual node's rollup config
// genesis to sit AT the anchor whenever the anchor is above P's real block 0 (G3 F2), because
// op-node always derives from its rollup config's genesis. Anchoring at genesis is the one choice
// that needs no doctored rollup config, and it makes the whole of P's history proof-carried.

// SilhouetteRuntime is the silhouette half of a runtime: the chain that is proof-carried, the
// verifier that consumes its proofs, and the ordinary batcher that produces them.
type SilhouetteRuntime struct {
	// ChainKey is the runtime chain that is proof-carried, and ChainID its chain ID.
	ChainKey string
	ChainID  eth.ChainID

	// Verifier is the second supernode: chain P as a silhouette virtual node, every other chain
	// ordinary and derived from L1. It is the only node in the system whose verdict about P's
	// messages is a statement about proofs.
	Verifier *SuperNode
	// VerifierRoutes are the verifier's per-chain rollup routes, keyed by runtime chain key.
	VerifierRoutes map[string]*SuperNodeProxy
	// VerifierELs are the stateful execution clients the verifier's ORDINARY chains drive, keyed by
	// runtime chain key. The silhouette chain has no entry: its standalone magic EL serves
	// proof-committed facts, persists no state, and executes nothing.
	VerifierELs map[string]L2ELNode
	// VerifierMetricsURL is the verifier's Prometheus endpoint. It exists so a test can assert on
	// supernode_interop_invalidations_total, which is the only honest way to say "zero
	// invalidations" rather than "we did not notice one".
	VerifierMetricsURL string
	// MagicELRPC is the standalone op-silhouette-el endpoint used by the verifier virtual node.
	MagicELRPC string

	// Batcher drives and observes P's ordinary op-batcher in acceptance tests.
	Batcher *ProofBatchControl

	// ManifestPath and Config are the files the verifier was started from.
	ManifestPath string
	Config       silhouette.Config

	configDir     string
	l1ChainConfig *params.ChainConfig
	tryVerifier   func(manifestPath string) error
}

// TryVerifierWithManifest constructs (but does not start) a verifier supernode against an arbitrary
// silhouette manifest and RETURNS the construction error instead of failing the test.
//
// It exists because the interesting silhouette startup behaviours are refusals, and a refusal is only
// a gate if a test can observe it. The normal path fails the test on a construction error, which is
// right for a preset and useless for asserting that a misconfiguration is rejected.
func (s *SilhouetteRuntime) TryVerifierWithManifest(manifestPath string) error {
	return s.tryVerifier(manifestPath)
}

// BlockProvenance asks the silhouette chain's own route how the verifier knows about a block:
// "proven" (its hash and roots came off the wire inside something a proof committed to), "forced"
// (the forced-extension convention computed them) or "genesis" (configuration).
//
// This, and not eth_getBlockByNumber, is how a test should read the silhouette chain on the verifier.
// The shim's rpcBlock carries the hash the PROOF committed to, which is deliberately not
// keccak(RLP(header)) — the header's interior is a rendering of a block whose interior is private —
// so a client that verifies block hashes (every devstack execution-layer frontend does) rejects it,
// correctly. The declaration namespace exists precisely so a caller can ask what is known rather
// than pretend to re-derive it.
func (s *SilhouetteRuntime) BlockProvenance(t devtest.T, number uint64) *silhouette.BlockDeclaration {
	require := t.Require()
	out, err := s.TryBlockProvenance(t, number)
	require.NoErrorf(err, "silhouette_blockProvenance(%d)", number)
	return out
}

// TryBlockProvenance is the non-fatal form of BlockProvenance. It is useful while a denied proof
// suffix is being replaced, when the requested height may temporarily be absent.
func (s *SilhouetteRuntime) TryBlockProvenance(t devtest.T, number uint64) (*silhouette.BlockDeclaration, error) {
	if s.MagicELRPC == "" {
		return nil, fmt.Errorf("verifier has no magic EL endpoint for the silhouette chain")
	}
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), s.MagicELRPC, client.WithLazyDial())
	if err != nil {
		return nil, fmt.Errorf("failed to dial the silhouette chain's verifier route: %w", err)
	}
	defer rpcCl.Close()

	var out silhouette.BlockDeclaration
	if err := rpcCl.CallContext(t.Ctx(), &out, "silhouette_blockProvenance", hexutil.Uint64(number)); err != nil {
		return nil, err
	}
	return &out, nil
}

// ManifestWithWrongL1ChainID writes a second, complete set of silhouette config files that differ
// from the live ones in exactly one respect: the L1 chain-config file they name is for a different
// chain. Starting a verifier from it must fail.
func (s *SilhouetteRuntime) ManifestWithWrongL1ChainID(t devtest.T) string {
	wrong := *s.l1ChainConfig
	wrong.ChainID = new(big.Int).Add(s.l1ChainConfig.ChainID, big.NewInt(1))
	return writeSilhouetteManifest(t, filepath.Join(s.configDir, "wrong-l1"), &wrong, s.Config, s.ChainID)
}

// silhouetteBatcherOption starts the silhouette chain's ordinary op-batcher STOPPED and swaps only
// its terminal channel encoding and inbox. Devstack acceptance tests can control publication
// deterministically; op-up starts this same batcher for an autonomous network.
//
// Everything before encoding remains the stock path: unsafe-head polling, reorg reconciliation,
// block loading, channel retry, txmgr and blob submission. The encoder copies commitments, logs and
// imports from each real block but deliberately has no transaction-list field.
func silhouetteBatcherOption(chainID eth.ChainID, inbox common.Address, rollupConfigHash, depSetHash common.Hash) BatcherOption {
	testHooks := bss.NewProofBatchTestHooks()
	return func(target ComponentTarget, cfg *bss.CLIConfig) {
		if target.ChainID != chainID {
			return
		}
		cfg.Stopped = true
		cfg.DataAvailabilityType = batcherFlags.BlobsType
		// A proof batch is already a complete multi-blob payload. Keeping a single transaction in
		// flight avoids producing overlapping ranges while the verifier's safe-head acknowledgement
		// travels back through the LightCL follow route.
		cfg.MaxPendingTransactions = 1
		cfg.TargetNumFrames = params.DefaultPragueBlobConfig.Max
		cfg.ProofBatch = &bss.ProofBatchConfig{
			Inbox:            inbox,
			RollupConfigHash: rollupConfigHash,
			DepSetHash:       depSetHash,
			WireVersion:      proofbatch.Version,
			MaxBlocks:        300,
			TestHooks:        testHooks,
		}
	}
}

type silhouetteRPCOutput struct {
	OutputRoot            common.Hash `json:"outputRoot"`
	StateRoot             common.Hash `json:"stateRoot"`
	WithdrawalStorageRoot common.Hash `json:"withdrawalStorageRoot"`
	BlockRef              struct {
		Hash common.Hash `json:"hash"`
		Time uint64      `json:"timestamp"`
	} `json:"blockRef"`
}

// silhouetteAnchor reads P's genesis as the anchor of proven history.
//
// The output root is asked of the chain rather than computed here: it is keccak over the state root
// and the L2ToL1MessagePasser storage root, and the node already publishes exactly that value. A
// second computation could only be a way to disagree with it.
func silhouetteAnchor(t devtest.T, logger log.Logger, rollupRPC string, genesisL2 eth.BlockID, genesisTime uint64, genesisL1 eth.BlockID) silhouette.Anchor {
	require := t.Require()
	rpcCl, err := client.NewRPC(t.Ctx(), logger, rollupRPC, client.WithLazyDial())
	require.NoError(err, "failed to dial the silhouette chain's rollup node for its anchor")
	defer rpcCl.Close()

	var out silhouetteRPCOutput
	// The route is up (waitForSupernodeRoute ran) but the EL may still be answering; retry briefly.
	deadline := time.Now().Add(30 * time.Second)
	for {
		err = rpcCl.CallContext(t.Ctx(), &out, "optimism_outputAtBlock", hexutil.Uint64(genesisL2.Number))
		if err == nil {
			break
		}
		require.False(time.Now().After(deadline), "failed to read the silhouette chain's genesis output root: %v", err)
		time.Sleep(250 * time.Millisecond) // nosemgrep: flake-sleep-in-test -- polling a route that has just come up
	}
	require.Equal(genesisL2.Hash, out.BlockRef.Hash,
		"the silhouette chain's genesis block hash disagrees with its rollup config")
	require.NotEqual(common.Hash{}, out.OutputRoot, "the silhouette chain reported a zero genesis output root")

	return silhouette.Anchor{
		OutputRoot:  out.OutputRoot,
		BlockNumber: genesisL2.Number,
		BlockHash:   genesisL2.Hash,
		Timestamp:   genesisTime,
		// The anchor's L1 origin seeds the rendered-origin walk (G2 D4): without it the first batch
		// has no epoch to advance from.
		L1Origin: genesisL1,
	}
}

// writeSilhouetteManifest writes the three files a silhouette supernode reads — the settlement
// chain's params.ChainConfig, the verifier config, and the manifest naming it — into dir, and returns
// the manifest path. The verifier config is taken by value because this function OWNS setting
// L1ChainConfigPath: the path is a property of where the files were written, not of the caller's
// intent.
func writeSilhouetteManifest(t devtest.T, dir string, l1Chain *params.ChainConfig, cfg silhouette.Config, chainID eth.ChainID) string {
	require := t.Require()
	require.NoError(os.MkdirAll(dir, 0o755), "create silhouette config dir %s", dir)

	write := func(name string, v any) string {
		raw, err := json.MarshalIndent(v, "", "  ")
		require.NoErrorf(err, "encode %s", name)
		path := filepath.Join(dir, name)
		require.NoErrorf(os.WriteFile(path, append(raw, '\n'), 0o644), "write %s", path)
		return path
	}

	cfg.L1ChainConfigPath = write("l1-chain-config.json", l1Chain)
	cfgPath := write("verifier.json", cfg)
	return write("manifest.json", silhouette.Manifest{Chains: []silhouette.ManifestChain{{
		ChainID:        eth.EvilChainIDToUInt64(chainID),
		VerifierConfig: cfgPath,
	}}})
}

// silhouetteBindings computes the two wire commitments a verifier checks a batch against.
//
// Neither is derived from anything in the Go stack today — the real values come from what the prover
// derived under — so they are configuration on both sides. Hashing the artifacts the chain actually
// runs, rather than picking constants, buys the property that matters: change the chain and the
// commitment changes, so a batch built for one chain cannot be accepted as another's history.
func silhouetteBindings(t devtest.T, rollupCfg *rollup.Config, depSet *depset.StaticConfigDependencySet) (rollupConfigHash, depSetHash common.Hash) {
	var err error
	rollupConfigHash, depSetHash, err = silhouette.BindingHashes(rollupCfg, depSet)
	t.Require().NoError(err, "compute silhouette artifact bindings")
	return rollupConfigHash, depSetHash
}

// silhouetteInteropActivation is the first L2 timestamp the VERIFIER verifies, and it is one block
// above the silhouette chain's anchor rather than the cluster's activation timestamp.
//
// This is not a tuning choice, it is the only timestamp the verifier can start at. The interop round
// gates on every chain in the dependency set, and for each it opens the chain's block at the round's
// timestamp in that chain's message database. A silhouette chain's database is filled by its log
// sink, which seals the blocks that arrive in PROOF BATCHES — and a batch starts at anchor+1, because
// acceptance rule 3 requires it to extend the anchor rather than restate it. So the anchor block
// itself is never sealed: it is configuration on both sides, not something a proof delivered.
//
// A verifier started at the anchor's own timestamp therefore asks its message database for a block
// the database will never hold, and fails that round forever — observed as a hard loop of
// "chain <P>: failed to open block 0: skipped data" with the proof pipeline itself perfectly
// healthy. Starting one L2 block later asks for anchor+1, which the first accepted batch does seal,
// and until that batch lands the readiness check simply reports the chain not ready and WAITS,
// which is the correct response to a proof that has not arrived.
//
// Nothing below the anchor is referenceable anyway — a verifier's proven history starts above it —
// so this loses no coverage. It is written here rather than left to the caller because getting it
// wrong is invisible: the cluster looks healthy and the cross-safe frontier simply never starts.
func silhouetteInteropActivation(t devtest.T, clusterActivation *uint64, anchor silhouette.Anchor, blockTime uint64) *uint64 {
	t.Require().NotNil(clusterActivation, "a silhouette chain needs interop enabled")
	t.Require().NotZero(blockTime, "the silhouette chain has no block time")
	first := anchor.Timestamp + blockTime
	if *clusterActivation > first {
		first = *clusterActivation
	}
	return &first
}

// silhouetteVerifierChain is one chain as the verifier supernode should see it.
type silhouetteVerifierChain struct {
	key       string
	net       *L2Network
	engineRPC string
	// rollup overrides the private network's producer config for proof rendering. The generated
	// silhouette config keeps every fork active at genesis, so a verifier never has to reconstruct
	// an activation header from the deliberately minimal public rendering.
	rollup *rollup.Config
	// el is the execution client this chain's virtual node drives. It is nil for the silhouette
	// chain, which has none.
	el L2ELNode
}

// startSilhouetteVerifier brings up the second supernode: the one that does not derive P.
func startSilhouetteVerifier(
	t devtest.T,
	l1Net *L1Network,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	chains []silhouetteVerifierChain,
	silhouetteChainID eth.ChainID,
	manifestPath string,
	depSet *depset.StaticConfigDependencySet,
	interopActivationTimestamp *uint64,
	jwtSecret [32]byte,
) (*SuperNode, map[string]*SuperNodeProxy, string, func(string) error) {
	require := t.Require()
	logger := t.Logger().New("component", "supernode-verifier")

	makeVNCfg := func(chain silhouetteVerifierChain) *opnodeconfig.Config {
		engineAddr := ""
		if chain.el != nil {
			engineAddr = chain.el.EngineRPC()
		} else {
			engineAddr = chain.engineRPC
		}
		rollupCfg := chain.net.rollupCfg
		if chain.rollup != nil {
			rollupCfg = chain.rollup
		}
		cfg := &opnodeconfig.Config{
			L1: &opnodeconfig.L1EndpointConfig{
				L1NodeAddr:       l1EL.UserRPC(),
				L1TrustRPC:       false,
				L1RPCKind:        sources.RPCKindDebugGeth,
				RateLimit:        0,
				BatchSize:        20,
				HttpPollInterval: 100 * time.Millisecond,
				MaxConcurrency:   10,
				CacheSize:        0,
			},
			L1ChainConfig: l1Net.genesis.Config,
			L2: &opnodeconfig.L2EndpointConfig{
				L2EngineAddr:      engineAddr,
				L2EngineJWTSecret: jwtSecret,
			},
			DependencySet: depSet,
			// Blobs are not optional here: a silhouette chain's whole history travels in them, and
			// the ordinary chains' batchers may use them too.
			Beacon:                          &opnodeconfig.L1BeaconEndpointConfig{BeaconAddr: l1CL.beaconHTTPAddr},
			Driver:                          driver.Config{SequencerEnabled: false, SequencerConfDepth: 2},
			Rollup:                          *rollupCfg,
			RPC:                             oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
			L1EpochPollInterval:             2 * time.Second,
			RuntimeConfigReloadInterval:     0,
			Sync:                            nodeSync.Config{SyncMode: nodeSync.CLSync},
			ConfigPersistence:               opnodeconfig.DisabledConfigPersistence{},
			Metrics:                         opmetrics.CLIConfig{},
			Pprof:                           oppprof.CLIConfig{},
			IgnoreMissingPectraBlobSchedule: false,
			ExperimentalOPStackAPI:          true,
			// No P2P, deliberately. This node is a verifier: everything it believes about every
			// chain here comes from L1, which is exactly the claim the gate makes about P and is
			// worth making about the reference chain too. (silhouette.Assemble pins P's P2P to nil
			// regardless, because a gossiped payload at a proven height is the one input that can
			// halt a silhouette chain.)
			P2P: nil,
		}
		require.NoError(cfg.Check(), "invalid verifier op-node config for chain %s", chain.net.ChainID())
		return cfg
	}

	chainIDs := make([]uint64, 0, len(chains))
	newVNCfgs := func() map[eth.ChainID]*opnodeconfig.Config {
		out := make(map[eth.ChainID]*opnodeconfig.Config, len(chains))
		for _, chain := range chains {
			out[chain.net.ChainID()] = makeVNCfg(chain)
		}
		return out
	}
	chainList := make([]eth.ChainID, 0, len(chains))
	for _, chain := range chains {
		chainIDs = append(chainIDs, eth.EvilChainIDToUInt64(chain.net.ChainID()))
		chainList = append(chainList, chain.net.ChainID())
	}

	// The metrics endpoint is a fixed port because MetricsService takes an address and never reports
	// the one it bound. A port grabbed and released is a small race; the alternative is no
	// invalidation counter, and "zero invalidations" asserted from logs alone is not a gate.
	metricsPort := freeLoopbackPort(t)
	newSNCfg := func(manifest string, dataDirPrefix string) *snconfig.CLIConfig {
		return &snconfig.CLIConfig{
			Chains:                     chainIDs,
			DataDir:                    t.TempDirWithPrefix(dataDirPrefix),
			L1NodeAddr:                 l1EL.UserRPC(),
			L1HTTPPollInterval:         100 * time.Millisecond,
			L1BeaconAddr:               l1CL.beaconHTTPAddr,
			RPCConfig:                  oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
			MetricsConfig:              opmetrics.CLIConfig{Enabled: true, ListenAddr: "127.0.0.1", ListenPort: metricsPort},
			InteropActivationTimestamp: interopActivationTimestamp,
			SilhouetteManifestPath:     manifest,
		}
	}

	snCfg := newSNCfg(manifestPath, "supernode-verifier")
	require.NoError(snCfg.Check(), "invalid verifier supernode config")

	verifier := &SuperNode{
		p:            t,
		logger:       logger,
		chains:       chainList,
		l1UserRPC:    l1EL.UserRPC(),
		l1BeaconAddr: l1CL.beaconHTTPAddr,
		snCfg:        snCfg,
		vnCfgs:       newVNCfgs(),
	}
	verifier.Start()
	t.Cleanup(verifier.Stop)

	base := verifier.UserRPC()
	routes := make(map[string]*SuperNodeProxy, len(chains))
	for _, chain := range chains {
		route := base + "/" + strconv.FormatUint(eth.EvilChainIDToUInt64(chain.net.ChainID()), 10)
		waitForSupernodeRoute(t, logger, route)
		routes[chain.key] = &SuperNodeProxy{p: t, logger: logger, userRPC: route}
	}

	// tryVerifier constructs a supernode against an arbitrary manifest without starting it, so a
	// test can assert that a bad manifest is REFUSED. Construction is where the refusal lives:
	// supernode.New loads the manifest, resolves the L1 chain config and assembles every silhouette
	// chain before it starts anything.
	tryVerifier := func(manifest string) error {
		ctx, cancel := context.WithCancel(t.Ctx())
		cfg := newSNCfg(manifest, "supernode-verifier-try")
		sn, err := supernode.New(ctx, logger.New("attempt", "try"), "devstack", "",
			func(error) {}, cfg, newVNCfgs())
		if err != nil {
			cancel()
			return err
		}
		// It built. Tear it down rather than leaking a second supernode holding the same data dir.
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		_ = sn.Stop(stopCtx)
		return nil
	}

	logger.Info("started silhouette verifier supernode",
		"chains", len(chains), "silhouette_chain", silhouetteChainID,
		"rpc", base, "metrics", net.JoinHostPort("127.0.0.1", strconv.Itoa(metricsPort)))
	return verifier, routes, "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(metricsPort)), tryVerifier
}

// freeLoopbackPort reserves and immediately releases a loopback port, for a server whose API takes a
// port rather than reporting the one it bound.
func freeLoopbackPort(t devtest.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	t.Require().NoError(err, "failed to reserve a loopback port")
	port := l.Addr().(*net.TCPAddr).Port
	t.Require().NoError(l.Close(), "failed to release the reserved loopback port")
	return port
}

// setUpSilhouette turns one already-running chain of a multi-chain runtime into a proof-carried
// chain: it writes the config the verifier reads, starts the verifier supernode, and connects the
// already-created ordinary op-batcher to the verifier.
//
// It runs LAST in the runtime, after the sequencer supernode and the batchers, for one reason that is
// not stylistic: the anchor is read from the chain itself over RPC, so the chain has to be answering
// before the verifier's config can be written.
func setUpSilhouette(
	t devtest.T,
	keys devkeys.Keys,
	chainKey string,
	l1Net *L1Network,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	runtimeChains map[string]*MultiChainNodeRuntime,
	depSet *depset.StaticConfigDependencySet,
	interopActivationTimestamp *uint64,
	jwtPath string,
	jwtSecret [32]byte,
	sharedSupernode *SuperNode,
) *SilhouetteRuntime {
	require := t.Require()
	logger := t.Logger().New("component", "silhouette")

	p := runtimeChains[chainKey]
	require.NotNilf(p, "silhouette chain %q is not in the runtime", chainKey)
	require.NotNil(p.CL, "the silhouette chain needs its LightCL sequencing node to read its anchor from")
	pChainID := p.Network.ChainID()

	rollupConfigHash, depSetHash := silhouetteBindings(t, p.Network.rollupCfg, depSet)
	submitterKey, err := keys.Secret(devkeys.BatcherRole.Key(pChainID.ToBig()))
	require.NoError(err, "need the silhouette chain's batcher key")
	submitterAddr := crypto.PubkeyToAddress(submitterKey.PublicKey)

	anchor := silhouetteAnchor(t, logger, p.CL.UserRPC(),
		p.Network.rollupCfg.Genesis.L2, p.Network.rollupCfg.Genesis.L2Time, p.Network.rollupCfg.Genesis.L1)

	cfg := silhouette.Config{
		L1ChainID:        eth.EvilChainIDToUInt64(l1Net.ChainID()),
		Submitter:        submitterAddr,
		Inbox:            p.Network.rollupCfg.BatchInboxAddress,
		RollupConfigHash: rollupConfigHash,
		DepSetHash:       depSetHash,
		// Proven history is re-derived from here on every restart, and P starts at its own genesis,
		// so its genesis L1 origin is the first L1 block that can carry anything about it.
		L1StartBlock: p.Network.rollupCfg.Genesis.L1.Number,
		Anchor:       anchor,
		// v1's proving system: the operator's L1 signature over the blob transaction IS the proof
		// (acceptance rule 1), and the proof slot must be empty. The wire, the blob transport, the L1
		// bindings, the import list and the transcode are all exercised for real — this is the shipped
		// v1 configuration, not a test shortcut. See TRUST-MODEL.md.
		ProofType: silhouette.ProofTypeAttested,
	}
	verifierRollup := silhouetteVerifierRollup(t, p.Network.rollupCfg)

	configDir := t.TempDirWithPrefix("silhouette-config")
	liveConfigDir := filepath.Join(configDir, "live")
	manifestPath := writeSilhouetteManifest(t, liveConfigDir, l1Net.genesis.Config, cfg, pChainID)
	// writeSilhouetteManifest takes cfg by value and fills this path in the file it writes. The
	// standalone magic EL consumes the same config directly, so give its copy the identical path.
	cfg.L1ChainConfigPath = filepath.Join(liveConfigDir, "l1-chain-config.json")
	magicELRPC, magicFacts := startSilhouetteEL(t, logger, sharedSupernode, verifierRollup, cfg,
		p.EL, jwtSecret)

	// Every ordinary chain gets its own execution client on the verifier side. The silhouette chain
	// gets none: that absence is the deliverable.
	chains := make([]silhouetteVerifierChain, 0, len(runtimeChains))
	verifierELs := make(map[string]L2ELNode, len(runtimeChains))
	for _, key := range sortedChainKeys(runtimeChains) {
		chain := runtimeChains[key]
		vc := silhouetteVerifierChain{key: key, net: chain.Network}
		if key != chainKey {
			vc.el = startL2ELForKey(t, chain.Network, jwtPath, jwtSecret, "silhouette-verifier",
				NewELNodeIdentity(0), ResolveMixedL2ELOpts(t)...)
			verifierELs[key] = vc.el
		} else {
			vc.engineRPC = magicELRPC
			vc.rollup = verifierRollup
		}
		chains = append(chains, vc)
	}

	verifier, routes, metricsURL, tryVerifier := startSilhouetteVerifier(
		t, l1Net, l1EL, l1CL, chains, pChainID, manifestPath, depSet,
		silhouetteInteropActivation(t, interopActivationTimestamp, anchor, p.Network.rollupCfg.BlockTime),
		jwtSecret)
	// Follow-source forkchoice carries only block references. For ordinary chains, the replacement
	// payload itself lives first in the verifier supernode's EL, so peer that EL with the LightCL
	// sequencer's EL exactly as the normal light-sequencer topology does. The silhouette chain needs
	// no such link: its magic EL builds the replacement directly in P's private sequencing EL.
	for _, key := range sortedChainKeys(runtimeChains) {
		verifierEL, ok := verifierELs[key]
		if !ok {
			continue
		}
		connectL2ELPeers(t, logger, verifierEL.UserRPC(), runtimeChains[key].EL.UserRPC())
	}
	attachSilhouetteDeniedChecker(t, logger, magicFacts, verifier.UserRPC(), pChainID)

	batcherControl := newProofBatchControl(t, p.Batcher)

	logger.Info("silhouette chain configured",
		"chain", pChainID, "submitter", submitterAddr, "inbox", cfg.Inbox,
		"anchor_block", anchor.BlockNumber, "anchor_output_root", anchor.OutputRoot,
		"rollup_config_hash", rollupConfigHash, "dep_set_hash", depSetHash,
		"manifest", manifestPath)

	return &SilhouetteRuntime{
		ChainKey:           chainKey,
		ChainID:            pChainID,
		Verifier:           verifier,
		VerifierRoutes:     routes,
		VerifierELs:        verifierELs,
		VerifierMetricsURL: metricsURL,
		MagicELRPC:         magicELRPC,
		Batcher:            batcherControl,
		ManifestPath:       manifestPath,
		Config:             cfg,
		configDir:          configDir,
		l1ChainConfig:      l1Net.genesis.Config,
		tryVerifier:        tryVerifier,
	}
}

// silhouetteVerifierRollup derives the public verifier's rollup config from the private producer's
// config. Chain identity, timing, portal and genesis SystemConfig stay identical. Forks are active
// from genesis because the magic EL intentionally does not expose enough header interior to
// reconstruct one-time activation gas from an earlier rendered block.
func silhouetteVerifierRollup(t devtest.T, producer *rollup.Config) *rollup.Config {
	require := t.Require()
	require.NotNil(producer)
	scalar := producer.Genesis.SystemConfig.Scalar
	eip1559Params := producer.Genesis.SystemConfig.EIP1559Params
	if eip1559Params == (eth.Bytes8{}) {
		require.NotNil(producer.ChainOpConfig,
			"producer rollup config needs chain-op defaults when genesis EIP-1559 params are zero")
		require.LessOrEqual(producer.ChainOpConfig.EIP1559Denominator, uint64(^uint32(0)))
		require.LessOrEqual(producer.ChainOpConfig.EIP1559Elasticity, uint64(^uint32(0)))
		binary.BigEndian.PutUint32(eip1559Params[:4], uint32(producer.ChainOpConfig.EIP1559Denominator))
		binary.BigEndian.PutUint32(eip1559Params[4:], uint32(producer.ChainOpConfig.EIP1559Elasticity))
	}
	cfg, err := silhouette.RollupConfigFor(silhouette.SilhouetteParams{
		L2ChainID:         producer.L2ChainID,
		L1ChainID:         producer.L1ChainID,
		L1Genesis:         producer.Genesis.L1,
		L2Genesis:         producer.Genesis.L2,
		L2Time:            producer.Genesis.L2Time,
		BlockTime:         producer.BlockTime,
		SeqWindowSize:     silhouette.DefaultSeqWindowSize,
		MaxSequencerDrift: producer.MaxSequencerDrift,
		DepositContract:   producer.DepositContractAddress,
		BatchInbox:        producer.BatchInboxAddress,
		SystemConfigProxy: producer.L1SystemConfigAddress,
		GasLimit:          producer.Genesis.SystemConfig.GasLimit,
		EIP1559Params:     eip1559Params,
		MinBaseFee:        producer.Genesis.SystemConfig.MinBaseFee,
		BatcherAddr:       producer.Genesis.SystemConfig.BatcherAddr,
		BaseFeeScalar:     binary.BigEndian.Uint32(scalar[28:32]),
		BlobBaseFeeScalar: binary.BigEndian.Uint32(scalar[24:28]),
	})
	require.NoError(err, "generate silhouette verifier rollup config")
	return cfg
}

// startSilhouetteEL starts the devstack equivalent of the op-silhouette-el binary. It has its own
// fact store and proof walker; only the L1 transports are shared with the harness process.
func startSilhouetteEL(t devtest.T, logger log.Logger, sn *SuperNode, rollupCfg *rollup.Config,
	cfg silhouette.Config, privateEL L2ELNode, jwtSecret [32]byte,
) (string, *silhouette.FactStore) {
	require := t.Require()
	require.NotNil(sn, "the magic EL needs the runtime's shared L1 clients")
	require.NotNil(sn.L1Client())
	require.NotNil(sn.BeaconClient())

	l1Chain, err := silhouette.L1ChainConfig(&cfg)
	require.NoError(err)
	verifier, err := cfg.NewVerifier()
	require.NoError(err)
	facts := &silhouette.FactStore{}
	source := silhouette.NewDataSource(logger.New("component", "magic-el-proof-source"), &cfg, rollupCfg,
		l1Chain, rollupCfg.Genesis.SystemConfig, sn.L1Client(), sn.BeaconClient(), verifier, facts)
	shim := silhouette.NewShim(logger.New("component", "magic-el"), rollupCfg, l1Chain,
		rollupCfg.Genesis.SystemConfig, sn.L1Client(), facts)
	replacementRPC, err := client.NewRPC(t.Ctx(), logger, privateEL.EngineRPC(),
		client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(jwtSecret))),
		client.WithCallTimeout(30*time.Second))
	require.NoError(err, "dial private replacement engine")
	t.Cleanup(replacementRPC.Close)
	replacementEngine, err := sources.NewEngineClient(replacementRPC, logger.New("component", "replacement-engine"),
		nil, sources.EngineClientDefaultConfig(rollupCfg))
	require.NoError(err, "create private replacement engine client")
	shim.SetReplacementBuilder(silhouette.NewEngineReplacementBuilder(replacementEngine))
	start := cfg.L1StartBlock
	if start == 0 {
		start = rollupCfg.Genesis.L1.Number
	}
	tracker := silhouette.NewProvenHeadTracker(logger.New("component", "magic-el-proof-walker"),
		source, sn.L1Client(), start, 100*time.Millisecond)
	server := shim.Standalone("127.0.0.1", 0)
	require.NoError(server.Start())
	t.Cleanup(func() { require.NoError(server.Stop()) })
	go tracker.Run(t.Ctx())
	rpcURL := "http://" + server.Endpoint()
	logger.Info("started standalone silhouette magic EL", "rpc", rpcURL, "chain", rollupCfg.L2ChainID)
	return rpcURL, facts
}

// attachSilhouetteDeniedChecker gives the standalone magic EL the verifier's durable denial view.
// It is attached after the verifier starts because the verifier needs the EL endpoint during its
// own startup. No proof submitter is running yet, so no replacement proof can race this attachment.
func attachSilhouetteDeniedChecker(t devtest.T, logger log.Logger, facts *silhouette.FactStore, supernodeRPC string, chainID eth.ChainID) {
	rpcCl, err := client.NewRPC(t.Ctx(), logger, supernodeRPC, client.WithLazyDial(), client.WithCallTimeout(10*time.Second))
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)
	supernodeClient := sources.NewSuperNodeClient(rpcCl)
	facts.SetDeniedChecker(func(number uint64, hash common.Hash) (bool, error) {
		ctx, cancel := context.WithTimeout(t.Ctx(), 10*time.Second)
		defer cancel()
		return supernodeClient.IsDenied(ctx, chainID, number, hash)
	})
}

// sortedChainKeys keeps chain iteration deterministic, so the verifier's chain order and its EL
// naming do not depend on map ordering.
func sortedChainKeys(chains map[string]*MultiChainNodeRuntime) []string {
	keys := make([]string, 0, len(chains))
	for key := range chains {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
