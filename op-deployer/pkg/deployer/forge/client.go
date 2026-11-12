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
	"runtime"
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
	// FsPermissions are the file accesses of a script run. The project config's are ignored.
	FsPermissions []FsPermission
	// Timeout bounds each forge invocation. Zero means no limit.
	Timeout time.Duration
	// Env holds KEY=VALUE variables for each forge command, on top of the inherited environment.
	Env []string
	// AnyCacheProfile lets a script run take the compile settings of the project profile that the
	// compile cache was built with, such as a lite build's, instead of only the default profile's.
	// Such a build can hold contracts above the EIP-170 code size limit, so set it only for runs
	// that deploy no contract to a chain, such as an L2 genesis without a code size limit.
	AnyCacheProfile bool
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

// ArtifactsReadPermission lets the scripts of a contracts bundle read its artifacts, as
// DeployUtils.getCode does.
var ArtifactsReadPermission = FsPermission{Access: "read", Path: "./forge-artifacts/"}

// compileCacheFile is forge's compile cache, relative to the project.
const compileCacheFile = "cache/solidity-files-cache.json"

var (
	// ErrCompiled is returned by RunScript when forge starts to compile the project.
	ErrCompiled = errors.New("forge compiled the project")
	// ErrScriptReverted is returned by RunScript when the script itself reverts. Any other failure,
	// such as an RPC error or a failed cheatcode, may pass when the script runs again.
	ErrScriptReverted = errors.New("forge script reverted")
)

// scriptFailed precedes the reason that forge gives for a failed script.
const scriptFailed = "Error: script failed: "

// scriptReverted reports whether forge's stderr shows that the script itself reverted. forge also
// reports a failed cheatcode as a failed script, with the reason "vm.<cheatcode>: ...", for example
// when the RPC refuses the cheatcode's request or answers it with an HTTP error. That is not a revert.
func scriptReverted(stderr string) bool {
	for _, line := range strings.Split(stderr, "\n") {
		if _, reason, ok := strings.Cut(line, scriptFailed); ok {
			return !strings.HasPrefix(reason, "vm.")
		}
	}
	return false
}

// RunScript runs a prebuilt script without a compile. It returns forge's output and the transactions
// that the script would send, which forge records in the run's private dry-run file. Not --offline:
// offline, forge requires the solc versions of the cache installed even when it compiles nothing.
// Each run gets its own copy of the compile cache and its own broadcast directory, so concurrent
// runs write no shared file.
func (c *Client) RunScript(ctx context.Context, script string, sig string, args []byte, opts ...string) (string, []Transaction, error) {
	scratch, err := os.MkdirTemp("", "op-deployer-forge-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create forge scratch directory: %w", err)
	}
	defer os.RemoveAll(scratch)
	cacheDir := filepath.Join(scratch, "cache")
	if err := copyCompileCache(filepath.Join(c.Wd, compileCacheFile), cacheDir); err != nil {
		return "", nil, err
	}

	root, err := filepath.Abs(c.Wd)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve the forge project: %w", err)
	}
	configPath, built, err := c.sandboxConfig(root, scratch)
	if err != nil {
		return "", nil, err
	}
	if built != "" {
		built = " (it matches " + built + ")"
	}

	// forge runs in the scratch directory, so it does not load a .env file of the project. A target
	// that names only a contract stays as it is: forge looks it up in the project.
	if file, contract, ok := strings.Cut(script, ":"); (ok || strings.HasSuffix(file, ".sol")) && !filepath.IsAbs(file) {
		script = filepath.Join(root, file)
		if ok {
			script += ":" + contract
		}
	}

	buf := new(bytes.Buffer)
	cliOpts := []string{"script", "--root", root, "--config-path", configPath, "--cache-path", cacheDir}
	cliOpts = append(cliOpts, opts...)
	cliOpts = append(cliOpts, "--sig", sig, script, "0x"+hex.EncodeToString(args))
	release, err := runSlots.acquire(ctx)
	if err != nil {
		return "", nil, err
	}
	defer release()
	run := c.WithEnv("FOUNDRY_BROADCAST="+filepath.Join(scratch, "broadcast"), "FOUNDRY_PROFILE=default")
	// forge compiles before it runs the script, so a stop at the compile precedes any broadcast.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	watcher := &compileWatcher{cancel: cancel}
	compileErr := fmt.Errorf("%w in %s, so its compile cache is not up to date%s: build it with `just build-no-tests` in packages/contracts-bedrock, or use the embedded artifacts", ErrCompiled, c.Wd, built)
	// forge prints the revert reason of a failed script to stderr.
	errBuf := new(bytes.Buffer)
	err = run.execWithEnv(ctx, scratch, inheritedEnv(), io.MultiWriter(buf, watcher), errBuf, cliOpts...)
	if watcher.compiled.Load() {
		return "", nil, compileErr
	}
	if err != nil {
		stderr := strings.TrimSpace(errBuf.String())
		if scriptReverted(stderr) {
			err = fmt.Errorf("%w: %w", ErrScriptReverted, err)
		}
		return "", nil, fmt.Errorf("failed to execute forge script: %w\n%s", err, stderr)
	}
	if strings.Contains(buf.String(), "Compiler run successful") {
		return "", nil, compileErr
	}
	transactions, err := readTransactions(filepath.Join(scratch, "broadcast"), buf.String())
	if err != nil {
		return "", nil, err
	}
	return buf.String(), transactions, nil
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

// sandboxKeys are the keys of the project's default profile that its sandbox config keeps: the
// settings that the compile cache records, which must match or forge compiles, and the limits of a
// script run.
var sandboxKeys = []string{
	"src", "test", "script", "out", "libs", "remappings", "auto_detect_remappings", "allow_paths",
	"include_paths", "skip", "libraries", "evm_version", "optimizer", "optimizer_runs",
	"optimizer_details", "via_ir", "via_ssa_cfg", "bytecode_hash", "cbor_metadata", "revert_strings",
	"use_literal_content", "ast", "model_checker", "extra_output", "extra_output_files", "build_info",
	"build_info_path", "additional_compiler_profiles", "compilation_restrictions", "sparse_mode",
	"dynamic_test_linking", "ignored_error_codes", "ignored_error_codes_from", "ignored_warnings_from",
	"gas_limit", "memory_limit", "code_size_limit", "disable_block_gas_limit", "enable_tx_gas_limit",
	"isolate",
}

// solcVersionKeys are the keys that pin the compiler. The compile cache records the version, so a
// pinned version is kept; a path would make forge run that binary.
var solcVersionKeys = []string{"solc", "solc_version"}

var solcVersionRegexp = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// sandboxConfig writes the forge config of a script run into dir: a contracts bundle is a forge
// project, so its config is not trusted. It keeps only the sandboxKeys and a pinned compiler version
// of the project's default profile, with AnyCacheProfile the compile settings of the profile that
// the compile cache was built with, so no extends, RPC URL or sender, and it turns ffi off, grants
// only FsPermissions, denies nothing and makes libs absolute. It sets root, because forge otherwise
// takes the root from the config's directory and recompiles; forge warns that root is unknown, but
// uses it. It also returns which profile the compile cache matches, if it can tell.
func (c *Client) sandboxConfig(root string, dir string) (string, string, error) {
	if err := c.checkFsPermissions(root); err != nil {
		return "", "", err
	}
	var project struct {
		Profile map[string]map[string]any `toml:"profile"`
	}
	if _, err := toml.DecodeFile(filepath.Join(root, "foundry.toml"), &project); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", "", fmt.Errorf("failed to read the forge config of the project: %w", err)
	}
	// The copy of the compile cache that forge reads; an unreadable one matches no profile.
	cache, _ := os.ReadFile(filepath.Join(dir, compileCacheFile))
	settings, built := compileProfile(project.Profile, cache, c.AnyCacheProfile)
	profile := map[string]any{
		"root":           root,
		"ffi":            false,
		"fs_permissions": append([]FsPermission{}, c.FsPermissions...),
		// forge ignores its compile cache when it denies warnings, so it would compile the project.
		"deny": "never",
	}
	for key, value := range settings {
		if slices.Contains(solcVersionKeys, key) {
			if version, ok := value.(string); !ok || !solcVersionRegexp.MatchString(version) {
				return "", "", fmt.Errorf("the forge config of the project sets %s to %v; only a compiler version such as 0.8.15 is allowed", key, value)
			}
		} else if !slices.Contains(sandboxKeys, key) {
			continue
		}
		profile[key] = value
	}
	// forge detects the remappings of relative libs from the config's directory, not the root; without
	// them it finds the compile cache stale and compiles.
	libs := []any{"lib"} // forge's default when the project has a lib directory
	if value, set := profile["libs"]; set {
		var ok bool
		if libs, ok = value.([]any); !ok {
			return "", "", fmt.Errorf("the forge config of the project sets libs to %v; only a list of paths is allowed", value)
		}
	}
	// forge resolves links in the root, so the libs must too.
	libRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve the forge project: %w", err)
	}
	absLibs := make([]string, 0, len(libs))
	for _, lib := range libs {
		lib, ok := lib.(string)
		if !ok {
			return "", "", fmt.Errorf("the forge config of the project sets libs to %v; only a list of paths is allowed", profile["libs"])
		}
		if !filepath.IsAbs(lib) {
			lib = filepath.Join(libRoot, lib)
		}
		absLibs = append(absLibs, lib)
	}
	profile["libs"] = absLibs

	path := filepath.Join(dir, "foundry.toml")
	f, err := os.Create(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to create the forge config: %w", err)
	}
	defer f.Close()
	config := map[string]any{"profile": map[string]any{"default": profile}}
	if err := toml.NewEncoder(f).Encode(config); err != nil {
		return "", "", fmt.Errorf("failed to write the forge config: %w", err)
	}
	return path, built, nil
}

// checkFsPermissions refuses a link at or under a relative FsPermissions path. forge grants access to
// the target of a link, and when a chain of links resolves to a path longer than PATH_MAX, forge
// checks the unresolved path instead, so a link in the project could let its scripts read any file.
// A relative path is a part of the project, which is not trusted; an absolute path is a directory
// that op-deployer made. The directories above a relative path are not checked: op-deployer grants
// only forge-artifacts, at the project root.
func (c *Client) checkFsPermissions(root string) error {
	for _, p := range c.FsPermissions {
		if filepath.IsAbs(p.Path) {
			continue
		}
		err := filepath.WalkDir(filepath.Join(root, p.Path), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return fmt.Errorf("%s in the forge project is a link; the scripts may read only files of the project", path)
			}
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to check %s in the forge project: %w", p.Path, err)
		}
	}
	return nil
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
	if err := c.execWithEnv(ctx, c.Wd, withGoCertDirs(os.Environ()), buf, buf, cliOpts...); err != nil {
		return buf.String(), fmt.Errorf("failed to verify contract: %w", err)
	}
	return buf.String(), nil
}

func (c *Client) execCmd(ctx context.Context, stdout io.Writer, stderr io.Writer, args ...string) error {
	return c.execWithEnv(ctx, c.Wd, inheritedEnv(), stdout, stderr, args...)
}

// execWithEnv runs forge in dir with env and c.Env.
func (c *Client) execWithEnv(ctx context.Context, dir string, env []string, stdout io.Writer, stderr io.Writer, args ...string) error {
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
	cmd.Dir = dir
	cmd.Env = append(env, c.Env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run forge command: %w", err)
	}
	return nil
}

// inheritedEnvNames are the variables that forge inherits, except for verify-contract: what it needs
// to run and to reach the network. A script can read the environment, and FOUNDRY_*,
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
	return withGoCertDirs(env)
}

// goCertDirs are the directories that Go reads trusted certificates from on Linux when SSL_CERT_DIR
// is unset or empty.
const goCertDirs = "/etc/ssl/certs:/etc/pki/tls/certs"

// withGoCertDirs sets SSL_CERT_DIR to goCertDirs on Linux when env does not set it to a non-empty
// value, so that forge trusts the certificates that op-deployer trusts. Without it, forge 1.8.3 reads
// only the system bundle file: a CA certificate mounted into /etc/ssl/certs, as a Kubernetes
// ConfigMap often is, works for op-deployer, and fails forge's RPC calls with UnknownIssuer. With it,
// forge reads only these directories, which hold the system bundle on the common distributions.
func withGoCertDirs(env []string) []string {
	if runtime.GOOS != "linux" {
		return env
	}
	for _, kv := range env {
		if name, value, _ := strings.Cut(kv, "="); name == "SSL_CERT_DIR" && value != "" {
			return env
		}
	}
	return append(env, "SSL_CERT_DIR="+goCertDirs)
}

type ScriptCallEncoder[I any] interface {
	Encode(I) ([]byte, error)
}

type ScriptCallDecoder[O any] interface {
	Decode(raw []byte) (O, error)
}

// ScriptCaller calls a forge script. It returns the decoded return value and the transactions that
// the script would send.
type ScriptCaller[I any, O any] func(ctx context.Context, input I, opts ...string) (O, []Transaction, error)

func NewScriptCaller[I any, O any](client *Client, script string, sig string, encoder ScriptCallEncoder[I], decoder ScriptCallDecoder[O]) ScriptCaller[I, O] {
	return func(ctx context.Context, input I, opts ...string) (O, []Transaction, error) {
		var out O
		encArgs, err := encoder.Encode(input)
		if err != nil {
			return out, nil, fmt.Errorf("failed to encode forge args: %w", err)
		}
		rawOut, transactions, err := client.RunScript(ctx, script, sig, encArgs, opts...)
		if err != nil {
			return out, nil, fmt.Errorf("failed to run forge script: %w", err)
		}
		sigilMatches := sigilRegexp.FindAllStringSubmatch(rawOut, -1)
		if len(sigilMatches) != 1 || len(sigilMatches[0]) != 2 {
			return out, nil, fmt.Errorf("failed to find forge return value in output:\n%s", rawOut)
		}
		decoded, err := hex.DecodeString(sigilMatches[0][1])
		if err != nil {
			return out, nil, fmt.Errorf("failed to decode forge return value %s: %w", sigilMatches[0][1], err)
		}
		out, err = decoder.Decode(decoded)
		if err != nil {
			return out, nil, fmt.Errorf("failed to decode forge output: %w", err)
		}
		return out, transactions, nil
	}
}
