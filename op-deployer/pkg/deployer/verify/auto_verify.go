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

	// Right now we only support one api key for all the verifiers, would need to change this in the future
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
	totalFailed := 0
	allFailedContracts := make(map[string][]string)

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

		numVerified, numSkipped, numFailed, failedContracts := v.VerifyContracts(ctx, bundle)
		logger.Info("Verification complete", "verifier", verifierType, "verified", numVerified, "skipped", numSkipped, "failed", numFailed)

		totalVerified += numVerified
		totalSkipped += numSkipped
		totalFailed += numFailed

		if numFailed > 0 {
			allFailedContracts[verifierType] = failedContracts
		}
	}

	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("Verification Summary")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("Results", "verified", totalVerified, "skipped", totalSkipped, "failed", totalFailed)

	if len(allFailedContracts) > 0 {
		logger.Warn("Failed contracts by verifier:")
		for verifierType, contracts := range allFailedContracts {
			logger.Warn(fmt.Sprintf("  %s:", verifierType))
			for _, contract := range contracts {
				logger.Warn(fmt.Sprintf("    - %s", contract))
			}
		}
		logger.Warn("Deployment succeeded but verification incomplete")
		logger.Warn("You can retry verification later using: op-deployer verify --input-file <state-file>")
	} else if totalSkipped > 0 {
		logger.Info("All contracts verified or already verified")
	} else {
		logger.Info("All contracts verified successfully")
	}
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return nil
}
