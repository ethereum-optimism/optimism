package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-bindings/bindgen"
	"github.com/ethereum-optimism/optimism/op-bindings/etherscan"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const (
	// Base Flags
	MetadataOutFlagName         = "metadata-out"
	BindingsPackageNameFlagName = "bindings-package"
	ContractsListFlagName       = "contracts-list"

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
	oplog.SetupDefaults()

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

func setupLogger(c *cli.Context) log.Logger {
	logger := oplog.NewLogger(oplog.AppOut(c), oplog.ReadCLIConfig(c))
	oplog.SetGlobalLogHandler(logger.GetHandler())
	return logger
}

func generateBindings(c *cli.Context) error {
	logger := setupLogger(c)

	switch c.Command.Name {
	case "all":
		localBindingsGenerator, err := parseConfigLocal(logger, c)
		if err != nil {
			return err
		}
		if err := localBindingsGenerator.GenerateBindings(); err != nil {
			return fmt.Errorf("error generating local bindings: %w", err)
		}

		remoteBindingsGenerator, err := parseConfigRemote(logger, c)
		if err != nil {
			return err
		}
		if err := remoteBindingsGenerator.GenerateBindings(); err != nil {
			return fmt.Errorf("error generating remote bindings: %w", err)
		}

		return nil
	case "local":
		localBindingsGenerator, err := parseConfigLocal(logger, c)
		if err != nil {
			return err
		}
		if err := localBindingsGenerator.GenerateBindings(); err != nil {
			return fmt.Errorf("error generating local bindings: %w", err)
		}
		return nil
	case "remote":
		remoteBindingsGenerator, err := parseConfigRemote(logger, c)
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

func parseConfigBase(logger log.Logger, c *cli.Context) (bindgen.BindGenGeneratorBase, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return bindgen.BindGenGeneratorBase{}, err
	}

	monoRepoPath, err := op_service.FindMonorepoRoot(cwd)
	if err != nil {
		return bindgen.BindGenGeneratorBase{}, err
	}

	return bindgen.BindGenGeneratorBase{
		MetadataOut:         c.String(MetadataOutFlagName),
		BindingsPackageName: c.String(BindingsPackageNameFlagName),
		MonorepoBasePath:    monoRepoPath,
		ContractsListPath:   c.String(ContractsListFlagName),
		Logger:              logger,
	}, nil
}

func parseConfigLocal(logger log.Logger, c *cli.Context) (bindgen.BindGenGeneratorLocal, error) {
	baseConfig, err := parseConfigBase(logger, c)
	if err != nil {
		return bindgen.BindGenGeneratorLocal{}, err
	}
	return bindgen.BindGenGeneratorLocal{
		BindGenGeneratorBase: baseConfig,
		SourceMapsList:       c.String(SourceMapsListFlagName),
		ForgeArtifactsPath:   c.String(ForgeArtifactsFlagName),
	}, nil
}

func parseConfigRemote(logger log.Logger, c *cli.Context) (bindgen.BindGenGeneratorRemote, error) {
	baseConfig, err := parseConfigBase(logger, c)
	if err != nil {
		return bindgen.BindGenGeneratorRemote{}, err
	}
	generator := bindgen.BindGenGeneratorRemote{
		BindGenGeneratorBase: baseConfig,
	}

	generator.ContractDataClients.Eth = etherscan.NewEthereumClient(c.String(EtherscanApiKeyEthFlagName))
	generator.ContractDataClients.Op = etherscan.NewOptimismClient(c.String(EtherscanApiKeyOpFlagName))

	if generator.RpcClients.Eth, err = ethclient.Dial(c.String(RpcUrlEthFlagName)); err != nil {
		return bindgen.BindGenGeneratorRemote{}, fmt.Errorf("error initializing Ethereum client: %w", err)
	}
	if generator.RpcClients.Op, err = ethclient.Dial(c.String(RpcUrlOpFlagName)); err != nil {
		return bindgen.BindGenGeneratorRemote{}, fmt.Errorf("error initializing Optimism client: %w", err)
	}
	return generator, nil
}

func baseFlags() []cli.Flag {
	baseFlags := []cli.Flag{
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

	return append(baseFlags, oplog.CLIFlags("bindgen")...)
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
