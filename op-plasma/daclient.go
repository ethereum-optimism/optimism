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
var ErrCommitmentMismatch = errors.New("commitment mismatch")

// DAClient is an HTTP client to communicate with a DA storage service.
// It creates commitments and retrieves input data + verifies if needed.
// Currently only supports Keccak256 commitments but may be extended eventually.
type DAClient struct {
	url    string
	verify bool
}

func NewDAClient(url string) *DAClient {
	return &DAClient{url: url}
}

// VerifyOnRead sets the client to verify the commitment on read.
// SHOULD enable if the storage service is not trusted.
func (c *DAClient) VerifyOnRead(verify bool) {
	c.verify = verify
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
	input, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if c.verify {
		exp := crypto.Keccak256(input)
		if !bytes.Equal(exp, key) {
			return nil, ErrCommitmentMismatch
		}
	}
	return input, nil
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
