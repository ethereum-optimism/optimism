package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/time/rate"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

type Verifier struct {
	l1ChainID   uint64
	l2ChainID   common.Hash
	st          *state.State
	artifactsFS foundry.StatDirFs
	log         log.Logger
	etherscan   *EtherscanClient
	l1Client    *ethclient.Client
	numVerified int
	numSkipped  int
	numFailed   int
}

func NewVerifier(apiKey string, l1ChainID uint64, l2ChainID common.Hash, st *state.State, artifactsFS foundry.StatDirFs, l log.Logger, l1Client *ethclient.Client) (*Verifier, error) {
	etherscanUrl := getAPIEndpoint(l1ChainID)
	if etherscanUrl == "" {
		return nil, fmt.Errorf("unsupported L1 chain ID: %d", l1ChainID)
	}

	if l2ChainID == (common.Hash{}) {
		l2ChainID = st.AppliedIntent.Chains[0].ID
	}

	etherscan := NewEtherscanClient(apiKey, etherscanUrl, rate.NewLimiter(rate.Limit(3), 2))

	return &Verifier{
		l1ChainID:   l1ChainID,
		l2ChainID:   l2ChainID,
		st:          st,
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
	workdir := cliCtx.String(deployer.WorkdirFlagName)
	etherscanAPIKey := cliCtx.String(deployer.EtherscanAPIKeyFlagName)
	if etherscanAPIKey == "" {
		return fmt.Errorf("etherscan API key is required")
	}

	bundleName := cliCtx.String(deployer.ContractBundleFlagName)
	l2ChainIDRaw := cliCtx.String(deployer.L2ChainIDFlagName)

	var l2ChainID common.Hash
	var err error
	if l2ChainIDRaw != "" {
		l2ChainID, err = op_service.Parse256BitChainID(l2ChainIDRaw)
		if err != nil {
			return fmt.Errorf("invalid L2 chain ID '%s': %w", l2ChainIDRaw, err)
		}
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

	st, err := pipeline.ReadState(workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	if l1ChainId != st.AppliedIntent.L1ChainID {
		return fmt.Errorf("rpc l1 chain ID does not match state l1 chain ID: %d != %d", l1ChainId, st.AppliedIntent.L1ChainID)
	}

	artifactsFS, err := artifacts.Download(ctx, st.AppliedIntent.L1ContractsLocator, nil, deployer.DefaultCacheDir)
	if err != nil {
		return fmt.Errorf("failed to get artifacts: %w", err)
	}
	l.Info("Downloaded artifacts", "path", artifactsFS)

	v, err := NewVerifier(etherscanAPIKey, l1ChainId, l2ChainID, st, artifactsFS, l, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	defer func() {
		v.log.Info("final results", "numVerified", v.numVerified, "numSkipped", v.numSkipped, "numFailed", v.numFailed)
	}()

	if bundleName == "" {
		if err := v.verifyAll(ctx, workdir); err != nil {
			return err
		}
	} else if err := v.verifyContractBundle(ctx, workdir, bundleName); err != nil {
		return err
	}
	v.log.Info("--- COMPLETE ---")
	return nil
}

func (v *Verifier) verifyAll(ctx context.Context, workdir string) error {
	for _, bundleName := range inspect.ContractBundles {
		if err := v.verifyContractBundle(ctx, workdir, bundleName); err != nil {
			return fmt.Errorf("failed to verify bundle %s: %w", bundleName, err)
		}
	}
	return nil
}

func (v *Verifier) getContractBundle(workdir string, bundleName string) (interface{}, error) {
	bundleFilePath := filepath.Join(workdir, fmt.Sprintf("bootstrap_%s.json", bundleName))

	var bundle interface{}

	// Check if the bundle file exists
	if _, err := os.Stat(bundleFilePath); err == nil {
		// File exists, read and parse it
		v.log.Info("Found bundle file", "path", bundleFilePath)
		bundleData, err := os.ReadFile(bundleFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read bundle file %s: %w", bundleFilePath, err)
		}

		// Parse the file based on bundle type
		switch bundleName {
		case inspect.SuperchainBundle:
			var superchainBundle inspect.SuperchainDeployment
			if err := json.Unmarshal(bundleData, &superchainBundle); err != nil {
				return nil, fmt.Errorf("failed to parse superchain bundle: %w", err)
			}
			bundle = superchainBundle

		case inspect.ImplementationsBundle:
			var implBundle inspect.ImplementationsDeployment
			if err := json.Unmarshal(bundleData, &implBundle); err != nil {
				return nil, fmt.Errorf("failed to parse implementations bundle: %w", err)
			}
			bundle = implBundle

		case inspect.OpChainBundle:
			var opChainBundle inspect.OpChainDeployment
			if err := json.Unmarshal(bundleData, &opChainBundle); err != nil {
				return nil, fmt.Errorf("failed to parse opchain bundle: %w", err)
			}
			bundle = opChainBundle

		default:
			return nil, fmt.Errorf("invalid contract bundle: %s", bundleName)
		}
		v.log.Info("Using bundle file", "path", bundleFilePath)
	} else {
		v.log.Info("Bundle file not found, using state file", "bundle", bundleName)
		l1Contracts, err := inspect.L1(v.st, v.l2ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to extract L1 contracts from state: %w", err)
		}

		// Select the appropriate bundle based on the input bundleName.
		switch bundleName {
		case inspect.SuperchainBundle:
			bundle = l1Contracts.SuperchainDeployment
		case inspect.ImplementationsBundle:
			bundle = l1Contracts.ImplementationsDeployment
		case inspect.OpChainBundle:
			bundle = l1Contracts.OpChainDeployment
		default:
			return nil, fmt.Errorf("invalid contract bundle: %s", bundleName)
		}
	}

	return bundle, nil
}

func (v *Verifier) verifyContractBundle(ctx context.Context, workdir string, bundleName string) error {
	bundle, err := v.getContractBundle(workdir, bundleName)
	if err != nil {
		return fmt.Errorf("failed to retrieve bundle: %w", err)
	}

	// Use reflection to iterate over fields of the bundle.
	val := reflect.ValueOf(bundle)
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Type() == reflect.TypeOf(common.Address{}) {
			addr := field.Interface().(common.Address)
			if addr != (common.Address{}) { // Skip zero addresses
				contractName := typ.Field(i).Name
				if err := v.verifySingleContract(ctx, addr, contractName); err != nil {
					v.numFailed++
					v.log.Error("failed to verify contract", "name", contractName, "bundle", bundleName, "error", err)
				}
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
