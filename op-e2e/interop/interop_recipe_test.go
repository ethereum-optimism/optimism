package interop

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestInteropDevRecipe(t *testing.T) {
	rec := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2ChainIDs:       []uint64{900200, 900201},
		GenesisTimestamp: uint64(1234567),
	}
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	worldCfg, err := rec.Build(hd)
	require.NoError(t, err)

	logger := testlog.Logger(t, log.LevelDebug)
	require.NoError(t, worldCfg.Check(logger))

	fa := foundry.OpenArtifactsDir("../../packages/contracts-bedrock/forge-artifacts")
	srcFS := foundry.NewSourceMapFS(os.DirFS("../../packages/contracts-bedrock"))

	worldDeployment, worldOutput, err := interopgen.Deploy(logger, fa, srcFS, worldCfg)
	require.NoError(t, err)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("  ", "  ")
	require.NoError(t, enc.Encode(worldDeployment))
	logger.Info("L1 output", "accounts", len(worldOutput.L1.Genesis.Alloc))
	for id, l2Output := range worldOutput.L2s {
		logger.Info("L2 output", "chain", &id, "accounts", len(l2Output.Genesis.Alloc))
	}
}
