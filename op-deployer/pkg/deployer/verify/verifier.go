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
)

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
	totalFailed := 0
	allFailedContracts := make(map[string][]string)

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

		var numVerified, numSkipped, numFailed int
		var failedContracts []string

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
			} else {
				numFailed++
				failedContracts = append(failedContracts, contractName)
			}
		} else {
			numVerified, numSkipped, numFailed, failedContracts = v.VerifyContracts(ctx, bundle)
		}

		l.Info("Verification complete", "verifier", vt, "verified", numVerified, "skipped", numSkipped, "failed", numFailed)

		totalVerified += numVerified
		totalSkipped += numSkipped
		totalFailed += numFailed

		if numFailed > 0 {
			allFailedContracts[vt] = failedContracts
		}
	}

	l.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	l.Info("Verification Summary")
	l.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	l.Info("Results", "verified", totalVerified, "skipped", totalSkipped, "failed", totalFailed)

	if len(allFailedContracts) > 0 {
		l.Warn("Failed contracts by verifier:")
		for verifier, contracts := range allFailedContracts {
			l.Warn(fmt.Sprintf("  %s:", verifier))
			for _, contract := range contracts {
				l.Warn(fmt.Sprintf("    - %s", contract))
			}
		}
	}

	if totalFailed > 0 {
		l.Warn(fmt.Sprintf("Failed to verify %d contracts", totalFailed))
		return fmt.Errorf("failed to verify %d contracts", totalFailed)
	} else if totalSkipped > 0 {
		l.Info("All contracts verified or already verified")
	} else {
		l.Info("All contracts verified successfully")
	}
	l.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}
