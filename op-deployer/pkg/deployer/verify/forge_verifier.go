package verify

import (
	"context"
	"fmt"
	"os"
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
	apiChecker   APIChecker
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
	if _, err := os.Stat(forgeTomlPath); err != nil {
		opts.Logger.Warn("foundry.toml not found, checking parent directory", "path", forgeTomlPath)
		forgeTomlPath = filepath.Join(fmt.Sprintf("%v", opts.ArtifactsFS), "..", "foundry.toml")
		if _, err := os.Stat(forgeTomlPath); err != nil {
			return nil, fmt.Errorf("foundry.toml not found in any of the possible directories: %w", err)
		}
	}
	forgeClient, err := forge.NewStandardClient(forgeTomlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create forge client: %w", err)
	}

	var apiChecker APIChecker
	switch opts.VerifierType {
	case "etherscan":
		apiChecker = NewEtherscanChecker(opts.ApiKey, opts.ChainID, opts.Logger)
	case "blockscout":
		if opts.VerifierUrl == "" {
			url, err := getBlockscoutAPIEndpoint(opts.ChainID)
			if err != nil {
				return nil, fmt.Errorf("failed to get verifier URL for chain %d: %w", opts.ChainID, err)
			}
			opts.VerifierUrl = url
		}
		opts.VerifierUrl = strings.TrimSuffix(opts.VerifierUrl, "/")
		apiChecker = NewBlockscoutChecker(opts.VerifierUrl, opts.ChainID, opts.Logger)
	case "custom":
		apiChecker = NewBlockscoutChecker(opts.VerifierUrl, opts.ChainID, opts.Logger)
	default:
		apiChecker = nil
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
		apiChecker:   apiChecker,
	}, nil
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

	var initialStatus *VerificationStatus
	if v.apiChecker != nil && v.apiChecker.CanCheck() {
		v.logger.Info("Checking verification status via API before attempting forge verify", "address", address.Hex())
		status, err := v.apiChecker.CheckStatus(ctx, address)
		if err != nil {
			v.logger.Warn("Failed to check verification status via API, proceeding with forge verify", "address", address.Hex(), "error", err)
		} else if status != nil {
			initialStatus = status
			if status.IsFullyVerified {
				v.logger.Info("Contract already verified (confirmed via API), skipping forge verify", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
				return ErrAlreadyVerified
			} else if status.IsPartiallyVerified {
				v.logger.Info("Contract is partially verified, proceeding with forge verify to attempt full verification", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
			} else {
				v.logger.Info("Contract not verified according to API, proceeding with forge verify", "address", address.Hex())
			}
		}
	}

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

	if v.apiChecker != nil {
		chainArg, err := v.apiChecker.GetChainArg(v.chainID)
		if err != nil {
			return fmt.Errorf("failed to get chain argument: %w", err)
		}
		if v.verifierType == "blockscout" {
			args = append(args, "--chain-id", chainArg)
			args = append(args, "--verifier", "blockscout")
		} else if v.verifierType == "custom" {
			if v.verifierUrl == "" {
				return fmt.Errorf("--verifier-url is required when using custom verifier")
			}
			args = append(args, "--chain-id", chainArg)
			args = append(args, "--verifier", "custom")
		} else {
			args = append(args, "--chain", chainArg)
			args = append(args, "--verifier", "etherscan")
		}
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

		if strings.Contains(output, "Fail - Unable to verify") || strings.Contains(output, "Fail - Failed to Verify") {
			v.logger.Info("Block explorer reported verification failure, checking verification status via API", "address", address.Hex(), "error", output)
			if v.apiChecker != nil && v.apiChecker.CanCheck() {
				status, err := v.apiChecker.CheckStatus(ctx, address)
				if err != nil {
					v.logger.Warn("Failed to check verification status via API", "address", address.Hex(), "error", err)
				} else if status != nil {
					if status.IsFullyVerified {
						v.logger.Info("Contract is fully verified (confirmed via API despite forge error)", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
						return ErrAlreadyVerified
					} else if status.IsPartiallyVerified {
						v.logger.Info("Contract is partially verified but forge cannot upgrade to full verification (this is expected behavior per foundry-rs/foundry#8638)", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
						return ErrPartiallyVerified
					} else {
						v.logger.Info("API confirms contract is NOT verified", "address", address.Hex())
					}
				}
			}
			return fmt.Errorf("forge verification failed: block explorer reported verification failure")
		}

		if verifyErr == nil {
			break
		}

		if isAlreadyVerified(verifyErr.Error(), output) {
			// Forge says it's already verified, check API status to distinguish fully vs partially verified
			if err := v.checkVerificationStatusAfterForgeReport(ctx, address, contractName, initialStatus); err != nil {
				return err
			}
			v.logger.Info("Contract already verified", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
			return ErrAlreadyVerified
		}

		isIndexingError := isContractNotFound(verifyErr.Error(), output)

		if isIndexingError && v.apiChecker != nil && v.apiChecker.CanCheck() && v.verifierType == "etherscan" {
			v.logger.Info("Contract not found, checking if already verified via Etherscan API", "address", address.Hex())
			status, err := v.apiChecker.CheckStatus(ctx, address)
			if err != nil {
				v.logger.Warn("Failed to check Etherscan verification status", "address", address.Hex(), "error", err)
			} else if status != nil && status.IsFullyVerified {
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

	if strings.Contains(output, "is already verified") || strings.Contains(output, "Skipping verification") {
		if err := v.checkVerificationStatusAfterForgeReport(ctx, address, contractName, initialStatus); err != nil {
			return err
		}
	}

	v.logger.Info("Contract verified successfully", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
	return nil
}

var (
	ErrAlreadyVerified   = fmt.Errorf("contract already verified")
	ErrPartiallyVerified = fmt.Errorf("contract is partially verified but cannot be upgraded to full verification")
)

// checkVerificationStatusAfterForgeReport checks the API status after forge reports "already verified"
// to distinguish between fully verified and partially verified contracts.
// It re-checks the API status (as it might have changed) but falls back to initialStatus if the re-check fails.
func (v *ForgeVerifier) checkVerificationStatusAfterForgeReport(ctx context.Context, address common.Address, contractName string, initialStatus *VerificationStatus) error {
	var status *VerificationStatus
	if initialStatus != nil && v.apiChecker != nil && v.apiChecker.CanCheck() {
		var err error
		status, err = v.apiChecker.CheckStatus(ctx, address)
		if err != nil {
			status = initialStatus
		}
	} else if v.apiChecker != nil && v.apiChecker.CanCheck() {
		var err error
		status, err = v.apiChecker.CheckStatus(ctx, address)
		if err != nil {
			v.logger.Warn("Failed to check verification status via API after forge reported already verified", "address", address.Hex(), "error", err)
		}
	}

	if status != nil {
		if status.IsFullyVerified {
			v.logger.Info("Contract already verified (confirmed via API)", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
			return ErrAlreadyVerified
		} else if status.IsPartiallyVerified {
			v.logger.Info("Contract is partially verified but forge cannot upgrade to full verification (this is expected behavior per foundry-rs/foundry#8638)", "name", contractName, "address", address.Hex(), "verifier", v.verifierType)
			return ErrPartiallyVerified
		}
	}
	return nil
}

func (v *ForgeVerifier) VerifyContracts(ctx context.Context, contracts map[string]common.Address) (verified, skipped, partiallyVerified, failed int, failedContracts, partiallyVerifiedContracts []string) {
	for contractName, addr := range contracts {
		if addr == (common.Address{}) {
			continue
		}

		err := v.VerifyContract(ctx, addr, contractName)
		if err == nil {
			verified++
		} else if err == ErrAlreadyVerified {
			skipped++
		} else if err == ErrPartiallyVerified {
			partiallyVerified++
			partiallyVerifiedContracts = append(partiallyVerifiedContracts, fmt.Sprintf("%s (%s)", contractName, addr.Hex()))
		} else {
			v.logger.Error("Failed to verify contract", "name", contractName, "address", addr.Hex(), "error", err)
			failed++
			failedContracts = append(failedContracts, fmt.Sprintf("%s (%s)", contractName, addr.Hex()))
		}
	}

	return verified, skipped, partiallyVerified, failed, failedContracts, partiallyVerifiedContracts
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
