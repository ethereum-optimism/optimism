package feature

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type CoordinatorClient struct {
	sequencerId string
	rpc         *rpc.Client
}

func NewCoordinatorClient(url string, sequencerId string) (*CoordinatorClient, error) {
	rpc, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}
	return &CoordinatorClient{
		sequencerId: sequencerId,
		rpc:         rpc,
	}, nil
}

func (c *CoordinatorClient) RequestBuildingBlock() bool {
	var respErr error
	err := c.rpc.Call(respErr, "coordinator_requestBuildingBlock", c.sequencerId)
	if err != nil {
		log.Warn("Failed to call coordinator_requestBuildingBlock", "error", err)
		return false
	}
	if respErr != nil {
		log.Warn("coordinator_requestBuildingBlock refused request", "error", respErr)
		return false
	}
	return true
}
