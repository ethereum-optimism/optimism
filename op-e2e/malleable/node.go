package malleable

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sync"

	log "github.com/ethereum/go-ethereum/log"
	snappy "github.com/golang/snappy"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	host "github.com/libp2p/go-libp2p/core/host"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
)

// Malleable provides the necessary tooling for creating bad gossip messages.
type Malleable struct {
	h           host.Host
	l2ChainID   *big.Int
	blocksTopic *pubsub.Topic
	priv        crypto.PrivKey
}

// NewMalleable creates a new Malleable instance.
func NewMalleable(
	log log.Logger,
	l2ChainID *big.Int,
	topicScoreParams *pubsub.TopicScoreParams,
	priv crypto.PrivKey,
) (*Malleable, error) {
	// Create a new libp2p host.
	h, err := DefaultHost(priv)
	if err != nil {
		return nil, fmt.Errorf("failed to start p2p host: %w", err)
	}

	// Construct a new gossipsub router.
	ps, err := NewGossipSub(h)
	if err != nil {
		return nil, fmt.Errorf("failed to start gossipsub router: %w", err)
	}

	// Join the blocks gossip topic.
	blocksTopicName := getBlockTopicName(l2ChainID)
	blocksTopic, err := ps.Join(blocksTopicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join blocks gossip topic: %w", err)
	}

	// A [TimeInMeshQuantum] value of 0 means the topic score is disabled.
	if topicScoreParams != nil && topicScoreParams.TimeInMeshQuantum != 0 {
		if err = blocksTopic.SetScoreParams(topicScoreParams); err != nil {
			return nil, fmt.Errorf("failed to set topic score params: %w", err)
		}
	}

	// Subscribe to the blocks gossip topic.
	subscription, err := blocksTopic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to blocks gossip topic: %w", err)
	}
	subscriber := p2p.MakeSubscriber(log, p2p.BlocksHandler(OnUnsafeL2Payload))
	go subscriber(context.Background(), subscription)

	return &Malleable{
		h:           h,
		priv:        priv,
		l2ChainID:   l2ChainID,
		blocksTopic: blocksTopic,
	}, nil
}

// Connect connects the internal [host.Host] to a [peer].
func (m *Malleable) Connect(ctx context.Context, pi peer.AddrInfo) error {
	return m.h.Connect(ctx, pi)
}

// ID returns the [peer.ID] of the internal [host.Host].
func (m *Malleable) ID() peer.ID {
	return m.h.ID()
}

// Addrs returns the listen addresses [ma.Multiaddr] of the internal [host.Host]
func (m *Malleable) Addrs() []ma.Multiaddr {
	return m.h.Addrs()
}

// Network returns the [network.Network] of the internal [host.Host]
func (m *Malleable) Network() network.Network {
	return m.h.Network()
}

var msgBufPool = sync.Pool{New: func() any {
	// note: the topic validator concurrency is limited, so pool won't blow up, even with large pre-allocation.
	x := make([]byte, 0, maxGossipSize)
	return &x
}}

// PublishL2Payload publishes an [eth.ExecutionPayload] to its [pubsub.Topic].
func (m *Malleable) PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayload, signer p2p.Signer) error {
	res := msgBufPool.Get().(*[]byte)
	buf := bytes.NewBuffer((*res)[:0])
	defer func() {
		*res = buf.Bytes()
		defer msgBufPool.Put(res)
	}()

	buf.Write(make([]byte, 65))
	if _, err := payload.MarshalSSZ(buf); err != nil {
		return fmt.Errorf("failed to encoded execution payload to publish: %w", err)
	}
	data := buf.Bytes()
	payloadData := data[65:]
	sig, err := signer.Sign(ctx, p2p.SigningDomainBlocksV1, m.l2ChainID, payloadData)
	if err != nil {
		return fmt.Errorf("failed to sign execution payload with signer: %w", err)
	}
	copy(data[:65], sig[:])

	// compress the full message
	// This also copies the data, freeing up the original buffer to go back into the pool
	out := snappy.Encode(nil, data)

	return m.blocksTopic.Publish(ctx, out)
}
