package sysgo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// describe renders the requested configuration for diagnostics.
func (c lokahiSupernodeConfig) describe() string {
	chains := make([]string, 0, len(c.chains))
	for _, chain := range c.chains {
		chains = append(chains, fmt.Sprintf("%s(engine=%s)", chain.net.ChainID(), chain.el.EngineRPC()))
	}

	depSetChains := 0
	if c.depSet != nil {
		depSetChains = len(c.depSet.Chains())
	}

	return fmt.Sprintf("chains=[%s] l1=%s l1Beacon=%s depSetChains=%d sequencer=%t",
		strings.Join(chains, " "), c.l1ELRPC, c.l1BeaconAddr, depSetChains, c.sequencerEnabled)
}

// LokahiSupernode is a running lokahi process hosting one or more chains.
//
// Unlike the Go op-supernode, which serves every chain from one RPC under a /<chainID>
// route, lokahi gives each chain its own socket and keeps the method names a single-chain
// node has. Each of those sockets is fronted by a proxy so callers keep a stable URL, and
// ChainCL hands out an L2CLNode per chain addressing it.
type LokahiSupernode struct {
	mu sync.Mutex

	p        devtest.T
	logger   log.Logger
	execPath string
	args     []string

	// adminRPC is the process-wide admin/test RPC, discovered from the startup log line.
	// It is also where the supernode query API answers: supernode_syncStatus and
	// superroot_atTimestamp are statements about the whole chain set, so they are served by
	// the process rather than by any one chain's socket.
	adminRPC string
	// chains keeps the hosted chains in configuration order.
	chains []*lokahiChainCL

	// queryAPI is the client for the supernode query API, built on first use and reused. A
	// client rather than a connection: it dials lazily and survives a supernode restart,
	// because the admin RPC comes back on the same address.
	queryAPI *sources.SuperNodeClient

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
	// rpcPort is the port lokahi was configured to serve this chain on.
	rpcPort int
	// proxy fronts that port so callers keep one URL across a supernode restart.
	proxy   *tcpproxy.Proxy
	userRPC string
}

var _ L2CLNode = (*lokahiChainCL)(nil)

func (c *lokahiChainCL) Start()          {}
func (c *lokahiChainCL) Stop()           {}
func (c *lokahiChainCL) UserRPC() string { return c.userRPC }

// ChainID reports which chain this endpoint answers for.
func (c *lokahiChainCL) ChainID() eth.ChainID { return c.chainID }

// AdminRPC is the process-wide admin/test RPC of the supernode.
func (n *LokahiSupernode) AdminRPC() string { return n.adminRPC }

// QueryRPC is the endpoint the supernode query API answers on.
//
// The same socket as AdminRPC, named separately because they are different contracts: the
// admin namespace is lokahi's own and may change with lokahi, while the query API is the
// wire op-supernode defines and its consumers depend on. A preset wiring lokahi in as a
// stack.Supernode hands this to the frontend, not AdminRPC.
func (n *LokahiSupernode) QueryRPC() string { return n.adminRPC }

// QueryAPI returns the supernode query API of the running process.
//
// The client is op-service/sources.SuperNodeClient — the same one the in-process Go
// supernode is read through — so nothing about these two RPCs is reimplemented for lokahi
// on the Go side. Whether the answers match is a question about what lokahi serves, which
// is where it belongs, rather than about two Go clients agreeing.
func (n *LokahiSupernode) QueryAPI() apis.SupernodeQueryAPI {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.queryAPI == nil {
		n.p.Require().NotEmpty(n.adminRPC, "lokahi has no query RPC address yet")
		rpcCl, err := client.NewRPC(n.p.Ctx(), n.logger, n.adminRPC, client.WithLazyDial())
		n.p.Require().NoError(err, "dial the lokahi supernode query API")
		n.queryAPI = sources.NewSuperNodeClient(rpcCl)
		n.p.Cleanup(n.queryAPI.Close)
	}
	return n.queryAPI
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

	// Both are configurable in op-supernode and have no lokahi equivalent yet, so a caller
	// asking for them is told rather than quietly given a validator without interop.
	require.False(cfg.sequencerEnabled,
		"lokahi cannot sequence yet: its configuration has no sequencer p2p key, so a "+
			"sequencing chain would produce blocks it cannot sign. Requested: %s", cfg.describe())
	require.Nil(cfg.interopActivationTimestamp,
		"lokahi takes interop activation from the rollup config's Lagoon time and has no "+
			"override for it. Requested: %s", cfg.describe())

	dir := t.TempDirWithPrefix("l2-cl-lokahi")
	logger := t.Logger().New("component", "lokahi")

	// Every listener the process needs is reserved at once and only then released:
	// allocating them one at a time would hand out the same port twice, because each is
	// free again before the next is asked for.
	ports := reservePorts(t, 3*len(cfg.chains))

	chains := make([]*lokahiChainCL, len(cfg.chains))
	entries := make([]string, len(cfg.chains))
	for i, chain := range cfg.chains {
		rpcPort, tcpPort, udpPort := ports[3*i], ports[3*i+1], ports[3*i+2]
		chains[i] = &lokahiChainCL{chainID: chain.net.ChainID(), rpcPort: rpcPort}
		entries[i] = lokahiChainEntry(t, dir, chain, cfg, rpcPort, tcpPort, udpPort)
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
	logger.Info("lokahi is up", "admin", n.adminRPC)

	return n
}

// Start launches the process and waits for its admin RPC to be listening.
func (n *LokahiSupernode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub != nil {
		n.logger.Warn("lokahi already started")
		return
	}

	// The per-chain proxies are created once and repointed on a restart, so components
	// wired to a chain keep one URL for the life of the test.
	for _, chain := range n.chains {
		if chain.proxy == nil {
			chain.proxy = tcpproxy.New(n.logger.New("proxy", "lokahi-"+chain.chainID.String()))
			n.p.Require().NoError(chain.proxy.Start(), "lokahi chain proxy failed to start")
			n.p.Cleanup(func() { _ = chain.proxy.Close() })
			chain.userRPC = "http://" + chain.proxy.Addr()
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

	for _, chain := range n.chains {
		chain.proxy.SetUpstream(net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", chain.rpcPort)))
	}
}

// Stop terminates the process, leaving the per-chain proxies in place so a later Start can
// repoint them.
func (n *LokahiSupernode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		n.logger.Warn("lokahi already stopped")
		return
	}
	for _, chain := range n.chains {
		if chain.proxy != nil {
			chain.proxy.ClearUpstream()
		}
	}
	n.p.Require().NoError(n.sub.Stop(true), "must stop lokahi")
	n.sub = nil
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
	for _, chain := range n.chains {
		if chain.proxy != nil {
			chain.proxy.ClearUpstream()
		}
	}
	if err := n.sub.StopControlled(ctx, controlledInterruptWait, controlledKillWait); err != nil {
		return err
	}
	n.sub = nil
	return nil
}

// requireHostedChains asserts that lokahi hosts exactly the chains it was configured with,
// on the ports it was given.
func (n *LokahiSupernode) requireHostedChains(cfg lokahiSupernodeConfig) {
	require := n.p.Require()

	rpcCl, err := client.NewRPC(n.p.Ctx(), n.logger, n.adminRPC, client.WithLazyDial())
	require.NoError(err, "dial the lokahi admin RPC")
	defer rpcCl.Close()

	var hosted []struct {
		ChainID uint64 `json:"chainId"`
		RPCAddr string `json:"rpcAddr"`
	}
	require.NoError(rpcCl.CallContext(n.p.Ctx(), &hosted, "lokahi_chains"), "call lokahi_chains")
	require.Len(hosted, len(cfg.chains), "lokahi hosts a different number of chains than configured")

	for i, chain := range n.chains {
		require.Equal(eth.EvilChainIDToUInt64(chain.chainID), hosted[i].ChainID,
			"lokahi hosts chain %s at index %d", chain.chainID, i)
		require.Equal(fmt.Sprintf("127.0.0.1:%d", chain.rpcPort), hosted[i].RPCAddr,
			"lokahi serves chain %s on another port", chain.chainID)
	}
}

// lokahiConfigFile renders the global layer of the configuration plus the chain entries.
//
// The admin RPC asks for port 0 and the chains name concrete ports: only the admin RPC
// reports the address it bound, so it is the one that can be discovered and the chains are
// the ones that have to be told.
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
	// Acceptance tests drive a node through its admin API, which kona only registers when
	// admin is enabled; op-node's devstack node enables it too.
	fmt.Fprintf(&b, "[defaults]\ndatadir = %q\nmode = \"validator\"\n"+
		"rpc-addr = \"127.0.0.1\"\nrpc-enable-admin = true\np2p-listen-ip = \"127.0.0.1\"\n\n",
		filepath.Join(dir, "data"))
	b.WriteString(strings.Join(entries, "\n"))
	return b.String()
}

// lokahiChainEntry renders one [[chains]] entry, writing out the files it names.
func lokahiChainEntry(
	t devtest.T,
	dir string,
	chain lokahiSupernodeChain,
	cfg lokahiSupernodeConfig,
	rpcPort, tcpPort, udpPort int,
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
	fmt.Fprintf(&b, "rpc-port = %d\np2p-tcp-port = %d\np2p-udp-port = %d\n", rpcPort, tcpPort, udpPort)
	// Stated rather than read from L1. kona-node resolves the unsafe block signer from the
	// chain's SystemConfig contract when it is not given one; lokahi's configuration has no
	// such path, and a devnet chain is not in the superchain registry either, so the devstack
	// has to supply the address the sequencer's P2P key signs with.
	signer, err := chain.net.keys.Address(devkeys.SequencerP2PRole.Key(chain.net.ChainID().ToBig()))
	require.NoError(err, "need the sequencer p2p address of chain %d", chainID)
	fmt.Fprintf(&b, "unsafe-block-signer = %q\n", signer.Hex())

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

// failLokahiUnsupportedByPresets rejects DEVSTACK_SUPERNODE_KIND=lokahi for the multi-chain
// and interop presets.
//
// The launch above works, and so does the harness: stack.SupernodeTestControl now asks for
// apis.SupernodeInteropTestAPI, four context-and-error methods over plain data that a
// supernode in another process can serve. What is missing is the lokahi side of it. Until
// lokahi answers those calls there is nothing for the presets to drive it through, so they
// are turned away here rather than part-way through a run — and the failure names the one
// remaining gap instead of an impossible signature.
func failLokahiUnsupportedByPresets(t devtest.T, cfg lokahiSupernodeConfig) {
	t.Require().FailNowf("lokahi cannot back the shared-supernode presets yet",
		"%s=%s selected, but these presets drive the supernode's interop verifier through "+
			"apis.SupernodeInteropTestAPI (pause, resume, status, sealed blocks), which "+
			"lokahi does not serve yet. Requested: %s. Unset %s (or set it to %q) to run "+
			"the in-process Go op-supernode; the lokahi component itself is covered by the "+
			"two-chain lokahi preset.",
		devstackSupernodeKindEnv, SupernodeLokahi, cfg.describe(),
		devstackSupernodeKindEnv, SupernodeOpSupernode)
}
