package bootnode

import (
	"context"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/opio"
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
	logger := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(logger.GetHandler())
	m := metrics.NewMetrics("default")
	ctx := context.Background()

	config, err := opnode.NewRollupConfig(logger, cliCtx)
	if err != nil {
		return err
	}
	if err = validateConfig(config); err != nil {
		return err
	}

	p2pConfig, err := p2pcli.NewConfig(cliCtx, config)
	if err != nil {
		return fmt.Errorf("failed to load p2p config: %w", err)
	}

	p2pNode, err := p2p.NewNodeP2P(ctx, config, logger, p2pConfig, &gossipNoop{}, &l2Chain{}, &gossipConfig{}, m, false)
	if err != nil || p2pNode == nil {
		return err
	}
	if p2pNode.Dv5Udp() == nil {
		return fmt.Errorf("uninitialized discovery service")
	}

	go p2pNode.DiscoveryProcess(ctx, logger, config, p2pConfig.TargetPeers())

	metricsCfg := opmetrics.ReadCLIConfig(cliCtx)
	if metricsCfg.Enabled {
		log.Debug("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		metricsSrv, err := m.StartServer(metricsCfg.ListenAddr, metricsCfg.ListenPort)
		if err != nil {
			return fmt.Errorf("failed to start metrics server: %w", err)
		}
		defer func() {
			if err := metricsSrv.Stop(context.Background()); err != nil {
				log.Error("failed to stop metrics server", "err", err)
			}
		}()
		log.Info("started metrics server", "addr", metricsSrv.Addr())
		m.RecordUp()
	}

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
