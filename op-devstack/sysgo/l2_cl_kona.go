package sysgo

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

type KonaNode struct {
	mu sync.Mutex

	id stack.L2CLNodeID

	userRPC          string
	interopEndpoint  string // warning: currently not fully supported
	interopJwtSecret eth.Bytes32
	el               stack.L2ELNodeID

	userProxy *tcpproxy.Proxy

	execPath string
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.P

	sub *SubProcess
}

func (k *KonaNode) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), k.userRPC, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	sysL2CL := shim.NewL2CLNode(shim.L2CLNodeConfig{
		CommonConfig:     shim.NewCommonConfig(system.T()),
		ID:               k.id,
		Client:           rpcCl,
		InteropEndpoint:  k.interopEndpoint,
		InteropJwtSecret: k.interopJwtSecret,
	})
	sysL2CL.SetLabel(match.LabelVendor, string(match.KonaNode))
	l2Net := system.L2Network(stack.L2NetworkID(k.id.ChainID()))
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
	sysL2CL.(stack.LinkableL2CLNode).LinkEL(l2Net.L2ELNode(k.el))
}

func (k *KonaNode) Start() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.sub != nil {
		k.p.Logger().Warn("Kona-node already started")
		return
	}
	// Create a proxy for the user RPC,
	// so other services can connect, and stay connected, across restarts.
	if k.userProxy == nil {
		k.userProxy = tcpproxy.New(k.p.Logger())
		k.p.Require().NoError(k.userProxy.Start())
		k.p.Cleanup(func() {
			k.userProxy.Close()
		})
		k.userRPC = "http://" + k.userProxy.Addr()
	}
	// Create the sub-process.
	// We pipe sub-process logs to the test-logger.
	// And inspect them along the way, to get the RPC server address.
	logOut := logpipe.ToLogger(k.p.Logger().New("src", "stdout"))
	logErr := logpipe.ToLogger(k.p.Logger().New("src", "stderr"))
	userRPC := make(chan string, 1)
	onLogEntry := func(e logpipe.LogEntry) {
		switch e.LogMessage() {
		case "RPC server bound to address":
			userRPC <- "http://" + e.FieldValue("addr").(string)
		}
	}
	stdOutLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
	})
	stdErrLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logErr(e)
	})
	k.sub = NewSubProcess(k.p, stdOutLogs, stdErrLogs)

	err := k.sub.Start(k.execPath, k.args, k.env)
	k.p.Require().NoError(err, "Must start")

	var userRPCAddr string
	k.p.Require().NoError(tasks.Await(k.p.Ctx(), userRPC, &k.userRPC), "need user RPC")

	k.userProxy.SetUpstream(ProxyAddr(k.p.Require(), userRPCAddr))
}

// Stop stops the kona node.
// warning: no restarts supported yet, since the RPC port is not remembered.
func (k *KonaNode) Stop() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.sub == nil {
		k.p.Logger().Warn("kona-node already stopped")
		return
	}
	err := k.sub.Stop()
	k.p.Require().NoError(err, "Must stop")
	k.sub = nil
}

func (k *KonaNode) UserRPC() string {
	return k.userRPC
}

func (k *KonaNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return k.interopEndpoint, k.interopJwtSecret
}

var _ L2CLNode = (*KonaNode)(nil)

func WithKonaNode(l2CLID stack.L2CLNodeID, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID, opts ...L2CLOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l2CLID))

		require := p.Require()

		l2Net, ok := orch.l2Nets.Get(l2CLID.ChainID())
		require.True(ok, "l2 network required")

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")

		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok, "l1 CL node required")

		l2EL, ok := orch.l2ELs.Get(l2ELID)
		require.True(ok, "l2 EL node required")

		cfg := DefaultL2CLConfig()
		orch.l2CLOptions.Apply(orch.P(), l2CLID, cfg)       // apply global options
		L2CLOptionBundle(opts).Apply(orch.P(), l2CLID, cfg) // apply specific options

		tempKonaDir := p.TempDir()

		tempP2PPath := filepath.Join(tempKonaDir, "p2pkey.txt")

		tempRollupCfgPath := filepath.Join(tempKonaDir, "rollup.json")
		rollupCfgData, err := json.Marshal(l2Net.rollupCfg)
		p.Require().NoError(err, "must write rollup config")
		p.Require().NoError(err, os.WriteFile(tempRollupCfgPath, rollupCfgData, 0o644))

		envVars := []string{
			"KONA_NODE_L1_ETH_RPC=" + l1EL.userRPC,
			"KONA_NODE_L1_BEACON=" + l1CL.beaconHTTPAddr,
			// TODO: WS RPC addresses do not work and will make the startup panic with a connection error in the
			// JWT validation / engine-capabilities setup code-path.
			"KONA_NODE_L2_ENGINE_RPC=" + strings.ReplaceAll(l2EL.EngineRPC(), "ws://", "http://"),
			"KONA_NODE_L2_ENGINE_AUTH=" + l2EL.JWTPath(),
			"KONA_NODE_ROLLUP_CONFIG=" + tempRollupCfgPath,
			"KONA_NODE_P2P_NO_DISCOVERY=true",
			"KONA_NODE_P2P_PRIV_PATH=" + tempP2PPath,
			"KONA_NODE_RPC_ADDR=127.0.0.1",
			"KONA_NODE_RPC_PORT=0",
			"KONA_NODE_RPC_WS_ENABLED=true",
			"KONA_METRICS_ENABLED=false",
			"KONA_NODE_LOG_LEVEL=3", // info level
			"KONA_NODE_LOG_STDOUT_FORMAT=json",
			// p2p ports
			"KONA_NODE_P2P_LISTEN_IP=127.0.0.1",
			"KONA_NODE_P2P_LISTEN_TCP_PORT=0",
			"KONA_NODE_P2P_LISTEN_UDP_PORT=0",
		}
		if cfg.IsSequencer {
			p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2CLID.ChainID().ToBig()))
			require.NoError(err, "need p2p key for sequencer")
			p2pKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSA(p2pKey))
			// TODO: Kona should support loading keys from a file
			//tempSeqKeyPath := filepath.Join(tempKonaDir, "p2p-sequencer.txt")
			//p.Require().NoError(err, os.WriteFile(tempSeqKeyPath, []byte(p2pKeyHex), 0o644))
			envVars = append(envVars,
				"KONA_NODE_P2P_SEQUENCER_KEY="+p2pKeyHex,
				"KONA_NODE_SEQUENCER_L1_CONFS=0",
				"KONA_NODE_MODE=Sequencer",
			)
		} else {
			envVars = append(envVars,
				"KONA_NODE_MODE=Validator",
			)
		}

		execPath := os.Getenv("KONA_NODE_EXEC_PATH")
		p.Require().NotEmpty(execPath, "KONA_NODE_EXEC_PATH environment variable must be set")
		_, err = os.Stat(execPath)
		p.Require().NotErrorIs(err, os.ErrNotExist, "executable must exist")

		k := &KonaNode{
			id:               l2CLID,
			userRPC:          "", // retrieved from logs
			interopEndpoint:  "", // retrieved from logs
			interopJwtSecret: eth.Bytes32{},
			el:               l2ELID,
			execPath:         execPath,
			args:             []string{"node"},
			env:              envVars,
			p:                p,
		}
		p.Logger().Info("Starting kona-node")
		k.Start()
		p.Cleanup(k.Stop)
		p.Logger().Info("Kona-node is up", "rpc", k.UserRPC())
		require.True(orch.l2CLs.SetIfMissing(l2CLID, k), "must not already exist")
	})
}
