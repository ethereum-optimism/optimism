package allocs

import (
	"os"
	"os/exec"
	"io/ioutil"
	"bytes"
	"time"
	"net/http"
	"encoding/json"

	"github.com/ethereum-optimism/optimism/ops-devnet/utils"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
)

// GenerateAllocs creates the L1 state dump.
func GenerateAllocs(monorepo string, endpoint string) (*state.Dump, error) {
	log.Info("Generating allocs.")

	cmd := exec.Command(
		"geth", "--dev", "--http", "--http.api", "eth,debug",
		"--verbosity", "4", "--gcmode", "archive", "--dev.gaslimit", "30000000",
	)
	geth := utils.NewProcessGroup(make(chan os.Signal, 1), cmd)
	if err := geth.Run(); err != nil {
		return nil, err
	}
	defer geth.Terminate()
	if err := DeployContracts(endpoint, monorepo); err != nil {
		return nil, err
	}
	return DebugDumpBlock(endpoint)
}

// DeployContracts deploys the contracts to the specified RPC endpoint.
func DeployContracts(endpoint string, monorepo string) error {
	log.Info("Deploying contracts.")

	utils.WaitUp(endpoint, 10, 1*time.Second)
	utils.WaitForRPC(endpoint, 10, 1*time.Second)
	accounts, err := EthAccounts(endpoint)
	if err != nil {
		return err
	}
	sender := accounts[0]

	log.Debug("Executing Deploy Script.")
	cmd := exec.Command(
		"forge", "script", "scripts/Deploy.s.sol:Deploy", "--sender", sender,
		"--rpc-url", utils.PrefixIfMissing(endpoint, "http://"), "--broadcast", "--unlocked",
	)
	cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
	cmd.Dir = utils.ContractsDirectory(monorepo)
	if err := cmd.Run(); err != nil {
		return err
	}

	data, err := ioutil.ReadFile(utils.L1DeploymentsPath(monorepo))
    if err != nil {
		return err
	}
    err = ioutil.WriteFile(utils.AddressesJsonPath(monorepo), data, 0644)
	if err != nil {
		return err
	}

	log.Debug("Executing Contract Sync.")
	cmd = exec.Command(
		"forge", "script", "scripts/Deploy.s.sol:Deploy", "--sig", "sync()",
		"--rpc-url", utils.PrefixIfMissing(endpoint, "http://"),
	)
	cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
	cmd.Dir = utils.ContractsDirectory(monorepo)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// DebugDumpBlock queries the service at [url] with the
// debug_dumpBlock RPC method and returns the result.
func DebugDumpBlock(endpoint string) (*state.Dump, error) {
	log.Info("Dumping state", "endpoint", endpoint)

	client := &http.Client{Timeout: 10 * time.Second}

	body := []byte(`{"id":"3","jsonrpc":"2.0","method":"debug_dumpBlock","params":["latest"]}`)
	req, _ := http.NewRequest("GET", utils.PrefixIfMissing(endpoint, "http://"), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	var target struct{ Result *state.Dump `json:result` }
	if err := json.NewDecoder(res.Body).Decode(&target); err != nil {
		return nil, err
	}

	return target.Result, nil
}

// EthAccounts fetches all accounts from the specified
// RPC endpoint.
func EthAccounts(endpoint string) ([]string, error) {
	log.Info("Fetching eth accounts", "endpoint", endpoint)

	client := &http.Client{Timeout: 10 * time.Second}

	reqBody := []byte(`{"id":"2","jsonrpc":"2.0","method":"eth_accounts","params":[]}`)
	req, err := http.NewRequest("POST", utils.PrefixIfMissing(endpoint, "http://"), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	var target struct{ Result []string `json:result` }
	if err := json.NewDecoder(res.Body).Decode(&target); err != nil {
		return nil, err
	}

	return target.Result, nil
}
