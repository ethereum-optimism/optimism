// Package scriptbackend abstracts over the two implementations that run op-deployer's OPCM forge
// scripts: the in-process Go script.Host (op-chain-ops/script) and the out-of-process Rust
// op-script-engine (op-chain-ops/script/rustengine). It lets the forked L1 callers (bootstrap,
// upgrade, manage, sysgo) pick the backend from the --script-engine flag (rust by default) without
// each embedding the spawn/fork/drain plumbing.
//
// It is a leaf package: it imports opcm and rustengine (which never import each other, so no cycle),
// plus env for the Go forked-host constructor. Nothing in op-deployer's lower layers imports it.
package scriptbackend

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
)

// Backend is either the in-process Go script.Host or the out-of-process Rust engine. Construct one
// with NewForkedL1 (which also wires broadcast draining and teardown), or wrap an existing host with
// FromHost for callers that already hold a *script.Host.
type Backend interface {
	isScriptBackend()
}

type hostBackend struct {
	host *script.Host
}

func (*hostBackend) isScriptBackend() {}

// Host returns the underlying Go script.Host, or nil if this backend is engine-backed. It exists for
// the callers that still reach for host-only surface not yet abstracted here.
func Host(b Backend) *script.Host {
	if hb, ok := b.(*hostBackend); ok {
		return hb.host
	}
	return nil
}

// FromHost wraps an existing Go script.Host as a Backend (the --script-engine=go path, or callers
// that build the host themselves).
func FromHost(host *script.Host) Backend {
	return &hostBackend{host: host}
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

// RunScriptSingle drives the OPCM input+output-precompile path on whichever backend b wraps, mirroring
// opcm.RunScriptSingle. The output struct O is byte-identical across backends (the engine path
// snapshots inputs, captures output set() calls and replays them into O — the unidirectional design).
func RunScriptSingle[I any, O any](b Backend, input I, scriptFile, contractName string) (O, error) {
	switch t := b.(type) {
	case *engineBackend:
		return rustengine.RunScriptSingle[I, O](t.eng, input, scriptFile, contractName, t.origin)
	case *hostBackend:
		return opcm.RunScriptSingle[I, O](t.host, input, scriptFile, contractName)
	default:
		var zero O
		return zero, errUnknownBackend(b)
	}
}

// RunScriptVoid drives the OPCM input-only path on whichever backend b wraps, mirroring
// opcm.RunScriptVoid.
func RunScriptVoid[I any](b Backend, input I, scriptFile, contractName string) error {
	switch t := b.(type) {
	case *engineBackend:
		return rustengine.RunScriptVoid[I](t.eng, input, scriptFile, contractName, t.origin)
	case *hostBackend:
		return opcm.RunScriptVoid[I](t.host, input, scriptFile, contractName)
	default:
		return errUnknownBackend(b)
	}
}

// Call executes a plain read CALL on whichever backend b wraps and returns the output data. It
// mirrors script.Host.Call (host backends run with a fixed 1M gas budget and zero value; the engine
// runs its script-call entry) and exists for callers that query view functions (e.g. version reads)
// against the forked state.
func Call(b Backend, from, to common.Address, data []byte) ([]byte, error) {
	switch t := b.(type) {
	case *engineBackend:
		return t.eng.Call(from, to, data)
	case *hostBackend:
		ret, _, err := t.host.Call(from, to, data, 1_000_000, uint256.NewInt(0))
		return ret, err
	default:
		return nil, errUnknownBackend(b)
	}
}

// DeployScriptWithoutOutput loads a typed void deploy script bound to whichever backend b wraps,
// mirroring script.NewDeployScriptWithoutOutputFromFile.
func DeployScriptWithoutOutput[I any](b Backend, file, contract string) (script.DeployScriptWithoutOutput[I], error) {
	switch t := b.(type) {
	case *engineBackend:
		artifact, err := t.fa.ReadArtifact(file, contract)
		if err != nil {
			return nil, fmt.Errorf("failed to load script %s from %s: %w", contract, file, err)
		}
		fs := rustengine.NewForgeScript(t.eng, artifact, file, contract, func() common.Address { return t.origin })
		return script.NewDeployScriptWithoutOutput[I](fs, "run")
	case *hostBackend:
		return script.NewDeployScriptWithoutOutputFromFile[I](t.host, file, contract)
	default:
		return nil, errUnknownBackend(b)
	}
}

// OPCMScripts builds the typed opcm.Scripts bundle on whichever backend b wraps, mirroring
// opcm.NewScripts. Both backends return the same *opcm.Scripts type, so callers invoke e.g.
// scripts.DeployImplementations.Run(input) identically.
func OPCMScripts(b Backend) (*opcm.Scripts, error) {
	switch t := b.(type) {
	case *engineBackend:
		return newEngineScripts(t.eng, t.fa, func() common.Address { return t.origin })
	case *hostBackend:
		return opcm.NewScripts(t.host)
	default:
		return nil, errUnknownBackend(b)
	}
}

func errUnknownBackend(b Backend) error {
	return &unknownBackendError{b}
}

type unknownBackendError struct{ b Backend }

func (e *unknownBackendError) Error() string {
	return "scriptbackend: unknown backend type"
}
