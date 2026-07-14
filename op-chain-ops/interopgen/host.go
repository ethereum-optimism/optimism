package interopgen

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/manage"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/scriptbackend"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// DeployHost abstracts the script host the interopgen deployment runs on: either the
// in-process Go script.Host or the out-of-process Rust op-script-engine. Exactly one of
// goHost/eng is set. origin tracks the current tx.origin (script calls run from it) for
// the engine, mirroring script.Host.SetTxOrigin.
type DeployHost struct {
	goHost *script.Host
	eng    *rustengine.Engine
	fa     *foundry.ArtifactsFS
	origin common.Address
}

// SetTxOrigin switches the sender/origin for subsequent script runs.
func (h *DeployHost) SetTxOrigin(addr common.Address) {
	h.origin = addr
	if h.goHost != nil {
		h.goHost.SetTxOrigin(addr)
	}
}

// EnableCheats enables the vm cheatcode precompile (a no-op for the engine, which always
// runs with cheats enabled).
func (h *DeployHost) EnableCheats() error {
	if h.goHost != nil {
		return h.goHost.EnableCheats()
	}
	return nil
}

// SetEnvVar sets a vm.env* environment variable.
func (h *DeployHost) SetEnvVar(key, value string) error {
	if h.goHost != nil {
		h.goHost.SetEnvVar(key, value)
		return nil
	}
	return h.eng.SetEnv(key, value)
}

// StateDump dumps the committed state into forge-allocs form.
func (h *DeployHost) StateDump() (*foundry.ForgeAllocs, error) {
	if h.goHost != nil {
		return h.goHost.StateDump()
	}
	return h.eng.StateDump()
}

// Close terminates the engine subprocess (no-op for the Go host).
func (h *DeployHost) Close() {
	if h.eng != nil {
		h.eng.Close()
	}
}

// hostFactory builds L1/L2 deploy hosts on the selected script engine.
type hostFactory struct {
	logger log.Logger
	fa     *foundry.ArtifactsFS
	srcFS  *foundry.SourceMapFS
	engine env.ScriptEngineKind
	// binPath/artifactsDir are resolved once when the rust engine is selected.
	binPath      string
	artifactsDir string
}

func newHostFactory(logger log.Logger, fa *foundry.ArtifactsFS, srcFS *foundry.SourceMapFS, engine env.ScriptEngineKind) (*hostFactory, error) {
	resolved, err := engine.Resolve()
	if err != nil {
		return nil, err
	}
	f := &hostFactory{logger: logger, fa: fa, srcFS: srcFS, engine: resolved}
	if f.engine == env.ScriptEngineRust {
		binPath, err := rustengine.EngineBinary(context.Background(), logger)
		if err != nil {
			return nil, fmt.Errorf("failed to locate op-script-engine binary: %w", err)
		}
		artifactsDir, err := rustengine.ArtifactsDir(fa.FS)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve artifacts dir for op-script-engine: %w", err)
		}
		f.binPath = binPath
		f.artifactsDir = artifactsDir
	}
	return f, nil
}

// newHost builds a DeployHost for the given script context and host options on the selected
// engine. The Go branch mirrors script.NewHost exactly; the engine branch maps the context
// onto the engine's spawn options.
func (f *hostFactory) newHost(logger log.Logger, ctx script.Context, create2Deployer, noMaxCodeSize bool) (*DeployHost, error) {
	if f.engine == env.ScriptEngineRust {
		eng, err := rustengine.Spawn(f.binPath, rustengine.SpawnOpts{
			ArtifactsDir:    f.artifactsDir,
			ChainID:         bigs.Uint64Strict(ctx.ChainID),
			Create2Deployer: create2Deployer,
			NoMaxCodeSize:   noMaxCodeSize,
			BlockNum:        ctx.BlockNum,
			Timestamp:       ctx.Timestamp,
			PrevRandao:      ctx.PrevRandao,
		}, rustengine.NewLogWriter(logger))
		if err != nil {
			return nil, fmt.Errorf("failed to spawn op-script-engine: %w", err)
		}
		return &DeployHost{eng: eng, fa: f.fa, origin: ctx.Origin}, nil
	}
	var opts []script.HostOption
	if create2Deployer {
		opts = append(opts, script.WithCreate2Deployer())
	}
	if noMaxCodeSize {
		opts = append(opts, script.WithNoMaxCodeSize())
	}
	return &DeployHost{goHost: script.NewHost(logger, f.fa, f.srcFS, ctx, opts...), fa: f.fa, origin: ctx.Origin}, nil
}

// loadScriptWithOutput loads a typed deploy script (ABI-validated against I/O) bound to the
// host's engine. The engine branch reuses the same generic wrapper over an engine-backed
// script.ForgeScript, so packing/validation are identical to the Go host path.
func loadScriptWithOutput[I any, O any](h *DeployHost, file, name string) (script.DeployScriptWithOutput[I, O], error) {
	if h.goHost != nil {
		return script.NewDeployScriptWithOutputFromFile[I, O](h.goHost, file, name)
	}
	fs, err := engineForgeScript(h, file, name)
	if err != nil {
		return nil, err
	}
	return script.NewDeployScriptWithOutput[I, O](fs, "run")
}

// loadScriptWithoutOutput is loadScriptWithOutput for void-returning scripts.
func loadScriptWithoutOutput[I any](h *DeployHost, file, name string) (script.DeployScriptWithoutOutput[I], error) {
	if h.goHost != nil {
		return script.NewDeployScriptWithoutOutputFromFile[I](h.goHost, file, name)
	}
	fs, err := engineForgeScript(h, file, name)
	if err != nil {
		return nil, err
	}
	return script.NewDeployScriptWithoutOutput[I](fs, "run")
}

func engineForgeScript(h *DeployHost, file, name string) (script.ForgeScript, error) {
	artifact, err := h.fa.ReadArtifact(file, name)
	if err != nil {
		return nil, fmt.Errorf("failed to load script %s from %s: %w", name, file, err)
	}
	return rustengine.NewForgeScript(h.eng, artifact, file, name, func() common.Address { return h.origin }), nil
}

// insertPreinstalls runs SetPreinstalls.setPreinstalls() on the host, matching
// opcm.InsertPreinstalls (deploy script, call, wipe script).
func insertPreinstalls(h *DeployHost) error {
	if h.goHost != nil {
		return opcm.InsertPreinstalls(h.goHost)
	}
	artifact, err := h.fa.ReadArtifact("SetPreinstalls.s.sol", "SetPreinstalls")
	if err != nil {
		return fmt.Errorf("failed to load SetPreinstalls script: %w", err)
	}
	calldata, err := artifact.ABI.Pack("setPreinstalls")
	if err != nil {
		return fmt.Errorf("failed to pack setPreinstalls() call: %w", err)
	}
	if _, err := h.eng.RunScript("SetPreinstalls.s.sol", "SetPreinstalls", calldata, h.origin); err != nil {
		return fmt.Errorf("failed to set preinstalls: %w", err)
	}
	return nil
}

// migrate runs the InteropMigration script (the OPCM RunScriptSingle input/output-precompile
// path) on the host.
func migrate(h *DeployHost, input manage.InteropMigrationInput) (manage.InteropMigrationOutput, error) {
	if h.goHost != nil {
		return manage.Migrate(scriptbackend.FromHost(h.goHost), input)
	}
	if input.MigrateInputV2 == nil {
		return manage.InteropMigrationOutput{}, fmt.Errorf("MigrateInputV2 is required")
	}
	encodedMigrateInput, err := input.EncodedMigrateInputV2()
	if err != nil {
		return manage.InteropMigrationOutput{}, err
	}
	scriptInput := manage.ScriptInput{
		Prank:        input.Prank,
		Opcm:         input.Opcm,
		MigrateInput: encodedMigrateInput,
	}
	return rustengine.RunScriptSingle[manage.ScriptInput, manage.InteropMigrationOutput](
		h.eng, scriptInput, "InteropMigration.s.sol", "InteropMigration", h.origin)
}
