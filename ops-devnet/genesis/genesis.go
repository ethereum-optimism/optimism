package genesis

import (
	"os"
	"os/exec"
	"io/ioutil"
	"time"

	chainops "github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/ops-devnet/utils"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

// Generate creates the L1 genesis.
func Generate(monorepo string, endpoint string) error {
	log.Info("Generating L1 genesis.")

	allocsPath := utils.AllocsJsonPath(monorepo)
	genesisPath := utils.GenesisJsonPath(monorepo)
	addressesPath := utils.AddressesJsonPath(monorepo)
	devnetConfigPath := utils.DevnetConfigPath(monorepo)
	devnetConfigBackup := utils.DevnetConfigBackup(monorepo)

	data, err := ioutil.ReadFile(devnetConfigPath)
    if err != nil {
		return err
	}
    err = ioutil.WriteFile(devnetConfigBackup, data, 0644)
	if err != nil {
		return err
	}

	deployConfig, err := chainops.NewDeployConfig(devnetConfigPath)
	if err != nil {
		return err
	}
	deployConfig.L1GenesisBlockTimestamp = hexutil.Uint64(time.Now().Unix())

	utils.WriteJson(devnetConfigPath, deployConfig)

	cmd := exec.Command(
		"go", "run", "cmd/main.go", "genesis", "l1",
            "--deploy-config", devnetConfigPath,
            "--l1-allocs", allocsPath,
            "--l1-deployments", addressesPath,
            "--outfile.l1", genesisPath,
	)
	cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
	cmd.Dir = utils.OpNodeDirectory(monorepo)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// GenerateL2 creates the L2 genesis.
func GenerateL2(monorepo string, endpoint string) error {
	log.Info("Generating L2 genesis.")

	opNodeDirectory := utils.OpNodeDirectory(monorepo)
	genesisL2Path := utils.GenesisL2JsonPath(monorepo)
	devnetConfigPath := utils.DevnetConfigPath(monorepo)
	deploymentDir := utils.DeploymentDirectory(monorepo)
	rollupPath := utils.RollupPath(monorepo)
	cmd := exec.Command(
		"go", "run", "cmd/main.go",
		"genesis", "l2",
		"--l1-rpc", utils.PrefixIfMissing(endpoint, "http://"),
		"--deploy-config", devnetConfigPath,
		"--deployment-dir", deploymentDir,
		"--outfile.l2", genesisL2Path,
		"--outfile.rollup", rollupPath,
	)
	cmd.Dir = opNodeDirectory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// RestoreBackup restores the devnet config from the backup.
func RestoreBackup(monorepo string) error {
	log.Info("Restoring devnet config from backup.")

	devnetConfigPath := utils.DevnetConfigPath(monorepo)
	devnetConfigBackup := utils.DevnetConfigBackup(monorepo)

	data, err := ioutil.ReadFile(devnetConfigBackup)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(devnetConfigPath, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
