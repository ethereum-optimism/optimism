package forge

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

// waitDelay bounds how long execCmd waits for the output pipes to close after
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
	// Timeout bounds each forge invocation. Zero means no limit.
	Timeout time.Duration
	// Env holds environment variables, in KEY=VALUE form, that each forge command gets in
	// addition to the environment of this process.
	Env []string
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

// NewClient returns a client that runs binary. The output of forge goes to stderr, because a
// command of op-deployer can write its result to stdout.
func NewClient(binary Binary) *Client {
	return &Client{
		Binary: binary,
		Stdout: os.Stderr,
		Stderr: os.Stderr,
	}
}

// WithEnv returns a copy of the client that also sets the given KEY=VALUE environment variables.
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

// compileCacheFile is forge's compile cache, relative to the project.
const compileCacheFile = "cache/solidity-files-cache.json"

// RunScript runs a script of the project, offline and without a compile: the project must hold the
// compiled scripts and their compile cache. The run gets its own copy of the compile cache and its
// own broadcast directory, so concurrent runs in one project write no shared file, and the run
// leaves nothing in the project.
func (c *Client) RunScript(ctx context.Context, script string, sig string, args []byte, opts ...string) (string, error) {
	scratch, err := os.MkdirTemp("", "op-deployer-forge-")
	if err != nil {
		return "", fmt.Errorf("failed to create forge scratch directory: %w", err)
	}
	defer os.RemoveAll(scratch)
	cacheDir := filepath.Join(scratch, "cache")
	if err := copyCompileCache(filepath.Join(c.Wd, compileCacheFile), cacheDir); err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	cliOpts := []string{"script", "--offline", "--cache-path", cacheDir}
	cliOpts = append(cliOpts, opts...)
	// A broadcast sends one transaction at a time, and waits for its receipt before it sends the
	// next. forge otherwise waits forever on the receipt of a transaction that raced an in-flight one
	// out of the node's pool.
	if slices.Contains(opts, "--broadcast") {
		cliOpts = append(cliOpts, "--slow")
	}
	cliOpts = append(cliOpts, "--sig", sig, script, "0x"+hex.EncodeToString(args))
	run := c.WithEnv("FOUNDRY_BROADCAST=" + filepath.Join(scratch, "broadcast"))
	// forge compiles before it runs the script, so stopping it when the compile starts also stops it
	// before any broadcast.
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
	return buf.String(), nil
}

// compileStart is what forge prints when it starts to compile.
var compileStart = []byte("Compiling ")

// compileWatcher reads the output of forge and calls cancel when forge starts to compile.
type compileWatcher struct {
	cancel   context.CancelFunc
	compiled atomic.Bool
	// tail holds the end of the previous write, so that a match can span two writes.
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

func copyCompileCache(src string, dstDir string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read the compile cache of the forge project: %w", err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("failed to create the compile cache directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, filepath.Base(src)), data, 0o644); err != nil {
		return fmt.Errorf("failed to copy the compile cache: %w", err)
	}
	return nil
}

func (c *Client) VerifyContract(ctx context.Context, opts ...string) (string, error) {
	buf := new(bytes.Buffer)
	cliOpts := []string{"verify-contract"}
	cliOpts = append(cliOpts, opts...)
	if err := c.execCmd(ctx, buf, buf, cliOpts...); err != nil {
		return buf.String(), fmt.Errorf("failed to verify contract: %w", err)
	}
	return buf.String(), nil
}

func (c *Client) execCmd(ctx context.Context, stdout io.Writer, stderr io.Writer, args ...string) error {
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
	cmd.Env = append(slices.DeleteFunc(cmd.Environ(), isForgeConfig), c.Env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run forge command: %w", err)
	}
	return nil
}

// isForgeConfig reports whether an environment variable configures forge. forge reads its
// configuration from FOUNDRY_*, DAPP_* and ETH_* variables, which could make a run compile, or run
// against another chain than the one op-deployer selects, so forge does not inherit them.
func isForgeConfig(kv string) bool {
	return strings.HasPrefix(kv, "FOUNDRY_") || strings.HasPrefix(kv, "DAPP_") || strings.HasPrefix(kv, "ETH_")
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
