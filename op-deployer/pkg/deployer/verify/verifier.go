package verify

import (
	"context"
	"fmt"
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
	bundleName := cliCtx.String(deployer.ContractBundleFlagName)
	contractName := cliCtx.String(deployer.ContractNameFlagName)
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

	if bundleName == "" && contractName == "" {
		if err := v.verifyAll(ctx); err != nil {
			return err
		}
	} else if bundleName != "" && contractName == "" {
		if err := v.verifyContractBundle(bundleName); err != nil {
			return err
		}
	} else if bundleName != "" && contractName != "" {
		if err := v.verifySingleContract(ctx, contractName, bundleName); err != nil {
			return err
		}
	} else {
		// If a contract name is provided without a contract bundle, report an error.
		return fmt.Errorf("contract-name flag provided without contract-bundle flag")
	}
	v.log.Info("--- SUCCESS ---")
	return nil
}

func (v *Verifier) verifyAll(ctx context.Context) error {
	for _, bundleName := range inspect.ContractBundles {
		if err := v.verifyContractBundle(bundleName); err != nil {
			return fmt.Errorf("failed to verify bundle %s: %w", bundleName, err)
		}
	}
	return nil
}

func (v *Verifier) verifyContractBundle(bundleName string) error {
	// Retrieve the L1 contracts from state.
	l1Contracts, err := inspect.L1(v.st, v.l2ChainID)
	if err != nil {
		return fmt.Errorf("failed to extract L1 contracts from state: %w", err)
	}

	// Select the appropriate bundle based on the input bundleName.
	var bundle interface{}
	switch bundleName {
	case inspect.SuperchainBundle:
		bundle = l1Contracts.SuperchainDeployment
	case inspect.ImplementationsBundle:
		bundle = l1Contracts.ImplementationsDeployment
	case inspect.OpChainBundle:
		bundle = l1Contracts.OpChainDeployment
	default:
		return fmt.Errorf("invalid contract bundle: %s", bundleName)
	}

	// Use reflection to iterate over fields of the bundle.
	val := reflect.ValueOf(bundle)
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Type() == reflect.TypeOf(common.Address{}) {
			addr := field.Interface().(common.Address)
			if addr != (common.Address{}) { // Skip zero addresses
				name := typ.Field(i).Name
				if err := v.verifySingleContract(context.Background(), name, bundleName); err != nil {
					v.log.Error("failed to verify contract", "name", name, "bundle", bundleName, "error", err)
				}
			}
		}
	}
	return nil
}

func (v *Verifier) verifySingleContract(ctx context.Context, contractName string, bundleName string) error {
	l1Contracts, err := inspect.L1(v.st, v.l2ChainID)
	if err != nil {
		return fmt.Errorf("failed to extract L1 contracts from state: %w", err)
	}

	v.log.Info("Looking up contract address", "name", contractName, "bundle", bundleName)
	addr, err := l1Contracts.GetContractAddress(contractName, bundleName)
	if err != nil {
		return fmt.Errorf("failed to find address for contract %s: %w", contractName, err)
	}

	if err := v.verifyContract(addr, contractName); err != nil {
		v.numFailed++
		return err
	}

	return nil
}

func (v *Verifier) verifyContract(address common.Address, contractName string) error {
	verified, err := v.etherscan.isVerified(address)
	if err != nil {
		return fmt.Errorf("failed to check verification status: %w", err)
	}
	if verified {
		v.log.Info("Contract is already verified", "name", contractName, "address", address.Hex())
		v.numSkipped++
		return nil
	}

	v.log.Info("Formatting etherscan verification request", "name", contractName, "address", address.Hex())
	artifact, err := v.getContractArtifact(contractName)
	if err != nil {
		return fmt.Errorf("failed to get contract source: %w", err)
	}

	constructorArgs, err := v.getConstructorArgs(context.Background(), address, contractName)
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
