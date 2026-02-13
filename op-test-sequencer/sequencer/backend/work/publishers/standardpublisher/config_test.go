package standardpublisher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type apiFrontend struct {
	testAPI
}

func (t *apiFrontend) PublishBlockV1(ctx context.Context, envelope *opsigner.SignedExecutionPayloadEnvelope) error {
	t.v = envelope
	return t.err
}

func TestConfig(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	server := oprpc.NewServer("localhost", 0,
		"v0.0.1", oprpc.WithLogger(logger))
	api := &apiFrontend{}
	server.AddAPI(rpc.API{
		Namespace: "opstack",
		Service:   api,
	})
	require.NoError(t, server.Start())
	t.Cleanup(func() {
		_ = server.Stop()
	})
	cfg := &Config{
		RPC: endpoint.MustRPC{
			Value: endpoint.HttpURL("http://" + server.Endpoint()),
		},
	}
	id := seqtypes.PublisherID("test")
	ensemble := &work.Ensemble{}
	opts := &work.ServiceOpts{
		StartOpts: &work.StartOpts{
			Log:     logger,
			Metrics: &metrics.NoopMetrics{},
		},
		Services: ensemble,
	}
	publisher, err := cfg.Start(context.Background(), id, opts)
	require.NoError(t, err)
	require.Equal(t, id, publisher.ID())

	signed := &opsigner.SignedExecutionPayloadEnvelope{
		Envelope: &eth.ExecutionPayloadEnvelope{
			ParentBeaconBlockRoot: nil,
			ExecutionPayload: &eth.ExecutionPayload{
				BlockHash: common.Hash{123},
			},
		},
		Signature: eth.Bytes65{42},
	}
	err = publisher.Publish(context.Background(), signed)
	require.NoError(t, err)
	require.Equal(t, signed.Signature, api.v.Signature)
	require.Equal(t, signed.Envelope.ExecutionPayload.BlockHash, api.v.Envelope.ExecutionPayload.BlockHash)
}
