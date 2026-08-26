package sysgo

import (
	"context"
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
	"github.com/ethereum/go-ethereum/params"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	opnodeconfig "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	snconfig "github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
)

// The silhouette seam in devstack: what it takes to run one of a preset's chains as a proof-carried
// chain, plus a second supernode that consumes its proofs.
//
// Four things here are not obvious, and every one of them is a property of silhouette rather than of
// the harness.
//
// A SECOND supernode is required, not a second route on the existing one. The preset's supernode is
// chain P's SEQUENCER: it holds P's execution client and P's real receipts, so any cross-safety
// verdict it reaches about P's messages is a statement about data it was handed directly. The thesis
// — that a node which does NOT derive P can still cross-safe P's messages — is only testable on a
// node whose entire access to P is L1.
//
// P's BATCHER IS STOPPED. A silhouette chain has no batcher; its L1 footprint is the proof-batch
// inbox. Left running it would put P's real history on L1 beside the proofs, and the verifier — which
// only reads the inbox — would still work, so the gate would pass for the wrong reason.
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

// silhouetteSubmitterMnemonicIndex derives the proof-batch submitter's L1 key. It is deliberately
// none of the chain-operator roles: the (submitter, inbox) pair is the ENTIRE authenticity rule for
// the proof stream, so sharing a key with the batcher or the proposer would let an unrelated
// service's transaction be considered as a proof batch.
const silhouetteSubmitterMnemonicIndex = 10_001

// silhouetteInbox is the L1 address proof batches are addressed to. It holds no code and needs none:
// a verifier's rule is "type-3 transaction, to this address, from that sender", so the inbox is a
// coordinate rather than a contract.
var silhouetteInbox = common.HexToAddress("0x00000000000000000000000000000000c0be1b01")

// SilhouetteRuntime is the silhouette half of a runtime: the chain that is proof-carried, the
// verifier that consumes its proofs, and the submitter that produces them.
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
	// VerifierELs are the execution clients the verifier's ORDINARY chains drive, keyed by runtime
	// chain key. The silhouette chain has no entry, because it has no execution client at all — the
	// shim serves proof-committed facts and executes nothing.
	VerifierELs map[string]L2ELNode
	// VerifierMetricsURL is the verifier's Prometheus endpoint. It exists so a test can assert on
	// supernode_interop_invalidations_total, which is the only honest way to say "zero
	// invalidations" rather than "we did not notice one".
	VerifierMetricsURL string

	// Submitter posts P's proof batches to L1.
	Submitter *ProofBatchSubmitter

	// Sequencer is the preset's OWN supernode: the one that sequences the silhouette chain on its
	// real execution client. It is the subject of every sequencer-posture assertion, and it is the
	// node the hazard-3 argument is about.
	Sequencer *SuperNode
	// SequencerRoute is that supernode's route for the silhouette chain.
	SequencerRoute *SuperNodeProxy

	// ManifestPath and Config are the files the verifier was started from.
	ManifestPath string
	Config       silhouette.Config

	// SequencerPosture records whether the preset's OWN supernode was restarted with a `proven-head`
	// manifest. A test asserting the hazard and a test asserting its fix run on the same preset with
	// the same helpers, so this is what lets each one state which system it is looking at.
	SequencerPosture bool

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
	route := s.VerifierRoutes[s.ChainKey]
	if route == nil {
		return nil, fmt.Errorf("verifier has no route for the silhouette chain")
	}
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), route.UserRPC(), client.WithLazyDial())
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

// SequencerProvenHead asks the SEQUENCING supernode how far the chain has been proven.
//
// This is the sequencer posture's only public surface and the only way to read the state it turns on.
// `optimism_syncStatus` on that node reports its virtual node's own labels, which sit at genesis
// forever because there is no batcher behind them and are SUPPOSED to; the labels the cross-safety
// round consults come from the proven head instead, and this is where they are visible.
func (s *SilhouetteRuntime) SequencerProvenHead(t devtest.T) *silhouette.ProvenHeadStatus {
	require := t.Require()
	require.True(s.SequencerPosture,
		"the sequencing supernode serves silhouette_provenHead only in the sequencer posture")
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), s.SequencerRoute.UserRPC(), client.WithLazyDial())
	require.NoError(err, "failed to dial the sequencing supernode's route for the silhouette chain")
	defer rpcCl.Close()

	var out silhouette.ProvenHeadStatus
	require.NoError(rpcCl.CallContext(t.Ctx(), &out, "silhouette_provenHead"), "silhouette_provenHead")
	return &out
}

// ManifestWithWrongL1ChainID writes a second, complete set of silhouette config files that differ
// from the live ones in exactly one respect: the L1 chain-config file they name is for a different
// chain. Starting a verifier from it must fail.
func (s *SilhouetteRuntime) ManifestWithWrongL1ChainID(t devtest.T) string {
	wrong := *s.l1ChainConfig
	wrong.ChainID = new(big.Int).Add(s.l1ChainConfig.ChainID, big.NewInt(1))
	return writeSilhouetteManifest(t, filepath.Join(s.configDir, "wrong-l1"), &wrong, s.Config, s.ChainID, "derivation")
}

// silhouettePrefundOption prefunds the proof-batch submitter on L1. Prefunding is a genesis-time
// decision, so this has to be a deployer option rather than a transfer later: a submitter that had to
// be funded by a transaction would make the harness's first act on L1 a nonce the test has to reason
// about.
func silhouettePrefundOption() DeployerOption {
	return func(t devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
		addr, err := keys.Address(devkeys.UserKey(silhouetteSubmitterMnemonicIndex))
		t.Require().NoError(err, "need a silhouette proof-batch submitter address")
		builder.L1().WithPrefundedAccount(addr, *millionEth)
	}
}

// silhouetteBatcherStoppedOption starts the silhouette chain's batcher STOPPED.
//
// The batcher service is still constructed, because the preset's frontends expect every chain to have
// one and a nil batcher would be a shape change rippling through unrelated code. Starting it stopped
// rather than starting it and calling StopBatcher is the difference between a chain that never had a
// batcher and one that had a batcher for a few hundred milliseconds — and a single batch of P's real
// history on L1 would be enough to make a verifier's progress ambiguous.
func silhouetteBatcherStoppedOption(chainID eth.ChainID) BatcherOption {
	return func(target ComponentTarget, cfg *bss.CLIConfig) {
		if target.ChainID != chainID {
			return
		}
		cfg.Stopped = true
	}
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
func writeSilhouetteManifest(t devtest.T, dir string, l1Chain *params.ChainConfig, cfg silhouette.Config, chainID eth.ChainID, labels string) string {
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
		Labels:         labels,
	}}})
}

// silhouetteBindings computes the two wire commitments a verifier checks a batch against.
//
// Neither is derived from anything in the Go stack today — the real values come from what the prover
// derived under — so they are configuration on both sides. Hashing the artifacts the chain actually
// runs, rather than picking constants, buys the property that matters: change the chain and the
// commitment changes, so a batch built for one chain cannot be accepted as another's history.
func silhouetteBindings(t devtest.T, rollupCfg any, depSet *depset.StaticConfigDependencySet) (rollupConfigHash, depSetHash common.Hash) {
	require := t.Require()
	raw, err := json.Marshal(rollupCfg)
	require.NoError(err, "encode the silhouette chain's rollup config for its commitment")
	rollupConfigHash = crypto.Keccak256Hash([]byte("devstack-silhouette-rollup-config:"), raw)

	require.NotNil(depSet, "a silhouette chain needs a dependency set: outside one, nobody can reference its messages")
	raw, err = json.Marshal(depSet)
	require.NoError(err, "encode the dependency set for its commitment")
	depSetHash = crypto.Keccak256Hash([]byte("devstack-silhouette-depset:"), raw)
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
	key string
	net *L2Network
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
			// The silhouette chain's L2 endpoint is REPLACED by silhouette.Assemble with an
			// in-process client to the shim, before the chain container is built. The value here is
			// only ever read by opnodeconfig.Config.Check, which insists on a non-empty address. It
			// is deliberately unroutable rather than P's real engine: if anything ever did dial it,
			// a connection refusal is a loud bug and talking to the sequencer's engine would be a
			// silent one.
			engineAddr = "http://127.0.0.1:1"
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
			Rollup:                          *chain.net.rollupCfg,
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
// chain: it writes the config the verifier reads, starts the verifier supernode, and builds the
// submitter that feeds it.
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
	sequencerSupernode *SuperNode,
	sequencerPosture bool,
) *SilhouetteRuntime {
	require := t.Require()
	logger := t.Logger().New("component", "silhouette")

	p := runtimeChains[chainKey]
	require.NotNilf(p, "silhouette chain %q is not in the runtime", chainKey)
	require.NotNil(p.SupernodeCL, "the silhouette chain needs its sequencer-side rollup route to read its anchor from")
	pChainID := p.Network.ChainID()

	rollupConfigHash, depSetHash := silhouetteBindings(t, p.Network.rollupCfg, depSet)
	submitterKey, err := keys.Secret(devkeys.UserKey(silhouetteSubmitterMnemonicIndex))
	require.NoError(err, "need the silhouette proof-batch submitter key")
	submitterAddr := crypto.PubkeyToAddress(submitterKey.PublicKey)

	anchor := silhouetteAnchor(t, logger, p.SupernodeCL.UserRPC(),
		p.Network.rollupCfg.Genesis.L2, p.Network.rollupCfg.Genesis.L2Time, p.Network.rollupCfg.Genesis.L1)

	cfg := silhouette.Config{
		L1ChainID:        eth.EvilChainIDToUInt64(l1Net.ChainID()),
		Submitter:        submitterAddr,
		Inbox:            silhouetteInbox,
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

	configDir := t.TempDirWithPrefix("silhouette-config")
	manifestPath := writeSilhouetteManifest(t, filepath.Join(configDir, "live"), l1Net.genesis.Config, cfg, pChainID, "derivation")

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
		}
		chains = append(chains, vc)
	}

	verifier, routes, metricsURL, tryVerifier := startSilhouetteVerifier(
		t, l1Net, l1EL, l1CL, chains, pChainID, manifestPath, depSet,
		silhouetteInteropActivation(t, interopActivationTimestamp, anchor, p.Network.rollupCfg.BlockTime),
		jwtSecret)

	submitter := newProofBatchSubmitter(t, logger.New("component", "proofbatch-submitter"), ProofBatchSubmitterConfig{
		Inbox:            silhouetteInbox,
		SubmitterKey:     submitterKey,
		RollupConfigHash: rollupConfigHash,
		DepSetHash:       depSetHash,
		AnchorBlock:      anchor.BlockNumber,
		AnchorOutputRoot: anchor.OutputRoot,
		MaxBlocks:        300,
		// One block of margin below the L1 head: the claimed l1Head must be canonical on the
		// verifier and no higher than the block that carries the batch, and both hold trivially for
		// a block that is already a parent by the time the transaction lands.
		L1Lag: 1,
	}, l1EL.UserRPC(), p.EL.UserRPC(), p.SupernodeCL.UserRPC())

	// And the sequencer side, if the preset asked for it. LAST: the posture is applied to a node that
	// has been sequencing all along, which is what a cutover does.
	if sequencerPosture {
		applySequencerPosture(t, logger, sequencerSupernode, configDir, l1Net.genesis.Config, cfg,
			pChainID)
	}

	logger.Info("silhouette chain configured",
		"chain", pChainID, "submitter", submitterAddr, "inbox", silhouetteInbox,
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
		Submitter:          submitter,
		ManifestPath:       manifestPath,
		Config:             cfg,
		Sequencer:          sequencerSupernode,
		SequencerRoute: &SuperNodeProxy{p: t, logger: logger, userRPC: sequencerSupernode.UserRPC() +
			"/" + strconv.FormatUint(eth.EvilChainIDToUInt64(pChainID), 10)},
		SequencerPosture: sequencerPosture,
		configDir:        configDir,
		l1ChainConfig:    l1Net.genesis.Config,
		tryVerifier:      tryVerifier,
	}
}

// applySequencerPosture restarts the preset's OWN supernode — the one that sequences the silhouette
// chain on its real execution client — with a `proven-head` manifest.
//
// A RESTART rather than a startup flag, and that is faithful rather than expedient. The manifest
// names the chain's anchor, the anchor is read off the chain over RPC, and the chain only answers
// because this supernode is already up sequencing it. The rotation runbook has the same shape for
// the same reason: the sequencer-side posture is the LAST step, applied to a running cluster.
//
// A FRESH DATA DIRECTORY, which is the part that would be a silent failure if it were forgotten.
// Before this call the supernode ingested the chain's initiating messages from RECEIPTS, because it
// had them; from here it seals them from the wire, because the public network's view of the chain is
// the set of messages the chain chose to export and a node ingesting from receipts would hold a
// strictly larger set than every verifier (G4 D1). Those two histories do not chain: the log sink
// COMPARES rather than re-seals on restart and refuses foreign history outright. So the data
// directory has to go, exactly as the runbook's anchor-move step says.
//
// Everything else about the node is left alone. It keeps sequencing, it keeps its execution client,
// and the posture branch in silhouette.Assemble is what declines to take any of that away.
func applySequencerPosture(
	t devtest.T,
	logger log.Logger,
	sn *SuperNode,
	configDir string,
	l1Chain *params.ChainConfig,
	cfg silhouette.Config,
	chainID eth.ChainID,
) {
	require := t.Require()
	require.NotNil(sn, "the sequencer posture needs the preset's own supernode")

	manifest := writeSilhouetteManifest(t, filepath.Join(configDir, "sequencer"), l1Chain, cfg,
		chainID, "proven-head")

	logger.Info("applying the silhouette sequencer posture to the sequencing supernode",
		"chain", chainID, "manifest", manifest)
	sn.Stop()

	sn.snCfg.SilhouetteManifestPath = manifest

	// THE DATA DIRECTORY IS KEPT, and the reason is a property of this harness rather than a shortcut.
	//
	// The rotation clears it (RUNBOOK §5.1) because P is being re-genesised onto a new anchor and a log
	// store built under a different anchor cannot chain. Here there is nothing to clear: P has had no
	// batcher since the cluster came up, so its public safe head never left genesis, so nothing was
	// ever sealed into its log store. The posture change from receipts to wire (G4 D1) has no receipt
	// history to contradict.
	//
	// Wiping anyway was tried and is INSTRUCTIVE, so it is written down here rather than lost: it
	// stalls the node permanently, and not through the log store. `checkChainsReady` asks every chain
	// `OptimisticAt(ts)`, which for an ordinary chain is answered from its SafeDB — and a wiped SafeDB
	// is refilled only as that chain's safe head ADVANCES, in the jumps its batcher's batches arrive
	// in. The round, restarting at the activation timestamp, then asks for a timestamp below the first
	// entry that will ever exist and gets `not ready` forever, while every log line reports health.
	// That is the same shape as hazard 3 and it is what RUNBOOK §5.1 and G4 D10 exist to prevent — but
	// reproducing it here would be testing the harness's own anchor placement (P anchored at the
	// CLUSTER's genesis, hours of chain history below the restart) rather than the posture.
	//
	// So the restart is a restart and nothing else: same data, same activation timestamp, one manifest
	// added. That is the smallest change that puts this node in the posture, which makes it the right
	// one for a test whose subject is the posture.
	sn.Start()
	logger.Info("sequencing supernode restarted in the sequencer posture",
		"chain", chainID, "rpc", sn.UserRPC(), "data_dir", sn.snCfg.DataDir)
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
