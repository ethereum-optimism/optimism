package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/inject"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "inject-state",
		Usage: "Inject state into the geth db",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db-path",
				Required: true,
				Usage:    "Path to the geth db",
				EnvVars:  []string{"DB_PATH"},
			},
			&cli.StringFlag{
				Name:     "transition-file-path",
				Required: true,
				Usage:    "Path to the transition file",
				EnvVars:  []string{"TRANSITION_FILE_PATH"},
			},
			&cli.StringFlag{
				Name:     "deploy-config-path",
				Usage:    "Path to the deploy config file",
				Required: true,
				EnvVars:  []string{"DEPLOY_CONFIG_PATH"},
			},
			&cli.IntFlag{
				Name:  "hardfork-block",
				Usage: "Block number to hardfork at",
			},
			&cli.IntFlag{
				Name:  "db-cache",
				Usage: "LevelDB cache size in mb",
				Value: 1024,
			},
			&cli.IntFlag{
				Name:  "db-handles",
				Usage: "LevelDB number of handles",
				Value: 60,
			},
		},
		Action: entrypoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error inject-state", "err", err)
	}
}

func entrypoint(ctx *cli.Context) error {
	transitionFilePath := ctx.String("transition-file-path")

	transitionFile, err := os.Open(transitionFilePath)
	if err != nil {
		log.Error("error opening state file", "err", err)
		return err
	}
	defer transitionFile.Close()

	transitionState := new(core.Genesis)
	if err := json.NewDecoder(transitionFile).Decode(transitionState); err != nil {
		log.Error("failed to decode transition config file", "err", err)
		return err
	}

	deployConfigPath := ctx.String("deploy-config-path")
	deployConfig, err := genesis.NewDeployConfig(deployConfigPath)
	if err != nil {
		return fmt.Errorf("error loading deploy config: %w", err)
	}

	dbCache := ctx.Int("db-cache")
	dbHandles := ctx.Int("db-handles")
	dbPath := ctx.String("db-path")
	hardforkBlock := ctx.Int("hardfork-block")

	db, err := Open(dbPath, dbCache, dbHandles)
	if err != nil {
		return fmt.Errorf("cannot open DB: %w", err)
	}

	if err := inject.InjectState(transitionState, db, deployConfig, hardforkBlock); err != nil {
		return fmt.Errorf("error injecting state: %w", err)
	}

	// Close the database handle
	if err := db.Close(); err != nil {
		return err
	}

	return nil
}

func Open(path string, cache int, handles int) (ethdb.Database, error) {
	chaindataPath := filepath.Join(path, "geth", "chaindata")
	ancientPath := filepath.Join(chaindataPath, "ancient")
	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         chaindataPath,
		AncientsDirectory: ancientPath,
		Namespace:         "",
		Cache:             cache,
		Handles:           handles,
		ReadOnly:          false,
	})
	if err != nil {
		return nil, err
	}
	return ldb, nil
}
