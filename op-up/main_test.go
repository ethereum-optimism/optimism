package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestPresetsCommand(t *testing.T) {
	var stdout bytes.Buffer
	err := run(context.Background(), []string{"op-up", "presets"}, &stdout, io.Discard)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "minimal")
	require.Contains(t, stdout.String(), "interop")
	require.Contains(t, stdout.String(), "flashblocks")
}

func TestPresetAliases(t *testing.T) {
	spec, err := resolvePreset("two_l2_interop")
	require.NoError(t, err)
	require.Equal(t, "interop", spec.Name)

	spec, err = resolvePreset("multi-node")
	require.NoError(t, err)
	require.Equal(t, "multinode", spec.Name)
}

func TestUnknownPresetFailsBeforeStarting(t *testing.T) {
	err := run(context.Background(), []string{"op-up", "--dir", t.TempDir(), "--preset", "does-not-exist"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "unknown preset")
}

func TestGlobalFlagsApplyToUpCommand(t *testing.T) {
	err := run(context.Background(), []string{"op-up", "--preset", "does-not-exist", "up", "--l2-el-kind", "definitely-invalid"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "unknown preset")
	require.NotContains(t, err.Error(), "unsupported L2 EL kind")
}

func TestInteropCannotOverridePositionalPreset(t *testing.T) {
	err := run(context.Background(), []string{"op-up", "up", "--interop", "two-l2"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "--interop cannot be combined with preset argument two-l2")
}

func TestRuntimeFlagsBeforePresetAreParsed(t *testing.T) {
	err := run(context.Background(), []string{"op-up", "up", "--l2-el-kind", "definitely-invalid", "minimal"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "unsupported L2 EL kind")
}

func TestTempDirIncludesPresetName(t *testing.T) {
	tempRoot := t.TempDir()
	tt := newTestingT(context.Background(), io.Discard, tempRoot, "op-up-interop")
	defer tt.doCleanup()

	dir := tt.TempDir()
	require.DirExists(t, dir)
	require.Contains(t, filepath.Base(dir), "op-up-interop-")
}

func TestTempDirWithPrefixIncludesPresetAndServiceName(t *testing.T) {
	tempRoot := t.TempDir()
	tt := newTestingT(context.Background(), io.Discard, tempRoot, "op-up-interop")
	defer tt.doCleanup()

	dir := tt.TempDirWithPrefix("l2-el-sequencer-901")
	require.DirExists(t, dir)
	require.Contains(t, filepath.Base(dir), "op-up-interop-l2-el-sequencer-901-")
}

func TestCreateRunLogFile(t *testing.T) {
	cfg := opUpConfig{Dir: t.TempDir()}
	logFile, logFilePath, err := createRunLogFile(cfg, "op-up-minimal-test")
	require.NoError(t, err)
	defer logFile.Close()

	_, err = logFile.WriteString("hello\n")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(cfg.Dir, "logs", "op-up-minimal-test.log"), logFilePath)

	data, err := os.ReadFile(logFilePath)
	require.NoError(t, err)
	require.Equal(t, "hello\n", string(data))
	info, err := os.Stat(logFilePath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestNewLoggerWritesToFileSink(t *testing.T) {
	var stderr bytes.Buffer
	var logs bytes.Buffer
	logger := newLogger(context.Background(), &stderr, &logs)

	logger.Info("service started", "service", "l2-el")

	require.Contains(t, logs.String(), "service started")
	require.Contains(t, logs.String(), "l2-el")
	require.NotContains(t, stderr.String(), "service started")
}

func TestRPCProxyHandlerPreservesBackendPath(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/901", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"optimism":"1.0"}}`))
	}))
	defer backend.Close()

	handler, err := rpcProxyHandler(context.Background(), io.Discard, nil, backend.URL, nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "http://op-up.local/901", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"rpc_modules","params":[]}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":{"optimism":"1.0"}}`, rec.Body.String())
}

func TestSmokeInteropRequiresRPCFlags(t *testing.T) {
	err := run(context.Background(), []string{"op-up", "smoke-interop", "identity"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "provide --l2a-rpc and --l2b-rpc")
}

func TestLocalEndpointsUseRandomPortsByDefault(t *testing.T) {
	devnet := &runningDevnet{
		L1CL: &devnetEndpoint{
			Name:      "L1 Beacon",
			Layer:     "HTTP",
			ChainID:   eth.ChainIDFromUInt64(900),
			DirectURL: "http://127.0.0.1:12345",
		},
		L2ELs: []*devnetEndpoint{{
			Name:    "L2",
			Layer:   "EL",
			ChainID: eth.ChainIDFromUInt64(901),
		}},
		Services: []*devnetEndpoint{{
			Name:       "Aux RPC",
			Layer:      "RPC",
			ChainLabel: "shared",
		}},
	}

	endpoints, err := devnet.localEndpoints()
	require.NoError(t, err)
	defer closeLocalEndpointListeners(endpoints)
	require.Len(t, endpoints, 3)

	for _, endpoint := range endpoints {
		if endpoint.DirectURL != "" {
			require.Nil(t, endpoint.Listener)
			require.Equal(t, endpoint.DirectURL, endpoint.LocalURL)
			continue
		}
		boundPort := uint(endpoint.Listener.Addr().(*net.TCPAddr).Port)
		require.NotZero(t, boundPort)
		require.Equal(t, localURL(boundPort), endpoint.LocalURL)
	}
}

func TestExportDevnetConfigsWritesStandardFiles(t *testing.T) {
	tempRoot := t.TempDir()
	generatedDir := filepath.Join(tempRoot, "op-up-interop-l2-el-sequencer-901-123")
	require.NoError(t, os.MkdirAll(generatedDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(generatedDir, "genesis.json"), []byte(`{"config":{"chainId":901}}`), 0o600))

	cfg := opUpConfig{Dir: t.TempDir()}
	spec := &devnetPreset{Name: "interop"}
	devnet := &runningDevnet{
		ExportDepset: true,
		Contracts: []*contractSet{{
			Network: "L2A",
			ChainID: eth.ChainIDFromUInt64(901),
			Contracts: []contractAddress{{
				Name:    "OptimismPortalProxy",
				Address: common.HexToAddress("0x1234"),
			}},
		}},
	}
	endpoints := []*localEndpoint{{
		devnetEndpoint: &devnetEndpoint{
			Name:    "L2A CL",
			Layer:   "CL",
			ChainID: eth.ChainIDFromUInt64(901),
			RPC:     staticRawRPC{raw: json.RawMessage(`{"dependencies":{"901":{}}}`)},
		},
		LocalURL: "http://localhost:12345",
	}}

	export, err := exportDevnetConfigs(context.Background(), cfg, spec, tempRoot, devnet, endpoints)
	require.NoError(t, err)
	require.Contains(t, export.Dir, filepath.Join(cfg.Dir, "configs", "op-up-interop-"))
	require.FileExists(t, filepath.Join(export.Dir, "manifest.json"))
	require.FileExists(t, filepath.Join(export.Dir, "endpoints.json"))
	require.FileExists(t, filepath.Join(export.Dir, "contracts.json"))
	require.FileExists(t, filepath.Join(export.Dir, "depset.json"))
	require.FileExists(t, filepath.Join(export.Dir, "generated", "op-up-interop-l2-el-sequencer-901-123", "genesis.json"))

	var manifest configExportManifest
	manifestData, err := os.ReadFile(filepath.Join(export.Dir, "manifest.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(manifestData, &manifest))
	require.Equal(t, "interop", manifest.Preset)
	require.Contains(t, manifest.Files, configExportFile{Path: "depset.json", Kind: "dependency-set", Source: "optimism_dependencySet"})
}

func TestControlledServiceStopFailureBlocksStartAndAllowsRetry(t *testing.T) {
	lifecycle := &fakeControlledLifecycle{running: true, stopErr: errors.New("stop failed")}
	svc := &controlledService{ID: "l2-el", Name: "L2 EL", Kind: "EL", Control: lifecycle}
	svc.initState()

	err := svc.stop(context.Background())
	require.ErrorContains(t, err, "stop failed")
	require.Equal(t, serviceStateStopFailed, svc.status(context.Background()).State)

	err = svc.start(context.Background())
	require.ErrorContains(t, err, "cannot start from state stop_failed")

	lifecycle.stopErr = nil
	err = svc.stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, serviceStateStopped, svc.status(context.Background()).State)

	err = svc.start(context.Background())
	require.NoError(t, err)
	require.Equal(t, serviceStateRunning, svc.status(context.Background()).State)
}

func TestControlledServiceRestartSkipsStartAfterStopFailure(t *testing.T) {
	lifecycle := &fakeControlledLifecycle{running: true, stopErr: errors.New("stop failed")}
	svc := &controlledService{ID: "l2-cl", Name: "L2 CL", Kind: "CL", Control: lifecycle}
	svc.initState()

	err := svc.restart(context.Background())
	require.ErrorContains(t, err, "stop failed")
	require.Equal(t, serviceStateStopFailed, svc.status(context.Background()).State)
	require.Zero(t, lifecycle.startCalls)
}

func TestControlledServiceStartTimeoutStaysStartingUntilLateResult(t *testing.T) {
	startBlock := make(chan struct{})
	lifecycle := &fakeControlledLifecycle{startBlock: startBlock}
	svc := &controlledService{ID: "l2-cl", Name: "L2 CL", Kind: "CL", Control: lifecycle}
	svc.initState()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := svc.start(ctx)
	require.ErrorIs(t, err, errControlOperationPending)
	status := svc.status(context.Background())
	require.Equal(t, serviceStateStarting, status.State)
	require.Contains(t, status.LastError, "start still in progress")

	close(startBlock)
	require.Eventually(t, func() bool {
		return svc.status(context.Background()).State == serviceStateRunning
	}, time.Second, 10*time.Millisecond)
}

func TestControlledServiceStopTimeoutAppliesLateSuccess(t *testing.T) {
	stopBlock := make(chan struct{})
	lifecycle := &fakeControlledLifecycle{running: true, stopBlock: stopBlock}
	svc := &controlledService{ID: "l2-cl", Name: "L2 CL", Kind: "CL", Control: lifecycle}
	svc.initState()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := svc.stop(ctx)
	require.Error(t, err)
	status := svc.status(context.Background())
	require.Equal(t, serviceStateStopFailed, status.State)

	close(stopBlock)
	require.Eventually(t, func() bool {
		return svc.status(context.Background()).State == serviceStateStopped
	}, time.Second, 10*time.Millisecond)
}

func TestControlServerRequiresAuth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc := &controlledService{ID: "l2-el", Name: "L2 EL", Kind: "EL", Control: &fakeControlledLifecycle{running: true}}
	server, err := newControlServer(ctx, opUpConfig{Dir: t.TempDir()}, &devnetPreset{Name: "minimal"}, []*controlledService{svc}, "op-up-minimal-test", "/tmp/op-up-minimal-test.log")
	require.NoError(t, err)
	defer server.Close()

	resp, err := http.Get(server.session.ControlURL + "/services")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	controlResp, err := callControlServer(ctx, server.session, "services", "")
	require.NoError(t, err)
	require.Len(t, controlResp.Services, 1)
	require.Equal(t, "l2-el", controlResp.Services[0].ID)
}

func TestControlCLIAliasesAndUnknownService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	server, err := newControlServer(ctx, opUpConfig{Dir: dir}, &devnetPreset{Name: "minimal"}, []*controlledService{
		{ID: "l2-el", Name: "L2 EL", Kind: "EL", Control: &fakeControlledLifecycle{running: true}},
	}, "op-up-minimal-test", "/tmp/op-up-minimal-test.log")
	require.NoError(t, err)
	defer server.Close()

	for _, alias := range []string{"services", "ls", "list", "ps"} {
		var stdout bytes.Buffer
		err := run(ctx, []string{"op-up", "control", alias, "--dir", dir, "--session", server.session.ID}, &stdout, io.Discard)
		require.NoError(t, err)
		require.Contains(t, stdout.String(), "Session: "+server.session.ID)
		require.Contains(t, stdout.String(), "Logs: /tmp/op-up-minimal-test.log")
		require.Contains(t, stdout.String(), "l2-el")
	}

	var stdout bytes.Buffer
	err = run(ctx, []string{"op-up", "control", "status", "--dir", dir, "--session", server.session.ID, "missing"}, &stdout, io.Discard)
	require.ErrorContains(t, err, "unknown service id missing")
	require.Contains(t, stdout.String(), "Session: "+server.session.ID)
}

func TestSelectControlSessionDefaultsToNewestLiveSession(t *testing.T) {
	dir := t.TempDir()
	oldCtx, oldCancel := context.WithCancel(context.Background())
	defer oldCancel()
	oldServer, err := newControlServer(oldCtx, opUpConfig{Dir: dir}, &devnetPreset{Name: "minimal"}, []*controlledService{
		{ID: "l2-el", Name: "L2 EL", Kind: "EL", Control: &fakeControlledLifecycle{running: true}},
	}, "op-up-minimal-old", "/tmp/op-up-minimal-old.log")
	require.NoError(t, err)
	defer oldServer.Close()

	newCtx, newCancel := context.WithCancel(context.Background())
	defer newCancel()
	newServer, err := newControlServer(newCtx, opUpConfig{Dir: dir}, &devnetPreset{Name: "interop"}, []*controlledService{
		{ID: "l2a-el", Name: "L2A EL", Kind: "EL", Control: &fakeControlledLifecycle{running: true}},
	}, "op-up-interop-new", "/tmp/op-up-interop-new.log")
	require.NoError(t, err)
	defer newServer.Close()

	controlDir := filepath.Join(dir, "control")
	require.NoError(t, os.Remove(oldServer.file))
	require.NoError(t, os.Remove(newServer.file))
	oldSession := oldServer.session
	oldSession.ID = "old"
	oldSession.StartedAt = "2026-05-26T10:00:00Z"
	newSession := newServer.session
	newSession.ID = "new"
	newSession.StartedAt = "2026-05-26T11:00:00Z"
	staleSession := controlSessionMetadata{
		ID:         "stale",
		Preset:     "interop",
		PID:        os.Getpid(),
		StartedAt:  "2026-05-26T12:00:00Z",
		ControlURL: "http://127.0.0.1:3",
		Token:      "stale-token",
	}
	require.NoError(t, writeControlSessionFile(filepath.Join(controlDir, "old.json"), oldSession))
	require.NoError(t, writeControlSessionFile(filepath.Join(controlDir, "new.json"), newSession))
	require.NoError(t, writeControlSessionFile(filepath.Join(controlDir, "stale.json"), staleSession))

	selected, err := selectControlSession(dir, "")
	require.NoError(t, err)
	require.Equal(t, "new", selected.ID)

	selected, err = selectControlSession(dir, "old")
	require.NoError(t, err)
	require.Equal(t, "old", selected.ID)
}

func TestRun(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stderr lockedBuffer
	errCh := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(errCh)
		if err := run(ctx, []string{"op-up", "--dir", t.TempDir(), "--l2-el-kind", "op-geth"}, io.Discard, &stderr); err != nil {
			errCh <- err
		}
	}()

	ticker := time.NewTicker(time.Millisecond * 250)
	defer ticker.Stop()
	for {
		select {
		case e := <-errCh:
			require.NoError(t, e)
		case <-ticker.C:
			url := firstL2ELURL(stderr.String())
			if url == "" {
				continue
			}
			client, err := ethclient.DialContext(ctx, url)
			require.NoError(t, err)
			chainID, err := client.ChainID(ctx)
			if err != nil {
				t.Logf("error while querying chain ID, will retry: %s", err)
				continue
			}
			require.Equal(t, sysgo.DefaultL2AID.ToBig(), chainID)
			return
		}
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

var l2ELURLPattern = regexp.MustCompile(`(?m)^\s+L2\s+EL\s+control=\S+\s+chain=\S+\s+(http://localhost:\d+)\s*$`)

func firstL2ELURL(output string) string {
	match := l2ELURLPattern.FindStringSubmatch(output)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

type staticRawRPC struct {
	raw json.RawMessage
}

func (s staticRawRPC) Close() {}

func (s staticRawRPC) CallContext(_ context.Context, result any, method string, _ ...any) error {
	if method != "optimism_dependencySet" {
		return errors.New("unexpected method")
	}
	out, ok := result.(*json.RawMessage)
	if !ok {
		return errors.New("unexpected result type")
	}
	*out = append((*out)[:0], s.raw...)
	return nil
}

func (s staticRawRPC) BatchCallContext(context.Context, []rpc.BatchElem) error {
	return errors.New("unexpected batch call")
}

func (s staticRawRPC) Subscribe(context.Context, string, any, ...any) (ethereum.Subscription, error) {
	return nil, errors.New("unexpected subscription")
}

type fakeControlledLifecycle struct {
	mu         sync.Mutex
	running    bool
	startErr   error
	stopErr    error
	startBlock <-chan struct{}
	stopBlock  <-chan struct{}
	startCalls int
	stopCalls  int
}

func (f *fakeControlledLifecycle) StartControlled(context.Context) error {
	f.mu.Lock()
	f.startCalls++
	startBlock := f.startBlock
	f.mu.Unlock()
	if startBlock != nil {
		<-startBlock
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.startErr != nil {
		return f.startErr
	}
	f.running = true
	return nil
}

func (f *fakeControlledLifecycle) StopControlled(context.Context) error {
	f.mu.Lock()
	f.stopCalls++
	stopBlock := f.stopBlock
	f.mu.Unlock()
	if stopBlock != nil {
		<-stopBlock
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stopErr != nil {
		return f.stopErr
	}
	f.running = false
	return nil
}

func (f *fakeControlledLifecycle) Running() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}
