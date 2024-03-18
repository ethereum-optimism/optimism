package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/clients"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/safe"
	"github.com/ethereum-optimism/optimism/op-chain-ops/upgrades"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"

	"github.com/ethereum-optimism/superchain-registry/superchain"
)

// deployments contains the L1 addresses of the contracts that are being upgraded to.
// Note that the key is the L2 chain id. This is because the L1 contracts must be specific
// for a particular OP Stack chain and cannot currently be used by multiple chains.
var deployments = map[uint64]superchain.ImplementationList{
	// OP Mainnet
	10: {
		L1CrossDomainMessenger: superchain.VersionedContract{
			Version: "2.3.0",
			Address: superchain.HexToAddress("0xD3494713A5cfaD3F5359379DfA074E2Ac8C6Fd65"),
		},
		L1ERC721Bridge: superchain.VersionedContract{
			Version: "2.1.0",
			Address: superchain.HexToAddress("0xAE2AF01232a6c4a4d3012C5eC5b1b35059caF10d"),
		},
		L1StandardBridge: superchain.VersionedContract{
			Version: "2.1.0",
			Address: superchain.HexToAddress("0x64B5a5Ed26DCb17370Ff4d33a8D503f0fbD06CfF"),
		},
		OptimismPortal: superchain.VersionedContract{
			Version: "2.5.0",
			Address: superchain.HexToAddress("0x2D778797049FE9259d947D1ED8e5442226dFB589"),
		},
		SystemConfig: superchain.VersionedContract{
			Version: "1.12.0",
			Address: superchain.HexToAddress("0xba2492e52F45651B60B8B38d4Ea5E2390C64Ffb1"),
		},
		L2OutputOracle: superchain.VersionedContract{
			Version: "1.8.0",
			Address: superchain.HexToAddress("0xF243BEd163251380e78068d317ae10f26042B292"),
		},
		OptimismMintableERC20Factory: superchain.VersionedContract{
			Version: "1.9.0",
			Address: superchain.HexToAddress("0xE01efbeb1089D1d1dB9c6c8b135C934C0734c846"),
		},
	},
	// OP Sepolia
	11155420: {
		L1CrossDomainMessenger: superchain.VersionedContract{
			Version: "2.3.0",
			Address: superchain.HexToAddress("0xD3494713A5cfaD3F5359379DfA074E2Ac8C6Fd65"),
		},
		L1ERC721Bridge: superchain.VersionedContract{
			Version: "2.1.0",
			Address: superchain.HexToAddress("0xAE2AF01232a6c4a4d3012C5eC5b1b35059caF10d"),
		},
		L1StandardBridge: superchain.VersionedContract{
			Version: "2.1.0",
			Address: superchain.HexToAddress("0x64B5a5Ed26DCb17370Ff4d33a8D503f0fbD06CfF"),
		},
		OptimismPortal: superchain.VersionedContract{
			Version: "2.5.0",
			Address: superchain.HexToAddress("0x2D778797049FE9259d947D1ED8e5442226dFB589"),
		},
		SystemConfig: superchain.VersionedContract{
			Version: "1.12.0",
			Address: superchain.HexToAddress("0xba2492e52F45651B60B8B38d4Ea5E2390C64Ffb1"),
		},
		L2OutputOracle: superchain.VersionedContract{
			Version: "1.8.0",
			Address: superchain.HexToAddress("0xF243BEd163251380e78068d317ae10f26042B292"),
		},
		OptimismMintableERC20Factory: superchain.VersionedContract{
			Version: "1.9.0",
			Address: superchain.HexToAddress("0xE01efbeb1089D1d1dB9c6c8b135C934C0734c846"),
		},
	},
}

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	oplog.SetGlobalLogHandler(log.NewTerminalHandler(os.Stderr, color))

	app := &cli.App{
		Name:  "op-upgrade",
		Usage: "Build transactions useful for upgrading the Superchain",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "l1-rpc-url",
				Value:    "http://127.0.0.1:8545",
				Usage:    "L1 RPC URL, the chain ID will be used to determine the superchain",
				Required: true,
				EnvVars:  []string{"L1_RPC_URL"},
			},
			&cli.StringFlag{
				Name:     "l2-rpc-url",
				Value:    "http://127.0.0.1:9545",
				Usage:    "L2 RPC URL",
				Required: true,
				EnvVars:  []string{"L2_RPC_URL"},
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
	config, err := genesis.NewDeployConfig(ctx.Path("deploy-config"))
	if err != nil {
		return err
	}
	if err := config.Check(); err != nil {
		return fmt.Errorf("error checking deploy config: %w", err)
	}

	clients, err := clients.NewClients(ctx.String("l1-rpc-url"), ctx.String("l2-rpc-url"))
	if err != nil {
		return fmt.Errorf("cannot create RPC clients: %w", err)
	}
	if clients.L1Client == nil {
		return errors.New("Cannot create L1 client")
	}
	if clients.L2Client == nil {
		return errors.New("Cannot create L2 client")
	}

	l1ChainID, err := clients.L1Client.ChainID(ctx.Context)
	if err != nil {
		return fmt.Errorf("cannot fetch L1 chain ID: %w", err)
	}
	l2ChainID, err := clients.L2Client.ChainID(ctx.Context)
	if err != nil {
		return fmt.Errorf("cannot fetch L2 chain ID: %w", err)
	}

	log.Info("connected to chains", "l1-chain-id", l1ChainID, "l2-chain-id", l2ChainID)

	// Create a batch of transactions
	batch := safe.Batch{}

	list, ok := deployments[l2ChainID.Uint64()]
	if !ok {
		return fmt.Errorf("no implementations for chain ID %d", l2ChainID)
	}

	proxyAddresses, ok := superchain.Addresses[l2ChainID.Uint64()]
	if !ok {
		return fmt.Errorf("no proxy addresses for chain ID %d", l2ChainID)
	}

	chainConfig, ok := superchain.OPChains[l2ChainID.Uint64()]
	if !ok {
		return fmt.Errorf("no chain config for chain ID %d", l2ChainID)
	}

	log.Info("Upgrading to the following versions")
	log.Info("L1CrossDomainMessenger", "version", list.L1CrossDomainMessenger.Version, "address", list.L1CrossDomainMessenger.Address)
	log.Info("L1ERC721Bridge", "version", list.L1ERC721Bridge.Version, "address", list.L1ERC721Bridge.Address)
	log.Info("L1StandardBridge", "version", list.L1StandardBridge.Version, "address", list.L1StandardBridge.Address)
	log.Info("L2OutputOracle", "version", list.L2OutputOracle.Version, "address", list.L2OutputOracle.Address)
	log.Info("OptimismMintableERC20Factory", "version", list.OptimismMintableERC20Factory.Version, "address", list.OptimismMintableERC20Factory.Address)
	log.Info("OptimismPortal", "version", list.OptimismPortal.Version, "address", list.OptimismPortal.Address)
	log.Info("SystemConfig", "version", list.SystemConfig.Version, "address", list.SystemConfig.Address)

	if err := upgrades.CheckL1(ctx.Context, &list, clients.L1Client); err != nil {
		return fmt.Errorf("error checking L1 contracts: %w", err)
	}

	// Build the batch
	if err := upgrades.L1(&batch, list, *proxyAddresses, config, chainConfig, clients.L1Client); err != nil {
		return fmt.Errorf("cannot build L1 upgrade batch: %w", err)
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
