package altda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// =========== SetInput (PUT path) errors ===========

// ErrInvalidInput is returned when the input is not valid for posting to the DA storage.
var ErrInvalidInput = errors.New("invalid input")

// ErrAltDADown is returned when the alt DA returns a 503 status code.
// It is used to signify that the alt DA is down and the client should failover to the eth DA.
// See https://github.com/ethereum-optimism/specs/issues/434
var ErrAltDADown = errors.New("alt DA is down: failover to eth DA")

// =========== GetInput (GET path) errors ===========

// ErrNotFound is returned when the server could not find the input.
// Note: this error only applies to keccak commitments, and not to EigenDA altda commitments,
// because a cert that parses correctly and passes the recency check by definition proves
// the availability of the blob that is certifies.
// See https://github.com/Layr-Labs/eigenda/blob/f4ef5cd5/docs/spec/src/integration/spec/6-secure-integration.md#derivation-process for more info.
var ErrNotFound = errors.New("not found")

// DropEigenDACommitmentError is returned when the eigenda-proxy returns a 418 TEAPOT error,
// which signifies that the commitment should be dropped/skipped from the derivation pipeline, as either:
//  1. the cert in the commitment is invalid
//  2. the cert's blob cannot be decoded into a frame (it was not encoded according to one of the supported codecs,
//     see https://github.com/Layr-Labs/eigenda/blob/f4ef5cd5/api/clients/codecs/blob_codec.go#L7-L15)
//
// See https://github.com/Layr-Labs/eigenda/blob/f4ef5cd5/docs/spec/src/integration/spec/6-secure-integration.md#derivation-process for more info.
//
// This error is parsed from the json body of the 418 TEAPOT error response.
// DropEigenDACommitmentError is the only error that can lead to a cert being dropped from the derivation pipeline.
// It is needed to protect the rollup from liveness attacks (derivation pipeline stalled by malicious batcher).
type DropEigenDACommitmentError struct {
	// The StatusCode field MUST be contained in the response body of the 418 TEAPOT error.
	StatusCode int
	// The Msg field is a human-readable string that explains the error.
	// It is optional, but should ideally be set to a meaningful value.
	Msg string
}

func (e DropEigenDACommitmentError) Error() string {
	return fmt.Sprintf("Invalid AltDA Commitment: cert verification failed with status code %v: %v", e.StatusCode, e.Msg)
}

// Validate that the status code is an integer between 1 and 4, and panics if it is not.
func (e DropEigenDACommitmentError) Validate() {
	if e.StatusCode < 1 || e.StatusCode > 4 {
		panic(fmt.Sprintf("DropEigenDACommitmentError: invalid status code %d, must be between 1 and 4", e.StatusCode))
	}
	// The Msg field should ideally be a human-readable string that explains the error,
	// but we don't enforce it.
}

// DAClient is an HTTP client to communicate with a DA storage service.
// It creates commitments and retrieves input data + verifies if needed.
type DAClient struct {
	url string
	// verify sets the client to verify a Keccak256 commitment on read.
	verify bool
	// whether commitment is precomputable (only applicable to keccak256)
	precompute bool
	getTimeout time.Duration
	putTimeout time.Duration
}

var _ DAStorage = (*DAClient)(nil)

func NewDAClient(url string, verify bool, pc bool) *DAClient {
	return &DAClient{
		url:        url,
		verify:     verify,
		precompute: pc,
	}
}

// GetInput returns the input data for the given encoded commitment bytes.
// The l1InclusionBlock at which the commitment was included in the batcher-inbox is submitted
// to the DA server as a query parameter.
// It is used to discard old commitments whose blobs have a risk of not being available anymore.
// It is optional, and passing a 0 value will tell the DA server to skip the check.
func (c *DAClient) GetInput(ctx context.Context, comm CommitmentData, l1InclusionBlockNumber uint64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/get/0x%x?l1_inclusion_block_number=%d", c.url, comm.Encode(), l1InclusionBlockNumber), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	client := &http.Client{Timeout: c.getTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode == http.StatusTeapot {
		defer resp.Body.Close()
		// Limit the body to 5000 bytes to prevent being DDoSed with a large error message.
		bytesLimitedBody := io.LimitReader(resp.Body, 5000)
		bodyBytes, _ := io.ReadAll(bytesLimitedBody)

		var invalidCommitmentErr DropEigenDACommitmentError
		if err := json.Unmarshal(bodyBytes, &invalidCommitmentErr); err != nil {
			return nil, fmt.Errorf("failed to decode 418 TEAPOT HTTP error body into a DropEigenDACommitmentError. "+
				"Consider updating proxy to a more recent version that contains https://github.com/Layr-Labs/eigenda/pull/1736: "+
				"%w", err)
		}
		invalidCommitmentErr.Validate()
		return nil, invalidCommitmentErr
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
func (c *DAClient) SetInput(ctx context.Context, img []byte) (CommitmentData, error) {
	if len(img) == 0 {
		return nil, ErrInvalidInput
	}

	if c.precompute { // precompute commitment (only applicable to keccak256)
		comm := NewKeccak256Commitment(img)
		if err := c.setInputWithCommit(ctx, comm, img); err != nil {
			return nil, err
		}

		return comm, nil
	}

	// let DA server generate commitment
	return c.setInput(ctx, img)

}

// setInputWithCommit sets a precomputed commitment for some pre-image data.
func (c *DAClient) setInputWithCommit(ctx context.Context, comm CommitmentData, img []byte) error {
	// encode with commitment type prefix
	key := comm.Encode()
	body := bytes.NewReader(img)
	url := fmt.Sprintf("%s/put/0x%x", c.url, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	client := &http.Client{Timeout: c.putTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to store preimage: %v", resp.StatusCode)
	}

	return nil
}

// setInput sets the input data and reads the respective DA generated commitment.
func (c *DAClient) setInput(ctx context.Context, img []byte) (CommitmentData, error) {
	if len(img) == 0 {
		return nil, ErrInvalidInput
	}

	body := bytes.NewReader(img)
	url := fmt.Sprintf("%s/put", c.url)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	client := &http.Client{Timeout: c.putTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, ErrAltDADown
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to store data: %v", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	comm, err := DecodeCommitmentData(b)
	if err != nil {
		return nil, err
	}

	return comm, nil
}
