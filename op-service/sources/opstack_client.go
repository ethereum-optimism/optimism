package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
)

type OPStackClient struct {
	rpc client.RPC
}

var _ apis.OPStackAPI = (*OPStackClient)(nil)

func NewOPStackClient(rpc client.RPC) *OPStackClient {
	return &OPStackClient{rpc}
}

func (r *OPStackClient) OpenBlock(ctx context.Context, parent eth.BlockID, attrs *eth.PayloadAttributes) (eth.PayloadInfo, error) {
	var result eth.PayloadInfo
	err := r.rpc.CallContext(ctx, &result, "opstack_openBlockV1", parent, attrs)
	return result, err
}

func (r *OPStackClient) CancelBlock(ctx context.Context, id eth.PayloadInfo) error {
	return r.rpc.CallContext(ctx, nil, "opstack_cancelBlockV1", id)
}

func (r *OPStackClient) SealBlock(ctx context.Context, id eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error) {
	var result *eth.ExecutionPayloadEnvelope
	err := r.rpc.CallContext(ctx, &result, "opstack_sealBlockV1", id)
	return result, err
}

func (r *OPStackClient) CommitBlock(ctx context.Context, envelope *opsigner.SignedExecutionPayloadEnvelope) error {
	return r.rpc.CallContext(ctx, nil, "opstack_commitBlockV1", envelope)
}

func (r *OPStackClient) PublishBlock(ctx context.Context, signed *opsigner.SignedExecutionPayloadEnvelope) error {
	return r.rpc.CallContext(ctx, nil, "opstack_publishBlockV1", signed)
}

func (r *OPStackClient) Close() {
	r.rpc.Close()
}
