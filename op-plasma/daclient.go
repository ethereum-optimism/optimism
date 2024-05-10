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
type DAClient struct {
	url string
	// verify sets the client to verify a Keccak256 commitment on read.
	verify bool

	ct CommitmentType
}

func NewDAClient(url string, verify bool, ct CommitmentType) *DAClient {
	return &DAClient{url, verify, ct}
}

// GetInput returns the input data for the given encoded commitment bytes.
func (c *DAClient) GetInput(ctx context.Context, comm Commit) ([]byte, error) {
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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get preimage: %v", resp.StatusCode)
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

// SetInput sets the input data and returns the respective commitment.
func (c *DAClient) SetInput(ctx context.Context, img []byte) (Commit, error) {
	if len(img) == 0 {
		return nil, ErrInvalidInput
	}

	if c.ct == Keccak256CommitmentType { // precompute commitment
		comm := Keccak256(img)
		if err := c.setInputWithCommit(ctx, comm, img); err != nil {
			return nil, err
		}

		return comm, nil
	}

	if c.ct == ServiceCommitmentType { // let DA server generate commitment
		return c.setInput(ctx, img)
	}

	return nil, fmt.Errorf("unknown commitment type provided")
}

// setInputWithCommit sets a precomputed commitment for some pre-image data.
func (c *DAClient) setInputWithCommit(ctx context.Context, comm Keccak256Commitment, img []byte) error {
	// encode with commitment type prefix
	key := comm.Encode()
	body := bytes.NewReader(img)
	url := fmt.Sprintf("%s/put/0x%x", c.url, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to store preimage: %v", resp.StatusCode)
	}

	return nil
}

// setInputs sets some data to a plasma DA server and reads the generated commitment from
// http response.
func (c *DAClient) setInput(ctx context.Context, img []byte) (Commit, error) {
	if len(img) == 0 {
		return nil, ErrInvalidInput
	}

	body := bytes.NewReader(img)
	url := fmt.Sprintf("%s/put/", c.url)
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
		return nil, fmt.Errorf("failed to store data: %v", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	comm, err := DecodeSvcCommit(b)
	if err != nil {
		return nil, err
	}

	return comm, nil
}
