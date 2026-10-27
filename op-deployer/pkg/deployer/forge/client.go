package forge

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
)

// waitDelay bounds how long execWithEnv waits for the output pipes to close after
// forge exits or its context is canceled, so a killed forge whose children
// still hold the pipes cannot block Wait forever. Variable to allow tests to
// shorten it.
var waitDelay = 30 * time.Second

var (
	versionRegexp = regexp.MustCompile(`(?i)forge version: (.*)\ncommit sha: ([a-f0-9]+)\n`)
	sigilRegexp   = regexp.MustCompile(`(?i)== Return ==\n0: bytes 0x([a-f0-9]*)\n`)
)

type VersionInfo struct {
	Semver string
	SHA    string
}

type Client struct {
	Binary Binary
	Stdout io.Writer
	Stderr io.Writer
	Wd     string
	// FsPermissions, when not nil, replaces the fs_permissions of the project config.
	FsPermissions []FsPermission
	// Timeout bounds each forge invocation. Zero means no limit.
	Timeout time.Duration
	// Env holds KEY=VALUE variables for each forge command, on top of the inherited environment.
	Env []string
	// capture is set only on the private client copy used by CaptureTransactions.
	capture *[]Transaction
}

func NewStandardClient(workdir string, binOpts ...StandardBinOpt) (*Client, error) {
	forgeBinary, err := NewStandardBinary(binOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize forge binary: %w", err)
	}
	if err := forgeBinary.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure forge binary: %w", err)
	}

	forgeClient := NewClient(forgeBinary)

	// Determine the working directory for forge
	// The artifacts FS points to a subdirectory (e.g., "forge-artifacts" or "out"),
	// but forge needs to run from the parent directory where foundry.toml is located.
	// This matches the structure from ExtractEmbedded/ExtractFromFile where:
	// - untarPath/forge-artifacts/ contains the artifacts
	// - untarPath/foundry.toml is the config file
	// This can be removed once we remove the OPCM code and use the artifacts FS directly.
	var forgeWd string
	info, err := os.Stat(workdir)
	if err != nil {
		forgeWd = workdir
	} else if info.IsDir() {
		foundryToml := filepath.Join(workdir, "foundry.toml")
		if _, err := os.Stat(foundryToml); err == nil {
			forgeWd = workdir
		} else {
			// foundry.toml not found, check parent directory
			// This handles the case where workdir points to forge-artifacts/ or out/
			parent := filepath.Dir(workdir)
			parentFoundryToml := filepath.Join(parent, "foundry.toml")
			if _, err := os.Stat(parentFoundryToml); err == nil {
				forgeWd = parent
			} else {
				forgeWd = workdir
			}
		}
	} else {
		forgeWd = filepath.Dir(workdir)
	}

	if err := os.MkdirAll(forgeWd, 0o755); err != nil {
		return nil, fmt.Errorf("failed to ensure forge working directory exists: %w", err)
	}

	forgeClient.Wd = forgeWd

	return forgeClient, nil
}

// NewClient returns a client for binary. forge output goes to stderr; stdout is for results.
func NewClient(binary Binary) *Client {
	return &Client{
		Binary: binary,
		Stdout: os.Stderr,
		Stderr: os.Stderr,
	}
}

// WithEnv returns a copy of the client with the given KEY=VALUE variables added.
func (c *Client) WithEnv(env ...string) *Client {
	cp := *c
	cp.Env = append(slices.Clone(c.Env), env...)
	return &cp
}

func (c *Client) Version(ctx context.Context) (VersionInfo, error) {
	buf := new(bytes.Buffer)
	if err := c.execCmd(ctx, buf, io.Discard, "--version"); err != nil {
		return VersionInfo{}, fmt.Errorf("failed to run forge version command: %w", err)
	}
	outputStr := buf.String()
	matches := versionRegexp.FindAllStringSubmatch(outputStr, -1)
	if len(matches) != 1 || len(matches[0]) != 3 {
		return VersionInfo{}, fmt.Errorf("failed to find forge version in output:\n%s", outputStr)
	}
	return VersionInfo{
		Semver: matches[0][1],
		SHA:    matches[0][2],
	}, nil
}

func (c *Client) Build(ctx context.Context, opts ...string) error {
	return c.execCmd(ctx, io.Discard, io.Discard, append([]string{"build"}, opts...)...)
}

func (c *Client) Clean(ctx context.Context, opts ...string) error {
	return c.execCmd(ctx, io.Discard, io.Discard, append([]string{"clean"}, opts...)...)
}

// FsPermission is one entry of the fs_permissions of a forge config.
type FsPermission struct {
	Access string `toml:"access"`
	Path   string `toml:"path"`
}

// compileCacheFile is forge's compile cache, relative to the project.
const compileCacheFile = "cache/solidity-files-cache.json"

// CaptureTransactions runs one script on a private copy of the client and returns only the
// transactions recorded by forge. The callback must not broadcast: Go owns live delivery.
func (c *Client) CaptureTransactions(run func(*Client) error) ([]Transaction, error) {
	var transactions []Transaction
	client := *c
	client.capture = &transactions
	if err := run(&client); err != nil {
		return nil, err
	}
	if transactions == nil {
		return nil, errors.New("forge did not capture a script run")
	}
	return transactions, nil
}

// RunScript runs a prebuilt script without a compile. Not --offline: offline, forge requires the solc
// versions of the cache installed even when it compiles nothing. Each run gets its own copy of the
// compile cache and its own broadcast directory, so concurrent runs write no shared file.
func (c *Client) RunScript(ctx context.Context, script string, sig string, args []byte, opts ...string) (string, error) {
	if c.capture != nil {
		if *c.capture != nil {
			return "", errors.New("transaction capture requires exactly one script run")
		}
		for _, opt := range opts {
			name, _, _ := strings.Cut(opt, "=")
			if name == "--broadcast" || name == "--resume" {
				return "", errors.New("transaction capture requires a keyless dry run, not broadcast or resume")
			}
		}
	}
	scratch, err := os.MkdirTemp("", "op-deployer-forge-")
	if err != nil {
		return "", fmt.Errorf("failed to create forge scratch directory: %w", err)
	}
	defer os.RemoveAll(scratch)
	cacheDir := filepath.Join(scratch, "cache")
	if err := copyCompileCache(filepath.Join(c.Wd, compileCacheFile), cacheDir); err != nil {
		return "", err
	}

	root, err := filepath.Abs(c.Wd)
	if err != nil {
		return "", fmt.Errorf("failed to resolve the forge project: %w", err)
	}
	configPath, err := c.sandboxConfig(root, scratch)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	cliOpts := []string{"script", "--root", root, "--config-path", configPath, "--cache-path", cacheDir}
	cliOpts = append(cliOpts, opts...)
	// --slow: forge otherwise waits forever on a receipt when the node drops one of several transactions
	// sent at once. The node's gas estimate, as with txmgr: forge's own simulation does not know every
	// L1 fork (it underestimated a Glamsterdam L1), and its default +30% lifts the OPCM deployment over
	// the Osaka per-transaction gas cap (EIP-7825).
	if slices.Contains(opts, "--broadcast") {
		cliOpts = append(cliOpts, "--slow", "--skip-simulation", "--gas-estimate-multiplier", "100")
	}
	cliOpts = append(cliOpts, "--sig", sig, script, "0x"+hex.EncodeToString(args))
	release, err := runSlots.acquire(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	run := c.WithEnv("FOUNDRY_BROADCAST=" + filepath.Join(scratch, "broadcast"))
	// forge compiles before it runs the script, so a stop at the compile precedes any broadcast.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	watcher := &compileWatcher{cancel: cancel}
	compileErr := fmt.Errorf("forge compiled the project in %s; its compile cache must be up to date", c.Wd)
	// forge prints the revert reason of a failed script to stderr.
	errBuf := new(bytes.Buffer)
	err = run.execCmd(ctx, io.MultiWriter(buf, watcher), errBuf, cliOpts...)
	if watcher.compiled.Load() {
		return "", compileErr
	}
	if err != nil {
		return "", fmt.Errorf("failed to execute forge script: %w\n%s", err, strings.TrimSpace(errBuf.String()))
	}
	if strings.Contains(buf.String(), "Compiler run successful") {
		return "", compileErr
	}
	if c.capture != nil {
		transactions, err := readTransactions(filepath.Join(scratch, "broadcast"), buf.String())
		if err != nil {
			return "", err
		}
		*c.capture = transactions
	}
	return buf.String(), nil
}

// compileStart is what forge prints when it starts to compile.
var compileStart = []byte("Compiling ")

// compileWatcher calls cancel when forge starts to compile.
type compileWatcher struct {
	cancel   context.CancelFunc
	compiled atomic.Bool
	// tail lets a match span two writes.
	tail []byte
}

func (w *compileWatcher) Write(p []byte) (int, error) {
	if w.compiled.Load() {
		return len(p), nil
	}
	data := append(w.tail, p...)
	if bytes.Contains(data, compileStart) {
		w.compiled.Store(true)
		w.cancel()
		return len(p), nil
	}
	keep := min(len(data), len(compileStart)-1)
	w.tail = append(w.tail[:0], data[len(data)-keep:]...)
	return len(p), nil
}

// sandboxConfig writes a copy of the project config into dir, with ffi off and FsPermissions when
// set: a contracts bundle is a forge project, so its config is not trusted. The copy sets root,
// because forge otherwise takes the root from the config's directory and recompiles; forge warns
// that root is unknown, but uses it. forge runs the default profile, not FOUNDRY_PROFILE.
func (c *Client) sandboxConfig(root string, dir string) (string, error) {
	config := make(map[string]any)
	if _, err := toml.DecodeFile(filepath.Join(root, "foundry.toml"), &config); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("failed to read the forge config of the project: %w", err)
	}
	profiles, ok := config["profile"].(map[string]any)
	if !ok {
		profiles = make(map[string]any)
		config["profile"] = profiles
	}
	profile, ok := profiles["default"].(map[string]any)
	if !ok {
		profile = make(map[string]any)
		profiles["default"] = profile
	}
	profile["root"] = root
	profile["ffi"] = false
	if c.FsPermissions != nil {
		profile["fs_permissions"] = c.FsPermissions
	}

	path := filepath.Join(dir, "foundry.toml")
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create the forge config: %w", err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(config); err != nil {
		return "", fmt.Errorf("failed to write the forge config: %w", err)
	}
	return path, nil
}

func copyCompileCache(src string, dstDir string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read the compile cache of the forge project: %w", err)
	}
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return fmt.Errorf("failed to create the compile cache directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, filepath.Base(src)), data, 0o600); err != nil {
		return fmt.Errorf("failed to copy the compile cache: %w", err)
	}
	return nil
}

func (c *Client) VerifyContract(ctx context.Context, opts ...string) (string, error) {
	buf := new(bytes.Buffer)
	cliOpts := []string{"verify-contract"}
	cliOpts = append(cliOpts, opts...)
	// forge reads the verifier API key, such as ETHERSCAN_API_KEY, from its environment.
	if err := c.execWithEnv(ctx, os.Environ(), buf, buf, cliOpts...); err != nil {
		return buf.String(), fmt.Errorf("failed to verify contract: %w", err)
	}
	return buf.String(), nil
}

func (c *Client) execCmd(ctx context.Context, stdout io.Writer, stderr io.Writer, args ...string) error {
	return c.execWithEnv(ctx, inheritedEnv(), stdout, stderr, args...)
}

// execWithEnv runs forge with env and c.Env.
func (c *Client) execWithEnv(ctx context.Context, env []string, stdout io.Writer, stderr io.Writer, args ...string) error {
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	if err := c.Binary.Ensure(ctx); err != nil {
		return fmt.Errorf("failed to ensure binary: %w", err)
	}

	cmd := exec.CommandContext(ctx, c.Binary.Path(), args...)
	cmd.WaitDelay = waitDelay
	cStdout := c.Stdout
	if cStdout == nil {
		cStdout = os.Stdout
	}
	cStderr := c.Stderr
	if cStderr == nil {
		cStderr = os.Stderr
	}

	mwStdout := io.MultiWriter(cStdout, stdout)
	mwStderr := io.MultiWriter(cStderr, stderr)
	cmd.Stdout = mwStdout
	cmd.Stderr = mwStderr
	cmd.Dir = c.Wd
	cmd.Env = append(env, c.Env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run forge command: %w", err)
	}
	return nil
}

// inheritedEnvNames are the variables that forge (except verify-contract) and anvil inherit: what
// they need to run and to reach the network. A script can read the environment, and FOUNDRY_*,
// DAPP_* and ETH_* variables could make a run compile or target another chain.
var inheritedEnvNames = []string{
	"PATH", "HOME", "USER", "LOGNAME", "TMPDIR", "TMP", "TEMP", "LANG", "LC_ALL", "LC_CTYPE", "TZ",
	"XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "SSL_CERT_FILE", "SSL_CERT_DIR",
	"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "no_proxy", "all_proxy",
}

// inheritedEnv returns the inheritedEnvNames variables of this process.
func inheritedEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if slices.Contains(inheritedEnvNames, name) {
			env = append(env, kv)
		}
	}
	return env
}

type ScriptCallEncoder[I any] interface {
	Encode(I) ([]byte, error)
}

type ScriptCallDecoder[O any] interface {
	Decode(raw []byte) (O, error)
}

// ScriptCaller calls a forge script and returns its decoded return value.
type ScriptCaller[I any, O any] func(ctx context.Context, input I, opts ...string) (O, error)

func NewScriptCaller[I any, O any](client *Client, script string, sig string, encoder ScriptCallEncoder[I], decoder ScriptCallDecoder[O]) ScriptCaller[I, O] {
	return func(ctx context.Context, input I, opts ...string) (O, error) {
		var out O
		encArgs, err := encoder.Encode(input)
		if err != nil {
			return out, fmt.Errorf("failed to encode forge args: %w", err)
		}
		rawOut, err := client.RunScript(ctx, script, sig, encArgs, opts...)
		if err != nil {
			return out, fmt.Errorf("failed to run forge script: %w", err)
		}
		sigilMatches := sigilRegexp.FindAllStringSubmatch(rawOut, -1)
		if len(sigilMatches) != 1 || len(sigilMatches[0]) != 2 {
			return out, fmt.Errorf("failed to find forge return value in output:\n%s", rawOut)
		}
		decoded, err := hex.DecodeString(sigilMatches[0][1])
		if err != nil {
			return out, fmt.Errorf("failed to decode forge return value %s: %w", sigilMatches[0][1], err)
		}
		out, err = decoder.Decode(decoded)
		if err != nil {
			return out, fmt.Errorf("failed to decode forge output: %w", err)
		}
		return out, nil
	}
}
