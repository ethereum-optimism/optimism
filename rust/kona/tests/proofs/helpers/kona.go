package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// ErrClaimNotValid is returned when the fault proof program rejects the claimed output root.
var ErrClaimNotValid = errors.New("invalid claim")

var konaHostPath string

func init() {
	konaHostPath = os.Getenv("KONA_HOST_PATH")
}

func writeConfigs[T any](t helpers.Testing, workDir string, name string, cfg []*T, cfgPaths []string) {
	for i, cfg := range cfg {
		cfgPath := filepath.Join(workDir, fmt.Sprintf("%s_%d.json", name, i))
		writeConfig(t, workDir, name, cfg, cfgPath)
		cfgPaths[i] = cfgPath
	}
}

func writeConfig[T any](t helpers.Testing, workDir string, name string, cfg *T, cfgPath string) {
	ser, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cfgPath, ser, fs.ModePerm))
}

func RunKonaNative(
	t helpers.Testing,
	workDir string,
	rollupCfgs []*rollup.Config,
	l1chainConfig *params.ChainConfig,
	l1Rpc string,
	l1BeaconRpc string,
	l2Rpcs []string,
	fixtureInputs FixtureInputs,
) error {
	// Write rollup config to tempdir.
	rollupCfgPaths := make([]string, len(rollupCfgs))
	writeConfigs(t, workDir, "rollup", rollupCfgs, rollupCfgPaths)

	// Write l1 chain config to tempdir.
	l1chainConfigPath := filepath.Join(workDir, "l1chain.json")
	writeConfig(t, workDir, "l1chain", l1chainConfig, l1chainConfigPath)

	// Run the fault proof program from the state transition from L2 block L2Blocknumber - 1 -> L2BlockNumber.
	vmCfg := vm.Config{
		L1:                l1Rpc,
		L1Beacon:          l1BeaconRpc,
		L2s:               l2Rpcs,
		RollupConfigPaths: rollupCfgPaths,
		L1GenesisPath:     l1chainConfigPath,
		Server:            konaHostPath,
	}
	inputs := utils.LocalGameInputs{
		L1Head:           fixtureInputs.L1Head,
		L2Head:           fixtureInputs.L2Head,
		L2Claim:          fixtureInputs.L2Claim,
		L2SequenceNumber: big.NewInt(int64(fixtureInputs.L2BlockNumber)),
		L2OutputRoot:     fixtureInputs.L2OutputRoot,
	}

	logger := log.NewLogger(os.Stdout, log.DefaultCLIConfig())

	if !rustbin.RunKonaNative(t, logger, &vmCfg, workDir, &inputs) {
		return ErrClaimNotValid
	}
	return nil
}
