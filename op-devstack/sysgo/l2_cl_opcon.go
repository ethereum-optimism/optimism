package sysgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum/go-ethereum/log"
)

// OpConNode launches `op-con-node` (the opql consensus-only verifier) as an
// external subprocess and fronts its JSON-RPC, mirroring the kona-node path.
//
// op-con-node is verifier-only: it derives the safe L2 chain from L1 and drives
// its paired execution engine over the Engine API. It does not sequence and has
// no P2P gossip, so the unsafe head is followed from a trusted L2 execution RPC
// (`--l2-follow-rpc`) when a follow source is wired, rather than over gossip.
type OpConNode struct {
	mu sync.Mutex

	name    string
	chainID eth.ChainID

	userRPC          string
	interopEndpoint  string // not supported yet
	interopJwtSecret eth.Bytes32

	execPath string
	args     []string
	env      []string

	metricsHost string
	metricsPort string

	p   devtest.T
	sub *SubProcess

	l2MetricsRegistrar L2MetricsRegistrar
}

func (n *OpConNode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub != nil {
		n.p.Logger().Warn("op-con-node already started")
		return
	}

	logOut := logpipe.ToLoggerWithMinLevel(n.p.Logger().New("component", "op-con-node", "src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(n.p.Logger().New("component", "op-con-node", "src", "stderr"), log.LevelWarn)
	stdOutLogs := logpipe.LogCallback(func(line []byte) { logOut(logpipe.ParseRustStructuredLogs(line)) })
	stdErrLogs := logpipe.LogCallback(func(line []byte) { logErr(logpipe.ParseRustStructuredLogs(line)) })
	n.sub = NewSubProcess(n.p, stdOutLogs, stdErrLogs)

	err := n.sub.Start(n.execPath, n.args, n.env)
	n.p.Require().NoError(err, "must start op-con-node")

	// op-con-node binds the RPC to the fixed port we chose, so poll the endpoint
	// for readiness rather than coupling to a specific startup log line.
	n.awaitRPCReady()

	if areMetricsEnabled() && n.metricsPort != "" && n.l2MetricsRegistrar != nil {
		n.l2MetricsRegistrar.RegisterL2MetricsTargets(n.name, NewPrometheusMetricsTarget(n.metricsHost, n.metricsPort, false))
	}
}

// awaitRPCReady blocks until the JSON-RPC server answers, or the test context /
// a 90s deadline elapses. An HTTP 200 (even a JSON-RPC error body) means the
// server is up and accepting requests.
func (n *OpConNode) awaitRPCReady() {
	ctx, cancel := context.WithTimeout(n.p.Ctx(), 90*time.Second)
	defer cancel()
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"optimism_syncStatus","params":[]}`)
	for {
		select {
		case <-ctx.Done():
			n.p.Require().NoError(ctx.Err(), "op-con-node RPC did not become ready at %s", n.userRPC)
			return
		default:
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.userRPC, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			resp, doErr := http.DefaultClient.Do(req)
			if doErr == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func (n *OpConNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		n.p.Logger().Warn("op-con-node already stopped")
		return
	}
	err := n.sub.Stop(true)
	n.p.Require().NoError(err, "must stop op-con-node")
	n.sub = nil
}

func (n *OpConNode) UserRPC() string {
	return n.userRPC
}

func (n *OpConNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return n.interopEndpoint, n.interopJwtSecret
}

var _ L2CLNode = (*OpConNode)(nil)

// startMixedOpConNode bootstraps op-con-node's flat rollup config + genesis
// checkpoint from the standard rollup config, then launches `op-con-node run`
// as a verifier paired with l2EL.
//
// The binary is resolved via rustbin, which honors RUST_BINARY_PATH_OP_CON_NODE
// (an absolute binary path) or RUST_SRC_DIR_OP_CON_NODE (the opql cargo project
// root) — op-con-node lives in a sibling repo, not this monorepo.
func startMixedOpConNode(
	t devtest.T,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	l2EL L2ELNode,
	clKey string,
	elKey string,
	followSource string,
	metricsRegistrar L2MetricsRegistrar,
) *OpConNode {
	dir := t.TempDir()

	// Standard op-node rollup config -> file (the bootstrap step's input).
	stdRollupPath := filepath.Join(dir, "rollup.json")
	rollupData, err := json.Marshal(l2Net.rollupCfg)
	t.Require().NoError(err, "marshal rollup config")
	t.Require().NoError(os.WriteFile(stdRollupPath, rollupData, 0o644))

	execPath, err := rustbin.Spec{
		SrcDir:  "op-con-node",
		Package: "op-con-node",
		Binary:  "op-con-node",
	}.EnsureExists(t.Ctx(), t.Logger())
	t.Require().NoError(err, "prepare op-con-node binary (set RUST_BINARY_PATH_OP_CON_NODE or RUST_SRC_DIR_OP_CON_NODE)")
	t.Require().NotEmpty(execPath, "op-con-node binary path resolved")

	l2ELUserRPC := strings.ReplaceAll(l2EL.UserRPC(), "ws://", "http://")
	l2ELEngineRPC := strings.ReplaceAll(l2EL.EngineRPC(), "ws://", "http://")

	// Optional debug: copy the rollup config to a stable dir for inspection (the
	// working dir is a cleaned-up t.TempDir()).
	if dbg := os.Getenv("DEVSTACK_OPCON_DEBUG_DIR"); dbg != "" {
		_ = os.MkdirAll(dbg, 0o755)
		if data, rErr := os.ReadFile(stdRollupPath); rErr == nil {
			_ = os.WriteFile(filepath.Join(dbg, clKey+"-rollup.json"), data, 0o644)
		}
	}

	rpcPort, err := getAvailableLocalPort()
	t.Require().NoError(err, "op-con-node rpc port")

	// op-con-node consumes the standard op-node rollup config directly and anchors
	// at L2 genesis with no checkpoint: it fetches the genesis block via
	// --l2-eth-rpc to build the anchor in-process.
	args := []string{
		"run",
		"--datadir", filepath.Join(dir, "datadir"),
		"--l1-rpc", l1EL.UserRPC(),
		"--l1-beacon-rpc", l1CL.beaconHTTPAddr,
		"--l1-source-tag", "latest",
		"--l2-engine-rpc", l2ELEngineRPC,
		"--l2.jwt-secret", l2EL.JWTPath(),
		"--l2-eth-rpc", l2ELUserRPC,
		"--rollup-config", stdRollupPath,
		"--rpc-addr", "127.0.0.1",
		"--rpc-port", rpcPort,
	}

	// Follow the unsafe head from a trusted L2 execution RPC when one is wired
	// (op-con-node has no P2P). Without it, op-con-node tracks the safe chain via
	// L1 derivation only.
	if followSource != "" {
		args = append(args, "--l2-follow-rpc", strings.ReplaceAll(followSource, "ws://", "http://"))
	}

	// Devstack L1 finality may not advance; default the derive-side guard to
	// disabled so startup does not stall waiting for a finalized tag. Override
	// with DEVSTACK_OPCON_L1_FINALIZED_GUARD=required against a finalizing L1.
	args = append(args, "--l1-finalized-guard", getEnvVarOrDefault("DEVSTACK_OPCON_L1_FINALIZED_GUARD", "disabled"))

	// Expected unsafe-block (gossip) signer for admin_verifyUnsafePayload, used by
	// the op-conp2p P2P sidecar's verdict delegation. When set, op-con-node can
	// attribute gossiped blocks to the sequencer; otherwise the verdict RPC
	// returns `ignore`. In a with-P2P preset this is the sequencer's P2P signer
	// address.
	if signer := os.Getenv("DEVSTACK_OPCON_UNSAFE_SIGNER"); signer != "" {
		args = append(args, "--unsafe-block-signer", signer)
	}

	var metricsHost, metricsPort string
	if areMetricsEnabled() {
		metricsHost = "127.0.0.1"
		metricsPort, err = getAvailableLocalPort()
		t.Require().NoError(err, "op-con-node metrics port")
		args = append(args, "--metrics-enabled", "--metrics-addr", metricsHost, "--metrics-port", metricsPort)
	}

	n := &OpConNode{
		name:               clKey,
		chainID:            l2Net.ChainID(),
		userRPC:            fmt.Sprintf("http://127.0.0.1:%s", rpcPort),
		execPath:           execPath,
		args:               args,
		env:                nil, // inherits os.Environ via SubProcess
		metricsHost:        metricsHost,
		metricsPort:        metricsPort,
		p:                  t,
		l2MetricsRegistrar: metricsRegistrar,
	}
	t.Logger().Info("Starting op-con-node", "name", clKey, "chain", l2Net.ChainID(), "el", elKey, "rpc", n.userRPC)
	n.Start()
	t.Cleanup(n.Stop)
	t.Logger().Info("op-con-node is up", "name", clKey, "chain", l2Net.ChainID(), "rpc", n.UserRPC())
	return n
}
