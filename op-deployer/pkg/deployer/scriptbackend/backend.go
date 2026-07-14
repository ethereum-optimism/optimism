// Package scriptbackend runs op-deployer's OPCM forge scripts on the out-of-process Rust
// op-script-engine (op-chain-ops/script/rustengine). It lets the forked L1 callers (bootstrap,
// upgrade, manage, sysgo) share the spawn/fork/drain plumbing behind a small Backend seam without
// each embedding it.
//
// It is a leaf package: it imports opcm and rustengine (which never import each other, so no cycle).
// Nothing in op-deployer's lower layers imports it.
package scriptbackend

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
)

// Backend wraps the out-of-process Rust engine. Construct one with NewForkedL1 (which also wires
// broadcast draining and teardown) or FromEngine.
type Backend interface {
	isScriptBackend()
}

type engineBackend struct {
	eng    *rustengine.Engine
	origin common.Address
	fa     *foundry.ArtifactsFS
}

func (*engineBackend) isScriptBackend() {}

// FromEngine wraps a spawned Rust engine as a Backend. origin is the run() tx.origin the OPCM scripts
// execute as; fa supplies the ABIs for Go-side packing/validation.
func FromEngine(eng *rustengine.Engine, origin common.Address, fa *foundry.ArtifactsFS) Backend {
	return &engineBackend{eng: eng, origin: origin, fa: fa}
}

// RunScriptSingle drives the OPCM input+output-precompile path on backend b: it snapshots inputs,
// captures the script's output set() calls and replays them into O (the unidirectional design).
func RunScriptSingle[I any, O any](b Backend, input I, scriptFile, contractName string) (O, error) {
	switch t := b.(type) {
	case *engineBackend:
		return rustengine.RunScriptSingle[I, O](t.eng, input, scriptFile, contractName, t.origin)
	default:
		var zero O
		return zero, errUnknownBackend(b)
	}
}

// RunScriptVoid drives the OPCM input-only path on backend b.
func RunScriptVoid[I any](b Backend, input I, scriptFile, contractName string) error {
	switch t := b.(type) {
	case *engineBackend:
		return rustengine.RunScriptVoid[I](t.eng, input, scriptFile, contractName, t.origin)
	default:
		return errUnknownBackend(b)
	}
}

// Call executes a plain read CALL on backend b and returns the output data. It exists for callers
// that query view functions (e.g. version reads) against the forked state.
func Call(b Backend, from, to common.Address, data []byte) ([]byte, error) {
	switch t := b.(type) {
	case *engineBackend:
		return t.eng.Call(from, to, data)
	default:
		return nil, errUnknownBackend(b)
	}
}

// DeployScriptWithoutOutput loads a typed void deploy script bound to backend b.
func DeployScriptWithoutOutput[I any](b Backend, file, contract string) (script.DeployScriptWithoutOutput[I], error) {
	switch t := b.(type) {
	case *engineBackend:
		artifact, err := t.fa.ReadArtifact(file, contract)
		if err != nil {
			return nil, fmt.Errorf("failed to load script %s from %s: %w", contract, file, err)
		}
		fs := rustengine.NewForgeScript(t.eng, artifact, file, contract, func() common.Address { return t.origin })
		return script.NewDeployScriptWithoutOutput[I](fs, "run")
	default:
		return nil, errUnknownBackend(b)
	}
}

// OPCMScripts builds the typed opcm.Scripts bundle on backend b. Callers invoke e.g.
// scripts.DeployImplementations.Run(input).
func OPCMScripts(b Backend) (*opcm.Scripts, error) {
	switch t := b.(type) {
	case *engineBackend:
		return NewEngineScripts(t.eng, t.fa, func() common.Address { return t.origin })
	default:
		return nil, errUnknownBackend(b)
	}
}

func errUnknownBackend(b Backend) error {
	return &unknownBackendError{b}
}

type unknownBackendError struct{ b Backend }

func (e *unknownBackendError) Error() string {
	return fmt.Sprintf("scriptbackend: unknown backend type %T", e.b)
}
