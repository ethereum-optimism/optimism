package sysgo

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

type KonaSupervisor struct {
	mu sync.Mutex

	id      stack.SupervisorID
	userRPC string

	userProxy *tcpproxy.Proxy

	execPath string
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.P

	sub *SubProcess
}

var _ stack.Lifecycle = (*OpSupervisor)(nil)

func (s *KonaSupervisor) hydrate(sys stack.ExtensibleSystem) {
	tlog := sys.Logger().New("id", s.id)
	supClient, err := client.NewRPC(sys.T().Ctx(), tlog, s.userRPC, client.WithLazyDial())
	sys.T().Require().NoError(err)
	sys.T().Cleanup(supClient.Close)

	sys.AddSupervisor(shim.NewSupervisor(shim.SupervisorConfig{
		CommonConfig: shim.NewCommonConfig(sys.T()),
		ID:           s.id,
		Client:       supClient,
	}))
}

func (s *KonaSupervisor) UserRPC() string {
	return s.userRPC
}

func (s *KonaSupervisor) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sub != nil {
		s.p.Logger().Warn("Kona-supervisor already started")
		return
	}

	// Create a proxy for the user RPC,
	// so other services can connect, and stay connected, across restarts.
	if s.userProxy == nil {
		s.userProxy = tcpproxy.New(s.p.Logger())
		s.p.Require().NoError(s.userProxy.Start())
		s.p.Cleanup(func() {
			s.userProxy.Close()
		})
		s.userRPC = "http://" + s.userProxy.Addr()
	}

	// Create the sub-process.
	// We pipe sub-process logs to the test-logger.
	// And inspect them along the way, to get the RPC server address.
	logOut := logpipe.ToLogger(s.p.Logger().New("src", "stdout"))
	logErr := logpipe.ToLogger(s.p.Logger().New("src", "stderr"))
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

	s.sub = NewSubProcess(s.p, stdOutLogs, stdErrLogs)
	err := s.sub.Start(s.execPath, s.args, s.env)
	s.p.Require().NoError(err, "Must start")

	var userRPCAddr string
	s.p.Require().NoError(tasks.Await(s.p.Ctx(), userRPC, &userRPCAddr), "need user RPC")

	s.userProxy.SetUpstream(ProxyAddr(s.p.Require(), userRPCAddr))
}

func (s *KonaSupervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sub == nil {
		s.p.Logger().Warn("kona-supervisor already stopped")
		return
	}
	err := s.sub.Stop()
	s.p.Require().NoError(err, "Must stop")
	s.sub = nil
}

func WithKonaSupervisor(supervisorID stack.SupervisorID, clusterID stack.ClusterID, l1ELID stack.L1ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), supervisorID))
		require := p.Require()

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "need L1 EL node to connect supervisor to")

		cluster, ok := orch.clusters.Get(clusterID)
		require.True(ok, "need cluster to determine dependency set")

		require.NotNil(cluster.cfgset, "need a full config set")
		require.NoError(cluster.cfgset.CheckChains(), "config set must be valid")

		tempDataDir := p.TempDir()

		cfgDir := p.TempDir()

		depsetCfgPath := cfgDir + "/depset.json"
		depsetData, err := cluster.DepSet().MarshalJSON()
		require.NoError(err, "failed to marshal dependency set")
		p.Require().NoError(err, os.WriteFile(depsetCfgPath, depsetData, 0o644))

		rollupCfgPath := cfgDir + "/rollup-config-*.json"
		for _, l2Net := range orch.l2Nets.Values() {
			chainID := l2Net.id.ChainID()
			rollupData, err := json.Marshal(l2Net.rollupCfg)
			require.NoError(err, "failed to marshal rollup config")
			p.Require().NoError(err, os.WriteFile(cfgDir+"/rollup-config-"+chainID.String()+".json", rollupData, 0o644))
		}

		envVars := []string{
			"RPC_ADDR=127.0.0.1",
			"DATADIR=" + tempDataDir,
			"DEPENDENCY_SET=" + depsetCfgPath,
			"ROLLUP_CONFIG_PATHS=" + rollupCfgPath,
			"L1_RPC=" + l1EL.UserRPC(),
			"RPC_ENABLE_ADMIN=true",
			"L2_CONSENSUS_NODES=",
			"L2_CONSENSUS_JWT_SECRET=",
			"KONA_LOG_STDOUT_FORMAT=json",
		}

		execPath := os.Getenv("KONA_SUPERVISOR_EXEC_PATH")
		p.Require().NotEmpty(execPath, "KONA_SUPERVISOR_EXEC_PATH environment variable must be set")
		_, err = os.Stat(execPath)
		p.Require().NotErrorIs(err, os.ErrNotExist, "executable must exist")

		konaSupervisor := &KonaSupervisor{
			id:       supervisorID,
			userRPC:  "", // retrieved from logs
			execPath: execPath,
			args:     []string{},
			env:      envVars,
			p:        p,
		}
		orch.supervisors.Set(supervisorID, konaSupervisor)
		p.Logger().Info("Starting kona-supervisor")
		konaSupervisor.Start()
		p.Cleanup(konaSupervisor.Stop)
		p.Logger().Info("Kona-supervisor is up", "rpc", konaSupervisor.UserRPC())
	})
}
