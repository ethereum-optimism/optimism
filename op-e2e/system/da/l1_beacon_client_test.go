package da

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	op_e2e.InitParallel(t)

	l := testlog.Logger(t, log.LevelInfo)

	blobStore := e2eutils.NewBlobStore()
	beaconApi := fakebeacon.NewBeacon(l, blobStore, uint64(0), uint64(0))
	t.Cleanup(func() {
		_ = beaconApi.Close()
	})
	require.NoError(t, beaconApi.Start("127.0.0.1:0"))

	beaconCfg := sources.L1BeaconClientConfig{FetchAllSidecars: false}
	cl := sources.NewL1BeaconClient(sources.NewBeaconHTTPClient(client.NewBasicHTTPClient(beaconApi.BeaconAddr(), l)), beaconCfg)

	version, err := cl.GetVersion(context.Background())
	require.NoError(t, err)
	require.Equal(t, "fakebeacon 1.2.3", version)
}

func Test404NotFound(t *testing.T) {
	op_e2e.InitParallel(t)

	l := testlog.Logger(t, log.LevelInfo)

	blobStore := e2eutils.NewBlobStore()
	beaconApi := fakebeacon.NewBeacon(l, blobStore, uint64(0), uint64(12))
	t.Cleanup(func() {
		_ = beaconApi.Close()
	})
	require.NoError(t, beaconApi.Start("127.0.0.1:0"))

	beaconCfg := sources.L1BeaconClientConfig{FetchAllSidecars: false}
	cl := sources.NewL1BeaconClient(sources.NewBeaconHTTPClient(client.NewBasicHTTPClient(beaconApi.BeaconAddr(), l)), beaconCfg)

	hashes := []eth.IndexedBlobHash{{Index: 1}}
	_, err := cl.GetBlobs(context.Background(), eth.L1BlockRef{Number: 10, Time: 120}, hashes)
	require.ErrorIs(t, err, ethereum.NotFound)
}
