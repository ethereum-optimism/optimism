package main

import (
	"fmt"
	"os"

	"github.com/bobanetwork/boba/boba-bindings/bindgen"
	"github.com/bobanetwork/boba/boba-bindings/ethclient"
	"github.com/bobanetwork/boba/boba-bindings/etherscan"
	"github.com/urfave/cli/v2"

	"github.com/ledgerwatch/log/v3"
)

const (
	// Base Flags
	MetadataOutFlagName         = "metadata-out"
	BindingsPackageNameFlagName = "bindings-package"
	ContractsListFlagName       = "contracts-list"
	MonorepoBasePathFlagName    = "monorepo-base"

	// Local Contracts Flags
	SourceMapsListFlagName = "source-maps-list"
	ForgeArtifactsFlagName = "forge-artifacts"

	// Remote Contracts Flags
	EtherscanApiKeyEthFlagName = "etherscan.apikey.eth"
	EtherscanApiKeyOpFlagName  = "etherscan.apikey.op"
	RpcUrlEthFlagName          = "rpc.url.eth"
	RpcUrlOpFlagName           = "rpc.url.op"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat()))

	app := &cli.App{
		Name:  "BindGen",
		Usage: "Generate contract bindings using Foundry artifacts and/or remotely sourced contract data",
		Commands: []*cli.Command{
			{
				Name:  "generate",
				Usage: "Generate contract bindings",
				Flags: baseFlags(),
				Subcommands: []*cli.Command{
					{
						Name:   "all",
						Usage:  "Generate bindings for local and remote contracts",
						Flags:  append(localFlags(), remoteFlags()...),
						Action: generateBindings,
					},
					{
						Name:   "local",
						Usage:  "Generate bindings for locally sourced contracts",
						Flags:  localFlags(),
						Action: generateBindings,
					},
					{
						Name:   "remote",
						Usage:  "Generate bindings for remotely sourced contracts",
						Flags:  remoteFlags(),
						Action: generateBindings,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("BindGen error", "error", err.Error())
	}
}

func generateBindings(c *cli.Context) error {
	switch c.Command.Name {
	case "all":
		localBindingsGenerator, err := parseConfigLocal(c)
		if err != nil {
			return err
		}
		if err := localBindingsGenerator.GenerateBindings(); err != nil {
			return fmt.Errorf("error generating local bindings: %w", err)
		}

		remoteBindingsGenerator, err := parseConfigRemote(c)
		if err != nil {
			return err
		}
		if err := remoteBindingsGenerator.GenerateBindings(); err != nil {
			return fmt.Errorf("error generating remote bindings: %w", err)
		}

		return nil
	case "local":
		localBindingsGenerator, err := parseConfigLocal(c)
		if err != nil {
			return err
		}
		if err := localBindingsGenerator.GenerateBindings(); err != nil {
			return fmt.Errorf("error generating local bindings: %w", err)
		}
		return nil
	case "remote":
		remoteBindingsGenerator, err := parseConfigRemote(c)
		if err != nil {
			return err
		}
		if err := remoteBindingsGenerator.GenerateBindings(); err != nil {
			return fmt.Errorf("error generating remote bindings: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown command: %s", c.Command.Name)
	}
}

func parseConfigBase(c *cli.Context) (bindgen.BindGenGeneratorBase, error) {
	fmt.Println(c.String(MonorepoBasePathFlagName))
	return bindgen.BindGenGeneratorBase{
		MetadataOut:         c.String(MetadataOutFlagName),
		BindingsPackageName: c.String(BindingsPackageNameFlagName),
		MonorepoBasePath:    c.String(MonorepoBasePathFlagName),
		ContractsListPath:   c.String(ContractsListFlagName),
		Logger:              log.New(),
	}, nil
}

func parseConfigLocal(c *cli.Context) (bindgen.BindGenGeneratorLocal, error) {
	baseConfig, err := parseConfigBase(c)
	if err != nil {
		return bindgen.BindGenGeneratorLocal{}, err
	}
	return bindgen.BindGenGeneratorLocal{
		BindGenGeneratorBase: baseConfig,
		SourceMapsList:       c.String(SourceMapsListFlagName),
		ForgeArtifactsPath:   c.String(ForgeArtifactsFlagName),
	}, nil
}

func parseConfigRemote(c *cli.Context) (bindgen.BindGenGeneratorRemote, error) {
	baseConfig, err := parseConfigBase(c)
	if err != nil {
		return bindgen.BindGenGeneratorRemote{}, err
	}
	generator := bindgen.BindGenGeneratorRemote{
		BindGenGeneratorBase: baseConfig,
	}

	generator.ContractDataClients.Eth = etherscan.NewEthereumClient(c.String(EtherscanApiKeyEthFlagName))
	generator.ContractDataClients.Op = etherscan.NewOptimismClient(c.String(EtherscanApiKeyOpFlagName))

	if generator.RpcClients.Eth, err = ethclient.NewEthClient(c.String(RpcUrlEthFlagName)); err != nil {
		return bindgen.BindGenGeneratorRemote{}, fmt.Errorf("error initializing Ethereum client: %w", err)
	}
	if generator.RpcClients.Op, err = ethclient.NewEthClient(c.String(RpcUrlOpFlagName)); err != nil {
		return bindgen.BindGenGeneratorRemote{}, fmt.Errorf("error initializing Optimism client: %w", err)
	}
	return generator, nil
}

func baseFlags() []cli.Flag {
	baseFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     MonorepoBasePathFlagName,
			Usage:    "Path to monorepo base directory",
			Required: true,
		},
		&cli.StringFlag{
			Name:     MetadataOutFlagName,
			Usage:    "Output directory to put contract metadata files in",
			Required: true,
		},
		&cli.StringFlag{
			Name:     BindingsPackageNameFlagName,
			Usage:    "Go package name given to generated bindings",
			Required: true,
		},
		&cli.StringFlag{
			Name:     ContractsListFlagName,
			Usage:    "Path to file containing list of contract names to generate bindings for",
			Required: true,
		},
	}
	return baseFlags
}

func localFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  SourceMapsListFlagName,
			Usage: "Comma-separated list of contracts to generate source-maps for",
		},
		&cli.StringFlag{
			Name:     ForgeArtifactsFlagName,
			Usage:    "Path to forge-artifacts directory, containing compiled contract artifacts",
			Required: true,
		},
	}
}

func remoteFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     EtherscanApiKeyEthFlagName,
			Usage:    "API key to make queries to Etherscan for Ethereum",
			Required: true,
		},
		&cli.StringFlag{
			Name:     EtherscanApiKeyOpFlagName,
			Usage:    "API key to make queries to Etherscan for Optimism",
			Required: true,
		},
		&cli.StringFlag{
			Name:     RpcUrlEthFlagName,
			Usage:    "RPC URL (with API key if required) to query Ethereum",
			Required: true,
		},
		&cli.StringFlag{
			Name:     RpcUrlOpFlagName,
			Usage:    "RPC URL (with API key if required) to query Optimism",
			Required: true,
		},
	}
}
