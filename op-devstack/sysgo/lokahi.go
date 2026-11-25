package sysgo

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

// lokahiRPCBoundMessage is the log line lokahi emits once its authority RPC listens, with the
// bound address in its "addr" field.
const lokahiRPCBoundMessage = "RPC server bound to address"

// LokahiChain is one chain lokahi verifies: the chain's rollup config and the node it reads the
// chain's local-safe blocks from.
type LokahiChain struct {
	Network *L2Network
	// SourceCL serves superroot_atTimestamp from a safe head database.
	SourceCL L2CLNode
	// SourceEL serves the chain's blocks and receipts.
	SourceEL L2ELNode
}

// Lokahi is a lokahi process: the out-of-process interop verifier the chains' nodes follow.
//
// Its authority RPC is fronted by a proxy, so the URL the nodes are configured with is known
// before lokahi starts and stays the same across a restart. While lokahi is stopped the proxy
// has no upstream, and the nodes polling it see it as unreachable.
type Lokahi struct {
	mu sync.Mutex

	p        devtest.T
	execPath string
	args     []string
	// dataDir holds lokahi's store, which persists across Stop and Start.
	dataDir string

	userProxy *tcpproxy.Proxy
	userRPC   string

	sub *SubProcess
}

// newLokahi resolves the lokahi binary and opens the proxy its authority RPC is reached through.
func newLokahi(t devtest.T) *Lokahi {
	execPath, err := rustbin.Spec{
		SrcDir:  "rust",
		Package: "lokahi",
		Binary:  "lokahi",
	}.EnsureExists(t.Ctx(), t.Logger())
	t.Require().NoError(err, "prepare lokahi binary")

	proxy := tcpproxy.New(t.Logger().New("proxy", "lokahi"))
	t.Require().NoError(proxy.Start())
	t.Cleanup(func() { proxy.Close() })

	return &Lokahi{
		p:         t,
		execPath:  execPath,
		userProxy: proxy,
		userRPC:   "http://" + proxy.Addr(),
	}
}

// configure writes lokahi's configuration for the given chains.
func (l *Lokahi) configure(l1EL L1ELNode, depSetJSON []byte, chains []LokahiChain) {
	require := l.p.Require()
	dir := l.p.TempDirWithPrefix("lokahi")

	depSetPath := filepath.Join(dir, "depset.json")
	require.NoError(os.WriteFile(depSetPath, depSetJSON, 0o640))

	type chainFile struct {
		RollupConfig string `json:"rollup_config"`
		CLRPC        string `json:"cl_rpc"`
		ELRPC        string `json:"el_rpc"`
	}
	type configFile struct {
		L1RPC         string      `json:"l1_rpc"`
		RPCAddr       string      `json:"rpc_addr"`
		DependencySet string      `json:"dependency_set"`
		Chains        []chainFile `json:"chains"`
	}
	cfg := configFile{
		L1RPC:         l1EL.UserRPC(),
		RPCAddr:       "127.0.0.1:0",
		DependencySet: depSetPath,
	}
	for _, chain := range chains {
		rollupData, err := json.Marshal(chain.Network.rollupCfg)
		require.NoError(err, "marshal rollup config")
		rollupPath := filepath.Join(dir, "rollup-"+chain.Network.ChainID().String()+".json")
		require.NoError(os.WriteFile(rollupPath, rollupData, 0o640))
		cfg.Chains = append(cfg.Chains, chainFile{
			RollupConfig: rollupPath,
			CLRPC:        chain.SourceCL.UserRPC(),
			ELRPC:        chain.SourceEL.UserRPC(),
		})
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(err, "marshal lokahi config")
	configPath := filepath.Join(dir, "lokahi.json")
	require.NoError(os.WriteFile(configPath, data, 0o640))
	l.dataDir = filepath.Join(dir, "data")
	l.args = []string{"--config", configPath, "--datadir", l.dataDir}
}

// WipeData removes lokahi's store, so that the next Start verifies again from activation. lokahi
// must be stopped.
func (l *Lokahi) WipeData() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.p.Require().Nil(l.sub, "lokahi must be stopped to wipe its store")
	l.p.Require().NoError(os.RemoveAll(l.dataDir), "wipe lokahi's store")
}

// Start starts lokahi and points the proxy at its authority RPC once it listens.
func (l *Lokahi) Start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sub != nil {
		l.p.Logger().Warn("lokahi already started")
		return
	}
	logger := l.p.Logger().New("component", "lokahi")
	logOut := logpipe.ToLoggerWithMinLevel(logger.New("src", "stdout"), log.LevelInfo)
	logErr := logpipe.ToLoggerWithMinLevel(logger.New("src", "stderr"), log.LevelWarn)
	rpcAddr := make(chan string, 1)
	stdOut := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
		if e.LogMessage() == lokahiRPCBoundMessage {
			select {
			case rpcAddr <- "http://" + e.FieldValue("addr").(string):
			default:
			}
		}
	})
	stdErr := logpipe.LogCallback(func(line []byte) {
		logErr(logpipe.ParseRustStructuredLogs(line))
	})
	l.sub = NewSubProcess(l.p, stdOut, stdErr)
	l.p.Logger().Info("Starting lokahi", "exec", l.execPath, "args", l.args)
	l.p.Require().NoError(l.sub.Start(l.execPath, l.args, nil), "start lokahi")

	var addr string
	select {
	case addr = <-rpcAddr:
	case <-l.sub.Exited():
		l.p.Require().FailNow("lokahi exited before its RPC server became ready")
	case <-l.p.Ctx().Done():
		l.p.Require().NoError(l.p.Ctx().Err(), "need lokahi RPC")
	}
	l.userProxy.SetUpstream(ProxyAddr(l.p.Require(), addr))
	l.p.Logger().Info("lokahi is up", "rpc", l.userRPC, "upstream", addr)
}

// Stop stops lokahi. The proxy stays, without an upstream, so its URL keeps working after a
// later Start.
func (l *Lokahi) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sub == nil {
		return
	}
	l.userProxy.ClearUpstream()
	l.p.Require().NoError(l.sub.Stop(true), "stop lokahi")
	l.sub = nil
}

// StopControlled stops lokahi, for the generic lifecycle control.
func (l *Lokahi) StopControlled(_ context.Context) error {
	l.Stop()
	return nil
}

// Running reports whether the lokahi process runs.
func (l *Lokahi) Running() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.sub != nil
}

// UserRPC is lokahi's authority RPC, through the proxy.
func (l *Lokahi) UserRPC() string {
	return l.userRPC
}

// ExecPath is the lokahi binary the process runs.
func (l *Lokahi) ExecPath() string {
	return l.execPath
}
