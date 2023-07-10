package devnet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	genesis "github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

// Prestate builds the genesis files prior to bringing up services.
func Prestate(cliCtx *cli.Context, paths *paths) {
	log.Info("Generating prestate")

	if _, err := os.Stat(filepath.Join(paths.devnetDir, "done")); err == nil {
		log.Info("Genesis files already exist")
	} else {
		log.Info("Creating genesis files")

		content := ReadFile(filepath.Join(paths.deploymentConfigDir, "devnetL1.json"))
		var deployConfig genesis.DeployConfig
		err := json.Unmarshal(content, &deployConfig)
		if err != nil {
			log.Crit("Failed to unmarshal deploy config", "err", err)
		}
		deployConfig.L1GenesisBlockTimestamp = hexutil.Uint64(time.Now().Unix())
		tempDeployConfig := filepath.Join(paths.devnetDir, "deploy-config.json")
		content, err = json.Marshal(deployConfig)
		if err != nil {
			log.Crit("Failed to marshal deploy config", "err", err)
		}
		WriteFile(tempDeployConfig, content)

		outfileL1 := paths.genesisL1Path
		outfileL2 := paths.genesisL2Path
		outfileRollup := paths.rollupConfigPath

		RunCommand(
			[]string{"go", "run", "cmd/main.go", "genesis", "devnet", "--deploy-config",
				tempDeployConfig, "--outfile-l1", outfileL1, "--outfile-l2", outfileL2, "--outfile-rollup",
				outfileRollup},
			[]string{},
			paths.opNodeDir,
		)
		content, err = json.Marshal("{}")
		if err != nil {
			log.Crit("Failed to marshal empty json", "err", err)
		}
		WriteFile(filepath.Join(paths.devnetDir, "done"), content)
	}

	log.Info("Bringing up L1.")
	additionalEnvs := []string{fmt.Sprintf("PWD=%s", paths.opsBedrockDir)}
	RunCommand([]string{"docker-compose", "up", "-d", "l1"}, additionalEnvs, paths.opsBedrockDir)
	WaitUp(8545, WaitOpts{})
	WaitForRpcServer("127.0.0.1:8545")

	log.Info("Bringing up L2.")
	additionalEnvs = []string{fmt.Sprintf("PWD=%s", paths.opsBedrockDir)}
	RunCommand([]string{"docker-compose", "up", "-d", "l2"}, additionalEnvs, paths.opsBedrockDir)
	WaitUp(9545, WaitOpts{})
	WaitForRpcServer("127.0.0.1:9545")

	log.Info("Bringing up the services.")
	additionalEnvs = []string{
		fmt.Sprintf("PWD=%s", paths.opsBedrockDir),
		"L2OO_ADDRESS=0x6900000000000000000000000000000000000000",
	}
	RunCommand([]string{"docker-compose", "up", "-d", "op-proposer", "op-batcher"}, additionalEnvs, paths.opsBedrockDir)
}
