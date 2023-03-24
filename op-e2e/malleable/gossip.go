package malleable

import (
	"context"
	"fmt"
	"testing"
	"time"

	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	rollup "github.com/ethereum-optimism/optimism/op-node/rollup"
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

// SetupGossip sets up gossip for the [MalleableNode].
func (m *MalleableNode) SetupGossip(t *testing.T, conf *p2p.Config, snapLog log.Logger) error {
	rollupCfg := GetRollupConfig(t)

	// Enable extra features, if any. During testing we don't setup the most advanced host all the time.
	if extra, ok := m.innerHost.(p2p.ExtraHostFeatures); ok {
		m.gater = extra.ConnectionGater()
		m.connMgr = extra.ConnectionManager()
	}

	// Setup the host network
	notifiee := p2p.NewNetworkNotifier(snapLog, nil)
	m.innerHost.Network().Notify(notifiee)

	// Construct the pubsub instance with a GossipSub router internally
	var err error
	m.gs, err = p2p.NewGossipSub(context.Background(), m.innerHost, m.gater, &rollupCfg, conf, nil, snapLog)
	if err != nil {
		return fmt.Errorf("failed to start gossipsub router: %w", err)
	}

	// Register to the associated gossip
	m.gsOut, err = JoinGossip(context.Background(), m.innerHost.ID(), conf.TopicScoringParams(), m.gs, snapLog, &rollupCfg, nil, m)
	if err != nil {
		return fmt.Errorf("failed to join blocks gossip topic: %w", err)
	}

	return nil
}

func guardGossipValidator(log log.Logger, fn pubsub.ValidatorEx) pubsub.ValidatorEx {
	return func(ctx context.Context, id peer.ID, message *pubsub.Message) (result pubsub.ValidationResult) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("gossip validation panic", "err", err, "peer", id)
				result = pubsub.ValidationReject
			}
		}()
		return fn(ctx, id, message)
	}
}

// JoinGossip joins the gossip network for the peer's given topic parameters.
func JoinGossip(
	p2pCtx context.Context,
	self peer.ID,
	topicScoreParams *pubsub.TopicScoreParams,
	ps *pubsub.PubSub,
	log log.Logger,
	cfg *rollup.Config,
	runCfg p2p.GossipRuntimeConfig,
	gossipIn p2p.GossipIn,
) (p2p.GossipOut, error) {
	val := guardGossipValidator(log, p2p.BuildBlocksValidator(log, cfg, runCfg))
	blocksTopicName := fmt.Sprintf("/optimism/%s/0/blocks", cfg.L2ChainID.String())
	err := ps.RegisterTopicValidator(blocksTopicName,
		val,
		pubsub.WithValidatorTimeout(3*time.Second),
		pubsub.WithValidatorConcurrency(4),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register blocks gossip topic: %w", err)
	}
	blocksTopic, err := ps.Join(blocksTopicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join blocks gossip topic: %w", err)
	}
	blocksTopicEvents, err := blocksTopic.EventHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to create blocks gossip topic handler: %w", err)
	}
	go p2p.LogTopicEvents(p2pCtx, log.New("topic", "blocks"), blocksTopicEvents)

	// A [TimeInMeshQuantum] value of 0 means the topic score is disabled.
	// If we passed a topicScoreParams with [TimeInMeshQuantum] set to 0,
	// libp2p errors since the params will be rejected.
	if topicScoreParams != nil && topicScoreParams.TimeInMeshQuantum != 0 {
		if err = blocksTopic.SetScoreParams(topicScoreParams); err != nil {
			return nil, fmt.Errorf("failed to set topic score params: %w", err)
		}
	}

	subscription, err := blocksTopic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to blocks gossip topic: %w", err)
	}

	subscriber := p2p.MakeSubscriber(log, p2p.BlocksHandler(gossipIn.OnUnsafeL2Payload))
	go subscriber(p2pCtx, subscription)

	return &malpub{cfg: cfg, blocksTopic: blocksTopic}, nil
}
