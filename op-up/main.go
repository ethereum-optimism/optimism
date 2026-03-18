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
	"sync"
	"syscall"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
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
	app.Flags = cliapp.ProtectFlags([]cli.Flag{dirFlag})
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
		return runOpUp(cliCtx.Context, cliCtx.App.ErrWriter, cliCtx.String(dirFlag.Name))
	}
	return app.RunContext(ctx, args)
}

func runOpUp(ctx context.Context, stderr io.Writer, opUpDir string) error {
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

	sys, err := newMinimalSystem(t)
	if err != nil {
		return err
	}
	if err := runSystem(ctx, stderr, sys); err != nil {
		return err
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
	return presets.NewMinimal(t), nil
}

func runSystem(ctx context.Context, stderr io.Writer, sys *presets.Minimal) error {
	// Print available account.
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
	fmt.Fprintf(stderr, "EL Node URL: %s\n", "http://localhost:8545")

	elNode := sys.L2EL

	// Log on new blocks.
	go func() {
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
					fmt.Fprintf(stderr, "New L2 block: number %d, hash %s\n", unsafe.Number, unsafe.Hash)
					lastBlock = unsafe.Number
				}
			}
		}
	}()

	// Proxy L2 EL requests.
	go func() {
		if err := proxyEL(ctx, stderr, elNode.Escape().L2EthClient().RPC()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "error: %v", err)
		}
	}()

	<-ctx.Done()

	return nil
}

// proxyEL is a hacky way to intercept EL json rpc requests for logging to get around log filtering
// bugs.
func proxyEL(ctx context.Context, stderr io.Writer, client client.RPC) error {
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

	server := &http.Server{Addr: "localhost:8545", Handler: mux}
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
	dir, err := os.MkdirTemp(t.state.tempRoot, "op-up-*")
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
