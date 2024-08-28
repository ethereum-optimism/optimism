package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"golang.org/x/exp/maps"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/clients"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/safe"
	"github.com/ethereum-optimism/optimism/op-chain-ops/upgrades"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"

	"github.com/ethereum-optimism/superchain-registry/superchain"
)

var (
	TARGET_RELEASE = "op-contracts/v1.3.0"
)

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	oplog.SetGlobalLogHandler(log.NewTerminalHandler(os.Stderr, color))

	app := &cli.App{
		Name:  "op-upgrade",
		Usage: "Build transactions useful for upgrading the Superchain",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "l1-rpc-url",
				Value:   "http://127.0.0.1:8545",
				Usage:   "L1 RPC URL, the chain ID will be used to determine the superchain",
				EnvVars: []string{"L1_RPC_URL"},
			},
			&cli.Uint64SliceFlag{
				Name:  "chain-ids",
				Usage: "L2 Chain IDs corresponding to chains to upgrade. Corresponds to all chains if empty",
			},
			&cli.StringFlag{
				Name:    "superchain-target",
				Usage:   "The name of the superchain target to upgrade. For example: mainnet or sepolia.",
				EnvVars: []string{"SUPERCHAIN_TARGET"},
			},
			&cli.PathFlag{
				Name:    "deploy-config",
				Usage:   "The path to the deploy config file",
				EnvVars: []string{"DEPLOY_CONFIG"},
			},
			&cli.PathFlag{
				Name:    "outfile",
				Usage:   "The file to write the output to. If not specified, output is written to stdout",
				EnvVars: []string{"OUTFILE"},
			},
		},
		Action: entrypoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error op-upgrade", "err", err)
	}
}

// entrypoint contains the main logic of the script
func entrypoint(ctx *cli.Context) error {
	client, err := ethclient.Dial(ctx.String("l1-rpc-url"))
	if err != nil {
		return err
	}

	// Fetch the L1 chain ID to determine the superchain name
	l1ChainID, err := client.ChainID(ctx.Context)
	if err != nil {
		return err
	}

	superchainName := ctx.String("superchain-target")
	sc, ok := superchain.Superchains[superchainName]
	if !ok {
		return fmt.Errorf("superchain name %s not registered", superchainName)
	}

	declaredChainID := sc.Config.L1.ChainID

	if declaredChainID != l1ChainID.Uint64() {
		return fmt.Errorf("superchain %s has chainID %d, but the l1-rpc-url returned a chainId of %d",
			superchainName, declaredChainID, l1ChainID.Uint64())
	}

	chainIDs := ctx.Uint64Slice("chain-ids")
	if len(chainIDs) != 1 {
		// This requirement is due to the `SYSTEM_CONFIG_START_BLOCK` environment variable
		// that we read from in `op-chain-ops/upgrades/l1.go`
		panic("op-upgrade currently only supports upgrading a single chain at a time")
	}
	deployConfig := ctx.Path("deploy-config")

	// If no chain IDs are specified, upgrade all chains
	if len(chainIDs) == 0 {
		chainIDs = maps.Keys(superchain.OPChains)
	}
	slices.Sort(chainIDs)

	targets := make([]*superchain.ChainConfig, 0)
	for _, chainConfig := range superchain.OPChains {
		if chainConfig.Superchain == superchainName && slices.Contains(chainIDs, chainConfig.ChainID) {
			targets = append(targets, chainConfig)
		}
	}

	if len(targets) == 0 {
		return fmt.Errorf("no chains found for superchain target %s with chain IDs %v, are you sure this chain is in the superchain registry?", superchainName, chainIDs)
	}

	slices.SortFunc(targets, func(i, j *superchain.ChainConfig) int {
		return int(i.ChainID) - int(j.ChainID)
	})

	// Create a batch of transactions
	batch := safe.Batch{}

	for _, chainConfig := range targets {

		name, _ := toDeployConfigName(chainConfig)
		config, err := genesis.NewDeployConfigWithNetwork(name, deployConfig)
		if err != nil {
			log.Warn("Cannot find deploy config for network, so validity checks will be skipped", "name", chainConfig.Name, "deploy-config-name", name, "path", deployConfig, "err", err)
		}

		if config != nil {
			log.Info("Checking deploy config validity", "name", chainConfig.Name)
			if err := config.Check(); err != nil {
				return fmt.Errorf("error checking deploy config: %w", err)
			}
		}

		clients, err := clients.NewClients(ctx.String("l1-rpc-url"), chainConfig.PublicRPC)
		if err != nil {
			return fmt.Errorf("cannot create RPC clients: %w", err)
		}
		// The L1Client is required
		if clients.L1Client == nil {
			return errors.New("Cannot create L1 client")
		}

		l1ChainID, err := clients.L1Client.ChainID(ctx.Context)
		if err != nil {
			return fmt.Errorf("cannot fetch L1 chain ID: %w", err)
		}

		// The L2Client is not required, but double check the chain id matches if possible
		if clients.L2Client != nil {
			l2ChainID, err := clients.L2Client.ChainID(ctx.Context)
			if err != nil {
				return fmt.Errorf("cannot fetch L2 chain ID: %w", err)
			}
			if chainConfig.ChainID != l2ChainID.Uint64() {
				return fmt.Errorf("Mismatched chain IDs: %d != %d", chainConfig.ChainID, l2ChainID)
			}
		}

		log.Info(chainConfig.Name, "l1-chain-id", l1ChainID, "l2-chain-id", chainConfig.ChainID)

		log.Info("Detecting on chain contracts")
		// Tracking the individual addresses can be deprecated once the system is upgraded
		// to the new contracts where the system config has a reference to each address.
		addresses, ok := superchain.Addresses[chainConfig.ChainID]
		if !ok {
			return fmt.Errorf("no addresses for chain ID %d", chainConfig.ChainID)
		}
		versions, err := upgrades.GetContractVersions(ctx.Context, addresses, chainConfig, clients.L1Client)
		if err != nil {
			return fmt.Errorf("error getting contract versions: %w", err)
		}

		log.Info("L1CrossDomainMessenger", "version", versions.L1CrossDomainMessenger, "address", addresses.L1CrossDomainMessengerProxy)
		log.Info("L1ERC721Bridge", "version", versions.L1ERC721Bridge, "address", addresses.L1ERC721BridgeProxy)
		log.Info("L1StandardBridge", "version", versions.L1StandardBridge, "address", addresses.L1StandardBridgeProxy)
		log.Info("L2OutputOracle", "version", versions.L2OutputOracle, "address", addresses.L2OutputOracleProxy)
		log.Info("OptimismMintableERC20Factory", "version", versions.OptimismMintableERC20Factory, "address", addresses.OptimismMintableERC20FactoryProxy)
		log.Info("OptimismPortal", "version", versions.OptimismPortal, "address", addresses.OptimismPortalProxy)
		log.Info("SystemConfig", "version", versions.SystemConfig, "address", addresses.SystemConfigProxy)

		superchainTarget := superchain.OPChains[chainConfig.ChainID].Superchain
		if !ok {
			return fmt.Errorf("no implementations for chain ID %d", chainConfig.ChainID)
		}
		implementationAddresses, ok := superchain.Superchains[superchainTarget].Config.L1.ContractImplementations[TARGET_RELEASE]
		if !ok {
			return fmt.Errorf("implementation contract addresses missing in superchain registry for release: %s", TARGET_RELEASE)
		}
		contractVersions, ok := superchain.Superchains[superchainTarget].Config.L1.Versions[TARGET_RELEASE]
		if !ok {
			return fmt.Errorf("implementation contract versions missing in superchain registry for release: %s", TARGET_RELEASE)
		}

		// TODO This looks for the latest implementations defined for each contract, and for
		// OptimismPortal that's the FPAC v3.3.0. However we do not want to upgrade other chains to
		// that yet so we hardcode v2.5.0 which corresponds to the pre-FPAC op-contracts/v1.3.0 tag.
		// See comments in isAllowedChainID to learn more.
		// targetUpgrade := superchain.SuperchainSemver[superchainName]
		// targetUpgrade.OptimismPortal = "2.5.0"

		log.Info("Upgrading to the following versions")
		log.Info("L1CrossDomainMessenger", "version", contractVersions.L1CrossDomainMessenger, "address", implementationAddresses.L1CrossDomainMessenger)
		log.Info("L1ERC721Bridge", "version", contractVersions.L1ERC721Bridge, "address", implementationAddresses.L1ERC721Bridge)
		log.Info("L1StandardBridge", "version", contractVersions.L1StandardBridge, "address", implementationAddresses.L1StandardBridge)
		log.Info("L2OutputOracle", "version", contractVersions.L2OutputOracle, "address", implementationAddresses.L2OutputOracle)
		log.Info("OptimismMintableERC20Factory", "version", contractVersions.OptimismMintableERC20Factory, "address", implementationAddresses.OptimismMintableERC20Factory)
		log.Info("OptimismPortal", "version", contractVersions.OptimismPortal, "address", implementationAddresses.OptimismPortal)
		log.Info("SystemConfig", "version", contractVersions.SystemConfig, "address", implementationAddresses.SystemConfig)

		// Ensure that the superchain registry information is correct by checking the
		// actual versions based on what the registry says is true.
		if err := upgrades.CheckL1(ctx.Context, implementationAddresses, contractVersions, clients.L1Client); err != nil {
			return fmt.Errorf("error checking L1: %w", err)
		}

		// Build the batch
		// op-upgrade assumes a superchain config for L1 contract-implementations set.
		if err := upgrades.L1(&batch, implementationAddresses, *addresses, config, chainConfig, sc, clients.L1Client); err != nil {
			return err
		}
	}

	// Write the batch to disk or stdout
	if outfile := ctx.Path("outfile"); outfile != "" {
		if err := jsonutil.WriteJSON(outfile, batch, 0o666); err != nil {
			return err
		}
	} else {
		data, err := json.MarshalIndent(batch, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}

// toDeployConfigName is a temporary function that maps the chain config names
// to deploy config names. This should be able to be removed in the future
// with a canonical naming scheme. If an empty string is returned, then
// it means that the chain is not supported yet.
func toDeployConfigName(cfg *superchain.ChainConfig) (string, error) {
	if cfg.Name == "OP-Sepolia" {
		return "sepolia", nil
	}
	if cfg.Name == "OP-Goerli" {
		return "goerli", nil
	}
	if cfg.Name == "PGN" {
		return "pgn", nil
	}
	if cfg.Name == "Zora" {
		return "zora", nil
	}
	if cfg.Name == "OP-Mainnet" {
		return "mainnet", nil
	}
	if cfg.Name == "Zora Goerli" {
		return "zora-goerli", nil
	}
	return "", fmt.Errorf("unsupported chain name %s", cfg.Name)
}
