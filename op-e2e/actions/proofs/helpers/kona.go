package helpers

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/stretchr/testify/require"
)

var konaHostPath, konaClientPath string

func init() {
	konaHostPath = os.Getenv("KONA_HOST_PATH")
	konaClientPath = os.Getenv("KONA_CLIENT_PATH")
}

func IsKonaConfigured() bool {
	return konaHostPath != "" && konaClientPath != ""
}

func RunKonaNative(
	t helpers.Testing,
	workDir string,
	env *L2FaultProofEnv,
	l1Rpc string,
	l1BeaconRpc string,
	l2Rpc string,
	fixtureInputs FixtureInputs,
) error {
	// Write rollup config to tempdir.
	rollupConfigPath := filepath.Join(workDir, "rollup.json")
	ser, err := json.Marshal(env.Sd.RollupCfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(rollupConfigPath, ser, fs.ModePerm))

	// Run the fault proof program from the state transition from L2 block L2Blocknumber - 1 -> L2BlockNumber.
	vmCfg := vm.Config{
		L1:               l1Rpc,
		L1Beacon:         l1BeaconRpc,
		L2:               l2Rpc,
		RollupConfigPath: rollupConfigPath,
		Server:           konaHostPath,
	}
	inputs := utils.LocalGameInputs{
		L1Head:        fixtureInputs.L1Head,
		L2Head:        fixtureInputs.L2Head,
		L2OutputRoot:  fixtureInputs.L2OutputRoot,
		L2Claim:       fixtureInputs.L2Claim,
		L2BlockNumber: big.NewInt(int64(fixtureInputs.L2BlockNumber)),
	}
	hostCmd, err := vm.NewNativeKonaExecutor(konaClientPath).OracleCommand(vmCfg, workDir, inputs)
	require.NoError(t, err)

	cmd := exec.Command(hostCmd[0], hostCmd[1:]...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	status := cmd.Run()
	switch status := status.(type) {
	case *exec.ExitError:
		if status.ExitCode() == 1 {
			return claim.ErrClaimNotValid
		}
		return fmt.Errorf("kona exited with status %d", status.ExitCode())
	default:
		return status
	}
}
