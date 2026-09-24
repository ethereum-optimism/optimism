package p2p

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type cancelingSigner struct {
	cancel context.CancelFunc
	called bool
}

func (s *cancelingSigner) SignBlockV1(context.Context, eth.ChainID, common.Hash) (eth.Bytes65, error) {
	s.called = true
	s.cancel()
	// A response may arrive concurrently with cancellation, or a signer may
	// not honor it. A successful signature is not permission to publish anymore.
	return eth.Bytes65{}, nil
}

func (*cancelingSigner) Close() error { return nil }

func TestPublisherDoesNotPublishAfterSigningCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	signer := &cancelingSigner{cancel: cancel}
	// Topics are deliberately absent: a canceled signature must not reach
	// topic selection or Topic.Publish at all.
	p := &publisher{cfg: &rollup.Config{L2ChainID: big.NewInt(10)}}
	envelope := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{}}
	err := p.SignAndPublishL2Payload(ctx, envelope, signer)
	require.True(t, signer.called)
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, async.ErrPermanentPublish, "cancellation is not a permanent payload error")
}
