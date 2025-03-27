package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BeaconClient is a thin wrapper over the Beacon APIs.
type BeaconClient interface {
	NodeVersion(ctx context.Context) (string, error)
	ConfigSpec(ctx context.Context) (eth.APIConfigResponse, error)
	BeaconGenesis(ctx context.Context) (eth.APIGenesisResponse, error)
	BlobSideCarsClient
}

// BlobSideCarsClient provides beacon blob sidecars
type BlobSideCarsClient interface {
	BeaconBlobSideCars(ctx context.Context, fetchAllSidecars bool, slot uint64, hashes []eth.IndexedBlobHash) (eth.APIGetBlobSidecarsResponse, error)
}
