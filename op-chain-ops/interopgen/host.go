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
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// DeployHost runs the interopgen deployment on the out-of-process Rust op-script-engine. origin
// tracks the current tx.origin (script calls run from it).
type DeployHost struct {
	eng    *rustengine.Engine
	fa     *foundry.ArtifactsFS
	origin common.Address
}

// SetTxOrigin switches the sender/origin for subsequent script runs.
func (h *DeployHost) SetTxOrigin(addr common.Address) {
	h.origin = addr
}

// EnableCheats is a no-op: the engine always runs with cheats enabled.
func (h *DeployHost) EnableCheats() error {
	return nil
}

// SetEnvVar sets a vm.env* environment variable.
func (h *DeployHost) SetEnvVar(key, value string) error {
	return h.eng.SetEnv(key, value)
}

// StateDump dumps the committed state into forge-allocs form.
func (h *DeployHost) StateDump() (*foundry.ForgeAllocs, error) {
	return h.eng.StateDump()
}

// Close terminates the engine subprocess.
func (h *DeployHost) Close() {
	if h.eng != nil {
		h.eng.Close()
	}
}

// hostFactory builds L1/L2 deploy hosts on the Rust op-script-engine.
type hostFactory struct {
	logger       log.Logger
	fa           *foundry.ArtifactsFS
	binPath      string
	artifactsDir string
}

func newHostFactory(logger log.Logger, fa *foundry.ArtifactsFS) (*hostFactory, error) {
	binPath, err := rustengine.EngineBinary(context.Background(), logger)
	if err != nil {
		return nil, fmt.Errorf("failed to locate op-script-engine binary: %w", err)
	}
	artifactsDir, err := rustengine.ArtifactsDir(fa.FS)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve artifacts dir for op-script-engine: %w", err)
	}
	return &hostFactory{logger: logger, fa: fa, binPath: binPath, artifactsDir: artifactsDir}, nil
}

// newHost builds a DeployHost for the given script context and host options on the engine.
func (f *hostFactory) newHost(logger log.Logger, ctx script.Context, create2Deployer, noMaxCodeSize bool) (*DeployHost, error) {
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

// loadScriptWithOutput loads a typed deploy script (ABI-validated against I/O) bound to the engine.
func loadScriptWithOutput[I any, O any](h *DeployHost, file, name string) (script.DeployScriptWithOutput[I, O], error) {
	fs, err := engineForgeScript(h, file, name)
	if err != nil {
		return nil, err
	}
	return script.NewDeployScriptWithOutput[I, O](fs, "run")
}

// loadScriptWithoutOutput is loadScriptWithOutput for void-returning scripts.
func loadScriptWithoutOutput[I any](h *DeployHost, file, name string) (script.DeployScriptWithoutOutput[I], error) {
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

// insertPreinstalls runs SetPreinstalls.setPreinstalls() on the host (deploy script, call, wipe script).
func insertPreinstalls(h *DeployHost) error {
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

// migrate runs the InteropMigration script (the OPCM RunScriptSingle input/output-precompile path).
func migrate(h *DeployHost, input manage.InteropMigrationInput) (manage.InteropMigrationOutput, error) {
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
