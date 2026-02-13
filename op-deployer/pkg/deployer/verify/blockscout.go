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

type BlockscoutChecker struct {
	verifierUrl string
	chainID     uint64
	logger      log.Logger
	apiUrl      string
	client      *http.Client
}

func NewBlockscoutChecker(verifierUrl string, chainID uint64, logger log.Logger) *BlockscoutChecker {
	apiUrl := verifierUrl
	if apiUrl == "" {
		if url, err := getBlockscoutAPIEndpoint(chainID); err == nil {
			apiUrl = url
		}
	}
	// Normalize to base URL (remove /api or /api/ if present)
	apiUrl = strings.TrimSuffix(apiUrl, "/api/")
	apiUrl = strings.TrimSuffix(apiUrl, "/api")
	apiUrl = strings.TrimSuffix(apiUrl, "/")

	return &BlockscoutChecker{
		verifierUrl: verifierUrl,
		chainID:     chainID,
		logger:      logger,
		apiUrl:      apiUrl,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (b *BlockscoutChecker) CanCheck() bool {
	return true
}

func (b *BlockscoutChecker) GetDefaultURL(chainID uint64) (string, error) {
	return getBlockscoutAPIEndpoint(chainID)
}

func (b *BlockscoutChecker) GetChainArg(chainID uint64) (string, error) {
	return fmt.Sprintf("%d", chainID), nil
}

func getBlockscoutAPIEndpoint(l1ChainID uint64) (string, error) {
	switch l1ChainID {
	case 1:
		return "https://eth.blockscout.com/api/", nil
	case 11155111:
		return "https://eth-sepolia.blockscout.com/api/", nil
	default:
		return "", fmt.Errorf("unsupported L1 chain ID: %d", l1ChainID)
	}
}

func (b *BlockscoutChecker) CheckStatus(ctx context.Context, address common.Address) (*VerificationStatus, error) {
	checkUrl := fmt.Sprintf("%s/api/v2/smart-contracts/%s", b.apiUrl, address.Hex())

	b.logger.Debug("Checking Blockscout verification status via API", "url", checkUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", checkUrl, nil)
	if err != nil {
		b.logger.Warn("Failed to create HTTP request for Blockscout API", "error", err)
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		b.logger.Warn("Failed to query Blockscout API", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.logger.Info("Blockscout API returned non-OK status", "status", resp.StatusCode)
		return &VerificationStatus{IsVerified: false, IsFullyVerified: false, IsPartiallyVerified: false}, nil
	}

	var result struct {
		Message                 string `json:"message"`
		IsVerified              *bool  `json:"is_verified"`
		IsFullyVerified         *bool  `json:"is_fully_verified"`
		IsPartiallyVerified     *bool  `json:"is_partially_verified"`
		IsVerifiedViaBytecodeDB *bool  `json:"is_verified_via_eth_bytecode_db"`
		IsVerifiedViaSourcify   *bool  `json:"is_verified_via_sourcify"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		b.logger.Warn("Failed to parse Blockscout API response", "error", err)
		return nil, err
	}

	if result.Message == "Not found" {
		b.logger.Info("Blockscout API: contract not found", "address", address.Hex())
		return &VerificationStatus{IsVerified: false, IsFullyVerified: false, IsPartiallyVerified: false}, nil
	}

	status := &VerificationStatus{}
	if result.IsFullyVerified != nil && *result.IsFullyVerified {
		status.IsFullyVerified = true
		status.IsVerified = true
		b.logger.Info("Blockscout API: contract is fully verified", "address", address.Hex(),
			"is_verified", result.IsVerified,
			"is_fully_verified", result.IsFullyVerified,
			"is_partially_verified", result.IsPartiallyVerified)
	} else if result.IsPartiallyVerified != nil && *result.IsPartiallyVerified {
		status.IsPartiallyVerified = true
		status.IsVerified = true
		b.logger.Info("Blockscout API: contract is partially verified, will attempt full verification", "address", address.Hex(),
			"is_verified", result.IsVerified,
			"is_fully_verified", result.IsFullyVerified,
			"is_partially_verified", result.IsPartiallyVerified,
			"is_verified_via_eth_bytecode_db", result.IsVerifiedViaBytecodeDB,
			"is_verified_via_sourcify", result.IsVerifiedViaSourcify)
	} else if result.IsVerified != nil && *result.IsVerified {
		status.IsFullyVerified = true
		status.IsVerified = true
	} else {
		b.logger.Info("Blockscout API: contract is not verified", "address", address.Hex())
	}

	return status, nil
}
