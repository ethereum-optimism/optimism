package plasma

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

var ErrNotFound = errors.New("not found")

type DAClient struct {
	url string
}

func NewDAClient(url string) *DAClient {
	return &DAClient{url}
}

// GetInput returns the input data for the given commitment bytes.
func (c *DAClient) GetInput(ctx context.Context, key []byte) ([]byte, error) {
	k := hexutil.Bytes(key)
	resp, err := http.Get(fmt.Sprintf("%s/get/%s", c.url, k))
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// SetInput sets the input data and returns the keccak256 hash commitment.
func (c *DAClient) SetInput(ctx context.Context, img []byte) ([]byte, error) {
	key := crypto.Keccak256(img)
	k := hexutil.Bytes(key)
	body := bytes.NewReader(img)
	url := fmt.Sprintf("%s/put/%s", c.url, k)
	resp, err := http.Post(url, "application/octet-stream", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to store preimage: %s", resp.Status)
	}
	return key, nil
}
