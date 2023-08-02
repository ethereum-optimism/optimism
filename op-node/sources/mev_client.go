package sources

import (
	"context"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/pkg/errors"
)

type MevClient struct {
	mevEndpointAddr string
}

func NewMevClient(mevEndpointAddr string) (*MevClient, error) {
	if mevEndpointAddr == "" {
		return nil, errors.New("empty MEV Endpoint Address")
	}
	return &MevClient{
		mevEndpointAddr: mevEndpointAddr,
	}, nil
}

func (mc *MevClient) GetMevPayload(context.Context) (*eth.ExecutionPayload, error) {
	return nil, errors.New("not implemented")
}
