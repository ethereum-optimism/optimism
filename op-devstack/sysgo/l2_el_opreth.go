package sysgo

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type OpReth struct {
	mu sync.Mutex

	id        stack.L2ELNodeID
	l2Net     *L2Network
	jwtPath   string
	jwtSecret [32]byte
	authRPC   string
	userRPC   string

	authProxy *tcpproxy.Proxy
	userProxy *tcpproxy.Proxy

	execPath string
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.P

	sub *SubProcess

	l2MetricsRegistrar L2MetricsRegistrar
}

var _ L2ELNode = (*OpReth)(nil)

func (n *OpReth) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.userRPC, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	// Do not have to check whether client is readOnly because
	// all external L2 Clients will be wrapped with op-geth sysgo devstack, supporting readOnly
	var engineCl client.RPC
	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(n.jwtSecret))
	engineCl, err = client.NewRPC(system.T().Ctx(), system.Logger(), n.authRPC, client.WithGethRPCOptions(auth))
	require.NoError(err)
	system.T().Cleanup(engineCl.Close)

	l2Net := system.L2Network(stack.L2NetworkID(n.id.ChainID()))
	sysL2EL := shim.NewL2ELNode(shim.L2ELNodeConfig{
		RollupCfg: l2Net.RollupConfig(),
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			Client:       rpcCl,
			ChainID:      n.id.ChainID(),
		},
		EngineClient: engineCl,
		ID:           n.id,
	})
	sysL2EL.SetLabel(match.LabelVendor, string(match.OpReth))
	l2Net.(stack.ExtensibleL2Network).AddL2ELNode(sysL2EL)
}

func (n *OpReth) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub != nil {
		n.p.Logger().Warn("op-reth already started")
		return
	}
	if n.authProxy == nil {
		n.authProxy = tcpproxy.New(n.p.Logger())
		n.p.Require().NoError(n.authProxy.Start())
		n.p.Cleanup(func() {
			n.authProxy.Close()
		})
		n.authRPC = "ws://" + n.authProxy.Addr()
	}
	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.p.Logger())
		n.p.Require().NoError(n.userProxy.Start())
		n.p.Cleanup(func() {
			n.userProxy.Close()
		})
		n.userRPC = "ws://" + n.userProxy.Addr()
	}
	logOut := logpipe.ToLogger(n.p.Logger().New("component", "op-reth", "src", "stdout"))
	logErr := logpipe.ToLogger(n.p.Logger().New("component", "op-reth", "src", "stderr"))

	authRPCChan := make(chan string, 1)
	defer close(authRPCChan)

	metricsTargetChan := make(chan PrometheusMetricsTarget, 1)
	defer close(metricsTargetChan)

	userRPCChan := make(chan string, 1)
	defer close(userRPCChan)
	onLogEntry := func(e logpipe.LogEntry) {
		msg := e.LogMessage()
		if msg == "RPC WS server started" {
			select {
			case userRPCChan <- "ws://" + e.FieldValue("url").(string):
			default:
			}
		} else if msg == "RPC auth server started" {
			select {
			case authRPCChan <- "ws://" + e.FieldValue("url").(string):
			default:
			}
		} else if metricsUrl, found := strings.CutPrefix(msg, "Starting metrics endpoint at "); found {
			// expected format: "Starting metrics endpoint at 127.0.0.1:9091"
			if !strings.HasPrefix(metricsUrl, "http") {
				metricsUrl = fmt.Sprintf("http://%s", metricsUrl)
			}
			parsedUrl, err := url.Parse(metricsUrl)
			n.p.Require().NoError(err, "invalid metrics url output to logs", "log", msg)
			n.p.Require().NotEmpty(parsedUrl.Port(), "empty port in logged metrics url", "log", msg)
			metricsTargetChan <- NewPrometheusMetricsTarget(parsedUrl.Hostname(), parsedUrl.Port(), false)
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
	n.sub = NewSubProcess(n.p, stdOutLogs, stdErrLogs)

	err := n.sub.Start(n.execPath, n.args, n.env)
	n.p.Require().NoError(err, "Must start")

	var userRPCAddr, authRPCAddr string
	n.p.Require().NoError(tasks.Await(n.p.Ctx(), userRPCChan, &userRPCAddr), "need user RPC")
	n.p.Require().NoError(tasks.Await(n.p.Ctx(), authRPCChan, &authRPCAddr), "need auth RPC")

	if areMetricsEnabled() {
		var metricsTarget PrometheusMetricsTarget
		n.p.Require().NoError(tasks.Await(n.p.Ctx(), metricsTargetChan, &metricsTarget), "need metrics endpoint")
		n.l2MetricsRegistrar.RegisterL2MetricsTargets(n.id, metricsTarget)
	}

	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), userRPCAddr))
	n.authProxy.SetUpstream(ProxyAddr(n.p.Require(), authRPCAddr))
}

// Stop stops the op-reth node.
// warning: no restarts supported yet, since the RPC port is not remembered.
func (n *OpReth) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	err := n.sub.Stop(true)
	n.p.Require().NoError(err, "Must stop")
	n.sub = nil
}

func (n *OpReth) UserRPC() string {
	return n.userRPC
}

func (n *OpReth) EngineRPC() string {
	return n.authRPC
}

func (n *OpReth) JWTPath() string {
	return n.jwtPath
}

func WithOpReth(id stack.L2ELNodeID, opts ...L2ELOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), id))
		require := p.Require()

		l2Net, ok := orch.l2Nets.Get(id.ChainID())
		require.True(ok, "L2 network required")

		cfg := DefaultL2ELConfig()
		orch.l2ELOptions.Apply(p, id, cfg)       // apply global options
		L2ELOptionBundle(opts).Apply(p, id, cfg) // apply specific options

		jwtPath, jwtSecret := orch.writeDefaultJWT()

		useInterop := l2Net.genesis.Config.InteropTime != nil

		supervisorRPC := ""
		if useInterop {
			require.NotNil(cfg.SupervisorID, "supervisor is required for interop")
			sup, ok := orch.supervisors.Get(*cfg.SupervisorID)
			require.True(ok, "supervisor is required for interop")
			supervisorRPC = sup.UserRPC()
		}

		tempDir := p.TempDir()
		data, err := json.Marshal(l2Net.genesis)
		p.Require().NoError(err, "must json-encode genesis")
		chainConfigPath := filepath.Join(tempDir, "genesis.json")
		p.Require().NoError(os.WriteFile(chainConfigPath, data, 0o644), "must write genesis file")

		dataDirPath := filepath.Join(tempDir, "data")
		p.Require().NoError(os.MkdirAll(dataDirPath, 0o755), "must create datadir")

		// reth writes logs not just to stdout, but also to file,
		// and to global user-cache by default, rather than the datadir.
		// So we customize this to temp-dir too, to not pollute the user-cache dir.
		logDirPath := filepath.Join(tempDir, "logs")
		p.Require().NoError(os.MkdirAll(dataDirPath, 0o755), "must create logs dir")

		tempP2PPath := filepath.Join(tempDir, "p2pkey.txt")

		execPath := os.Getenv("OP_RETH_EXEC_PATH")
		p.Require().NotEmpty(execPath, "OP_RETH_EXEC_PATH environment variable must be set")
		_, err = os.Stat(execPath)
		p.Require().NotErrorIs(err, os.ErrNotExist, "executable must exist")

		// reth does not support env-var configuration like the Go services,
		// so we use the CLI flags instead.
		args := []string{
			"node",
			"--addr=127.0.0.1",
			"--authrpc.addr=127.0.0.1",
			"--authrpc.jwtsecret=" + jwtPath,
			"--authrpc.port=0",
			"--builder.deadline=2",
			"--builder.interval=100ms",
			"--chain=" + chainConfigPath,
			"--color=never",
			"--datadir=" + dataDirPath,
			"--disable-discovery",
			"--http",
			"--http.api=admin,debug,eth,net,trace,txpool,web3,rpc,reth,miner",
			"--http.addr=127.0.0.1",
			"--http.port=0",
			"--ipcdisable",
			"--log.file.directory=" + logDirPath,
			"--log.stdout.format=json",
			"--nat=none",
			"--p2p-secret-key=" + tempP2PPath,
			"--port=0",
			"--rpc.eth-proof-window=30",
			"--txpool.minimum-priority-fee=1",
			"--txpool.nolocals",
			"--with-unused-ports",
			"--ws",
			"--ws.api=admin,debug,eth,net,trace,txpool,web3,rpc,reth,miner",
			"--ws.addr=127.0.0.1",
			"--ws.port=0",
			"-vvvv",
		}

		if areMetricsEnabled() {
			// NB: Instead of getAvailableLocalPort, we should pass "0" so the OS picks its
			// own port, but that is not currently logged properly so we cannot parse it.
			// See: https://github.com/op-rs/op-reth/issues/333
			metricsPort, err := getAvailableLocalPort()
			p.Require().NoError(err, "WithOpReth: getting metrics port")
			args = append(args, "--metrics="+metricsPort)
		}

		if supervisorRPC != "" {
			args = append(args, "--rollup.supervisor-http="+supervisorRPC)
		}

		l2EL := &OpReth{
			id:                 id,
			l2Net:              l2Net,
			jwtPath:            jwtPath,
			jwtSecret:          jwtSecret,
			authRPC:            "",
			userRPC:            "",
			execPath:           execPath,
			args:               args,
			env:                []string{},
			p:                  orch.p,
			l2MetricsRegistrar: orch,
		}

		p.Logger().Info("Starting op-reth")
		l2EL.Start()
		p.Cleanup(l2EL.Stop)
		p.Logger().Info("op-reth is ready", "userRPC", l2EL.userRPC, "authRPC", l2EL.authRPC)
		require.True(orch.l2ELs.SetIfMissing(id, l2EL), "must be unique L2 EL node")
	})
}
