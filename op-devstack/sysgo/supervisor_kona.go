package sysgo

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/log"
)

type KonaSupervisor struct {
	mu sync.Mutex

	name    string
	userRPC string

	userProxy *tcpproxy.Proxy

	execPath string
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.CommonT

	sub *SubProcess
}

var _ stack.Lifecycle = (*KonaSupervisor)(nil)

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
	logOut := logpipe.ToLoggerWithMinLevel(s.p.Logger().New("src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(s.p.Logger().New("src", "stderr"), log.LevelWarn)
	userRPC := make(chan string, 1)
	onLogEntry := func(e logpipe.LogEntry) {
		switch e.LogMessage() {
		case "RPC server bound to address":
			userRPC <- "http://" + e.FieldValue("addr").(string)
		}
	}
	stdOutLogs := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
	})
	stdErrLogs := logpipe.LogCallback(func(line []byte) {
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
	err := s.sub.Stop(true)
	s.p.Require().NoError(err, "Must stop")
	s.sub = nil
}

func WithKonaSupervisor(supervisorID stack.ComponentID, clusterID stack.ComponentID, l1ELID stack.ComponentID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), supervisorID))
		require := p.Require()

		l1EL, ok := orch.GetL1EL(l1ELID)
		require.True(ok, "need L1 EL node to connect supervisor to")

		cluster, ok := orch.GetCluster(clusterID)
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
		for _, l2NetID := range orch.registry.IDsByKind(stack.KindL2Network) {
			l2Net, ok := orch.GetL2Network(l2NetID)
			require.True(ok, "need l2 network")
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
			"KONA_LOG_LEVEL=3", // info level, consistent with l2_cl_kona.go
			"KONA_LOG_STDOUT_FORMAT=json",
		}

		execPath, err := EnsureRustBinary(p, RustBinarySpec{
			SrcDir:  "rust",
			Package: "kona-supervisor",
			Binary:  "kona-supervisor",
		})
		p.Require().NoError(err, "prepare kona-supervisor binary")
		p.Require().NotEmpty(execPath, "kona-supervisor binary path resolved")

		konaSupervisor := &KonaSupervisor{
			id:       supervisorID,
			userRPC:  "", // retrieved from logs
			execPath: execPath,
			args:     []string{},
			env:      envVars,
			p:        p,
		}
		orch.registry.Register(supervisorID, konaSupervisor)
		p.Logger().Info("Starting kona-supervisor")
		konaSupervisor.Start()
		p.Cleanup(konaSupervisor.Stop)
		p.Logger().Info("Kona-supervisor is up", "rpc", konaSupervisor.UserRPC())
	})
}
