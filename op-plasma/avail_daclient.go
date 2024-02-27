package plasma

import "context"

type AvailDAClient struct {
	*DAClient
}

// GetInput returns the input data corresponding to a given commitment bytes
func (c *AvailDAClient) GetInput(ctx context.Context, key []byte) ([]byte, error) {
	return nil, nil
}

// SetInput sets the input data and returns the KZG commitment hash from Avail DA
func (c *AvailDAClient) SetInput(ctx context.Context, img []byte) ([]byte, error) {
	return nil, nil
}
