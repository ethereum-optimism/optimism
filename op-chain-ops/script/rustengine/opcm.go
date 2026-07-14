package rustengine

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// RunScriptSingle drives the OPCM RunScriptSingle path (op-deployer/pkg/deployer/opcm.RunScriptSingle)
// through the out-of-process Rust engine using the unidirectional design (design §4):
//
//  1. Snapshot input I's getters on the Go side (script.NewPrecompile reflection) and install them
//     as an input precompile in the engine.
//  2. Install an output setter-capture precompile for O.
//  3. Run the script with run(inputAddr, outputAddr) — the engine answers input getters from the
//     snapshot and captures output set() calls, with no callback into Go.
//  4. Replay the captured set() calldata through the real Go WithFieldSetter precompile to populate
//     the Go output struct O.
//
// deployer is the run() tx.origin (the Go host uses DefaultContext.Origin).
func RunScriptSingle[I any, O any](
	e *Engine,
	input I,
	scriptFile string,
	contractName string,
	deployer common.Address,
) (O, error) {
	var output O

	inP, err := script.NewPrecompile[*I](&input)
	if err != nil {
		return output, fmt.Errorf("failed to build input precompile: %w", err)
	}
	inputAddr, err := e.InstallInputPrecompile(inP.Snapshot())
	if err != nil {
		return output, fmt.Errorf("failed to install input precompile: %w", err)
	}
	defer func() { _ = e.RemovePrecompile(inputAddr) }()

	outP, err := script.NewPrecompile[*O](&output, script.WithFieldSetter[*O])
	if err != nil {
		return output, fmt.Errorf("failed to build output precompile: %w", err)
	}
	outputAddr, err := e.InstallOutputPrecompile(outP.SettableSelectors())
	if err != nil {
		return output, fmt.Errorf("failed to install output precompile: %w", err)
	}
	defer func() { _ = e.RemovePrecompile(outputAddr) }()

	calldata := packRun("run(address,address)", inputAddr, outputAddr)
	if _, err := e.RunScript(scriptFile, contractName, calldata, deployer); err != nil {
		return output, fmt.Errorf("failed to run %s: %w", scriptFile, err)
	}

	sets, err := e.TakeCapturedSets(outputAddr)
	if err != nil {
		return output, fmt.Errorf("failed to take captured sets: %w", err)
	}
	if err := replaySets(outP, sets); err != nil {
		return output, err
	}
	return output, nil
}

// RunScriptVoid drives the OPCM RunScriptVoid path (input-only) through the Rust engine.
func RunScriptVoid[I any](
	e *Engine,
	input I,
	scriptFile string,
	contractName string,
	deployer common.Address,
) error {
	inP, err := script.NewPrecompile[*I](&input)
	if err != nil {
		return fmt.Errorf("failed to build input precompile: %w", err)
	}
	inputAddr, err := e.InstallInputPrecompile(inP.Snapshot())
	if err != nil {
		return fmt.Errorf("failed to install input precompile: %w", err)
	}
	defer func() { _ = e.RemovePrecompile(inputAddr) }()

	calldata := packRun("run(address)", inputAddr)
	if _, err := e.RunScript(scriptFile, contractName, calldata, deployer); err != nil {
		return fmt.Errorf("failed to run %s: %w", scriptFile, err)
	}
	return nil
}

// packRun builds run(address[,address]) calldata: 4-byte selector + left-padded address words.
func packRun(sig string, addrs ...common.Address) []byte {
	out := make([]byte, 0, 4+32*len(addrs))
	out = append(out, crypto.Keccak256([]byte(sig))[:4]...)
	for _, a := range addrs {
		out = append(out, common.LeftPadBytes(a.Bytes(), 32)...)
	}
	return out
}

// replaySets writes each captured set() call into the Go output struct via the WithFieldSetter
// precompile, exactly as the in-process opcm.RunScriptSingle would have during the run.
func replaySets[O any](outP *script.Precompile[*O], sets [][]byte) error {
	for i, s := range sets {
		if _, err := outP.Run(s); err != nil {
			return fmt.Errorf("failed to replay output set %d (0x%x): %w", i, s, err)
		}
	}
	return nil
}
