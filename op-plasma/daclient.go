package plasma

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrNotFound is returned when the server could not find the input.
var ErrNotFound = errors.New("not found")

// ErrInvalidInput is returned when the input is not valid for posting to the DA storage.
var ErrInvalidInput = errors.New("invalid input")

// DAClient is an HTTP client to communicate with a DA storage service.
// It creates commitments and retrieves input data + verifies if needed.
// Currently only supports Keccak256 commitments but may be extended eventually.
type DAClient struct {
	url string
	// VerifyOnRead sets the client to verify the commitment on read.
	// SHOULD enable if the storage service is not trusted.
	verify bool
}

func NewDAClient(url string, verify bool) *DAClient {
	return &DAClient{url, verify}
}

// GetInput returns the input data for the given encoded commitment bytes.
func (c *DAClient) GetInput(ctx context.Context, comm Keccak256Commitment) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/get/0x%x", c.url, comm.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	defer resp.Body.Close()
	input, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if c.verify {
		if err := comm.Verify(input); err != nil {
			return nil, err
		}

	}
	return input, nil
}

// SetInput sets the input data and returns the keccak256 hash commitment.
func (c *DAClient) SetInput(ctx context.Context, img []byte) (Keccak256Commitment, error) {
	if len(img) == 0 {
		return nil, ErrInvalidInput
	}
	comm := Keccak256(img)
	// encode with commitment type prefix
	key := comm.Encode()
	body := bytes.NewReader(img)
	url := fmt.Sprintf("%s/put/0x%x", c.url, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to store preimage: %v", resp.StatusCode)
	}
	return comm, nil
}
