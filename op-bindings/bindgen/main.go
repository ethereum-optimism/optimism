package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-e2e/config"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type bindGenGeneratorBase struct {
	metadataOut         string
	bindingsPackageName string
	monorepoBasePath    string
	contractsListPath   string
	logger              log.Logger
}

const (
	// Base Flags
	MetadataOutFlagName         = "metadata-out"
	BindingsPackageNameFlagName = "bindings-package"
	ContractsListFlagName       = "contracts-list"

	// Local Contracts Flags
	SourceMapsListFlagName = "source-maps-list"
	ForgeArtifactsFlagName = "forge-artifacts"
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
						Name:   "local",
						Usage:  "Generate bindings for locally sourced contracts",
						Flags:  localFlags(),
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
	case "local":
		localBindingsGenerator, err := parseConfigLocal(logger, c)
		if err != nil {
			return err
		}
		if err := localBindingsGenerator.generateBindings(); err != nil {
			return fmt.Errorf("error generating local bindings: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown command: %s", c.Command.Name)
	}
}

func parseConfigBase(logger log.Logger, c *cli.Context) (bindGenGeneratorBase, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return bindGenGeneratorBase{}, err
	}

	monoRepoPath, err := config.FindMonorepoRoot(cwd)
	if err != nil {
		return bindGenGeneratorBase{}, err
	}

	return bindGenGeneratorBase{
		metadataOut:         c.String(MetadataOutFlagName),
		bindingsPackageName: c.String(BindingsPackageNameFlagName),
		monorepoBasePath:    monoRepoPath,
		contractsListPath:   c.String(ContractsListFlagName),
		logger:              logger,
	}, nil
}

func parseConfigLocal(logger log.Logger, c *cli.Context) (bindGenGeneratorLocal, error) {
	baseConfig, err := parseConfigBase(logger, c)
	if err != nil {
		return bindGenGeneratorLocal{}, err
	}
	return bindGenGeneratorLocal{
		bindGenGeneratorBase: baseConfig,
		sourceMapsList:       c.String(SourceMapsListFlagName),
		forgeArtifactsPath:   c.String(ForgeArtifactsFlagName),
	}, nil
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
