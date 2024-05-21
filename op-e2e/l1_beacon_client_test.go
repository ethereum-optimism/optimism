package op_e2e

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	InitParallel(t)

	l := testlog.Logger(t, log.LevelInfo)

	beaconApi := fakebeacon.NewBeacon(l, t.TempDir(), uint64(0), uint64(0))
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
	InitParallel(t)

	l := testlog.Logger(t, log.LevelInfo)

	beaconApi := fakebeacon.NewBeacon(l, t.TempDir(), uint64(0), uint64(12))
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

func TestInvalidBlobResponse(t *testing.T) {
	InitParallel(t)

	l := testlog.Logger(t, log.LevelInfo)

	beaconApi := fakebeacon.NewBeacon(l, t.TempDir(), uint64(0), uint64(12))
	t.Cleanup(func() {
		_ = beaconApi.Close()
	})
	require.NoError(t, beaconApi.Start("127.0.0.1:0"))

	blob := kzg4844.Blob{}
	blob[0] = byte(1)
	commit, _ := kzg4844.BlobToCommitment(blob)
	proof, _ := kzg4844.ComputeBlobProof(blob, commit)
	hash := eth.KZGToVersionedHash(commit)
	require.NoError(t, beaconApi.StoreBlobsBundle(10, &engine.BlobsBundleV1{
		Commitments: []hexutil.Bytes{hexutil.Bytes(commit[:])},
		Proofs:      []hexutil.Bytes{hexutil.Bytes(proof[:])},
		Blobs:       []hexutil.Bytes{hexutil.Bytes(blob[:])},
	}))

	beaconCfg := sources.L1BeaconClientConfig{FetchAllSidecars: true}
	cl := sources.NewL1BeaconClient(sources.NewBeaconHTTPClient(client.NewBasicHTTPClient(beaconApi.BeaconAddr(), l)), beaconCfg)

	// Succeed case
	hashes := []eth.IndexedBlobHash{{Index: 0, Hash: hash}}
	_, err := cl.GetBlobs(context.Background(), eth.L1BlockRef{Number: 10, Time: 120}, hashes)
	require.NoError(t, err)

	// Index in response of beaconApi is mismatched, should return ethereum.NotFound
	hashes = []eth.IndexedBlobHash{{Index: 1, Hash: hash}}
	_, err = cl.GetBlobs(context.Background(), eth.L1BlockRef{Number: 10, Time: 120}, hashes)
	require.ErrorIs(t, err, ethereum.NotFound)

	// Hash in response of beaconApi is mismatched, should return ethereum.NotFound
	hashes = []eth.IndexedBlobHash{{Index: 0, Hash: common.Hash{}}}
	_, err = cl.GetBlobs(context.Background(), eth.L1BlockRef{Number: 10, Time: 120}, hashes)
	require.ErrorIs(t, err, ethereum.NotFound)
}
