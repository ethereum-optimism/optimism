package verify

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/flags"
	"github.com/ethereum/go-ethereum/log"
)

func AutoVerify(ctx context.Context, logger log.Logger, rpcUrl string, chainID uint64, stateFile string, artifactsLocator *artifacts.Locator, verifierTypes string, verifierUrl string, apiKey string) error {
	verifiers := strings.Split(verifierTypes, ",")
	for i := range verifiers {
		verifiers[i] = strings.TrimSpace(verifiers[i])
	}

	needsAPIKey := false
	for _, verifierType := range verifiers {
		if verifierType == "etherscan" {
			needsAPIKey = true
			break
		}
	}

	if needsAPIKey && apiKey == "" {
		logger.Warn("Skipping auto-verification: etherscan verifier requires an API key")
		return nil
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

	for _, verifierType := range verifiers {
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
			errMsg := fmt.Sprintf("failed to create %s verifier: %v", verifierType, err)
			logger.Error(errMsg)
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

	printVerificationSummary(logger, totalVerified, totalSkipped, totalPartiallyVerified, totalFailed, allPartiallyVerifiedContracts, allFailedContracts)

	if len(allFailedContracts) > 0 {
		logger.Warn("Deployment succeeded but verification incomplete")
		logger.Warn("You can retry verification later using: op-deployer verify --input-file <state-file>")
	}

	return nil
}
