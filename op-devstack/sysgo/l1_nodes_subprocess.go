package sysgo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/blobstore"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ExternalL1Geth struct {
	mu sync.Mutex

	id    stack.L1ELNodeID
	l1Net *L1Network
	// authRPC points to a proxy that forwards to geth's endpoint
	authRPC string
	// userRPC points to a proxy that forwards to geth's endpoint
	userRPC string

	authProxy *tcpproxy.Proxy
	userProxy *tcpproxy.Proxy

	execPath string
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.P

	sub *SubProcess
}

func (*ExternalL1Geth) l1ELNode() {}

func (n *ExternalL1Geth) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.userRPC, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	l1Net := system.L1Network(stack.L1NetworkID(n.id.ChainID()))
	sysL1EL := shim.NewL1ELNode(shim.L1ELNodeConfig{
		ID: n.id,
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			Client:       rpcCl,
			ChainID:      n.id.ChainID(),
		},
	})
	sysL1EL.SetLabel(match.LabelVendor, string(match.Geth))
	l1Net.(stack.ExtensibleL1Network).AddL1ELNode(sysL1EL)
}

func (n *ExternalL1Geth) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub != nil {
		n.p.Logger().Warn("geth already started")
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
	logOut := logpipe.ToLogger(n.p.Logger().New("src", "stdout"))
	logErr := logpipe.ToLogger(n.p.Logger().New("src", "stderr"))
	userRPC := make(chan string, 1)
	authRPC := make(chan string, 1)
	onLogEntry := func(e logpipe.LogEntry) {
		switch e.LogMessage() {
		case "WebSocket enabled":
			select {
			case userRPC <- e.FieldValue("url").(string):
			default:
			}
		case "HTTP server started":
			if e.FieldValue("auth").(bool) {
				select {
				case authRPC <- "http://" + e.FieldValue("endpoint").(string):
				default:
				}
			}
		}
	}
	stdOutLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseGoStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
	})
	stdErrLogs := logpipe.LogProcessor(func(line []byte) {
		e := logpipe.ParseGoStructuredLogs(line)
		logErr(e)
		onLogEntry(e)
	})
	n.sub = NewSubProcess(n.p, stdOutLogs, stdErrLogs)

	err := n.sub.Start(n.execPath, n.args, n.env)
	n.p.Require().NoError(err, "Must start")

	var userRPCAddr, authRPCAddr string
	n.p.Require().NoError(tasks.Await(n.p.Ctx(), userRPC, &userRPCAddr), "need user RPC")
	n.p.Require().NoError(tasks.Await(n.p.Ctx(), authRPC, &authRPCAddr), "need auth RPC")

	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), userRPCAddr))
	n.authProxy.SetUpstream(ProxyAddr(n.p.Require(), authRPCAddr))
}

func (n *ExternalL1Geth) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	err := n.sub.Stop()
	n.p.Require().NoError(err, "Must stop")
	n.sub = nil
}

func (n *ExternalL1Geth) UserRPC() string {
	return n.userRPC
}

func (n *ExternalL1Geth) AuthRPC() string {
	return n.authRPC
}

const GethExecPathEnvVar = "SYSGO_GETH_EXEC_PATH"

func WithL1NodesSubprocess(id stack.L1ELNodeID, clID stack.L1CLNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), id))
		require := p.Require()

		execPath, ok := os.LookupEnv(GethExecPathEnvVar)
		require.True(ok)
		_, err := os.Stat(execPath)
		p.Require().NotErrorIs(err, os.ErrNotExist, "geth executable must exist")

		l1Net, ok := orch.l1Nets.Get(id.ChainID())
		require.True(ok, "L1 network required")

		jwtPath, jwtSecret := orch.writeDefaultJWT()

		tempDir := p.TempDir()
		data, err := json.Marshal(l1Net.genesis)
		p.Require().NoError(err, "must json-encode genesis")
		chainConfigPath := filepath.Join(tempDir, "genesis.json")
		p.Require().NoError(os.WriteFile(chainConfigPath, data, 0o644), "must write genesis file")

		dataDirPath := filepath.Join(tempDir, "data")
		p.Require().NoError(os.MkdirAll(dataDirPath, 0o755), "must create datadir")

		cmd := exec.Command(execPath, "--datadir", dataDirPath, "init", chainConfigPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		require.NoError(cmd.Run(), "initialize geth datadir")

		args := []string{
			"--log.format", "json",
			"--datadir", dataDirPath,
			"--ws", "--ws.addr", "127.0.0.1", "--ws.port", "0", "--ws.origins", "*", "--ws.api", "admin,debug,eth,net,txpool",
			"--authrpc.addr", "127.0.0.1", "--authrpc.port", "0", "--authrpc.jwtsecret", jwtPath,
			"--ipcdisable",
			"--nodiscover",
			"--verbosity", "5",
			"--miner.recommit", "2s",
			"--gcmode", "archive",
		}

		l1EL := &ExternalL1Geth{
			id:       id,
			l1Net:    l1Net,
			authRPC:  "",
			userRPC:  "",
			execPath: execPath,
			args:     args,
			env:      []string{},
			p:        p,
		}

		p.Logger().Info("Starting geth")
		l1EL.Start()
		p.Cleanup(l1EL.Stop)
		p.Logger().Info("geth is ready", "userRPC", l1EL.userRPC, "authRPC", l1EL.authRPC)
		require.True(orch.l1ELs.SetIfMissing(id, l1EL), "must be unique L2 EL node")

		backend, err := ethclient.DialContext(p.Ctx(), l1EL.userRPC)
		require.NoError(err)

		l1Clock := clock.SystemClock
		if orch.timeTravelClock != nil {
			l1Clock = orch.timeTravelClock
		}

		bcn := fakebeacon.NewBeacon(p.Logger(), blobstore.New(), l1Net.genesis.Timestamp, l1Net.blockTime)
		p.Cleanup(func() {
			_ = bcn.Close()
		})
		require.NoError(bcn.Start("127.0.0.1:0"))
		beaconApiAddr := bcn.BeaconAddr()
		require.NotEmpty(beaconApiAddr, "beacon API listener must be up")

		engineCl, err := dialEngine(p.Ctx(), l1EL.AuthRPC(), jwtSecret)
		require.NoError(err)
		fp := &FakePoS{
			p:       p,
			fakepos: geth.NewFakePoS(backend, engineCl, l1Clock, p.Logger(), l1Net.blockTime, 20, bcn, l1Net.genesis.Config),
		}
		fp.Start()
		p.Cleanup(fp.Stop)
		orch.l1CLs.Set(clID, &L1CLNode{
			id:             clID,
			beaconHTTPAddr: bcn.BeaconAddr(),
			beacon:         bcn,
			fakepos:        fp,
		})
	})
}
