package sources

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/ethereum-optimism/optimism/op-node/derive"
)

type BeaconClient struct {
	BeaconAddress string
}

func NewBeaconClient(url string) (*BeaconClient, error) {
	if url == "" {
		return nil, errors.New("empty Beacon Client URL provided")
	}

	return &BeaconClient{
		BeaconAddress: url,
	}, nil
}

// "/eth/v1/blobs/sidecar/{block_id}"
func (cl *BeaconClient) FetchSidecar(ctx context.Context, slot uint64) (*derive.BlobsSidecar, error) {
	url := url.JoinPath(cl.BeaconAddress, "/eth/v1/blobs/sidecar/", slot)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode > 204 {
		return nil, errors.New("status code %d", res.StatusCode)
	}

	var sidecar derive.BlobsSidecar
	if err := json.Unmarshal(body, sidecar); err != nil {
		return nil, err
	}

	return sidecar, nil
}
