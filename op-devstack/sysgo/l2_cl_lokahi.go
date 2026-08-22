package sysgo

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

// lokahiAdminBoundMessage is the log line lokahi emits once its process-wide admin RPC is
// listening. It carries the bound address, so the launch below can ask for port 0 and still
// know where to find it — the same handshake startMixedKonaNode performs on kona-node's
// "RPC server bound to address".
//
// Only the admin RPC is discovered this way. A chain's own RPC server is bound inside kona,
// which logs the address without saying which chain it belongs to, so the chains are given
// concrete ports here instead and lokahi's admin RPC is asked to confirm them.
const lokahiAdminBoundMessage = "Admin RPC server bound to address"

// lokahiSequencerL1Confs is how many L1 blocks a sequencing chain keeps away from the L1 head
// when it picks an L1 origin. It matches the SequencerConfDepth the in-process Go supernode
// gives its virtual sequencers, so the two implementations sequence the same distance behind
// the L1 head; kona-node's own default of 4 is meant for a production L1.
const lokahiSequencerL1Confs = 2

// lokahiSupernodeChain is one chain the supernode must host: the L2 network whose rollup
// config drives derivation, and the EL that chain's node advances.
type lokahiSupernodeChain struct {
	net *L2Network
	el  L2ELNode
}

// lokahiSupernodeConfig is everything an out-of-process supernode needs in order to stand
// in for the in-process Go op-supernode. It deliberately mirrors the inputs
// startTwoL2SharedSupernode and startSingleChainSharedSupernode turn into an
// snconfig.CLIConfig plus one op-node config per chain, so the launch path below has the
// same information available and the two implementations cannot drift apart in what they
// are given.
type lokahiSupernodeConfig struct {
	// l1Net supplies the L1 chain config the supernode validates L1 blocks against.
	l1Net *L1Network
	// l1ELRPC and l1BeaconAddr are the L1 data sources shared by every hosted chain.
	l1ELRPC      string
	l1BeaconAddr string
	// chains are the L2 chains to host, in the order the caller wants them routed.
	chains []lokahiSupernodeChain
	// depSet is the interop dependency set, or nil when interop is not configured.
	depSet *depset.StaticConfigDependencySet
	// interopActivationTimestamp enables the interop activity at the given timestamp; nil
	// leaves interop off.
	interopActivationTimestamp *uint64
	// sequencerEnabled runs the hosted nodes as sequencers rather than verifiers.
	sequencerEnabled bool
}

// LokahiSupernode is a running lokahi process hosting one or more chains.
//
// Like the Go op-supernode, lokahi serves every chain from one RPC under a /<chainID> route,
// keeping the method names a single-chain node has. That one socket is fronted by a proxy so
// callers keep a stable URL across a restart, and ChainCL hands out an L2CLNode per chain
// whose address is that proxy plus the chain's route.
type LokahiSupernode struct {
	mu sync.Mutex

	p        devtest.T
	logger   log.Logger
	execPath string
	args     []string
	// dataDir is the on-disk state directory named by the generated configuration. The
	// configuration itself lives beside it, not inside it, so wiping this discards the
	// node's state without taking away what it was configured with.
	dataDir string

	// adminRPC is the address lokahi actually bound its process-wide RPC to, discovered from
	// the startup log line. It moves on every restart, because the configuration asks for
	// port 0 — only the bound address is reported, so only a port lokahi chose can be
	// discovered.
	adminRPC string
	// adminProxy fronts adminRPC, and adminUserRPC is its stable address.
	//
	// The indirection is what makes the RPC survive a restart. Without it every holder of this
	// address would be dialling a dead port after a Stop/Start: the preset frontend builds its
	// query-API client once, at construction, and the acceptance tests that wipe the data
	// directory or take the execution layers away mid-test go on to read through it.
	//
	// One proxy for the whole process, because there is one socket to front: the chains are
	// routes on it, so a chain's stable address is this one plus /<chainID>. It is also where
	// the supernode query API answers, at the root: supernode_syncStatus and
	// superroot_atTimestamp are statements about the whole chain set rather than about a chain.
	adminProxy   *tcpproxy.Proxy
	adminUserRPC string
	// chains keeps the hosted chains in configuration order.
	chains []*lokahiChainCL

	// queryAPI and interopTestAPI are the clients for the two surfaces the process-wide RPC
	// serves, each built on first use. One client for the life of the supernode is enough
	// because both dial adminProxy, whose address does not move; before that proxy existed
	// they had to be rebuilt whenever lokahi came back on a new port.
	queryAPI       *sources.SuperNodeClient
	interopTestAPI *sources.SupernodeInteropTestClient

	sub *SubProcess
}

// LokahiSupernode serves the supernode query API, which is the read-only surface every
// consumer of a supernode uses: op-proposer and op-challenger read superroot_atTimestamp,
// and the devstack DSL's SuperRootSource reads both methods through this interface.
var _ interface {
	QueryAPI() apis.SupernodeQueryAPI
} = (*LokahiSupernode)(nil)

// lokahiChainCL addresses one chain of a running lokahi supernode.
//
// It is the counterpart of SuperNodeProxy: the process is shared, so starting and stopping
// one chain on its own is not a thing lokahi offers, and the lifecycle methods are no-ops
// that defer to the supernode as a whole.
type lokahiChainCL struct {
	chainID eth.ChainID
	// userRPC is the supernode's stable address with this chain's route appended, so it moves
	// only when the supernode's proxy address does — which is never.
	userRPC string
}

var _ L2CLNode = (*lokahiChainCL)(nil)

func (c *lokahiChainCL) Start()          {}
func (c *lokahiChainCL) Stop()           {}
func (c *lokahiChainCL) UserRPC() string { return c.userRPC }

// ChainID reports which chain this endpoint answers for.
func (c *lokahiChainCL) ChainID() eth.ChainID { return c.chainID }

// AdminRPC is the process-wide admin/test RPC of the supernode: one stable address across
// restarts, whatever port the process is on behind it.
//
// Read without the mutex, as QueryRPC below is, because the address is written exactly once —
// startLokahiSupernode starts the process before it hands the supernode out, and the proxy it
// names is created on that first start and never replaced.
func (n *LokahiSupernode) AdminRPC() string { return n.adminUserRPC }

// QueryRPC is the endpoint the supernode query API answers on.
//
// The same socket as AdminRPC, named separately because they are different contracts: the
// admin namespace is lokahi's own and may change with lokahi, while the query API is the
// wire op-supernode defines and its consumers depend on. A preset wiring lokahi in as a
// stack.Supernode hands this to the frontend, not AdminRPC.
func (n *LokahiSupernode) QueryRPC() string { return n.adminUserRPC }

// QueryAPI returns the supernode query API of the running process.
//
// The client is op-service/sources.SuperNodeClient — the same one the in-process Go
// supernode is read through — so nothing about these two RPCs is reimplemented for lokahi
// on the Go side. Whether the answers match is a question about what lokahi serves, which
// is where it belongs, rather than about two Go clients agreeing.
// It is safe to hold across a Stop/Start: the address it dials is the process-wide proxy's,
// which does not move even though the process behind it binds a new port each time.
func (n *LokahiSupernode) QueryAPI() apis.SupernodeQueryAPI {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.p.Require().NotEmpty(n.adminUserRPC, "lokahi has no query RPC address yet")
	if n.queryAPI == nil {
		rpcCl, err := client.NewRPC(n.p.Ctx(), n.logger, n.adminUserRPC, client.WithLazyDial())
		n.p.Require().NoError(err, "dial the lokahi supernode query API")
		n.queryAPI = sources.NewSuperNodeClient(rpcCl)
		n.p.Cleanup(n.queryAPI.Close)
	}
	return n.queryAPI
}

// InteropTestAPI returns the test-control surface for the interop verifier: pause, resume,
// status and one chain's sealed range.
//
// Nil when the process is not running. The DSL treats that as "supernode stopped or interop
// disabled" and says so, which is what a caller driving a stopped supernode should hear —
// whereas a client dialling a dead port would surface as a timeout naming nothing.
//
// The four methods are served by lokahi on its process-wide RPC (rust/lokahi/src/interop/
// test_api.rs). Interop being off is not distinguished here: lokahi answers those calls with an
// error saying it has no verifier, which is the honest answer and reaches the test as one.
//
// Reached through the process-wide proxy, so the returned surface stays valid across a
// Stop/Start — unlike the in-process supernode's, which is bound to the instance. The DSL
// re-fetches it per operation either way.
func (n *LokahiSupernode) InteropTestAPI() apis.SupernodeInteropTestAPI {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		return nil
	}
	if n.interopTestAPI == nil {
		rpcCl, err := client.NewRPC(n.p.Ctx(), n.logger, n.adminUserRPC, client.WithLazyDial())
		n.p.Require().NoError(err, "dial the lokahi interop test-control API")
		n.interopTestAPI = sources.NewSupernodeInteropTestClient(rpcCl)
		n.p.Cleanup(n.interopTestAPI.Close)
	}
	return n.interopTestAPI
}

// ChainCL returns the L2CLNode addressing the chain at index i, in configuration order.
func (n *LokahiSupernode) ChainCL(i int) L2CLNode { return n.chains[i] }

// startLokahiSupernode runs lokahi as the shared multi-chain consensus layer.
//
// This is the out-of-process counterpart of startMixedKonaNode (mixed_runtime.go): the
// binary is built or located through rustbin, its logs are piped into the test logger, and
// the process is waited on until it says it is listening rather than being polled blindly.
// What differs is the configuration: with N chains a flag per chain per setting is not a
// usable interface, so lokahi is handed a file, which this function renders.
func startLokahiSupernode(t devtest.T, cfg lokahiSupernodeConfig) *LokahiSupernode {
	require := t.Require()
	require.NotEmpty(cfg.chains, "a supernode hosts at least one chain")

	dir := t.TempDirWithPrefix("l2-cl-lokahi")
	logger := t.Logger().New("component", "lokahi")

	// Every listener the process needs is reserved at once and only then released:
	// allocating them one at a time would hand out the same port twice, because each is
	// free again before the next is asked for. Two per chain, both P2P: a chain has no RPC
	// port of its own any more, it is a route on the supernode's one socket.
	ports := reservePorts(t, 2*len(cfg.chains))

	chains := make([]*lokahiChainCL, len(cfg.chains))
	entries := make([]string, len(cfg.chains))
	for i, chain := range cfg.chains {
		tcpPort, udpPort := ports[2*i], ports[2*i+1]
		chains[i] = &lokahiChainCL{chainID: chain.net.ChainID()}
		entries[i] = lokahiChainEntry(t, dir, chain, cfg, tcpPort, udpPort)
	}

	configPath := filepath.Join(dir, "lokahi.toml")
	require.NoError(os.WriteFile(configPath, []byte(lokahiConfigFile(t, dir, cfg, entries)), 0o640),
		"must write the lokahi configuration")

	execPath, err := rustbin.Spec{
		SrcDir:  "rust/lokahi",
		Package: "lokahi",
		Binary:  "lokahi",
	}.EnsureExists(t.Ctx(), t.Logger())
	require.NoError(err, "prepare lokahi binary")
	require.NotEmpty(execPath, "lokahi binary path resolved")

	n := &LokahiSupernode{
		p:        t,
		logger:   logger,
		execPath: execPath,
		args:     []string{"node", "--config", configPath},
		dataDir:  lokahiDataDir(dir),
		chains:   chains,
	}

	logger.Info("Starting lokahi", "chains", len(chains), "config", configPath)
	n.Start()
	t.Cleanup(n.Stop)

	// The chain set lokahi resolved, asked of lokahi itself. The file above is generated,
	// so this is what turns a mistake in generating it into a failure here rather than into
	// a chain that quietly never started.
	n.requireHostedChains(cfg)

	for _, chain := range n.chains {
		waitForSupernodeRoute(t, logger.New("chain_id", chain.chainID.String()), chain.userRPC)
	}
	logger.Info("lokahi is up", "admin", n.adminUserRPC, "admin_upstream", n.adminRPC)

	return n
}

// Start launches the process and waits for its admin RPC to be listening.
func (n *LokahiSupernode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.startLocked()
}

// startLocked launches the process, waits for its RPC to be listening, and repoints the proxy at
// the port it bound. Caller must hold n.mu.
func (n *LokahiSupernode) startLocked() {
	if n.sub != nil {
		n.logger.Warn("lokahi already started")
		return
	}

	// One proxy for the whole process: created once and repointed on a restart, so every
	// holder of the supernode's address keeps one URL for the life of the test even though the
	// process binds a new port each time.
	//
	// The chains ride on it. A chain's address is this proxy plus its route, so there is one
	// proxy to keep pointed at a live port instead of one per chain — and every chain's
	// endpoint shares a host and port, which is what a caller of a supernode expects and what
	// TestTwoChainProgress asserts.
	if n.adminProxy == nil {
		n.adminProxy = tcpproxy.New(n.logger.New("proxy", "lokahi-admin"))
		n.p.Require().NoError(n.adminProxy.Start(), "lokahi admin proxy failed to start")
		n.p.Cleanup(func() { _ = n.adminProxy.Close() })
		n.adminUserRPC = "http://" + n.adminProxy.Addr()
		for _, chain := range n.chains {
			chain.userRPC = n.adminUserRPC + "/" + chain.chainID.String()
		}
	}

	adminRPCChan := make(chan string, 1)
	onLogEntry := func(e logpipe.LogEntry) {
		if e.LogMessage() != lokahiAdminBoundMessage {
			return
		}
		addr, ok := e.FieldValue("addr").(string)
		if !ok {
			return
		}
		select {
		case adminRPCChan <- "http://" + addr:
		default:
		}
	}

	logOut := logpipe.ToLoggerWithMinLevel(n.logger.New("src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(n.logger.New("src", "stderr"), log.LevelWarn)
	stdOut := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
	})
	stdErr := logpipe.LogCallback(func(line []byte) {
		logErr(logpipe.ParseRustStructuredLogs(line))
	})

	n.sub = NewSubProcess(n.p, stdOut, stdErr)
	n.p.Require().NoError(n.sub.Start(n.execPath, n.args, lokahiEnv()), "must start lokahi")

	// Fail fast if the process exits first — a chain that cannot be composed is reported by
	// lokahi as a startup error, and blocking on the context until the test times out would
	// hide it.
	select {
	case n.adminRPC = <-adminRPCChan:
	case <-n.sub.Exited():
		select {
		case n.adminRPC = <-adminRPCChan:
		default:
			n.p.Require().FailNow("lokahi exited before its admin RPC became ready")
		}
	case <-n.p.Ctx().Done():
		n.p.Require().NoError(n.p.Ctx().Err(), "need the lokahi admin RPC")
	}

	// One upstream to repoint: the chains are routes behind it, so they follow automatically.
	n.adminProxy.SetUpstream(strings.TrimPrefix(n.adminRPC, "http://"))
}

// Stop terminates the process, leaving the proxy in place so a later Start can repoint it.
func (n *LokahiSupernode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.stopLocked()
}

// stopLocked terminates the process, leaving the proxy in place so a later startLocked can
// repoint it. Caller must hold n.mu.
func (n *LokahiSupernode) stopLocked() {
	if n.sub == nil {
		n.logger.Warn("lokahi already stopped")
		return
	}
	n.clearProxyUpstream()
	n.p.Require().NoError(n.sub.Stop(true), "must stop lokahi")
	n.sub = nil
}

// clearProxyUpstream points the proxy at nothing, so a caller reaching the supernode while it is
// down is refused rather than left hanging on a connection to a port the process no longer holds.
// Clearing the one proxy covers the chains too, since their addresses are routes on it. Caller
// must hold n.mu.
func (n *LokahiSupernode) clearProxyUpstream() {
	if n.adminProxy != nil {
		n.adminProxy.ClearUpstream()
	}
}

func (n *LokahiSupernode) Running() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.sub != nil
}

func (n *LokahiSupernode) StartControlled(ctx context.Context) error {
	return runControlStart(ctx, n.Running, n.Start)
}

func (n *LokahiSupernode) StopControlled(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		return nil
	}
	n.clearProxyUpstream()
	if err := n.sub.StopControlled(ctx, controlledInterruptWait, controlledKillWait); err != nil {
		return err
	}
	n.sub = nil
	return nil
}

// RestartWithFreshDataDir stops the lokahi process, deletes its on-disk data directory, and
// starts a fresh process against the same chain containers, the same generated configuration,
// and the same externally-visible per-chain RPC addresses. Test-only.
//
// The out-of-process counterpart of (*SuperNode).RestartWithFreshDataDir, with the same
// semantics: what is discarded is the node's own state, not what it was configured with. The
// Every proxy survives, so nothing wired to this supernode — a chain's socket or the
// process-wide RPC — changes address across the restart, even though the process binds new
// ports for the latter and startLocked has to rediscover it from the startup log.
func (n *LokahiSupernode) RestartWithFreshDataDir() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		return errSupernodeNotRunning
	}
	if n.dataDir == "" {
		return errors.New("sysgo: RestartWithFreshDataDir requires a configured lokahi data dir")
	}
	n.logger.Info("restarting lokahi with fresh data dir", "data_dir", n.dataDir)
	n.stopLocked()
	if err := os.RemoveAll(n.dataDir); err != nil {
		return fmt.Errorf("sysgo: wipe lokahi data dir %s: %w", n.dataDir, err)
	}
	// 0o750, matching the 0o640 the config files beside it use: the node is the only reader.
	if err := os.MkdirAll(n.dataDir, 0o750); err != nil {
		return fmt.Errorf("sysgo: recreate lokahi data dir %s: %w", n.dataDir, err)
	}
	n.startLocked()

	// startLocked returns as soon as the admin RPC is listening, and on lokahi that is not the
	// same as being usable: each chain's RPC server is bound inside kona, after the admin RPC.
	// The initial launch waits for the per-chain routes for that reason, so a restart that
	// claims to be finished has to wait for them too, or the next call a test makes lands on a
	// port with nothing behind it yet.
	for _, chain := range n.chains {
		waitForSupernodeRoute(n.p, n.logger.New("chain_id", chain.chainID.String()), chain.userRPC)
	}
	return nil
}

// requireHostedChains asserts that lokahi hosts exactly the chains it was configured with, each
// under the route its chain id names.
func (n *LokahiSupernode) requireHostedChains(cfg lokahiSupernodeConfig) {
	require := n.p.Require()

	rpcCl, err := client.NewRPC(n.p.Ctx(), n.logger, n.adminUserRPC, client.WithLazyDial())
	require.NoError(err, "dial the lokahi admin RPC")
	defer rpcCl.Close()

	var hosted []struct {
		ChainID uint64 `json:"chainId"`
		RPCPath string `json:"rpcPath"`
	}
	require.NoError(rpcCl.CallContext(n.p.Ctx(), &hosted, "lokahi_chains"), "call lokahi_chains")
	require.Len(hosted, len(cfg.chains), "lokahi hosts a different number of chains than configured")

	for i, chain := range n.chains {
		require.Equal(eth.EvilChainIDToUInt64(chain.chainID), hosted[i].ChainID,
			"lokahi hosts chain %s at index %d", chain.chainID, i)
		require.Equal("/"+chain.chainID.String(), hosted[i].RPCPath,
			"lokahi serves chain %s under another route", chain.chainID)
	}
}

// lokahiDataDir is the on-disk state directory the generated configuration names, given the
// directory that configuration was generated into. It is a subdirectory rather than the
// generated directory itself so that RestartWithFreshDataDir can discard the state without
// also discarding the configuration files that sit alongside it.
func lokahiDataDir(dir string) string { return filepath.Join(dir, "data") }

// lokahiConfigFile renders the global layer of the configuration plus the chain entries.
//
// The RPC asks for port 0 and is discovered from the startup log, which is the only address that
// has to be discovered: the chains are routes on it, so knowing it is knowing where they all are.
func lokahiConfigFile(t devtest.T, dir string, cfg lokahiSupernodeConfig, entries []string) string {
	l1CfgPath := filepath.Join(dir, "l1-chain-config.json")
	l1CfgData, err := json.Marshal(cfg.l1Net.genesis.Config)
	t.Require().NoError(err, "must marshal the l1 chain config")
	t.Require().NoError(os.WriteFile(l1CfgPath, l1CfgData, 0o640), "must write the l1 chain config")

	var b strings.Builder
	fmt.Fprintf(&b, "[l1]\neth-rpc = %q\nbeacon = %q\nchain-config = %q\n\n",
		cfg.l1ELRPC, cfg.l1BeaconAddr, l1CfgPath)
	// Loopback and port 0: the harness reads the port back out of the startup log.
	fmt.Fprintf(&b, "[admin]\nrpc-addr = \"127.0.0.1\"\nrpc-port = 0\n\n")
	// The activation the preset computed, passed on rather than rederived. lokahi normally reads
	// it from each chain's Lagoon time, which is where the message rules read it too; the
	// simple-interop presets hand op-supernode a timestamp and write rollup configs with no
	// Lagoon at all, so there is nothing for lokahi to read and it has to be told. lokahi still
	// refuses a value that disagrees with a fork a chain does schedule.
	if cfg.interopActivationTimestamp != nil {
		fmt.Fprintf(&b, "[interop]\nactivation-timestamp = %d\n\n", *cfg.interopActivationTimestamp)
	}
	// Acceptance tests drive a node through its admin API, which kona only registers when
	// admin is enabled; op-node's devstack node enables it too. The experimental opstack
	// namespace mirrors op-supernode's virtual nodes, which always run with
	// ExperimentalOPStackAPI (multichain_supernode_runtime.go): the test sequencer drives
	// block building through it on each chain's route.
	fmt.Fprintf(&b, "[defaults]\ndatadir = %q\nmode = \"validator\"\n"+
		"rpc-enable-admin = true\nexperimental-opstack-api = true\np2p-listen-ip = \"127.0.0.1\"\n\n",
		lokahiDataDir(dir))
	b.WriteString(strings.Join(entries, "\n"))
	return b.String()
}

// lokahiChainEntry renders one [[chains]] entry, writing out the files it names.
func lokahiChainEntry(
	t devtest.T,
	dir string,
	chain lokahiSupernodeChain,
	cfg lokahiSupernodeConfig,
	tcpPort, udpPort int,
) string {
	require := t.Require()
	chainID := eth.EvilChainIDToUInt64(chain.net.ChainID())

	rollupCfgPath := filepath.Join(dir, fmt.Sprintf("rollup-%d.json", chainID))
	rollupCfgData, err := json.Marshal(chain.net.rollupCfg)
	require.NoError(err, "must marshal the rollup config of chain %d", chainID)
	require.NoError(os.WriteFile(rollupCfgPath, rollupCfgData, 0o640),
		"must write the rollup config of chain %d", chainID)

	var b strings.Builder
	fmt.Fprintf(&b, "[[chains]]\nl2-chain-id = %d\nrollup-config = %q\n", chainID, rollupCfgPath)
	// kona speaks the engine API over HTTP; the devstack EL advertises a websocket URL for it.
	fmt.Fprintf(&b, "engine-rpc = %q\njwt-secret = %q\n",
		strings.ReplaceAll(chain.el.EngineRPC(), "ws://", "http://"), chain.el.JWTPath())
	fmt.Fprintf(&b, "p2p-tcp-port = %d\np2p-udp-port = %d\n", tcpPort, udpPort)
	// Stated rather than read from L1. kona-node resolves the unsafe block signer from the
	// chain's SystemConfig contract when it is not given one; lokahi's configuration has no
	// such path, and a devnet chain is not in the superchain registry either, so the devstack
	// has to supply the address the sequencer's P2P key signs with.
	signerKey := devkeys.SequencerP2PRole.Key(chain.net.ChainID().ToBig())
	signer, err := chain.net.keys.Address(signerKey)
	require.NoError(err, "need the sequencer p2p address of chain %d", chainID)
	fmt.Fprintf(&b, "unsafe-block-signer = %q\n", signer.Hex())

	// A sequencing chain signs the blocks it gossips with the same key, handed over as a file
	// so the configuration does not carry it. `[defaults]` makes every chain a validator, so
	// this entry is what turns one on. The file has to be written here rather than left for
	// lokahi to generate: lokahi would generate some other key, and the address printed above
	// as `unsafe-block-signer` is this one, so the chain's peers would drop every block it
	// signed.
	if cfg.sequencerEnabled {
		secret, err := chain.net.keys.Secret(signerKey)
		require.NoError(err, "need the sequencer p2p key of chain %d", chainID)
		keyPath := filepath.Join(dir, fmt.Sprintf("sequencer-key-%d.hex", chainID))
		require.NoError(os.WriteFile(keyPath, []byte(hex.EncodeToString(crypto.FromECDSA(secret))), 0o600),
			"must write the sequencer key of chain %d", chainID)
		fmt.Fprintf(&b, "mode = \"sequencer\"\nsequencer-key-path = %q\n", keyPath)
		// The L1 confirmation distance the Go op-supernode gives its virtual sequencers
		// (SequencerConfDepth), rather than kona-node's production default of 4: a devnet L1
		// has few blocks to stay behind.
		fmt.Fprintf(&b, "sequencer-l1-confs = %d\n", lokahiSequencerL1Confs)
	}

	if cfg.depSet != nil {
		depSetPath := filepath.Join(dir, fmt.Sprintf("interop-depset-%d.json", chainID))
		depSetData, err := json.Marshal(cfg.depSet)
		require.NoError(err, "must marshal the interop dependency set")
		require.NoError(os.WriteFile(depSetPath, depSetData, 0o640),
			"must write the interop dependency set")
		fmt.Fprintf(&b, "interop-dependency-set = %q\n", depSetPath)
	}

	return b.String()
}

// lokahiEnv is the environment the process runs with. Structured logs on stdout are what
// the startup handshake parses, and what makes the child's log lines readable in the test
// logger rather than arriving as pre-coloured text.
func lokahiEnv() []string {
	return []string{
		propagateEnvVarOrDefault("KONA_LOG_LEVEL", "3"),
		propagateEnvVarOrDefault("KONA_LOG_STDOUT_FORMAT", "json"),
	}
}

// reservePorts returns n ports nothing is listening on.
func reservePorts(t devtest.T, n int) []int {
	listeners := make([]net.Listener, 0, n)
	ports := make([]int, 0, n)
	for range n {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		t.Require().NoError(err, "reserve an ephemeral port")
		listeners = append(listeners, l)
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	for _, l := range listeners {
		_ = l.Close()
	}
	return ports
}

// var assertion for the seam the presets reach lokahi through. It is here rather than beside the
// type so that a method removed from LokahiSupernode fails to compile at the seam that needs it.
var _ SharedSupernode = (*LokahiSupernode)(nil)

// startSharedLokahiSupernode brings up lokahi as the shared supernode of a multi-chain preset,
// and hands back the per-chain endpoints in configuration order.
//
// This is what DEVSTACK_SUPERNODE_KIND=lokahi selects, and it is the counterpart of the Go
// supernode bring-up in multichain_supernode_runtime.go rather than a second bring-up path: both
// are handed the same lokahiSupernodeConfig-shaped inputs, which is what keeps the two
// implementations from being given different worlds to run in.
//
// The endpoints are L2CLNode, not *SuperNodeProxy: a chain of a Go supernode is a route on one
// shared RPC, and a chain of lokahi is its own socket behind its own proxy. Everything a preset
// does with them — peer them, follow them, batch from them, propose from them — is on that
// interface.
func startSharedLokahiSupernode(t devtest.T, cfg lokahiSupernodeConfig) (*LokahiSupernode, []L2CLNode) {
	sn := startLokahiSupernode(t, cfg)
	cls := make([]L2CLNode, len(cfg.chains))
	for i := range cfg.chains {
		cls[i] = sn.ChainCL(i)
	}
	return sn, cls
}
