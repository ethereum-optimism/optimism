package rustengine

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
)

// EngineSpec provisions the op-script-engine binary the same way the rest of the monorepo
// provisions Rust binaries (op-reth, kona-host, rollup-boost): via RUST_BINARY_PATH_OP_SCRIPT_ENGINE
// for a pre-built path, RUST_JIT_BUILD=1 for an on-demand debug build, or a pre-built binary under
// rust/target.
var EngineSpec = rustbin.Spec{
	SrcDir:  "rust",
	Package: "op-script-engine",
	Binary:  "op-script-engine",
}

// EngineBinaryPathEnv is the env var holding a pre-built op-script-engine path. It must match the
// name rustbin.Spec.EnsureExists derives from EngineSpec.Binary ("op-script-engine" ->
// "OP_SCRIPT_ENGINE"). CI's rust-workspace-binaries job builds the whole rust/ workspace at release
// and persists rust/target/release/op-script-engine; the go-tests jobs point this env at it so the
// cargo-less Go executors resolve the pre-built binary instead of rebuilding.
const EngineBinaryPathEnv = "RUST_BINARY_PATH_OP_SCRIPT_ENGINE"

// EngineBinary locates or builds the op-script-engine binary and returns its absolute path.
func EngineBinary(ctx context.Context, logger log.Logger) (string, error) {
	return EngineSpec.EnsureExists(ctx, logger)
}

// PrebuiltEngineBinary returns the path in EngineBinaryPathEnv when it is set and exists on disk,
// and whether such a binary was found. Test provisioning helpers prefer a CI-supplied pre-built
// binary (Go executors without cargo) over an in-test cargo build.
func PrebuiltEngineBinary() (string, bool) {
	p := os.Getenv(EngineBinaryPathEnv)
	if p == "" {
		return "", false
	}
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

// ArtifactsDir recovers the on-disk directory backing a forge-artifacts filesystem so it can be
// passed to the engine's --artifacts flag. op-deployer always builds these from os.DirFS (local
// file://, extracted embedded, or extracted downloaded tarballs), whose concrete type is a string
// kind wrapping the directory path.
func ArtifactsDir(artifacts foundry.StatDirFs) (string, error) {
	v := reflect.ValueOf(artifacts)
	if v.Kind() == reflect.String {
		return v.String(), nil
	}
	return "", fmt.Errorf("cannot recover on-disk artifacts directory from %T (not an os.DirFS)", artifacts)
}

// NewLogWriter adapts a logger into the io.Writer that Spawn forwards the engine's stderr to,
// emitting each line at debug level under the "op-script-engine" component.
func NewLogWriter(logger log.Logger) io.Writer {
	return &logWriter{logger: logger.New("component", "op-script-engine")}
}

type logWriter struct {
	logger log.Logger
	buf    strings.Builder
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	s := w.buf.String()
	for {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(s[:i], "\r")
		if line != "" {
			w.logger.Debug(line)
		}
		s = s[i+1:]
	}
	w.buf.Reset()
	w.buf.WriteString(s)
	return len(p), nil
}
