package malleable

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	snappy "github.com/golang/snappy"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	p2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	rollup "github.com/ethereum-optimism/optimism/op-node/rollup"
)

// malpub is a malleable publisher for gossip.
type malpub struct {
	cfg         *rollup.Config
	blocksTopic *pubsub.Topic
}

// BlocksTopicPeers returns the list of peers subscribed to the blocks topic.
func (m *malpub) BlocksTopicPeers() []peer.ID {
	return m.blocksTopic.ListPeers()
}

// Close shuts down the [malpub], closing the pubsub topic.
func (m *malpub) Close() error {
	return m.blocksTopic.Close()
}

const maxGossipSize = 10 * (1 << 20)

var msgBufPool = sync.Pool{New: func() any {
	// note: the topic validator concurrency is limited, so pool won't blow up, even with large pre-allocation.
	x := make([]byte, 0, maxGossipSize)
	return &x
}}

// PublishL2Payload publishes an L2 payload to the gossip network.
func (m *malpub) PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayload, signer p2p.Signer) error {
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
	sig, err := signer.Sign(ctx, p2p.SigningDomainBlocksV1, m.cfg.L2ChainID, payloadData)
	if err != nil {
		return fmt.Errorf("failed to sign execution payload with signer: %w", err)
	}
	copy(data[:65], sig[:])

	// compress the full message
	// This also copies the data, freeing up the original buffer to go back into the pool
	out := snappy.Encode(nil, data)

	return m.blocksTopic.Publish(ctx, out)
}
