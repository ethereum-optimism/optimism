package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/interop"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

var configFlag = &cli.StringFlag{
	Name:    "config",
	Value:   "./postie.toml",
	Aliases: []string{"c"},
	Usage:   "path to config file",
	EnvVars: []string{"INTEROP_POSTIE_CONFIG"},
}

func newCli() *cli.App {
	flags := oplog.CLIFlags("INTEROP_POSTIE")
	flags = append(flags, configFlag)

	return &cli.App{
		Name: "interop",
		Commands: []*cli.Command{
			{
				Name:        "postie",
				Description: "daemon that watches connected chains relative to the destination chain for inbox updates",
				Flags:       flags,
				Action:      cliapp.LifecycleCmd(runPostie),
			},
		},
	}
}

func runPostie(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx)).New("role", "postie")
	metricsRegistry := metrics.NewRegistry()

	/** Load Config **/

	var cfg struct {
		PrivateKey string `toml:"private-key"`

		DestinationChainRPC string   `toml:"destination-rpc"`
		ConnectedChainRPCs  []string `toml:"connected-rpcs"`

		UpdateIntervalMinutes int64 `toml:"update-interval-minutes"`
	}

	log.Info("reading config")
	data, err := os.ReadFile(ctx.String(configFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("unable to read conifg: %w", err)
	}

	data = []byte(os.ExpandEnv(string(data)))
	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	} else if len(md.Undecoded()) > 0 {
		return nil, fmt.Errorf("unknown fields in config file: %s", md.Undecoded())
	}

	/** Setup Key & Clients **/

	if len(cfg.PrivateKey) >= 2 && strings.ToLower(cfg.PrivateKey)[:2] == "0x" {
		cfg.PrivateKey = cfg.PrivateKey[2:]
	}

	postieSecret, err := crypto.HexToECDSA(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to create ecdsa key from provided private key: %w", err)
	}

	connectedClients := make([]node.EthClient, len(cfg.ConnectedChainRPCs))
	for i, rpcUrl := range cfg.ConnectedChainRPCs {
		clnt, err := node.DialEthClient(context.Background(), rpcUrl, node.NewMetrics(metricsRegistry, ""))
		if err != nil {
			return nil, fmt.Errorf("unable to dial client %s: %w", rpcUrl, err)
		}
		connectedClients[i] = clnt
	}

	if !client.IsURLAvailable(cfg.DestinationChainRPC) {
		return nil, fmt.Errorf("address unavailable (%s)", cfg.DestinationChainRPC)
	}
	destinationClient, err := ethclient.Dial(cfg.DestinationChainRPC)
	if err != nil {
		return nil, fmt.Errorf("unable to dial client %s: %w", cfg.DestinationChainRPC, err)
	}

	/** Start the Postie Daemon **/

	return interop.NewPostie(log, interop.PostieConfig{
		Postie:           postieSecret,
		DestinationChain: destinationClient,
		ConnectedChains:  connectedClients,
		UpdateInterval:   time.Duration(cfg.UpdateIntervalMinutes) * time.Minute,
	})
}
