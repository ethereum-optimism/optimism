package rustengine

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// ForgeScript implements script.ForgeScript over the out-of-process Rust engine, so the
// generic typed wrappers (script.NewDeployScriptWithOutput etc., including their ABI/type
// validation) work unchanged against the engine.
//
// Call maps to the engine's runScript RPC, which mirrors the in-process
// forgeScriptImpl.Call flow exactly: deploy the script from the script-deployer (size
// checks lifted, cheatcodes allowed), call it with the given calldata from origin(), and
// wipe the script account again.
type ForgeScript struct {
	eng      *Engine
	abi      abi.ABI
	file     string
	contract string
	origin   func() common.Address
}

var _ script.ForgeScript = (*ForgeScript)(nil)

// NewForgeScript builds an engine-backed script.ForgeScript. The artifact provides the ABI
// (Go-side packing/validation); file/contract address the same artifact inside the engine.
// origin is resolved per call, matching the Go host's use of the current tx.origin.
func NewForgeScript(eng *Engine, artifact *foundry.Artifact, file, contract string, origin func() common.Address) *ForgeScript {
	return &ForgeScript{eng: eng, abi: artifact.ABI, file: file, contract: contract, origin: origin}
}

func (s *ForgeScript) ABI() abi.ABI {
	return s.abi
}

func (s *ForgeScript) Name() string {
	return s.contract
}

func (s *ForgeScript) Call(input []byte) ([]byte, error) {
	return s.eng.RunScript(s.file, s.contract, input, s.origin())
}
