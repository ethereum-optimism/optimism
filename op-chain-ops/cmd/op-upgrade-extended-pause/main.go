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

	"github.com/ethereum-optimism/superchain-registry/superchain"
)

// deployments contains the L1 addresses of the contracts that are being upgraded to.
// Note that the key is the L2 chain id. This is because the L1 contracts must be specific
// for a particular OP Stack chain and cannot currently be used by multiple chains.
var deployments = map[uint64]superchain.ImplementationList{
	// Base Sepolia
	84532: {
		L1CrossDomainMessenger: superchain.VersionedContract{
			Version: "2.1.1",
			Address: superchain.HexToAddress("0x442d5c024a80c34d64fed048bdc7c50dd84665c4"),
		},
		L1ERC721Bridge: superchain.VersionedContract{
			Version: "2.0.0",
			Address: superchain.HexToAddress("0x30e2c20c73353b8ddb6021d5636aef1b91727077"),
		},
		L1StandardBridge: superchain.VersionedContract{
			Version: "2.0.0",
			Address: superchain.HexToAddress("0xf71db0a6955b3edc78a267cd6441feed4ee0197b"),
		},
		OptimismPortal: superchain.VersionedContract{
			Version: "2.1.0",
			Address: superchain.HexToAddress("0x770d02b87e081e61ab30713b0ece6dfade792aff"),
		},
		SystemConfig: superchain.VersionedContract{
			Version: "1.11.0",
			Address: superchain.HexToAddress("0xf55b3dbb3bd2f2fa9236b0be6e8b9e91b819fd14"),
		},
		L2OutputOracle: superchain.VersionedContract{
			Version: "1.7.0",
			Address: superchain.HexToAddress("0x1187d73b0580f607e1b9c03698238fcad483e776"),
		},
		OptimismMintableERC20Factory: superchain.VersionedContract{
			Version: "1.8.0",
			Address: superchain.HexToAddress("0x6B047052dc3DafbA003e2fA4fEEe2e883dd5575B"),
		},
	},
	// OP Sepolia
	11155420: {
		L1CrossDomainMessenger: superchain.VersionedContract{
			Version: "2.1.1",
			Address: superchain.HexToAddress("0xc3c7e6f4ad6a593a9731a39fa883ec1999d7d873"),
		},
		L1ERC721Bridge: superchain.VersionedContract{
			Version: "2.0.0",
			Address: superchain.HexToAddress("0x532cad52e1f812eeb9c9a9571e07fef55993fefa"),
		},
		L1StandardBridge: superchain.VersionedContract{
			Version: "2.0.0",
			Address: superchain.HexToAddress("0xe19c7a2c0bb32287731ea75da9b1c836815964f1"),
		},
		OptimismPortal: superchain.VersionedContract{
			Version: "2.1.0",
			Address: superchain.HexToAddress("0x592B7D3255a8037307d23C16cC8c13a9563c8Ab1"),
		},
		SystemConfig: superchain.VersionedContract{
			Version: "1.11.0",
			Address: superchain.HexToAddress("0xce77d580e0befbb1561376a722217017651b9dbf"),
		},
		L2OutputOracle: superchain.VersionedContract{
			Version: "1.7.0",
			Address: superchain.HexToAddress("0x83aEb8B156cD90E64C702781C84A681DADb1DDe2"),
		},
		OptimismMintableERC20Factory: superchain.VersionedContract{
			Version: "1.8.0",
			Address: superchain.HexToAddress("0xd7e63ec8ec03803236be93642a610641dee51e62"),
		},
	},
	// Zora Sepolia
	999999999: {
		L1CrossDomainMessenger: superchain.VersionedContract{
			Version: "2.1.1",
			Address: superchain.HexToAddress("0xb74e6f01cddfc53cd48fb94e14137a0801a67ee4"),
		},
		L1ERC721Bridge: superchain.VersionedContract{
			Version: "2.0.0",
			Address: superchain.HexToAddress("0x5ff51b220049151710752ebe65d0a060020f6018"),
		},
		L1StandardBridge: superchain.VersionedContract{
			Version: "2.0.0",
			Address: superchain.HexToAddress("0xf8e25ec7ca94a960a9392c56c55b68414f5c7ded"),
		},
		OptimismPortal: superchain.VersionedContract{
			Version: "2.1.0",
			Address: superchain.HexToAddress("0xd2b5f6dfa6fdfd89327a5aa4c787a89456ef0ca8"),
		},
		SystemConfig: superchain.VersionedContract{
			Version: "1.11.0",
			Address: superchain.HexToAddress("0xaeb5f8ed2977e70f4ddacf2f603c0dcf8e561873"),
		},
		L2OutputOracle: superchain.VersionedContract{
			Version: "1.7.0",
			Address: superchain.HexToAddress("0x1d5a9755983fa8520bb0fc5caf7904fac77ede76"),
		},
		OptimismMintableERC20Factory: superchain.VersionedContract{
			Version: "1.8.0",
			Address: superchain.HexToAddress("0xc1fa0ca70cd4f392883d2abe00d3971230382996"),
		},
	},
	// PGN Sepolia
	58008: {
		L1CrossDomainMessenger: superchain.VersionedContract{
			Version: "2.1.1",
			Address: superchain.HexToAddress("0x99bb19a985e1def20d363405c5943d10e715dc12"),
		},
		L1ERC721Bridge: superchain.VersionedContract{
			Version: "2.0.0",
			Address: superchain.HexToAddress("0x89eba5aeb024534e6e1575c6bdb0f4f70d32f7da"),
		},
		L1StandardBridge: superchain.VersionedContract{
			Version: "2.0.0",
			Address: superchain.HexToAddress("0x9cde10006cac4423505864c904e2cfcf124dcaee"),
		},
		OptimismPortal: superchain.VersionedContract{
			Version: "2.1.0",
			Address: superchain.HexToAddress("0x725da050f385e52c0ae700e8c433c3636aba4592"),
		},
		SystemConfig: superchain.VersionedContract{
			Version: "1.11.0",
			Address: superchain.HexToAddress("0xd1557adfee8eda61619fc227c3dbb41fc16fc840"),
		},
		L2OutputOracle: superchain.VersionedContract{
			Version: "1.7.0",
			Address: superchain.HexToAddress("0xfae8e4695a0c96ea7ce20e1ed8d401604964315a"),
		},
		OptimismMintableERC20Factory: superchain.VersionedContract{
			Version: "1.8.0",
			Address: superchain.HexToAddress("0x8b55bf68569a9561a60d48419453ee570f87f7f0"),
		},
	},
}

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

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
