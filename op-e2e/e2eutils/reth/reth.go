// Package reth runs op-reth as an external subprocess and exposes it as a
// services.EthInstance, so op-e2e can use op-reth as the L2 execution layer.
// The launch/genesis/JWT/readiness machinery mirrors op-devstack/sysgo, which
// already drives op-reth this way.
package reth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
)

const (
	proofsHistoryVersionV1      = "v1"
	defaultProofsHistoryVersion = "v2"
)

// Config carries the op-e2e-level EL knobs that translate onto the op-reth CLI.
type Config struct {
	// SequencerHTTP, when non-empty, wires op-reth to forward transactions to the
	// sequencer via --rollup.sequencer-http (sentry tx-forwarding).
	SequencerHTTP string
	// ProofsHistoryVersion selects the proof-history storage version; defaults to v2.
	ProofsHistoryVersion string
	// DataDir is the base directory for the op-reth datadir/logs/proof-history.
	// Callers should pass t.TempDir() so the test framework owns cleanup even if
	// the test panics before Close.
	DataDir string
	// ExtraArgs are appended verbatim to the op-reth `node` invocation.
	ExtraArgs []string
}

// Instance is a running op-reth process exposed through services.EthInstance.
type Instance struct {
	logger log.Logger

	userRPC endpoint.RPC
	authRPC endpoint.RPC

	cmd     *exec.Cmd
	exited  chan struct{}
	waitErr error

	stdoutPipe *logpipe.LineBuffer
	stderrPipe *logpipe.LineBuffer
}

var _ services.EthInstance = (*Instance)(nil)

func (i *Instance) UserRPC() endpoint.RPC { return i.userRPC }

func (i *Instance) AuthRPC() endpoint.RPC { return i.authRPC }

// Close interrupts the process, waits for it to exit, and closes the log pipes.
func (i *Instance) Close() error {
	var errs []error
	if i.cmd != nil && i.cmd.Process != nil {
		select {
		case <-i.exited:
			// already exited; nothing to interrupt
		default:
			if err := i.cmd.Process.Signal(os.Interrupt); err != nil {
				i.logger.Warn("failed to interrupt op-reth", "err", err)
			}
		}
		<-i.exited
		// An interrupted process exits non-zero; that's expected, not an error.
		var exitErr *exec.ExitError
		if i.waitErr != nil && !errors.As(i.waitErr, &exitErr) {
			errs = append(errs, fmt.Errorf("op-reth wait: %w", i.waitErr))
		}
	}
	if i.stdoutPipe != nil {
		_ = i.stdoutPipe.Close()
	}
	if i.stderrPipe != nil {
		_ = i.stderrPipe.Close()
	}
	return errors.Join(errs...)
}

// InitL2 resolves the op-reth binary, initializes a chain + proof-history DB
// from the given genesis, starts an op-reth node, and waits for its RPCs to come
// up. cfg.DataDir must be created with t.TempDir() by the caller.
func InitL2(ctx context.Context, lgr log.Logger, name string, genesis *core.Genesis, jwtPath string, cfg Config) (*Instance, error) {
	proofsVersion := cfg.ProofsHistoryVersion
	if proofsVersion == "" {
		proofsVersion = defaultProofsHistoryVersion
	}

	execPath, err := rustbin.Spec{
		SrcDir:  "rust",
		Package: "op-reth",
		Binary:  "op-reth",
	}.EnsureExists(ctx, lgr)
	if err != nil {
		return nil, fmt.Errorf("op-reth binary not available: %w", err)
	}

	baseDir := cfg.DataDir
	if baseDir == "" {
		return nil, errors.New("op-reth DataDir must be set with t.TempDir()")
	}

	dataDir := filepath.Join(baseDir, "data")
	logDir := filepath.Join(baseDir, "logs")
	proofHistoryDir := filepath.Join(baseDir, "proof-history")
	chainConfigPath := filepath.Join(baseDir, "genesis.json")

	for _, dir := range []string{dataDir, logDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	genesisData, err := json.Marshal(genesis)
	if err != nil {
		return nil, fmt.Errorf("marshal genesis: %w", err)
	}
	if err := os.WriteFile(chainConfigPath, genesisData, 0o600); err != nil {
		return nil, fmt.Errorf("write genesis: %w", err)
	}

	if err := runToCompletion(ctx, execPath,
		"init",
		"--datadir="+dataDir,
		"--chain="+chainConfigPath,
	); err != nil {
		return nil, fmt.Errorf("op-reth init: %w", err)
	}

	proofsInitArgs := []string{
		"proofs", "init",
		"--datadir=" + dataDir,
		"--chain=" + chainConfigPath,
		"--proofs-history.storage-path=" + proofHistoryDir,
		"--proofs-history.storage-version=" + proofsVersion,
	}
	// `proofs init` runs snapshot-accelerated backfill by default, which v1
	// storage does not support, so opt out explicitly for v1 (mirrors sysgo).
	if proofsVersion == proofsHistoryVersionV1 {
		proofsInitArgs = append(proofsInitArgs, "--proofs-history.skip-backfill")
	}
	if err := runToCompletion(ctx, execPath, proofsInitArgs...); err != nil {
		return nil, fmt.Errorf("op-reth proofs init: %w", err)
	}

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
		"--datadir=" + dataDir,
		"--disable-discovery",
		"--http",
		"--http.api=admin,debug,eth,net,trace,txpool,web3,rpc,reth,miner",
		"--http.addr=127.0.0.1",
		"--http.port=0",
		"--ipcdisable",
		"--log.file.directory=" + logDir,
		"--log.stdout.format=json",
		"--port=0",
		"--proofs-history",
		"--proofs-history.window=10000",
		"--proofs-history.storage-path=" + proofHistoryDir,
		"--proofs-history.storage-version=" + proofsVersion,
		"--with-unused-ports",
		"--ws",
		"--ws.api=admin,debug,eth,net,trace,txpool,web3,rpc,reth,miner",
		"--ws.addr=127.0.0.1",
		"--ws.port=0",
		"-vvvv",
	}
	if cfg.SequencerHTTP != "" {
		args = append(args, "--rollup.sequencer-http="+cfg.SequencerHTTP)
	}
	args = append(args, cfg.ExtraArgs...)

	inst := &Instance{
		logger: lgr,
	}

	// op-reth reports the ephemeral http/ws/auth ports via structured logs once
	// the servers bind; capture the URLs from those readiness lines.
	userHTTPRPCChan := make(chan string, 1)
	userWSRPCChan := make(chan string, 1)
	authRPCChan := make(chan string, 1)
	onLogEntry := func(e logpipe.LogEntry) {
		switch e.LogMessage() {
		case "RPC HTTP server started":
			if u, ok := e.FieldValue("url").(string); ok {
				select {
				case userHTTPRPCChan <- u:
				default:
				}
			}
		case "RPC WS server started":
			if u, ok := e.FieldValue("url").(string); ok {
				select {
				case userWSRPCChan <- u:
				default:
				}
			}
		case "RPC auth server started":
			if u, ok := e.FieldValue("url").(string); ok {
				select {
				case authRPCChan <- u:
				default:
				}
			}
		}
	}

	logOut := logpipe.ToLoggerWithMinLevel(lgr.New("component", "op-reth", "src", "stdout", "name", name), slog.LevelInfo)
	logErr := logpipe.ToLoggerWithMinLevel(lgr.New("component", "op-reth", "src", "stderr", "name", name), slog.LevelWarn)
	inst.stdoutPipe = logpipe.NewLineBuffer(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
	})
	inst.stderrPipe = logpipe.NewLineBuffer(func(line []byte) {
		logErr(logpipe.ParseRustStructuredLogs(line))
	})

	cmd := exec.Command(execPath, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = inst.stdoutPipe
	cmd.Stderr = inst.stderrPipe
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start op-reth node: %w", err)
	}
	inst.cmd = cmd
	inst.exited = make(chan struct{})
	go func() {
		inst.waitErr = cmd.Wait()
		close(inst.exited)
	}()

	// From here on, Close() owns process cleanup.
	closeOnErr := func(cause error) (*Instance, error) {
		_ = inst.Close()
		return nil, cause
	}

	var userHTTPURL, userWSURL, authURL string
	if err := tasks.Await(ctx, userHTTPRPCChan, &userHTTPURL); err != nil {
		return closeOnErr(fmt.Errorf("waiting for op-reth user HTTP RPC: %w", err))
	}
	if err := tasks.Await(ctx, userWSRPCChan, &userWSURL); err != nil {
		return closeOnErr(fmt.Errorf("waiting for op-reth user WS RPC: %w", err))
	}
	if err := tasks.Await(ctx, authRPCChan, &authURL); err != nil {
		return closeOnErr(fmt.Errorf("waiting for op-reth auth RPC: %w", err))
	}

	inst.userRPC = endpoint.WsOrHttpRPC{
		WsURL:   "ws://" + userWSURL,
		HttpURL: "http://" + userHTTPURL,
	}
	inst.authRPC = endpoint.WsOrHttpRPC{
		WsURL:   "ws://" + authURL,
		HttpURL: "http://" + authURL,
	}

	client, err := ethclient.DialContext(ctx, endpoint.SelectRPC(endpoint.PreferHttpRPC, inst.userRPC))
	if err != nil {
		return closeOnErr(fmt.Errorf("dial op-reth user RPC: %w", err))
	}
	defer client.Close()
	if err := wait.ForNodeUp(ctx, client, lgr); err != nil {
		return closeOnErr(fmt.Errorf("op-reth did not come up: %w", err))
	}

	lgr.Info("op-reth is ready", "name", name, "userHTTPRPC", userHTTPURL, "userWSRPC", userWSURL, "authRPC", authURL)
	return inst, nil
}

func runToCompletion(ctx context.Context, execPath string, args ...string) error {
	out, err := exec.CommandContext(ctx, execPath, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}
