package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/time/rate"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

type Verifier struct {
	l1ChainID   uint64
	artifactsFS foundry.StatDirFs
	log         log.Logger
	etherscan   *EtherscanClient
	l1Client    *ethclient.Client
	numVerified int
	numSkipped  int
	numFailed   int
}

func NewVerifier(apiKey string, l1ChainID uint64, artifactsFS foundry.StatDirFs, l log.Logger, l1Client *ethclient.Client) (*Verifier, error) {
	etherscanUrl, err := getAPIEndpoint(l1ChainID)
	if err != nil {
		return nil, fmt.Errorf("unsupported L1 chain ID: %d", l1ChainID)
	}

	etherscan := NewEtherscanClient(apiKey, etherscanUrl, rate.NewLimiter(rate.Limit(3), 2))

	return &Verifier{
		l1ChainID:   l1ChainID,
		artifactsFS: artifactsFS,
		log:         l,
		l1Client:    l1Client,
		etherscan:   etherscan,
	}, nil
}

func VerifyCLI(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	l1RPCUrl := cliCtx.String(deployer.L1RPCURLFlagName)
	etherscanAPIKey := cliCtx.String(deployer.EtherscanAPIKeyFlagName)
	if etherscanAPIKey == "" {
		return fmt.Errorf("etherscan-api-key is required")
	}

	inputFile := cliCtx.String(deployer.InputFileFlagName)
	if inputFile == "" {
		return fmt.Errorf("input-file is required")
	}
	contractName := cliCtx.String(deployer.ContractNameFlagName)

	l1ContractsLocator := cliCtx.String(deployer.ArtifactsLocatorFlagName)
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
	artifactsFS, err := artifacts.Download(ctx, locator, nil, deployer.GetDefaultCacheDir())
	if err != nil {
		return fmt.Errorf("failed to get artifacts: %w", err)
	}
	l.Info("Downloaded artifacts", "path", artifactsFS)

	v, err := NewVerifier(etherscanAPIKey, l1ChainId, artifactsFS, l, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	defer func() {
		v.log.Info("final results", "numVerified", v.numVerified, "numSkipped", v.numSkipped, "numFailed", v.numFailed)
	}()

	if err := v.verifyContractBundle(ctx, inputFile, contractName); err != nil {
		return err
	}
	v.log.Info("--- COMPLETE ---")
	return nil
}

func (v *Verifier) getContractBundle(filepath string) (map[string]common.Address, error) {
	_, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("input file not found: %s", filepath)
	}

	bundleData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundle file %s: %w", filepath, err)
	}

	var bundle map[string]common.Address
	if err := json.Unmarshal(bundleData, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse superchain bundle: %w", err)
	}

	return bundle, nil
}

func (v *Verifier) verifyContractBundle(ctx context.Context, filepath string, contractName string) error {
	bundle, err := v.getContractBundle(filepath)
	if err != nil {
		return fmt.Errorf("failed to retrieve bundle: %w", err)
	}

	if contractName != "" {
		addr, ok := bundle[contractName]
		if !ok {
			return fmt.Errorf("contract %s not found in bundle", contractName)
		}
		if err := v.verifySingleContract(ctx, addr, contractName); err != nil {
			return fmt.Errorf("failed to verify contract %s: %w", contractName, err)
		}
		return nil
	}

	for contractName, addr := range bundle {
		if addr != (common.Address{}) { // skip zero addresses
			if err := v.verifySingleContract(ctx, addr, contractName); err != nil {
				v.numFailed++
				v.log.Error("failed to verify contract", "name", contractName, "error", err)
			}
		}
	}
	return nil
}

func (v *Verifier) verifySingleContract(ctx context.Context, address common.Address, contractName string) error {
	verified, err := v.etherscan.isVerified(address)
	if err != nil {
		return fmt.Errorf("failed to check verification status: %w", err)
	}
	if verified {
		v.log.Info("Contract is already verified", "name", contractName, "address", address.Hex())
		v.numSkipped++
		return nil
	}

	v.log.Info("Formatting etherscan verify request", "name", contractName, "address", address.Hex())
	artifact, err := v.getContractArtifact(contractName)
	if err != nil {
		return fmt.Errorf("failed to get contract source: %w", err)
	}

	constructorArgs, err := v.getConstructorArgs(ctx, address, artifact)
	if err != nil {
		return fmt.Errorf("failed to get constructor args: %w", err)
	}

	reqId, err := v.etherscan.verifySourceCode(address, artifact, constructorArgs)
	if err != nil {
		return fmt.Errorf("failed to verify contract: %w", err)
	}
	v.log.Info("Verification request submitted", "name", contractName, "address", address.Hex())

	if err = v.etherscan.pollVerificationStatus(reqId); err != nil {
		return fmt.Errorf("failed when checking verification status: %w", err)
	}

	v.log.Info("Verification complete", "name", contractName, "address", address.Hex())
	v.numVerified++
	return nil
}
