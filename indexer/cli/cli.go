package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/indexer/processor"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

type Cli struct {
	GitVersion string
	GitCommit  string
	GitDate    string
	app        *cli.App
	Flags      []cli.Flag
}

func runProcessor(ctx *cli.Context) error {
	var l1Proc *processor.L1Processor
	var l2Proc *processor.L2Processor
	configPath := ctx.String(ConfigFlag.Name)
	conf, err := config.LoadConfig(configPath)

	fmt.Println(conf)

	if err != nil {
		log.Crit("Failed to load config", "message", err)
	}

	db, err := database.NewDB(getDsn(conf.DB))

	if err != nil {
		log.Crit("Failed to connect to database", "message", err)
	}

	// L1 Processor
	l1EthClient, err := node.NewEthClient(conf.RPCs.L1RPC)
	if err != nil {
		return err
	}
	l1Proc, err = processor.NewL1Processor(l1EthClient, db)
	if err != nil {
		return err
	}

	// L2Processor
	l2EthClient, err := node.NewEthClient(conf.RPCs.L2RPC)
	if err != nil {
		return err
	}
	l2Proc, err = processor.NewL2Processor(l2EthClient, db)

	go l1Proc.Start()
	go l2Proc.Start()

	return nil
}

// Maybe make NewDB take a config.DBConfig instead of a string in future cleanup
func getDsn(dbConf config.DBConfig) string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.Name)
}

func runApi(ctx *cli.Context) error {
	configPath := ctx.String(ConfigFlag.Name)
	conf, err := config.LoadConfig(configPath)

	fmt.Println(conf)

	if err != nil {
		log.Crit("Failed to load config", "message", err)
	}

	db, err := database.NewDB(getDsn(conf.DB))

	if err != nil {
		log.Crit("Failed to connect to database", "message", err)
	}

	server := api.NewApi(db.Bridge)

	return server.Listen(strconv.Itoa(conf.API.Port))
}

var (
	ConfigFlag = &cli.StringFlag{
		Name:    "config",
		Value:   "./indexer.toml",
		Aliases: []string{"c"},
		Usage:   "path to config file",
		EnvVars: []string{"INDEXER_CONFIG"},
	}
	// Not used yet.  Use this flag to run legacy app instead
	// Remove me after indexer is released
	IndexerRefreshFlag = &cli.BoolFlag{
		Name:    "indexer-refresh",
		Value:   false,
		Aliases: []string{"i"},
		Usage:   "run new unreleased indexer by passing in flag",
		EnvVars: []string{"INDEXER_REFRESH"},
	}
)

// make a instance method on Cli called Run that runs cli
// and returns an error
func (c *Cli) Run(args []string) error {
	return c.app.Run(args)
}

func NewCli(GitVersion string, GitCommit string, GitDate string) *Cli {
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
		),
	)

	flags := []cli.Flag{
		ConfigFlag,
	}

	app := &cli.App{
		Version:     fmt.Sprintf("%s-%s", GitVersion, params.VersionWithCommit(GitCommit, GitDate)),
		Description: "An indexer of all optimism events with a serving api layer",
		Commands: []*cli.Command{
			{
				Name:        "api",
				Flags:       flags,
				Description: "Runs the api service",
				Action:      runApi,
			},
			{
				Name:        "indexer",
				Flags:       flags,
				Description: "Runs the indexing service",
				Action:      runProcessor,
			},
		},
	}

	return &Cli{
		app:   app,
		Flags: flags,
	}
}
