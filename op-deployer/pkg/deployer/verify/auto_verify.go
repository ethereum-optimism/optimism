package verify

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/flags"
	"github.com/ethereum/go-ethereum/log"
)

func AutoVerify(ctx context.Context, logger log.Logger, rpcUrl string, chainID uint64, stateFile string, retryStateFile string, artifactsLocator *artifacts.Locator, verifierTypes string, verifierUrl string, apiKey string) error {
	verifiers := strings.Split(verifierTypes, ",")
	for i := range verifiers {
		verifiers[i] = strings.TrimSpace(verifiers[i])
	}

	logger.Info("Starting automatic contract verification", "verifiers", verifierTypes)

	cacheDir := flags.DefaultCacheDir()
	artifactsFS, err := artifacts.Download(ctx, artifactsLocator, nil, cacheDir)
	if err != nil {
		return fmt.Errorf("failed to download artifacts: %w", err)
	}

	bundle, err := GetBundleFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("failed to get contract bundle: %w", err)
	}

	totalVerified := 0
	totalSkipped := 0
	totalPartiallyVerified := 0
	totalFailed := 0
	allFailedContracts := make(map[string][]string)
	allPartiallyVerifiedContracts := make(map[string][]string)
	unavailableVerifiers := make([]string, 0)
	missingEtherscanKey := false

	for _, verifierType := range verifiers {
		if verifierType == "etherscan" && apiKey == "" {
			logger.Error("Contract verifier unavailable", "verifier", verifierType, "reason", "API key is required")
			unavailableVerifiers = append(unavailableVerifiers, verifierType)
			missingEtherscanKey = true
			continue
		}

		logger.Info("Verifying contracts", "verifier", verifierType)

		v, err := NewForgeVerifier(ForgeVerifierOpts{
			RpcUrl:       rpcUrl,
			VerifierType: verifierType,
			VerifierUrl:  verifierUrl,
			ApiKey:       apiKey,
			ChainID:      chainID,
			ArtifactsFS:  artifactsFS,
			Logger:       logger,
		})
		if err != nil {
			logger.Error("Contract verifier unavailable", "verifier", verifierType, "err", err)
			unavailableVerifiers = append(unavailableVerifiers, verifierType)
			continue
		}

		numVerified, numSkipped, numPartiallyVerified, numFailed, failedContracts, partiallyVerifiedContracts := v.VerifyContracts(ctx, bundle)
		logger.Info("Verification complete", "verifier", verifierType, "verified", numVerified, "skipped", numSkipped, "partially_verified", numPartiallyVerified, "failed", numFailed)

		totalVerified += numVerified
		totalSkipped += numSkipped
		totalPartiallyVerified += numPartiallyVerified
		totalFailed += numFailed

		if numFailed > 0 {
			allFailedContracts[verifierType] = failedContracts
		}
		if numPartiallyVerified > 0 {
			allPartiallyVerifiedContracts[verifierType] = partiallyVerifiedContracts
		}
	}

	printVerificationSummary(logger, totalVerified, totalSkipped, totalPartiallyVerified, totalFailed, len(unavailableVerifiers), allPartiallyVerifiedContracts, allFailedContracts)

	if totalFailed > 0 || len(unavailableVerifiers) > 0 {
		logger.Error("Deployment succeeded but contract verification incomplete",
			"failed", totalFailed,
			"partially_verified", totalPartiallyVerified,
			"unavailable", strings.Join(unavailableVerifiers, ","))
		logRetryCommand(logger, retryStateFile, missingEtherscanKey)
	}

	return nil
}

func LogAutoVerifyFailure(logger log.Logger, stateFile string, err error) {
	logger.Error("Deployment succeeded but contract verification incomplete", "err", err)
	logRetryCommand(logger, stateFile, false)
}

func logRetryCommand(logger log.Logger, stateFile string, missingEtherscanKey bool) {
	if stateFile == "" || stateFile == "-" {
		stateFile = "<state-file>"
		logger.Error("Save the deployment output to a state file before retrying verification")
	}
	command := fmt.Sprintf("op-deployer verify --input-file %s", stateFile)
	if missingEtherscanKey {
		command += " --verifier-api-key <your-etherscan-api-key>"
	}
	logger.Error("Retry contract verification", "command", command)
}
