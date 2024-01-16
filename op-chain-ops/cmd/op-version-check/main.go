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
	"github.com/ethereum-optimism/optimism/op-chain-ops/upgrades"

	op_node_genesis "github.com/ethereum-optimism/optimism/op-node/cmd/genesis"
	"github.com/ethereum-optimism/superchain-registry/superchain"
)

type Contract struct {
	Version string             `yaml:"version"`
	Address superchain.Address `yaml:"address"`
}

type ChainVersionCheck struct {
	Name      string              `yaml:"name"`
	ChainID   uint64              `yaml:"chain_id"`
	Contracts map[string]Contract `yaml:"contracts"`
}

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "op-version-check",
		Usage: "Determine which contract versions are deployed for chains in a superchain",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "l1-rpc-url",
				Value:   "http://127.0.0.1:8545",
				Usage:   "L1 RPC URL, the chain ID will be used to determine the superchain",
				EnvVars: []string{"L1_RPC_URL"},
			},
			&cli.Uint64SliceFlag{
				Name:  "chain-ids",
				Usage: "L2 Chain IDs corresponding to chains to check versions for. Corresponds to all chains if empty",
			},
			&cli.StringFlag{
				Name:    "superchain-target",
				Usage:   "The name of the superchain",
				EnvVars: []string{"SUPERCHAIN_TARGET"},
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
		log.Crit("error op-version-check", "err", err)
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
		superchainName, err = upgrades.ToSuperchainName(l1ChainID.Uint64())
		if err != nil {
			return err
		}
	}

	chainIDs := ctx.Uint64Slice("chain-ids")

	// If no chain IDs are specified, check all chains
	if len(chainIDs) == 0 {
		chainIDs = maps.Keys(superchain.OPChains)
	}
	slices.Sort(chainIDs)

	targets := make([]*superchain.ChainConfig, 0)
	// TODO: Need some logging here if a chain ID is filtered out for whatever reason.
	for _, chainConfig := range superchain.OPChains {
		if chainConfig.Superchain == superchainName && slices.Contains(chainIDs, chainConfig.ChainID) {
			targets = append(targets, chainConfig)
		}
	}

	slices.SortFunc(targets, func(i, j *superchain.ChainConfig) int {
		return int(i.ChainID) - int(j.ChainID)
	})

	output := []ChainVersionCheck{}

	for _, chainConfig := range targets {
		clients, err := clients.NewClients(ctx.String("l1-rpc-url"), chainConfig.PublicRPC)
		if err != nil {
			return fmt.Errorf("cannot create RPC clients: %w", err)
		}
		// The L1Client is required
		if clients.L1Client == nil {
			return errors.New("cannot create L1 client")
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
				return fmt.Errorf("mismatched chain IDs: %d != %d", chainConfig.ChainID, l2ChainID)
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

		contracts := make(map[string]Contract)

		contracts["AddressManager"] = Contract{Version: "null", Address: addresses.AddressManager}
		contracts["L1CrossDomainMessenger"] = Contract{Version: versions.L1CrossDomainMessenger, Address: addresses.L1CrossDomainMessengerProxy}
		contracts["L1ERC721Bridge"] = Contract{Version: versions.L1ERC721Bridge, Address: addresses.L1ERC721BridgeProxy}
		contracts["L1StandardBridge"] = Contract{Version: versions.L1ERC721Bridge, Address: addresses.L1StandardBridgeProxy}
		contracts["L2OutputOracle"] = Contract{Version: versions.L2OutputOracle, Address: addresses.L2OutputOracleProxy}
		contracts["OptimismMintableERC20Factory"] = Contract{Version: versions.OptimismMintableERC20Factory, Address: addresses.OptimismMintableERC20FactoryProxy}
		contracts["OptimismPortal"] = Contract{Version: versions.OptimismPortal, Address: addresses.OptimismPortalProxy}
		contracts["SystemConfig"] = Contract{Version: versions.SystemConfig, Address: chainConfig.SystemConfigAddr}
		contracts["ProxyAdmin"] = Contract{Version: "null", Address: addresses.ProxyAdmin}

		output = append(output, ChainVersionCheck{Name: chainConfig.Name, ChainID: chainConfig.ChainID, Contracts: contracts})
	}

	// Write contract versions to disk or stdout
	if outfile := ctx.Path("outfile"); outfile != "" {
		if err := op_node_genesis.WriteJSONFile(outfile, output); err != nil {
			return err
		}
	} else {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}
