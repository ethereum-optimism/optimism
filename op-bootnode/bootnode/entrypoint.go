package bootnode

import (
	"context"
	"errors"
	"fmt"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/urfave/cli"
)

type gossipNoop struct{}

func (g *gossipNoop) OnUnsafeL2Payload(_ context.Context, _ peer.ID, _ *eth.ExecutionPayload) error {
	return nil
}

type gossipConfig struct{}

func (g *gossipConfig) P2PSequencerAddress() common.Address {
	return common.Address{}
}

type l2Chain struct{}

func (l *l2Chain) PayloadByNumber(_ context.Context, _ uint64) (*eth.ExecutionPayload, error) {
	return nil, nil
}

func Main(cliCtx *cli.Context) error {
	log.Info("Initializing bootnode")
	logCfg := oplog.ReadCLIConfig(cliCtx)
	logger := oplog.NewLogger(logCfg)
	m := metrics.NewMetrics("default")
	ctx := context.Background()

	config, err := opnode.NewRollupConfig(cliCtx)
	if err != nil {
		return err
	}
	if err = validateConfig(config); err != nil {
		return err
	}

	p2pConfig, err := p2pcli.NewConfig(cliCtx, config.BlockTime)
	if err != nil {
		return fmt.Errorf("failed to load p2p config: %w", err)
	}

	p2pNode, err := p2p.NewNodeP2P(ctx, config, logger, p2pConfig, &gossipNoop{}, &l2Chain{}, &gossipConfig{}, m)
	if err != nil || p2pNode == nil {
		return err
	}
	if p2pNode.Dv5Udp() == nil {
		return fmt.Errorf("uninitialized discovery service")
	}

	go p2pNode.DiscoveryProcess(ctx, logger, config, p2pConfig.TargetPeers())

	opio.BlockOnInterrupts()

	return nil
}

// validateConfig ensures the minimal config required to run a bootnode
func validateConfig(config *rollup.Config) error {
	if config.L2ChainID == nil || config.L2ChainID.Uint64() == 0 {
		return errors.New("chain ID is not set")
	}
	if config.Genesis.L2Time <= 0 {
		return errors.New("genesis timestamp is not set")
	}
	if config.BlockTime <= 0 {
		return errors.New("block time is not set")
	}
	return nil
}
