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

	"github.com/ethereum-optimism/superchain-registry/superchain"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

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
				Usage:   "The name of the superchain to upgrade",
				EnvVars: []string{"SUPERCHAIN_TARGET"},
			},
			&cli.PathFlag{
				Name:     "deploy-config",
				Usage:    "The path to the deploy config file",
				Required: true,
				EnvVars:  []string{"DEPLOY_CONFIG"},
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
	if superchainName == "" {
		superchainName, err = toSuperchainName(l1ChainID.Uint64())
		if err != nil {
			return err
		}
	}

	chainIDs := ctx.Uint64Slice("chain-ids")
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

	slices.SortFunc(targets, func(i, j *superchain.ChainConfig) int {
		return int(i.ChainID) - int(j.ChainID)
	})

	// Create a batch of transactions
	batch := safe.Batch{}

	for _, chainConfig := range targets {
		name, _ := toDeployConfigName(chainConfig)
		config, err := genesis.NewDeployConfigWithNetwork(name, deployConfig)
		if err != nil {
			log.Warn("Cannot find deploy config for network", "name", chainConfig.Name, "deploy-config-name", name, "path", deployConfig, "err", err)
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
		log.Info("SystemConfig", "version", versions.SystemConfig, "address", chainConfig.SystemConfigAddr)

		implementations, ok := superchain.Implementations[l1ChainID.Uint64()]
		if !ok {
			return fmt.Errorf("no implementations for chain ID %d", l1ChainID.Uint64())
		}

		list, err := implementations.Resolve(superchain.SuperchainSemver)
		if err != nil {
			return err
		}

		log.Info("Upgrading to the following versions")
		log.Info("L1CrossDomainMessenger", "version", list.L1CrossDomainMessenger.Version, "address", list.L1CrossDomainMessenger.Address)
		log.Info("L1ERC721Bridge", "version", list.L1ERC721Bridge.Version, "address", list.L1ERC721Bridge.Address)
		log.Info("L1StandardBridge", "version", list.L1StandardBridge.Version, "address", list.L1StandardBridge.Address)
		log.Info("L2OutputOracle", "version", list.L2OutputOracle.Version, "address", list.L2OutputOracle.Address)
		log.Info("OptimismMintableERC20Factory", "version", list.OptimismMintableERC20Factory.Version, "address", list.OptimismMintableERC20Factory.Address)
		log.Info("OptimismPortal", "version", list.OptimismPortal.Version, "address", list.OptimismPortal.Address)
		log.Info("SystemConfig", "version", list.SystemConfig.Version, "address", list.SystemConfig.Address)

		// Ensure that the superchain registry information is correct by checking the
		// actual versions based on what the registry says is true.
		if err := upgrades.CheckL1(ctx.Context, &list, clients.L1Client); err != nil {
			return fmt.Errorf("error checking L1: %w", err)
		}

		// Build the batch
		if err := upgrades.L1(&batch, list, *addresses, config, chainConfig, clients.L1Client); err != nil {
			return err
		}
	}

	// Write the batch to disk or stdout
	if outfile := ctx.Path("outfile"); outfile != "" {
		if err := writeJSON(outfile, batch); err != nil {
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

// toSuperchainName turns a base layer chain id into a superchain
// network name.
func toSuperchainName(chainID uint64) (string, error) {
	if chainID == 1 {
		return "mainnet", nil
	}
	if chainID == 5 {
		return "goerli", nil
	}
	if chainID == 11155111 {
		return "sepolia", nil
	}
	return "", fmt.Errorf("unsupported chain ID %d", chainID)
}

func writeJSON(outfile string, input interface{}) error {
	f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(input)
}
