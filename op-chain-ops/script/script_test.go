package script

import (
	"errors"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestArtifacts(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	af := foundry.OpenArtifactsDir("../../packages/contracts-bedrock/forge-artifacts")
	artifacts, err := af.ListArtifacts()
	require.NoError(t, err)
	for _, name := range artifacts {
		contracts, err := af.ListContracts(name)
		require.NoError(t, err, "failed to list %s", name)
		for _, contract := range contracts {
			artifact, err := af.ReadArtifact(name, contract)
			if err != nil {
				if errors.Is(err, foundry.ErrLinkingUnsupported) {
					logger.Info("linking not supported", "name", name, "contract", contract, "err", err)
					continue
				}
				require.NoError(t, err, "failed to read artifact %s / %s", name, contract)
			}
			logger.Info("artifact",
				"name", name,
				"contract", contract,
				"compiler", artifact.Metadata.Compiler.Version,
				"sources", len(artifact.Metadata.Sources),
				"evmVersion", artifact.Metadata.Settings.EVMVersion,
			)
		}
	}
}

func TestScript(t *testing.T) {

	logger := testlog.Logger(t, log.LevelInfo)
	af := foundry.OpenArtifactsDir("../../packages/contracts-bedrock/forge-artifacts")

	// TODO default sender nonce appears to be 3 in foundry

	// TODO emulate initial test env of
	// https://github.com/foundry-rs/foundry/blob/master/testdata/default/core/ContractEnvironment.t.sol

	scriptContext := DefaultContext
	h := NewHost(logger, af, scriptContext)
	addr, err := h.LoadContract("Experiment.s", "Experiment")
	require.NoError(t, err)

	// TODO: Go interface -> use reflection to turn it into ABI
	// Like automated bindings
	// Do some assertions against artifact ABI definition, to validate bindings

	input := bytes4("doThing()")
	returnData, _, err := h.Call(scriptContext.sender, addr, input[:], DefaultFoundryGasLimit, uint256.NewInt(0))
	require.NoError(t, err, "call failed: %x", string(returnData))
	t.Logf("call succeeded: %x", string(returnData))
}
