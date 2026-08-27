package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
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
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopsmoke"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const asciiArt = ` ____  ____        _     ____
/  _ \/  __\      / \ /\/  __\
| / \||  \/|_____ | | |||  \/|
| \_/||  __/\____\| \_/||  __/
\____/\_/         \____/\_/`

const (
	opUpInteropDelay        = uint64(2)
	opUpSilhouetteBlockTime = uint64(1)
	opUpDemoTxPoolSlots     = uint64(256)
)

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
	interopFlag = &cli.BoolFlag{
		Name:    "interop",
		Usage:   "start a 2-chain interop devnet backed by op-supernode.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "INTEROP"),
	}
	silhouetteFlag = &cli.BoolFlag{
		Name: "silhouette",
		Usage: "start a 2-chain interop devnet with L2B carried by proof batches " +
			"from the ordinary batcher's silhouette encoder.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "SILHOUETTE"),
	}
	l2ARPCPortFlag = &cli.UintFlag{
		Name:    "l2a-rpc-port",
		Usage:   "host port for the L2A execution-layer JSON-RPC proxy.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "L2A_RPC_PORT"),
		Value:   8545,
	}
	l2BRPCPortFlag = &cli.UintFlag{
		Name:    "l2b-rpc-port",
		Usage:   "host port for the L2B execution-layer JSON-RPC proxy.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "L2B_RPC_PORT"),
		Value:   8546,
	}
	explorersFlag = &cli.BoolFlag{
		Name:    "explorers",
		Usage:   "start one Otterscan explorer GUI per L2 in two-chain devnets.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "EXPLORERS"),
		Value:   false,
	}
	publicExplorerPortFlag = &cli.UintFlag{
		Name:    "public-explorer-port",
		Usage:   "host port for the light-themed chain 901 Otterscan GUI.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "PUBLIC_EXPLORER_PORT"),
		Value:   defaultPublicExplorerPort,
	}
	privateExplorerPortFlag = &cli.UintFlag{
		Name:    "private-explorer-port",
		Usage:   "host port for the dark-themed chain 902 Otterscan GUI.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "PRIVATE_EXPLORER_PORT"),
		Value:   defaultPrivateExplorerPort,
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
	app.Usage = "deploys an in-memory OP Stack devnet."
	app.Flags = cliapp.ProtectFlags([]cli.Flag{
		dirFlag, interopFlag, silhouetteFlag, l2ARPCPortFlag, l2BRPCPortFlag,
		explorersFlag, publicExplorerPortFlag, privateExplorerPortFlag,
	})
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
		return runOpUp(cliCtx.Context, cliCtx.App.ErrWriter, cliCtx.String(dirFlag.Name),
			cliCtx.Bool(interopFlag.Name), cliCtx.Bool(silhouetteFlag.Name),
			cliCtx.Uint(l2ARPCPortFlag.Name), cliCtx.Uint(l2BRPCPortFlag.Name),
			cliCtx.Bool(explorersFlag.Name), cliCtx.Uint(publicExplorerPortFlag.Name),
			cliCtx.Uint(privateExplorerPortFlag.Name))
	}
	app.Commands = []*cli.Command{
		interopsmoke.Command(envPrefix),
	}
	return app.RunContext(ctx, args)
}

func runOpUp(ctx context.Context, stderr io.Writer, opUpDir string, interop, silhouette bool,
	l2APort, l2BPort uint, explorers bool, publicExplorerPort, privateExplorerPort uint,
) error {
	fmt.Fprintf(stderr, "%s\n", asciiArt)

	if err := os.MkdirAll(opUpDir, 0o755); err != nil {
		return fmt.Errorf("create the op-up dir: %w", err)
	}
	tempRoot := filepath.Join(opUpDir, "tmp")
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return fmt.Errorf("create the op-up temp dir: %w", err)
	}

	devtest.RootContext = ctx
	t := newTestingT(ctx, stderr, tempRoot)
	defer t.doCleanup()

	if silhouette {
		sys, err := newSilhouetteSystem(t, explorers)
		if err != nil {
			return err
		}
		if err := runSupernodeSystem(ctx, stderr, t, sys, l2APort, l2BPort,
			explorers, publicExplorerPort, privateExplorerPort); err != nil {
			return err
		}
	} else if interop {
		sys, err := newSupernodeInteropSystem(t, explorers)
		if err != nil {
			return err
		}
		if err := runSupernodeSystem(ctx, stderr, t, sys, l2APort, l2BPort,
			explorers, publicExplorerPort, privateExplorerPort); err != nil {
			return err
		}
	} else {
		sys, err := newMinimalSystem(t)
		if err != nil {
			return err
		}
		if err := runSystem(ctx, stderr, sys, l2APort); err != nil {
			return err
		}
	}
	fmt.Fprintf(stderr, "\nPlease consider filling out this survey to influence future development: https://www.surveymonkey.com/r/JTGHFK3\n")
	return nil
}

func newLogger(ctx context.Context, stderr io.Writer) log.Logger {
	logHandler := oplog.NewLogHandler(stderr, oplog.DefaultCLIConfig())
	logHandler = logfilter.WrapFilterHandler(logHandler)
	logHandler.(logfilter.FilterHandler).Set(logfilter.DefaultMute())
	logHandler = logfilter.WrapContextHandler(logHandler)
	logger := log.NewLogger(logHandler)
	oplog.SetGlobalLogHandler(logHandler)
	logger.SetContext(ctx)
	return logger
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
	// op-up exposes a lightweight devnet; it does not need dispute-game helpers,
	// and go-tests-short does not build kona-host for the challenger.
	return presets.NewMinimalNoFaultProofs(t), nil
}

func newSupernodeInteropSystem(t *testingT, explorers bool) (sys *presets.TwoL2SupernodeInterop, err error) {
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
	return presets.NewTwoL2SupernodeInterop(t, opUpInteropDelay, opUpInteropOptions(explorers)...), nil
}

func newSilhouetteSystem(t *testingT, explorers bool) (sys *presets.TwoL2SupernodeInterop, err error) {
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
	opts := append(opUpInteropOptions(explorers),
		presets.WithSilhouetteChain(presets.SilhouetteChainB),
		presets.WithUniformL2BlockTimes(opUpSilhouetteBlockTime),
		presets.WithOpRethOption(sysgo.OpRethWithExtraArgs(
			fmt.Sprintf("--txpool.max-account-slots=%d", opUpDemoTxPoolSlots),
		)),
	)
	sys = presets.NewTwoL2SupernodeLightSequencerInterop(t, opUpInteropDelay, opts...)
	// The private chains are always sequenced by their LightCL nodes. Be explicit here because the
	// light-sequencer preset is also used by handoff tests that intentionally start them stopped.
	for _, cl := range []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL} {
		active, activeErr := cl.Escape().RollupAPI().SequencerActive(t.Ctx())
		if activeErr != nil {
			return nil, activeErr
		}
		if !active {
			cl.StartSequencer()
		}
	}
	return sys, nil
}

// opUpInteropOptions is shared so silhouette mode changes only L2B's publication and verification
// topology, not the activation configuration users get from ordinary --interop mode.
func opUpInteropOptions(explorers bool) []presets.Option {
	opts := []presets.Option{presets.WithSuggestedLagoonActivationOffset(opUpInteropDelay)}
	if explorers {
		opts = append(opts, presets.WithOpRethOption(sysgo.OpRethWithOtterscanAPI()))
	}
	return opts
}

func runSystem(ctx context.Context, stderr io.Writer, sys *presets.Minimal, l2Port uint) error {
	if err := printAccountInfo(stderr); err != nil {
		return err
	}
	rpcAddr := fmt.Sprintf("127.0.0.1:%d", l2Port)
	fmt.Fprintf(stderr, "EL Node URL: http://%s\n", rpcAddr)

	elNode := sys.L2EL
	go logBlocks(ctx, stderr, "L2", elNode)

	listener, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		return fmt.Errorf("listen for L2 RPC on %s: %w", rpcAddr, err)
	}
	defer listener.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- proxyEL(ctx, stderr, listener, elNode.Escape().L2EthClient().RPC()) }()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func runSupernodeSystem(ctx context.Context, stderr io.Writer, t *testingT,
	sys *presets.TwoL2SupernodeInterop, l2APort, l2BPort uint,
	explorers bool, publicExplorerPort, privateExplorerPort uint,
) error {
	if err := printAccountInfo(stderr); err != nil {
		return err
	}
	if sys.Silhouette != nil {
		switch sys.Silhouette.ChainKey {
		case presets.SilhouetteChainA:
			sys.L2BatcherA.Start()
		case presets.SilhouetteChainB:
			sys.L2BatcherB.Start()
		default:
			return fmt.Errorf("unknown silhouette chain %q", sys.Silhouette.ChainKey)
		}
		fmt.Fprintf(stderr, "L2A submitter: op-batcher\n")
		fmt.Fprintf(stderr, "L2B submitter: op-batcher (silhouette proof encoding)\n")
		fmt.Fprintf(stderr, "Silhouette verifier manifest: %s\n", sys.Silhouette.Runtime.ManifestPath)
	}
	l2AAddr := fmt.Sprintf("127.0.0.1:%d", l2APort)
	l2BAddr := fmt.Sprintf("127.0.0.1:%d", l2BPort)
	fmt.Fprintf(stderr, "L2A Chain ID: %s\n", sys.L2A.ChainID())
	fmt.Fprintf(stderr, "L2A EL Node URL: http://%s\n", l2AAddr)
	fmt.Fprintf(stderr, "L2B Chain ID: %s\n", sys.L2B.ChainID())
	fmt.Fprintf(stderr, "L2B EL Node URL: http://%s\n", l2BAddr)

	go logBlocks(ctx, stderr, "L2A", sys.L2ELA)
	go logBlocks(ctx, stderr, "L2B", sys.L2ELB)
	go logInterop(ctx, stderr, sys)

	l2AListener, err := net.Listen("tcp", l2AAddr)
	if err != nil {
		return fmt.Errorf("listen for L2A RPC on %s: %w", l2AAddr, err)
	}
	defer l2AListener.Close()
	l2BListener, err := net.Listen("tcp", l2BAddr)
	if err != nil {
		return fmt.Errorf("listen for L2B RPC on %s: %w", l2BAddr, err)
	}
	defer l2BListener.Close()
	explorerConfig := otterscanConfig{
		publicRPCPort:       l2APort,
		privateRPCPort:      l2BPort,
		publicExplorerPort:  publicExplorerPort,
		privateExplorerPort: privateExplorerPort,
		publicChainID:       sys.L2A.ChainID().String(),
		privateChainID:      sys.L2B.ChainID().String(),
	}
	var l2ACORSOrigins, l2BCORSOrigins []string
	if explorers {
		if err := explorerConfig.validate(); err != nil {
			return err
		}
		l2ACORSOrigins = []string{explorerConfig.publicURL()}
		l2BCORSOrigins = []string{explorerConfig.privateURL()}
	}
	errCh := make(chan error, 2)
	go func() {
		errCh <- proxyEL(ctx, stderr, l2AListener, sys.L2ELA.Escape().L2EthClient().RPC(), l2ACORSOrigins...)
	}()
	go func() {
		errCh <- proxyEL(ctx, stderr, l2BListener, sys.L2ELB.Escape().L2EthClient().RPC(), l2BCORSOrigins...)
	}()

	var explorerErrCh <-chan error
	if explorers {
		stack, err := startOtterscanStack(ctx, stderr, t.TempDirWithPrefix("otterscan"), explorerConfig)
		if err != nil {
			return err
		}
		t.Cleanup(func() {
			if err := stack.stop(); err != nil {
				fmt.Fprintf(stderr, "failed to stop Otterscan explorer GUIs: %v\n", err)
			}
		})
		fmt.Fprintf(stderr, "Public chain explorer (%s, light): %s\n", sys.L2A.ChainID(), explorerConfig.publicURL())
		fmt.Fprintf(stderr, "Private chain explorer (%s, dark): %s\n", sys.L2B.ChainID(), explorerConfig.privateURL())
		monitorErrCh := make(chan error, 1)
		explorerErrCh = monitorErrCh
		go func() { monitorErrCh <- stack.monitor(ctx) }()
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	case err := <-explorerErrCh:
		return err
	}
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
	lastUnsafe := make(map[string]uint64)

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

				if cs.UnsafeL2.Number != lastUnsafe[id] && cs.UnsafeL2.Number > 0 {
					fmt.Fprintf(stderr, "[interop] Chain %s unsafe: #%d\n", id, cs.UnsafeL2.Number)
					lastUnsafe[id] = cs.UnsafeL2.Number
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

func logBlocks(ctx context.Context, stderr io.Writer, name string, elNode *dsl.L2ELNode) {
	const blockPollInterval = 500 * time.Millisecond
	var lastBlock uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(blockPollInterval):
			unsafe, err := elNode.EthClient().BlockRefByLabel(ctx, eth.Unsafe)
			if err != nil {
				continue
			}
			if unsafe.Number != lastBlock {
				fmt.Fprintf(stderr, "New %s block: number %d, hash %s\n", name, unsafe.Number, unsafe.Hash)
				lastBlock = unsafe.Number
			}
		}
	}
}

// proxyEL intercepts EL JSON-RPC requests for logging. It also supports the batched requests and
// narrowly scoped browser CORS access required by the optional local Otterscan explorers.
func proxyEL(ctx context.Context, stderr io.Writer, listener net.Listener, rpcClient client.RPC,
	allowedOrigins ...string,
) error {
	server := &http.Server{Handler: newELProxyHandler(stderr, rpcClient, allowedOrigins...)}
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
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	}
}

type elProxyRequest struct {
	JSONRPC    string          `json:"jsonrpc"`
	ID         json.RawMessage `json:"id"`
	Method     string          `json:"method"`
	Params     json.RawMessage `json:"params"`
	callParams []any
}

type elProxyError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type elProxyResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *elProxyError   `json:"error,omitempty"`
}

func newELProxyHandler(stderr io.Writer, rpcClient client.RPC, allowedOrigins ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !slices.Contains(allowedOrigins, origin) {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()
		requestBody, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		requests, batch, err := decodeELProxyRequests(requestBody)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		forwardCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		responses := forwardELProxyRequests(forwardCtx, stderr, rpcClient, requests, batch)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		if batch {
			_ = encoder.Encode(responses)
		} else {
			_ = encoder.Encode(responses[0])
		}
	})
}

func decodeELProxyRequests(body []byte) ([]elProxyRequest, bool, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, false, fmt.Errorf("empty JSON RPC request")
	}
	batch := strings.HasPrefix(trimmed, "[")
	var requests []elProxyRequest
	if batch {
		if err := json.Unmarshal(body, &requests); err != nil {
			return nil, true, fmt.Errorf("invalid JSON RPC batch request: %w", err)
		}
		if len(requests) == 0 {
			return nil, true, fmt.Errorf("empty JSON RPC batch request")
		}
	} else {
		var request elProxyRequest
		if err := json.Unmarshal(body, &request); err != nil {
			return nil, false, fmt.Errorf("invalid JSON RPC request: %w", err)
		}
		requests = []elProxyRequest{request}
	}
	for i := range requests {
		if requests[i].Method == "" {
			return nil, batch, fmt.Errorf("missing or invalid 'method' field in JSON RPC request")
		}
		params, err := decodeELProxyParams(requests[i].Params)
		if err != nil {
			return nil, batch, fmt.Errorf("%s params: %w", requests[i].Method, err)
		}
		requests[i].callParams = params
	}
	return requests, batch, nil
}

func decodeELProxyParams(raw json.RawMessage) ([]any, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var params []any
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		return params, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var params map[string]any
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		return []any{params}, nil
	}
	return nil, fmt.Errorf("must be an array, object, or null")
}

func forwardELProxyRequests(ctx context.Context, stderr io.Writer, rpcClient client.RPC,
	requests []elProxyRequest, batch bool,
) []elProxyResponse {
	responses := make([]elProxyResponse, len(requests))
	results := make([]json.RawMessage, len(requests))
	for i, request := range requests {
		fmt.Fprintln(stderr, request.Method)
		responses[i] = elProxyResponse{JSONRPC: "2.0", ID: request.ID}
	}
	if !batch {
		err := rpcClient.CallContext(ctx, &results[0], requests[0].Method, requests[0].callParams...)
		setELProxyResult(stderr, &responses[0], results[0], requests[0].Method, err)
		return responses
	}

	elements := make([]gethrpc.BatchElem, len(requests))
	for i, request := range requests {
		elements[i] = gethrpc.BatchElem{Method: request.Method, Args: request.callParams, Result: &results[i]}
	}
	batchErr := rpcClient.BatchCallContext(ctx, elements)
	for i, request := range requests {
		err := elements[i].Error
		if err == nil {
			err = batchErr
		}
		setELProxyResult(stderr, &responses[i], results[i], request.Method, err)
	}
	return responses
}

func setELProxyResult(stderr io.Writer, response *elProxyResponse, result json.RawMessage, method string, err error) {
	if err == nil {
		if len(result) == 0 {
			result = json.RawMessage("null")
		}
		response.Result = result
		return
	}
	message := fmt.Sprintf("RPC call to backend failed for method '%s': %v", method, err)
	fmt.Fprintf(stderr, "RPC error: %s\n", message)
	response.Error = &elProxyError{Code: -32000, Message: message}
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
	mu       sync.Mutex
	tempRoot string
	cleanups []func()
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

func newTestingT(ctx context.Context, stderr io.Writer, tempRoot string) *testingT {
	logger := newLogger(ctx, stderr)
	t := &testingT{
		state: &testingState{
			tempRoot: tempRoot,
			cleanups: make([]func(), 0),
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
	return t.TempDirWithPrefix("op-up")
}

// TempDirWithPrefix implements devtest.T.
func (t *testingT) TempDirWithPrefix(prefix string) string {
	prefix = strings.NewReplacer("/", "-", "\\", "-", " ", "-", "_", "-").Replace(strings.TrimSpace(prefix))
	prefix = strings.Trim(prefix, "-")
	if prefix == "" {
		prefix = "op-up"
	}
	dir, err := os.MkdirTemp(t.state.tempRoot, prefix+"-*")
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
