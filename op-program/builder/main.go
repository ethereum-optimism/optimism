package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func runBuildPrestate(cliCtx *cli.Context) error {
	ctx := cliCtx.Context
	cfg := oplog.DefaultCLIConfig()
	logger := oplog.NewLogger(os.Stderr, cfg)
	oplog.SetGlobalLogHandler(logger.Handler())

	programELF := cliCtx.String("program-elf")
	version := cliCtx.String("version")
	suffix := cliCtx.String("suffix")
	if programELF == "" {
		return fmt.Errorf("program-elf is required")
	}
	if version == "" {
		return fmt.Errorf("version is required")
	}
	if suffix == "" {
		return fmt.Errorf("suffix is required")
	}
	ver, err := versions.ParseStateVersion(version)
	if err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	return buildPrestate(ctx, logger, programELF, ver, suffix)
}

func buildPrestate(ctx context.Context, log log.Logger, programELF string, version versions.StateVersion, suffix string) error {
	root, err := opservice.FindMonorepoRoot(".")
	if err != nil {
		return fmt.Errorf("failed to find monorepo root: %w", err)
	}
	if version < versions.VersionMultiThreaded64_v3 { // any version looks like 32-bits is unsupported
		return fmt.Errorf("version %d is not supported", version)
	}

	cannonBin := filepath.Join(root, "cannon", "bin", "cannon")
	if _, err := os.Stat(cannonBin); err != nil {
		return fmt.Errorf("cannon binary not found: %w. make sure it's built with `make cannon`", err)
	}
	prestate := filepath.Join(root, "op-program", "bin", "prestate"+suffix+".bin.gz")
	if _, err := os.Stat(programELF); err != nil {
		return fmt.Errorf("op-program elf not found: %w. make sure it's built with `make op-program`", err)
	}
	meta := filepath.Join(root, "op-program", "bin", "meta"+suffix+".json")

	cmd := exec.CommandContext(ctx, cannonBin, "load-elf", "--type", version.String(), "--path", programELF, "--out", prestate, "--meta", meta)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	proofFmt := filepath.Join(root, "op-program", "bin", "%d"+suffix+".json")
	cmd = exec.CommandContext(ctx, cannonBin, "run", "--proof-at", "=0", "--stop-at", "=1", "--input", prestate, "--meta", meta, "--proof-fmt", proofFmt, "--output", "")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	proof := filepath.Join(root, "op-program", "bin", "0"+suffix+".json")
	target := filepath.Join(root, "op-program", "bin", "prestate-proof"+suffix+".json")
	if err := os.Rename(proof, target); err != nil {
		return fmt.Errorf("failed to rename proof file: %w", err)
	}
	log.Info("Prestate proof saved", "path", target)
	return nil
}

func runBuildPrestates(cliCtx *cli.Context) error {
	logger := setupLogger()
	ctx := cliCtx.Context
	releasesOnly := cliCtx.Bool("releases-only")
	buildNext := versions.GetCurrentVersion() != versions.GetExperimentalVersion()

	root, err := opservice.FindMonorepoRoot(".")
	if err != nil {
		return fmt.Errorf("failed to find monorepo root: %w", err)
	}

	type prestateInfo struct {
		programELF string
		version    versions.StateVersion
		suffix     string
	}
	programELF := filepath.Join(root, "op-program", "bin", "op-program-client64.elf")
	interopProgramELF := filepath.Join(root, "op-program", "bin", "op-program-client-interop.elf")

	prestates := []prestateInfo{
		{programELF, versions.GetCurrentVersion(), "-mt64"},
		{interopProgramELF, versions.GetCurrentVersion(), "-interop"},
	}
	if !releasesOnly && buildNext {
		prestates = append(prestates, prestateInfo{programELF, versions.GetExperimentalVersion(), "-mt64Next"})
		prestates = append(prestates, prestateInfo{interopProgramELF, versions.GetExperimentalVersion(), "-interopNext"})
	}
	for _, prestate := range prestates {
		logger.Info("Building prestate", "version", prestate.version, "suffix", prestate.suffix)
		if err := buildPrestate(ctx, logger, prestate.programELF, prestate.version, prestate.suffix); err != nil {
			return fmt.Errorf("failed to build dev %s prestate: %w", prestate.suffix, err)
		}
	}

	if !releasesOnly && !buildNext {
		// some tests expect a "next" prestate to exist. So let's fake them if they weren't built.
		copies := []struct {
			src, dst string
		}{
			{filepath.Join(root, "op-program", "bin", "prestate-mt64.bin.gz"), filepath.Join(root, "op-program", "bin", "prestate-mt64Next.bin.gz")},
			{filepath.Join(root, "op-program", "bin", "prestate-interop.bin.gz"), filepath.Join(root, "op-program", "bin", "prestate-interopNext.bin.gz")},
		}
		for _, copy := range copies {
			data, err := os.ReadFile(copy.src)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", copy.src, err)
			}
			if err := os.WriteFile(copy.dst, data, 0644); err != nil {
				return fmt.Errorf("failed to copy %s: %w", copy.src, err)
			}
		}
	}
	return nil
}

func setupLogger() log.Logger {
	cfg := oplog.DefaultCLIConfig()
	logger := oplog.NewLogger(os.Stderr, cfg)
	oplog.SetGlobalLogHandler(logger.Handler())
	return logger
}

func main() {
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Name = "op-program-builder"
	app.Usage = "Tool to build op-program prestates in the monorepo"
	app.Commands = []*cli.Command{
		buildPrestateCommand,
		buildAllPrestates,
	}
	app.Action = func(c *cli.Context) error {
		return cli.ShowAppHelp(c)
	}
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("Application failed", "err", err)
	}
}

var buildPrestateCommand = &cli.Command{
	Name:  "build-prestate",
	Usage: "Build an op-program prestate",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "program-elf",
			Usage: "path to the program elf",
		},
		&cli.StringFlag{
			Name:  "version",
			Usage: "version of the program",
		},
		&cli.StringFlag{
			Name:  "suffix",
			Usage: "suffix used for the prestate filename. ex: -mt64, -interop, -mt64Next",
		},
	},
	Action: runBuildPrestate,
}

var buildAllPrestates = &cli.Command{
	Name:   "build-all-prestates",
	Usage:  "Build all op-program prestates",
	Action: runBuildPrestates,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "releases-only",
			Usage: "only build release versions of the prestates",
		},
	},
}
