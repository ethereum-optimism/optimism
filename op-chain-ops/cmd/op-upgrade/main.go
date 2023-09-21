package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"

	"github.com/ethereum-optimism/optimism/op-chain-ops/clients"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/safe"
	"github.com/ethereum-optimism/optimism/op-chain-ops/upgrades"

	"github.com/ethereum-optimism/superchain-registry/superchain"

	"github.com/urfave/cli/v2"
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
				Usage:   "L1 RPC URL",
				EnvVars: []string{"L1_RPC_URL"},
			},
			&cli.StringFlag{
				Name:    "l2-rpc-url",
				Value:   "http://127.0.0.1:9545",
				Usage:   "L2 RPC URL",
				EnvVars: []string{"L2_RPC_URL"},
			},
			&cli.PathFlag{
				Name:     "deploy-config",
				Required: true,
				EnvVars:  []string{"DEPLOY_CONFIG"},
			},
			&cli.PathFlag{
				Name:    "outfile",
				Usage:   "",
				EnvVars: []string{"OUTFILE"},
			},
		},
		Action: entrypoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error op-upgrade", "err", err)
	}
}

func entrypoint(ctx *cli.Context) error {
	config, err := genesis.NewDeployConfig(ctx.Path("deploy-config"))
	if err != nil {
		return err
	}

	clients, err := clients.NewClients(ctx)
	if err != nil {
		return err
	}

	l1ChainID, err := clients.L1Client.ChainID(ctx.Context)
	if err != nil {
		return err
	}
	l2ChainID, err := clients.L2Client.ChainID(ctx.Context)
	if err != nil {
		return err
	}
	log.Info("Chain IDs", "l1", l1ChainID, "l2", l2ChainID)

	chainConfig, ok := superchain.OPChains[l2ChainID.Uint64()]
	if !ok {
		return fmt.Errorf("no chain config for chain ID %d", l2ChainID.Uint64())
	}

	// Tracking the individual addresses can be deprecated once the system is upgraded
	// to the new contracts where the system config has a reference to each address.
	addresses, ok := superchain.Addresses[l2ChainID.Uint64()]
	if !ok {
		return fmt.Errorf("no addresses for chain ID %d", l2ChainID.Uint64())
	}
	versions, err := upgrades.GetContractVersions(ctx.Context, addresses, chainConfig, clients.L1Client)
	if err != nil {
		return fmt.Errorf("error getting contract versions: %w", err)
	}

	log.Info(
		"Current Versions",
		"L1CrossDomainMessenger", versions.L1CrossDomainMessenger,
		"L1ERC721Bridge", versions.L1ERC721Bridge,
		"L1StandardBridge", versions.L1StandardBridge,
		"L2OutputOracle", versions.L2OutputOracle,
		"OptimismMintableERC20Factory", versions.OptimismMintableERC20Factory,
		"OptimismPortal", versions.OptimismPortal,
		"SystemConfig", versions.SystemConfig,
	)

	implementations, ok := superchain.Implementations[l1ChainID.Uint64()]
	if !ok {
		return fmt.Errorf("no implementations for chain ID %d", l1ChainID.Uint64())
	}

	list, err := implementations.Resolve(superchain.SuperchainSemver)
	if err != nil {
		return err
	}
	if err := upgrades.CheckL1(ctx.Context, &list, clients.L1Client); err != nil {
		return fmt.Errorf("error checking L1: %w", err)
	}

	batch := safe.Batch{}
	if err := upgrades.L1(&batch, list, *addresses, config, chainConfig); err != nil {
		return err
	}

	if outfile := ctx.Path("outfile"); outfile != "" {
		if err := writeJSON(outfile, batch); err != nil {
			return err
		}
	} else {
		data, err := json.MarshalIndent(batch, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(data)
	}

	return nil
}

func writeJSON(outfile string, input interface{}) error {
	f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(input)
}
