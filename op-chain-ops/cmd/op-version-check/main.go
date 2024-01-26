package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"

	"github.com/ethereum-optimism/optimism/op-chain-ops/upgrades"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
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
			&cli.StringSliceFlag{
				Name:    "l1-rpc-urls",
				Usage:   "L1 RPC URLs, the chain ID will be used to determine the superchain",
				EnvVars: []string{"L1_RPC_URLS"},
			},
			&cli.StringSliceFlag{
				Name:    "l2-rpc-urls",
				Usage:   "L2 RPC URLs, corresponding to chains to check versions for. Corresponds to all chains if empty",
				EnvVars: []string{"L2_RPC_URLS"},
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
	l1RPCURLs := ctx.StringSlice("l1-rpc-urls")
	l2RPCURLs := ctx.StringSlice("l2-rpc-urls")

	var l2ChainIDs []uint64

	// If no L2 RPC URLs are specified, we check all chains for the L1 RPC URL
	if len(l2RPCURLs) == 0 {
		l2ChainIDs = maps.Keys(superchain.OPChains)
	} else {
		for _, l2RPCURL := range l2RPCURLs {
			client, err := ethclient.Dial(l2RPCURL)
			if err != nil {
				return errors.New("cannot create L2 client")
			}

			l2ChainID, err := client.ChainID(ctx.Context)
			if err != nil {
				return fmt.Errorf("cannot fetch L2 chain ID: %w", err)
			}

			l2ChainIDs = append(l2ChainIDs, l2ChainID.Uint64())
		}
	}

	output := []ChainVersionCheck{}

	for _, l2ChainID := range l2ChainIDs {
		chainConfig := superchain.OPChains[l2ChainID]

		if chainConfig.ChainID != l2ChainID {
			return fmt.Errorf("mismatched chain IDs: %d != %d", chainConfig.ChainID, l2ChainID)
		}

		for _, l1RPCURL := range l1RPCURLs {
			client, err := ethclient.Dial(l1RPCURL)
			if err != nil {
				return errors.New("cannot create L1 client")
			}

			l1ChainID, err := client.ChainID(ctx.Context)
			if err != nil {
				return fmt.Errorf("cannot fetch L1 chain ID: %w", err)
			}

			declaredL1ChainID, err := upgrades.SuperChainID((chainConfig.Superchain))
			if err != nil {
				return err
			}

			if l1ChainID.Uint64() != declaredL1ChainID {
				// L2 corresponds to a different superchain than L1, skip
				log.Info("Ignoring L1/L2", "l1-chain-id", l1ChainID, "l2-chain-id", l2ChainID)
				continue
			}

			log.Info(chainConfig.Name, "l1-chain-id", l1ChainID, "l2-chain-id", l2ChainID)

			log.Info("Detecting on chain contracts")
			// Tracking the individual addresses can be deprecated once the system is upgraded
			// to the new contracts where the system config has a reference to each address.
			addresses, ok := superchain.Addresses[l2ChainID]
			if !ok {
				return fmt.Errorf("no addresses for chain ID %d", l2ChainID)
			}
			versions, err := upgrades.GetContractVersions(ctx.Context, addresses, chainConfig, client)
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

			output = append(output, ChainVersionCheck{Name: chainConfig.Name, ChainID: l2ChainID, Contracts: contracts})

			log.Info("Successfully processed contract versions", "chain", chainConfig.Name, "l1-chain-id", l1ChainID, "l2-chain-id", l2ChainID)
			break
		}
	}
	// Write contract versions to disk or stdout
	if outfile := ctx.Path("outfile"); outfile != "" {
		if err := jsonutil.WriteJSON(outfile, output); err != nil {
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
