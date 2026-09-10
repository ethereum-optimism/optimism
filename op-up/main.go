package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
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
	interopFlag = &cli.BoolFlag{
		Name:    "interop",
		Usage:   "start a 2-chain interop devnet backed by op-supernode.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "INTEROP"),
	}
	privateInteropFlag = &cli.BoolFlag{
		Name: "private-interop",
		Usage: "start the 2-chain interop devnet with chain B as a private-interop pair: " +
			"a private sequenced chain plus the public rendering the supernode judges. Implies --interop.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "PRIVATE_INTEROP"),
	}
	smokeFlag = &cli.BoolFlag{
		Name: "smoke",
		Usage: "run the interop smoke tests against the devnet once it is up, then exit with their result " +
			"instead of staying up. Implies --interop.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE"),
	}
)

// devstack environment variables op-up reads to fail early rather than half-build a system. They
// are sysgo's (op-devstack/sysgo/mixed_runtime.go), which does not export them.
const (
	devstackL2ELKindEnv           = "DEVSTACK_L2EL_KIND"
	devstackL2ELOverrideBinaryEnv = "DEVSTACK_L2EL_OVERRIDE_BINARY"
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
	app.Flags = cliapp.ProtectFlags([]cli.Flag{dirFlag, interopFlag, privateInteropFlag, smokeFlag})
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
		return runOpUp(cliCtx.Context, cliCtx.App.ErrWriter, opUpConfig{
			dir:            cliCtx.String(dirFlag.Name),
			interop:        cliCtx.Bool(interopFlag.Name),
			privateInterop: cliCtx.Bool(privateInteropFlag.Name),
			smoke:          cliCtx.Bool(smokeFlag.Name),
		})
	}
	app.Commands = []*cli.Command{
		interopsmoke.Command(envPrefix),
	}
	return app.RunContext(ctx, args)
}

// opUpConfig is what the command line asked for.
type opUpConfig struct {
	dir string
	// interop starts the two-chain supernode devnet.
	interop bool
	// privateInterop makes that devnet's chain B a private-interop pair. It implies interop.
	privateInterop bool
	// smoke runs the interop smoke tests against the devnet and exits with their result. It
	// implies interop: there is nothing for an interop smoke to do on a single chain.
	smoke bool
}

// interopTopology reports whether the two-chain supernode devnet is what was asked for.
func (c opUpConfig) interopTopology() bool {
	return c.interop || c.privateInterop || c.smoke
}

func runOpUp(ctx context.Context, stderr io.Writer, cfg opUpConfig) error {
	fmt.Fprintf(stderr, "%s\n", asciiArt)

	if err := os.MkdirAll(cfg.dir, 0o755); err != nil {
		return fmt.Errorf("create the op-up dir: %w", err)
	}
	tempRoot := filepath.Join(cfg.dir, "tmp")
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return fmt.Errorf("create the op-up temp dir: %w", err)
	}

	devtest.RootContext = ctx
	t := newTestingT(ctx, stderr, tempRoot)
	defer t.doCleanup()

	if cfg.privateInterop {
		// Checked before anything is built: both failures below would otherwise land as a fatal
		// "unexpected skip" or a stack trace out of the middle of a half-built system.
		if err := checkPrivateInteropPrerequisites(ctx, t.Logger()); err != nil {
			return err
		}
		sys, err := newPrivateInteropSystem(t)
		if err != nil {
			return err
		}
		if err := runSupernodeSystem(ctx, stderr, sys, cfg); err != nil {
			return err
		}
	} else if cfg.interopTopology() {
		sys, err := newSupernodeInteropSystem(t)
		if err != nil {
			return err
		}
		if err := runSupernodeSystem(ctx, stderr, sys, cfg); err != nil {
			return err
		}
	} else {
		sys, err := newMinimalSystem(t)
		if err != nil {
			return err
		}
		if err := runSystem(ctx, stderr, sys); err != nil {
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

// buildSystem turns a preset constructor's fatal-failure panic back into an error, so op-up can
// report it and exit rather than unwinding through main.
func buildSystem[S any](build func() S) (sys S, err error) {
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
	return build(), nil
}

func newMinimalSystem(t *testingT) (*presets.Minimal, error) {
	// op-up exposes a lightweight devnet; it does not need dispute-game helpers,
	// and go-tests-short does not build kona-host for the challenger.
	return buildSystem(func() *presets.Minimal { return presets.NewMinimalNoFaultProofs(t) })
}

// interopDelay is a small activation delay, so that interop bridge contracts
// (SuperchainETHBridge, ETHLiquidity) get properly initialized.
const interopDelay = uint64(2)

func newSupernodeInteropSystem(t *testingT) (*presets.TwoL2SupernodeInterop, error) {
	return buildSystem(func() *presets.TwoL2SupernodeInterop {
		return presets.NewTwoL2SupernodeInterop(t, interopDelay,
			presets.WithSuggestedLagoonActivationOffset(interopDelay),
		)
	})
}

// newPrivateInteropSystem is the interop system above with chain B replaced by a private-interop
// pair: a private sequenced chain that holds the RPC surface, and the public rendering the
// supernode judges. Every other handle keeps its ordinary meaning, which is the whole point of the
// preset option (op-devstack/presets/options.go, WithPrivateInteropChain).
//
// Interop is at GENESIS here, where the public devnet above activates it two seconds in. That is
// not a preference: the pair's genesis is generated by a contracts-bedrock dev feature that
// refuses to build otherwise ("private interop private chain requires interop at genesis",
// L2Genesis.s.sol), and the private-interop acceptance tests construct their presets the same way.
// A delay -- or a suggested Lagoon offset, which is the same statement made through the deployer --
// fails during genesis generation, before any node starts.
//
// The standing rendering-invariant checker is off. It asserts at t.Cleanup, and op-up's cleanup
// runs on Ctrl-C: a devnet the operator stopped mid-range would fail the assertion and panic out
// of the shutdown path, which is not what "the operator pressed Ctrl-C" should look like.
func newPrivateInteropSystem(t *testingT) (*presets.TwoL2SupernodeInterop, error) {
	return buildSystem(func() *presets.TwoL2SupernodeInterop {
		return presets.NewTwoL2SupernodeInterop(t, 0,
			presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck()),
		)
	})
}

// checkPrivateInteropPrerequisites reports the two environment problems that stop a pair before it
// starts, in the operator's terms.
//
// A pair is two op-reths on one chain ID, and the rendering's genesis comes from a dev feature only
// the op-reth lane runs, so the preset skips on op-geth (presets/twol2.go). op-up's testing handle
// treats a skip as fatal, and the resulting message is about a skip rather than about the
// environment that caused it.
func checkPrivateInteropPrerequisites(ctx context.Context, logger log.Logger) error {
	if kind := os.Getenv(devstackL2ELKindEnv); kind == string(sysgo.MixedL2ELOpGeth) {
		return fmt.Errorf("--private-interop needs op-reth, but %s=%s; unset it or set it to %q",
			devstackL2ELKindEnv, kind, sysgo.MixedL2ELOpReth)
	}
	// The same binary the devstack will launch: an override names a CLI-superset of op-reth, and
	// rustbin derives its path variable from the name (RUST_BINARY_PATH_OP_RETH by default).
	binary := os.Getenv(devstackL2ELOverrideBinaryEnv)
	if binary == "" {
		binary = "op-reth"
	}
	path, err := rustbin.Spec{SrcDir: "rust", Package: binary, Binary: binary}.EnsureExists(ctx, logger)
	if err != nil {
		return fmt.Errorf("--private-interop needs the %s binary: %w\n"+
			"Build it ('cd rust && just build-%s'), or set RUST_BINARY_PATH_%s to a prebuilt one",
			binary, err, binary, strings.ToUpper(strings.ReplaceAll(binary, "-", "_")))
	}
	logger.Info("Using L2 execution binary", "binary", binary, "path", path)
	return nil
}

func runSystem(ctx context.Context, stderr io.Writer, sys *presets.Minimal) error {
	if err := printAccountInfo(stderr); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "EL Node URL: %s\n", "http://localhost:8545")

	elNode := sys.L2EL
	go logBlocks(ctx, stderr, "L2", elNode)

	// Proxy L2 EL requests.
	go func() {
		if err := proxyEL(ctx, stderr, "localhost:8545", elNode.Escape().L2EthClient().RPC()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "error: %v", err)
		}
	}()

	<-ctx.Done()

	return nil
}

func runSupernodeSystem(ctx context.Context, stderr io.Writer, sys *presets.TwoL2SupernodeInterop, cfg opUpConfig) error {
	if err := printAccountInfo(stderr); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "L2A Chain ID: %s\n", sys.L2A.ChainID())
	fmt.Fprintf(stderr, "L2A EL Node URL: %s\n", "http://localhost:8545")
	fmt.Fprintf(stderr, "L2B Chain ID: %s\n", sys.L2B.ChainID())
	fmt.Fprintf(stderr, "L2B EL Node URL: %s\n", "http://localhost:8546")
	// The two URLs above are op-up's logging proxy, which speaks one JSON-RPC object per request.
	// A client that batches (the interop smoke does) needs the node itself.
	fmt.Fprintf(stderr, "L2A EL Node URL (direct, batch-capable): %s\n", sys.L2ELA.Escape().UserRPC())
	fmt.Fprintf(stderr, "L2B EL Node URL (direct, batch-capable): %s\n", sys.L2ELB.Escape().UserRPC())

	if pi := sys.PrivateInterop; pi != nil {
		// L2B above is the PRIVATE chain: it holds the RPC surface, and its blocks are not public.
		// What every counterparty means by chain B is the rendering, which is what the supernode
		// judges and what message identifiers name.
		fmt.Fprintf(stderr, "L2B is a private-interop pair; the endpoints above are its PRIVATE half.\n")
		fmt.Fprintf(stderr, "L2B Rendering EL Node URL: %s\n", sys.L2BSupernodeEL.Escape().UserRPC())
		fmt.Fprintf(stderr, "L2B Rendering CL Node URL: %s\n", sys.L2BSupernodeCL.Escape().UserRPC())
		fmt.Fprintf(stderr, "L2B Private op-node follow source: %s\n", pi.FollowSource())
	}

	go logBlocks(ctx, stderr, "L2A", sys.L2ELA)
	go logBlocks(ctx, stderr, "L2B", sys.L2ELB)
	go logInterop(ctx, stderr, sys)

	go func() {
		if err := proxyEL(ctx, stderr, "localhost:8545", sys.L2ELA.Escape().L2EthClient().RPC()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "error: %v", err)
		}
	}()
	go func() {
		if err := proxyEL(ctx, stderr, "localhost:8546", sys.L2ELB.Escape().L2EthClient().RPC()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "error: %v", err)
		}
	}()

	if cfg.smoke {
		return runSmoke(ctx, stderr, sys)
	}

	<-ctx.Done()

	return nil
}

// runSmoke runs the interop smoke tests in THIS process, against the nodes' own RPCs.
//
// Both details are load-bearing. The URLs are the ELs' own, not op-up's 8545/8546 proxy, which
// speaks one JSON-RPC object per request and cannot serve a batching client. And in-process is the
// only place a private-interop pair can be smoked at all: a message initiated on the private chain
// is named by its position on the rendering, and that correction lives in a resolver the devstack
// registers in the process that built the pair.
func runSmoke(ctx context.Context, stderr io.Writer, sys *presets.TwoL2SupernodeInterop) error {
	privKeyHex, _, err := funderAccount()
	if err != nil {
		return err
	}
	return interopsmoke.Run(ctx, stderr, interopsmoke.Config{
		L2AURL:     sys.L2ELA.Escape().UserRPC(),
		L2BURL:     sys.L2ELB.Escape().UserRPC(),
		PrivateKey: privKeyHex,
		// Read off the system rather than off the command line: the topology is what decides how
		// the private chain's messages are named, not what the operator typed.
		PrivatePairB: sys.PrivateInterop != nil,
	})
}

// funderAccount is the prefunded devkeys account op-up hands out, as a hex secret and its address.
func funderAccount() (privKeyHex string, address common.Address, err error) {
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	if err != nil {
		return "", common.Address{}, fmt.Errorf("new mnemonic dev keys: %w", err)
	}
	const funderIndex = 10_000
	funderUserKey := devkeys.UserKey(funderIndex)
	funderAddress, err := hd.Address(funderUserKey)
	if err != nil {
		return "", common.Address{}, fmt.Errorf("address: %w", err)
	}
	funderPrivKey, err := hd.Secret(funderUserKey)
	if err != nil {
		return "", common.Address{}, fmt.Errorf("secret: %w", err)
	}
	return "0x" + common.Bytes2Hex(crypto.FromECDSA(funderPrivKey)), funderAddress, nil
}

func printAccountInfo(stderr io.Writer) error {
	privKeyHex, address, err := funderAccount()
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Test Account Address: %s\n", address)
	fmt.Fprintf(stderr, "Test Account Private Key: %s\n", privKeyHex)
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

				// Cross-safe (safe head) -- reorg shows as decrease
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

// proxyEL is a hacky way to intercept EL json rpc requests for logging to get around log filtering
// bugs.
func proxyEL(ctx context.Context, stderr io.Writer, addr string, client client.RPC) error {
	mux := http.NewServeMux()
	// Set up the HTTP handler for all incoming requests.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Ensure the request method is POST, as JSON RPC typically uses POST.
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the entire request body.
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close() // Close the request body after reading

		// Parse the incoming JSON RPC request. We use a map to dynamically
		// extract the method, parameters, and ID.
		var req map[string]any
		if err := json.Unmarshal(requestBody, &req); err != nil {
			http.Error(w, "Invalid JSON RPC request format", http.StatusBadRequest)
			return
		}

		// Extract the RPC method name.
		method, ok := req["method"].(string)
		if !ok {
			http.Error(w, "Missing or invalid 'method' field in JSON RPC request", http.StatusBadRequest)
			return
		}

		// Extract RPC parameters. JSON RPC parameters can be an array, an object, or null/missing.
		var callParams []any
		if p, ok := req["params"]; ok && p != nil {
			if arr, isArray := p.([]any); isArray {
				// If parameters are an array, spread them directly.
				callParams = arr
			} else if obj, isObject := p.(map[string]any); isObject {
				// If parameters are a JSON object, pass the entire object as a single argument.
				callParams = []any{obj}
			} else {
				http.Error(w, "Invalid 'params' field in JSON RPC request (must be array, object, or null)", http.StatusBadRequest)
				return
			}
		}
		// If 'params' is missing or null, `callParams` remains empty, which is correct for methods without parameters.

		// Extract the request ID. This is crucial for matching responses to requests.
		id := req["id"] // ID can be string, number, or null. We don't need to check `ok` for this.

		// Prepare a variable to hold the RPC response result.
		// `json.RawMessage` is used to capture the raw JSON value from the backend
		// without needing to know its specific Go type beforehand.
		var rpcResult json.RawMessage

		// Create a context with a timeout for the RPC call to the backend.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30-second timeout
		defer cancel()                                                           // Ensure the context is cancelled to release resources

		fmt.Fprintf(stderr, "%s\n", method)

		// Use the rpc.Client to make the actual call to the backend Ethereum node.
		// The `callParams...` syntax unpacks the slice into variadic arguments.
		err = client.CallContext(ctx, &rpcResult, method, callParams...)
		if err != nil {
			message := fmt.Sprintf("RPC call to backend failed for method '%s': %v", method, err)
			// If the RPC call to the backend fails, construct a JSON RPC error response.
			rpcErr := map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]any{
					"code":    -32000, // Standard JSON RPC server error code for internal errors
					"message": message,
				},
			}
			fmt.Fprintf(stderr, "RPC error: %s\n", message)
			jsonResponse, _ := json.Marshal(rpcErr) // Marshaling error is unlikely here, so we ignore it.
			w.Header().Set("Content-Type", "application/json")
			// For JSON-RPC, errors are typically returned with an HTTP 200 OK status,
			// with the error details within the JSON payload.
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(jsonResponse); err != nil {
				return
			}
			return
		}

		// If the RPC call was successful, construct the JSON RPC success response.
		responseMap := map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  rpcResult, // The raw JSON result from the backend node
		}

		jsonResponse, err := json.Marshal(responseMap)
		if err != nil {
			http.Error(w, "Failed to marshal RPC success response", http.StatusInternalServerError)
			return
		}

		// Set the Content-Type header and write the successful JSON RPC response.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(jsonResponse); err != nil {
			return
		}
	})

	server := &http.Server{Addr: addr, Handler: mux}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown proxy server: %w", err)
		}
		return <-errCh
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
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
