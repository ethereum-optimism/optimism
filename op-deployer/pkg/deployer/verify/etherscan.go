package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/time/rate"
)

type EtherscanGenericResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

type EtherscanContractCreationResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  []struct {
		ContractCreator string `json:"contractCreator"`
		TxHash          string `json:"txHash"`
	} `json:"result"`
}

type EtherscanClient struct {
	apiKey      string
	url         string
	rateLimiter *rate.Limiter
}

func getAPIEndpoint(l1ChainID uint64) (string, error) {
	switch l1ChainID {
	case 1:
		return "https://api.etherscan.io/api", nil // mainnet
	case 11155111:
		return "https://api-sepolia.etherscan.io/api", nil // sepolia
	default:
		return "", fmt.Errorf("unsupported L1 chain ID: %d", l1ChainID)
	}
}

func NewEtherscanClient(apiKey string, url string, rateLimiter *rate.Limiter) *EtherscanClient {
	return &EtherscanClient{
		apiKey:      apiKey,
		url:         url,
		rateLimiter: rateLimiter,
	}
}

// sendRateLimitedRequest is a helper function which waits for a rate limit token
// before sending a request
func (c *EtherscanClient) sendRateLimitedRequest(req *http.Request) (*http.Response, error) {
	if err := c.rateLimiter.Wait(context.Background()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	return http.DefaultClient.Do(req)
}

// getContractCreation returns the txHash of the contract creation tx
// (useful for extracting constructor args)
func (c *EtherscanClient) getContractCreation(address common.Address) (common.Hash, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?module=contract&action=getcontractcreation&contractaddresses=%s&apikey=%s",
		c.url, address.Hex(), c.apiKey), nil)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create contract creation request: %w", err)
	}

	resp, err := c.sendRateLimitedRequest(req)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send contract creation request: %w", err)
	}
	defer resp.Body.Close()

	var creationResp EtherscanContractCreationResp
	if err := json.NewDecoder(resp.Body).Decode(&creationResp); err != nil {
		return common.Hash{}, fmt.Errorf("failed to decode contract creation response: %w", err)
	}
	if creationResp.Status != "1" {
		return common.Hash{}, fmt.Errorf("contract creation query failed: %s", creationResp.Message)
	}

	txHash := common.HexToHash(creationResp.Result[0].TxHash)
	return txHash, nil
}

func (c *EtherscanClient) verifySourceCode(address common.Address, artifact *contractArtifact, constructorArgs string) (string, error) {
	optimized := "0"
	if artifact.Optimizer.Enabled {
		optimized = "1"
	}

	standardInput := newStandardInput(artifact)
	standardInputJSON, err := json.Marshal(standardInput)
	if err != nil {
		return "", fmt.Errorf("failed to generate standard input: %w", err)
	}

	data := url.Values{
		"apikey":                {c.apiKey},
		"module":                {"contract"},
		"action":                {"verifysourcecode"},
		"contractaddress":       {address.Hex()},
		"codeformat":            {"solidity-standard-json-input"},
		"sourceCode":            {string(standardInputJSON)},
		"contractname":          {artifact.ContractName},
		"compilerversion":       {fmt.Sprintf("v%s", artifact.CompilerVersion)},
		"optimizationUsed":      {optimized},
		"runs":                  {fmt.Sprintf("%d", artifact.Optimizer.Runs)},
		"evmversion":            {artifact.EVMVersion},
		"constructorArguements": {constructorArgs},
	}

	req, err := http.NewRequest("POST", c.url, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create verification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.sendRateLimitedRequest(req)
	if err != nil {
		return "", fmt.Errorf("failed to submit verification request: %w", err)
	}
	defer resp.Body.Close()

	var result EtherscanGenericResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "1" {
		return "", fmt.Errorf("verification request failed: status=%s message=%s result=%s",
			result.Status, result.Message, result.Result)
	}

	return result.Result, nil
}

func (c *EtherscanClient) isVerified(address common.Address) (bool, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?module=contract&action=getabi&address=%s&apikey=%s",
		c.url, address.Hex(), c.apiKey), nil)
	if err != nil {
		return false, err
	}

	resp, err := c.sendRateLimitedRequest(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result EtherscanGenericResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Status == "1", nil
}

func (c *EtherscanClient) pollVerificationStatus(reqId string) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?apikey=%s&module=contract&action=checkverifystatus&guid=%s",
		c.url, c.apiKey, reqId), nil)
	if err != nil {
		return fmt.Errorf("failed to create checkverifystatus request: %w", err)
	}

	for i := 0; i < 10; i++ { // Try 10 times with increasing delays
		resp, err := c.sendRateLimitedRequest(req)
		if err != nil {
			return fmt.Errorf("failed to send checkverifystatus request: %w", err)
		}
		defer resp.Body.Close()

		var result EtherscanGenericResp
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to decode checkverifystatus response: %w", err)
		}

		if result.Status == "1" {
			return nil
		}
		if result.Result == "Already Verified" {
			return nil
		}
		if result.Result != "Pending in queue" {
			return fmt.Errorf("verification failed: %s, %s", result.Result, result.Message)
		}
		time.Sleep(time.Duration(i+2) * time.Second)
	}
	return fmt.Errorf("verification timed out")
}

type StandardInput struct {
	Language string                   `json:"language"`
	Sources  map[string]SourceContent `json:"sources"`
	Settings Settings                 `json:"settings"`
}

type SourceContent struct {
	Content string `json:"content"`
}

type Settings struct {
	Optimizer       OptimizerSettings `json:"optimizer"`
	EVMVersion      string            `json:"evmVersion"`
	Metadata        MetadataSettings  `json:"metadata"`
	OutputSelection OutputSelection   `json:"outputSelection"`
}

type OptimizerSettings struct {
	Enabled bool `json:"enabled"`
	Runs    int  `json:"runs"`
}

type MetadataSettings struct {
	UseLiteralContent bool   `json:"useLiteralContent"`
	BytecodeHash      string `json:"bytecodeHash"`
}

type OutputSelection struct {
	All map[string]OutputSelectionDetails `json:"*"`
}

type OutputSelectionDetails struct {
	All []string `json:"*"`
}

func newStandardInput(artifact *contractArtifact) StandardInput {
	return StandardInput{
		Language: "Solidity",
		Sources:  artifact.Sources,
		Settings: Settings{
			Optimizer: OptimizerSettings{
				Enabled: artifact.Optimizer.Enabled,
				Runs:    artifact.Optimizer.Runs,
			},
			EVMVersion: artifact.EVMVersion,
			Metadata: MetadataSettings{
				UseLiteralContent: true,
				BytecodeHash:      "none",
			},
			OutputSelection: OutputSelection{
				All: map[string]OutputSelectionDetails{
					"*": {
						All: []string{
							"abi",
							"evm.bytecode.object",
							"evm.bytecode.sourceMap",
							"evm.deployedBytecode.object",
							"evm.deployedBytecode.sourceMap",
							"metadata",
						},
					},
				},
			},
		},
	}
}
