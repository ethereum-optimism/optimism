package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type EtherscanChecker struct {
	apiKey  string
	chainID uint64
	logger  log.Logger
	client  *http.Client
}

func NewEtherscanChecker(apiKey string, chainID uint64, logger log.Logger) *EtherscanChecker {
	return &EtherscanChecker{
		apiKey:  apiKey,
		chainID: chainID,
		logger:  logger,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (e *EtherscanChecker) CanCheck() bool {
	return e.apiKey != ""
}

func (e *EtherscanChecker) GetDefaultURL(chainID uint64) (string, error) {
	return "", nil
}

func (e *EtherscanChecker) GetChainArg(chainID uint64) (string, error) {
	return getChainName(chainID)
}

func getChainName(chainID uint64) (string, error) {
	switch chainID {
	case 1:
		return "mainnet", nil
	case 11155111:
		return "sepolia", nil
	default:
		return "", fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func (e *EtherscanChecker) CheckStatus(ctx context.Context, address common.Address) (*VerificationStatus, error) {
	baseURL := "https://api.etherscan.io/v2/api"
	checkUrl := fmt.Sprintf("%s?chainid=%d&module=contract&action=getsourcecode&address=%s&apikey=%s", baseURL, e.chainID, address.Hex(), e.apiKey)

	e.logger.Debug("Checking Etherscan verification status via V2 API", "url", checkUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", checkUrl, nil)
	if err != nil {
		e.logger.Warn("Failed to create HTTP request for Etherscan API", "error", err)
		return nil, err
	}

	resp, err := e.client.Do(req)
	if err != nil {
		e.logger.Warn("Failed to query Etherscan API", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		e.logger.Info("Etherscan API returned non-OK status", "status", resp.StatusCode)
		return &VerificationStatus{IsVerified: false, IsFullyVerified: false, IsPartiallyVerified: false}, nil
	}

	var response struct {
		Status  string      `json:"status"`
		Message string      `json:"message"`
		Result  interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		e.logger.Warn("Failed to parse Etherscan API response", "error", err)
		return nil, err
	}

	if response.Status == "0" && strings.Contains(response.Message, "deprecated") {
		e.logger.Warn("Etherscan API returned deprecation message, cannot check verification status via API. Will rely on forge's own verification detection.")
		return &VerificationStatus{IsVerified: false, IsFullyVerified: false, IsPartiallyVerified: false}, nil
	}

	isVerified := false
	// Marshal and unmarshal result to handle both string and array cases
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		e.logger.Warn("Failed to marshal Etherscan API result", "error", err)
		return &VerificationStatus{IsVerified: false, IsFullyVerified: false, IsPartiallyVerified: false}, nil
	}

	var resultArray []struct {
		SourceCode string `json:"SourceCode"`
		ABI        string `json:"ABI"`
	}
	if err := json.Unmarshal(resultBytes, &resultArray); err == nil && len(resultArray) > 0 {
		sourceCode := resultArray[0].SourceCode
		abi := resultArray[0].ABI
		isVerified = sourceCode != "" &&
			sourceCode != "{{" &&
			abi != "" &&
			abi != "Contract source code not verified"
	} else {
		// If unmarshaling as array failed, result might be a string (error case)
		var resultStr string
		if err := json.Unmarshal(resultBytes, &resultStr); err == nil {
			if resultStr == "" || resultStr == "Contract source code not verified" {
				isVerified = false
			}
		}
	}

	e.logger.Info("Etherscan API verification status", "address", address.Hex(), "is_verified", isVerified)
	return &VerificationStatus{
		IsVerified:          isVerified,
		IsFullyVerified:     isVerified,
		IsPartiallyVerified: false,
	}, nil
}
