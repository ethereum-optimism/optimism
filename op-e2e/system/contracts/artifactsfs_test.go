package contracts

import (
	"errors"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestArtifacts(t *testing.T) {
	op_e2e.InitParallel(t)
	logger := testlog.Logger(t, log.LevelWarn) // lower this log level to get verbose test dump of all artifacts
	af := foundry.OpenArtifactsDir("../../../packages/contracts-bedrock/forge-artifacts")
	artifacts, err := af.ListArtifacts()
	require.NoError(t, err)
	require.NotEmpty(t, artifacts)
	for _, name := range artifacts {
		contracts, err := af.ListContracts(name)
		require.NoError(t, err, "failed to list %s", name)
		require.NotEmpty(t, contracts)
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
