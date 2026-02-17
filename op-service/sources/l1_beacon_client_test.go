package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	client_mocks "github.com/ethereum-optimism/optimism/op-service/client/mocks"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/mocks"
)

//go:generate mockery --srcpkg=github.com/ethereum-optimism/optimism/op-service/apis --name BeaconClient --with-expecter=true

func makeTestBlob(index uint64) (eth.IndexedBlobHash, *eth.Blob) {
	blob := kzg4844.Blob{}
	// make first byte of test blob match its index so we can easily verify if is returned in the
	// expected order
	blob[0] = byte(index)
	commit, _ := kzg4844.BlobToCommitment(&blob)
	hash := eth.KZGToVersionedHash(commit)

	idh := eth.IndexedBlobHash{
		Index: index,
		Hash:  hash,
	}
	ethBLob := eth.Blob(blob)
	return idh, &ethBLob
}

func KZGProofFromHex(s string) (kzg4844.Proof, error) {
	var out kzg4844.Proof // underlying size is 48 bytes
	b, err := hexutil.Decode(s)
	if err != nil {
		return out, err
	}
	if len(b) != 48 {
		return out, fmt.Errorf("want 48 bytes, got %d", len(b))
	}
	copy(out[:], b)
	return out, nil
}

func TestBeaconClientNoErrorPrimary(t *testing.T) {
	indices := []uint64{5, 7, 2}
	index0, blob0 := makeTestBlob(indices[0])
	index1, blob1 := makeTestBlob(indices[1])
	index2, blob2 := makeTestBlob(indices[2])
	hashes := []common.Hash{index0.Hash, index1.Hash, index2.Hash}
	blobs := []*eth.Blob{blob0, blob1, blob2}

	ctx := context.Background()
	p := mocks.NewBeaconClient(t)
	f := mocks.NewBeaconClient(t)
	c := NewL1BeaconClient(p, L1BeaconClientConfig{}, f)
	p.EXPECT().BeaconGenesis(ctx).Return(eth.APIGenesisResponse{Data: eth.ReducedGenesisData{GenesisTime: 10}}, nil)
	p.EXPECT().ConfigSpec(ctx).Return(eth.APIConfigResponse{Data: eth.ReducedConfigData{SecondsPerSlot: 2}}, nil)
	// Timestamp 12 = Slot 1
	p.EXPECT().BeaconBlobs(ctx, uint64(1), hashes).Return(eth.APIBeaconBlobsResponse{Data: blobs}, nil)

	resp, err := c.GetBlobsByHash(ctx, 12, hashes)
	require.NoError(t, err)
	require.Equal(t, blobs, resp)

}

func TestBeaconClientFallback(t *testing.T) {
	indices := []uint64{5, 7, 2}
	index0, blob0 := makeTestBlob(indices[0])
	index1, blob1 := makeTestBlob(indices[1])
	index2, blob2 := makeTestBlob(indices[2])
	hashes := []common.Hash{index0.Hash, index1.Hash, index2.Hash}
	blobs := []*eth.Blob{blob0, blob1, blob2}

	ctx := context.Background()
	p := mocks.NewBeaconClient(t)
	f := mocks.NewBeaconClient(t)
	c := NewL1BeaconClient(p, L1BeaconClientConfig{}, f)
	p.EXPECT().BeaconGenesis(ctx).Return(eth.APIGenesisResponse{Data: eth.ReducedGenesisData{GenesisTime: 10}}, nil)
	p.EXPECT().ConfigSpec(ctx).Return(eth.APIConfigResponse{Data: eth.ReducedConfigData{SecondsPerSlot: 2}}, nil)
	// Timestamp 12 = Slot 1
	p.EXPECT().BeaconBlobs(ctx, uint64(1), hashes).Return(eth.APIBeaconBlobsResponse{}, errors.New("404 not found"))
	f.EXPECT().BeaconBlobs(ctx, uint64(1), hashes).Return(eth.APIBeaconBlobsResponse{Data: blobs}, nil)

	resp, err := c.GetBlobsByHash(ctx, 12, hashes)
	require.NoError(t, err)
	require.Equal(t, blobs, resp)

	// Second set of calls. This time rotate back to the primary
	indices = []uint64{3, 9, 11}
	index0, blob0 = makeTestBlob(indices[0])
	index1, blob1 = makeTestBlob(indices[1])
	index2, blob2 = makeTestBlob(indices[2])
	hashes = []common.Hash{index0.Hash, index1.Hash, index2.Hash}
	blobs = []*eth.Blob{blob0, blob1, blob2}

	// Timestamp 14 = Slot 2
	f.EXPECT().BeaconBlobs(ctx, uint64(2), hashes).Return(eth.APIBeaconBlobsResponse{}, errors.New("404 not found"))
	p.EXPECT().BeaconBlobs(ctx, uint64(2), hashes).Return(eth.APIBeaconBlobsResponse{Data: blobs}, nil)

	resp, err = c.GetBlobsByHash(ctx, 14, hashes)
	require.NoError(t, err)
	require.Equal(t, blobs, resp)
}

func TestBeaconHTTPClient(t *testing.T) {
	c := client_mocks.NewHTTP(t)
	b := NewBeaconHTTPClient(c)

	ctx := context.Background()

	indices := []uint64{3, 9, 11}
	index0, _ := makeTestBlob(indices[0])
	index1, _ := makeTestBlob(indices[1])
	index2, _ := makeTestBlob(indices[2])

	hashes := []common.Hash{index0.Hash, index1.Hash, index2.Hash}

	// mocks returning a 200 with empty list
	respBytes, _ := json.Marshal(eth.APIBeaconBlobsResponse{})
	slot := uint64(2)
	path := path.Join(blobsMethodPrefix, strconv.FormatUint(slot, 10))
	reqQuery := url.Values{}
	for i := range hashes {
		reqQuery.Add("versioned_hashes", hashes[i].Hex())
	}
	headers := http.Header{}
	headers.Add("Accept", "application/json")
	c.EXPECT().Get(ctx, path, reqQuery, headers).Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(respBytes))}, nil)

	// BeaconBlobs should return error when client.HTTP returns a 200 with empty list
	_, err := b.BeaconBlobs(ctx, slot, hashes)
	require.Error(t, err)
	require.Equal(t, err.Error(), fmt.Sprintf("#returned blobs(%d) != #requested blobs(%d)", 0, len(hashes)))
}

func TestClientPoolSingle(t *testing.T) {
	p := NewClientPool(1)
	for i := 0; i < 10; i++ {
		require.Equal(t, 1, p.Get())
		p.MoveToNext()
	}
}
func TestClientPoolSeveral(t *testing.T) {
	p := NewClientPool(0, 1, 2, 3)
	for i := 0; i < 25; i++ {
		require.Equal(t, i%4, p.Get())
		p.MoveToNext()
	}
}

func TestVerifyBlob(t *testing.T) {
	blob := eth.Blob{}
	blob[0] = byte(7)
	versionedHash := common.HexToHash("0x0164e32184169f11528f72aeb318f94d958aa28fba0731a52aead6df0104a98e")
	require.NoError(t, verifyBlob(&blob, versionedHash))

	differentBlob := eth.Blob{}
	differentBlob[0] = byte(8)
	require.Error(t, verifyBlob(&differentBlob, versionedHash))
}

func TestGetBlobs(t *testing.T) {
	hash0, blob0 := makeTestBlob(0)
	hash1, blob1 := makeTestBlob(1)
	hash2, blob2 := makeTestBlob(2)

	hashes := []common.Hash{hash0.Hash, hash2.Hash, hash1.Hash} // Mix up the order.
	beaconBlobs := []*eth.Blob{blob0, blob2, blob1}

	ctx := context.Background()
	p := mocks.NewBeaconClient(t)
	p.EXPECT().BeaconGenesis(ctx).Return(eth.APIGenesisResponse{Data: eth.ReducedGenesisData{GenesisTime: 10}}, nil)
	p.EXPECT().ConfigSpec(ctx).Return(eth.APIConfigResponse{Data: eth.ReducedConfigData{SecondsPerSlot: 2}}, nil)
	client := NewL1BeaconClient(p, L1BeaconClientConfig{})
	ref := eth.L1BlockRef{Time: 12}

	// construct the mock response for the beacon blobs call
	var beaconBlobsResponse eth.APIBeaconBlobsResponse
	var err error
	beaconBlobsResponse = eth.APIBeaconBlobsResponse{Data: beaconBlobs}
	p.EXPECT().BeaconBlobs(ctx, uint64(1), hashes).Return(beaconBlobsResponse, err)

	resp, err := client.GetBlobsByHash(ctx, ref.Time, hashes)
	require.NoError(t, err)
	require.Equal(t, beaconBlobs, resp)
}

func TestRequestDuplicateBlobHashes(t *testing.T) {
	ctx := context.Background()
	p := mocks.NewBeaconClient(t)
	p.EXPECT().BeaconGenesis(ctx).Return(eth.APIGenesisResponse{Data: eth.ReducedGenesisData{GenesisTime: 10}}, nil)
	p.EXPECT().ConfigSpec(ctx).Return(eth.APIConfigResponse{Data: eth.ReducedConfigData{SecondsPerSlot: 2}}, nil)
	client := NewL1BeaconClient(p, L1BeaconClientConfig{})
	ref := eth.L1BlockRef{Time: 12}

	hash0, blob0 := makeTestBlob(0)
	hash1, blob1 := makeTestBlob(1)
	hash2, blob2 := makeTestBlob(2)
	sameHash := eth.IndexedBlobHash{
		Index: 3,
		Hash:  hash0.Hash,
	}

	hashes := []common.Hash{hash0.Hash, hash2.Hash, hash1.Hash, sameHash.Hash}
	beaconBlobs := []*eth.Blob{blob0, blob2, blob1, blob0}

	// construct the mock response for the beacon blobs call
	beaconBlobsResponse := eth.APIBeaconBlobsResponse{Data: beaconBlobs}
	p.EXPECT().BeaconBlobs(ctx, uint64(1), hashes).Return(beaconBlobsResponse, nil)

	resp, err := client.GetBlobsByHash(ctx, ref.Time, hashes)
	require.NoError(t, err)
	for i, blob := range resp {
		require.NotNil(t, blob, fmt.Sprintf("blob at index %d should not be nil", i))
	}
	require.Equal(t, []*eth.Blob{blob0, blob2, blob1, blob0}, resp)
}
