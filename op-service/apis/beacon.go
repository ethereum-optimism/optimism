package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// BeaconClient is a thin wrapper over the Beacon APIs.
type BeaconClient interface {
	NodeVersion(ctx context.Context) (string, error)
	ConfigSpec(ctx context.Context) (eth.APIConfigResponse, error)
	BeaconGenesis(ctx context.Context) (eth.APIGenesisResponse, error)
	BeaconBlobs(ctx context.Context, slot uint64, hashes []common.Hash) (eth.APIBeaconBlobsResponse, error)
}
