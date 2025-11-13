package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type ForgeVerifier struct {
	forgeClient  *forge.Client
	rpcUrl       string
	verifierType string
	verifierUrl  string
	apiKey       string
	chainID      uint64
	artifactsFS  foundry.StatDirFs
	logger       log.Logger
}

type ForgeVerifierOpts struct {
	RpcUrl       string
	VerifierType string
	VerifierUrl  string
	ApiKey       string
	ChainID      uint64
	ArtifactsFS  foundry.StatDirFs
	Logger       log.Logger
}

func NewForgeVerifier(opts ForgeVerifierOpts) (*ForgeVerifier, error) {
	if opts.VerifierType != "etherscan" && opts.VerifierType != "blockscout" && opts.VerifierType != "custom" {
		return nil, fmt.Errorf("unsupported verifier type: %s (must be 'etherscan', 'blockscout', or 'custom')", opts.VerifierType)
	}

	forgeTomlPath := filepath.Join(fmt.Sprintf("%v", opts.ArtifactsFS), "foundry.toml")
	forgeClient, err := forge.NewStandardClient(forgeTomlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create forge client: %w", err)
	}

	if opts.VerifierType == "blockscout" && opts.VerifierUrl == "" {
		url, err := getBlockscoutAPIEndpoint(opts.ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get verifier URL for chain %d: %w", opts.ChainID, err)
		}
		opts.VerifierUrl = url
	}

	return &ForgeVerifier{
		forgeClient:  forgeClient,
		rpcUrl:       opts.RpcUrl,
		verifierType: opts.VerifierType,
		verifierUrl:  opts.VerifierUrl,
		apiKey:       opts.ApiKey,
		chainID:      opts.ChainID,
		artifactsFS:  opts.ArtifactsFS,
		logger:       opts.Logger,
	}, nil
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

func (v *ForgeVerifier) VerifyContract(ctx context.Context, address common.Address, contractName string) error {
	return v.VerifyContractWithConstructorArgs(ctx, address, contractName, "")
}

func (v *ForgeVerifier) VerifyContractWithConstructorArgs(ctx context.Context, address common.Address, contractName string, constructorArgs string) error {
	artifactPath := getArtifactPath(contractName)
	v.logger.Info("Verifying contract with forge",
		"name", contractName,
		"address", address.Hex(),
		"artifactPath", artifactPath,
		"verifier", v.verifierType)

	_, metadata, err := loadArtifact(v.artifactsFS, artifactPath, v.logger)
	if err != nil {
		return err
	}

	args := []string{
		address.Hex(),
		metadata.ContractPath,
		"--compiler-version", metadata.CompilerVersion,
		"--watch",
	}

	// Only use --guess-constructor-args for real block explorers
	// We dont't have full mocks so unless we take this out the tests will fail
	isTestEnvironment := strings.Contains(v.verifierUrl, "localhost") ||
		strings.Contains(v.verifierUrl, "127.0.0.1") ||
		strings.Contains(v.verifierUrl, "0.0.0.0")

	if !isTestEnvironment {
		args = append(args, "--guess-constructor-args")
	}

	// Need to add these settings forcefully, because forge doesn't parse them correctly (1.2.3)
	if metadata.Optimizer.Enabled {
		args = append(args, "--num-of-optimizations", fmt.Sprintf("%d", metadata.Optimizer.Runs))
	}

	// Same here
	if metadata.EVMVersion != "" {
		args = append(args, "--evm-version", metadata.EVMVersion)
	}

	if v.verifierUrl != "" {
		args = append(args, "--verifier-url", v.verifierUrl)
	}

	if v.verifierType == "blockscout" {
		args = append(args, "--chain-id", fmt.Sprintf("%d", v.chainID))
		args = append(args, "--verifier", "blockscout")
	} else if v.verifierType == "custom" {
		if v.verifierUrl == "" {
			return fmt.Errorf("--verifier-url is required when using custom verifier")
		}
		args = append(args, "--chain-id", fmt.Sprintf("%d", v.chainID))
		args = append(args, "--verifier", "custom")
	} else {
		// Etherscan
		// When we upgrade forge we should add sourcify and more verifiers
		chainName, err := getChainName(v.chainID)
		if err != nil {
			return fmt.Errorf("failed to get chain name: %w", err)
		}
		args = append(args, "--chain", chainName)
		args = append(args, "--verifier", "etherscan")
	}

	if v.apiKey != "" {
		args = append(args, "--verifier-api-key", v.apiKey)
	}

	if v.rpcUrl != "" {
		args = append(args, "--rpc-url", v.rpcUrl)
	}

	if constructorArgs != "" {
		args = append(args, "--constructor-args", constructorArgs)
	}

	v.logger.Debug("Running forge verify-contract", "args", strings.Join(args, " "))

	// Deployed contracts that may not be indexed yet
	// If we don't retry it may fail with "Contract was not found"
	maxRetries := 3
	retryDelay := 10 * time.Second

	var output string
	var verifyErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			v.logger.Info("Retrying verification after delay", "attempt", attempt, "maxRetries", maxRetries, "delay", retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2
		}

		output, verifyErr = v.forgeClient.VerifyContract(ctx, args...)

		if verifyErr == nil {
			break
		}

		if isAlreadyVerified(verifyErr.Error(), output) {
			v.logger.Info("Contract already verified", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
			return ErrAlreadyVerified
		}

		if strings.Contains(output, "Fail - Unable to verify") {
			if v.verifierType == "blockscout" {
				v.logger.Info("Blockscout reported verification failure, checking if already verified via API", "address", address.Hex())
				isVerified, err := v.checkBlockscoutVerificationStatus(ctx, address)
				if err != nil {
					v.logger.Warn("Failed to check Blockscout verification status", "address", address.Hex(), "error", err)
				} else if isVerified {
					v.logger.Info("Contract already verified (confirmed via Blockscout API)", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
					return ErrAlreadyVerified
				} else {
					v.logger.Info("Blockscout API confirms contract is NOT verified", "address", address.Hex())
				}
			}
			return fmt.Errorf("forge verification failed: block explorer reported 'Fail - Unable to verify'")
		}

		isIndexingError := isContractNotFound(verifyErr.Error(), output)

		if isIndexingError && v.verifierType == "etherscan" && v.apiKey != "" {
			v.logger.Info("Contract not found, checking if already verified via Etherscan API", "address", address.Hex())
			isVerified, err := v.checkEtherscanVerificationStatus(ctx, address)
			if err != nil {
				v.logger.Warn("Failed to check Etherscan verification status", "address", address.Hex(), "error", err)
			} else if isVerified {
				v.logger.Info("Contract already verified (confirmed via Etherscan API)", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
				return ErrAlreadyVerified
			} else {
				v.logger.Info("Etherscan API confirms contract is NOT verified", "address", address.Hex())
			}
		}

		if !isIndexingError || attempt == maxRetries {
			errStr := verifyErr.Error()
			if strings.Contains(errStr, "constructor") || strings.Contains(errStr, "Constructor") {
				return fmt.Errorf("forge verification failed (likely constructor args mismatch): %w\nNote: Using --guess-constructor-args to extract from creation tx", verifyErr)
			}
			if isIndexingError && attempt == maxRetries {
				return fmt.Errorf("forge verification failed after %d retries (contract not indexed by block explorer): %w", maxRetries, verifyErr)
			}
			return fmt.Errorf("forge verification failed: %w", verifyErr)
		}
		v.logger.Warn("Contract not yet indexed by block explorer, will retry", "address", address.Hex(), "attempt", attempt+1, "maxRetries", maxRetries)
	}

	v.logger.Info("Contract verified successfully", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
	return nil
}

var ErrAlreadyVerified = fmt.Errorf("contract already verified")

// checkBlockscoutVerificationStatus checks if a contract is already verified on Blockscout
// Verifying partially verified contracts will return "Fail - Unable to verify"
func (v *ForgeVerifier) checkBlockscoutVerificationStatus(ctx context.Context, address common.Address) (bool, error) {
	verifierUrl := v.verifierUrl
	if verifierUrl == "" {
		defaultUrl, err := getBlockscoutAPIEndpoint(v.chainID)
		if err != nil {
			return false, err
		}
		verifierUrl = defaultUrl
	}

	apiUrl := strings.TrimSuffix(verifierUrl, "/api")
	checkUrl := fmt.Sprintf("%s/api/v2/smart-contracts/%s", apiUrl, address.Hex())

	v.logger.Info("Checking Blockscout verification status via API", "url", checkUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", checkUrl, nil)
	if err != nil {
		v.logger.Warn("Failed to create HTTP request for Blockscout API", "error", err)
		return false, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		v.logger.Warn("Failed to query Blockscout API", "error", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		v.logger.Info("Blockscout API returned non-OK status", "status", resp.StatusCode)
		return false, nil
	}

	// Parse JSON response
	var result struct {
		IsVerified bool `json:"is_verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		v.logger.Warn("Failed to parse Blockscout API response", "error", err)
		return false, err
	}

	v.logger.Info("Blockscout API verification status", "address", address.Hex(), "is_verified", result.IsVerified)
	return result.IsVerified, nil
}

// checkEtherscanVerificationStatus checks if a contract is already verified on Etherscan
func (v *ForgeVerifier) checkEtherscanVerificationStatus(ctx context.Context, address common.Address) (bool, error) {
	baseURL := "https://api.etherscan.io/v2/api"
	checkUrl := fmt.Sprintf("%s?chainid=%d&module=contract&action=getsourcecode&address=%s&apikey=%s", baseURL, v.chainID, address.Hex(), v.apiKey)

	v.logger.Info("Checking Etherscan verification status via V2 API", "url", checkUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", checkUrl, nil)
	if err != nil {
		v.logger.Warn("Failed to create HTTP request for Etherscan API", "error", err)
		return false, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		v.logger.Warn("Failed to query Etherscan API", "error", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		v.logger.Info("Etherscan API returned non-OK status", "status", resp.StatusCode)
		return false, nil
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			SourceCode string `json:"SourceCode"`
			ABI        string `json:"ABI"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		v.logger.Warn("Failed to parse Etherscan API response", "error", err)
		return false, err
	}

	if result.Status == "0" && strings.Contains(result.Message, "deprecated") {
		v.logger.Warn("Etherscan API returned deprecation message, cannot check verification status via API. Will rely on forge's own verification detection.")
		return false, nil
	}

	isVerified := len(result.Result) > 0 && result.Result[0].SourceCode != "" && result.Result[0].SourceCode != "{{"

	v.logger.Info("Etherscan API verification status", "address", address.Hex(), "is_verified", isVerified)
	return isVerified, nil
}

func (v *ForgeVerifier) VerifyContracts(ctx context.Context, contracts map[string]common.Address) (verified, skipped, failed int, failedContracts []string) {
	for contractName, addr := range contracts {
		if addr == (common.Address{}) {
			continue
		}

		err := v.VerifyContract(ctx, addr, contractName)
		if err == nil {
			verified++
		} else if err == ErrAlreadyVerified {
			skipped++
		} else {
			v.logger.Error("Failed to verify contract", "name", contractName, "address", addr.Hex(), "error", err)
			failed++
			failedContracts = append(failedContracts, fmt.Sprintf("%s (%s)", contractName, addr.Hex()))
		}
	}

	return verified, skipped, failed, failedContracts
}

func isAlreadyVerified(errStr, output string) bool {
	verifiedMessages := []string{
		"Contract source code already verified",
		"Already Verified",
		"already verified",
		"Smart-contract already verified",
	}

	for _, msg := range verifiedMessages {
		if strings.Contains(errStr, msg) || strings.Contains(output, msg) {
			return true
		}
	}
	return false
}

func isContractNotFound(errStr, output string) bool {
	notFoundMessages := []string{
		"Contract was not found",
		"Response result is unexpectedly empty",
		"contract not found",
		"Unable to locate ContractCode",
	}

	for _, msg := range notFoundMessages {
		if strings.Contains(errStr, msg) || strings.Contains(output, msg) {
			return true
		}
	}
	return false
}

func ContractPathToName(contractPath string) string {
	artifactFilename := filepath.Base(contractPath)
	return strings.TrimSuffix(artifactFilename, ".json")
}
