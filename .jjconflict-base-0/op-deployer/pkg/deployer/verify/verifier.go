package verify

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/flags"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

func printVerificationSummary(logger log.Logger, verified, skipped, partiallyVerified, failed int, partiallyVerifiedContracts, failedContracts map[string][]string) {
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("Verification Summary")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("Results", "verified", verified, "skipped", skipped, "partially_verified", partiallyVerified, "failed", failed)

	if len(partiallyVerifiedContracts) > 0 {
		logger.Info("Partially verified contracts by verifier (forge cannot upgrade to full verification - this is expected behavior per foundry-rs/foundry#8638):")
		for verifier, contracts := range partiallyVerifiedContracts {
			logger.Info(fmt.Sprintf("  %s:", verifier))
			for _, contract := range contracts {
				logger.Info(fmt.Sprintf("    - %s", contract))
			}
		}
	}

	if len(failedContracts) > 0 {
		logger.Warn("Failed contracts by verifier:")
		for verifier, contracts := range failedContracts {
			logger.Warn(fmt.Sprintf("  %s:", verifier))
			for _, contract := range contracts {
				logger.Warn(fmt.Sprintf("    - %s", contract))
			}
		}
	}

	if failed > 0 {
		logger.Warn(fmt.Sprintf("Failed to verify %d contracts", failed))
	} else if partiallyVerified > 0 && verified == 0 && skipped == 0 {
		logger.Info("All contracts are partially verified (forge cannot upgrade to full verification - this is expected behavior per foundry-rs/foundry#8638)")
	} else if skipped > 0 {
		logger.Info("All contracts verified or already verified")
	} else {
		logger.Info("All contracts verified successfully")
	}
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func VerifyCLI(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	l1RPCUrl := cliCtx.String(flags.L1RPCURLFlagName)
	verifierAPIKey := cliCtx.String(flags.VerifierAPIKeyFlagName)
	verifierType := cliCtx.String(flags.VerifierTypeFlagName)
	verifierUrl := cliCtx.String(flags.VerifierUrlFlagName)

	verifiers := strings.Split(verifierType, ",")
	for i := range verifiers {
		verifiers[i] = strings.TrimSpace(verifiers[i])
	}

	needsAPIKey := false
	for _, v := range verifiers {
		if v == "etherscan" {
			needsAPIKey = true
			break
		}
	}

	if needsAPIKey && verifierAPIKey == "" {
		return fmt.Errorf("verifier-api-key is required for etherscan")
	}

	inputFile := cliCtx.String(flags.InputFileFlagName)
	if inputFile == "" {
		return fmt.Errorf("input-file is required")
	}
	contractName := cliCtx.String(flags.ContractNameFlagName)

	l1ContractsLocator := cliCtx.String(flags.ArtifactsLocatorFlagName)
	if l1ContractsLocator == "" {
		return fmt.Errorf("artifacts-locator is required")
	}

	ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

	l1Client, err := ethclient.Dial(l1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1: %w", err)
	}
	defer l1Client.Close()

	chainId, err := l1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}
	l1ChainId := chainId.Uint64()

	locator, err := artifacts.NewLocatorFromURL(l1ContractsLocator)
	if err != nil {
		return fmt.Errorf("failed to parse l1 contracts release locator: %w", err)
	}

	cacheDir := flags.DefaultCacheDir()
	artifactsFS, err := artifacts.Download(ctx, locator, nil, cacheDir)
	if err != nil {
		return fmt.Errorf("failed to get artifacts: %w", err)
	}
	l.Info("Downloaded artifacts")

	bundle, err := GetBundleFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to retrieve bundle: %w", err)
	}

	l.Info("Starting contract verification", "verifiers", verifierType)

	totalVerified := 0
	totalSkipped := 0
	totalPartiallyVerified := 0
	totalFailed := 0
	allFailedContracts := make(map[string][]string)
	allPartiallyVerifiedContracts := make(map[string][]string)

	for _, vt := range verifiers {
		l.Info("Verifying contracts", "verifier", vt)

		v, err := NewForgeVerifier(ForgeVerifierOpts{
			RpcUrl:       l1RPCUrl,
			VerifierType: vt,
			VerifierUrl:  verifierUrl,
			ApiKey:       verifierAPIKey,
			ChainID:      l1ChainId,
			ArtifactsFS:  artifactsFS,
			Logger:       l,
		})
		if err != nil {
			errMsg := fmt.Sprintf("failed to create %s verifier: %v", vt, err)
			l.Error(errMsg)
			continue
		}

		var numVerified, numSkipped, numPartiallyVerified, numFailed int
		var failedContracts, partiallyVerifiedContracts []string

		if contractName != "" {
			addr, ok := bundle[contractName]
			if !ok {
				return fmt.Errorf("contract %s not found in bundle", contractName)
			}

			err := v.VerifyContract(ctx, addr, contractName)
			if err == nil {
				numVerified++
			} else if err == ErrAlreadyVerified {
				numSkipped++
			} else if err == ErrPartiallyVerified {
				numPartiallyVerified++
				partiallyVerifiedContracts = append(partiallyVerifiedContracts, contractName)
			} else {
				numFailed++
				failedContracts = append(failedContracts, contractName)
			}
		} else {
			numVerified, numSkipped, numPartiallyVerified, numFailed, failedContracts, partiallyVerifiedContracts = v.VerifyContracts(ctx, bundle)
		}

		l.Info("Verification complete", "verifier", vt, "verified", numVerified, "skipped", numSkipped, "partially_verified", numPartiallyVerified, "failed", numFailed)

		totalVerified += numVerified
		totalSkipped += numSkipped
		totalPartiallyVerified += numPartiallyVerified
		totalFailed += numFailed

		if numFailed > 0 {
			allFailedContracts[vt] = failedContracts
		}
		if numPartiallyVerified > 0 {
			allPartiallyVerifiedContracts[vt] = partiallyVerifiedContracts
		}
	}

	printVerificationSummary(l, totalVerified, totalSkipped, totalPartiallyVerified, totalFailed, allPartiallyVerifiedContracts, allFailedContracts)

	if totalFailed > 0 {
		return fmt.Errorf("failed to verify %d contracts", totalFailed)
	}
	return nil
}
