package superchain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

const (
	L2FlagName      = "l2"
	L2PeersFlagName = "l2-peers"
)

func superchainEnv(envprefix, v string) []string {
	return []string{envprefix + "_SUPERCHAIN_" + v}
}

func parseMapFlag(input string) (map[string]string, error) {
	result := map[string]string{}
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("Invalid key-value pair: %s\n", pair)
		}
		result[strings.TrimSpace(keyValue[0])] = strings.TrimSpace(keyValue[1])
	}
	return result, nil
}

func L2PeersFlag(envPrefix string) cli.Flag {
	return &cli.StringFlag{
		Name:    L2PeersFlagName,
		Usage:   "List of L2 Peers JSON-rpc endpoints. 'chain1=url1,chain2=url2...'",
		Value:   "",
		EnvVars: superchainEnv(envPrefix, "L2_PEERS_ETH_RPCS"),
	}
}

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		L2PeersFlag(envPrefix),
		&cli.StringFlag{
			Name:    L2FlagName,
			Usage:   "Address of L2 JSON-RPC endpoint to use (eth namespace required)",
			Value:   "http://127.0.0.1:9545",
			EnvVars: superchainEnv(envPrefix, "L2_ETH_RPC"),
		},
	}
}

type CLIConfig struct {
	L2RPCUrl      string
	L2PeersRPCUrl string
}

func (c CLIConfig) Check() error {
	if c.L2RPCUrl == "" {
		return errors.New("missing l2 rpc")
	}

	l2PeersMap, err := parseMapFlag(c.L2PeersRPCUrl)
	if err != nil {
		return fmt.Errorf("l2 peer rpcs encoded incorrectly: %w", err)
	}
	for id, _ := range l2PeersMap {
		_, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse chain id (%s): %w", id, err)
		}
	}

	return nil
}

func (c CLIConfig) Config() (*Config, error) {
	cfg := &Config{L2NodeAddr: c.L2PeersRPCUrl, PeerL2NodeAddrs: map[uint64]string{}}

	l2PeersMap, err := parseMapFlag(c.L2PeersRPCUrl)
	if err != nil {
		return nil, fmt.Errorf("l2 peer rpcs encoded incorrectly: %w", err)
	}
	for id, url := range l2PeersMap {
		chainId, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse chain id: %w", err)
		}
		cfg.PeerL2NodeAddrs[chainId] = url
	}

	return cfg, nil
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		L2RPCUrl:      ctx.String(L2FlagName),
		L2PeersRPCUrl: ctx.String(L2PeersFlagName),
	}
}
