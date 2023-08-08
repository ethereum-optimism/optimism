package devnet

import (
	"os"
	"os/exec"
	"io/ioutil"
	"time"
	"encoding/json"

	"github.com/ethereum-optimism/optimism/ops-devnet/utils"

	"github.com/ethereum/go-ethereum/log"
)

// StartL1 starts L1 services.
func StartL1(monorepo string, endpoint string) error {
	log.Info("Starting L1.")

	cmd := exec.Command("docker-compose", "up", "-d", "l1")
 	cmd.Dir = utils.OpsDirectory(monorepo)
 	cmd.Env = append(os.Environ(), "PWD="+utils.OpsDirectory(monorepo))
 	cmd.Stdout = os.Stdout
 	cmd.Stderr = os.Stderr
 	if err := cmd.Run(); err != nil {
 		return err
 	}

 	utils.WaitUp(endpoint, 10, 1*time.Second)
	utils.WaitForRPC(endpoint, 10, 1*time.Second)
	return nil
}

// StartL2 starts L2 services.
func StartL2(monorepo string, endpoint string) error {
 	log.Info("Starting L2.")

	cmd := exec.Command("docker-compose", "up", "-d", "l2")
 	cmd.Dir = utils.OpsDirectory(monorepo)
 	cmd.Env = append(os.Environ(), "PWD="+utils.OpsDirectory(monorepo))
 	cmd.Stdout = os.Stdout
 	cmd.Stderr = os.Stderr
 	if err := cmd.Run(); err != nil {
 		return err
 	}

 	utils.WaitUp(endpoint, 10, 1*time.Second)
	utils.WaitForRPC(endpoint, 10, 1*time.Second)
	return nil
}

// ReadL2OutputOracleAddress reads the L2 output oracle address from the addresses.json file.
func ReadL2OutputOracleAddress(monorepo string) (string, error) {
	data, err := ioutil.ReadFile(utils.AddressesJsonPath(monorepo))
	if err != nil {
		return "", err
	}
	var addresses map[string]string
	if err := json.Unmarshal(data, &addresses); err != nil {
		return "", err
	}
	return addresses["L2OutputOracleProxy"], nil
}

// ReadeBatchInboxAddress reads the batch inbox address from the rollup.json file.
func ReadeBatchInboxAddress(monorepo string) (string, error) {
	data, err := ioutil.ReadFile(utils.RollupPath(monorepo))
	if err != nil {
		return "", err
	}
	var rollup map[string]interface{}
	if err := json.Unmarshal(data, &rollup); err != nil {
		return "", err
	}
	return rollup["batch_inbox_address"].(string), nil
}

// StartOpServices starts the Optimism services.
func StartOpServices(monorepo string) error {
 	log.Info("Starting Optimism services.")

 	l2OutputOracle, err := ReadL2OutputOracleAddress(monorepo)
 	if err != nil {
 		return err
 	}
 	log.Debug("Read L2 output oracle address.", "address", l2OutputOracle)

 	batchInboxAddress, err := ReadeBatchInboxAddress(monorepo)
 	if err != nil {
 		return err
	}
 	log.Debug("Read batch inbox address.", "address", batchInboxAddress)

	cmd := exec.Command("docker-compose", "up", "-d", "op-node", "op-proposer", "op-batcher")
 	cmd.Dir = utils.OpsDirectory(monorepo)
 	cmd.Env = append(os.Environ(), "PWD="+utils.OpsDirectory(monorepo))
 	cmd.Env = append(cmd.Env, "L2OO_ADDRESS="+l2OutputOracle)
 	cmd.Env = append(cmd.Env, "SEQUENCER_BATCH_INBOX_ADDRESS="+batchInboxAddress)
 	cmd.Stdout = os.Stdout
 	cmd.Stderr = os.Stderr
 	if err := cmd.Run(); err != nil {
 		return err
 	}

	return nil
}
