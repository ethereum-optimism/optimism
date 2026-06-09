package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const asciiArt = ` ____  ____        _     ____
/  _ \/  __\      / \ /\/  __\
| / \||  \/|_____ | | |||  \/|
| \_/||  __/\____\| \_/||  __/
\____/\_/         \____/\_/`

var (
	Version     = "v0.0.0"
	VersionMeta = "dev"
	GitCommit   string
	GitDate     string

	envPrefix = "OP_UP"
	dirFlag   = &cli.PathFlag{
		Name:    "dir",
		Usage:   "the path to the op-up directory, which is used for caching among other things.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "DIR"),
		Value: func() string {
			parentDir, err := os.UserHomeDir()
			if err != nil {
				parentDir, err = os.Getwd()
				if err != nil {
					return "error: could not find home or working directories"
				}
			}
			return filepath.Join(parentDir, ".op-up")
		}(),
	}
	presetFlag = &cli.StringFlag{
		Name:    "preset",
		Aliases: []string{"p"},
		Usage:   "devnet preset to start. Run `op-up presets` to list available presets.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "PRESET"),
		Value:   defaultPresetName,
	}
	interopFlag = &cli.BoolFlag{
		Name:    "interop",
		Usage:   "alias for --preset interop.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "INTEROP"),
	}
	l2ELKindFlag = &cli.StringFlag{
		Name:    "l2-el-kind",
		Usage:   "override the L2 EL implementation: op-reth, op-reth-proof-v2, or op-geth.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "L2_EL_KIND"),
	}
	l2CLKindFlag = &cli.StringFlag{
		Name:    "l2-cl-kind",
		Usage:   "override the L2 CL implementation: op-node or kona-node.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "L2_CL_KIND"),
	}
	metricsFlag = &cli.BoolFlag{
		Name:    "metrics",
		Usage:   "enable sysgo metrics and dashboards for supported components.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "METRICS"),
	}
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	app := cli.NewApp()
	app.Writer = stdout
	app.ErrWriter = stderr
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, VersionMeta)
	app.Name = "op-up"
	app.Usage = "starts op-devstack devnet presets with printed local RPC endpoints."
	app.Flags = opUpFlags()
	// The default OnUsageError behavior will print the error twice: once in the cli package and
	// once in our main function.
	// The function below prints help and returns the error for further handling/error messages.
	app.OnUsageError = func(cliCtx *cli.Context, err error, isSubcommand bool) error {
		if !cliCtx.App.HideHelp {
			_ = cli.ShowAppHelp(cliCtx)
		}
		return err
	}
	app.Action = func(cliCtx *cli.Context) error {
		cfg, err := opUpConfigFromCLI(cliCtx)
		if err != nil {
			return err
		}
		return runOpUp(cliCtx.Context, cliCtx.App.Writer, cliCtx.App.ErrWriter, cfg)
	}
	app.Commands = []*cli.Command{
		upCommand(),
		controlCommand(),
		presetsCommand(),
		smokeCommand(),
	}
	return app.RunContext(ctx, args)
}

func opUpFlags() []cli.Flag {
	return cliapp.ProtectFlags([]cli.Flag{
		dirFlag,
		presetFlag,
		interopFlag,
		l2ELKindFlag,
		l2CLKindFlag,
		metricsFlag,
	})
}

func upCommand() *cli.Command {
	return &cli.Command{
		Name:      "up",
		Aliases:   []string{"run", "deploy"},
		Usage:     "start a devnet preset",
		ArgsUsage: "[preset]",
		Flags:     opUpFlags(),
		Action: func(cliCtx *cli.Context) error {
			cfg, err := opUpConfigFromCLI(cliCtx)
			if err != nil {
				return err
			}
			return runOpUp(cliCtx.Context, cliCtx.App.Writer, cliCtx.App.ErrWriter, cfg)
		},
	}
}

func presetsCommand() *cli.Command {
	return &cli.Command{
		Name:    "presets",
		Aliases: []string{"list"},
		Usage:   "list available devnet presets",
		Action: func(cliCtx *cli.Context) error {
			return printPresetList(cliCtx.App.Writer)
		},
	}
}

func opUpConfigFromCLI(cliCtx *cli.Context) (opUpConfig, error) {
	cfg := opUpConfig{
		Dir:           cliPath(cliCtx, dirFlag.Name),
		Preset:        cliString(cliCtx, presetFlag.Name),
		LegacyInterop: cliBool(cliCtx, interopFlag.Name),
		L2ELKind:      normalizePresetName(cliString(cliCtx, l2ELKindFlag.Name)),
		L2CLKind:      normalizePresetName(cliString(cliCtx, l2CLKindFlag.Name)),
		Metrics:       cliBool(cliCtx, metricsFlag.Name),
	}
	if cliCtx.NArg() > 1 {
		return cfg, fmt.Errorf("expected at most one preset argument, got %d", cliCtx.NArg())
	}
	if cliCtx.NArg() == 1 {
		if cliIsSet(cliCtx, presetFlag.Name) {
			return cfg, fmt.Errorf("provide a preset either as an argument or with --preset, not both")
		}
		cfg.Preset = cliCtx.Args().First()
	}
	if cfg.LegacyInterop {
		if cliCtx.NArg() == 1 && normalizePresetName(cliCtx.Args().First()) != "interop" {
			return cfg, fmt.Errorf("--interop cannot be combined with preset argument %s", cliCtx.Args().First())
		}
		if cliIsSet(cliCtx, presetFlag.Name) && normalizePresetName(cfg.Preset) != defaultPresetName && normalizePresetName(cfg.Preset) != "interop" {
			return cfg, fmt.Errorf("--interop cannot be combined with --preset %s", cfg.Preset)
		}
		cfg.Preset = "interop"
	}
	return cfg, nil
}

func cliString(cliCtx *cli.Context, name string) string {
	for _, ctx := range cliCtx.Lineage() {
		if ctx.IsSet(name) {
			return ctx.String(name)
		}
	}
	return cliCtx.String(name)
}

func cliPath(cliCtx *cli.Context, name string) string {
	for _, ctx := range cliCtx.Lineage() {
		if ctx.IsSet(name) {
			return ctx.Path(name)
		}
	}
	return cliCtx.Path(name)
}

func cliBool(cliCtx *cli.Context, name string) bool {
	for _, ctx := range cliCtx.Lineage() {
		if ctx.IsSet(name) {
			return ctx.Bool(name)
		}
	}
	return cliCtx.Bool(name)
}

func cliIsSet(cliCtx *cli.Context, name string) bool {
	for _, ctx := range cliCtx.Lineage() {
		if ctx.IsSet(name) {
			return true
		}
	}
	return false
}

func runOpUp(ctx context.Context, stdout io.Writer, stderr io.Writer, cfg opUpConfig) error {
	fmt.Fprintf(stderr, "%s\n", asciiArt)

	spec, err := resolvePreset(cfg.Preset)
	if err != nil {
		return err
	}
	restoreEnv, err := applyRuntimeOverrides(cfg)
	if err != nil {
		return err
	}
	defer restoreEnv()
	logRuntimeOverrides(stderr, cfg)

	if err := os.MkdirAll(cfg.Dir, 0o700); err != nil {
		return fmt.Errorf("create the op-up dir: %w", err)
	}
	if err := os.Chmod(cfg.Dir, 0o700); err != nil {
		return fmt.Errorf("restrict the op-up dir: %w", err)
	}
	tempRoot := filepath.Join(cfg.Dir, "tmp")
	if err := os.MkdirAll(tempRoot, 0o700); err != nil {
		return fmt.Errorf("create the op-up temp dir: %w", err)
	}
	if err := os.Chmod(tempRoot, 0o700); err != nil {
		return fmt.Errorf("restrict the op-up temp dir: %w", err)
	}

	runID := newOpUpRunID(spec.Name)
	logFile, logFilePath, err := createRunLogFile(cfg, runID)
	if err != nil {
		return err
	}
	defer logFile.Close()
	fmt.Fprintf(stderr, "\nLogs: %s\n", logFilePath)

	devtest.RootContext = ctx
	t := newTestingT(ctx, stderr, tempRoot, fmt.Sprintf("op-up-%s", spec.Name), logFile)
	defer t.doCleanup()

	devnet, err := spec.Build(t)
	if err != nil {
		return err
	}
	if err := runSelectedDevnet(ctx, stdout, stderr, cfg, spec, tempRoot, devnet, runID, logFilePath); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "\nPlease consider filling out this survey to influence future development: https://www.surveymonkey.com/r/JTGHFK3\n")
	return nil
}

func newOpUpRunID(preset string) string {
	return fmt.Sprintf("op-up-%s-%s-%d", tempDirSafePrefix(preset), time.Now().UTC().Format("20060102T150405Z"), os.Getpid())
}

func createRunLogFile(cfg opUpConfig, runID string) (*os.File, string, error) {
	logDir := filepath.Join(cfg.Dir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, "", fmt.Errorf("create log dir: %w", err)
	}
	if err := os.Chmod(logDir, 0o700); err != nil {
		return nil, "", fmt.Errorf("restrict log dir: %w", err)
	}
	logPath := filepath.Join(logDir, runID+".log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, "", fmt.Errorf("open run log: %w", err)
	}
	return file, logPath, nil
}

func newLogger(ctx context.Context, stderr io.Writer, logWriters ...io.Writer) log.Logger {
	terminalHandler := oplog.NewLogHandler(stderr, oplog.DefaultCLIConfig())
	terminalHandler = logfilter.WrapFilterHandler(terminalHandler)
	terminalHandler.(logfilter.FilterHandler).Set(logfilter.DefaultMute())

	handlers := []slog.Handler{terminalHandler}
	for _, logWriter := range logWriters {
		if logWriter == nil {
			continue
		}
		cfg := oplog.DefaultCLIConfig()
		cfg.Color = false
		handlers = append(handlers, oplog.NewLogHandler(logWriter, cfg))
	}

	logHandler := newTeeLogHandler(handlers...)
	logHandler = logfilter.WrapContextHandler(logHandler)
	logger := log.NewLogger(logHandler)
	oplog.SetGlobalLogHandler(logHandler)
	logger.SetContext(ctx)
	return logger
}

type teeLogHandler struct {
	handlers []slog.Handler
}

func newTeeLogHandler(handlers ...slog.Handler) slog.Handler {
	out := make([]slog.Handler, 0, len(handlers))
	for _, handler := range handlers {
		if handler != nil {
			out = append(out, handler)
		}
	}
	return teeLogHandler{handlers: out}
}

func (h teeLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h teeLogHandler) Handle(ctx context.Context, record slog.Record) error {
	var result error
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		result = errors.Join(result, handler.Handle(ctx, record.Clone()))
	}
	return result
}

func (h teeLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithAttrs(attrs))
	}
	return teeLogHandler{handlers: next}
}

func (h teeLogHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithGroup(name))
	}
	return teeLogHandler{handlers: next}
}

func newMinimalSystem(t *testingT) (sys *presets.Minimal, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			var failure testingFailure
			if errors.As(asError(recovered), &failure) {
				err = failure.err
				return
			}
			panic(recovered)
		}
	}()
	return presets.NewMinimal(t), nil
}

func newSupernodeInteropSystem(t *testingT) (sys *presets.TwoL2SupernodeInterop, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			var failure testingFailure
			if errors.As(asError(recovered), &failure) {
				err = failure.err
				return
			}
			panic(recovered)
		}
	}()
	return presets.NewTwoL2SupernodeInterop(t, defaultInteropActivationSecs,
		presets.WithSuggestedInteropActivationOffset(defaultInteropActivationSecs),
	), nil
}

func printAccountInfo(stderr io.Writer) error {
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	if err != nil {
		return fmt.Errorf("new mnemonic dev keys: %w", err)
	}
	const funderIndex = 10_000
	funderUserKey := devkeys.UserKey(funderIndex)
	funderAddress, err := hd.Address(funderUserKey)
	if err != nil {
		return fmt.Errorf("address: %w", err)
	}
	funderPrivKey, err := hd.Secret(funderUserKey)
	if err != nil {
		return fmt.Errorf("secret: %w", err)
	}

	fmt.Fprintf(stderr, "Test Account Address: %s\n", funderAddress)
	fmt.Fprintf(stderr, "Test Account Private Key: %s\n", "0x"+common.Bytes2Hex(crypto.FromECDSA(funderPrivKey)))
	return nil
}

func logInterop(ctx context.Context, stderr io.Writer, sys *presets.TwoL2SupernodeInterop) {
	const pollInterval = 2 * time.Second
	queryAPI := sys.Supernode.QueryAPI()

	var lastSafeTS, lastLocalSafeTS uint64
	lastSafe := make(map[string]uint64)
	lastLocalSafe := make(map[string]uint64)
	lastCrossUnsafe := make(map[string]uint64)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
			status, err := queryAPI.SyncStatus(ctx)
			if err != nil {
				continue
			}

			// Global safe timestamp
			if status.SafeTimestamp != lastSafeTS && status.SafeTimestamp > 0 {
				fmt.Fprintf(stderr, "[interop] Cross-safe timestamp: %d\n", status.SafeTimestamp)
				lastSafeTS = status.SafeTimestamp
			}

			// Global local-safe timestamp
			if status.LocalSafeTimestamp != lastLocalSafeTS && status.LocalSafeTimestamp > 0 {
				fmt.Fprintf(stderr, "[interop] Local-safe timestamp: %d\n", status.LocalSafeTimestamp)
				lastLocalSafeTS = status.LocalSafeTimestamp
			}

			// Per-chain details
			for chainID, cs := range status.Chains {
				id := chainID.String()

				// Cross-unsafe progression
				if cs.CrossUnsafeL2.Number != lastCrossUnsafe[id] && cs.CrossUnsafeL2.Number > 0 {
					fmt.Fprintf(stderr, "[interop] Chain %s cross-unsafe: #%d\n", id, cs.CrossUnsafeL2.Number)
					lastCrossUnsafe[id] = cs.CrossUnsafeL2.Number
				}

				// Local-safe progression
				if cs.LocalSafeL2.Number != lastLocalSafe[id] {
					fmt.Fprintf(stderr, "[interop] Chain %s local-safe: #%d\n", id, cs.LocalSafeL2.Number)
					lastLocalSafe[id] = cs.LocalSafeL2.Number
				}

				// Cross-safe (safe head) — reorg shows as decrease
				if cs.SafeL2.Number != lastSafe[id] {
					if cs.SafeL2.Number < lastSafe[id] {
						fmt.Fprintf(stderr, "[interop] Chain %s REORG: safe #%d -> #%d\n",
							id, lastSafe[id], cs.SafeL2.Number)
					} else {
						fmt.Fprintf(stderr, "[interop] Chain %s cross-safe: #%d\n", id, cs.SafeL2.Number)
					}
					lastSafe[id] = cs.SafeL2.Number
				}
			}
		}
	}
}

func proxyEL(ctx context.Context, stderr io.Writer, listener net.Listener, client client.RPC, backendURL string, controlled *controlledService) error {
	mux := http.NewServeMux()
	handler, err := rpcProxyHandler(ctx, stderr, client, backendURL, controlled)
	if err != nil {
		return err
	}
	mux.Handle("/", handler)

	server := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown proxy server: %w", err)
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	}
}

func rpcProxyHandler(ctx context.Context, stderr io.Writer, client client.RPC, backendURL string, controlled *controlledService) (http.Handler, error) {
	if backendURL != "" {
		target, err := url.Parse(backendURL)
		if err != nil {
			return nil, fmt.Errorf("parse backend URL %q: %w", backendURL, err)
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		originalDirector := proxy.Director
		proxy.Director = func(r *http.Request) {
			originalDirector(r)
			r.Host = target.Host
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !prepareRPCProxyResponse(w, r) {
				return
			}
			if controlled != nil && !controlled.proxyAvailable() {
				writeRPCProxyStopped(ctx, stderr, w, r, controlled)
				return
			}
			proxy.ServeHTTP(w, r)
		}), nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !prepareRPCProxyResponse(w, r) {
			return
		}

		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		response, ok := handleRPCProxyRequest(ctx, stderr, client, controlled, requestBody)
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeRPCProxyJSON(w, response)
	}), nil
}

func prepareRPCProxyResponse(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func writeRPCProxyStopped(ctx context.Context, stderr io.Writer, w http.ResponseWriter, r *http.Request, controlled *controlledService) {
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	response, ok := handleRPCProxyRequest(ctx, stderr, nil, controlled, requestBody)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeRPCProxyJSON(w, response)
}

func writeRPCProxyJSON(w http.ResponseWriter, response any) {
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal RPC response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(jsonResponse); err != nil {
		return
	}
}

type rpcProxyRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type rpcProxyResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcProxyError  `json:"error,omitempty"`
}

type rpcProxyError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func handleRPCProxyRequest(ctx context.Context, stderr io.Writer, client client.RPC, controlled *controlledService, body []byte) (any, bool) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return rpcProxyErr(nil, -32700, "empty JSON-RPC request"), true
	}

	isBatch := trimmed[0] == '['
	var requests []rpcProxyRequest
	if isBatch {
		if err := json.Unmarshal(trimmed, &requests); err != nil {
			return rpcProxyErr(nil, -32700, "invalid JSON-RPC batch"), true
		}
		if len(requests) == 0 {
			return rpcProxyErr(nil, -32600, "empty JSON-RPC batch"), true
		}
	} else {
		var req rpcProxyRequest
		if err := json.Unmarshal(trimmed, &req); err != nil {
			return rpcProxyErr(nil, -32700, "invalid JSON-RPC request"), true
		}
		requests = []rpcProxyRequest{req}
	}

	responses := make([]rpcProxyResponse, 0, len(requests))
	for _, req := range requests {
		resp, reply := executeRPCProxyRequest(ctx, stderr, client, controlled, req)
		if reply {
			responses = append(responses, resp)
		}
	}
	if len(responses) == 0 {
		return nil, false
	}
	if isBatch {
		return responses, true
	}
	return responses[0], true
}

func executeRPCProxyRequest(ctx context.Context, stderr io.Writer, client client.RPC, controlled *controlledService, req rpcProxyRequest) (rpcProxyResponse, bool) {
	if req.Method == "" {
		return rpcProxyErr(req.ID, -32600, "missing JSON-RPC method"), true
	}
	if controlled != nil && !controlled.proxyAvailable() {
		return backendStoppedError(req.ID, req.Method), true
	}

	params, err := decodeRPCProxyParams(req.Params)
	if err != nil {
		return rpcProxyErr(req.ID, -32602, err.Error()), true
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var result json.RawMessage
	if err := client.CallContext(callCtx, &result, req.Method, params...); err != nil {
		message := fmt.Sprintf("RPC call to backend failed for method %q: %v", req.Method, err)
		fmt.Fprintf(stderr, "RPC error: %s\n", message)
		return rpcProxyErr(req.ID, -32000, message), true
	}
	if len(result) == 0 {
		result = json.RawMessage("null")
	}
	if len(req.ID) == 0 {
		return rpcProxyResponse{}, false
	}
	return rpcProxyResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, true
}

func decodeRPCProxyParams(raw json.RawMessage) ([]any, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	switch raw[0] {
	case '[':
		var params []any
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC params array")
		}
		return params, nil
	case '{':
		var param map[string]any
		if err := json.Unmarshal(raw, &param); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC params object")
		}
		return []any{param}, nil
	default:
		return nil, fmt.Errorf("invalid JSON-RPC params: expected array, object, or null")
	}
}

func rpcProxyErr(id json.RawMessage, code int, message string) rpcProxyResponse {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return rpcProxyResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcProxyError{
			Code:    code,
			Message: message,
		},
	}
}

type testingT struct {
	state  *testingState
	ctx    context.Context
	logger log.Logger
	tracer trace.Tracer
	req    *testreq.Assertions
	gate   *testreq.Assertions
}

type testingState struct {
	mu            sync.Mutex
	tempRoot      string
	tempDirPrefix string
	cleanups      []func()
}

type testingFailure struct {
	err error
}

func (f testingFailure) Error() string {
	return f.err.Error()
}

func asError(v any) error {
	if err, ok := v.(error); ok {
		return err
	}
	return nil
}

func newTestingT(ctx context.Context, stderr io.Writer, tempRoot string, tempDirPrefix string, logWriters ...io.Writer) *testingT {
	logger := newLogger(ctx, stderr, logWriters...)
	t := &testingT{
		state: &testingState{
			tempRoot:      tempRoot,
			tempDirPrefix: tempDirPrefix,
			cleanups:      make([]func(), 0),
		},
		ctx:    ctx,
		logger: logger,
		tracer: otel.Tracer("op-up"),
	}
	t.req = testreq.New(t)
	t.gate = testreq.New(t)
	return t
}

func (t *testingT) failf(format string, args ...any) {
	err := fmt.Errorf(format, args...)
	t.logger.Error("op-up runtime failure", "err", err)
	debug.PrintStack()
	panic(testingFailure{err: err})
}

var _ devtest.T = (*testingT)(nil)
var _ testreq.TestingT = (*testingT)(nil)

func (t *testingT) doCleanup() {
	t.state.mu.Lock()
	cleanups := append([]func(){}, t.state.cleanups...)
	t.state.cleanups = nil
	t.state.mu.Unlock()
	for _, cleanup := range slices.Backward(cleanups) {
		cleanup()
	}
}

// Cleanup implements devtest.T.
func (t *testingT) Cleanup(fn func()) {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	t.state.cleanups = append(t.state.cleanups, fn)
}

// Ctx implements devtest.T.
func (t *testingT) Ctx() context.Context {
	return t.ctx
}

// Deadline implements devtest.T.
func (t *testingT) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

// Error implements devtest.T.
func (t *testingT) Error(args ...any) {
	t.failf("%s", fmt.Sprint(args...))
}

// Errorf implements devtest.T.
func (t *testingT) Errorf(format string, args ...any) {
	t.failf(format, args...)
}

// Fail implements devtest.T.
func (t *testingT) Fail() {
	t.failf("test failed")
}

// FailNow implements devtest.T.
func (t *testingT) FailNow() {
	t.failf("test failed immediately")
}

// Gate implements devtest.T.
func (t *testingT) Gate() *testreq.Assertions {
	return t.gate
}

// MarkFlaky implements devtest.T.
func (t *testingT) MarkFlaky(string) {
}

// Helper implements devtest.T.
func (t *testingT) Helper() {
}

// Log implements devtest.T.
func (t *testingT) Log(args ...any) {
	t.logger.Info(fmt.Sprint(args...))
}

// Logf implements devtest.T.
func (t *testingT) Logf(format string, args ...any) {
	t.logger.Info(fmt.Sprintf(format, args...))
}

func (t *testingT) Logger() log.Logger {
	return t.logger
}

func (t *testingT) Name() string {
	return "dev"
}

func (t *testingT) Parallel() {
}

func (t *testingT) Require() *testreq.Assertions {
	return t.req
}

func (t *testingT) Run(name string, fn func(devtest.T)) {
	subCtx := devtest.AddTestScope(t.ctx, name)
	fn(t.WithCtx(subCtx))
}

func (t *testingT) Skip(args ...any) {
	t.failf("unexpected skip: %s", fmt.Sprint(args...))
}

func (t *testingT) SkipNow() {
	t.failf("unexpected skip")
}

// Skipf implements devtest.T.
func (t *testingT) Skipf(format string, args ...any) {
	t.failf("unexpected skip: "+format, args...)
}

// Skipped implements devtest.T.
func (t *testingT) Skipped() bool {
	return false
}

// TempDir implements devtest.T.
func (t *testingT) TempDir() string {
	return t.TempDirWithPrefix("")
}

func (t *testingT) TempDirWithPrefix(prefix string) string {
	tempDirPrefix := t.state.tempDirPrefix
	if servicePrefix := tempDirSafePrefix(prefix); servicePrefix != "" {
		tempDirPrefix += "-" + servicePrefix
	}
	dir, err := os.MkdirTemp(t.state.tempRoot, tempDirPrefix+"-*")
	if err != nil {
		t.failf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.logger.Error("failed to clean up temp dir", "dir", dir, "err", err)
		}
	})
	return dir
}

func tempDirSafePrefix(prefix string) string {
	prefix = strings.NewReplacer("/", "-", "\\", "-", " ", "-", "_", "-").Replace(strings.TrimSpace(prefix))
	return strings.Trim(prefix, "-")
}

// Tracer implements devtest.T.
func (t *testingT) Tracer() trace.Tracer {
	return t.tracer
}

// WithCtx implements devtest.T.
func (t *testingT) WithCtx(ctx context.Context) devtest.T {
	logger := t.logger.New()
	logger.SetContext(ctx)
	out := &testingT{
		state:  t.state,
		ctx:    ctx,
		logger: logger,
		tracer: t.tracer,
	}
	out.req = testreq.New(out)
	out.gate = testreq.New(out)
	return out
}

// _TestOnly implements devtest.T.
func (t *testingT) TestOnly() {
}
