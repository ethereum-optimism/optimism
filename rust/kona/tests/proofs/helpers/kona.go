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
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
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

// writeDepset writes the dependency set ds in the JSON shape that both op-node's
// --interop.dependency-set and kona's --depset-cfg read.
//
// kona-host super always needs a dependency set: StatefulAttributesBuilder consults it to decide
// whether an interop activation block carries the multi-chain wrapper deposits, and kona-host
// refuses to start when the first chain of the agreed pre-state is not in the
// superchain-registry and no --depset-cfg is given. kona must get the same set as the
// sequencer, so that op-node and kona agree on the activation block's contents.
func writeDepset(t helpers.Testing, path string, ds depset.DependencySet) {
	require.NotNil(t, ds, "the program needs the dependency set the sequencer derives with")
	ser, err := json.Marshal(ds)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, ser, fs.ModePerm))
}

// RunKonaSuperNative runs the interop client program (kona-host super --native) over the
// super-root sub-transition described by fixtureInputs. It returns ErrClaimNotValid when the
// program rejects the claimed post-state.
func RunKonaSuperNative(
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

	depsetPath := filepath.Join(workDir, "depset.json")
	writeDepset(t, depsetPath, fixtureInputs.DependencySet)

	vmCfg := vm.Config{
		L1:                l1Rpc,
		L1Beacon:          l1BeaconRpc,
		L2s:               l2Rpcs,
		RollupConfigPaths: rollupCfgPaths,
		L1GenesisPath:     l1chainConfigPath,
		DepsetConfigPath:  depsetPath,
		Server:            konaHostPath,
	}
	// The interop program is timestamp-addressed, so L2SequenceNumber carries the claimed
	// timestamp (it becomes --claimed-l2-timestamp) rather than a block number.
	inputs := utils.LocalGameInputs{
		L1Head:           fixtureInputs.L1Head,
		AgreedPreState:   fixtureInputs.AgreedPrestate,
		L2Claim:          fixtureInputs.L2Claim,
		L2SequenceNumber: new(big.Int).SetUint64(fixtureInputs.ClaimTimestamp),
	}

	logger := logcli.NewLogger(os.Stdout, logcli.DefaultCLIConfig())

	if !rustbin.RunKonaSuperNative(t, logger, &vmCfg, workDir, &inputs) {
		return ErrClaimNotValid
	}
	return nil
}
