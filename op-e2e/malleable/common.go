package malleable

import (
	"testing"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	predeploys "github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	genesis "github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	e2e "github.com/ethereum-optimism/optimism/op-e2e"
	e2eutils "github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	eth "github.com/ethereum-optimism/optimism/op-node/eth"

	// rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	rollup "github.com/ethereum-optimism/optimism/op-node/rollup"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
)

// DefaultConfig returns a default [p2p.Config] for the malleable package.
func DefaultConfig() (*p2p.Config, error) {
	blockTime := uint64(2)
	params, err := p2p.GetPeerScoreParams("light", blockTime)
	if err != nil {
		return nil, err
	}
	topicScores, err := p2p.GetTopicScoreParams("light", blockTime)
	if err != nil {
		return nil, err
	}
	bandScoreThresholds, err := p2p.NewBandScorer("-40:graylist;-20:restricted;0:nopx;20:friend;")
	if err != nil {
		return nil, err
	}
	p2pPrivKey := secp256k1.NewPrivateKey(&secp256k1.ModNScalar{})
	return &p2p.Config{
		Priv:                (*crypto.Secp256k1PrivateKey)(p2pPrivKey),
		PeerScoring:         params,
		BanningEnabled:      false,
		BandScoreThresholds: *bandScoreThresholds,
		TopicScoring:        topicScores,
	}, nil
}

// GetRollupConfig returns a [rollup.Config] for the [MalleableNode].
func GetRollupConfig(t *testing.T) rollup.Config {
	defaultConfig := e2e.DefaultSystemConfig(t)
	defaultConfig.DeployConfig.L1BlockTime = 10
	l1Genesis, _ := genesis.BuildL1DeveloperGenesis(defaultConfig.DeployConfig)
	l1Block := l1Genesis.ToBlock()
	l2Genesis, _ := genesis.BuildL2DeveloperGenesis(defaultConfig.DeployConfig, l1Block)
	return rollup.Config{
		Genesis: rollup.Genesis{
			L1: eth.BlockID{
				Hash:   l1Block.Hash(),
				Number: 0,
			},
			L2: eth.BlockID{
				Hash:   l2Genesis.ToBlock().Hash(),
				Number: 0,
			},
			L2Time:       uint64(defaultConfig.DeployConfig.L1GenesisBlockTimestamp),
			SystemConfig: e2eutils.SystemConfigFromDeployConfig(defaultConfig.DeployConfig),
		},
		BlockTime:              defaultConfig.DeployConfig.L2BlockTime,
		MaxSequencerDrift:      defaultConfig.DeployConfig.MaxSequencerDrift,
		SeqWindowSize:          defaultConfig.DeployConfig.SequencerWindowSize,
		ChannelTimeout:         defaultConfig.DeployConfig.ChannelTimeout,
		L1ChainID:              defaultConfig.L1ChainIDBig(),
		L2ChainID:              defaultConfig.L2ChainIDBig(),
		BatchInboxAddress:      defaultConfig.DeployConfig.BatchInboxAddress,
		DepositContractAddress: predeploys.DevOptimismPortalAddr,
		L1SystemConfigAddress:  predeploys.DevSystemConfigAddr,
		RegolithTime:           defaultConfig.DeployConfig.RegolithTime(uint64(defaultConfig.DeployConfig.L1GenesisBlockTimestamp)),
	}
}
